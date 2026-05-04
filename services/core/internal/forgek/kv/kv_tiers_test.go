package kv

import "testing"

func TestKVTierPromotionAndDemotionAreMetadataOnly(t *testing.T) {
	if PromoteTier(TierNone) != TierDiskCold {
		t.Fatal("NONE should promote to DISK_COLD")
	}
	if PromoteTier(TierDiskCold) != TierCPUWarm {
		t.Fatal("DISK_COLD should promote to CPU_WARM")
	}
	if PromoteTier(TierCPUWarm) != TierGPUHot {
		t.Fatal("CPU_WARM should promote to GPU_HOT")
	}
	if DemoteTier(TierGPUHot) != TierCPUWarm {
		t.Fatal("GPU_HOT should demote to CPU_WARM")
	}
	if DemoteTier(TierCPUWarm) != TierDiskCold {
		t.Fatal("CPU_WARM should demote to DISK_COLD")
	}
	if DemoteTier(TierRemoteCold) != TierNone {
		t.Fatal("REMOTE_COLD should demote to NONE")
	}
	service := NewService()
	manifest, err := service.RegisterManifest(validManifestInput())
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	promoted, err := service.Promote(manifest.CacheID, "event-promote")
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if promoted.MemoryTier != TierDiskCold || promoted.BundleID != manifest.BundleID {
		t.Fatalf("promotion should only alter tier metadata: %#v", promoted)
	}
	demoted, err := service.Demote(manifest.CacheID, "event-demote")
	if err != nil {
		t.Fatalf("demote: %v", err)
	}
	if demoted.MemoryTier != TierRemoteCold || demoted.BundleID != manifest.BundleID {
		t.Fatalf("demotion should only alter tier metadata: %#v", demoted)
	}
}
