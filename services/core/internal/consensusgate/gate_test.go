package consensusgate

import "testing"

func TestGateWithholdsUnsupportedHighRiskActionClaim(t *testing.T) {
	decision := Gate(Input{
		Content:           "I deleted the workspace file.",
		Surface:           SurfaceChatFinal,
		WorkspaceID:       "workspace-a",
		CorrelationID:     "corr-1",
		ModelProposalOnly: true,
	})

	if decision.Status != StatusWithheld {
		t.Fatalf("status=%q, want %q: %#v", decision.Status, StatusWithheld, decision)
	}
	if decision.Content == "I deleted the workspace file." || decision.WithheldClaimCount != 1 {
		t.Fatalf("expected content to be withheld, got %#v", decision)
	}
	if decision.CanonicalTruth || decision.MemoryMutation || decision.EvidenceAdmission || decision.GatewayExecution || decision.LiveAuthorityMigration {
		t.Fatalf("consensus gate claimed forbidden authority: %#v", decision)
	}
}

func TestGateAllowsGatewayEvidenceWithoutCanonicalTruth(t *testing.T) {
	decision := Gate(Input{
		Content:               "I wrote the file through the gateway.",
		Surface:               SurfaceActionProposal,
		EvidenceRefs:          []string{"gateway_invocation:1", "gateway_invocation:1"},
		GatewayExecutionState: "ok",
	})

	if decision.Status != StatusAcceptedMetadataOnly {
		t.Fatalf("status=%q, want %q: %#v", decision.Status, StatusAcceptedMetadataOnly, decision)
	}
	if decision.EvidenceRefCount != 1 || !decision.GatewayEvidence {
		t.Fatalf("expected normalized gateway evidence, got %#v", decision)
	}
	if decision.CanonicalTruth || decision.MemoryMutation || decision.EvidenceAdmission || decision.GatewayExecution {
		t.Fatalf("consensus gate must not claim canonical effects: %#v", decision)
	}
}

func TestGateMarksModelOnlyClaimsUncertainAndDetectsConflict(t *testing.T) {
	decision := Gate(Input{
		Content:           "The file exists. The file does not exist.",
		ModelProposalOnly: true,
	})

	if decision.Status != StatusUncertain || !decision.ConflictDetected || decision.UncertainClaimCount != 1 {
		t.Fatalf("expected uncertain conflict, got %#v", decision)
	}
	if len(decision.RiskFlags) == 0 {
		t.Fatalf("expected conflict risk flags, got %#v", decision)
	}
}
