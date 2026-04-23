package autonomy

import (
	"context"
	"fmt"
	"testing"

	"forge/projectforge/services/core/internal/aios/compute/librarian"
	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/truth"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/store"
)

type allowAllApprovalGate struct{}

func (allowAllApprovalGate) Evaluate(_ context.Context, _ domain.SyscallRequest, _ controllane.ActionDefinition) (controllane.ApprovalDecision, error) {
	return controllane.ApprovalDecision{Status: domain.ApprovalAllowed, Reason: "test allow"}, nil
}

type autonomyHarness struct {
	t           *testing.T
	store       *store.Store
	now         int64
	scope       domain.ForgeScope
	kernel      *controllane.Processor
	truth       *truth.Engine
	bundle      InMemoryBundle
	intentQueue *IntentQueueService
	budget      *FreedomBudgetService
	evaluator   *AutonomyPolicyEvaluator
	runner      *SelfInitiatedSyscallRunner
	explainer   *ExplanationService
	noteRepo    controllane.MemoryNoteRepository
	loopRepo    controllane.OpenLoopRepository
	stateRepo   controllane.StateRepository
}

func newAutonomyHarness(t *testing.T) *autonomyHarness {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	h := &autonomyHarness{
		t:     t,
		store: st,
		now:   1764000000000,
		scope: domain.ForgeScope{WorkspaceID: "ws-autonomy", LaneID: "control.semantic"},
	}
	h.noteRepo = controllane.NewSQLiteMemoryNoteRepository(st.DB)
	h.loopRepo = controllane.NewSQLiteOpenLoopRepository(st.DB)
	h.stateRepo = controllane.NewSQLiteStateRepository(st.DB)
	txRunner := controllane.NewSQLiteTransactionRunner(st.DB)
	h.kernel = controllane.NewProcessor(controllane.ProcessorOptions{
		Registry:     controllane.NewStaticActionRegistry(),
		Validator:    controllane.NewDeterministicValidator(),
		Capabilities: controllane.NewStaticCapabilityService(),
		ApprovalGate: allowAllApprovalGate{},
		TxRunner:     txRunner,
		AuditSink:    controllane.NewCoreAuditSink(audit.New(st.DB)),
		NowMillis:    h.nextMillis,
	})
	h.truth = truth.NewEngine(truth.EngineOptions{
		Kernel: h.kernel,
		Repositories: truth.Repositories{
			State:         h.stateRepo,
			Loops:         h.loopRepo,
			Notes:         h.noteRepo,
			Models:        controllane.NewSQLiteDerivedModelRepository(st.DB),
			Contradiction: controllane.NewSQLiteContradictionRepository(st.DB),
			Supersession:  controllane.NewSQLiteSupersessionRepository(st.DB),
		},
		NowMillis: h.nextMillis,
	})

	h.bundle = NewInMemoryBundle()
	h.intentQueue = NewIntentQueueService(h.bundle.Intents, h.nextMillis)
	h.budget = NewFreedomBudgetService(h.bundle.Budgets, h.bundle.Reservations, h.nextMillis)
	h.evaluator = NewAutonomyPolicyEvaluator(PolicyEvaluatorOptions{
		Charters:       h.bundle.Charters,
		Risk:           NewDeterministicRiskClassifier(),
		Budgets:        h.budget,
		Kernel:         h.kernel,
		NowMillis:      h.nextMillis,
		CharterService: NewCharterService(),
		HasActiveStateDependency: func(scope domain.ForgeScope, objectID string) bool {
			rows, err := h.stateRepo.ListCurrent(context.Background(), controllane.ScopeFilter{WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID}, 200)
			if err != nil {
				return false
			}
			for _, row := range rows {
				for _, id := range row.DerivedFrom {
					if id == objectID {
						return true
					}
				}
			}
			return false
		},
	})
	h.runner = NewSelfInitiatedSyscallRunner(SelfInitiatedRunnerOptions{
		Kernel:    h.kernel,
		Policy:    h.evaluator,
		Intents:   h.intentQueue,
		Decisions: h.bundle.Decisions,
		Budgets:   h.budget,
		Approval:  NewStaticApprovalEscalator(),
		NowMillis: h.nextMillis,
	})
	h.explainer = NewExplanationService(h.bundle.Intents, h.bundle.Decisions, h.bundle.Budgets, h.bundle.Charters)

	for _, budget := range DefaultBudgets(h.scope, h.nextMillis(), "tester") {
		if err := h.bundle.Budgets.Create(context.Background(), budget); err != nil {
			t.Fatalf("create default budget: %v", err)
		}
	}
	for _, charter := range DefaultCharters(h.scope, h.nextMillis(), "tester") {
		if err := h.bundle.Charters.Create(context.Background(), charter); err != nil {
			t.Fatalf("create default charter: %v", err)
		}
	}
	return h
}

