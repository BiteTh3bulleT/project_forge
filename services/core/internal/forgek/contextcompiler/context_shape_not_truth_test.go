package contextcompiler

import "testing"

func TestCompilePreservesShapeNotTruthInvariants(t *testing.T) {
	service := NewService()
	result, err := service.Compile(ContextCompileRequest{
		RequestID:           "request-a",
		BundleID:            "bundle-a",
		WorkspaceID:         "workspace-a",
		CaseID:              "case-a",
		AdmittedExhibitRefs: []string{"admitted-a"},
		RejectedExhibitRefs: []string{"rejected-a"},
		DerivedObjectRefs:   []string{"derived-a"},
		CreatedBy:           "operator",
		CreatedAt:           testBlockInput().CreatedAt,
	})
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if hasBlockType(result.Blocks, BlockRejectedEvidenceSummary) {
		t.Fatal("rejected evidence appeared without explicit rejected summary request")
	}
	for _, block := range result.Blocks {
		if block.IsCanonicalTruth() || block.IsKVCache() {
			t.Fatalf("block claimed truth or KV cache authority: %#v", block)
		}
	}
	if result.Bundle.IsCanonicalTruth() || result.Bundle.IsModelResponse() || result.Bundle.IsKVCache() {
		t.Fatalf("bundle claimed wrong authority: %#v", result.Bundle)
	}
}
