package shadowharness

import "testing"

func TestRAGShadowReportIsRefOnlyAndNoExecution(t *testing.T) {
	report, err := NewRAGShadowReport(RAGShadowReport{
		ReportID:                "rag-shadow-a",
		RequestID:               "request-a",
		RetrievalRefs:           []string{"retrieval-b", "retrieval-a", "retrieval-a"},
		EvidenceRefs:            []string{"evidence-a"},
		SourceRefs:              []string{"source-a"},
		NormalizedEvidenceCount: 1,
		Tier1Count:              1,
		Warnings:                []string{"observed existing refs only"},
		NoExecutionVerified:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.RetrievalRefs) != 2 || report.RetrievalRefs[0] != "retrieval-a" {
		t.Fatalf("retrieval refs not normalized: %#v", report.RetrievalRefs)
	}
	if report.CanExecuteRetrieval() || report.CanCallEmbeddings() || report.CanWriteMemory() ||
		report.CanCompileContext() || report.CanAffectUserVisibleOutput() {
		t.Fatalf("RAG shadow report has forbidden authority: %#v", report)
	}
}

func TestConsensusContextRuntimeKVLymphaticReportsAreDiagnostic(t *testing.T) {
	consensus := ConsensusShadowReport{
		ReportID:               "consensus-shadow-a",
		RequestID:              "request-a",
		AcceptedClaimCount:     1,
		UnsupportedFactCount:   2,
		ComposerGuardPassed:    true,
		DiagnosticOnlyVerified: true,
	}
	if consensus.AcceptedClaimsBecomeTruth() || consensus.EmitsUserVisibleOutput() {
		t.Fatal("consensus shadow report must be diagnostic only")
	}
	context := ContextShadowReport{
		ReportID:                     "context-shadow-a",
		RequestID:                    "request-a",
		RejectedEvidenceLeakDetected: true,
		DiagnosticOnlyVerified:       true,
	}
	if !context.RejectedEvidenceLeakDetected {
		t.Fatal("context report should expose rejected evidence leak detection")
	}
	if context.ModifiesContextBundle() || context.AltersLiveCompileContext() {
		t.Fatal("context shadow report must not mutate context")
	}
	runtime := RuntimeShadowReport{
		ReportID:             "runtime-shadow-a",
		RequestID:            "request-a",
		ProposalOnlyVerified: true,
	}
	if runtime.CanCallModelRuntime() || !runtime.ProposalOnlyVerified {
		t.Fatal("runtime shadow report must be proposal-only and no-call")
	}
	kv := KVShadowReport{
		ReportID:                      "kv-shadow-a",
		RequestID:                     "request-a",
		AccelerationNotMemoryVerified: true,
	}
	if kv.CanReuseLiveKV() || !kv.AccelerationNotMemoryVerified {
		t.Fatal("KV shadow report must verify acceleration-not-memory without live reuse")
	}
	lymphatic := LymphaticShadowReport{
		ReportID:                 "lymph-shadow-a",
		RequestID:                "request-a",
		NoSilentMutationVerified: true,
		ProposalsDoNotExecute:    true,
	}
	if lymphatic.CanExecuteProposals() || !lymphatic.NoSilentMutationVerified {
		t.Fatal("lymphatic shadow report proposals must not execute")
	}
}