func (h *autonomyHarness) nextMillis() int64 {
	h.now += 17
	return h.now
}

func (h *autonomyHarness) newIntent(id string, source domain.IntentSource, charterID string) domain.AutonomyIntent {
	now := h.nextMillis()
	return domain.AutonomyIntent{
		ID:            id,
		Type:          domain.IntentSelfMaintenance,
		Title:         "autonomy test intent",
		Description:   "test intent",
		Source:        source,
		ProposedBy:    "autonomy_test",
		Scope:         h.scope,
		Status:        domain.IntentStatusProposed,
		Risk:          domain.AutonomyRiskLow,
		AutonomyLevel: domain.AutonomyLevelAutoCommitSafe,
		CharterID:     charterID,
		BudgetID:      "budget_memory_maintenance",
		Provenance:    domain.Provenance{Actor: "autonomy_test", ActorType: "test", Source: "test", TraceID: "trace-" + id},
		CorrelationID: "corr-" + id,
		TraceID:       "trace-" + id,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func createSystemNoteAction(scope domain.ForgeScope, id string, now int64) domain.SyscallRequest {
	return domain.SyscallRequest{
		ID:     "act-" + id,
		Action: domain.ActionCreateNote,
		Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
		Source: domain.SourceSystem,
		Scope:  scope,
		Payload: map[string]any{
			"id":         id,
			"type":       string(domain.NoteSystem),
			"title":      "Autonomy diagnostic note",
			"content":    "autonomy diagnostic",
			"confidence": 0.8,
			"status":     string(domain.NoteActive),
		},
		Provenance:    domain.Provenance{Actor: "forge.autonomy", ActorType: "system", Source: "autonomy", TraceID: "trace-" + id},
		CorrelationID: "corr-" + id,
		TraceID:       "trace-" + id,
		RequestedAt:   now,
	}
}

func TestRiskClassifierMapsLevelsAndRequiresApproval(t *testing.T) {
	h := newAutonomyHarness(t)
	intent := h.newIntent("risk-1", domain.IntentSourceForge, "charter_memory_maintenance")
	classifier := NewDeterministicRiskClassifier()

	low := classifier.ClassifyAction(intent, domain.SyscallRequest{Action: domain.ActionCreateLink, Scope: h.scope, Provenance: domain.Provenance{Actor: "a", ActorType: "t", Source: "s"}})
	if low.Risk != domain.AutonomyRiskLow || low.AutonomyLevel != domain.AutonomyLevelAutoCommitSafe {
		t.Fatalf("expected low risk mapping, got %+v", low)
	}

	high := classifier.ClassifyAction(intent, domain.SyscallRequest{Action: domain.SemanticActionType("DELETE_ARTIFACT"), Scope: h.scope, Provenance: domain.Provenance{Actor: "a", ActorType: "t", Source: "s"}})
	if high.Risk != domain.AutonomyRiskCritical || !high.RequiresApproval {
		t.Fatalf("expected critical risk and approval, got %+v", high)
	}

	cross := classifier.ClassifyAction(intent, domain.SyscallRequest{Action: domain.ActionCreateNote, Scope: domain.ForgeScope{WorkspaceID: "ws-other", LaneID: h.scope.LaneID}, Provenance: domain.Provenance{Actor: "a", ActorType: "t", Source: "s"}})
	if cross.Risk != domain.AutonomyRiskCritical {
		t.Fatalf("expected cross-workspace risk escalation, got %+v", cross)
	}
}

func TestPolicyEvaluatorRespectsCharterStatusAndDeniedActions(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	if err := h.bundle.Charters.UpdateStatus(ctx, "charter_memory_maintenance", domain.CharterSuspended, h.nextMillis(), nil); err != nil {
		t.Fatalf("suspend charter: %v", err)
	}
	intent := h.newIntent("charter-suspended", domain.IntentSourceForge, "charter_memory_maintenance")
	action := createSystemNoteAction(h.scope, "note-suspended", h.nextMillis())
	decision, err := h.evaluator.Evaluate(ctx, EvaluationInput{Intent: intent, Actions: []domain.SyscallRequest{action}, Mode: domain.AutonomyModeMaintain})
	if err != nil {
		t.Fatalf("evaluate suspended charter: %v", err)
	}
	if decision.Decision != domain.DecisionBlockedByCharter {
		t.Fatalf("expected blocked_by_charter for suspended charter, got %+v", decision)
	}

	active, ok, err := h.bundle.Charters.GetByID(ctx, "charter_context_preparation")
	if err != nil || !ok {
		t.Fatalf("get charter_context_preparation: %v ok=%v", err, ok)
	}
	active.DeniedActions = append(active.DeniedActions, domain.ActionCreateNote)
	if err := h.bundle.Charters.Create(ctx, active); err == nil {
		// expected duplicate failure; use update via status+metadata approach by replacing internal map in in-memory repo.
	}
	if err := h.bundle.Charters.UpdateStatus(ctx, active.ID, domain.CharterActive, h.nextMillis(), map[string]any{"denied_override": true}); err != nil {
		t.Fatalf("update status: %v", err)
	}
	// replace row directly via create/update cycle by using in-memory semantics.
	h.bundle.Charters.store.mu.Lock()
	h.bundle.Charters.store.charters[active.ID] = active
	h.bundle.Charters.store.mu.Unlock()

	intent2 := h.newIntent("charter-deny", domain.IntentSourceForge, "charter_context_preparation")
	decision2, err := h.evaluator.Evaluate(ctx, EvaluationInput{Intent: intent2, Actions: []domain.SyscallRequest{action}, Mode: domain.AutonomyModeMaintain})
	if err != nil {
		t.Fatalf("evaluate denied override: %v", err)
	}
	if decision2.Decision != domain.DecisionBlockedByCharter {
		t.Fatalf("expected denied action override, got %+v", decision2)
	}
}

func TestIntentQueueLifecycleAndScopeIsolation(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	intentA := h.newIntent("intent-a", domain.IntentSourceForge, "charter_memory_maintenance")
	if _, err := h.intentQueue.Enqueue(ctx, intentA); err != nil {
		t.Fatalf("enqueue intent a: %v", err)
	}
	if err := h.intentQueue.Approve(ctx, intentA.ID); err != nil {
		t.Fatalf("approve intent a: %v", err)
	}
	if err := h.intentQueue.MarkRunning(ctx, intentA.ID); err != nil {
		t.Fatalf("mark running intent a: %v", err)
	}
	if err := h.intentQueue.MarkCompleted(ctx, intentA.ID, nil); err != nil {
		t.Fatalf("complete intent a: %v", err)
	}

	intentB := h.newIntent("intent-b", domain.IntentSourceForge, "charter_memory_maintenance")
	if _, err := h.intentQueue.Enqueue(ctx, intentB); err != nil {
		t.Fatalf("enqueue intent b: %v", err)
	}
	if err := h.intentQueue.Reject(ctx, intentB.ID, "rejected for test"); err != nil {
		t.Fatalf("reject intent b: %v", err)
	}
	if err := h.intentQueue.MarkRunning(ctx, intentB.ID); err == nil {
		t.Fatalf("expected terminal intent transition to fail")
	}

	active, err := h.intentQueue.ListActive(ctx, h.scope, 20)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("expected no active intents, got %d", len(active))
	}

	otherScope := domain.ForgeScope{WorkspaceID: "ws-other", LaneID: h.scope.LaneID}
	otherIntent := intentA
	otherIntent.ID = "intent-other"
	otherIntent.Scope = otherScope
	otherIntent.Status = domain.IntentStatusProposed
	if _, err := h.intentQueue.Enqueue(ctx, otherIntent); err != nil {
		t.Fatalf("enqueue other scope intent: %v", err)
	}
	rows, err := h.intentQueue.ListByStatus(ctx, h.scope, domain.IntentStatusProposed, 20)
	if err != nil {
		t.Fatalf("list by status scope: %v", err)
	}
	for _, row := range rows {
		if row.Scope.WorkspaceID != h.scope.WorkspaceID {
			t.Fatalf("scope isolation failed: %+v", row.Scope)
		}
	}
}

func TestBudgetServiceCheckConsumeAndDryRun(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	action := createSystemNoteAction(h.scope, "note-budget", h.nextMillis())
	check, err := h.budget.CheckBudget(ctx, BudgetCheckRequest{Scope: h.scope, BudgetID: "budget_memory_maintenance", IntentID: "intent-budget", RequestedFor: "self_maintenance", Actions: []domain.SyscallRequest{action}, DryRun: true})
	if err != nil {
		t.Fatalf("budget check dry-run: %v", err)
	}
	if !check.Allowed {
		t.Fatalf("expected dry-run budget check allowed: %+v", check)
	}

	reservation, err := h.budget.ReserveBudget(ctx, "budget_memory_maintenance", "intent-budget", h.scope, "self_maintenance", 1, nil)
	if err != nil {
		t.Fatalf("reserve budget: %v", err)
	}
	budget, err := h.budget.ConsumeBudget(ctx, reservation.ID)
	if err != nil {
		t.Fatalf("consume budget: %v", err)
	}
	if budget.Usage.CommittedActionsPeriod == 0 {
		t.Fatalf("expected committed usage increment")
	}

	budget.ResetsAt = h.nextMillis() - 1
	if err := h.bundle.Budgets.Update(ctx, budget); err != nil {
		t.Fatalf("set reset boundary: %v", err)
	}
	check2, err := h.budget.CheckBudget(ctx, BudgetCheckRequest{Scope: h.scope, BudgetID: budget.ID, IntentID: "intent-budget-2", RequestedFor: "self_maintenance", Actions: []domain.SyscallRequest{action}})
	if err != nil {
		t.Fatalf("budget check after reset: %v", err)
	}
	if !check2.Allowed {
		t.Fatalf("expected budget allowed after reset, got %+v", check2)
	}
}

func TestPolicyEvaluatorModesAndApprovalEscalation(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()
	intent := h.newIntent("policy-mode", domain.IntentSourceForge, "charter_memory_maintenance")
	action := createSystemNoteAction(h.scope, "note-policy", h.nextMillis())

	observe, err := h.evaluator.Evaluate(ctx, EvaluationInput{Intent: intent, Actions: []domain.SyscallRequest{action}, Mode: domain.AutonomyModeObserve})
	if err != nil {
		t.Fatalf("evaluate observe: %v", err)
	}
	if observe.Decision != domain.DecisionAllowProposeOnly {
		t.Fatalf("expected observe => propose_only, got %+v", observe)
	}

	highAction := domain.SyscallRequest{
		ID:          "act-close-high",
		Action:      domain.ActionCloseLoop,
		Actor:       domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
		Source:      domain.SourceSystem,
		Scope:       h.scope,
		Payload:     map[string]any{"loopId": "loop-1", "priority": "high", "reason": "auto close"},
		Provenance:  domain.Provenance{Actor: "forge.autonomy", ActorType: "system", Source: "autonomy", TraceID: "trace-close"},
		RequestedAt: h.nextMillis(),
	}
	highIntent := h.newIntent("policy-high", domain.IntentSourceForge, "charter_open_loop_review")
	high, err := h.evaluator.Evaluate(ctx, EvaluationInput{Intent: highIntent, Actions: []domain.SyscallRequest{highAction}, Mode: domain.AutonomyModeMaintain})
	if err != nil {
		t.Fatalf("evaluate high risk: %v", err)
	}
	if high.Decision != domain.DecisionApprovalRequired {
		t.Fatalf("expected high risk approval_required, got %+v", high)
	}
}

func TestRunnerQuarantinesAutoCommitWithoutDurableBacking(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()
	intent := h.newIntent("runner-commit", domain.IntentSourceForge, "charter_memory_maintenance")
	action := createSystemNoteAction(h.scope, "note-runner-commit", h.nextMillis())

	summary, err := h.runner.Run(ctx, intent, []domain.SyscallRequest{action}, domain.RunModeCommitIfAuthorized, domain.AutonomyModeMaintain)
	if err != nil {
		t.Fatalf("runner execution failed: %v", err)
	}
	if summary.Decision == domain.DecisionAllowAutoCommit {
		t.Fatalf("expected persistence gate to block auto-commit, got %+v", summary)
	}
	if len(summary.CommittedObjectIDs) != 0 {
		t.Fatalf("expected no committed object ids in quarantine mode, got %+v", summary.CommittedObjectIDs)
	}
	if _, ok, err := h.noteRepo.GetByID(ctx, "note-runner-commit"); err != nil {
		t.Fatalf("expected note lookup success err=%v", err)
	} else if ok {
		t.Fatalf("expected no committed note in quarantine mode")
	}
	explain, err := h.explainer.ExplainIntent(ctx, intent.ID)
	if err != nil {
		t.Fatalf("explain intent: %v", err)
	}
	if _, ok := explain["decisions"]; !ok {
		t.Fatalf("expected decision trace in explainIntent")
	}
}

func TestRunnerWithoutCharterDoesNotAutoCommit(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()
	intent := h.newIntent("runner-no-charter", domain.IntentSourceForge, "")
	intent.BudgetID = ""
	action := createSystemNoteAction(h.scope, "note-no-charter", h.nextMillis())

	summary, err := h.runner.Run(ctx, intent, []domain.SyscallRequest{action}, domain.RunModeCommitIfAuthorized, domain.AutonomyModeMaintain)
	if err != nil {
		t.Fatalf("runner no charter returned error: %v", err)
	}
	if summary.Decision == domain.DecisionAllowAutoCommit {
		t.Fatalf("expected no auto-commit without charter, got %+v", summary)
	}
	if _, ok, err := h.noteRepo.GetByID(ctx, "note-no-charter"); err != nil {
		t.Fatalf("get note no charter: %v", err)
	} else if ok {
		t.Fatalf("unexpected commit without charter")
	}
}

func TestRuleAgentIntegrationStaleLoopIntent(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	openReq := domain.SyscallRequest{
		ID:          "seed-loop",
		Action:      domain.ActionOpenLoop,
		Actor:       domain.ActorIdentity{ID: "seed", Kind: "test"},
		Source:      domain.SourceTest,
		Scope:       h.scope,
		Payload:     map[string]any{"id": "loop-stale-1", "title": "Stale loop", "state": string(domain.LoopOpen), "priority": "medium"},
		Provenance:  domain.Provenance{Actor: "seed", ActorType: "test", Source: "test", TraceID: "trace-seed-loop"},
		RequestedAt: h.nextMillis(),
	}
	if res, err := h.kernel.Process(ctx, openReq); err != nil || !res.Success {
		t.Fatalf("seed loop failed err=%v res=%+v", err, res)
	}

	runtime := NewRuleAgentRuntime([]RuleAgent{OpenLoopStalenessAgent{StaleCutoffMillis: h.nextMillis() + 1000}}, h.runner, domain.AutonomyModeMaintain, h.nextMillis)
	input := BuildRuleAgentInput(h.scope, "corr-rule", "trace-rule", h.truth, 0, "test", h.nextMillis())
	runs, err := runtime.RunOnce(ctx, input)
	if err != nil {
		t.Fatalf("rule runtime run once: %v", err)
	}
	if len(runs) == 0 {
		t.Fatalf("expected at least one autonomy run from staleness agent")
	}
	foundProposeOnly := false
	for _, run := range runs {
		if run.Decision == domain.DecisionAllowProposeOnly && len(run.CommittedObjectIDs) == 0 {
			foundProposeOnly = true
		}
	}
	if !foundProposeOnly {
		t.Fatalf("expected staleness agent to run in propose-only mode")
	}
}

func TestIngestIntegrationDepthCap(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := int64(1765000000000)
	nextNow := func() int64 { now += 10; return now }
	kernel := controllane.NewProcessor(controllane.ProcessorOptions{
		Registry:     controllane.NewStaticActionRegistry(),
		Validator:    controllane.NewDeterministicValidator(),
		Capabilities: controllane.NewStaticCapabilityService(),
		ApprovalGate: allowAllApprovalGate{},
		TxRunner:     controllane.NewSQLiteTransactionRunner(st.DB),
		AuditSink:    controllane.NewCoreAuditSink(audit.New(st.DB)),
		NowMillis:    nextNow,
	})
	repos := librarian.CellReadRepositories{
		Journal:        controllane.NewSQLiteJournalRepository(st.DB),
		Notes:          controllane.NewSQLiteMemoryNoteRepository(st.DB),
		Links:          controllane.NewSQLiteSemanticLinkRepository(st.DB),
		State:          controllane.NewSQLiteStateRepository(st.DB),
		Loops:          controllane.NewSQLiteOpenLoopRepository(st.DB),
		Artifacts:      controllane.NewSQLiteArtifactRefRepository(st.DB),
		Models:         controllane.NewSQLiteDerivedModelRepository(st.DB),
		Contradictions: controllane.NewSQLiteContradictionRepository(st.DB),
		Supersessions:  controllane.NewSQLiteSupersessionRepository(st.DB),
		ContextPackets: controllane.NewSQLiteContextPacketRepository(st.DB),
	}
	calls := 0
	pipeline := librarian.NewIngestPipeline(librarian.IngestPipelineOptions{
		Kernel:       kernel,
		Repositories: repos,
		NowMillis:    nextNow,
		AutonomyPass: func(_ context.Context, _ domain.IngestRequest, _ domain.IngestResult, _ truth.TruthEngine, _ int) ([]domain.AutonomyRunSummary, error) {
			calls++
			return []domain.AutonomyRunSummary{{IntentID: "intent-ingest", Decision: domain.DecisionAllowProposeOnly}}, nil
		},
		MaxAutonomyDepth: 1,
	})

	req := domain.IngestRequest{
		ID:            "ingest-autonomy-depth",
		InputKind:     domain.IngestUserMessage,
		Content:       "hello",
		Actor:         domain.ActorIdentity{ID: "operator", Kind: string(domain.SourceUser)},
		Source:        domain.SourceUser,
		Scope:         domain.ForgeScope{WorkspaceID: "ws-autonomy-depth", LaneID: "control.semantic"},
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "test"},
		CorrelationID: "corr-ingest-depth",
		TraceID:       "trace-ingest-depth",
		CommitMode:    domain.IngestCommitValid,
		RequestedAt:   nextNow(),
		Metadata:      map[string]any{"autonomyDepth": 1},
	}
	res, err := pipeline.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("pipeline run with depth cap: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected autonomy callback to be skipped due depth cap")
	}
	if len(res.AutonomyRuns) != 0 {
		t.Fatalf("expected no autonomy runs in result when depth capped")
	}
}

