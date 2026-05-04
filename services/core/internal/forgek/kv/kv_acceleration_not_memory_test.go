package kv

import "testing"

func TestKVManifestIsAccelerationMetadataNotMemoryOrTruth(t *testing.T) {
	manifest, err := NewManifest(validManifestInput())
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if manifest.IsCanonicalTruth() || manifest.IsSemanticEvidence() || manifest.IsMemory() {
		t.Fatalf("manifest claimed forbidden authority: %#v", manifest)
	}
	service := NewService()
	registered, err := service.RegisterManifest(validManifestInput())
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	evicted, err := service.Evict(registered.CacheID, "capacity", testKVTime(), "event-evict")
	if err != nil {
		t.Fatalf("evict: %v", err)
	}
	if evicted.Status != StatusEvicted || evicted.BundleID == "" || evicted.TokenInputHash == "" {
		t.Fatalf("eviction should not delete source identity metadata: %#v", evicted)
	}
	result, err := service.Lookup(validLookupRequest(), "validation-1", testKVTime())
	if err != nil {
		t.Fatalf("lookup after eviction: %v", err)
	}
	if result.Hit {
		t.Fatalf("evicted metadata must not produce a hit: %#v", result)
	}
}
