package truth

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/store"
)

type allowAllGate struct{}

func (allowAllGate) Evaluate(_ context.Context, _ domain.SyscallRequest, _ controllane.ActionDefinition) (controllane.ApprovalDecision, error) {
	return controllane.ApprovalDecision{Status: domain.ApprovalAllowed, Reason: "test override"}, nil
}

type truthHarness struct {
	t      *testing.T
	store  *store.Store
	kernel *controllane.Processor
	engine *Engine
	now    int64
	scope  domain.ForgeScope
	repos  Repositories
}

func newTruthHarness(t *testing.T, approval controllane.ApprovalGate) *truthHarness {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	txRunner := controllane.NewSQLiteTransactionRunner(st.DB)
	kernel := controllane.NewProcessor(controllane.ProcessorOptions{
		Registry:     controllane.NewStaticActionRegistry(),
		Validator:    controllane.NewDeterministicValidator(),
		Capabilities: controllane.NewStaticCapabilityService(),
		ApprovalGate: approval,
		TxRunner:     txRunner,
		AuditSink:    controllane.NewCoreAuditSink(audit.New(st.DB)),
		NowMillis:    func() int64 { return 1763000000000 },
	})
	repos := Repositories{
		State:         controllane.NewSQLiteStateRepository(st.DB),
		Loops:         controllane.NewSQLiteOpenLoopRepository(st.DB),
		Notes:         controllane.NewSQLiteMemoryNoteRepository(st.DB),
		Models:        controllane.NewSQLiteDerivedModelRepository(st.DB),
		Contradiction: controllane.NewSQLiteContradictionRepository(st.DB),
		Supersession:  controllane.NewSQLiteSupersessionRepository(st.DB),
	}
	h := &truthHarness{
		t:      t,
		store:  st,
		kernel: kernel,
		repos:  repos,
		now:    1763000001000,
		scope:  domain.ForgeScope{WorkspaceID: "ws-truth", LaneID: "control.semantic"},
	}
	h.engine = NewEngine(EngineOptions{
		Kernel:       kernel,
		Repositories: repos,
		NowMillis:    h.nextMillis,
	})
	return h
}

func (h *truthHarness) nextMillis() int64 {
	h.now += 13
	return h.now
}

func (h *truthHarness) syscall(ctx context.Context, action domain.SemanticActionType, id string, payload map[string]any, source domain.ActionSource, scope domain.ForgeScope) domain.SyscallResult {
	req := domain.SyscallRequest{
		ID:          id,
		Action:      action,
		Actor:       domain.ActorIdentity{ID: "tester", Kind: string(source)},
		Source:      source,
		Scope:       scope,
		Payload:     payload,
		RequestedAt: h.nextMillis(),
		Provenance: domain.Provenance{
			Actor:     "tester",
			ActorType: "test",
			Source:    "truth_test",
			TraceID:   "trace-" + id,
		},
		CorrelationID: "corr-" + id,
		TraceID:       "trace-" + id,
	}
	res, err := h.kernel.Process(ctx, req)
	if err != nil {
		h.t.Fatalf("syscall %s failed: %v", action, err)
	}
	return res
}

