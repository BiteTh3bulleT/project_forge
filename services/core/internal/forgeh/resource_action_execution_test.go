package forgeh

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestResourceActionExecutorApprovedProposalExecutes(t *testing.T) {
	executor, adapters := testExecutor()
	execution, err := executor.Execute(approvedProposalForAction(ProposalActionDeferBackgroundIngest))
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != ExecutionStatusExecuted {
		t.Fatalf("status = %q", execution.Status)
	}
	if execution.Result != ExecutionResultBackgroundIngestDeferred {
		t.Fatalf("result = %q", execution.Result)
	}
	if !execution.ApprovedBeforeExecution || !execution.Bounded || execution.HostMutation || execution.SemanticMemoryWrite || execution.ModelruntimeMutation {
		t.Fatalf("invalid execution boundary fields: %#v", execution)
	}
	if adapters.lane.calls != 1 {
		t.Fatalf("expected one lane adapter call, got %d", adapters.lane.calls)
	}
}

func TestResourceActionExecutorBlocksIneligibleProposalStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{name: "proposed", status: ProposalStatusProposed},
		{name: "rejected", status: ProposalStatusRejected},
		{name: "expired", status: ProposalStatusExpired},
		{name: "superseded", status: ProposalStatusSuperseded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, adapters := testExecutor()
			proposal := approvedProposalForAction(ProposalActionWarnOperator)
			proposal.Status = tt.status
			execution, err := executor.Execute(proposal)
			if !errors.Is(err, ErrExecutionBlocked) {
				t.Fatalf("expected blocked error, got %v", err)
			}
			if execution.Status != ExecutionStatusBlocked || execution.Reason != ExecutionReasonProposalNotApproved {
				t.Fatalf("unexpected blocked execution: %#v", execution)
			}
			if adapters.totalCalls() != 0 {
				t.Fatalf("adapter should not be called, calls=%d", adapters.totalCalls())
			}
		})
	}
}

func TestResourceActionExecutorBlocksExpiredApprovedProposal(t *testing.T) {
	executor, adapters := testExecutor()
	proposal := approvedProposalForAction(ProposalActionWarnOperator)
	proposal.ExpiresAt = fixedNow().Add(-time.Second)
	execution, err := executor.Execute(proposal)
	if !errors.Is(err, ErrExecutionBlocked) {
		t.Fatalf("expected blocked error, got %v", err)
	}
	if execution.Status != ExecutionStatusBlocked || execution.Reason != ExecutionReasonProposalExpired {
		t.Fatalf("unexpected execution: %#v", execution)
	}
	if adapters.totalCalls() != 0 {
		t.Fatalf("adapter should not be called, calls=%d", adapters.totalCalls())
	}
}

func TestResourceActionExecutorBlocksNonAdvisoryProposal(t *testing.T) {
	executor, adapters := testExecutor()
	proposal := approvedProposalForAction(ProposalActionWarnOperator)
	proposal.AdvisoryOnly = false
	execution, err := executor.Execute(proposal)
	if !errors.Is(err, ErrExecutionBlocked) {
		t.Fatalf("expected blocked error, got %v", err)
	}
	if execution.Status != ExecutionStatusBlocked || execution.Reason != ExecutionReasonProposalNotAdvisory {
		t.Fatalf("unexpected execution: %#v", execution)
	}
	if adapters.totalCalls() != 0 {
		t.Fatalf("adapter should not be called, calls=%d", adapters.totalCalls())
	}
}

func TestResourceActionExecutorBlocksUnknownAction(t *testing.T) {
	executor, adapters := testExecutor()
	proposal := approvedProposalForAction("unknown_action")
	execution, err := executor.Execute(proposal)
	if !errors.Is(err, ErrExecutionActionUnknown) {
		t.Fatalf("expected unknown action error, got %v", err)
	}
	if execution.Status != ExecutionStatusBlocked || execution.Reason != ExecutionReasonActionNotAllowed {
		t.Fatalf("unexpected execution: %#v", execution)
	}
	if adapters.totalCalls() != 0 {
		t.Fatalf("adapter should not be called, calls=%d", adapters.totalCalls())
	}
}

func TestResourceActionExecutorRequiresAdapter(t *testing.T) {
	executor := NewResourceActionExecutor(ExecutorOptions{Now: fixedNow})
	execution, err := executor.Execute(approvedProposalForAction(ProposalActionWarnOperator))
	if !errors.Is(err, ErrExecutionAdapterMissing) {
		t.Fatalf("expected missing adapter error, got %v", err)
	}
	if execution.Status != ExecutionStatusBlocked || execution.Reason != ExecutionReasonExecutionAdapterMissing {
		t.Fatalf("unexpected execution: %#v", execution)
	}
}

