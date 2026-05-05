package consensus

import (
	"errors"
	"testing"
	"time"
)

func TestComposerGuardAcceptedClaimsOnly(t *testing.T) {
	accepted := claimWithEvidence("claim-accepted", "evidence-a", "agent-a", 1)
	accepted.Status = StatusAccepted
	rejected := claimWithEvidence("claim-rejected", "evidence-b", "agent-b", 1)
	rejected.Status = StatusRejected
	uncertainInput := testClaimInput()
	uncertainInput.ClaimID = "claim-uncertain"
	uncertainInput.ClaimType = ClaimTypeUncertainty
	uncertainInput.EvidenceRefs = nil
	uncertain, _ := NewClaim(uncertainInput)
	uncertain.Status = StatusUncertain
	action := claimWithEvidence("claim-action", "evidence-c", "agent-c", 1)
	action.ClaimType = ClaimTypeActionProposal
	action.Status = StatusAccepted
	memory := claimWithEvidence("claim-memory", "evidence-d", "agent-d", 1)
	memory.ClaimType = ClaimTypeMemoryUpdateProposal
	memory.Status = StatusAccepted
	report := ConsensusReport{
		ReportID:         "report-a",
		RequestID:        "request-a",
		WorkspaceID:      "workspace-a",
		AcceptedClaimIDs: []string{accepted.ClaimID, action.ClaimID, memory.ClaimID},
		UncertainClaimIDs: []string{
			uncertain.ClaimID,
		},
		RejectedClaimIDs: []string{rejected.ClaimID},
	}
	input, err := BuildComposerInput("composer-a", report, []Claim{accepted, rejected, uncertain, action, memory}, []string{"concise"}, "current turn", time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatalf("BuildComposerInput failed: %v", err)
	}
	if len(input.AcceptedClaims) != 1 || input.AcceptedClaims[0].ClaimID != accepted.ClaimID {
		t.Fatalf("accepted factual claims not included correctly: %#v", input)
	}
	if len(input.UncertainClaims) != 1 || input.UncertainClaims[0].ClaimID != uncertain.ClaimID {
		t.Fatalf("uncertain claim not isolated: %#v", input)
	}
	if len(input.ApprovedActionProposals) != 1 || len(input.MemoryUpdateProposals) != 1 {
		t.Fatalf("action/memory proposals not separated: %#v", input)
	}
}

func TestComposerGuardRejectsRejectedClaimReference(t *testing.T) {
	claim := claimWithEvidence("claim-rejected", "evidence-a", "agent-a", 1)
	claim.Status = StatusRejected
	input := ResponseCompositionInput{InputID: "input-a", ReportID: "report-a", WorkspaceID: "workspace-a", AcceptedClaims: []Claim{claim}}
	report := ConsensusReport{ReportID: "report-a", RejectedClaimIDs: []string{claim.ClaimID}}
	if err := ValidateComposerInput(input, report); !errors.Is(err, ErrRejectedClaimReference) {
		t.Fatalf("expected rejected claim reference denial, got %v", err)
	}
}
