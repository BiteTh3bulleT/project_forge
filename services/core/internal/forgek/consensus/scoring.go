package consensus

import "sort"

type ScoreBreakdown struct {
	ClaimID            string  `json:"claim_id"`
	Weight             float64 `json:"weight"`
	EvidenceTier       string  `json:"evidence_tier"`
	IndependenceFactor float64 `json:"independence_factor"`
}

type EvaluationStats struct {
	SupportWeight       float64
	OpposingWeight      float64
	TotalEligibleWeight float64
	SupportRatio        float64
	ConflictRatio       float64
	Tier1Count          int
	Tier2Count          int
	AgentCount          int
	Breakdowns          []ScoreBreakdown
}

func ScoreClaims(supporting []Claim, opposing []Claim, evidence map[string]EvidenceRef, agentReliability map[string]float64) EvaluationStats {
	sourceUse := evidenceSourceUse(append(append([]Claim{}, supporting...), opposing...), evidence)
	stats := EvaluationStats{}
	agents := map[string]struct{}{}
	for _, claim := range supporting {
		weight, tier, factor := ScoreClaim(claim, evidence, agentReliability, sourceUse)
		stats.SupportWeight += weight
		if weight > 0 {
			agents[claim.AgentID] = struct{}{}
		}
		stats.Tier1Count += countTier(claim, evidence, EvidenceTier1)
		stats.Tier2Count += countTier(claim, evidence, EvidenceTier2)
		stats.Breakdowns = append(stats.Breakdowns, ScoreBreakdown{ClaimID: claim.ClaimID, Weight: weight, EvidenceTier: string(tier), IndependenceFactor: factor})
	}
	for _, claim := range opposing {
		weight, tier, factor := ScoreClaim(claim, evidence, agentReliability, sourceUse)
		stats.OpposingWeight += weight
		if weight > 0 {
			agents[claim.AgentID] = struct{}{}
		}
		stats.Breakdowns = append(stats.Breakdowns, ScoreBreakdown{ClaimID: claim.ClaimID, Weight: weight, EvidenceTier: string(tier), IndependenceFactor: factor})
	}
	stats.AgentCount = len(agents)
	stats.TotalEligibleWeight = stats.SupportWeight + stats.OpposingWeight
	if stats.TotalEligibleWeight > 0 {
		stats.SupportRatio = stats.SupportWeight / stats.TotalEligibleWeight
		stats.ConflictRatio = stats.OpposingWeight / stats.TotalEligibleWeight
	}
	sort.Slice(stats.Breakdowns, func(i, j int) bool { return stats.Breakdowns[i].ClaimID < stats.Breakdowns[j].ClaimID })
	return stats
}

func ScoreClaim(claim Claim, evidence map[string]EvidenceRef, agentReliability map[string]float64, sourceUse map[string]int) (float64, EvidenceTier, float64) {
	if requiresEvidenceByType(claim.ClaimType) && len(claim.EvidenceRefs) == 0 {
		return 0, "", 0
	}
	bestQuality := 0.0
	bestTier := EvidenceTier("")
	bestFactor := 1.0
	for _, refID := range claim.EvidenceRefs {
		ref, ok := evidence[refID]
		if !ok {
			continue
		}
		factor := independenceFactor(ref, sourceUse)
		quality := EvidenceQuality(ref) * ref.FreshnessScore * factor
		if quality > bestQuality {
			bestQuality = quality
			bestTier = ref.Tier
			bestFactor = factor
		}
	}
	if len(claim.EvidenceRefs) > 0 && bestQuality == 0 {
		return 0, bestTier, bestFactor
	}
	if !requiresEvidenceByType(claim.ClaimType) && bestQuality == 0 {
		bestQuality = 0.5
	}
	reliability := agentReliability[claim.AgentID]
	if reliability == 0 {
		reliability = 1
	}
	return reliability * bestQuality * claim.Confidence, bestTier, bestFactor
}

func evidenceSourceUse(claims []Claim, evidence map[string]EvidenceRef) map[string]int {
	uses := make(map[string]int)
	for _, claim := range claims {
		for _, refID := range claim.EvidenceRefs {
			ref, ok := evidence[refID]
			if ok {
				uses[ref.Source]++
			}
		}
	}
	return uses
}

func independenceFactor(ref EvidenceRef, sourceUse map[string]int) float64 {
	uses := sourceUse[ref.Source]
	if uses <= 1 {
		return 1
	}
	if ref.Tier == EvidenceTier1 {
		return 0.6
	}
	return 0.3
}

func countTier(claim Claim, evidence map[string]EvidenceRef, tier EvidenceTier) int {
	count := 0
	for _, refID := range claim.EvidenceRefs {
		if ref, ok := evidence[refID]; ok && ref.Tier == tier {
			count++
		}
	}
	return count
}
