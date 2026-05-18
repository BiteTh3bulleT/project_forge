package admissionvalidation

import (
	"testing"

	"forge/projectforge/services/core/internal/refvalidation"
)

func TestAllowedAdmissionModesReturnsCopy(t *testing.T) {
	modes := AllowedAdmissionModes()
	if len(modes) == 0 {
		t.Fatal("expected admission modes")
	}
	modes[0] = "mutated"
	if AllowedAdmissionModes()[0] == "mutated" {
		t.Fatal("AllowedAdmissionModes must return a copy")
	}
}

func TestValidateAdmissionNormalizesRefsWithoutAuthorityClaims(t *testing.T) {
	res := ValidateAdmission(AdmissionRequest{
		ResultID:      "result-a",
		WorkspaceID:   "ws-a",
		CaseID:        "case-a",
		AdmissionMode: "ADMISSION_CANDIDATE",
		EvidenceRefs: []refvalidation.ObjectRef{
			{RefType: "exhibit", RefID: "exhibit-b"},
			{RefType: "exhibit", RefID: "exhibit-a"},
			{RefType: "exhibit", RefID: "exhibit-a"},
		},
		SourceRefs: []refvalidation.ObjectRef{
			{RefType: "memory_note", RefID: "note-a"},
		},
		PolicyRefs: []refvalidation.ObjectRef{
			{RefType: "diagnostic_report", RefID: "policy-a"},
		},
		Claims: map[string]bool{
			"admission_decision": false,
		},
	})

	if !res.Passed {
		t.Fatalf("expected admission validation to pass, got failures: %#v", res.Failures)
	}
	if res.AdmissionMode != "admission_candidate" {
		t.Fatalf("expected normalized mode, got %q", res.AdmissionMode)
	}
	if len(res.NormalizedEvidenceRefs) != 2 || res.NormalizedEvidenceRefs[0].RefID != "exhibit-a" || res.NormalizedEvidenceRefs[1].RefID != "exhibit-b" {
		t.Fatalf("evidence refs were not normalized deterministically: %#v", res.NormalizedEvidenceRefs)
	}
	if res.CanonicalCommit || res.MemoryMutation || res.ModelRuntimeCall || res.GatewayExecution || res.ContextCompilation || res.LiveAuthorityMigration {
		t.Fatalf("pure admission validation must not claim authority effects: %#v", res)
	}
}

func TestValidateAdmissionRejectsMissingRequiredShape(t *testing.T) {
	res := ValidateAdmission(AdmissionRequest{})

	if res.Passed {
		t.Fatal("expected missing shape to fail")
	}
	for _, gate := range []string{GateWorkspace, GateCase, GateAdmissionMode, GateEvidenceRefs} {
		if !hasFailure(res.Failures, gate) {
			t.Fatalf("expected gate %q in failures: %#v", gate, res.Failures)
		}
	}
	if len(res.NormalizedEvidenceRefs) != 0 {
		t.Fatalf("failed validation should not produce evidence refs: %#v", res.NormalizedEvidenceRefs)
	}
}

func TestValidateAdmissionRejectsUnsafeRefsAndAuthorityClaims(t *testing.T) {
	res := ValidateAdmission(AdmissionRequest{
		ResultID:      "result-a",
		WorkspaceID:   "ws-a",
		CaseID:        "case-a",
		AdmissionMode: "admission_shadow",
		EvidenceRefs: []refvalidation.ObjectRef{
			{RefType: "raw_prompt", RefID: "prompt-a"},
			{RefType: "exhibit", RefID: "token=secret"},
		},
		Claims: map[string]bool{
			"admit_evidence": true,
			"call_model":     true,
		},
	})

	if res.Passed {
		t.Fatal("expected unsafe refs and authority claims to fail")
	}
	if !hasFailure(res.Failures, GateRefValidation) {
		t.Fatalf("expected ref validation failure, got %#v", res.Failures)
	}
	if !hasFailure(res.Failures, GateNoAuthorityClaim) {
		t.Fatalf("expected authority claim failure, got %#v", res.Failures)
	}
	if res.CanonicalCommit || res.MemoryMutation || res.ModelRuntimeCall || res.GatewayExecution || res.ContextCompilation || res.LiveAuthorityMigration {
		t.Fatalf("failed pure validation still must not claim authority effects: %#v", res)
	}
}

func TestValidateAdmissionRejectsCrossWorkspaceRefs(t *testing.T) {
	res := ValidateAdmission(AdmissionRequest{
		ResultID:      "result-a",
		WorkspaceID:   "ws-a",
		CaseID:        "case-a",
		AdmissionMode: "admission_only",
		EvidenceRefs: []refvalidation.ObjectRef{
			{RefType: "exhibit", RefID: "exhibit-a", WorkspaceID: "ws-other"},
		},
	})

	if res.Passed {
		t.Fatal("expected cross-workspace ref to fail")
	}
	if !hasFailure(res.Failures, GateRefValidation) {
		t.Fatalf("expected ref validation failure, got %#v", res.Failures)
	}
}

func hasFailure(failures []ValidationFailure, gate string) bool {
	for _, failure := range failures {
		if failure.Gate == gate {
			return true
		}
	}
	return false
}
