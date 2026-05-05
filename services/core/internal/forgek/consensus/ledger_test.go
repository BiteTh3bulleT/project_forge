package consensus

import "testing"

func TestLedgerStoresAndListsDeterministically(t *testing.T) {
	ledger := NewClaimLedger()
	b := testClaimInput()
	b.ClaimID = "claim-b"
	b.Subject = "b"
	a := testClaimInput()
	a.ClaimID = "claim-a"
	a.Subject = "a"
	claimB, _ := NewClaim(b)
	claimA, _ := NewClaim(a)
	if err := ledger.SubmitClaim(claimB); err != nil {
		t.Fatal(err)
	}
	if err := ledger.SubmitClaim(claimA); err != nil {
		t.Fatal(err)
	}
	claims := ledger.ListClaims(ClaimListFilter{RequestID: "request-a"})
	again := ledger.ListClaims(ClaimListFilter{RequestID: "request-a"})
	if len(claims) != 2 || StableHash(claims) != StableHash(again) {
		t.Fatalf("claims not listed deterministically: %#v", claims)
	}
	claims[0].Subject = "mutated"
	got, _ := ledger.GetClaim("claim-a")
	if got.Subject == "mutated" {
		t.Fatal("ledger returned mutable claim")
	}
	if groups := ledger.GroupByClaimKey("request-a"); len(groups) != 2 {
		t.Fatalf("expected two claim groups, got %#v", groups)
	}
}
