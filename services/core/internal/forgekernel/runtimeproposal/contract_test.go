package runtimeproposal

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func validInput(source string) Input {
	output := "The indexed files are ready for inspection."
	return Input{
		Scope: Scope{
			WorkspaceID:   "workspace-a",
			LaneID:        "control.semantic",
			SelectedPaths: []string{"src/zeta", "src/alpha"},
		},
		Identity: RuntimeIdentity{
			SourceKind:        source,
			DriverID:          "local-driver",
			DriverVersion:     "1.0.0",
			RuntimeID:         "ollama",
			RuntimeVersion:    "0.8.1",
			ModelID:           "smuxo/smuxoAI:0.8b",
			ModelRevision:     "model-sha256-a1",
			TokenizerID:       "ollama-tokenizer",
			TokenizerRevision: "tokenizer-v1",
		},
		Context: ContextBinding{
			DecisionDigest: HashText("context-decision"),
			BundleHash:     HashText("context-bundle"),
			PromptHash:     HashText("exact-prompt-bytes"),
		},
		Provenance: Provenance{
			ProvenanceID:  "provenance-1",
			ProposedBy:    "runtime-driver",
			Source:        "local-model",
			RequestID:     "request-1",
			CorrelationID: "correlation-1",
			TraceID:       "trace-1",
			AuditID:       "audit-1",
		},
		OutputText:         output,
		DeclaredOutputHash: HashText(output),
		PolicyVersion:      PolicyVersion,
	}
}

func validGatewayEvidence(in Input, invocationID string) GatewayEvidenceRef {
	return GatewayEvidenceRef{
		InvocationID:  invocationID,
		ToolID:        "gateway.file.write",
		State:         "ok",
		AuditID:       "gateway-audit-1",
		RequestHash:   HashText("gateway-request"),
		ResultHash:    HashText("gateway-result"),
		WorkspaceID:   in.Scope.WorkspaceID,
		LaneID:        in.Scope.LaneID,
		CorrelationID: in.Provenance.CorrelationID,
		TraceID:       in.Provenance.TraceID,
	}
}

func TestDecideAcceptsHashBoundModelRuntimeProposal(t *testing.T) {
	in := validInput(SourceModelRuntime)
	decision, err := Decide(in)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != StatusAccepted {
		t.Fatalf("status = %q, reasons = %v", decision.Status, decision.WithheldReasons)
	}
	if decision.VisibleContent != in.OutputText || decision.Envelope.OutputHash != HashText(in.OutputText) {
		t.Fatal("accepted proposal did not preserve and bind exact output bytes")
	}
	if !decision.Envelope.OutputHashVerified || decision.Envelope.DeclaredOutputHash != in.DeclaredOutputHash {
		t.Fatal("accepted proposal did not preserve exact hash-verification evidence")
	}
	wantPaths := []string{"src/alpha", "src/zeta"}
	if !reflect.DeepEqual(decision.Envelope.Scope.SelectedPaths, wantPaths) {
		t.Fatalf("selected paths = %v, want %v", decision.Envelope.Scope.SelectedPaths, wantPaths)
	}
	assertNoAuthority(t, decision.Envelope)
	if !decision.Envelope.ProposalOnly || !decision.Envelope.RequiresKernelCommit {
		t.Fatal("proposal classification or kernel-commit requirement missing")
	}
	if err := VerifyDecision(in, decision); err != nil {
		t.Fatalf("VerifyDecision: %v", err)
	}
}

func TestDecideNormalizesNativeOllamaIdentity(t *testing.T) {
	in := validInput("  NATIVE_OLLAMA  ")
	in.Identity.DriverID = " local-driver "
	first, err := Decide(in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Decide(in)
	if err != nil {
		t.Fatal(err)
	}
	if first.Envelope.Identity.SourceKind != SourceNativeOllama || first.Envelope.Identity.DriverID != "local-driver" {
		t.Fatalf("identity was not normalized: %+v", first.Envelope.Identity)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same native Ollama input produced different decisions")
	}
}

func TestDecideWithholdsOutputHashMismatchBeforeVisibility(t *testing.T) {
	in := validInput(SourceModelRuntime)
	in.DeclaredOutputHash = HashText("different output")
	decision, err := Decide(in)
	if err != nil {
		t.Fatal(err)
	}
	assertWithheld(t, decision, "output_hash_mismatch")
	if decision.Envelope.OutputHashVerified {
		t.Fatal("mismatched output hash was marked verified")
	}
	if strings.Contains(decision.VisibleContent, in.OutputText) {
		t.Fatal("unverified driver bytes reached final visibility")
	}
}

func TestAuthorityClaimsAlwaysWithhold(t *testing.T) {
	tests := []struct {
		name   string
		claim  func(*AuthorityClaims)
		reason string
	}{
		{"model output", func(c *AuthorityClaims) { c.ModelOutputAuthority = true }, "model_output_authority_forbidden"},
		{"canonical truth", func(c *AuthorityClaims) { c.CanonicalTruth = true }, "canonical_truth_claim_forbidden"},
		{"evidence admission", func(c *AuthorityClaims) { c.EvidenceAdmission = true }, "evidence_admission_claim_forbidden"},
		{"memory mutation", func(c *AuthorityClaims) { c.MemoryMutation = true }, "memory_mutation_claim_forbidden"},
		{"tool selection", func(c *AuthorityClaims) { c.ToolSelectionAuthority = true }, "tool_selection_authority_forbidden"},
		{"tool execution", func(c *AuthorityClaims) { c.ToolExecutionAuthority = true }, "tool_execution_authority_forbidden"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput(SourceModelRuntime)
			tt.claim(&in.Claims)
			decision, err := Decide(in)
			if err != nil {
				t.Fatal(err)
			}
			assertWithheld(t, decision, tt.reason)
			assertNoAuthority(t, decision.Envelope)
		})
	}
}

