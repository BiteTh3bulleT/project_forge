package domain

import (
	"fmt"
	"strings"
)

type AutonomyLevel string

const (
	AutonomyLevelObserveOnly      AutonomyLevel = "level_0_observe_only"
	AutonomyLevelInternalPrep     AutonomyLevel = "level_1_internal_preparation"
	AutonomyLevelProposeOnly      AutonomyLevel = "level_2_propose_semantic_actions"
	AutonomyLevelAutoCommitSafe   AutonomyLevel = "level_3_auto_commit_safe_internal"
	AutonomyLevelApprovalRequired AutonomyLevel = "level_4_approval_required"
	AutonomyLevelMissionDelegated AutonomyLevel = "level_5_delegated_mission"
)

type AutonomyMode string

const (
	AutonomyModeOff      AutonomyMode = "off"
	AutonomyModeObserve  AutonomyMode = "observe"
	AutonomyModePropose  AutonomyMode = "propose"
	AutonomyModeMaintain AutonomyMode = "maintain"
	AutonomyModeMission  AutonomyMode = "mission"
)

type AutonomyRisk string

const (
	AutonomyRiskNone     AutonomyRisk = "none"
	AutonomyRiskLow      AutonomyRisk = "low"
	AutonomyRiskMedium   AutonomyRisk = "medium"
	AutonomyRiskHigh     AutonomyRisk = "high"
	AutonomyRiskCritical AutonomyRisk = "critical"
)

func (r AutonomyRisk) Rank() int {
	switch r {
	case AutonomyRiskNone:
		return 0
	case AutonomyRiskLow:
		return 1
	case AutonomyRiskMedium:
		return 2
	case AutonomyRiskHigh:
		return 3
	case AutonomyRiskCritical:
		return 4
	default:
		return 5
	}
}

type CharterStatus string

const (
	CharterDraft     CharterStatus = "draft"
	CharterActive    CharterStatus = "active"
	CharterSuspended CharterStatus = "suspended"
	CharterRevoked   CharterStatus = "revoked"
	CharterExpired   CharterStatus = "expired"
)

type IntentType string

const (
	IntentSelfMaintenance     IntentType = "self_maintenance"
	IntentMemoryCleanup       IntentType = "memory_cleanup"
	IntentContextPreparation  IntentType = "context_preparation"
	IntentContradictionReview IntentType = "contradiction_review"
	IntentStaleLoopReview     IntentType = "stale_loop_review"
	IntentProjectionRepair    IntentType = "projection_repair"
	IntentModelReview         IntentType = "model_review"
	IntentWorkspaceHygiene    IntentType = "workspace_hygiene"
	IntentUserRequested       IntentType = "user_requested"
	IntentSystemGenerated     IntentType = "system_generated"
	IntentFutureIRISGenerated IntentType = "future_iris_generated"
)

type IntentSource string

const (
	IntentSourceForge      IntentSource = "forge"
	IntentSourceRuleAgent  IntentSource = "rule_agent"
	IntentSourceCell       IntentSource = "cell"
	IntentSourceSystem     IntentSource = "system"
	IntentSourceUser       IntentSource = "user"
	IntentSourceFutureIRIS IntentSource = "future_iris"
	IntentSourceTest       IntentSource = "test"
)

type IntentStatus string

const (
	IntentStatusProposed  IntentStatus = "proposed"
	IntentStatusApproved  IntentStatus = "approved"
	IntentStatusRunning   IntentStatus = "running"
	IntentStatusCompleted IntentStatus = "completed"
	IntentStatusBlocked   IntentStatus = "blocked"
	IntentStatusRejected  IntentStatus = "rejected"
	IntentStatusCancelled IntentStatus = "cancelled"
	IntentStatusExpired   IntentStatus = "expired"
)

type FreedomBudgetStatus string

const (
	BudgetStatusActive    FreedomBudgetStatus = "active"
	BudgetStatusSuspended FreedomBudgetStatus = "suspended"
	BudgetStatusExhausted FreedomBudgetStatus = "exhausted"
	BudgetStatusExpired   FreedomBudgetStatus = "expired"
)

type FreedomBudgetPeriod string

