package kv

func PromoteTier(tier MemoryTier) MemoryTier {
	switch tier {
	case TierNone:
		return TierDiskCold
	case TierRemoteCold:
		return TierDiskCold
	case TierDiskCold:
		return TierCPUWarm
	case TierCPUWarm:
		return TierGPUHot
	default:
		return TierGPUHot
	}
}

func DemoteTier(tier MemoryTier) MemoryTier {
	switch tier {
	case TierGPUHot:
		return TierCPUWarm
	case TierCPUWarm:
		return TierDiskCold
	case TierDiskCold:
		return TierRemoteCold
	default:
		return TierNone
	}
}
