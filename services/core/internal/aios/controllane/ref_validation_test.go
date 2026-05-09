package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestValidateRefShapeLiveSyscallSucceedsWithoutMemoryMutation(t *testing.T) {
	ctx := context.Background()
	store := NewInMemorySemanticStore()
	auditSink := NewInMemoryAuditSink()
	k := NewProcessor(ProcessorOptions{
		Registry:     NewStaticActionRegistry(),
		Validator:    NewDeterministicValidator(),
		Capabilities: NewStaticCapabilityService(),
		ApprovalGate: NewStaticApprovalGate(),
		TxRunner:     NewInMemoryTransactionRunner(store),
		AuditSink:    auditSink,
		NowMillis:    func() int64 { return 1770000000000 },
	})
	req := validRefShapeRequest()
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got %#v", res)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("ref validation must not commit semantic objects, got %v", res.CommittedObjectIDs)
	}
	if res.StateSummary["passed"] != true {
		t.Fatalf("expected passed state summary, got %#v", res.StateSummary)
	}
	if res.StateSummary["memoryMutation"] != false || res.StateSummary["runtimeMutation"] != false || res.StateSummary["liveAuthorityMigration"] != false {
		t.Fatalf("ref validation claimed mutation/authority migration: %#v", res.StateSummary)
	}
	refs, ok := res.StateSummary["normalizedRefs"].([]map[string]string)
	if !ok || len(refs) != 2 || refs[0]["ref_id"] != "note-a" || refs[1]["ref_id"] != "note-b" {
		t.Fatalf("expected deterministic normalized refs, got %#v", res.StateSummary["normalizedRefs"])
	}
	if res.AuditID == "" || len(auditSink.Records) == 0 || !auditSink.Records[len(auditSink.Records)-1].Success {
		t.Fatalf("expected successful audit record, auditID=%q records=%#v", res.AuditID, auditSink.Records)
	}
	auditDecision := auditSink.Records[len(auditSink.Records)-1].RefShapeValidation
	if auditDecision["decision"] != RefShapeDecisionAccepted || auditDecision["liveAuthorityMigration"] != false {
		t.Fatalf("audit missing ref shape decision: %#v", auditDecision)
	}
}

func TestValidateRefShapeRejectsInvalidRefsWithoutStateMutation(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()
	req := validRefShapeRequest()
	req.ID = "ref-shape-bad"
	req.IdempotencyKey = "ref-shape-bad"
	req.Payload["refs"] = []any{
		map[string]any{"ref_type": "raw_prompt", "ref_id": "prompt-a"},
		map[string]any{"ref_type": "memory_note", "ref_id": "token=secret"},
	}
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected invalid ref shape to fail")
	}
	if res.DeterministicErrCode != domain.ErrInvalidPayload {
		t.Fatalf("expected invalid payload, got %s", res.DeterministicErrCode)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("invalid ref validation committed ids: %v", res.CommittedObjectIDs)
	}
	if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
		t.Fatal("failed validation must not persist idempotency state")
	}
}

func TestValidateRefShapeCapabilityDeniedForProposeOnlySource(t *testing.T) {
	ctx := context.Background()
	k, _, _ := newTestKernel()
	req := validRefShapeRequest()
	req.ID = "ref-shape-future-iris"
	req.Source = domain.SourceFutureIRIS
	req.Actor.Kind = string(domain.SourceFutureIRIS)
	req.Provenance.ActorType = string(domain.SourceFutureIRIS)
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected propose-only source to be denied")
	}
	if res.DeterministicErrCode != domain.ErrCapabilityDenied {
		t.Fatalf("expected capability denied, got %s", res.DeterministicErrCode)
	}
}

func TestValidateRefShapeDryRunReturnsValidationSummary(t *testing.T) {
	ctx := context.Background()
	k, _, _ := newTestKernel()
	req := validRefShapeRequest()
	req.ID = "ref-shape-dry-run"
	req.IdempotencyKey = ""
	req.DryRun = true
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected dry-run validation success, got %#v", res)
	}
	if res.StateSummary["passed"] != true || res.StateSummary["dryRun"] != true {
		t.Fatalf("dry-run lost validation summary: %#v", res.StateSummary)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("dry-run must not commit ids: %v", res.CommittedObjectIDs)
	}
}

func validRefShapeRequest() domain.SyscallRequest {
	req := validBaseRequest(domain.ActionValidateRefShape)
	req.ID = "ref-shape-pass"
	req.Payload = map[string]any{
		"workspace_id": "ws-main",
		"refs": []any{
			map[string]any{"ref_type": "memory_note", "ref_id": "note-b"},
			map[string]any{"ref_type": "memory_note", "ref_id": "note-a", "workspace_id": "ws-main"},
			map[string]any{"ref_type": "memory_note", "ref_id": "note-a", "workspace_id": "ws-main"},
		},
	}
	return req
}
