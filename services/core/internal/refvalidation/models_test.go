package refvalidation

import "testing"

func TestValidateRefsNormalizesAndDeduplicatesStableRefs(t *testing.T) {
	req := ValidationRequest{
		ResultID:    " result-a ",
		WorkspaceID: " ws-main ",
		Refs: []ObjectRef{
			{RefType: " memory_note ", RefID: " note-b "},
			{RefType: "memory_note", RefID: "note-a", WorkspaceID: "ws-main"},
			{RefType: "memory_note", RefID: "note-a", WorkspaceID: "ws-main"},
		},
	}
	res := ValidateRefs(req)
	if !res.Passed {
		t.Fatalf("expected validation to pass, got %#v", res)
	}
	if len(res.NormalizedRefs) != 2 {
		t.Fatalf("expected duplicate refs to be normalized away, got %#v", res.NormalizedRefs)
	}
	if res.NormalizedRefs[0].RefID != "note-a" || res.NormalizedRefs[1].RefID != "note-b" {
		t.Fatalf("refs not sorted deterministically: %#v", res.NormalizedRefs)
	}
	for _, ref := range res.NormalizedRefs {
		if ref.WorkspaceID != "ws-main" {
			t.Fatalf("workspace not normalized onto ref: %#v", ref)
		}
	}
}

func TestValidateRefsFailsClosedForMissingWorkspace(t *testing.T) {
	res := ValidateRefs(ValidationRequest{
		ResultID: "result-a",
		Refs: []ObjectRef{
			{RefType: "memory_note", RefID: "note-a"},
		},
	})
	if res.Passed {
		t.Fatalf("expected missing workspace to fail")
	}
	if !hasFailure(res.Failures, GateWorkspace) {
		t.Fatalf("missing workspace gate failure: %#v", res.Failures)
	}
}

func TestValidateRefsFailsClosedForUnknownTypeAndSecretLikeID(t *testing.T) {
	res := ValidateRefs(ValidationRequest{
		ResultID:    "result-a",
		WorkspaceID: "ws-main",
		Refs: []ObjectRef{
			{RefType: "raw_prompt", RefID: "prompt-a"},
			{RefType: "memory_note", RefID: "token=secret"},
		},
	})
	if res.Passed {
		t.Fatalf("expected invalid refs to fail")
	}
	if !hasFailure(res.Failures, GateRefType) {
		t.Fatalf("missing ref type failure: %#v", res.Failures)
	}
	if !hasFailure(res.Failures, GateSafeRefID) {
		t.Fatalf("missing safe ref id failure: %#v", res.Failures)
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