func TestResourceActionExecutorAllowedActions(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		result     string
		sideEffect string
	}{
		{name: "warn operator", action: ProposalActionWarnOperator, result: ExecutionResultOperatorWarned, sideEffect: SideEffectOperatorNotificationRecorded},
		{name: "defer background ingest", action: ProposalActionDeferBackgroundIngest, result: ExecutionResultBackgroundIngestDeferred, sideEffect: SideEffectBackgroundIngestDeferred},
		{name: "pause background ingest", action: ProposalActionPauseBackgroundIngest, result: ExecutionResultBackgroundIngestPaused, sideEffect: SideEffectBackgroundIngestPaused},
		{name: "defer embedding", action: ProposalActionDeferEmbedding, result: ExecutionResultEmbeddingDeferred, sideEffect: SideEffectEmbeddingDeferred},
		{name: "deny new model load", action: ProposalActionDenyNewModelLoad, result: ExecutionResultNewModelLoadDenied, sideEffect: SideEffectNewModelLoadsDenied},
		{name: "defer large model load", action: ProposalActionDeferLargeModelLoad, result: ExecutionResultLargeModelLoadDeferred, sideEffect: SideEffectLargeModelLoadsDeferred},
		{name: "prefer current model only", action: ProposalActionPreferCurrentModelOnly, result: ExecutionResultCurrentModelOnlyPreferred, sideEffect: SideEffectCurrentModelOnlyPreferred},
		{name: "prefer cpu safe mode", action: ProposalActionPreferCPUSafeMode, result: ExecutionResultCPUSafeModePreferred, sideEffect: SideEffectCPUSafeModePreferred},
		{name: "enter degraded mode", action: ProposalActionEnterDegradedMode, result: ExecutionResultDegradedModeEntered, sideEffect: SideEffectDegradedModeFlagRecorded},
		{name: "schedule maintenance later", action: ProposalActionScheduleMaintenance, result: ExecutionResultMaintenanceScheduledLater, sideEffect: SideEffectMaintenanceScheduledLater},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, _ := testExecutor()
			execution, err := executor.Execute(approvedProposalForAction(tt.action))
			if err != nil {
				t.Fatal(err)
			}
			if execution.Result != tt.result {
				t.Fatalf("result = %q want %q", execution.Result, tt.result)
			}
			if len(execution.SideEffects) != 1 || execution.SideEffects[0] != tt.sideEffect {
				t.Fatalf("side effects = %#v want %q", execution.SideEffects, tt.sideEffect)
			}
		})
	}
}

func TestResourceActionExecutorAdapterErrorFailsWithoutMutationFlags(t *testing.T) {
	adapters := newTestAdapters()
	adapters.notifier.err = errors.New("bounded adapter failed")
	executor := NewResourceActionExecutor(ExecutorOptions{Now: fixedNow, Notifier: adapters.notifier, LanePolicy: adapters.lane, ModelPolicy: adapters.model, DegradedMode: adapters.degraded})
	execution, err := executor.Execute(approvedProposalForAction(ProposalActionWarnOperator))
	if err == nil {
		t.Fatal("expected adapter error")
	}
	if execution.Status != ExecutionStatusFailed || execution.Reason != ExecutionReasonAdapterReturnedError {
		t.Fatalf("unexpected execution: %#v", execution)
	}
	if execution.HostMutation || execution.SemanticMemoryWrite || execution.ModelruntimeMutation {
		t.Fatalf("mutation flags must remain false: %#v", execution)
	}
}

func TestResourceActionExecutorIdempotentByProposalID(t *testing.T) {
	executor, adapters := testExecutor()
	proposal := approvedProposalForAction(ProposalActionWarnOperator)
	first, err := executor.Execute(proposal)
	if err != nil {
		t.Fatal(err)
	}
	second, err := executor.Execute(proposal)
	if err != nil {
		t.Fatal(err)
	}
	if first.ExecutionID != second.ExecutionID {
		t.Fatalf("execution id changed: %q vs %q", first.ExecutionID, second.ExecutionID)
	}
	if adapters.notifier.calls != 1 {
		t.Fatalf("adapter called more than once: %d", adapters.notifier.calls)
	}
	if len(executor.Executions()) != 1 {
		t.Fatalf("expected one execution record, got %#v", executor.Executions())
	}
}

func TestResourceActionExecutionJSONIncludesGovernanceFields(t *testing.T) {
	executor, _ := testExecutor()
	execution, err := executor.Execute(approvedProposalForAction(ProposalActionDeferBackgroundIngest))
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(execution)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"execution_id", "proposal_id", "source_policy_id", "source_snapshot_id", "action_type", "target_lane", "status", "started_at", "finished_at", "result", "reason", "side_effects", "operator_approval_required", "approved_before_execution", "bounded", "host_mutation", "semantic_memory_write", "modelruntime_mutation", "errors"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q in %s", key, string(body))
		}
	}
}

