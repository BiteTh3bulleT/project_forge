package consensus

import "time"

func DefaultPolicy(workspaceID string, criticality Criticality, createdAt time.Time) ConsensusPolicy {
	if criticality == "" {
		criticality = CriticalityLow
	}
	policy := ConsensusPolicy{
		PolicyID:                 "consensus-policy-" + string(criticality),
		WorkspaceID:              trim(workspaceID),
		Criticality:              criticality,
		MaxAgentCount:            8,
		MaxToolCalls:             0,
		MaxModelCalls:            0,
		MaxWallTimeMS:            1000,
		AllowTier3ForFacts:       false,
		RequireHumanConfirmation: false,
		CreatedAt:                createdAt,
	}
	switch criticality {
	case CriticalityLow:
		policy.RequiredAgents = 1
		policy.MinSupportRatio = 0.60
		policy.MaxConflictRatio = 0.40
	case CriticalityMedium:
		policy.RequiredAgents = 2
		policy.RequiredTier2Count = 2
		policy.MinSupportRatio = 0.67
		policy.MaxConflictRatio = 0.33
	case CriticalityHigh:
		policy.RequiredAgents = 3
		policy.RequiredTier1Count = 1
		policy.MinSupportRatio = 0.80
		policy.MaxConflictRatio = 0.20
	case CriticalityCritical:
		policy.RequiredAgents = 3
		policy.RequiredTier1Count = 1
		policy.MinSupportRatio = 0.80
		policy.MaxConflictRatio = 0
		policy.RequireHumanConfirmation = true
	default:
		policy.Criticality = CriticalityLow
		policy.RequiredAgents = 1
		policy.MinSupportRatio = 0.60
		policy.MaxConflictRatio = 0.40
	}
	return policy
}

func NormalizePolicy(policy ConsensusPolicy) ConsensusPolicy {
	policy.PolicyID = trim(policy.PolicyID)
	policy.WorkspaceID = trim(policy.WorkspaceID)
	if policy.Criticality == "" {
		policy.Criticality = CriticalityLow
	}
	if policy.RequiredAgents == 0 {
		defaulted := DefaultPolicy(policy.WorkspaceID, policy.Criticality, policy.CreatedAt)
		if policy.PolicyID != "" {
			defaulted.PolicyID = policy.PolicyID
		}
		if !policy.CreatedAt.IsZero() {
			defaulted.CreatedAt = policy.CreatedAt
		}
		defaulted.Metadata = CloneMap(policy.Metadata)
		return defaulted
	}
	policy.Metadata = CloneMap(policy.Metadata)
	return policy
}

func ValidatePolicy(policy ConsensusPolicy) error {
	policy = NormalizePolicy(policy)
	if policy.PolicyID == "" || policy.WorkspaceID == "" || !ValidCriticality(policy.Criticality) ||
		policy.RequiredAgents < 1 || policy.MinSupportRatio < 0 || policy.MinSupportRatio > 1 ||
		policy.MaxConflictRatio < 0 || policy.MaxConflictRatio > 1 || policy.MaxAgentCount < 0 ||
		policy.MaxToolCalls != 0 || policy.MaxModelCalls != 0 || containsSecretMetadata(policy.Metadata) {
		return ErrInvalidPolicy
	}
	return nil
}
