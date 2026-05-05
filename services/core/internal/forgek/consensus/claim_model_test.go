package consensus

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func testClaimInput() ClaimInput {
	return ClaimInput{
		ClaimID:      "claim-a",
		RequestID:    "request-a",
		ClaimType:    ClaimTypeFact,
		Subject:      "build",
		Predicate:    "status",
		ValueJSON:    map[string]any{"state": "green", "count": "3"},
		Scope:        "workspace-a",
		Temporal:     "2026-05-05",
		EvidenceRefs: []string{"evidence-a", "evidence-a"},
		Confidence:   0.9,
		AgentID:      "agent-a",
		CreatedAt:    time.Unix(100, 0).UTC(),
	}
}

func TestValidClaimCanBeCreatedAndSerializesDeterministically(t *testing.T) {
	claim, err := NewClaim(testClaimInput())
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}
	if claim.Status != StatusProposed || len(claim.EvidenceRefs) != 1 || claim.EvidenceRefs[0] != "evidence-a" {
		t.Fatalf("claim was not normalized: %#v", claim)
	}
	first, _ := json.Marshal(claim)
	second, _ := json.Marshal(claim)
	if string(first) != string(second) {
		t.Fatal("claim serialization drifted")
	}
	if claim.IsCanonicalTruth() || claim.IsMemoryTruth() || claim.IsActionExecution() {
		t.Fatalf("claim crossed authority boundary: %#v", claim)
	}
}

func TestInvalidClaimMissingRequiredFields(t *testing.T) {
	input := testClaimInput()
	input.Subject = ""
	if _, err := NewClaim(input); !errors.Is(err, ErrInvalidClaim) {
		t.Fatalf("expected invalid claim, got %v", err)
	}
}

func TestFactualClaimRequiresEvidenceButUncertaintyMayExistWithoutTier1(t *testing.T) {
	input := testClaimInput()
	input.EvidenceRefs = nil
	if _, err := NewClaim(input); !errors.Is(err, ErrInvalidClaim) {
		t.Fatalf("expected factual claim without evidence to be invalid, got %v", err)
	}
	input.ClaimType = ClaimTypeUncertainty
	claim, err := NewClaim(input)
	if err != nil {
		t.Fatalf("uncertainty claim should not require evidence: %v", err)
	}
	if claim.ClaimType != ClaimTypeUncertainty {
		t.Fatalf("unexpected uncertainty claim: %#v", claim)
	}
}
