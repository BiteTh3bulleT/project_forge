package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/semanticvalidation"
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
	for _, claim := range semanticvalidation.ForbiddenAuthorityClaims() {
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

func TestValidateSemanticOperationRejectsNormalizedTruthyForbiddenClaims(t *testing.T) {
	for _, tc := range []struct {
		name  string
		claim string
		value any
	}{
		{name: "bool true", claim: "  WRITE_MEMORY  ", value: true},
		{name: "string true", claim: "  CALL_MODELRUNTIME  ", value: "TRUE"},
		{name: "string one", claim: "  COMPILE_CONTEXT  ", value: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			k, store, auditSink := newTestKernel()
			req := validSemanticOperationRequest()
			req.ID = "semantic-op-normalized-" + tc.name
			req.IdempotencyKey = req.ID
			req.Payload["claims"] = map[string]any{tc.claim: tc.value}

			res, err := k.Process(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertSemanticOperationRejectedWithoutEffects(t, res, auditSink)
			if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
				t.Fatal("failed validation must not persist idempotency state")
			}
		})
	}
}

func TestValidateSemanticOperationRejectsMixedSafeAndForbiddenClaims(t *testing.T) {
	ctx := context.Background()
	k, store, auditSink := newTestKernel()
	req := validSemanticOperationRequest()
	req.ID = "semantic-op-mixed-claims"
	req.IdempotencyKey = req.ID
	req.Payload["claims"] = map[string]any{
		"advisory_only":    true,
		"operator_visible": "true",
		" run_retrieval ":  "1",
	}

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSemanticOperationRejectedWithoutEffects(t, res, auditSink)
	if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
		t.Fatal("failed validation must not persist idempotency state")
	}
}

func TestValidateSemanticOperationAllowsFalseAndNonTruthyForbiddenClaims(t *testing.T) {
	ctx := context.Background()
	k, _, _ := newTestKernel()
	req := validSemanticOperationRequest()
	req.ID = "semantic-op-false-claims"
	req.Payload["claims"] = map[string]any{
		"write_memory":      false,
		"call_modelruntime": "false",
		"compile_context":   "0",
		"execute_tool":      "no",
	}

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("false/non-truthy forbidden claims must not reject by themselves: %#v", res)
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

func assertSemanticOperationRejectedWithoutEffects(t *testing.T, res domain.SyscallResult, auditSink *InMemoryAuditSink) {
	t.Helper()
	if res.Success {
		t.Fatalf("expected semantic operation validation rejection, got %#v", res)
	}
	if res.DeterministicErrCode != domain.ErrInvalidPayload {
		t.Fatalf("expected invalid payload, got %s", res.DeterministicErrCode)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("rejected semantic operation validation must not commit objects, got %v", res.CommittedObjectIDs)
	}
	if res.StateSummary["memoryMutation"] != false ||
		res.StateSummary["modelRuntimeCall"] != false ||
		res.StateSummary["evidenceAdmission"] != false ||
		res.StateSummary["contextCompilation"] != false ||
		res.StateSummary["liveAuthorityMigration"] != false {
		t.Fatalf("rejected validation claimed forbidden effect: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateSemanticOperation), res.StateSummary)
	nested, ok := res.StateSummary["semanticOperationValidation"].(map[string]any)
	if !ok {
		t.Fatalf("missing nested semantic operation summary: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateSemanticOperation), nested)
	if nested["decision"] != SemanticOperationDecisionRejected {
		t.Fatalf("expected rejected decision, got %#v", nested)
	}
	if len(auditSink.Records) == 0 {
		t.Fatal("expected audit record")
	}
	auditDecision := auditSink.Records[len(auditSink.Records)-1].SemanticOperationValidation
	if auditDecision["decision"] != SemanticOperationDecisionRejected ||
		auditDecision["memoryMutation"] != false ||
		auditDecision["modelRuntimeCall"] != false ||
		auditDecision["evidenceAdmission"] != false ||
		auditDecision["contextCompilation"] != false ||
		auditDecision["liveAuthorityMigration"] != false {
		t.Fatalf("audit decision claimed forbidden effect: %#v", auditDecision)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateSemanticOperation), auditDecision)
}
