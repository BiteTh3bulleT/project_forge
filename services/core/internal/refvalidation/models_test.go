package refvalidation

import (
	"strings"
	"testing"
)

func TestAllowedRefTypesReturnsCopiedCanonicalList(t *testing.T) {
	types := AllowedRefTypes()
	if len(types) == 0 {
		t.Fatal("expected allowed ref types")
	}
	seen := map[string]struct{}{}
	for _, refType := range types {
		if refType == "" {
			t.Fatal("allowed ref type list contains empty type")
		}
		if refType != strings.ToLower(strings.TrimSpace(refType)) {
			t.Fatalf("allowed ref type is not canonical: %q", refType)
		}
		if _, ok := seen[refType]; ok {
			t.Fatalf("duplicate allowed ref type: %q", refType)
		}
		seen[refType] = struct{}{}
	}
	types[0] = "mutated_by_test"
	if AllowedRefTypes()[0] == "mutated_by_test" {
		t.Fatal("AllowedRefTypes must return a copy")
	}
}

func TestAllowedRefTypesPassValidation(t *testing.T) {
	for _, refType := range AllowedRefTypes() {
		t.Run(refType, func(t *testing.T) {
			res := ValidateRefs(ValidationRequest{
				ResultID:    "result-" + refType,
				WorkspaceID: "ws-main",
				Refs: []ObjectRef{
					{RefType: strings.ToUpper(refType), RefID: "ref-" + strings.ReplaceAll(refType, "_", "-")},
				},
			})
			if !res.Passed {
				t.Fatalf("expected allowed ref type %q to pass, got %#v", refType, res)
			}
			if len(res.NormalizedRefs) != 1 || res.NormalizedRefs[0].RefType != refType {
				t.Fatalf("expected canonical normalized ref type %q, got %#v", refType, res.NormalizedRefs)
			}
		})
	}
}

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

func TestValidateRefsFailsClosedForLaneClosureCases(t *testing.T) {
	cases := []struct {
		name string
		req  ValidationRequest
		gate string
	}{
		{
			name: "empty refs",
			req: ValidationRequest{
				ResultID:    "result-a",
				WorkspaceID: "ws-main",
				Refs:        nil,
			},
			gate: GateRefs,
		},
		{
			name: "missing ref id",
			req: ValidationRequest{
				ResultID:    "result-a",
				WorkspaceID: "ws-main",
				Refs: []ObjectRef{
					{RefType: "memory_note", RefID: ""},
				},
			},
			gate: GateRefID,
		},
		{
			name: "unknown ref type",
			req: ValidationRequest{
				ResultID:    "result-a",
				WorkspaceID: "ws-main",
				Refs: []ObjectRef{
					{RefType: "raw_prompt", RefID: "prompt-a"},
				},
			},
			gate: GateRefType,
		},
		{
			name: "unsafe ref id",
			req: ValidationRequest{
				ResultID:    "result-a",
				WorkspaceID: "ws-main",
				Refs: []ObjectRef{
					{RefType: "memory_note", RefID: "token=secret"},
				},
			},
			gate: GateSafeRefID,
		},
		{
			name: "workspace mismatch",
			req: ValidationRequest{
				ResultID:    "result-a",
				WorkspaceID: "ws-main",
				Refs: []ObjectRef{
					{RefType: "memory_note", RefID: "note-a", WorkspaceID: "ws-other"},
				},
			},
			gate: GateScope,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := ValidateRefs(tc.req)
			if res.Passed {
				t.Fatalf("expected validation to fail closed")
			}
			if !hasFailure(res.Failures, tc.gate) {
				t.Fatalf("missing gate %s in failures: %#v", tc.gate, res.Failures)
			}
			if len(res.NormalizedRefs) != 0 {
				t.Fatalf("failed validation should not normalize invalid refs: %#v", res.NormalizedRefs)
			}
		})
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