const (
	BudgetPeriodPerRun  FreedomBudgetPeriod = "per_run"
	BudgetPeriodHourly  FreedomBudgetPeriod = "hourly"
	BudgetPeriodDaily   FreedomBudgetPeriod = "daily"
	BudgetPeriodWeekly  FreedomBudgetPeriod = "weekly"
	BudgetPeriodMission FreedomBudgetPeriod = "mission"
)

type CuriosityStatus string

const (
	CuriosityOpen             CuriosityStatus = "open"
	CuriosityPromotedToIntent CuriosityStatus = "promoted_to_intent"
	CuriosityDismissed        CuriosityStatus = "dismissed"
	CuriosityExpired          CuriosityStatus = "expired"
)

type AutonomyDecisionType string

const (
	DecisionAllowAutoCommit  AutonomyDecisionType = "allow_auto_commit"
	DecisionAllowProposeOnly AutonomyDecisionType = "allow_propose_only"
	DecisionApprovalRequired AutonomyDecisionType = "approval_required"
	DecisionDeny             AutonomyDecisionType = "deny"
	DecisionBlockedByBudget  AutonomyDecisionType = "blocked_by_budget"
	DecisionBlockedByCharter AutonomyDecisionType = "blocked_by_charter"
	DecisionBlockedByRisk    AutonomyDecisionType = "blocked_by_risk"
	DecisionBlockedByScope   AutonomyDecisionType = "blocked_by_scope"
	DecisionBlockedByKernel  AutonomyDecisionType = "blocked_by_kernel"
)

type AutonomyErrorCode string

const (
	AutonomyErrInvalidInput      AutonomyErrorCode = "AUTONOMY_INVALID_INPUT"
	AutonomyErrInvalidScope      AutonomyErrorCode = "AUTONOMY_INVALID_SCOPE"
	AutonomyErrMissingCharter    AutonomyErrorCode = "AUTONOMY_MISSING_CHARTER"
	AutonomyErrCharterInactive   AutonomyErrorCode = "AUTONOMY_CHARTER_INACTIVE"
	AutonomyErrBudgetDenied      AutonomyErrorCode = "AUTONOMY_BUDGET_DENIED"
	AutonomyErrApprovalRequired  AutonomyErrorCode = "AUTONOMY_APPROVAL_REQUIRED"
	AutonomyErrGuardrail         AutonomyErrorCode = "AUTONOMY_GUARDRAIL_BLOCKED"
	AutonomyErrKernelBlocked     AutonomyErrorCode = "AUTONOMY_KERNEL_BLOCKED"
	AutonomyErrInvalidTransition AutonomyErrorCode = "AUTONOMY_INVALID_TRANSITION"
	AutonomyErrNotFound          AutonomyErrorCode = "AUTONOMY_NOT_FOUND"
)

type AutonomyError struct {
	Code    AutonomyErrorCode `json:"code"`
	Field   string            `json:"field,omitempty"`
	Message string            `json:"message"`
}

func (e AutonomyError) Error() string {
	if strings.TrimSpace(e.Field) == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s (%s): %s", e.Code, e.Field, e.Message)
}

type ConditionRule struct {
	Action               SemanticActionType `json:"action"`
	Conditions           []string           `json:"conditions"`
	MaxRisk              AutonomyRisk       `json:"maxRisk"`
	RequiresEvidence     bool               `json:"requiresEvidence"`
	AllowedObjectKinds   []string           `json:"allowedObjectKinds"`
	DeniedObjectKinds    []string           `json:"deniedObjectKinds"`
	WorkspaceConstraints []string           `json:"workspaceConstraints"`
	BudgetCost           int                `json:"budgetCost"`
	ApprovalRequiredWhen []string           `json:"approvalRequiredWhen"`
}

