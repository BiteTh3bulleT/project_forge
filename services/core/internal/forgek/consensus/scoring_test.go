package consensus

import (
	"testing"
	"time"
)

func claimWithEvidence(id string, evidenceID string, agentID string, confidence float64) Claim {
	input := testClaimInput()
	input.ClaimID = id
	input.EvidenceRefs = []string{evidenceID}
	input.AgentID = agentID
	input.Confidence = confidence
	claim, err := NewClaim(input)
	if err != nil {
		panic(err)
	}
	return claim
}

func TestWeightedScoringAndQuorum(t *testing.T) {
	tier1 := testEvidence("tier1", EvidenceTier1)
	staleTier2 := testEvidence("tier2", EvidenceTier2)
	staleTier2.FreshnessScore = 0.2
	evidence := map[string]EvidenceRef{"tier1": tier1, "tier2": staleTier2}
	one := claimWithEvidence("claim-1", "tier1", "agent-a", 0.8)
	two := claimWithEvidence("claim-2", "tier2", "agent-b", 0.9)
	oneScore, _, _ := ScoreClaim(one, evidence, nil, evidenceSourceUse([]Claim{one}, evidence))
	twoScore, _, _ := ScoreClaim(two, evidence, nil, evidenceSourceUse([]Claim{two}, evidence))
	if oneScore <= twoScore {
		t.Fatalf("tier 1 should beat stale tier 2: tier1=%f tier2=%f", oneScore, twoScore)
	}
	medium := DefaultPolicy("workspace-a", CriticalityMedium, time.Unix(100, 0).UTC())
	stats := ScoreClaims([]Claim{one, two}, nil, evidence, nil)
	if !QuorumMet(medium, stats) {
		t.Fatalf("expected two agents to meet medium quorum: %#v", stats)
	}
}

func TestSameDerivedSourceDiscountedAndUnsupportedFactRejected(t *testing.T) {
	leftEvidence := testEvidence("evidence-a", EvidenceTier2)
	leftEvidence.Source = "same-derived"
	rightEvidence := testEvidence("evidence-b", EvidenceTier2)
	rightEvidence.Source = "same-derived"
	evidence := map[string]EvidenceRef{"evidence-a": leftEvidence, "evidence-b": rightEvidence}
	left := claimWithEvidence("claim-a", "evidence-a", "agent-a", 1)
	right := claimWithEvidence("claim-b", "evidence-b", "agent-b", 1)
	_, _, factor := ScoreClaim(left, evidence, nil, evidenceSourceUse([]Claim{left, right}, evidence))
	if factor != 0.3 {
		t.Fatalf("same derived source should be discounted, got %f", factor)
	}
	missing := claimWithEvidence("claim-missing", "missing", "agent-a", 1)
	stats := ScoreClaims([]Claim{missing}, nil, evidence, nil)
	if stats.SupportWeight != 0 {
		t.Fatalf("unsupported factual claim should have zero support: %#v", stats)
	}
}

func TestConflictRatioBlocksAcceptanceAndCriticalRequiresConfirmation(t *testing.T) {
	left := claimWithEvidence("claim-a", "evidence-a", "agent-a", 1)
	rightInput := testClaimInput()
	rightInput.ClaimID = "claim-b"
	rightInput.AgentID = "agent-b"
	rightInput.ValueJSON = "opposed"
	rightInput.EvidenceRefs = []string{"evidence-b"}
	right, _ := NewClaim(rightInput)
	evidence := map[string]EvidenceRef{
		"evidence-a": testEvidence("evidence-a", EvidenceTier1),
		"evidence-b": testEvidence("evidence-b", EvidenceTier1),
	}
	policy := DefaultPolicy("workspace-a", CriticalityLow, time.Unix(100, 0).UTC())
	decision := EvaluateClaims("request-a", []Claim{left, right}, evidence, policy, time.Unix(100, 0).UTC())[0]
	if decision.Status != StatusConflicted {
		t.Fatalf("expected conflict to block acceptance, got %#v", decision)
	}
	critical := DefaultPolicy("workspace-a", CriticalityCritical, time.Unix(100, 0).UTC())
	stats := ScoreClaims([]Claim{left}, nil, evidence, nil)
	if RiskPolicyPassed(critical, []Claim{left}) || !EvidencePolicyPassed(critical, left.ClaimType, stats, []Claim{left}, evidence) {
		t.Fatal("critical policy should require human confirmation while evidence passes")
	}
}
