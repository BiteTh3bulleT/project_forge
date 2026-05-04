package runtime

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func testManifest() RuntimeDriverManifest {
	return RuntimeDriverManifest{
		DriverID:       "driver-a",
		DriverName:     "Mock Driver",
		DriverKind:     DriverKindMock,
		Version:        "v1",
		RuntimeBackend: "mock",
		RuntimeVersion: "v1",
		SupportedModels: []string{
			"model-b",
			"model-a",
			"model-a",
		},
		AuthorityLevel: RuntimeAuthorityProposalOnly,
		CreatedAt:      time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
	}
}

func testGenerateRequest() RuntimeGenerateRequest {
	return RuntimeGenerateRequest{
		RequestID:           "request-a",
		DriverID:            "driver-a",
		WorkspaceID:         "workspace-a",
		CaseID:              "case-a",
		BundleID:            "bundle-a",
		CanonicalPromptText: "compiled context",
		ModelID:             "model-a",
		ModelRevision:       "rev-a",
		TokenizerID:         "tokenizer-a",
		TokenizerRevision:   "tok-rev-a",
		ChatTemplateHash:    "template-hash-a",
		PromptLayoutHash:    "layout-hash-a",
		PolicySchemaHash:    "policy-hash-a",
		SyscallSchemaHash:   "syscall-hash-a",
		TokenInputHash:      "token-input-hash-a",
		RequestedBy:         "operator",
		CreatedAt:           time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
	}
}

func TestRuntimeDriverManifestValidationAndNormalization(t *testing.T) {
	manifest, err := NewRuntimeDriverManifest(testManifest())
	if err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	if got, want := manifest.SupportedModels, []string{"model-a", "model-b"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("supported models not normalized: %#v", got)
	}
	if manifest.AuthorityLevel != RuntimeAuthorityProposalOnly {
		t.Fatalf("manifest claimed authority: %#v", manifest)
	}

	invalid := testManifest()
	invalid.DriverKind = DriverKindRemoteAPI
	if _, err := NewRuntimeDriverManifest(invalid); err != nil {
		t.Fatalf("non-mock driver kinds are valid model data before syscall gating: %v", err)
	}

	invalid.AuthorityLevel = "AUTHORITY"
	if _, err := NewRuntimeDriverManifest(invalid); !errors.Is(err, ErrInvalidDriverManifest) {
		t.Fatalf("expected invalid authority rejection, got %v", err)
	}

	secret := testManifest()
	secret.Metadata = map[string]any{"api_key": "redacted"}
	if _, err := NewRuntimeDriverManifest(secret); !errors.Is(err, ErrSecretInManifest) {
		t.Fatalf("expected secret metadata rejection, got %v", err)
	}
}

func TestRuntimeModelsSerializeAndValidateGenerateRequests(t *testing.T) {
	if err := ValidateGenerateRequest(testGenerateRequest()); err != nil {
		t.Fatalf("valid generate request rejected: %v", err)
	}
	missing := testGenerateRequest()
	missing.TokenInputHash = ""
	if err := ValidateGenerateRequest(missing); !errors.Is(err, ErrInvalidGenerateRequest) {
		t.Fatalf("expected missing identity rejection, got %v", err)
	}
	encoded, err := json.Marshal(testGenerateRequest())
	if err != nil {
		t.Fatalf("generate request did not serialize: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("empty serialized request")
	}
}