type AutonomyCharter struct {
	ID                      string               `json:"id"`
	Name                    string               `json:"name"`
	Description             string               `json:"description"`
	Scope                   ForgeScope           `json:"scope"`
	Status                  CharterStatus        `json:"status"`
	Purpose                 string               `json:"purpose"`
	AllowedActions          []SemanticActionType `json:"allowedActions"`
	DeniedActions           []SemanticActionType `json:"deniedActions"`
	ConditionalActions      []ConditionRule      `json:"conditionalActions"`
	RequiresApprovalActions []SemanticActionType `json:"requiresApprovalActions"`
	AllowedSources          []IntentSource       `json:"allowedSources"`
	RiskLimits              []AutonomyRisk       `json:"riskLimits"`
	FreedomBudgetID         string               `json:"freedomBudgetId,omitempty"`
	EffectiveFrom           int64                `json:"effectiveFrom"`
	ExpiresAt               *int64               `json:"expiresAt,omitempty"`
	CreatedBy               string               `json:"createdBy"`
	ApprovedBy              string               `json:"approvedBy,omitempty"`
	Provenance              Provenance           `json:"provenance"`
	CreatedAt               int64                `json:"createdAt"`
	UpdatedAt               int64                `json:"updatedAt"`
	Metadata                map[string]any       `json:"metadata,omitempty"`
}

func (c AutonomyCharter) IsActive(now int64) bool {
	if c.Status != CharterActive {
		return false
	}
	if c.EffectiveFrom > 0 && now > 0 && now < c.EffectiveFrom {
		return false
	}
	if c.ExpiresAt != nil && now > 0 && now > *c.ExpiresAt {
		return false
	}
	return true
}

func (c AutonomyCharter) AllowsAction(action SemanticActionType) bool {
	if len(c.AllowedActions) == 0 {
		return false
	}
	for _, item := range c.AllowedActions {
		if item == action {
			return true
		}
	}
	return false
}

func (c AutonomyCharter) DeniesAction(action SemanticActionType) bool {
	for _, item := range c.DeniedActions {
		if item == action {
			return true
		}
	}
	return false
}

func (c AutonomyCharter) RequiresApproval(action SemanticActionType) bool {
	for _, item := range c.RequiresApprovalActions {
		if item == action {
			return true
		}
	}
	return false
}

func (c AutonomyCharter) Validate(now int64) []AutonomyError {
	errs := []AutonomyError{}
	if strings.TrimSpace(c.ID) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "charter.id", Message: "charter id is required"})
	}
	if strings.TrimSpace(c.Scope.WorkspaceID) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "charter.scope.workspaceId", Message: "charter scope.workspaceId is required"})
	}
	if strings.TrimSpace(c.Name) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "charter.name", Message: "charter name is required"})
	}
	if strings.TrimSpace(c.CreatedBy) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "charter.createdBy", Message: "charter createdBy is required"})
	}
	if c.ExpiresAt != nil && c.EffectiveFrom > 0 && *c.ExpiresAt < c.EffectiveFrom {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "charter.expiresAt", Message: "charter expiresAt must be after effectiveFrom"})
	}
	if c.Status == CharterActive && c.EffectiveFrom == 0 {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "charter.effectiveFrom", Message: "active charter requires effectiveFrom"})
	}
	if c.Status == CharterActive && !c.IsActive(now) && c.ExpiresAt != nil && now > *c.ExpiresAt {
		errs = append(errs, AutonomyError{Code: AutonomyErrCharterInactive, Field: "charter.status", Message: "charter is expired"})
	}
	return errs
}

type AutonomyIntent struct {
	ID               string           `json:"id"`
	Type             IntentType       `json:"type"`
	Title            string           `json:"title"`
	Description      string           `json:"description"`
	Source           IntentSource     `json:"source"`
	ProposedBy       string           `json:"proposedBy"`
	Scope            ForgeScope       `json:"scope"`
	Status           IntentStatus     `json:"status"`
	Risk             AutonomyRisk     `json:"risk"`
	AutonomyLevel    AutonomyLevel    `json:"autonomyLevel"`
	CharterID        string           `json:"charterId,omitempty"`
	BudgetID         string           `json:"budgetId,omitempty"`
	RequiredApproval bool             `json:"requiredApproval"`
	ApprovalID       string           `json:"approvalId,omitempty"`
	ProposedActions  []SyscallRequest `json:"proposedActions"`
	CommittedActions []SyscallResult  `json:"committedActions"`
	BlockedReasons   []string         `json:"blockedReasons"`
	Evidence         []string         `json:"evidence"`
	Provenance       Provenance       `json:"provenance"`
	CorrelationID    string           `json:"correlationId,omitempty"`
	TraceID          string           `json:"traceId,omitempty"`
	CreatedAt        int64            `json:"createdAt"`
	UpdatedAt        int64            `json:"updatedAt"`
	ExpiresAt        *int64           `json:"expiresAt,omitempty"`
	Metadata         map[string]any   `json:"metadata,omitempty"`
}