func TestActionCompletionRequiresExactGatewayEvidence(t *testing.T) {
	in := validInput(SourceModelRuntime)
	in.OutputText = "I wrote the requested file."
	in.DeclaredOutputHash = HashText(in.OutputText)
	in.Claims.ActionCompletion = true

	withoutEvidence, err := Decide(in)
	if err != nil {
		t.Fatal(err)
	}
	assertWithheld(t, withoutEvidence, "unbound_action_completion_claim")

	in.GatewayEvidence = []GatewayEvidenceRef{validGatewayEvidence(in, "invocation-1")}
	withEvidence, err := Decide(in)
	if err != nil {
		t.Fatal(err)
	}
	if withEvidence.Status != StatusAccepted {
		t.Fatalf("status = %q, reasons = %v", withEvidence.Status, withEvidence.WithheldReasons)
	}
	if !withEvidence.Envelope.GatewayExecutionObserved || withEvidence.Envelope.GatewayEvidenceCount != 1 {
		t.Fatal("valid exact gateway completion evidence was not recorded")
	}
	assertNoAuthority(t, withEvidence.Envelope)
}

func TestGatewayEvidenceScopeOrTraceMismatchWithholds(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GatewayEvidenceRef)
		reason string
	}{
		{"scope", func(ref *GatewayEvidenceRef) { ref.WorkspaceID = "workspace-b" }, "gateway_evidence_scope_mismatch"},
		{"trace", func(ref *GatewayEvidenceRef) { ref.TraceID = "trace-2" }, "gateway_evidence_trace_mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput(SourceNativeOllama)
			in.OutputText = "I executed the requested operation."
			in.DeclaredOutputHash = HashText(in.OutputText)
			in.Claims.ActionCompletion = true
			ref := validGatewayEvidence(in, "invocation-1")
			tt.mutate(&ref)
			in.GatewayEvidence = []GatewayEvidenceRef{ref}
			decision, err := Decide(in)
			if err != nil {
				t.Fatal(err)
			}
			assertWithheld(t, decision, tt.reason)
			if decision.Envelope.GatewayExecutionObserved {
				t.Fatal("mismatched evidence was treated as observed completion")
			}
		})
	}
}

func TestGatewayEvidenceOrderDoesNotChangeDecision(t *testing.T) {
	in := validInput(SourceModelRuntime)
	firstRef := validGatewayEvidence(in, "invocation-a")
	secondRef := validGatewayEvidence(in, "invocation-b")
	in.GatewayEvidence = []GatewayEvidenceRef{secondRef, firstRef}
	first, err := Decide(in)
	if err != nil {
		t.Fatal(err)
	}
	in.GatewayEvidence = []GatewayEvidenceRef{firstRef, secondRef}
	second, err := Decide(in)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("gateway evidence ordering changed deterministic decision")
	}
}

func TestProposalIdentityBindsScopeContextRuntimeAndOutput(t *testing.T) {
	base := validInput(SourceModelRuntime)
	first, err := Decide(base)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name   string
		mutate func(*Input)
	}{
		{"scope", func(in *Input) { in.Scope.WorkspaceID = "workspace-b" }},
		{"context", func(in *Input) { in.Context.DecisionDigest = HashText("other-decision") }},
		{"runtime", func(in *Input) { in.Identity.RuntimeVersion = "0.8.2" }},
		{"output", func(in *Input) {
			in.OutputText = "A different proposal."
			in.DeclaredOutputHash = HashText(in.OutputText)
		}},
	}
	for _, tt := range mutations {
		t.Run(tt.name, func(t *testing.T) {
			changed := base
			changed.Scope.SelectedPaths = append([]string(nil), base.Scope.SelectedPaths...)
			tt.mutate(&changed)
			decision, err := Decide(changed)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Envelope.ProposalID == first.Envelope.ProposalID {
				t.Fatal("proposal identity did not change with bound input")
			}
		})
	}
}