func TestResourceActionExecutionIDIsDeterministicForProposal(t *testing.T) {
	proposal := approvedProposalForAction(ProposalActionWarnOperator)
	leftExecutor, _ := testExecutor()
	rightExecutor, _ := testExecutor()
	left, err := leftExecutor.Execute(proposal)
	if err != nil {
		t.Fatal(err)
	}
	right, err := rightExecutor.Execute(proposal)
	if err != nil {
		t.Fatal(err)
	}
	if left.ExecutionID == "" || left.ExecutionID != right.ExecutionID {
		t.Fatalf("execution ids differ: %#v %#v", left, right)
	}
}

func approvedProposalForAction(action string) ResourceActionProposal {
	proposal := ResourceActionProposal{
		CreatedAt:                fixedNow(),
		SourcePolicyID:           "forgeh_policy_test",
		SourceSnapshotID:         "hostdiag_test",
		ActionType:               action,
		TargetLane:               targetLaneForAction(action),
		RecommendedDecision:      PolicyDecisionDefer,
		Reason:                   "test proposal",
		RiskLevel:                ProposalRiskModerate,
		RequiresOperatorApproval: true,
		Status:                   ProposalStatusApproved,
		ExpiresAt:                fixedNow().Add(2 * time.Hour),
		AdvisoryOnly:             true,
		DecisionReason:           "approved for test",
		DecidedAt:                fixedNow().Add(time.Minute),
	}
	proposal.ProposalID = proposalID(proposal)
	return proposal
}

func targetLaneForAction(action string) string {
	switch action {
	case ProposalActionDeferBackgroundIngest, ProposalActionPauseBackgroundIngest:
		return WorkloadLaneBackgroundIngest
	case ProposalActionDeferEmbedding:
		return WorkloadLaneEmbedding
	case ProposalActionScheduleMaintenance:
		return WorkloadLaneMaintenance
	case ProposalActionDenyNewModelLoad, ProposalActionDeferLargeModelLoad, ProposalActionPreferCurrentModelOnly, ProposalActionPreferCPUSafeMode:
		return WorkloadLaneModelLoad
	default:
		return ""
	}
}

type testAdapters struct {
	notifier *testNotifier
	lane     *testLanePolicy
	model    *testModelPolicy
	degraded *testDegradedMode
}

func testExecutor() (*ResourceActionExecutor, testAdapters) {
	adapters := newTestAdapters()
	executor := NewResourceActionExecutor(ExecutorOptions{
		Now:          fixedNow,
		Notifier:     adapters.notifier,
		LanePolicy:   adapters.lane,
		ModelPolicy:  adapters.model,
		DegradedMode: adapters.degraded,
	})
	return executor, adapters
}

func newTestAdapters() testAdapters {
	return testAdapters{
		notifier: &testNotifier{},
		lane:     &testLanePolicy{},
		model:    &testModelPolicy{},
		degraded: &testDegradedMode{},
	}
}

func (a testAdapters) totalCalls() int {
	return a.notifier.calls + a.lane.calls + a.model.calls + a.degraded.calls
}

type testNotifier struct {
	calls int
	err   error
}

func (a *testNotifier) NotifyOperator(ResourceActionProposal) ([]string, error) {
	a.calls++
	if a.err != nil {
		return nil, a.err
	}
	return []string{SideEffectOperatorNotificationRecorded}, nil
}

type testLanePolicy struct {
	calls int
}

func (a *testLanePolicy) DeferBackgroundIngest(ResourceActionProposal) ([]string, error) {
	a.calls++
	return []string{SideEffectBackgroundIngestDeferred}, nil
}

func (a *testLanePolicy) PauseBackgroundIngest(ResourceActionProposal) ([]string, error) {
	a.calls++
	return []string{SideEffectBackgroundIngestPaused}, nil
}

func (a *testLanePolicy) DeferEmbedding(ResourceActionProposal) ([]string, error) {
	a.calls++
	return []string{SideEffectEmbeddingDeferred}, nil
}

func (a *testLanePolicy) ScheduleMaintenanceLater(ResourceActionProposal) ([]string, error) {
	a.calls++
	return []string{SideEffectMaintenanceScheduledLater}, nil
}

type testModelPolicy struct {
	calls int
}

func (a *testModelPolicy) DenyNewModelLoad(ResourceActionProposal) ([]string, error) {
	a.calls++
	return []string{SideEffectNewModelLoadsDenied}, nil
}

func (a *testModelPolicy) DeferLargeModelLoad(ResourceActionProposal) ([]string, error) {
	a.calls++
	return []string{SideEffectLargeModelLoadsDeferred}, nil
}

func (a *testModelPolicy) PreferCurrentModelOnly(ResourceActionProposal) ([]string, error) {
	a.calls++
	return []string{SideEffectCurrentModelOnlyPreferred}, nil
}

func (a *testModelPolicy) PreferCPUSafeMode(ResourceActionProposal) ([]string, error) {
	a.calls++
	return []string{SideEffectCPUSafeModePreferred}, nil
}

type testDegradedMode struct {
	calls int
}

func (a *testDegradedMode) EnterDegradedMode(ResourceActionProposal) ([]string, error) {
	a.calls++
	return []string{SideEffectDegradedModeFlagRecorded}, nil
}
