package kvidentity

import (
	"testing"
	"time"
)

func TestValidateIdentityPassesForMatchingIdentity(t *testing.T) {
	manifest := testManifestIdentity()
	request := testRequestIdentity()
	result := ValidateIdentity("result-1", manifest, request, true, testKVTime())
	if !result.Passed {
		t.Fatalf("expected pass, failed gates=%v", result.FailedGates)
	}
	if result.CandidateCacheID != manifest.CacheID {
		t.Fatalf("candidate cache id = %q", result.CandidateCacheID)
	}
	if len(result.FailedGates) != 0 {
		t.Fatalf("unexpected failed gates: %v", result.FailedGates)
	}
}

func TestValidateIdentityFailsClosedForTokenMismatch(t *testing.T) {
	manifest := testManifestIdentity()
	request := testRequestIdentity()
	request.TokenInputHash = "other-token-input"
	result := ValidateIdentity("result-1", manifest, request, true, testKVTime())
	if result.Passed {
		t.Fatal("expected token identity mismatch to fail")
	}
	if !hasString(result.FailedGates, GateTokenIdentity) {
		t.Fatalf("missing token gate failure: %v", result.FailedGates)
	}
}

func TestValidateIdentityFailsClosedForUnavailableManifest(t *testing.T) {
	result := ValidateIdentity("result-1", testManifestIdentity(), testRequestIdentity(), false, testKVTime())
	if result.Passed {
		t.Fatal("expected unavailable manifest to fail")
	}
	if !hasString(result.FailedGates, GateManifestAvailable) {
		t.Fatalf("missing manifest gate failure: %v", result.FailedGates)
	}
}

func testManifestIdentity() ManifestIdentity {
	return ManifestIdentity{
		CacheID:            "cache-a",
		CacheMode:          "STRICT_PREFIX",
		WorkspaceID:        "ws-main",
		BundleID:           "bundle-a",
		BlockID:            "block-a",
		BundleHash:         "bundle-hash",
		StablePrefixHash:   "stable-hash",
		VolatileSuffixHash: "volatile-hash",
		ModelID:            "model-a",
		ModelRevision:      "rev-a",
		TokenizerID:        "tok-a",
		TokenizerRevision:  "tok-rev-a",
		ChatTemplateHash:   "chat-template",
		PromptLayoutHash:   "layout",
		PolicySchemaHash:   "policy",
		SyscallSchemaHash:  "syscall",
		TokenInputHash:     "token-input",
		RuntimeBackend:     "mock",
		RuntimeVersion:     "1",
		AttentionBackend:   "attn",
		RopeConfigHash:     "rope",
		KVPrecision:        "fp16",
		CacheSalt:          "salt",
		Status:             "AVAILABLE",
	}
}

func testRequestIdentity() RequestIdentity {
	m := testManifestIdentity()
	return RequestIdentity{
		RequestID:          "request-a",
		CacheMode:          m.CacheMode,
		WorkspaceID:        m.WorkspaceID,
		BundleID:           m.BundleID,
		BlockID:            m.BlockID,
		BundleHash:         m.BundleHash,
		StablePrefixHash:   m.StablePrefixHash,
		VolatileSuffixHash: m.VolatileSuffixHash,
		ModelID:            m.ModelID,
		ModelRevision:      m.ModelRevision,
		TokenizerID:        m.TokenizerID,
		TokenizerRevision:  m.TokenizerRevision,
		ChatTemplateHash:   m.ChatTemplateHash,
		PromptLayoutHash:   m.PromptLayoutHash,
		PolicySchemaHash:   m.PolicySchemaHash,
		SyscallSchemaHash:  m.SyscallSchemaHash,
		TokenInputHash:     m.TokenInputHash,
		RuntimeBackend:     m.RuntimeBackend,
		RuntimeVersion:     m.RuntimeVersion,
		AttentionBackend:   m.AttentionBackend,
		RopeConfigHash:     m.RopeConfigHash,
		KVPrecision:        m.KVPrecision,
		CacheSalt:          m.CacheSalt,
	}
}

func testKVTime() time.Time {
	return time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
