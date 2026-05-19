package autonomy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/truth"
)

type staleLoopLifecycleStub struct {
	loops []domain.OpenLoop
}

func (s staleLoopLifecycleStub) OpenLoop(context.Context, truth.LoopMutationRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}
func (s staleLoopLifecycleStub) TransitionLoop(context.Context, truth.LoopMutationRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}
func (s staleLoopLifecycleStub) ResolveLoop(context.Context, truth.LoopMutationRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}
func (s staleLoopLifecycleStub) BlockLoop(context.Context, truth.LoopMutationRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}
func (s staleLoopLifecycleStub) ReopenLoop(context.Context, truth.LoopMutationRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}
func (s staleLoopLifecycleStub) ArchiveLoop(context.Context, truth.LoopMutationRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}
func (s staleLoopLifecycleStub) ListActiveLoops(context.Context, domain.ForgeScope, int) ([]domain.OpenLoop, error) {
	return nil, nil
}
func (s staleLoopLifecycleStub) ListBlockedLoops(context.Context, domain.ForgeScope, int) ([]domain.OpenLoop, error) {
	return nil, nil
}
func (s staleLoopLifecycleStub) ListLoopsByPriority(context.Context, domain.ForgeScope, string, int) ([]domain.OpenLoop, error) {
	return nil, nil
}
func (s staleLoopLifecycleStub) ListLoopsByOwner(context.Context, domain.ForgeScope, string, int) ([]domain.OpenLoop, error) {
	return nil, nil
}
func (s staleLoopLifecycleStub) ListStaleLoops(context.Context, domain.ForgeScope, int64, int) ([]domain.OpenLoop, error) {
	return s.loops, nil
}
func (s staleLoopLifecycleStub) ExplainLoop(context.Context, string, domain.ForgeScope, int64) (domain.OpenLoopExplanation, error) {
	return domain.OpenLoopExplanation{}, nil
}

func TestPolicyEvaluatorPersistenceGatePreventsMaintainAutoCommit(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	intent := h.newIntent("persist-gate", domain.IntentSourceForge, "charter_memory_maintenance")
	action := createSystemNoteAction(h.scope, "note-persist-gate", h.nextMillis())

	decision, err := h.evaluator.Evaluate(ctx, EvaluationInput{
		Intent:  intent,
		Actions: []domain.SyscallRequest{action},
		Mode:    domain.AutonomyModeMaintain,
	})
	if err != nil {
		t.Fatalf("evaluate persistence gate: %v", err)
	}
	if decision.Decision != domain.DecisionAllowProposeOnly {
		t.Fatalf("expected propose_only under persistence gate, got %+v", decision)
	}
	if !strings.Contains(strings.Join(decision.DeniedReasons, ";"), "durable charter+budget backing required") {
		t.Fatalf("expected durable backing denial reason, got %+v", decision.DeniedReasons)
	}
}

func TestValidateAutonomyCommitActionBlocksRuleAgentDestructiveArchive(t *testing.T) {
	intent := domain.AutonomyIntent{
		ID:     "intent-rule-agent-destructive",
		Source: domain.IntentSourceRuleAgent,
		Scope:  domain.ForgeScope{WorkspaceID: "ws-1", LaneID: "control.semantic"},
	}
	action := domain.SyscallRequest{
		ID:     "act-archive",
		Action: domain.ActionArchiveNote,
		Scope:  intent.Scope,
		Payload: map[string]any{
			"noteId":     "note-real",
			"noteStatus": "superseded",
			"ageDays":    30,
		},
	}

	err := validateAutonomyCommitAction(intent, action)
	if err == nil {
		t.Fatalf("expected destructive rule-agent archive to be blocked")
	}
	if !strings.Contains(err.Message, "rule-agent intents cannot directly commit destructive actions") {
		t.Fatalf("unexpected guard error: %+v", err)
	}
}