func TestStateProjectionTimelineAndExplanation(t *testing.T) {
	h := newTruthHarness(t, allowAllGate{})
	ctx := context.Background()

	noteRes := h.syscall(ctx, domain.ActionCreateNote, "note-policy-a", map[string]any{
		"id":      "note-policy-a",
		"type":    string(domain.NotePreference),
		"title":   "Preference A",
		"content": "Prefer transcript replay",
	}, domain.SourceUser, h.scope)
	if !noteRes.Success {
		t.Fatalf("seed note failed")
	}
	h.syscall(ctx, domain.ActionUpdateState, "state-1", map[string]any{
		"id":          "state-policy",
		"key":         "context_policy",
		"value":       map[string]any{"value": "transcript_replay"},
		"derivedFrom": []string{"note-policy-a"},
	}, domain.SourceUser, h.scope)
	h.syscall(ctx, domain.ActionUpdateState, "state-2", map[string]any{
		"id":          "state-policy-2",
		"key":         "context_policy",
		"value":       map[string]any{"value": "structured_snapshot"},
		"derivedFrom": []string{"note-policy-a"},
	}, domain.SourceUser, h.scope)

	current, ok, err := h.engine.GetCurrentState(ctx, "context_policy", h.scope)
	if err != nil || !ok {
		t.Fatalf("expected current state, err=%v ok=%v", err, ok)
	}
	if current.Value["value"] != "structured_snapshot" {
		t.Fatalf("expected latest value, got %v", current.Value["value"])
	}
	timeline, err := h.engine.GetStateTimeline(ctx, "context_policy", h.scope, 20)
	if err != nil {
		t.Fatalf("timeline failed: %v", err)
	}
	if len(timeline) < 2 {
		t.Fatalf("expected at least 2 timeline entries")
	}
	explain, err := h.engine.ExplainState(ctx, "context_policy", h.scope)
	if err != nil {
		t.Fatalf("explain state failed: %v", err)
	}
	if explain.CurrentValue["value"] != "structured_snapshot" {
		t.Fatalf("unexpected explained current value: %v", explain.CurrentValue["value"])
	}
	if len(explain.PreviousValues) == 0 {
		t.Fatalf("expected previous values in explanation")
	}

	scopeB := domain.ForgeScope{WorkspaceID: "ws-other", LaneID: "control.semantic"}
	h.syscall(ctx, domain.ActionUpdateState, "state-ws-b", map[string]any{
		"id":          "state-ws-b",
		"key":         "context_policy",
		"value":       map[string]any{"value": "workspace_b_value"},
		"derivedFrom": []string{"note-policy-a"},
	}, domain.SourceUser, scopeB)
	currentB, ok, err := h.engine.GetCurrentState(ctx, "context_policy", scopeB)
	if err != nil || !ok {
		t.Fatalf("expected workspace B state")
	}
	if currentB.Value["value"] != "workspace_b_value" {
		t.Fatalf("scope isolation failed, got %v", currentB.Value["value"])
	}
}

func TestOpenLoopLifecycleAndStaleDetection(t *testing.T) {
	h := newTruthHarness(t, allowAllGate{})
	ctx := context.Background()

	base := LoopMutationRequest{
		MutationBase: MutationBase{
			Source:      domain.SourceSystem,
			Scope:       h.scope,
			RequestedAt: h.nextMillis(),
		},
		LoopID:      "loop-lifecycle-1",
		Title:       "Schema support",
		Priority:    "high",
		Owner:       "operator",
		Blocker:     "missing schema support",
		CreatedFrom: "event-loop-1",
	}
	if res, err := h.engine.OpenLoop(ctx, base); err != nil || !res.Success {
		t.Fatalf("open loop failed err=%v res=%+v", err, res)
	}
	blockReq := base
	blockReq.NextState = domain.LoopBlocked
	blockReq.Reason = "blocked by dependency"
	if res, err := h.engine.BlockLoop(ctx, blockReq); err != nil || !res.Success {
		t.Fatalf("block loop failed err=%v res=%+v", err, res)
	}
	progressReq := base
	progressReq.NextState = domain.LoopInProgress
	progressReq.Reason = "work resumed"
	if res, err := h.engine.TransitionLoop(ctx, progressReq); err != nil || !res.Success {
		t.Fatalf("in_progress transition failed err=%v res=%+v", err, res)
	}
	resolveReq := base
	resolveReq.Reason = "dependency landed"
	resolveReq.Outcome = "schema support is resolved"
	if res, err := h.engine.ResolveLoop(ctx, resolveReq); err != nil || !res.Success {
		t.Fatalf("resolve loop failed err=%v res=%+v", err, res)
	}
	archiveReq := base
	archiveReq.NextState = domain.LoopArchived
	archiveReq.Reason = "completed and archived"
	if res, err := h.engine.ArchiveLoop(ctx, archiveReq); err != nil || !res.Success {
		t.Fatalf("archive loop failed err=%v res=%+v", err, res)
	}
	reopenReq := base
	reopenReq.NextState = domain.LoopOpen
	res, err := h.engine.ReopenLoop(ctx, reopenReq)
	if err != nil {
		t.Fatalf("reopen loop syscall error: %v", err)
	}
	if res.Success || res.DeterministicErrCode != domain.ErrInvalidStateTransition {
		t.Fatalf("expected archived->open invalid transition, got %+v", res)
	}
	stale, err := h.engine.ListStaleLoops(ctx, h.scope, h.nextMillis()-1, 20)
	if err != nil {
		t.Fatalf("stale list failed: %v", err)
	}
	for _, loop := range stale {
		if loop.ID == "loop-lifecycle-1" {
			t.Fatalf("resolved/archived loop should not appear in stale list")
		}
	}

	if _, err := h.engine.OpenLoop(ctx, LoopMutationRequest{
		MutationBase: MutationBase{
			Source:      domain.SourceSystem,
			Scope:       h.scope,
			RequestedAt: h.nextMillis(),
		},
		LoopID:   "loop-priority-owner",
		Title:    "Owner filtered loop",
		Priority: "high",
		Owner:    "owner-a",
	}); err != nil {
		t.Fatalf("open second loop failed: %v", err)
	}
	byPriority, err := h.engine.ListLoopsByPriority(ctx, h.scope, "high", 20)
	if err != nil || len(byPriority) == 0 {
		t.Fatalf("expected loops by priority err=%v count=%d", err, len(byPriority))
	}
	byOwner, err := h.engine.ListLoopsByOwner(ctx, h.scope, "owner-a", 20)
	if err != nil || len(byOwner) == 0 {
		t.Fatalf("expected loops by owner err=%v count=%d", err, len(byOwner))
	}
}

