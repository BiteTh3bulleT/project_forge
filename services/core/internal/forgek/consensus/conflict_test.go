package consensus

import "testing"

func TestConflictDetection(t *testing.T) {
	left, _ := NewClaim(testClaimInput())
	rightInput := testClaimInput()
	rightInput.ClaimID = "claim-b"
	rightInput.ValueJSON = "red"
	right, _ := NewClaim(rightInput)
	conflicts := DetectConflicts([]Claim{left, right})
	if len(conflicts) != 1 || len(conflicts[0].ClaimIDs) != 2 {
		t.Fatalf("expected hard conflict, got %#v", conflicts)
	}
	differentScopeInput := testClaimInput()
	differentScopeInput.ClaimID = "claim-c"
	differentScopeInput.Scope = "workspace-b"
	differentScopeInput.ValueJSON = "red"
	differentScope, _ := NewClaim(differentScopeInput)
	if ClaimsConflict(left, differentScope) {
		t.Fatal("different scope should not conflict")
	}
	uncertainInput := testClaimInput()
	uncertainInput.ClaimID = "claim-u"
	uncertainInput.ClaimType = ClaimTypeUncertainty
	uncertainInput.EvidenceRefs = nil
	uncertainInput.ValueJSON = "red"
	uncertain, _ := NewClaim(uncertainInput)
	if ClaimsConflict(left, uncertain) {
		t.Fatal("uncertainty should not hard-conflict by default")
	}
}