func TestCommitAllowedActionsBlocksCleanupPlaceholderTarget(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	intent := h.newIntent("intent-cleanup-placeholder", domain.IntentSourceForge, "charter_memory_maintenance")
	decision := domain.AutonomyDecision{
		ID: "decision-cleanup-placeholder",
		AllowedActions: []domain.SyscallRequest{
			{
				ID:     "act-cleanup-placeholder",
				Action: domain.ActionArchiveNote,
				Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
				Source: domain.SourceSystem,
				Scope:  h.scope,
				Payload: map[string]any{
					"noteId":        "candidate-note",
					"noteStatus":    "active",
					"ageDays":       1,
					"archiveReason": "cleanup_review",
				},
				RequestedAt: h.nextMillis(),
			},
		},
	}

	results, committedIDs, errs := h.runner.commitAllowedActions(ctx, intent, decision)
	if len(results) != 0 {
		t.Fatalf("expected no syscall result for blocked placeholder commit, got %+v", results)
	}
	if len(committedIDs) != 0 {
		t.Fatalf("expected no committed ids for blocked placeholder commit, got %+v", committedIDs)
	}
	if len(errs) == 0 {
		t.Fatalf("expected placeholder commit guard error")
	}
	if !strings.Contains(errs[0].Message, "cleanup placeholder target cannot be committed") {
		t.Fatalf("unexpected placeholder guard error: %+v", errs[0])
	}
}

func TestCleanupProposalAgentNoPlaceholderActionTarget(t *testing.T) {
	t.Parallel()
	agent := CleanupProposalAgent{}
	input := BuildRuleAgentInput(
		domain.ForgeScope{WorkspaceID: "ws-cleanup", LaneID: "control.semantic"},
		"corr-cleanup-no-placeholder",
		"trace-cleanup-no-placeholder",
		nil,
		0,
		"manual",
		123456789,
	)
	res, err := agent.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("cleanup proposal evaluate: %v", err)
	}
	if len(res.Actions) != 0 {
		t.Fatalf("expected cleanup proposal to emit no direct actions by default, got %d", len(res.Actions))
	}
}

func TestExtractPreflightObjectRefsBuildsArchiveRef(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	action := domain.SyscallRequest{
		Action: domain.ActionArchiveNote,
		Scope:  scope,
		Payload: map[string]any{
			"noteId": "note-target-123",
		},
	}
	refs := extractPreflightObjectRefs(action, scope)
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	ref, ok := refs[0].(map[string]any)
	if !ok {
		t.Fatalf("ref is not map[string]any: %T", refs[0])
	}
	if ref["ref_type"] != "memory_note" {
		t.Fatalf("expected ref_type=memory_note, got %v", ref["ref_type"])
	}
	if ref["ref_id"] != "note-target-123" {
		t.Fatalf("expected ref_id=note-target-123, got %v", ref["ref_id"])
	}
	if ref["workspace_id"] != "ws-1" {
		t.Fatalf("expected workspace_id=ws-1, got %v", ref["workspace_id"])
	}
}

func TestExtractPreflightObjectRefsSkipsUnsupportedAction(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	action := domain.SyscallRequest{
		Action:  domain.ActionCreateNote,
		Scope:   scope,
		Payload: map[string]any{"id": "note-create"},
	}
	if refs := extractPreflightObjectRefs(action, scope); refs != nil {
		t.Fatalf("expected no refs for CREATE_NOTE, got %+v", refs)
	}
}

func TestExtractPreflightObjectRefsSkipsArchiveWithoutTarget(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	action := domain.SyscallRequest{
		Action:  domain.ActionArchiveNote,
		Scope:   scope,
		Payload: map[string]any{},
	}
	if refs := extractPreflightObjectRefs(action, scope); refs != nil {
		t.Fatalf("expected no refs for archive without target, got %+v", refs)
	}
}

func TestExtractPreflightObjectRefsFallsBackToIntentScope(t *testing.T) {
	t.Parallel()
	fallback := domain.ForgeScope{WorkspaceID: "ws-fallback"}
	action := domain.SyscallRequest{
		Action:  domain.ActionArchiveNote,
		Payload: map[string]any{"noteId": "note-x"},
	}
	refs := extractPreflightObjectRefs(action, fallback)
	if len(refs) != 1 {
		t.Fatalf("expected fallback scope to populate refs, got %+v", refs)
	}
	ref := refs[0].(map[string]any)
	if ref["workspace_id"] != "ws-fallback" {
		t.Fatalf("expected workspace_id=ws-fallback, got %v", ref["workspace_id"])
	}
}