func TestContradictionSupersessionAndResolver(t *testing.T) {
	h := newTruthHarness(t, allowAllGate{})
	ctx := context.Background()

	h.syscall(ctx, domain.ActionCreateNote, "note-a", map[string]any{
		"id":      "note-a",
		"type":    string(domain.NotePreference),
		"title":   "Preference A",
		"content": "prefer transcript replay",
	}, domain.SourceUser, h.scope)
	h.syscall(ctx, domain.ActionCreateNote, "note-b", map[string]any{
		"id":      "note-b",
		"type":    string(domain.NotePreference),
		"title":   "Preference B",
		"content": "prefer structured snapshots",
	}, domain.SourceUser, h.scope)

	contrReq := ContradictionRequest{
		MutationBase: MutationBase{Source: domain.SourceSystem, Scope: h.scope, RequestedAt: h.nextMillis()},
		LeftObjectID: "note-a", LeftObjectKind: "memory_note",
		RightObjectID: "note-b", RightObjectKind: "memory_note",
		Reason: "instead of transcript replay",
	}
	if res, err := h.engine.RegisterContradiction(ctx, contrReq); err != nil || !res.Success {
		t.Fatalf("register contradiction failed err=%v res=%+v", err, res)
	}
	explainC, err := h.engine.ExplainContradiction(ctx, "truth-contradiction-"+hashParts("note-a", "note-b", "instead of transcript replay")+":contradiction")
	if err != nil {
		t.Fatalf("expected contradiction explanation: %v", err)
	}
	if explainC.LeftObjectID != "note-a" || explainC.RightObjectID != "note-b" {
		t.Fatalf("unexpected contradiction explanation: %+v", explainC)
	}

	supReq := SupersessionRequest{
		MutationBase: MutationBase{Source: domain.SourceSystem, Scope: h.scope, RequestedAt: h.nextMillis()},
		OldObjectID:  "note-a", OldObjectKind: "memory_note",
		NewObjectID: "note-b", NewObjectKind: "memory_note",
		Reason: "new preference",
	}
	if res, err := h.engine.MarkSuperseded(ctx, supReq); err != nil || !res.Success {
		t.Fatalf("mark superseded failed err=%v res=%+v", err, res)
	}

	resolved, err := h.engine.Resolve(ctx, "note-a", h.scope)
	if err != nil {
		t.Fatalf("resolve note-a failed: %v", err)
	}
	if resolved.Current || resolved.CurrentObjectID != "note-b" {
		t.Fatalf("expected superseded note to resolve to note-b, got %+v", resolved)
	}

	cycleReq := SupersessionRequest{
		MutationBase: MutationBase{Source: domain.SourceSystem, Scope: h.scope, RequestedAt: h.nextMillis()},
		OldObjectID:  "note-b", OldObjectKind: "memory_note",
		NewObjectID: "note-a", NewObjectKind: "memory_note",
		Reason: "cycle attempt",
	}
	cycleRes, err := h.engine.MarkSuperseded(ctx, cycleReq)
	if err != nil {
		t.Fatalf("cycle mark returned error: %v", err)
	}
	if cycleRes.Success || cycleRes.DeterministicErrCode != domain.ErrInvalidStateTransition {
		t.Fatalf("expected cycle rejection, got %+v", cycleRes)
	}

	otherScope := domain.ForgeScope{WorkspaceID: "ws-other", LaneID: "control.semantic"}
	h.syscall(ctx, domain.ActionCreateNote, "note-other", map[string]any{
		"id":      "note-other",
		"type":    string(domain.NotePreference),
		"title":   "Other",
		"content": "other ws",
	}, domain.SourceUser, otherScope)
	crossReq := SupersessionRequest{
		MutationBase: MutationBase{Source: domain.SourceSystem, Scope: h.scope, RequestedAt: h.nextMillis()},
		OldObjectID:  "note-b", OldObjectKind: "memory_note",
		NewObjectID: "note-other", NewObjectKind: "memory_note",
		Reason: "cross scope attempt",
	}
	crossRes, err := h.engine.MarkSuperseded(ctx, crossReq)
	if err != nil {
		t.Fatalf("cross scope mark returned error: %v", err)
	}
	if crossRes.Success || crossRes.DeterministicErrCode != domain.ErrInvalidScope {
		t.Fatalf("expected cross scope rejection, got %+v", crossRes)
	}
}

