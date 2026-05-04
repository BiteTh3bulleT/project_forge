package kv

import "testing"

func TestKVServiceRegisterLookupHitMissAndList(t *testing.T) {
	service := NewService()
	manifest, err := service.RegisterManifest(validManifestInput())
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if got, ok := service.GetManifest(manifest.CacheID); !ok || got.CacheID != manifest.CacheID {
		t.Fatalf("get manifest failed: %#v ok=%v", got, ok)
	}
	if got := service.ListManifests(ManifestListFilter{WorkspaceID: "workspace-a"}); len(got) != 1 {
		t.Fatalf("workspace list failed: %#v", got)
	}
	if got := service.ListManifests(ManifestListFilter{BundleID: "bundle-a"}); len(got) != 1 {
		t.Fatalf("bundle list failed: %#v", got)
	}
	hit, err := service.Lookup(validLookupRequest(), "validation-1", testKVTime())
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !hit.Hit || hit.CacheID != manifest.CacheID || hit.Manifest == nil {
		t.Fatalf("expected lookup hit: %#v", hit)
	}
	missRequest := validLookupRequest()
	missRequest.BundleHash = "changed"
	miss, err := service.Lookup(missRequest, "validation-2", testKVTime())
	if err != nil {
		t.Fatalf("lookup miss: %v", err)
	}
	if miss.Hit || miss.MissReason != MissReasonIdentityGatesFailed || !contains(miss.FailedGates, GateContextBundle) {
		t.Fatalf("expected deterministic miss: %#v", miss)
	}
}

func TestKVServiceRecordHitAndLifecycle(t *testing.T) {
	service := NewService()
	manifest, err := service.RegisterManifest(validManifestInput())
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	hit, err := service.RecordHit(manifest.CacheID, testKVTime(), "event-hit")
	if err != nil {
		t.Fatalf("record hit: %v", err)
	}
	if hit.ReuseCount != 1 || hit.LastUsedAt == nil || hit.Status != StatusHitRecorded {
		t.Fatalf("hit metadata not updated: %#v", hit)
	}
	invalidated, err := service.Invalidate(manifest.CacheID, "policy changed", testKVTime(), "event-invalidate")
	if err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if invalidated.Status != StatusInvalidated || invalidated.InvalidatedAt == nil {
		t.Fatalf("invalidate did not preserve inspectable record: %#v", invalidated)
	}
	result, err := service.Lookup(validLookupRequest(), "validation-3", testKVTime())
	if err != nil {
		t.Fatalf("lookup invalidated: %v", err)
	}
	if result.Hit || !contains(result.FailedGates, GateManifestAvailable) {
		t.Fatalf("invalidated manifest should not hit: %#v", result)
	}
	evicted, err := service.Evict(manifest.CacheID, "capacity", testKVTime(), "event-evict")
	if err != nil {
		t.Fatalf("evict: %v", err)
	}
	if evicted.Status != StatusEvicted {
		t.Fatalf("evict did not mark record: %#v", evicted)
	}
}
