package consensus

import "testing"

func TestClaimKeyStableForMapOrderAndNumericStrings(t *testing.T) {
	left := testClaimInput()
	left.ValueJSON = map[string]any{"b": "3", "a": true}
	right := testClaimInput()
	right.ClaimID = "claim-b"
	right.ValueJSON = map[string]any{"a": "true", "b": 3}
	leftClaim, err := NewClaim(left)
	if err != nil {
		t.Fatal(err)
	}
	rightClaim, err := NewClaim(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftClaim.ClaimKey != rightClaim.ClaimKey {
		t.Fatalf("equivalent normalized values produced different keys: %s %s", leftClaim.ClaimKey, rightClaim.ClaimKey)
	}
}

func TestScopeTemporalAndValueAffectClaimKey(t *testing.T) {
	base, _ := NewClaim(testClaimInput())
	changedScope := testClaimInput()
	changedScope.Scope = "workspace-b"
	scopeClaim, _ := NewClaim(changedScope)
	changedTemporal := testClaimInput()
	changedTemporal.Temporal = "2026-05-06"
	temporalClaim, _ := NewClaim(changedTemporal)
	changedValue := testClaimInput()
	changedValue.ValueJSON = "red"
	valueClaim, _ := NewClaim(changedValue)
	if base.ClaimKey == scopeClaim.ClaimKey || base.ClaimKey == temporalClaim.ClaimKey || base.ClaimKey == valueClaim.ClaimKey {
		t.Fatal("scope, temporal bucket, or value did not affect claim key")
	}
}
