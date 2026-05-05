package consensus

import (
	"testing"
	"time"
)

func TestConsensusServiceEvaluateAcceptedRejectedUncertainAndConflicted(t *testing.T) {
	service := NewService()
	opened, err := service.OpenRequest(ConsensusRequest{
		RequestID:   "request-a",
		WorkspaceID: "workspace-a",
		OpenedBy:    "operator",
		OpenedAt:    time.Unix(100, 0).UTC(),
	}, DefaultPolicy("workspace-a", CriticalityLow, time.Unix(100, 0).UTC()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitEvidence(testEvidence("evidence-a", EvidenceTier1)); err != nil {
		t.Fatal(err)
	}
	acceptedInput := testClaimInput()
	acceptedInput.RequestID = opened.RequestID
	acceptedInput.EvidenceRefs = []string{"evidence-a"}
	if _, err := service.SubmitClaim(acceptedInput); err != nil {
		t.Fatal(err)
	}
	missingEvidence := testClaimInput()
	missingEvidence.ClaimID = "claim-missing"
	missingEvidence.RequestID = opened.RequestID
	missingEvidence.Subject = "missing"
	missingEvidence.EvidenceRefs = []string{"missing"}
	if _, err := service.SubmitClaim(missingEvidence); err != nil {
		t.Fatal(err)
	}
	uncertain := testClaimInput()
	uncertain.ClaimID = "claim-uncertain"
	uncertain.RequestID = opened.RequestID
	uncertain.ClaimType = ClaimTypeUncertainty
	uncertain.Subject = "uncertain"
	uncertain.EvidenceRefs = nil
	if _, err := service.SubmitClaim(uncertain); err != nil {
		t.Fatal(err)
	}
	conflict := testClaimInput()
	conflict.ClaimID = "claim-conflict"
	conflict.RequestID = opened.RequestID
	conflict.ValueJSON = "opposed"
	conflict.EvidenceRefs = []string{"evidence-a"}
	if _, err := service.SubmitClaim(conflict); err != nil {
		t.Fatal(err)
	}
	report, err := service.Evaluate(opened.RequestID, "report-a", time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.AcceptedClaimIDs) != 0 || len(report.ConflictedClaimIDs) != 2 || len(report.UncertainClaimIDs) != 1 || len(report.RejectedClaimIDs) != 1 {
		t.Fatalf("unexpected report shape: %#v", report)
	}
	again, err := service.Evaluate(opened.RequestID, "report-b", time.Unix(200, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if StableHash(report.Decisions) != StableHash(again.Decisions) {
		t.Fatal("decision shape should be deterministic excluding report IDs/timestamps")
	}
}
