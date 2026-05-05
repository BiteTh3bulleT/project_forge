package consensus

import (
	"errors"
	"testing"
	"time"
)

func testEvidence(id string, tier EvidenceTier) EvidenceRef {
	evidenceType := EvidenceSourceCode
	if tier == EvidenceTier3 {
		evidenceType = EvidenceModelInference
	}
	return EvidenceRef{
		EvidenceID:       id,
		EvidenceType:     evidenceType,
		Tier:             tier,
		Source:           "repo",
		Locator:          "file.go:12",
		RetrievedAt:      time.Unix(100, 0).UTC(),
		FreshnessScore:   1,
		ReliabilityScore: 1,
	}
}

func TestEvidenceTierValidation(t *testing.T) {
	for _, tier := range []EvidenceTier{EvidenceTier1, EvidenceTier2, EvidenceTier3} {
		if _, err := NewEvidenceRef(testEvidence(string(tier), tier)); err != nil {
			t.Fatalf("expected tier %s to validate: %v", tier, err)
		}
	}
	invalid := testEvidence("bad", EvidenceTier("bad"))
	if _, err := NewEvidenceRef(invalid); !errors.Is(err, ErrInvalidEvidenceRef) {
		t.Fatalf("expected invalid tier rejection, got %v", err)
	}
}

func TestTier3CannotSoleSupportFactualAcceptance(t *testing.T) {
	claim, err := NewClaim(testClaimInput())
	if err != nil {
		t.Fatal(err)
	}
	claim.EvidenceRefs = []string{"tier3"}
	evidence := map[string]EvidenceRef{"tier3": testEvidence("tier3", EvidenceTier3)}
	policy := DefaultPolicy("workspace-a", CriticalityLow, time.Unix(100, 0).UTC())
	stats := ScoreClaims([]Claim{claim}, nil, evidence, nil)
	if EvidencePolicyPassed(policy, ClaimTypeFact, stats, []Claim{claim}, evidence) {
		t.Fatal("tier 3 model inference should not sole-support factual acceptance")
	}
}