func TestExtractPreflightObjectRefsBuildsMarkSupersededRefs(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	action := domain.SyscallRequest{
		Action: domain.ActionMarkSuperseded,
		Scope:  scope,
		Payload: map[string]any{
			"oldObjectId":   "obj-old",
			"oldObjectKind": "memory_note",
			"newObjectId":   "obj-new",
			"newObjectKind": "memory_note",
		},
	}
	refs := extractPreflightObjectRefs(action, scope)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if got := refs[0].(map[string]any); got["ref_id"] != "obj-old" || got["ref_type"] != "memory_note" {
		t.Fatalf("old ref unexpected: %+v", got)
	}
	if got := refs[1].(map[string]any); got["ref_id"] != "obj-new" || got["ref_type"] != "memory_note" {
		t.Fatalf("new ref unexpected: %+v", got)
	}
}

func TestExtractPreflightObjectRefsMarkSupersededDefaultKindFallsBackToSemanticObject(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	action := domain.SyscallRequest{
		Action: domain.ActionMarkSuperseded,
		Scope:  scope,
		Payload: map[string]any{
			"oldObjectId":   "obj-old",
			"oldObjectKind": "object",
			"newObjectId":   "obj-new",
		},
	}
	refs := extractPreflightObjectRefs(action, scope)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	for idx, raw := range refs {
		ref := raw.(map[string]any)
		if ref["ref_type"] != "semantic_object" {
			t.Fatalf("ref #%d expected semantic_object fallback, got %v", idx, ref["ref_type"])
		}
	}
}

func TestExtractPreflightObjectRefsSkipsMarkSupersededWithoutBothIds(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	for label, payload := range map[string]map[string]any{
		"missing_new": {"oldObjectId": "obj-old"},
		"missing_old": {"newObjectId": "obj-new"},
		"empty":       {"oldObjectId": "", "newObjectId": ""},
	} {
		action := domain.SyscallRequest{Action: domain.ActionMarkSuperseded, Scope: scope, Payload: payload}
		if refs := extractPreflightObjectRefs(action, scope); refs != nil {
			t.Fatalf("case %s: expected no refs, got %+v", label, refs)
		}
	}
}

func TestExtractPreflightObjectRefsBuildsDeriveModelRefs(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	action := domain.SyscallRequest{
		Action: domain.ActionDeriveModel,
		Scope:  scope,
		Payload: map[string]any{
			"derivedFrom": []any{"src-1", "src-2", "src-3"},
		},
	}
	refs := extractPreflightObjectRefs(action, scope)
	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}
	for idx, raw := range refs {
		ref := raw.(map[string]any)
		if ref["ref_type"] != "semantic_object" {
			t.Fatalf("ref #%d expected semantic_object, got %v", idx, ref["ref_type"])
		}
	}
}

func TestExtractPreflightObjectRefsSkipsDeriveModelWithEmptySources(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	for label, payload := range map[string]map[string]any{
		"missing":     {},
		"empty_slice": {"derivedFrom": []any{}},
		"only_blank":  {"derivedFrom": []any{"  ", ""}},
	} {
		action := domain.SyscallRequest{Action: domain.ActionDeriveModel, Scope: scope, Payload: payload}
		if refs := extractPreflightObjectRefs(action, scope); refs != nil {
			t.Fatalf("case %s: expected no refs, got %+v", label, refs)
		}
	}
}

func TestPreflightObjectRefTypeMapsKnownKindsAndDefaultsRest(t *testing.T) {
	t.Parallel()
	known := []string{"memory_note", "semantic_link", "state_item", "open_loop", "derived_model"}
	for _, kind := range known {
		if got := preflightObjectRefType(kind); got != kind {
			t.Fatalf("kind %q: expected pass-through, got %q", kind, got)
		}
	}
	for _, kind := range []string{"", "object", "artifact", "unknown_kind"} {
		if got := preflightObjectRefType(kind); got != "semantic_object" {
			t.Fatalf("kind %q: expected semantic_object fallback, got %q", kind, got)
		}
	}
}

