package ephemeral

import (
	"errors"
	"testing"
	"time"
)

func TestKeyPolicyAcceptsSafeKey(t *testing.T) {
	policy := DefaultKeyPolicy("forge")
	key, err := policy.Key(KeyKindCache, "workspace-1", "job.123", "status_v1")
	if err != nil {
		t.Fatalf("expected safe key: %v", err)
	}
	if key != "forge:cache:workspace-1:job.123:status_v1" {
		t.Fatalf("unexpected key %q", key)
	}
}

func TestKeyPolicyRejectsUnsafeKeys(t *testing.T) {
	policy := DefaultKeyPolicy("forge")
	unsafe := []string{
		"raw-user-prompt",
		"document-content",
		"secret-value",
		"api_key",
		"token",
		"contains spaces",
		"../../escape",
	}
	for _, segment := range unsafe {
		if _, err := policy.Key(KeyKindCache, "workspace-1", segment); !errors.Is(err, ErrUnsafeKey) {
			t.Fatalf("expected unsafe segment %q to fail, got %v", segment, err)
		}
	}
}

func TestTTLRequiredForCacheProgressAndLocks(t *testing.T) {
	policy := DefaultKeyPolicy("forge")
	for _, kind := range []KeyKind{KeyKindCache, KeyKindProgress, KeyKindLock} {
		if err := policy.RequireTTL(kind, 0); !errors.Is(err, ErrTTLRequired) {
			t.Fatalf("expected ttl required for %q, got %v", kind, err)
		}
		if err := policy.RequireTTL(kind, time.Second); err != nil {
			t.Fatalf("expected ttl accepted for %q: %v", kind, err)
		}
	}
	if err := policy.RequireTTL(KeyKindQueue, 0); err != nil {
		t.Fatalf("queue keys do not require ttl in phase 13H: %v", err)
	}
}

func TestStableOpaqueSegmentIsDeterministic(t *testing.T) {
	left := StableOpaqueSegment("workspace", "source", "operation")
	right := StableOpaqueSegment("workspace", "source", "operation")
	changed := StableOpaqueSegment("workspace", "source", "other")
	if left != right {
		t.Fatalf("expected stable opaque segment")
	}
	if left == changed {
		t.Fatalf("expected changed input to alter segment")
	}
	if _, err := DefaultKeyPolicy("forge").Key(KeyKindCache, left); err != nil {
		t.Fatalf("opaque segment should be key-safe: %v", err)
	}
}