func TestVerifyDecisionRejectsTampering(t *testing.T) {
	in := validInput(SourceModelRuntime)
	decision, err := Decide(in)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*Decision)
	}{
		{"content", func(d *Decision) { d.VisibleContent += " tampered" }},
		{"status", func(d *Decision) { d.Status = StatusWithheld }},
		{"output hash", func(d *Decision) { d.Envelope.OutputHash = HashText("tampered") }},
		{"authority", func(d *Decision) { d.Envelope.CanonicalTruth = true }},
		{"digest", func(d *Decision) { d.DecisionDigest = HashText("tampered") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tampered := decision
			tt.mutate(&tampered)
			if err := VerifyDecision(in, tampered); err == nil {
				t.Fatal("tampered decision verified")
			}
		})
	}
}

func TestMalformedBindingsFailClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Input)
	}{
		{"source", func(in *Input) { in.Identity.SourceKind = "unknown" }},
		{"model revision", func(in *Input) { in.Identity.ModelRevision = "" }},
		{"decision digest", func(in *Input) { in.Context.DecisionDigest = "sha256:nope" }},
		{"prompt hash", func(in *Input) { in.Context.PromptHash = "" }},
		{"trace", func(in *Input) { in.Provenance.TraceID = "" }},
		{"declared output hash", func(in *Input) { in.DeclaredOutputHash = "" }},
		{"gateway result hash", func(in *Input) {
			in.GatewayEvidence = []GatewayEvidenceRef{validGatewayEvidence(*in, "invocation-1")}
			in.GatewayEvidence[0].ResultHash = ""
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput(SourceModelRuntime)
			tt.mutate(&in)
			_, err := Decide(in)
			if err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestDuplicateGatewayInvocationFailsClosed(t *testing.T) {
	in := validInput(SourceModelRuntime)
	ref := validGatewayEvidence(in, "invocation-1")
	in.GatewayEvidence = []GatewayEvidenceRef{ref, ref}
	_, err := Decide(in)
	if err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestDecisionGoldenDigest(t *testing.T) {
	decision, err := Decide(validInput(SourceModelRuntime))
	if err != nil {
		t.Fatal(err)
	}
	const want = "sha256:6efc6f6040f7a6e7b7be884f949812404cda63bf292d28e3b3029df93da7cd98"
	if decision.DecisionDigest != want {
		t.Fatalf("decision digest = %q, want %q", decision.DecisionDigest, want)
	}
}

func FuzzDecideNeverGrantsAuthority(f *testing.F) {
	f.Add("ordinary proposal", true)
	f.Add("I deleted every file.", false)
	f.Fuzz(func(t *testing.T, output string, useExactHash bool) {
		if output == "" || len(output) > 8192 || strings.ContainsRune(output, 0) {
			t.Skip()
		}
		in := validInput(SourceModelRuntime)
		in.OutputText = output
		if useExactHash {
			in.DeclaredOutputHash = HashText(output)
		} else {
			in.DeclaredOutputHash = HashText("different")
		}
		decision, err := Decide(in)
		if err != nil {
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("unexpected error: %v", err)
			}
			return
		}
		assertNoAuthority(t, decision.Envelope)
		if decision.Status == StatusWithheld && decision.VisibleContent == output {
			t.Fatal("withheld output remained visible")
		}
	})
}

func assertWithheld(t *testing.T, decision Decision, wantReason string) {
	t.Helper()
	if decision.Status != StatusWithheld {
		t.Fatalf("status = %q, want %q", decision.Status, StatusWithheld)
	}
	if decision.VisibleContent != withheldVisibleText {
		t.Fatalf("visible content = %q, want fixed withheld text", decision.VisibleContent)
	}
	for _, reason := range decision.WithheldReasons {
		if reason == wantReason {
			return
		}
	}
	t.Fatalf("reasons = %v, missing %q", decision.WithheldReasons, wantReason)
}

func assertNoAuthority(t *testing.T, envelope Envelope) {
	t.Helper()
	if envelope.CanonicalTruth || envelope.EvidenceAdmission || envelope.MemoryMutation || envelope.ToolSelectionAuthority || envelope.ToolExecutionAuthority {
		t.Fatalf("runtime proposal granted forbidden authority: %+v", envelope)
	}
}