func TestModelLifecycleAndResolver(t *testing.T) {
	h := newTruthHarness(t, allowAllGate{})
	ctx := context.Background()

	h.syscall(ctx, domain.ActionCreateNote, "note-model-a", map[string]any{
		"id":      "note-model-a",
		"type":    string(domain.NoteFact),
		"title":   "Model evidence",
		"content": "evidence",
	}, domain.SourceUser, h.scope)
	create := h.syscall(ctx, domain.ActionDeriveModel, "model-create", map[string]any{
		"id":           "model-1",
		"type":         "context_policy_preference",
		"expression":   map[string]any{"formula": "prefer_structured_snapshots"},
		"derivedFrom":  []string{"note-model-a"},
		"supportCount": 1,
		"confidence":   0.7,
	}, domain.SourceSystem, h.scope)
	if !create.Success {
		t.Fatalf("model create failed: %+v", create)
	}
	promote := h.syscall(ctx, domain.ActionDeriveModel, "model-promote", map[string]any{
		"id":     "model-1",
		"status": string(domain.ModelPromoted),
	}, domain.SourceSystem, h.scope)
	if !promote.Success {
		t.Fatalf("model promote failed: %+v", promote)
	}
	deprecate := h.syscall(ctx, domain.ActionDeriveModel, "model-deprecate", map[string]any{
		"id":     "model-1",
		"status": string(domain.ModelDeprecated),
	}, domain.SourceSystem, h.scope)
	if !deprecate.Success {
		t.Fatalf("model deprecate failed: %+v", deprecate)
	}
	badPromote := h.syscall(ctx, domain.ActionDeriveModel, "model-promote-again", map[string]any{
		"id":     "model-1",
		"status": string(domain.ModelPromoted),
	}, domain.SourceSystem, h.scope)
	if badPromote.Success || badPromote.DeterministicErrCode != domain.ErrInvalidStateTransition {
		t.Fatalf("expected deprecated->promoted rejection, got %+v", badPromote)
	}
	resolution, err := h.engine.Resolve(ctx, "model-1", h.scope)
	if err != nil {
		t.Fatalf("resolve model failed: %v", err)
	}
	if !resolution.Deprecated || resolution.Current {
		t.Fatalf("expected deprecated model non-current resolution, got %+v", resolution)
	}
}

