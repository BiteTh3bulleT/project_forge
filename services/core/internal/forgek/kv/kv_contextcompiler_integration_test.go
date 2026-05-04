package kv

import (
	"testing"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
)

func TestKVRegistersAndLooksUpCompiledContextBundleByRefs(t *testing.T) {
	compiler := contextcompiler.NewService()
	compiled, err := compiler.Compile(contextcompiler.ContextCompileRequest{
		RequestID:           "request-a",
		BundleID:            "bundle-a",
		WorkspaceID:         "workspace-a",
		CaseID:              "case-a",
		SourceObjectRefs:    []string{"case-a"},
		AdmittedExhibitRefs: []string{"exhibit-a"},
		CurrentTaskSummary:  "compile context for kv identity",
		CreatedBy:           "operator",
		CreatedAt:           testKVTime(),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	input := validManifestInput()
	input.BundleID = compiled.Bundle.BundleID
	input.BundleHash = compiled.Bundle.BundleHash
	input.StablePrefixHash = compiled.Bundle.StablePrefixHash
	input.VolatileSuffixHash = compiled.Bundle.VolatileSuffixHash
	input.TokenInputHash = compiled.Bundle.TokenInputHash
	service := NewService()
	manifest, err := service.RegisterManifest(input)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	request := validLookupRequest()
	request.BundleID = compiled.Bundle.BundleID
	request.BundleHash = compiled.Bundle.BundleHash
	request.StablePrefixHash = compiled.Bundle.StablePrefixHash
	request.VolatileSuffixHash = compiled.Bundle.VolatileSuffixHash
	request.TokenInputHash = compiled.Bundle.TokenInputHash
	hit, err := service.Lookup(request, "validation-1", testKVTime())
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !hit.Hit || hit.CacheID != manifest.CacheID {
		t.Fatalf("expected compiled bundle identity hit: %#v", hit)
	}

	changedBundle := request
	changedBundle.BundleHash = "changed"
	if result, err := service.Lookup(changedBundle, "validation-2", testKVTime()); err != nil || result.Hit || !contains(result.FailedGates, GateContextBundle) {
		t.Fatalf("changed bundle hash should miss: result=%#v err=%v", result, err)
	}
	changedLayout := request
	changedLayout.PromptLayoutHash = "changed"
	if result, err := service.Lookup(changedLayout, "validation-3", testKVTime()); err != nil || result.Hit || !contains(result.FailedGates, GatePromptLayout) {
		t.Fatalf("changed prompt layout should miss: result=%#v err=%v", result, err)
	}
	changedToken := request
	changedToken.TokenInputHash = "changed"
	if result, err := service.Lookup(changedToken, "validation-4", testKVTime()); err != nil || result.Hit || !contains(result.FailedGates, GateTokenIdentity) {
		t.Fatalf("changed token input should miss: result=%#v err=%v", result, err)
	}
	if got, ok := compiler.GetBundle(compiled.Bundle.BundleID); !ok || got.BundleHash != compiled.Bundle.BundleHash {
		t.Fatalf("kv lookup mutated context bundle: %#v ok=%v", got, ok)
	}
}
