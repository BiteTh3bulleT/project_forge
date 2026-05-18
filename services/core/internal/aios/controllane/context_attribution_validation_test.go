package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestValidateContextAttributionIsValidationOnly(t *testing.T) {
	kernel, store, auditSink := newTestKernel()
	req := validContextAttributionRequest()
	req.ID = "ctx-attr-1"

	res, err := kernel.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("process context attribution validation: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected context attribution validation success, got %#v", res)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("context attribution validation must not commit semantic objects, got %v", res.CommittedObjectIDs)
	}
	if len(store.state.contextSnapshots) != 0 {
		t.Fatalf("context attribution validation must not compile or persist context snapshots")
	}
	summary := res.StateSummary
	if summary["contextCompilation"] != false ||
		summary["memoryMutation"] != false ||
		summary["modelRuntimeCall"] != false ||
		summary["gatewayExecution"] != false ||
		summary["liveAuthorityMigration"] != false {
		t.Fatalf("context attribution validation claimed forbidden effects: %#v", summary)
	}
	if got := summary["passed"]; got != true {
		t.Fatalf("summary passed=%v, want true", got)
	}
	if len(auditSink.Records) == 0 || auditSink.Records[len(auditSink.Records)-1].ContextAttributionValidation == nil {
		t.Fatalf("expected context attribution validation audit metadata, got %#v", auditSink.Records)
	}
}

func validContextAttributionRequest() domain.SyscallRequest {
	req := validBaseRequest(domain.ActionValidateContextAttribution)
	req.Payload = map[string]any{
		"workspace_id":    "ws-main",
		"query":           "What should FORGE remember about the active task?",
		"context_purpose": "chat_turn",
		"source_refs": []any{
			map[string]any{"ref_type": "memory_note", "ref_id": "note-1"},
			map[string]any{"ref_type": "state_item", "ref_id": "state-1"},
		},
		"selection_reasons": map[string]any{
			"memory_note:note-1": "note directly matches the requested task",
			"state_item:state-1": "state item is current for this workspace",
		},
	}
	return req
}

func TestValidateContextAttributionRejectsAuthorityClaimsWithoutIdempotency(t *testing.T) {
	kernel, store, _ := newTestKernel()
	req := validBaseRequest(domain.ActionValidateContextAttribution)
	req.ID = "ctx-attr-2"
	req.IdempotencyKey = "ctx-attr-reject"
	req.Payload = map[string]any{
		"workspace_id":    "ws-main",
		"query":           "Explain active task",
		"context_purpose": "chat_turn",
		"source_refs": []any{
			map[string]any{"ref_type": "memory_note", "ref_id": "note-1"},
		},
		"selection_reasons": map[string]any{
			"memory_note:note-1": "note directly matches the requested task",
		},
		"claims": map[string]any{"compile_context": true},
	}

	res, err := kernel.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("process rejected context attribution validation: %v", err)
	}
	if res.Success {
		t.Fatalf("expected context attribution validation rejection, got %#v", res)
	}
	if res.DeterministicErrCode != domain.ErrInvalidPayload {
		t.Fatalf("error code=%q, want %q", res.DeterministicErrCode, domain.ErrInvalidPayload)
	}
	if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
		t.Fatal("failed context attribution validation must not persist idempotency state")
	}
}
