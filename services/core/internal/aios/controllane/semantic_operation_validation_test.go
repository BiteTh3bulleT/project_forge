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
	activation, ok := res.StateSummary["forgeKActivation"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKActivation summary: %#v", res.StateSummary)
	}
	if activation["mode"] != ForgeKActivationModePartialLiveEnforcement ||
		activation["action"] != string(domain.ActionValidateSemanticOperation) {
		t.Fatalf("unexpected activation summary: %#v", activation)
	}
	noEffect, ok := res.StateSummary["forgeKNoEffect"].(map[string]any)
	if !ok || noEffect["memoryMutation"] != false || noEffect["modelRuntimeCall"] != false || noEffect["contextCompilation"] != false {
		t.Fatalf("unexpected no-effect summary: %#v", res.StateSummary["forgeKNoEffect"])
	}
	auditDecision := auditSink.Records[len(auditSink.Records)-1].SemanticOperationValidation
	if auditDecision["decision"] != SemanticOperationDecisionAccepted || auditDecision["liveAuthorityMigration"] != false {
		t.Fatalf("audit missing semantic operation decision: %#v", auditDecision)
	}
}

func TestValidateSemanticOperationRejectsAllForbiddenAuthorityClaims(t *testing.T) {
	for _, claim := range []string{
		"execute",
		"commit",
		"admit_evidence",
		"reject_evidence",
		"write_memory",
		"call_model",
		"call_modelruntime",
		"execute_tool",
		"run_retrieval",
		"run_search",
		"run_embeddings",
		"compile_context",
		"live_authority_migration",
	} {
		t.Run(claim, func(t *testing.T) {
			ctx := context.Background()
			k, store, _ := newTestKernel()
			req := validSemanticOperationRequest()
			req.ID = "semantic-op-forbidden-" + claim
			req.IdempotencyKey = req.ID
			req.Payload["claims"] = map[string]any{claim: true}

			res, err := k.Process(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Success {
				t.Fatalf("expected claim %q to fail", claim)
			}
			if res.DeterministicErrCode != domain.ErrInvalidPayload {
				t.Fatalf("expected invalid payload, got %s", res.DeterministicErrCode)
			}
			if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
				t.Fatal("failed validation must not persist idempotency state")
			}
		})
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
