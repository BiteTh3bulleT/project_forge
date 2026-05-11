package semanticvalidation

import (
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/refvalidation"
)

func TestValidateOperationNormalizesRefsWithoutAuthorityClaims(t *testing.T) {
	res := ValidateOperation(OperationRequest{
		ResultID:      "result-a",
		WorkspaceID:   "ws-main",
		OperationType: " derive ",
		SourceRefs: []refvalidation.ObjectRef{
			{RefType: "semantic_object", RefID: "obj-b"},
			{RefType: "semantic_object", RefID: "obj-a"},
			{RefType: "semantic_object", RefID: "obj-a"},
		},
		DerivedRefs: []refvalidation.ObjectRef{
			{RefType: "semantic_object", RefID: "derived-a"},
		},
	})
	if !res.Passed {
		t.Fatalf("expected semantic operation shape to pass, got %#v", res)
	}
	if res.OperationType != "derive" {
		t.Fatalf("operation type was not normalized: %#v", res)
	}
	if len(res.NormalizedSourceRefs) != 2 || res.NormalizedSourceRefs[0].RefID != "obj-a" || res.NormalizedSourceRefs[1].RefID != "obj-b" {
		t.Fatalf("source refs were not normalized deterministically: %#v", res.NormalizedSourceRefs)
	}
	if res.MemoryMutation || res.ModelRuntimeCall || res.EvidenceAdmission || res.ContextCompilation {
		t.Fatalf("validator claimed forbidden authority: %#v", res)
	}
}

func TestValidateOperationRejectsForbiddenClaimsAndMissingSources(t *testing.T) {
	res := ValidateOperation(OperationRequest{
		ResultID:      "result-a",
		WorkspaceID:   "ws-main",
		OperationType: "derive",
		Claims: map[string]bool{
			"write_memory": true,
			"call_model":   true,
		},
	})
	if res.Passed {
		t.Fatal("expected forbidden claims and missing source refs to fail")
	}
	if !hasFailure(res.Failures, GateSourceRefs) {
		t.Fatalf("missing source refs failure: %#v", res.Failures)
	}
	if !hasFailure(res.Failures, GateNoAuthorityClaim) {
		t.Fatalf("missing no-authority failure: %#v", res.Failures)
	}
}

func TestForbiddenAuthorityClaimsRejectAfterNormalization(t *testing.T) {
	for _, claim := range ForbiddenAuthorityClaims() {
		t.Run(claim, func(t *testing.T) {
			res := ValidateOperation(OperationRequest{
				ResultID:      "result-a",
				WorkspaceID:   "ws-main",
				OperationType: "derive",
				SourceRefs: []refvalidation.ObjectRef{
					{RefType: "semantic_object", RefID: "obj-a"},
				},
				Claims: map[string]bool{
					"  " + strings.ToUpper(claim) + "  ": true,
				},
			})
			if res.Passed {
				t.Fatalf("expected forbidden claim %q to fail", claim)
			}
			if !hasFailure(res.Failures, GateNoAuthorityClaim) {
				t.Fatalf("missing no-authority failure for %q: %#v", claim, res.Failures)
			}
			if res.MemoryMutation || res.ModelRuntimeCall || res.EvidenceAdmission || res.ContextCompilation || res.LiveAuthorityMigration {
				t.Fatalf("validator claimed forbidden side effect for %q: %#v", claim, res)
			}
		})
	}
}

func TestForbiddenAuthorityClaimsFalseValuesDoNotReject(t *testing.T) {
	claims := map[string]bool{}
	for _, claim := range ForbiddenAuthorityClaims() {
		claims[claim] = false
	}
	res := ValidateOperation(OperationRequest{
		ResultID:      "result-a",
		WorkspaceID:   "ws-main",
		OperationType: "derive",
		SourceRefs: []refvalidation.ObjectRef{
			{RefType: "semantic_object", RefID: "obj-a"},
		},
		Claims: claims,
	})
	if !res.Passed {
		t.Fatalf("false forbidden claims must not reject by themselves: %#v", res.Failures)
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
