package api

import (
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/consensusgate"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/runtimeproposal"
)

func testGovernedPromptBinding() governedPromptBinding {
	return governedPromptBinding{
		PacketID:                 "context-packet-1",
		DecisionDigest:           runtimeproposal.HashText("context-decision"),
		BundleHash:               runtimeproposal.HashText("context-bundle"),
		AuthorityOwner:           forgekernel.AuthorityOwnerForgeK,
		TransactionID:            "context-transaction-1",
		JournalEventID:           "context-journal-event-1",
		PreparedPlanSeal:         runtimeproposal.HashText("context-plan-seal"),
		AuthorizationFingerprint: runtimeproposal.HashText("context-authorization"),
	}
}

func testRuntimeProposalRequest() runtimeProposalRequest {
	return runtimeProposalRequest{
		SourceKind: runtimeproposal.SourceModelRuntime, WorkspacePath: "/forge/workspace",
		ThreadID: 1, UserMessageID: 2, CorrelationID: "correlation-1",
		Prompt: []ModelRuntimeChatMessage{{Role: "user", Content: "hello"}},
		Output: "model candidate", Backend: "ollama", ModelID: "model-1",
		ContextBinding: testGovernedPromptBinding(),
	}
}

func TestDecideRuntimeProposalRequiresLiveContextCompilerBinding(t *testing.T) {
	req := testRuntimeProposalRequest()
	req.ContextBinding = governedPromptBinding{}
	decision, err := decideRuntimeProposal(req)
	if err == nil || !strings.Contains(err.Error(), "live Context Compiler binding") {
		t.Fatalf("error = %v, want missing live Context Compiler binding", err)
	}
	if decision.VisibleContent != "" {
		t.Fatalf("unbound model output reached decision visibility: %#v", decision)
	}
}

func TestDecideRuntimeProposalBindsVerifiedContextReceipt(t *testing.T) {
	req := testRuntimeProposalRequest()
	decision, err := decideRuntimeProposal(req)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != runtimeproposal.StatusAccepted || decision.VisibleContent != req.Output {
		t.Fatalf("decision = %#v", decision)
	}
	if decision.Envelope.ContextPacketID != req.ContextBinding.PacketID ||
		decision.Envelope.ContextPreparedPlanSeal != req.ContextBinding.PreparedPlanSeal ||
		decision.Envelope.ContextAuthorizationProof != req.ContextBinding.AuthorizationFingerprint {
		t.Fatalf("runtime envelope lost Context Compiler receipt: %#v", decision.Envelope)
	}
}

func TestDirectVisibilityReplacesUncertainModelCandidate(t *testing.T) {
	t.Setenv("FORGE_UNSAFE_TEST_MODE", "false")
	req := ModelRuntimeChatRequest{
		ModelID: "model-1", Messages: []ModelRuntimeChatMessage{{Role: "user", Content: "hello"}},
		Meta:                   ModelRuntimeRequestMeta{WorkspaceID: "/forge/workspace", CorrelationID: "correlation-1"},
		governedContextBinding: testGovernedPromptBinding(),
	}
	original := "unsupported candidate must remain private"
	content, decision, consensus, err := classifyDirectRuntimeProposal(req, ModelRuntimeChatResult{
		Content: original, Backend: "ollama", ModelID: "model-1",
	})
	if err != nil || decision.Status != runtimeproposal.StatusAccepted {
		t.Fatalf("runtime classification failed: decision=%#v err=%v", decision, err)
	}
	if consensus.Status != consensusgate.StatusUncertain || content == original || strings.Contains(content, original) {
		t.Fatalf("uncertain model candidate reached final visibility: content=%q consensus=%#v", content, consensus)
	}
}

