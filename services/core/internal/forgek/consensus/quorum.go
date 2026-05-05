package consensus

func QuorumMet(policy ConsensusPolicy, stats EvaluationStats) bool {
	policy = NormalizePolicy(policy)
	return stats.AgentCount >= policy.RequiredAgents
}

func EvidencePolicyPassed(policy ConsensusPolicy, claimType ClaimType, stats EvaluationStats, supporting []Claim, evidence map[string]EvidenceRef) bool {
	policy = NormalizePolicy(policy)
	if claimType == ClaimTypeUncertainty || claimType == ClaimTypeInference {
		return true
	}
	if requiresEvidenceByType(claimType) && stats.Tier1Count+stats.Tier2Count == 0 {
		if !policy.AllowTier3ForFacts {
			return false
		}
		return !tier3SoleSupport(supporting, evidence)
	}
	if policy.RequiredTier1Count > 0 && stats.Tier1Count < policy.RequiredTier1Count {
		return false
	}
	if policy.RequiredTier2Count > 0 && stats.Tier1Count == 0 && stats.Tier2Count < policy.RequiredTier2Count {
		return false
	}
	if claimType == ClaimTypeFact && tier3SoleSupport(supporting, evidence) {
		return false
	}
	return true
}

func RiskPolicyPassed(policy ConsensusPolicy, supporting []Claim) bool {
	if policy.RequireHumanConfirmation {
		for _, claim := range supporting {
			for _, flag := range claim.RiskFlags {
				if flag == "human_confirmed" {
					return true
				}
			}
		}
		return false
	}
	return true
}

func tier3SoleSupport(claims []Claim, evidence map[string]EvidenceRef) bool {
	hasEvidence := false
	for _, claim := range claims {
		for _, refID := range claim.EvidenceRefs {
			ref, ok := evidence[refID]
			if !ok {
				continue
			}
			hasEvidence = true
			if ref.Tier != EvidenceTier3 {
				return false
			}
		}
	}
	return hasEvidence
}
