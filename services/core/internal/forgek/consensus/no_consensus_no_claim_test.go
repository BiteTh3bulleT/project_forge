package consensus

import "testing"

func TestNoConsensusNoClaim(t *testing.T) {
	raw := claimWithEvidence("claim-raw", "evidence-a", "agent-a", 1)
	raw.Status = StatusProposed
	report := ConsensusReport{ReportID: "report-a", RequestID: "request-a", WorkspaceID: "workspace-a"}
	input, err := BuildComposerInput("input-a", report, []Claim{raw}, nil, "", raw.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if len(input.AcceptedClaims) != 0 || len(input.UncertainClaims) != 0 {
		t.Fatalf("raw proposed claim reached composer: %#v", input)
	}
}