func TestCommitAllowedActionsPreflightBlocksMarkSupersededOfMissingObject(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	intent := h.newIntent("intent-supersede-missing", domain.IntentSourceForge, "charter_memory_maintenance")
	decision := domain.AutonomyDecision{
		ID: "decision-supersede-missing",
		AllowedActions: []domain.SyscallRequest{
			{
				ID:     "act-supersede-missing-target",
				Action: domain.ActionMarkSuperseded,
				Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
				Source: domain.SourceSystem,
				Scope:  h.scope,
				Payload: map[string]any{
					"oldObjectId":   "obj-never-created-old",
					"oldObjectKind": "memory_note",
					"newObjectId":   "obj-never-created-new",
					"newObjectKind": "memory_note",
					"reason":        "preflight_missing_objects",
				},
				RequestedAt: h.nextMillis(),
			},
		},
	}

	results, committedIDs, errs := h.runner.commitAllowedActions(ctx, intent, decision)
	if len(results) != 0 {
		t.Fatalf("expected no syscall result when preflight rejects missing objects, got %+v", results)
	}
	if len(committedIDs) != 0 {
		t.Fatalf("expected no committed ids when preflight rejects missing objects, got %+v", committedIDs)
	}
	if len(errs) == 0 {
		t.Fatalf("expected preflight rejection error for missing supersession objects")
	}
	if !strings.Contains(errs[0].Message, "source object authority preflight failed") {
		t.Fatalf("expected preflight rejection message, got %+v", errs[0])
	}
}

func TestCommitAllowedActionsPreflightBlocksDeriveModelWithMissingSources(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	intent := h.newIntent("intent-derive-missing-source", domain.IntentSourceForge, "charter_memory_maintenance")
	decision := domain.AutonomyDecision{
		ID: "decision-derive-missing-source",
		AllowedActions: []domain.SyscallRequest{
			{
				ID:     "act-derive-missing-source",
				Action: domain.ActionDeriveModel,
				Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
				Source: domain.SourceSystem,
				Scope:  h.scope,
				Payload: map[string]any{
					"id":           "model-preflight-fail",
					"type":         "context_policy_preference",
					"expression":   map[string]any{"formula": "test"},
					"derivedFrom":  []any{"src-never-created"},
					"supportCount": 1,
					"confidence":   0.5,
				},
				RequestedAt: h.nextMillis(),
			},
		},
	}

	results, committedIDs, errs := h.runner.commitAllowedActions(ctx, intent, decision)
	if len(results) != 0 {
		t.Fatalf("expected no syscall result when preflight rejects derive-model with missing source, got %+v", results)
	}
	if len(committedIDs) != 0 {
		t.Fatalf("expected no committed ids when preflight rejects derive-model, got %+v", committedIDs)
	}
	if len(errs) == 0 {
		t.Fatalf("expected preflight rejection error for missing derive-model source")
	}
	if !strings.Contains(errs[0].Message, "source object authority preflight failed") {
		t.Fatalf("expected preflight rejection message, got %+v", errs[0])
	}
}

func TestExtractPreflightObjectRefsRegisterContradictPreflightsOnlyGovernedSides(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	action := domain.SyscallRequest{
		Action: domain.ActionRegisterContradict,
		Scope:  scope,
		Payload: map[string]any{
			"leftObjectId":    "state-current",
			"leftObjectKind":  "state_item",
			"rightObjectId":   "journal-event-42",
			"rightObjectKind": "journal_event",
		},
	}
	refs := extractPreflightObjectRefs(action, scope)
	if len(refs) != 1 {
		t.Fatalf("expected only governed left side to be preflighted, got %d refs: %+v", len(refs), refs)
	}
	ref := refs[0].(map[string]any)
	if ref["ref_id"] != "state-current" || ref["ref_type"] != "state_item" {
		t.Fatalf("unexpected ref: %+v", ref)
	}
}