func (i AutonomyIntent) Validate() []AutonomyError {
	errs := []AutonomyError{}
	if strings.TrimSpace(i.ID) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "intent.id", Message: "intent id is required"})
	}
	if strings.TrimSpace(i.Scope.WorkspaceID) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidScope, Field: "intent.scope.workspaceId", Message: "intent scope.workspaceId is required"})
	}
	if strings.TrimSpace(string(i.Type)) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "intent.type", Message: "intent type is required"})
	}
	if strings.TrimSpace(string(i.Source)) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "intent.source", Message: "intent source is required"})
	}
	if strings.TrimSpace(string(i.Status)) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "intent.status", Message: "intent status is required"})
	}
	if strings.TrimSpace(i.Title) == "" {
		errs = append(errs, AutonomyError{Code: AutonomyErrInvalidInput, Field: "intent.title", Message: "intent title is required"})
	}
	return errs
}

func (i AutonomyIntent) CanTransition(to IntentStatus) bool {
	if i.Status == to {
		return true
	}
	switch i.Status {
	case IntentStatusProposed:
		return to == IntentStatusApproved || to == IntentStatusRejected || to == IntentStatusCancelled || to == IntentStatusExpired
	case IntentStatusApproved:
		return to == IntentStatusRunning || to == IntentStatusCancelled || to == IntentStatusExpired
	case IntentStatusRunning:
		return to == IntentStatusCompleted || to == IntentStatusBlocked || to == IntentStatusCancelled
	case IntentStatusBlocked:
		return to == IntentStatusApproved || to == IntentStatusCancelled || to == IntentStatusExpired
	case IntentStatusCompleted, IntentStatusRejected, IntentStatusCancelled, IntentStatusExpired:
		return false
	default:
		return false
	}
}

type FreedomBudgetUsage struct {
	SelfActionsPerRun       int `json:"selfActionsPerRun"`
	RunsPerPeriod           int `json:"runsPerPeriod"`
	CommittedActionsPeriod  int `json:"committedActionsPerPeriod"`
	ProposedActionsPeriod   int `json:"proposedActionsPerPeriod"`
	InternalToolCalls       int `json:"internalToolCalls"`
	ExternalToolCallsNoAppr int `json:"externalToolCallsWithoutApproval"`
	ArchiveActions          int `json:"archiveActions"`
	ContextPrecompilations  int `json:"contextPrecompilations"`
	ProjectionRebuilds      int `json:"projectionRebuilds"`
	RuleAgentRuns           int `json:"ruleAgentRuns"`
	CostUnits               int `json:"costUnits"`
}

type FreedomBudget struct {
	ID                             string              `json:"id"`
	Name                           string              `json:"name"`
	Scope                          ForgeScope          `json:"scope"`
	Status                         FreedomBudgetStatus `json:"status"`
	Period                         FreedomBudgetPeriod `json:"period"`
	MaxSelfActionsPerRun           int                 `json:"maxSelfActionsPerRun"`
	MaxRunsPerPeriod               int                 `json:"maxRunsPerPeriod"`
	MaxCommittedActionsPerPeriod   int                 `json:"maxCommittedActionsPerPeriod"`
	MaxProposedActionsPerPeriod    int                 `json:"maxProposedActionsPerPeriod"`
	MaxInternalToolCalls           int                 `json:"maxInternalToolCalls"`
	MaxExternalCallsWithoutApprove int                 `json:"maxExternalToolCallsWithoutApproval"`
	MaxArchiveActions              int                 `json:"maxArchiveActions"`
	MaxContextPrecompilations      int                 `json:"maxContextPrecompilations"`
	MaxProjectionRebuilds          int                 `json:"maxProjectionRebuilds"`
	MaxRuleAgentRuns               int                 `json:"maxRuleAgentRuns"`
	MaxCostUnits                   *int                `json:"maxCostUnits,omitempty"`
	Usage                          FreedomBudgetUsage  `json:"usage"`
	ResetsAt                       int64               `json:"resetsAt"`
	CreatedAt                      int64               `json:"createdAt"`
	UpdatedAt                      int64               `json:"updatedAt"`
	Metadata                       map[string]any      `json:"metadata,omitempty"`
}

