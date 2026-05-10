package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestValidateSemanticOperationSucceedsWithoutLiveMutation(t *testing.T) {
	ctx := context.Background()
	k, _, auditSink := newTestKernel()
	req := validSemanticOperationRequest()
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected semantic operation validation success, got %#v", res)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("semantic operation validation must not commit objects, got %v", res.CommittedObjectIDs)
	}
	if res.StateSummary["passed"] != true ||
		res.StateSummary["memoryMutation"] != false ||
		res.StateSummary["modelRuntimeCall"] != false ||
		res.StateSummary["evidenceAdmission"] != false ||
		res.StateSummary["contextCompilation"] != false {
		t.Fatalf("unexpected state summary: %#v", res.StateSummary)
	}
	auditDecision := auditSink.Records[len(auditSink.Records)-1].SemanticOperationValidation
	if auditDecision["decision"] != SemanticOperationDecisionAccepted || auditDecision["liveAuthorityMigration"] != false {
		t.Fatalf("audit missing semantic operation decision: %#v", auditDecision)
	}
}

func TestValidateSemanticOperationRejectsForbiddenAuthorityClaims(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()
	req := validSemanticOperationRequest()
	req.ID = "semantic-op-bad"
	req.IdempotencyKey = "semantic-op-bad"
	req.Payload["claims"] = map[string]any{
		"write_memory": true,
		"call_model":   true,
	}
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected forbidden authority claims to fail")
	}
	if res.DeterministicErrCode != domain.ErrInvalidPayload {
		t.Fatalf("expected invalid payload, got %s", res.DeterministicErrCode)
	}
	if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
		t.Fatal("failed validation must not persist idempotency state")
	}
}

func TestValidateSemanticOperationCapabilityDeniedForProposeOnlySource(t *testing.T) {
	ctx := context.Background()
	k, _, _ := newTestKernel()
	req := validSemanticOperationRequest()
	req.ID = "semantic-op-future-iris"
	req.Source = domain.SourceFutureIRIS
	req.Actor.Kind = string(domain.SourceFutureIRIS)
	req.Provenance.ActorType = string(domain.SourceFutureIRIS)
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected propose-only source to be denied")
	}
	if res.DeterministicErrCode != domain.ErrCapabilityDenied {
		t.Fatalf("expected capability denied, got %s", res.DeterministicErrCode)
	}
}

func validSemanticOperationRequest() domain.SyscallRequest {
	req := validBaseRequest(domain.ActionValidateSemanticOperation)
	req.ID = "semantic-op-pass"
	req.Payload = map[string]any{
		"workspace_id":   "ws-main",
		"operation_type": "derive",
		"source_refs": []any{
			map[string]any{"ref_type": "semantic_object", "ref_id": "obj-b"},
			map[string]any{"ref_type": "semantic_object", "ref_id": "obj-a"},
			map[string]any{"ref_type": "semantic_object", "ref_id": "obj-a"},
		},
		"derived_refs": []any{
			map[string]any{"ref_type": "semantic_object", "ref_id": "derived-a"},
		},
	}
	return req
}