func TestRebuildProjectionDryRunDeterministic(t *testing.T) {
	h := newTruthHarness(t, allowAllGate{})
	ctx := context.Background()
	h.syscall(ctx, domain.ActionCreateNote, "note-rebuild", map[string]any{
		"id":      "note-rebuild",
		"type":    string(domain.NoteFact),
		"title":   "State evidence",
		"content": "state evidence",
	}, domain.SourceUser, h.scope)
	h.syscall(ctx, domain.ActionUpdateState, "state-rebuild", map[string]any{
		"id":          "state-rebuild",
		"key":         "runtime.mode",
		"value":       map[string]any{"value": "deterministic"},
		"derivedFrom": []string{"note-rebuild"},
	}, domain.SourceUser, h.scope)

	if _, err := h.store.DB.ExecContext(ctx, `
UPDATE state_items
SET value_json = '{"value":"corrupted"}'
WHERE workspace_id = ? AND lane_id = ? AND key = ?`, h.scope.WorkspaceID, h.scope.LaneID, "runtime.mode"); err != nil {
		t.Fatalf("corrupt state row: %v", err)
	}

	query := domain.TruthQuery{Scope: h.scope}
	report1, err := h.engine.RebuildProjection(ctx, query, true)
	if err != nil {
		t.Fatalf("rebuild dry-run failed: %v", err)
	}
	report2, err := h.engine.RebuildProjection(ctx, query, true)
	if err != nil {
		t.Fatalf("rebuild dry-run second failed: %v", err)
	}
	if len(report1.Differences) == 0 {
		t.Fatalf("expected rebuild differences")
	}
	if fmt.Sprintf("%v", report1.Differences) != fmt.Sprintf("%v", report2.Differences) {
		t.Fatalf("expected deterministic rebuild report")
	}
	current, ok, err := h.repos.State.GetCurrent(ctx, "runtime.mode", controllane.ScopeFilter{WorkspaceID: h.scope.WorkspaceID, LaneID: h.scope.LaneID})
	if err != nil || !ok {
		t.Fatalf("expected current state")
	}
	if current.Value["value"] != "corrupted" {
		t.Fatalf("dry-run rebuild should not mutate, got %v", current.Value["value"])
	}
}

func TestFutureIrisCannotBypassAndCanCommitWhenApproved(t *testing.T) {
	ctx := context.Background()

	deniedHarness := newTruthHarness(t, controllane.NewStaticApprovalGate())
	deniedReq := ContradictionRequest{
		MutationBase: MutationBase{
			Source:      domain.SourceFutureIRIS,
			Actor:       domain.ActorIdentity{ID: "iris.service", Kind: string(domain.SourceFutureIRIS)},
			Scope:       deniedHarness.scope,
			RequestedAt: deniedHarness.nextMillis(),
		},
		LeftObjectID: "missing-a", LeftObjectKind: "memory_note",
		RightObjectID: "missing-b", RightObjectKind: "memory_note",
		Reason: "future iris proposal",
	}
	deniedRes, err := deniedHarness.engine.RegisterContradiction(ctx, deniedReq)
	if err != nil {
		t.Fatalf("future_iris denied path errored: %v", err)
	}
	if deniedRes.Success || deniedRes.DeterministicErrCode != domain.ErrApprovalRequired {
		t.Fatalf("expected approval-required rejection for future_iris, got %+v", deniedRes)
	}

	allowedHarness := newTruthHarness(t, allowAllGate{})
	allowedHarness.syscall(ctx, domain.ActionCreateNote, "note-iris-a", map[string]any{
		"id": "note-iris-a", "type": string(domain.NoteFact), "title": "A", "content": "a",
	}, domain.SourceUser, allowedHarness.scope)
	allowedHarness.syscall(ctx, domain.ActionCreateNote, "note-iris-b", map[string]any{
		"id": "note-iris-b", "type": string(domain.NoteFact), "title": "B", "content": "b",
	}, domain.SourceUser, allowedHarness.scope)
	allowedReq := ContradictionRequest{
		MutationBase: MutationBase{
			Source:      domain.SourceFutureIRIS,
			Actor:       domain.ActorIdentity{ID: "iris.service", Kind: string(domain.SourceFutureIRIS)},
			Scope:       allowedHarness.scope,
			RequestedAt: allowedHarness.nextMillis(),
		},
		LeftObjectID: "note-iris-a", LeftObjectKind: "memory_note",
		RightObjectID: "note-iris-b", RightObjectKind: "memory_note",
		Reason: "future iris approved proposal",
	}
	allowedRes, err := allowedHarness.engine.RegisterContradiction(ctx, allowedReq)
	if err != nil || !allowedRes.Success {
		t.Fatalf("expected approved future_iris syscall commit, err=%v res=%+v", err, allowedRes)
	}
	rows, err := allowedHarness.engine.ListContradictionsForObject(ctx, "note-iris-a", allowedHarness.scope, 10)
	if err != nil || len(rows) == 0 {
		t.Fatalf("expected contradiction persisted through kernel path, err=%v count=%d", err, len(rows))
	}
}

