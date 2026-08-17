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
	original := "The file exists. The file does not exist."
	decision := Gate(Input{
		Content:           original,
		ModelProposalOnly: true,
	})

	if decision.Status != StatusUncertain || !decision.ConflictDetected || decision.UncertainClaimCount != 1 {
		t.Fatalf("expected uncertain conflict, got %#v", decision)
	}
	if len(decision.RiskFlags) == 0 {
		t.Fatalf("expected conflict risk flags, got %#v", decision)
	}
	if decision.Content == original || decision.Content != UncertainVisibleText {
		t.Fatalf("uncertain candidate remained visible: %#v", decision)
	}
}

func TestGateReplacesUngroundedModelOnlyContent(t *testing.T) {
	original := "A confident but unsupported model answer."
	decision := Gate(Input{Content: original, ModelProposalOnly: true})
	if decision.Status != StatusUncertain || decision.Content != UncertainVisibleText {
		t.Fatalf("ungrounded output was not replaced: %#v", decision)
	}
	if decision.Content == original {
		t.Fatal("raw uncertain model output reached final visibility")
	}
}

func TestGateReplacesConflictingContentEvenWithEvidenceMetadata(t *testing.T) {
	original := "The operation completed and did not complete."
	decision := Gate(Input{Content: original, EvidenceRefs: []string{"evidence:1"}})
	if decision.Status != StatusUncertain || decision.Content != UncertainVisibleText {
		t.Fatalf("conflicting evidence-backed output was not replaced: %#v", decision)
	}
}
