package shadowharness

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestShadowComparisonReportNormalizesAndVerifiesNoEffect(t *testing.T) {
	report, err := NewShadowComparisonReport(ShadowComparisonReport{
		ReportID:        "shadow-report-a",
		WorkspaceID:     "workspace-a",
		RequestID:       "request-a",
		GeneratedAt:     time.Unix(200, 0).UTC(),
		ObservationRefs: []string{"obs-b", "obs-a", "obs-a"},
		ConsensusShadow: ConsensusShadowReport{
			ReportID:               "consensus-shadow-a",
			RequestID:              "request-a",
			ConsensusReportRef:     "consensus-report-a",
			AcceptedClaimCount:     2,
			UnsupportedFactCount:   1,
			ComposerGuardPassed:    true,
			DiagnosticOnlyVerified: true,
		},
		ContextShadow: ContextShadowReport{
			ReportID:                     "context-shadow-a",
			RequestID:                    "request-a",
			ContextBundleRef:             "bundle-a",
			BlockCount:                   3,
			StablePrefixHash:             "stable",
			VolatileSuffixHash:           "volatile",
			CacheEligibleBlockCount:      2,
			RejectedEvidenceLeakDetected: false,
			DiagnosticOnlyVerified:       true,
		},
		RAGShadow: RAGShadowReport{
			ReportID:                "rag-shadow-a",
			RequestID:               "request-a",
			RetrievalRefs:           []string{"retrieval-b", "retrieval-a"},
			EvidenceRefs:            []string{"evidence-a"},
			SourceRefs:              []string{"source-a"},
			NormalizedEvidenceCount: 1,
			Tier1Count:              1,
			NoExecutionVerified:     true,
		},
		RuntimeShadow: RuntimeShadowReport{
			ReportID:             "runtime-shadow-a",
			RequestID:            "request-a",
			RuntimeResultRefs:    []string{"runtime-result-a"},
			DriverRefs:           []string{"driver-a"},
			ModelIdentityRefs:    []string{"model-a"},
			ProposalOnlyVerified: true,
		},
		KVShadow: KVShadowReport{
			ReportID:                      "kv-shadow-a",
			RequestID:                     "request-a",
			KVManifestRefs:                []string{"kv-b", "kv-a"},
			CacheHitCount:                 1,
			CacheMissCount:                2,
			AccelerationNotMemoryVerified: true,
		},
		LymphaticShadow: LymphaticShadowReport{
			ReportID:                 "lymph-shadow-a",
			RequestID:                "request-a",
			MaintenanceReportRefs:    []string{"maintenance-a"},
			CleanupProposalCount:     2,
			NoSilentMutationVerified: true,
			ProposalsDoNotExecute:    true,
		},
		Divergences:      []string{"context differs", "context differs"},
		Warnings:         []string{"diagnostic only"},
		NoEffectVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report.ObservationRefs, []string{"obs-a", "obs-b"}) {
		t.Fatalf("observation refs not normalized: %#v", report.ObservationRefs)
	}
	if err := ValidateNoEffect(DefaultShadowHarnessPolicy(), report); err != nil {
		t.Fatal(err)
	}
	if !report.IsDiagnosticOnly() || report.CanMutateLiveState() || report.CanExecuteActions() || report.CanWriteMemory() {
		t.Fatalf("report should be no-effect diagnostic: %#v", report)
	}
	first, _ := json.Marshal(report)
	again, err := NewShadowComparisonReport(report)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := json.Marshal(again)
	if string(first) != string(second) {
		t.Fatalf("report serialization should be deterministic\nfirst: %s\nsecond:%s", first, second)
	}
}

func TestNoEffectValidatorRejectsPolicyOrReportSideEffects(t *testing.T) {
	report := validComparisonReport(t)
	policy := DefaultShadowHarnessPolicy()
	policy.AllowMemoryWrites = true
	if err := ValidateNoEffect(policy, report); err == nil {
		t.Fatal("expected side-effectful policy to fail")
	}
	report = validComparisonReport(t)
	report.NoEffectVerified = false
	if err := ValidateNoEffect(DefaultShadowHarnessPolicy(), report); err == nil {
		t.Fatal("expected report without no-effect verification to fail")
	}
}

func validComparisonReport(t *testing.T) ShadowComparisonReport {
	t.Helper()
	report, err := NewShadowComparisonReport(ShadowComparisonReport{
		ReportID:        "shadow-report-valid",
		WorkspaceID:     "workspace-a",
		RequestID:       "request-a",
		GeneratedAt:     time.Unix(200, 0).UTC(),
		ObservationRefs: []string{"obs-a"},
		RAGShadow: RAGShadowReport{
			ReportID:            "rag-shadow-a",
			RequestID:           "request-a",
			NoExecutionVerified: true,
		},
		ConsensusShadow: ConsensusShadowReport{
			ReportID:               "consensus-shadow-a",
			RequestID:              "request-a",
			DiagnosticOnlyVerified: true,
		},
		ContextShadow: ContextShadowReport{
			ReportID:               "context-shadow-a",
			RequestID:              "request-a",
			DiagnosticOnlyVerified: true,
		},
		RuntimeShadow: RuntimeShadowReport{
			ReportID:             "runtime-shadow-a",
			RequestID:            "request-a",
			ProposalOnlyVerified: true,
		},
		KVShadow: KVShadowReport{
			ReportID:                      "kv-shadow-a",
			RequestID:                     "request-a",
			AccelerationNotMemoryVerified: true,
		},
		LymphaticShadow: LymphaticShadowReport{
			ReportID:                 "lymph-shadow-a",
			RequestID:                "request-a",
			NoSilentMutationVerified: true,
			ProposalsDoNotExecute:    true,
		},
		NoEffectVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return report
}