func TestExtractPreflightObjectRefsRegisterContradictSkipsWhenBothSidesUnresolvable(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	action := domain.SyscallRequest{
		Action: domain.ActionRegisterContradict,
		Scope:  scope,
		Payload: map[string]any{
			"leftObjectId":    "journal-event-1",
			"leftObjectKind":  "journal_event",
			"rightObjectId":   "artifact-2",
			"rightObjectKind": "artifact_ref",
		},
	}
	if refs := extractPreflightObjectRefs(action, scope); refs != nil {
		t.Fatalf("expected no refs when both kinds are unresolvable, got %+v", refs)
	}
}

func TestExtractPreflightObjectRefsRegisterContradictPreflightsBothGovernedSides(t *testing.T) {
	t.Parallel()
	scope := domain.ForgeScope{WorkspaceID: "ws-1"}
	action := domain.SyscallRequest{
		Action: domain.ActionRegisterContradict,
		Scope:  scope,
		Payload: map[string]any{
			"leftObjectId":    "note-a",
			"leftObjectKind":  "memory_note",
			"rightObjectId":   "note-b",
			"rightObjectKind": "memory_note",
		},
	}
	refs := extractPreflightObjectRefs(action, scope)
	if len(refs) != 2 {
		t.Fatalf("expected both governed sides to be preflighted, got %d", len(refs))
	}
}

func TestResolvablePreflightRefTypeMatchesResolverKinds(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"memory_note", "semantic_link", "state_item", "open_loop", "derived_model", "semantic_object"} {
		if got, ok := resolvablePreflightRefType(kind); !ok || got != kind {
			t.Fatalf("kind %q: expected (%q,true), got (%q,%v)", kind, kind, got, ok)
		}
	}
	for _, kind := range []string{"", "object", "journal_event", "artifact_ref", "evidence", "unknown"} {
		if got, ok := resolvablePreflightRefType(kind); ok || got != "" {
			t.Fatalf("kind %q: expected ('',false), got (%q,%v)", kind, got, ok)
		}
	}
}

func TestCommitAllowedActionsPreflightBlocksRegisterContradictOnMissingLeftObject(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	intent := h.newIntent("intent-contradict-missing-left", domain.IntentSourceForge, "charter_memory_maintenance")
	decision := domain.AutonomyDecision{
		ID: "decision-contradict-missing-left",
		AllowedActions: []domain.SyscallRequest{
			{
				ID:     "act-contradict-missing-left",
				Action: domain.ActionRegisterContradict,
				Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
				Source: domain.SourceSystem,
				Scope:  h.scope,
				Payload: map[string]any{
					"leftObjectId":    "note-never-created-left",
					"leftObjectKind":  "memory_note",
					"rightObjectId":   "journal-evt-rhs",
					"rightObjectKind": "journal_event",
					"reason":          "preflight_missing_left",
					"severity":        "low",
				},
				RequestedAt: h.nextMillis(),
			},
		},
	}

	results, committedIDs, errs := h.runner.commitAllowedActions(ctx, intent, decision)
	if len(results) != 0 {
		t.Fatalf("expected no syscall result when preflight rejects missing left, got %+v", results)
	}
	if len(committedIDs) != 0 {
		t.Fatalf("expected no committed ids when preflight rejects missing left, got %+v", committedIDs)
	}
	if len(errs) == 0 {
		t.Fatalf("expected preflight rejection error for missing contradiction left object")
	}
	if !strings.Contains(errs[0].Message, "source object authority preflight failed") {
		t.Fatalf("expected preflight rejection message, got %+v", errs[0])
	}
}

