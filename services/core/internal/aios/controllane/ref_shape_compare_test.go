package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestCompareRefShapeReportsDriftWithoutLiveMutation(t *testing.T) {
	ctx := context.Background()
	k, _, auditSink := newTestKernel()
	req := validRefShapeCompareRequest()
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected ref shape comparison success, got %#v", res)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("ref shape comparison must not commit objects, got %v", res.CommittedObjectIDs)
	}
	if res.StateSummary["match"] != false ||
		res.StateSummary["memoryMutation"] != false ||
		res.StateSummary["runtimeMutation"] != false ||
		res.StateSummary["liveAuthorityMigration"] != false {
		t.Fatalf("unexpected comparison summary: %#v", res.StateSummary)
	}
	activation, ok := res.StateSummary["forgeKActivation"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKActivation summary: %#v", res.StateSummary)
	}
	if activation["mode"] != ForgeKActivationModePartialLiveEnforcement ||
		activation["action"] != string(domain.ActionCompareRefShape) {
		t.Fatalf("unexpected activation summary: %#v", activation)
	}
	noEffect, ok := res.StateSummary["forgeKNoEffect"].(map[string]any)
	if !ok || noEffect["memoryMutation"] != false || noEffect["retrievalExecution"] != false {
		t.Fatalf("unexpected no-effect summary: %#v", res.StateSummary["forgeKNoEffect"])
	}
	auditDecision := auditSink.Records[len(auditSink.Records)-1].RefShapeComparison
	if auditDecision["decision"] != RefShapeCompareDecisionDrift || auditDecision["liveAuthorityMigration"] != false {
		t.Fatalf("audit missing ref shape comparison decision: %#v", auditDecision)
	}
	auditActivation := auditSink.Records[len(auditSink.Records)-1].RefShapeComparison["forgeKActivation"].(map[string]any)
	if auditActivation["mode"] != ForgeKActivationModePartialLiveEnforcement {
		t.Fatalf("audit activation summary missing partial enforcement: %#v", auditActivation)
	}
}

func TestCompareRefShapeRejectsInvalidObservedRefs(t *testing.T) {
	ctx := context.Background()
	k, store, auditSink := newTestKernel()
	req := validRefShapeCompareRequest()
	req.ID = "ref-compare-bad"
	req.IdempotencyKey = "ref-compare-bad"
	req.Payload["observed_refs"] = []any{
		map[string]any{"ref_type": "raw_prompt", "ref_id": "token=secret"},
	}
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected invalid observed refs to fail")
	}
	if res.DeterministicErrCode != domain.ErrInvalidPayload {
		t.Fatalf("expected invalid payload, got %s", res.DeterministicErrCode)
	}
	if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
		t.Fatal("failed comparison must not persist idempotency state")
	}
	assertRefShapeComparisonRejectedWithoutEffects(t, res, auditSink)
}

func validRefShapeCompareRequest() domain.SyscallRequest {
	req := validBaseRequest(domain.ActionCompareRefShape)
	req.ID = "ref-compare-pass"
	req.Payload = map[string]any{
		"workspace_id": "ws-main",
		"candidate_refs": []any{
			map[string]any{"ref_type": "memory_note", "ref_id": "note-b"},
			map[string]any{"ref_type": "memory_note", "ref_id": "note-a"},
		},
		"observed_refs": []any{
			map[string]any{"ref_type": "memory_note", "ref_id": "note-b"},
			map[string]any{"ref_type": "memory_note", "ref_id": "note-c"},
		},
	}
	return req
}

func assertRefShapeComparisonRejectedWithoutEffects(t *testing.T, res domain.SyscallResult, auditSink *InMemoryAuditSink) {
	t.Helper()
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("rejected comparison committed ids: %v", res.CommittedObjectIDs)
	}
	if res.StateSummary["memoryMutation"] != false || res.StateSummary["runtimeMutation"] != false || res.StateSummary["liveAuthorityMigration"] != false {
		t.Fatalf("rejected comparison claimed mutation/authority migration: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionCompareRefShape), res.StateSummary)
	nested, ok := res.StateSummary["refShapeComparison"].(map[string]any)
	if !ok {
		t.Fatalf("missing nested refShapeComparison summary: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionCompareRefShape), nested)
	if nested["accepted"] != false || nested["decision"] != RefShapeCompareDecisionRejected {
		t.Fatalf("expected rejected nested decision, got %#v", nested)
	}
	if nested["memoryMutation"] != false || nested["runtimeMutation"] != false || nested["liveAuthorityMigration"] != false {
		t.Fatalf("nested comparison claimed mutation/authority migration: %#v", nested)
	}
	if len(auditSink.Records) == 0 {
		t.Fatal("expected rejected comparison audit record")
	}
	auditRecord := auditSink.Records[len(auditSink.Records)-1]
	if auditRecord.Success {
		t.Fatalf("expected rejected audit record, got %#v", auditRecord)
	}
	auditDecision := auditRecord.RefShapeComparison
	if auditDecision == nil {
		t.Fatalf("audit missing ref shape comparison summary: %#v", auditRecord)
	}
	assertForgeKValidationContract(t, string(domain.ActionCompareRefShape), auditDecision)
	if auditDecision["accepted"] != false || auditDecision["decision"] != RefShapeCompareDecisionRejected {
		t.Fatalf("expected rejected audit decision, got %#v", auditDecision)
	}
	if auditDecision["memoryMutation"] != false || auditDecision["runtimeMutation"] != false || auditDecision["liveAuthorityMigration"] != false {
		t.Fatalf("audit comparison claimed mutation/authority migration: %#v", auditDecision)
	}
}
