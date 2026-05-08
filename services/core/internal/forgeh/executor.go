package forgeh

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrExecutionBlocked        = errors.New("resource action execution blocked")
	ErrExecutionAdapterMissing = errors.New("resource action execution adapter missing")
	ErrExecutionActionUnknown  = errors.New("resource action execution action unknown")
	ErrExecutionNoSideEffects  = errors.New("resource action execution produced no bounded side effects")
)

const (
	SideEffectOperatorNotificationRecorded      = "operator_notification_recorded"
	SideEffectBackgroundIngestDeferred          = "lane_policy_background_ingest_deferred"
	SideEffectBackgroundIngestPaused            = "lane_policy_background_ingest_paused"
	SideEffectEmbeddingDeferred                 = "lane_policy_embedding_deferred"
	SideEffectMaintenanceScheduledLater         = "lane_policy_maintenance_scheduled_later"
	SideEffectNewModelLoadsDenied               = "model_policy_new_loads_denied"
	SideEffectLargeModelLoadsDeferred           = "model_policy_large_model_loads_deferred"
	SideEffectCurrentModelOnlyPreferred         = "model_policy_current_model_only_preferred"
	SideEffectCPUSafeModePreferred              = "model_policy_cpu_safe_mode_preferred"
	SideEffectDegradedModeFlagRecorded          = "degraded_mode_flag_recorded"
	ExecutionReasonAlreadyExecuted              = "already_executed"
	ExecutionReasonProposalNotApproved          = "proposal_not_approved"
	ExecutionReasonProposalExpired              = "proposal_expired"
	ExecutionReasonProposalNotAdvisory          = "proposal_not_advisory"
	ExecutionReasonActionNotAllowed             = "action_not_allowed"
	ExecutionReasonExecutionAdapterMissing      = "execution_adapter_missing"
	ExecutionReasonAdapterReturnedError         = "execution_adapter_returned_error"
	ExecutionReasonApprovedBoundedActionApplied = "approved_bounded_action_applied"
)

type OperatorNotifier interface {
	NotifyOperator(ResourceActionProposal) ([]string, error)
}

type LanePolicyWriter interface {
	DeferBackgroundIngest(ResourceActionProposal) ([]string, error)
	PauseBackgroundIngest(ResourceActionProposal) ([]string, error)
	DeferEmbedding(ResourceActionProposal) ([]string, error)
	ScheduleMaintenanceLater(ResourceActionProposal) ([]string, error)
}

type ModelPolicyWriter interface {
	DenyNewModelLoad(ResourceActionProposal) ([]string, error)
	DeferLargeModelLoad(ResourceActionProposal) ([]string, error)
	PreferCurrentModelOnly(ResourceActionProposal) ([]string, error)
	PreferCPUSafeMode(ResourceActionProposal) ([]string, error)
}

type DegradedModeWriter interface {
	EnterDegradedMode(ResourceActionProposal) ([]string, error)
}

type ExecutorOptions struct {
	Now          func() time.Time
	Notifier     OperatorNotifier
	LanePolicy   LanePolicyWriter
	ModelPolicy  ModelPolicyWriter
	DegradedMode DegradedModeWriter
	PreviousRuns []ResourceActionExecution
}

type ResourceActionExecutor struct {
	now          func() time.Time
	notifier     OperatorNotifier
	lanePolicy   LanePolicyWriter
	modelPolicy  ModelPolicyWriter
	degradedMode DegradedModeWriter
	runs         map[string]ResourceActionExecution
}

func NewResourceActionExecutor(opts ExecutorOptions) *ResourceActionExecutor {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	runs := map[string]ResourceActionExecution{}
	for _, run := range opts.PreviousRuns {
		if strings.TrimSpace(run.ProposalID) != "" {
			runs[run.ProposalID] = run
		}
	}
	return &ResourceActionExecutor{
		now:          now,
		notifier:     opts.Notifier,
		lanePolicy:   opts.LanePolicy,
		modelPolicy:  opts.ModelPolicy,
		degradedMode: opts.DegradedMode,
		runs:         runs,
	}
}

func (e *ResourceActionExecutor) Execute(proposal ResourceActionProposal) (ResourceActionExecution, error) {
	if e == nil {
		e = NewResourceActionExecutor(ExecutorOptions{})
	}
	if e.now == nil {
		e.now = time.Now
	}
	if existing, ok := e.runs[proposal.ProposalID]; ok {
		return existing, nil
	}
	now := e.now().UTC()
	execution := newExecution(proposal, now)
	if proposal.Status != ProposalStatusApproved {
		execution = blockExecution(execution, ExecutionReasonProposalNotApproved)
		e.record(execution)
		return execution, ErrExecutionBlocked
	}
	if !proposal.ExpiresAt.IsZero() && !now.Before(proposal.ExpiresAt) {
		execution = blockExecution(execution, ExecutionReasonProposalExpired)
		e.record(execution)
		return execution, ErrExecutionBlocked
	}
	if !proposal.AdvisoryOnly {
		execution = blockExecution(execution, ExecutionReasonProposalNotAdvisory)
		e.record(execution)
		return execution, ErrExecutionBlocked
	}
	result, sideEffects, err := e.apply(proposal)
	if err != nil {
		if errors.Is(err, ErrExecutionAdapterMissing) {
			execution = blockExecution(execution, ExecutionReasonExecutionAdapterMissing)
			execution.Errors = []string{err.Error()}
			e.record(execution)
			return execution, err
		}
		if errors.Is(err, ErrExecutionActionUnknown) {
			execution = blockExecution(execution, ExecutionReasonActionNotAllowed)
			execution.Errors = []string{err.Error()}
			e.record(execution)
			return execution, err
		}
		execution = failExecution(execution, ExecutionReasonAdapterReturnedError)
		execution.Errors = []string{err.Error()}
		e.record(execution)
		return execution, err
	}
	sideEffects = compactStrings(sideEffects)
	if len(sideEffects) == 0 {
		execution = failExecution(execution, ExecutionReasonAdapterReturnedError)
		execution.Errors = []string{ErrExecutionNoSideEffects.Error()}
		e.record(execution)
		return execution, ErrExecutionNoSideEffects
	}
	execution.Status = ExecutionStatusExecuted
	execution.FinishedAt = now
	execution.Result = result
	execution.Reason = ExecutionReasonApprovedBoundedActionApplied
	execution.SideEffects = sideEffects
	execution.ExecutionID = executionID(execution)
	e.record(execution)
	return execution, nil
}

