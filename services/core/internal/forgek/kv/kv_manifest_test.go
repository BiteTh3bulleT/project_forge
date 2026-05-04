package kv

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func testKVTime() time.Time {
	return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
}

func validManifestInput() ManifestInput {
	return ManifestInput{
		CacheID:            "cache-1",
		CacheMode:          ModeStrictPrefix,
		WorkspaceID:        "workspace-a",
		CaseID:             "case-a",
		BundleID:           "bundle-a",
		BundleHash:         "bundle-hash-a",
		StablePrefixHash:   "stable-hash-a",
		VolatileSuffixHash: "volatile-hash-a",
		ModelID:            "model-a",
		ModelRevision:      "rev-a",
		TokenizerID:        "tokenizer-a",
		TokenizerRevision:  "tok-rev-a",
		ChatTemplateHash:   "template-hash-a",
		PromptLayoutHash:   "layout-hash-a",
		PolicySchemaHash:   "policy-hash-a",
		SyscallSchemaHash:  "syscall-hash-a",
		TokenInputHash:     "token-input-hash-a",
		RuntimeBackend:     "simulator",
		RuntimeVersion:     "v1",
		AttentionBackend:   "attention-a",
		RopeConfigHash:     "rope-hash-a",
		KVPrecision:        "fp16",
		MemoryTier:         TierNone,
		CacheSalt:          "salt-a",
		CreatedAt:          testKVTime(),
		Metadata:           map[string]any{"purpose": "test"},
	}
}

func validLookupRequest() KVLookupRequest {
	input := validManifestInput()
	return KVLookupRequest{
		RequestID:          "lookup-1",
		WorkspaceID:        input.WorkspaceID,
		CaseID:             input.CaseID,
		BundleID:           input.BundleID,
		BlockID:            input.BlockID,
		SnapshotID:         input.SnapshotID,
		RestoreSeedID:      input.RestoreSeedID,
		BundleHash:         input.BundleHash,
		StablePrefixHash:   input.StablePrefixHash,
		VolatileSuffixHash: input.VolatileSuffixHash,
		ModelID:            input.ModelID,
		ModelRevision:      input.ModelRevision,
		TokenizerID:        input.TokenizerID,
		TokenizerRevision:  input.TokenizerRevision,
		ChatTemplateHash:   input.ChatTemplateHash,
		PromptLayoutHash:   input.PromptLayoutHash,
		PolicySchemaHash:   input.PolicySchemaHash,
		SyscallSchemaHash:  input.SyscallSchemaHash,
		TokenInputHash:     input.TokenInputHash,
		RuntimeBackend:     input.RuntimeBackend,
		RuntimeVersion:     input.RuntimeVersion,
		AttentionBackend:   input.AttentionBackend,
		RopeConfigHash:     input.RopeConfigHash,
		KVPrecision:        input.KVPrecision,
		CacheSalt:          input.CacheSalt,
		CacheMode:          input.CacheMode,
		CreatedAt:          testKVTime(),
	}
}

func TestKVCacheManifestValidatesAndSerializes(t *testing.T) {
	manifest, err := NewManifest(validManifestInput())
	if err != nil {
		t.Fatalf("NewManifest failed: %v", err)
	}
	if manifest.Status != StatusAvailable || manifest.CacheMode != ModeStrictPrefix {
		t.Fatalf("unexpected defaults: %#v", manifest)
	}
	if manifest.BundleID != "bundle-a" || manifest.TokenInputHash == "" {
		t.Fatalf("manifest did not preserve context identity: %#v", manifest)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil || !json.Valid(encoded) {
		t.Fatalf("manifest must serialize as JSON: %v %s", err, encoded)
	}
	if IdentityHash(manifest) != IdentityHash(manifest.Clone()) {
		t.Fatal("manifest identity hash should be deterministic for clone")
	}
}

func TestKVCacheManifestRejectsInvalidInputs(t *testing.T) {
	input := validManifestInput()
	input.WorkspaceID = ""
	if _, err := NewManifest(input); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected missing workspace rejection, got %v", err)
	}
	input = validManifestInput()
	input.CacheMode = "UNKNOWN"
	if _, err := NewManifest(input); !errors.Is(err, ErrInvalidCacheMode) {
		t.Fatalf("expected invalid mode rejection, got %v", err)
	}
	input = validManifestInput()
	input.MemoryTier = "UNKNOWN"
	if _, err := NewManifest(input); !errors.Is(err, ErrInvalidMemoryTier) {
		t.Fatalf("expected invalid tier rejection, got %v", err)
	}
	input = validManifestInput()
	input.ModelID = ""
	if _, err := NewManifest(input); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected missing model rejection, got %v", err)
	}
	input = validManifestInput()
	input.TokenizerID = ""
	if _, err := NewManifest(input); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected missing tokenizer rejection, got %v", err)
	}
	input = validManifestInput()
	input.RuntimeBackend = ""
	if _, err := NewManifest(input); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected missing runtime rejection, got %v", err)
	}
}