func TestCuriosityQueuePromoteToIntentDoesNotCommit(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()
	queue := NewCuriosityQueueService(h.bundle.Curiosity, h.intentQueue, h.nextMillis)
	item, err := queue.Add(ctx, domain.CuriosityItem{
		ID:        "curiosity-1",
		Title:     "Investigate stale policies",
		Question:  "Should we revise context policy ranking?",
		Source:    domain.IntentSourceForge,
		Scope:     h.scope,
		Evidence:  []string{"note-a", "note-b"},
		Priority:  "medium",
		Status:    domain.CuriosityOpen,
		CreatedAt: h.nextMillis(),
		UpdatedAt: h.nextMillis(),
	})
	if err != nil {
		t.Fatalf("add curiosity item: %v", err)
	}
	intent, err := queue.PromoteToIntent(ctx, item.ID, h.newIntent("intent-from-curiosity", domain.IntentSourceForge, "charter_context_preparation"))
	if err != nil {
		t.Fatalf("promote curiosity to intent: %v", err)
	}
	if intent.ID == "" {
		t.Fatalf("expected promoted intent id")
	}
	if notes, err := h.noteRepo.ListByScope(ctx, controllane.ScopeFilter{WorkspaceID: h.scope.WorkspaceID, LaneID: h.scope.LaneID}); err != nil {
		t.Fatalf("list notes: %v", err)
	} else if len(notes) != 0 {
		t.Fatalf("curiosity promotion must not commit semantic writes")
	}
}