func (e *ResourceActionExecutor) Executions() []ResourceActionExecution {
	if e == nil {
		return nil
	}
	out := make([]ResourceActionExecution, 0, len(e.runs))
	for _, run := range e.runs {
		out = append(out, run)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExecutionID < out[j].ExecutionID
	})
	return out
}

func (e *ResourceActionExecutor) record(execution ResourceActionExecution) {
	if e.runs == nil {
		e.runs = map[string]ResourceActionExecution{}
	}
	execution.ExecutionID = executionID(execution)
	e.runs[execution.ProposalID] = execution
}

func (e *ResourceActionExecutor) apply(proposal ResourceActionProposal) (string, []string, error) {
	switch proposal.ActionType {
	case ProposalActionWarnOperator:
		if e.notifier == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.notifier.NotifyOperator(proposal)
		return ExecutionResultOperatorWarned, sideEffects, err
	case ProposalActionDeferBackgroundIngest:
		if e.lanePolicy == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.lanePolicy.DeferBackgroundIngest(proposal)
		return ExecutionResultBackgroundIngestDeferred, sideEffects, err
	case ProposalActionPauseBackgroundIngest:
		if e.lanePolicy == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.lanePolicy.PauseBackgroundIngest(proposal)
		return ExecutionResultBackgroundIngestPaused, sideEffects, err
	case ProposalActionDeferEmbedding:
		if e.lanePolicy == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.lanePolicy.DeferEmbedding(proposal)
		return ExecutionResultEmbeddingDeferred, sideEffects, err
	case ProposalActionScheduleMaintenance:
		if e.lanePolicy == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.lanePolicy.ScheduleMaintenanceLater(proposal)
		return ExecutionResultMaintenanceScheduledLater, sideEffects, err
	case ProposalActionDenyNewModelLoad:
		if e.modelPolicy == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.modelPolicy.DenyNewModelLoad(proposal)
		return ExecutionResultNewModelLoadDenied, sideEffects, err
	case ProposalActionDeferLargeModelLoad:
		if e.modelPolicy == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.modelPolicy.DeferLargeModelLoad(proposal)
		return ExecutionResultLargeModelLoadDeferred, sideEffects, err
	case ProposalActionPreferCurrentModelOnly:
		if e.modelPolicy == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.modelPolicy.PreferCurrentModelOnly(proposal)
		return ExecutionResultCurrentModelOnlyPreferred, sideEffects, err
	case ProposalActionPreferCPUSafeMode:
		if e.modelPolicy == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.modelPolicy.PreferCPUSafeMode(proposal)
		return ExecutionResultCPUSafeModePreferred, sideEffects, err
	case ProposalActionEnterDegradedMode:
		if e.degradedMode == nil {
			return "", nil, ErrExecutionAdapterMissing
		}
		sideEffects, err := e.degradedMode.EnterDegradedMode(proposal)
		return ExecutionResultDegradedModeEntered, sideEffects, err
	default:
		return "", nil, ErrExecutionActionUnknown
	}
}

func newExecution(proposal ResourceActionProposal, now time.Time) ResourceActionExecution {
	return ResourceActionExecution{
		ProposalID:               proposal.ProposalID,
		SourcePolicyID:           proposal.SourcePolicyID,
		SourceSnapshotID:         proposal.SourceSnapshotID,
		ActionType:               proposal.ActionType,
		TargetLane:               proposal.TargetLane,
		Status:                   ExecutionStatusPlanned,
		StartedAt:                now,
		OperatorApprovalRequired: proposal.RequiresOperatorApproval,
		ApprovedBeforeExecution:  proposal.Status == ProposalStatusApproved,
		Bounded:                  true,
		HostMutation:             false,
		SemanticMemoryWrite:      false,
		ModelruntimeMutation:     false,
	}
}

func blockExecution(execution ResourceActionExecution, reason string) ResourceActionExecution {
	execution.Status = ExecutionStatusBlocked
	execution.FinishedAt = execution.StartedAt
	execution.Reason = reason
	return execution
}

func failExecution(execution ResourceActionExecution, reason string) ResourceActionExecution {
	execution.Status = ExecutionStatusFailed
	execution.FinishedAt = execution.StartedAt
	execution.Reason = reason
	return execution
}

func executionID(execution ResourceActionExecution) string {
	clone := execution
	clone.ExecutionID = ""
	body, _ := json.Marshal(clone)
	sum := sha256.Sum256(body)
	return "forgeh_execution_" + hex.EncodeToString(sum[:8])
}
