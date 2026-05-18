package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestValidateAdmissionCandidateSucceedsWithoutEvidenceAdmission(t *testing.T) {
	ctx := context.Background()
	k, _, auditSink := newTestKernel()

	req := validAdmissionCandidateRequest()
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected admission candidate validation success, got %#v", res)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("admission candidate validation must not commit semantic objects, got %v", res.CommittedObjectIDs)
	}
	if res.StateSummary["passed"] != true {
		t.Fatalf("expected passed state summary, got %#v", res.StateSummary)
	}
	assertAdmissionCandidateNoForbiddenEffects(t, res.StateSummary)
	assertForgeKValidationContract(t, string(domain.ActionValidateAdmissionCandidate), res.StateSummary)
	nested, ok := res.StateSummary["admissionCandidateValidation"].(map[string]any)
	if !ok {
		t.Fatalf("missing nested admissionCandidateValidation: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateAdmissionCandidate), nested)
	if nested["decision"] != AdmissionDecisionAccepted || nested["normalizedEvidenceRefCount"] != 1 {
		t.Fatalf("unexpected nested decision: %#v", nested)
	}
	if len(auditSink.Records) == 0 {
		t.Fatal("expected audit record")
	}
	auditDecision := auditSink.Records[len(auditSink.Records)-1].AdmissionCandidateValidation
	if auditDecision["decision"] != AdmissionDecisionAccepted || auditDecision["evidenceAdmission"] != false {
		t.Fatalf("audit missing admission candidate decision: %#v", auditDecision)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateAdmissionCandidate), auditDecision)
}

func TestValidateAdmissionCandidateRejectsForbiddenAuthorityClaim(t *testing.T) {
	ctx := context.Background()
	k, store, auditSink := newTestKernel()
	req := validAdmissionCandidateRequest()
	req.ID = "admission-candidate-forbidden-claim"
	req.IdempotencyKey = "admission-candidate-forbidden-claim"
	req.Payload["claims"] = map[string]any{"admit_evidence": true}

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected forbidden authority claim to fail")
	}
	if res.DeterministicErrCode != domain.ErrInvalidPayload {
		t.Fatalf("expected invalid payload, got %s", res.DeterministicErrCode)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("failed validation committed ids: %v", res.CommittedObjectIDs)
	}
	if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
		t.Fatal("failed admission validation must not persist idempotency state")
	}
	assertAdmissionCandidateRejectedWithoutEffects(t, res, auditSink)
}

func TestValidateAdmissionCandidateRejectsCrossWorkspaceEvidence(t *testing.T) {
	ctx := context.Background()
	k, _, auditSink := newTestKernel()
	req := validAdmissionCandidateRequest()
	req.ID = "admission-candidate-cross-workspace"
	req.Payload["evidence_refs"] = []any{
		map[string]any{"ref_type": "memory_note", "ref_id": "note-other", "workspace_id": "ws-other"},
	}

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected cross-workspace evidence ref to fail")
	}
	if res.DeterministicErrCode != domain.ErrInvalidPayload {
		t.Fatalf("expected invalid payload, got %s", res.DeterministicErrCode)
	}
	assertAdmissionCandidateRejectedWithoutEffects(t, res, auditSink)
}

func TestValidateAdmissionCandidateCapabilityDeniedForProposeOnlySource(t *testing.T) {
	ctx := context.Background()
	k, _, _ := newTestKernel()
	req := validAdmissionCandidateRequest()
	req.ID = "admission-candidate-future-iris"
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

func validAdmissionCandidateRequest() domain.SyscallRequest {
	req := validBaseRequest(domain.ActionValidateAdmissionCandidate)
	req.ID = "admission-candidate-pass"
	req.Payload = map[string]any{
		"workspace_id":   "ws-main",
		"case_id":        "case-a",
		"admission_mode": "admission_candidate",
		"evidence_refs": []any{
			map[string]any{"ref_type": "memory_note", "ref_id": "note-evidence-a", "workspace_id": "ws-main"},
		},
		"source_refs": []any{
			map[string]any{"ref_type": "case_packet", "ref_id": "case-source-a", "workspace_id": "ws-main"},
		},
		"policy_refs": []any{
			map[string]any{"ref_type": "semantic_object", "ref_id": "policy-a", "workspace_id": "ws-main"},
		},
		"provenance_refs": []any{
			map[string]any{"ref_type": "diagnostic_report", "ref_id": "journal-a", "workspace_id": "ws-main"},
		},
		"claims": map[string]any{
			"admit_evidence": false,
		},
	}
	return req
}

func assertAdmissionCandidateRejectedWithoutEffects(t *testing.T, res domain.SyscallResult, auditSink *InMemoryAuditSink) {
	t.Helper()
	assertAdmissionCandidateNoForbiddenEffects(t, res.StateSummary)
	assertForgeKValidationContract(t, string(domain.ActionValidateAdmissionCandidate), res.StateSummary)
	nested, ok := res.StateSummary["admissionCandidateValidation"].(map[string]any)
	if !ok {
		t.Fatalf("missing nested admissionCandidateValidation summary: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateAdmissionCandidate), nested)
	if nested["accepted"] != false || nested["decision"] != AdmissionDecisionRejected {
		t.Fatalf("expected rejected nested decision, got %#v", nested)
	}
	if len(auditSink.Records) == 0 {
		t.Fatal("expected rejected validation audit record")
	}
	auditRecord := auditSink.Records[len(auditSink.Records)-1]
	if auditRecord.Success {
		t.Fatalf("expected rejected audit record, got %#v", auditRecord)
	}
	auditDecision := auditRecord.AdmissionCandidateValidation
	if auditDecision == nil {
		t.Fatalf("audit missing admission candidate summary: %#v", auditRecord)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateAdmissionCandidate), auditDecision)
	if auditDecision["accepted"] != false || auditDecision["decision"] != AdmissionDecisionRejected {
		t.Fatalf("expected rejected audit decision, got %#v", auditDecision)
	}
}

func assertAdmissionCandidateNoForbiddenEffects(t *testing.T, summary map[string]any) {
	t.Helper()
	if summary["canonicalCommit"] != false ||
		summary["memoryMutation"] != false ||
		summary["runtimeMutation"] != false ||
		summary["modelRuntimeCall"] != false ||
		summary["gatewayExecution"] != false ||
		summary["evidenceAdmission"] != false ||
		summary["contextCompilation"] != false ||
		summary["liveAuthorityMigration"] != false {
		t.Fatalf("admission candidate validation claimed forbidden effects: %#v", summary)
	}
}