func TestCommitAllowedActionsPreflightBlocksArchiveOfMissingNote(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	intent := h.newIntent("intent-archive-missing-target", domain.IntentSourceForge, "charter_memory_maintenance")
	decision := domain.AutonomyDecision{
		ID: "decision-archive-missing-target",
		AllowedActions: []domain.SyscallRequest{
			{
				ID:     "act-archive-missing-target",
				Action: domain.ActionArchiveNote,
				Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
				Source: domain.SourceSystem,
				Scope:  h.scope,
				Payload: map[string]any{
					"noteId":        "note-never-created",
					"noteStatus":    "active",
					"ageDays":       30,
					"archiveReason": "preflight_missing_target",
				},
				RequestedAt: h.nextMillis(),
			},
		},
	}

	results, committedIDs, errs := h.runner.commitAllowedActions(ctx, intent, decision)
	if len(results) != 0 {
		t.Fatalf("expected no syscall result when preflight rejects missing note, got %+v", results)
	}
	if len(committedIDs) != 0 {
		t.Fatalf("expected no committed ids when preflight rejects missing note, got %+v", committedIDs)
	}
	if len(errs) == 0 {
		t.Fatalf("expected preflight rejection error for missing note")
	}
	if !strings.Contains(errs[0].Message, "source object authority preflight failed") {
		t.Fatalf("expected preflight rejection message, got %+v", errs[0])
	}
}

func TestCommitAllowedActionsPreflightAllowsArchiveOfExistingNote(t *testing.T) {
	h := newAutonomyHarness(t)
	ctx := context.Background()

	noteID := "note-real-archive-target"
	createReq := createSystemNoteAction(h.scope, noteID, h.nextMillis())
	createRes, err := h.kernel.Process(ctx, createReq)
	if err != nil {
		t.Fatalf("seed CREATE_NOTE: %v", err)
	}
	if !createRes.Success {
		t.Fatalf("seed CREATE_NOTE not successful: %+v", createRes)
	}

	intent := h.newIntent("intent-archive-real-target", domain.IntentSourceForge, "charter_memory_maintenance")
	decision := domain.AutonomyDecision{
		ID: "decision-archive-real-target",
		AllowedActions: []domain.SyscallRequest{
			{
				ID:     "act-archive-real-target",
				Action: domain.ActionArchiveNote,
				Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
				Source: domain.SourceSystem,
				Scope:  h.scope,
				Payload: map[string]any{
					"noteId":        noteID,
					"noteStatus":    "active",
					"ageDays":       30,
					"archiveReason": "preflight_real_target",
				},
				RequestedAt: h.nextMillis(),
			},
		},
	}

	_, _, errs := h.runner.commitAllowedActions(ctx, intent, decision)
	for _, e := range errs {
		if strings.Contains(e.Message, "source object authority preflight failed") {
			t.Fatalf("preflight should pass for existing note, got %+v", e)
		}
	}
}

func TestHasPlaceholderArchiveTargetBlocksCandidatePrefixes(t *testing.T) {
	t.Parallel()
	cases := []map[string]any{
		{"noteId": "candidate-note"},
		{"id": "candidate-123"},
		{"targetId": "fake-archive-target"},
		{"objectId": "placeholder-item"},
	}
	for idx, payload := range cases {
		if !hasPlaceholderArchiveTarget(payload) {
			t.Fatalf("expected placeholder guard for payload #%d: %+v", idx, payload)
		}
	}
	if hasPlaceholderArchiveTarget(map[string]any{"noteId": "note-real"}) {
		t.Fatalf("did not expect placeholder guard for real note id")
	}
}

