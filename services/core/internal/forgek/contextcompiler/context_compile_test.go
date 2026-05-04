package contextcompiler

import "testing"

func TestCompileEmitsExpectedBlocksAndWarnings(t *testing.T) {
	service := NewService()
	result, err := service.Compile(ContextCompileRequest{
		RequestID:                      "request-a",
		BundleID:                       "bundle-a",
		WorkspaceID:                    "workspace-a",
		CaseID:                         "case-a",
		SourceObjectRefs:               []string{"case-a"},
		AdmittedExhibitRefs:            []string{"admitted-a"},
		RejectedExhibitRefs:            []string{"rejected-a"},
		ContradictionRefs:              []string{"contradiction-a"},
		SemanticOperationRefs:          []string{"operation-a"},
		DerivedObjectRefs:              []string{"derived-a"},
		PalaceRouteRefs:                []string{"route-a"},
		IncludeRejectedEvidenceSummary: true,
		IncludeContradictions:          true,
		CurrentTaskSummary:             "complete phase 7",
		UserMessage:                    "compile context",
		CreatedBy:                      "operator",
		CreatedAt:                      testBlockInput().CreatedAt,
	})
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	for _, blockType := range []ContextBlockType{
		BlockCaseSummary,
		BlockPalaceRouteSummary,
		BlockAdmittedEvidence,
		BlockRejectedEvidenceSummary,
		BlockContradictionSummary,
		BlockSemanticOperationSummary,
		BlockCurrentTask,
		BlockUserMessage,
	} {
		if !hasBlockType(result.Blocks, blockType) {
			t.Fatalf("missing block type %s in %#v", blockType, result.Blocks)
		}
	}
	for _, warning := range []string{WarningRejectedEvidenceSummary, WarningContradictionsPresent, WarningVolatileUserMessagePresent} {
		if !containsString(result.Warnings, warning) {
			t.Fatalf("missing warning %s in %#v", warning, result.Warnings)
		}
	}
	if hasBlockType(result.Blocks, BlockFutureKVPlaceholder) {
		t.Fatal("compile created KV cache placeholder unexpectedly")
	}
}

func TestCompileRejectsUnscopedRequestAndDoesNotFetchOrCallRuntime(t *testing.T) {
	service := NewService()
	if _, err := service.Compile(ContextCompileRequest{
		RequestID:   "request-a",
		BundleID:    "bundle-a",
		WorkspaceID: "workspace-a",
		UserMessage: "unscoped",
		CreatedBy:   "operator",
		CreatedAt:   testBlockInput().CreatedAt,
	}); err == nil {
		t.Fatal("unscoped compile succeeded")
	}
	result, err := service.Compile(ContextCompileRequest{
		RequestID:           "request-b",
		BundleID:            "bundle-b",
		WorkspaceID:         "workspace-a",
		SourceRefs:          []string{"external://not-fetched"},
		AdmittedExhibitRefs: []string{"admitted-a"},
		CreatedBy:           "operator",
		CreatedAt:           testBlockInput().CreatedAt,
	})
	if err != nil {
		t.Fatalf("scoped compile failed: %v", err)
	}
	if result.Bundle.IsModelResponse() || result.Bundle.IsKVCache() {
		t.Fatal("compiler produced model response or KV cache authority")
	}
}

func hasBlockType(blocks []ContextBlock, blockType ContextBlockType) bool {
	for _, block := range blocks {
		if block.BlockType == blockType {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