func TestStateScopeLookupUsesLaneIsolation(t *testing.T) {
	h := newTruthHarness(t, allowAllGate{})
	ctx := context.Background()
	laneA := domain.ForgeScope{WorkspaceID: "ws-lane", LaneID: "lane-a"}
	laneB := domain.ForgeScope{WorkspaceID: "ws-lane", LaneID: "lane-b"}
	h.syscall(ctx, domain.ActionCreateNote, "note-lane-a", map[string]any{
		"id": "note-lane-a", "type": string(domain.NoteFact), "title": "A", "content": "A",
	}, domain.SourceUser, laneA)
	h.syscall(ctx, domain.ActionCreateNote, "note-lane-b", map[string]any{
		"id": "note-lane-b", "type": string(domain.NoteFact), "title": "B", "content": "B",
	}, domain.SourceUser, laneB)
	h.syscall(ctx, domain.ActionUpdateState, "state-lane-a", map[string]any{
		"id": "state-lane-a", "key": "same_key", "value": map[string]any{"value": "a"}, "derivedFrom": []string{"note-lane-a"},
	}, domain.SourceUser, laneA)
	h.syscall(ctx, domain.ActionUpdateState, "state-lane-b", map[string]any{
		"id": "state-lane-b", "key": "same_key", "value": map[string]any{"value": "b"}, "derivedFrom": []string{"note-lane-b"},
	}, domain.SourceUser, laneB)

	curA, ok, err := h.engine.GetCurrentState(ctx, "same_key", laneA)
	if err != nil || !ok {
		t.Fatalf("lane-a current state lookup failed")
	}
	curB, ok, err := h.engine.GetCurrentState(ctx, "same_key", laneB)
	if err != nil || !ok {
		t.Fatalf("lane-b current state lookup failed")
	}
	if curA.Value["value"] == curB.Value["value"] {
		t.Fatalf("expected lane isolation for state key")
	}

	if _, err := h.engine.Resolve(ctx, "note-lane-a", laneB); err == nil {
		t.Fatalf("expected scope-isolated resolve to fail for cross-lane object lookup")
	}
}

func TestEngineUsesStructuredTruthErrorForMissingScope(t *testing.T) {
	h := newTruthHarness(t, allowAllGate{})
	_, err := h.engine.ExplainCurrentTruth(context.Background(), domain.TruthQuery{})
	if err == nil {
		t.Fatalf("expected missing scope error")
	}
	var truthErr domain.TruthError
	if !asTruthError(err, &truthErr) {
		t.Fatalf("expected domain.TruthError, got %T", err)
	}
	if truthErr.Code != domain.TruthErrMissingScope {
		t.Fatalf("expected missing scope code, got %s", truthErr.Code)
	}
}

func asTruthError(err error, out *domain.TruthError) bool {
	if err == nil || out == nil {
		return false
	}
	te, ok := err.(domain.TruthError)
	if !ok {
		return false
	}
	*out = te
	return true
}

func readScalar(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var value string
	if err := db.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatalf("read scalar failed: %v", err)
	}
	return value
}