func TestFutureIrisIntentCannotAutoApproveItself(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()
	intent := h.newIntent("future-iris-intent", domain.IntentSourceFutureIRIS, "")
	action := domain.SyscallRequest{
		ID:     "act-future-iris",
		Action: domain.ActionUpdateState,
		Actor:  domain.ActorIdentity{ID: "iris.service", Kind: string(domain.SourceFutureIRIS)},
		Source: domain.SourceFutureIRIS,
		Scope:  h.scope,
		Payload: map[string]any{
			"id":    "state-future-iris",
			"key":   "architecture_direction",
			"value": map[string]any{"value": "risky"},
		},
		Provenance:    domain.Provenance{Actor: "iris.service", ActorType: "future_iris", Source: "future_iris", TraceID: "trace-future-iris"},
		CorrelationID: "corr-future-iris",
		TraceID:       "trace-future-iris",
		RequestedAt:   h.nextMillis(),
	}
	summary, err := h.runner.Run(ctx, intent, []domain.SyscallRequest{action}, domain.RunModeCommitIfAuthorized, domain.AutonomyModeMaintain)
	if err != nil {
		t.Fatalf("future iris run error: %v", err)
	}
	if summary.Decision == domain.DecisionAllowAutoCommit {
		t.Fatalf("future_iris should not auto-commit without charter, got %+v", summary)
	}
	if _, ok, err := h.stateRepo.GetCurrent(ctx, "architecture_direction", controllane.ScopeFilter{WorkspaceID: h.scope.WorkspaceID, LaneID: h.scope.LaneID}); err != nil {
		t.Fatalf("state lookup failed: %v", err)
	} else if ok {
		t.Fatalf("future_iris action committed without charter")
	}
}

func TestBudgetAndDecisionExplanationIsDeterministic(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()
	intent := h.newIntent("intent-explain", domain.IntentSourceForge, "charter_memory_maintenance")
	action := createSystemNoteAction(h.scope, "note-explain", h.nextMillis())
	if _, err := h.runner.Run(ctx, intent, []domain.SyscallRequest{action}, domain.RunModeCommitIfAuthorized, domain.AutonomyModeMaintain); err != nil {
		t.Fatalf("runner execute: %v", err)
	}
	intentExplain, err := h.explainer.ExplainIntent(ctx, intent.ID)
	if err != nil {
		t.Fatalf("explain intent: %v", err)
	}
	if _, ok := intentExplain["decisions"]; !ok {
		t.Fatalf("expected decisions in intent explanation")
	}
	budgetExplain, err := h.explainer.ExplainBudgetUsage(ctx, "budget_memory_maintenance")
	if err != nil {
		t.Fatalf("explain budget: %v", err)
	}
	if fmt.Sprintf("%v", budgetExplain["budgetId"]) != "budget_memory_maintenance" {
		t.Fatalf("unexpected budget explanation: %+v", budgetExplain)
	}
}
