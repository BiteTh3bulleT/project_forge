package kv

import "testing"

func TestKVInvalidationAndEvictionPreserveManifestRecords(t *testing.T) {
	manifest, err := NewManifest(validManifestInput())
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	invalidated, err := InvalidateManifest(manifest, "bundle changed", testKVTime(), "event-invalidate")
	if err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if invalidated.Status != StatusInvalidated || invalidated.InvalidationReason != "bundle changed" {
		t.Fatalf("invalidated manifest lost metadata: %#v", invalidated)
	}
	if invalidated.BundleID != manifest.BundleID || invalidated.TokenInputHash != manifest.TokenInputHash {
		t.Fatalf("invalidated manifest should preserve identity refs: %#v", invalidated)
	}
	evicted := EvictManifest(invalidated, "capacity", testKVTime(), "event-evict")
	if evicted.Status != StatusEvicted || evicted.BundleID != manifest.BundleID {
		t.Fatalf("evicted manifest should remain inspectable: %#v", evicted)
	}
	if _, err := InvalidateManifest(evicted, "late", testKVTime(), "event-late"); err == nil {
		t.Fatal("evicted manifest should reject later invalidation")
	}
}