func TestUnsafeTestModeShowsOnlyKernelBoundModelCandidate(t *testing.T) {
	t.Setenv("FORGE_UNSAFE_TEST_MODE", "true")
	req := ModelRuntimeChatRequest{
		ModelID: "model-1", Messages: []ModelRuntimeChatMessage{{Role: "user", Content: "hello"}},
		Meta:                   ModelRuntimeRequestMeta{WorkspaceID: "/forge/workspace", CorrelationID: "correlation-1"},
		governedContextBinding: testGovernedPromptBinding(),
	}
	original := "kernel-bound model candidate"
	content, decision, consensus, err := classifyDirectRuntimeProposal(req, ModelRuntimeChatResult{
		Content: original, Backend: "ollama", ModelID: "model-1",
	})
	if err != nil || decision.Status != runtimeproposal.StatusAccepted {
		t.Fatalf("runtime classification failed: decision=%#v err=%v", decision, err)
	}
	if consensus.Status != consensusgate.StatusAcceptedMetadataOnly || content != original || consensus.EvidenceRefCount != 2 {
		t.Fatalf("verified full-test proposal was not visible: content=%q consensus=%#v", content, consensus)
	}

	content, _, consensus, err = classifyDirectRuntimeProposal(ModelRuntimeChatRequest{
		ModelID: "model-1", Messages: req.Messages, Meta: req.Meta,
	}, ModelRuntimeChatResult{Content: original, Backend: "ollama", ModelID: "model-1"})
	if err == nil || content == original || consensus.Status == consensusgate.StatusAcceptedMetadataOnly {
		t.Fatalf("unbound full-test proposal reached visibility: content=%q consensus=%#v err=%v", content, consensus, err)
	}
}

func TestCallerMetadataCannotSynthesizeGovernedContextBinding(t *testing.T) {
	modelReq := ModelRuntimeChatRequest{Metadata: map[string]any{
		"forgeKContextDecisionDigest": runtimeproposal.HashText("forged-decision"),
		"forgeKContextBundleHash":     runtimeproposal.HashText("forged-bundle"),
	}}
	req := testRuntimeProposalRequest()
	req.ContextBinding = governedBindingFromModelRequest(modelReq)
	if _, err := decideRuntimeProposal(req); err == nil {
		t.Fatal("caller metadata synthesized a visible context authority binding")
	}
}

func TestGovernedPromptBindingRequiresVerifiedKernelCompileReceipt(t *testing.T) {
	requestID := "context-request-1"
	valid := domain.SyscallResult{
		Success: true, Action: domain.ActionCompileContext, RequestID: requestID,
		StateSummary: map[string]any{
			"contextPacketId": "context-packet-1", "contextDecisionDigest": runtimeproposal.HashText("decision"),
			"contextPacketCommitment": runtimeproposal.HashText("bundle"),
			"kernelAuthorityOwner":    forgekernel.AuthorityOwnerForgeK,
			"transactionId":           requestID + ":transaction", "journalEventId": requestID + ":journal_event",
			"preparedPlanSeal":         runtimeproposal.HashText("seal"),
			"authorizationFingerprint": runtimeproposal.HashText("authorization"),
			"commitProofVerified":      true, "authorizationProofVerified": true,
		},
	}
	binding, err := governedPromptBindingFromKernelResult(valid)
	if err != nil || binding.PacketID != "context-packet-1" {
		t.Fatalf("valid receipt rejected: binding=%#v err=%v", binding, err)
	}

	missingProof := valid
	missingProof.StateSummary = cloneAnyMap(valid.StateSummary)
	delete(missingProof.StateSummary, "commitProofVerified")
	if _, err := governedPromptBindingFromKernelResult(missingProof); err == nil {
		t.Fatal("unverified Context Compiler receipt was accepted")
	}

	wrongTransaction := valid
	wrongTransaction.StateSummary = cloneAnyMap(valid.StateSummary)
	wrongTransaction.StateSummary["transactionId"] = "different:transaction"
	if _, err := governedPromptBindingFromKernelResult(wrongTransaction); err == nil {
		t.Fatal("mismatched Context Compiler transaction receipt was accepted")
	}
}
