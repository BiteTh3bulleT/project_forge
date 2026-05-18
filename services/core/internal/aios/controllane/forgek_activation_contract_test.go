package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestForgeKActivationContractForControlLaneValidationActions(t *testing.T) {
	cases := []struct {
		name      string
		action    domain.SemanticActionType
		request   func() domain.SyscallRequest
		setup     func(context.Context, *Processor)
		audit     func(SyscallAuditRecord) map[string]any
		summaryID string
	}{
		{
			name:    "kv identity",
			action:  domain.ActionValidateKVIdentity,
			request: validKVIdentityRequest,
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.KVIdentityEnforcement
			},
			summaryID: "kvIdentityEnforcement",
		},
		{
			name:    "ref shape",
			action:  domain.ActionValidateRefShape,
			request: validRefShapeRequest,
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.RefShapeValidation
			},
			summaryID: "refShapeValidation",
		},
		{
			name:    "ref shape comparison",
			action:  domain.ActionCompareRefShape,
			request: validRefShapeCompareRequest,
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.RefShapeComparison
			},
			summaryID: "refShapeComparison",
		},
		{
			name:    "semantic operation",
			action:  domain.ActionValidateSemanticOperation,
			request: validSemanticOperationRequest,
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.SemanticOperationValidation
			},
			summaryID: "semanticOperationValidation",
		},
		{
			name:    "source object authority",
			action:  domain.ActionValidateSourceObject,
			request: validSourceObjectAuthorityRequest,
			setup: func(ctx context.Context, k *Processor) {
				mustCreateNote(ctx, k, "note-source-a", "Source A")
			},
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.SourceObjectAuthority
			},
			summaryID: "sourceObjectAuthorityValidation",
		},
		{
			name:    "admission candidate",
			action:  domain.ActionValidateAdmissionCandidate,
			request: validAdmissionCandidateRequest,
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.AdmissionCandidateValidation
			},
			summaryID: "admissionCandidateValidation",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			k, _, auditSink := newTestKernel()
			if tc.setup != nil {
				tc.setup(ctx, k)
			}
			req := tc.request()
			req.ID = "forge-k-contract-" + string(tc.action)
			req.IdempotencyKey = ""

			res, err := k.Process(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !res.Success {
				t.Fatalf("expected validation success, got %#v", res)
			}
			assertForgeKValidationContract(t, string(tc.action), res.StateSummary)

			nested, ok := res.StateSummary[tc.summaryID].(map[string]any)
			if !ok {
				t.Fatalf("missing nested summary %q: %#v", tc.summaryID, res.StateSummary)
			}
			assertForgeKValidationContract(t, string(tc.action), nested)

			if len(auditSink.Records) == 0 {
				t.Fatal("expected audit record")
			}
			auditSummary := tc.audit(auditSink.Records[len(auditSink.Records)-1])
			if auditSummary == nil {
				t.Fatalf("missing audit summary for %s", tc.action)
			}
			assertForgeKValidationContract(t, string(tc.action), auditSummary)
		})
	}
}

func assertForgeKValidationContract(t *testing.T, action string, summary map[string]any) {
	t.Helper()
	activation, ok := summary["forgeKActivation"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKActivation in %#v", summary)
	}
	if activation["mode"] != ForgeKActivationModePartialLiveEnforcement {
		t.Fatalf("mode=%#v, want %q", activation["mode"], ForgeKActivationModePartialLiveEnforcement)
	}
	if activation["liveOwner"] != ForgeKActivationOwnerControlLane {
		t.Fatalf("liveOwner=%#v, want %q", activation["liveOwner"], ForgeKActivationOwnerControlLane)
	}
	if activation["action"] != action {
		t.Fatalf("action=%#v, want %q", activation["action"], action)
	}
	if activation["simulatorAuthority"] != false || activation["liveKernelAuthority"] != false || activation["shadowAuthoritative"] != false {
		t.Fatalf("activation summary claimed forbidden authority: %#v", activation)
	}

	noEffect, ok := summary["forgeKNoEffect"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKNoEffect in %#v", summary)
	}
	for _, key := range []string{
		"memoryMutation",
		"runtimeMutation",
		"modelRuntimeCall",
		"evidenceAdmission",
		"contextCompilation",
		"gatewayExecution",
		"retrievalExecution",
		"liveAuthorityMigration",
	} {
		if noEffect[key] != false {
			t.Fatalf("%s=%#v, want false in %#v", key, noEffect[key], noEffect)
		}
	}
}
