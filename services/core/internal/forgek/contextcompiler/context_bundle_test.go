package contextcompiler

import (
	"reflect"
	"testing"
)

func TestContextBundleOrdersBlocksAndComputesStableHashes(t *testing.T) {
	service := NewService()
	request := ContextCompileRequest{
		RequestID:             "request-a",
		BundleID:              "bundle-a",
		WorkspaceID:           "workspace-a",
		CaseID:                "case-a",
		AdmittedExhibitRefs:   []string{"exhibit-a"},
		ContradictionRefs:     []string{"contradiction-a"},
		IncludeContradictions: true,
		UserMessage:           "what changed?",
		CreatedBy:             "operator",
		CreatedAt:             testBlockInput().CreatedAt,
	}
	result, err := service.Compile(request)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if len(result.Bundle.Blocks) < 3 {
		t.Fatalf("expected multiple blocks, got %#v", result.Bundle.Blocks)
	}
	if result.Bundle.Blocks[len(result.Bundle.Blocks)-1].BlockType != BlockUserMessage {
		t.Fatalf("user message did not sort last: %#v", result.Bundle.Blocks)
	}
	if result.Bundle.BundleHash == "" || result.Bundle.StablePrefixHash == "" || result.Bundle.VolatileSuffixHash == "" {
		t.Fatalf("bundle hashes missing: %#v", result.Bundle)
	}
	if result.Bundle.IsCanonicalTruth() || result.Bundle.IsModelResponse() || result.Bundle.IsKVCache() {
		t.Fatal("bundle claimed truth/model/KV authority")
	}

	second, err := service.Compile(request)
	if err != nil {
		t.Fatalf("second compile failed: %v", err)
	}
	if result.Bundle.BundleHash != second.Bundle.BundleHash ||
		result.Bundle.TokenInputHash != second.Bundle.TokenInputHash ||
		!reflect.DeepEqual(result.Bundle.SourceRefs, second.Bundle.SourceRefs) {
		t.Fatalf("bundle is not deterministic: first=%#v second=%#v", result.Bundle, second.Bundle)
	}
	request.CurrentTaskSummary = "different task"
	changed, err := service.Compile(request)
	if err != nil {
		t.Fatalf("changed compile failed: %v", err)
	}
	if changed.Bundle.BundleHash == result.Bundle.BundleHash {
		t.Fatal("bundle hash did not change when block content changed")
	}
}

func TestLayoutVersionAffectsBundleHashButCreatedAtDoesNot(t *testing.T) {
	service := NewService()
	request := ContextCompileRequest{
		RequestID:           "request-a",
		BundleID:            "bundle-a",
		WorkspaceID:         "workspace-a",
		CaseID:              "case-a",
		AdmittedExhibitRefs: []string{"exhibit-a"},
		CreatedBy:           "operator",
		CreatedAt:           testBlockInput().CreatedAt,
	}
	first, err := service.Compile(request)
	if err != nil {
		t.Fatalf("first compile failed: %v", err)
	}
	request.CreatedAt = request.CreatedAt.AddDate(0, 0, 1)
	second, err := service.Compile(request)
	if err != nil {
		t.Fatalf("second compile failed: %v", err)
	}
	if first.Bundle.BundleHash != second.Bundle.BundleHash {
		t.Fatal("created_at affected bundle hash")
	}
	request.LayoutVersion = "context-layout-v2"
	changed, err := service.Compile(request)
	if err != nil {
		t.Fatalf("layout compile failed: %v", err)
	}
	if first.Bundle.BundleHash == changed.Bundle.BundleHash {
		t.Fatal("layout version did not affect bundle hash")
	}
}