// TestLiveNoteIDGeneratorsAvoidPlaceholderPrefixes is the regression net for the
// hasPlaceholderArchiveTarget string-prefix contract. The guard at
// runner.go:hasPlaceholderArchiveTarget filters payload IDs by literal prefix
// ("candidate-", "fake-", "placeholder-"), which is only safe if every live
// generator of those payload IDs is guaranteed not to collide. This test
// exercises the live generator shapes and asserts none of them produce a
// forbidden prefix across a broad sweep of synthetic inputs.
func TestLiveNoteIDGeneratorsAvoidPlaceholderPrefixes(t *testing.T) {
	t.Parallel()

	forbidden := []string{"candidate-", "fake-", "placeholder-"}
	assertSafe := func(t *testing.T, label, id string) {
		t.Helper()
		lower := strings.ToLower(strings.TrimSpace(id))
		if lower == "candidate-note" {
			t.Fatalf("%s: live generator produced reserved placeholder id %q", label, id)
		}
		for _, p := range forbidden {
			if strings.HasPrefix(lower, p) {
				t.Fatalf("%s: live generator produced forbidden prefix %q in id %q", label, p, id)
			}
		}
		// Equivalent assertion via the actual guard to keep the safety
		// contract co-located with what runner.go enforces.
		if hasPlaceholderArchiveTarget(map[string]any{"noteId": id}) {
			t.Fatalf("%s: live id %q is rejected by hasPlaceholderArchiveTarget", label, id)
		}
	}

	// 1. shortHash output is hex-only by construction; sweep a wide input
	//    space and confirm none of the live "<literal>-" + shortHash(...)
	//    note-id templates ever produce a forbidden prefix.
	workspaces := []string{"ws-1", "ws-cleanup", "ws/unicode-ñ", "WS_UPPER", strings.Repeat("w", 64), ""}
	loops := []string{"loop-1", "loop/abc", "", strings.Repeat("l", 256)}
	now := int64(1700000000000)
	for _, ws := range workspaces {
		for _, loop := range loops {
			for _, off := range []int64{0, 1, 12345, 9223372036854775} {
				ts := fmt.Sprintf("%d", now+off)
				// rule_agents.go:175 — OpenLoopStalenessAgent note payload "id"
				assertSafe(t, "rule_agents.OpenLoopStalenessAgent.note.id",
					"note-stale-loop-"+shortHash(loop))
				// rule_agents.go:169 — action ID (not a payload note id, but shares the literal)
				assertSafe(t, "rule_agents.OpenLoopStalenessAgent.action.id",
					"action-stale-loop-note-"+shortHash(loop, ts))
				// autonomy_maintenance_loop.go:1014 — dream-loop noteId
				assertSafe(t, "autonomy_maintenance_loop.dream.noteId",
					"note-dream-improvement-"+shortHash(ws, ts))
				// autonomy_maintenance_loop.go:1036 — dream-loop action id
				assertSafe(t, "autonomy_maintenance_loop.dream.action.id",
					"action-dream-note-"+shortHash("intent-dream-improvement-"+shortHash(ts, ws), "note"))
			}
		}
	}

	// 2. shortHash itself must never produce a forbidden prefix; it returns
	//    lowercase hex from sha1, so verify across a range of inputs.
	for i := 0; i < 256; i++ {
		h := shortHash(fmt.Sprintf("probe-%d", i), "salt")
		for _, p := range forbidden {
			if strings.HasPrefix(h, p) {
				t.Fatalf("shortHash(%d) produced forbidden prefix %q in %q", i, p, h)
			}
		}
		if len(h) != 16 {
			t.Fatalf("shortHash expected 16 hex chars, got %d (%q)", len(h), h)
		}
	}

	// 3. OpenLoopStalenessAgent end-to-end: drive the live agent and verify
	//    the payload it produces passes the guard for any non-placeholder
	//    upstream loop id.
	agent := OpenLoopStalenessAgent{}
	staleLoops := []domain.OpenLoop{{ID: "loop-real-1", Title: "stale review", UpdatedAt: 1, State: domain.LoopOpen}}
	in := BuildRuleAgentInput(
		domain.ForgeScope{WorkspaceID: "ws-prefix-guard", LaneID: "control.semantic"},
		"corr-prefix-guard",
		"trace-prefix-guard",
		staleLoopLifecycleStub{loops: staleLoops},
		0,
		"manual",
		now,
	)
	res, err := agent.Evaluate(context.Background(), in)
	if err != nil {
		t.Fatalf("OpenLoopStalenessAgent evaluate: %v", err)
	}
	if len(res.Actions) == 0 {
		t.Fatalf("expected at least one action from OpenLoopStalenessAgent")
	}
	for _, act := range res.Actions {
		assertSafe(t, "OpenLoopStalenessAgent.live.action.id", act.ID)
		if id, ok := act.Payload["id"].(string); ok {
			assertSafe(t, "OpenLoopStalenessAgent.live.payload.id", id)
		}
		if id, ok := act.Payload["noteId"].(string); ok {
			assertSafe(t, "OpenLoopStalenessAgent.live.payload.noteId", id)
		}
	}
}