type BudgetReservation struct {
	ID           string         `json:"id"`
	BudgetID     string         `json:"budgetId"`
	IntentID     string         `json:"intentId"`
	Scope        ForgeScope     `json:"scope"`
	RequestedFor string         `json:"requestedFor"`
	Units        int            `json:"units"`
	CreatedAt    int64          `json:"createdAt"`
	Consumed     bool           `json:"consumed"`
	Released     bool           `json:"released"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type PolicyKernelPreview struct {
	Action  SyscallRequest `json:"action"`
	Result  SyscallResult  `json:"result"`
	Blocked bool           `json:"blocked"`
}

type AutonomyDecision struct {
	ID                     string                `json:"id"`
	IntentID               string                `json:"intentId"`
	Decision               AutonomyDecisionType  `json:"decision"`
	AutonomyLevel          AutonomyLevel         `json:"autonomyLevel"`
	Risk                   AutonomyRisk          `json:"risk"`
	CharterID              string                `json:"charterId,omitempty"`
	BudgetID               string                `json:"budgetId,omitempty"`
	BudgetReservationID    string                `json:"budgetReservationId,omitempty"`
	RequiredApprovalReason string                `json:"requiredApprovalReason,omitempty"`
	DeniedReasons          []string              `json:"deniedReasons"`
	Warnings               []string              `json:"warnings"`
	AllowedActions         []SyscallRequest      `json:"allowedActions"`
	BlockedActions         []SyscallRequest      `json:"blockedActions"`
	KernelPreview          []PolicyKernelPreview `json:"kernelPreview,omitempty"`
	Explanation            string                `json:"explanation"`
	CorrelationID          string                `json:"correlationId,omitempty"`
	TraceID                string                `json:"traceId,omitempty"`
	Metadata               map[string]any        `json:"metadata,omitempty"`
	CreatedAt              int64                 `json:"createdAt"`
}

type ApprovalEscalationResult struct {
	Status            ApprovalStatus `json:"status"`
	ApprovalID        string         `json:"approvalId,omitempty"`
	Reason            string         `json:"reason,omitempty"`
	OperatorMessage   string         `json:"operatorMessage,omitempty"`
	RecommendedAction string         `json:"recommendedAction,omitempty"`
}

type CuriosityItem struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	Question  string          `json:"question"`
	Source    IntentSource    `json:"source"`
	Scope     ForgeScope      `json:"scope"`
	Evidence  []string        `json:"evidence"`
	Priority  string          `json:"priority"`
	Status    CuriosityStatus `json:"status"`
	CreatedAt int64           `json:"createdAt"`
	UpdatedAt int64           `json:"updatedAt"`
	ExpiresAt *int64          `json:"expiresAt,omitempty"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
}

type AutonomyRunMode string

const (
	RunModeExplainOnly        AutonomyRunMode = "explain_only"
	RunModeValidateOnly       AutonomyRunMode = "validate_only"
	RunModeProposeOnly        AutonomyRunMode = "propose_only"
	RunModeCommitIfAuthorized AutonomyRunMode = "commit_if_authorized"
)

type AutonomyRunSummary struct {
	IntentID           string                   `json:"intentId"`
	DecisionID         string                   `json:"decisionId,omitempty"`
	Decision           AutonomyDecisionType     `json:"decision"`
	CommittedObjectIDs []string                 `json:"committedObjectIds"`
	CommittedActions   []SyscallResult          `json:"committedActions"`
	Approval           ApprovalEscalationResult `json:"approval"`
	Warnings           []string                 `json:"warnings"`
	Errors             []AutonomyError          `json:"errors"`
	CorrelationID      string                   `json:"correlationId,omitempty"`
	TraceID            string                   `json:"traceId,omitempty"`
}

func (s AutonomyRunSummary) Error() error {
	if len(s.Errors) == 0 {
		return nil
	}
	first := s.Errors[0]
	if strings.TrimSpace(first.Field) == "" {
		return fmt.Errorf("%s: %s", first.Code, first.Message)
	}
	return fmt.Errorf("%s (%s): %s", first.Code, first.Field, first.Message)
}
