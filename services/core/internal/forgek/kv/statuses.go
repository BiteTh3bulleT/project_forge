package kv

type CacheMode string
type MemoryTier string
type ManifestStatus string

const (
	ModeStrictPrefix         CacheMode = "STRICT_PREFIX"
	ModeSnapshotPrefix       CacheMode = "SNAPSHOT_PREFIX"
	ModeBackendCompositional CacheMode = "BACKEND_COMPOSITIONAL"
)

const (
	TierGPUHot     MemoryTier = "GPU_HOT"
	TierCPUWarm    MemoryTier = "CPU_WARM"
	TierDiskCold   MemoryTier = "DISK_COLD"
	TierRemoteCold MemoryTier = "REMOTE_COLD"
	TierNone       MemoryTier = "NONE"
)

const (
	StatusAvailable   ManifestStatus = "AVAILABLE"
	StatusHitRecorded ManifestStatus = "HIT_RECORDED"
	StatusInvalidated ManifestStatus = "INVALIDATED"
	StatusEvicted     ManifestStatus = "EVICTED"
	StatusExpired     ManifestStatus = "EXPIRED"
)

const (
	DefaultCacheMode CacheMode  = ModeStrictPrefix
	DefaultTier      MemoryTier = TierNone
)

func ValidCacheMode(mode CacheMode) bool {
	switch mode {
	case ModeStrictPrefix, ModeSnapshotPrefix, ModeBackendCompositional:
		return true
	default:
		return false
	}
}

func ValidMemoryTier(tier MemoryTier) bool {
	switch tier {
	case TierGPUHot, TierCPUWarm, TierDiskCold, TierRemoteCold, TierNone:
		return true
	default:
		return false
	}
}

func ValidStatus(status ManifestStatus) bool {
	switch status {
	case StatusAvailable, StatusHitRecorded, StatusInvalidated, StatusEvicted, StatusExpired:
		return true
	default:
		return false
	}
}

func HitEligibleStatus(status ManifestStatus) bool {
	return status == StatusAvailable || status == StatusHitRecorded
}
