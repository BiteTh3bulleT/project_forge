package autonomy

import (
	"context"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

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
