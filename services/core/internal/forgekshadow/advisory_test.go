package forgekshadow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/forgek/shadowharness"
)

func TestShadowAdvisoryRequiresGlobalAndAdvisoryFlags(t *testing.T) {
	cases := []struct {
		name         string
		cfg          Config
		wantReports  int
		wantAdvisory bool
	}{
		{"both disabled", Config{Enabled: false, AdvisoryEnabled: false}, 0, false},
		{"global disabled advisory enabled", Config{Enabled: false, AdvisoryEnabled: true}, 0, false},
		{"global enabled advisory disabled", Config{Enabled: true, AdvisoryEnabled: false}, 1, false},
		{"both enabled", Config{Enabled: true, AdvisoryEnabled: true}, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserverWithSink(tc.cfg, nil, fixedNow)
			if err := observer.Observe(context.Background(), ObservationInput{
				WorkspaceID:    "workspace-a",
				RequestID:      "request-a",
				LivePath:       "GET /health",
				RequestSummary: "health metadata",
				Metadata: map[string]any{
					"route": "/health",
				},
			}); err != nil {
				t.Fatalf("observe: %v", err)
			}
			reports := observer.Reports()
			if len(reports) != tc.wantReports {
				t.Fatalf("reports=%d, want %d", len(reports), tc.wantReports)
			}
			if len(reports) > 0 && (reports[0].Advisory != nil) != tc.wantAdvisory {
				t.Fatalf("advisory present=%v, want %v; report=%#v", reports[0].Advisory != nil, tc.wantAdvisory, reports[0])
			}
		})
	}
}

func TestShadowAdvisoryDoesNotForceEnableChatOrRetrieval(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, AdvisoryEnabled: true}, nil, fixedNow)
	if err := observer.ObserveChatMetadata(context.Background(), ChatMetadataInput{
		WorkspaceID:   "workspace-a",
		RequestID:     "request-chat",
		OperationKind: "message_append",
		ThreadID:      "thread-1",
	}); err != nil {
		t.Fatalf("chat metadata observe: %v", err)
	}
	if err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
		WorkspaceID:       "workspace-a",
		RequestID:         "request-retrieval",
		RetrievalRunID:    "run-1",
		RetrievalStrategy: "keyword",
	}); err != nil {
		t.Fatalf("retrieval metadata observe: %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("advisory flag must not force-enable chat/retrieval observers: %#v", reports)
	}
}

func TestShadowAdvisoryBuildsMetadataOnlyContextAndConsensusSummary(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true, AdvisoryEnabled: true}, nil, fixedNow)
	if err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
		WorkspaceID:       "workspace-a",
		RequestID:         "request-a",
		RetrievalRunID:    "run-42",
		RetrievalResultID: "result-101",
		SourceType:        "chunk",
		SourceRefID:       "chunk-7",
		ResultCount:       3,
		SelectedCount:     1,
		ScoreSummary:      "max=0.900 avg=0.500",
		RetrievalStrategy: "hybrid",
		IndexType:         "fts",
		Duration:          4 * time.Millisecond,
	}); err != nil {
		t.Fatalf("observe retrieval metadata: %v", err)
	}
	report := observer.Reports()[0]
	if report.Advisory == nil {
		t.Fatalf("expected advisory report")
	}
	advisory := report.Advisory
	if !advisory.NoEffectVerified || !advisory.RiskSummary.NoRawContentVerified || !advisory.RiskSummary.MetadataOnly {
		t.Fatalf("advisory must remain no-effect metadata-only: %#v", advisory)
	}
	if advisory.EvidenceSummary.RetrievalMetadataCount != 1 || advisory.EvidenceSummary.SafeRefCount == 0 {
		t.Fatalf("unexpected evidence summary: %#v", advisory.EvidenceSummary)
	}
	if advisory.ContextCompilerAdvisory.Status != "shadow_context_summary_created" ||
		advisory.ContextCompilerAdvisory.BlockCount == 0 ||
		advisory.ContextCompilerAdvisory.BundleHash == "" {
		t.Fatalf("expected shadow context summary, got %#v", advisory.ContextCompilerAdvisory)
	}
	if advisory.ConsensusAdvisory.Status != "metadata_only_uncertain" ||
		advisory.ConsensusAdvisory.UncertainClaimCount == 0 ||
		advisory.ConsensusAdvisory.AcceptedClaimCount != 0 {
		t.Fatalf("expected metadata-only uncertain consensus advisory, got %#v", advisory.ConsensusAdvisory)
	}
	serialized := strings.ToLower(fmt.Sprint(advisory))
	for _, forbidden := range []string{
		"source text", "chunk text", "document content", "raw query", "embedding:", "vector:",
		"prompt", "completion", "model output", "request body", "response body", "memory content",
	} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("advisory leaked forbidden fragment %q in %q", forbidden, serialized)
		}
	}
}

func TestShadowAdvisoryWarnsWhenSafeMetadataIsInsufficient(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, AdvisoryEnabled: true}, nil, fixedNow)
	if err := observer.Observe(context.Background(), ObservationInput{
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		LivePath:       "GET /health",
		RequestSummary: "health metadata",
	}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	advisory := observer.Reports()[0].Advisory
	if advisory == nil {
		t.Fatalf("expected advisory report")
	}
	if advisory.ContextCompilerAdvisory.Status != "insufficient_safe_metadata" {
		t.Fatalf("expected insufficient metadata context warning, got %#v", advisory.ContextCompilerAdvisory)
	}
	if len(advisory.Warnings) == 0 {
		t.Fatalf("expected advisory warnings")
	}
}

func TestShadowAdvisoryRejectsUnsafeMetadataWithoutStoringReport(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, AdvisoryEnabled: true}, nil, fixedNow)
	err := observer.Observe(context.Background(), ObservationInput{
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		LivePath:       "GET /health",
		RequestSummary: "health metadata",
		Metadata: map[string]any{
			"prompt": "must not store",
		},
	})
	if !errors.Is(err, ErrUnsafeMetadata) {
		t.Fatalf("expected unsafe metadata, got %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("unsafe advisory input stored reports: %#v", reports)
	}
}

func TestShadowAdvisoryRefusesAnySideEffectPolicy(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*shadowharness.ShadowHarnessPolicy)
	}{
		{"live mutation", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowLiveMutation = true }},
		{"tool execution", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowToolExecution = true }},
		{"modelruntime call", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowModelRuntimeCalls = true }},
		{"retrieval execution", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowRetrievalExecution = true }},
		{"search execution", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowSearchExecution = true }},
		{"embedding call", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowEmbeddingCalls = true }},
		{"memory write", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowMemoryWrites = true }},
		{"controllane mutation", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowControllaneMutations = true }},
		{"user-visible output", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowUserVisibleOutput = true }},
		{"public API change", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowPublicAPIChanges = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserverWithSink(Config{Enabled: true, AdvisoryEnabled: true}, nil, fixedNow)
			tc.mut(&observer.policy)
			err := observer.Observe(context.Background(), ObservationInput{
				WorkspaceID:    "workspace-a",
				RequestID:      "request-a",
				LivePath:       "GET /health",
				RequestSummary: "health metadata",
			})
			if !errors.Is(err, ErrPolicyRejected) {
				t.Fatalf("expected policy rejection, got %v", err)
			}
			if reports := observer.Reports(); len(reports) != 0 {
				t.Fatalf("side-effectful policy stored advisory reports: %#v", reports)
			}
		})
	}
}

func TestShadowAdvisoryDeterministicSerializationForStableShape(t *testing.T) {
	build := func() DiagnosticReport {
		observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true, AdvisoryEnabled: true}, nil, fixedNow)
		if err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
			WorkspaceID:       "workspace-a",
			RequestID:         "request-a",
			RetrievalRunID:    "run-1",
			RetrievalResultID: "result-1",
			SourceType:        "chunk",
			SourceRefID:       "chunk-1",
			ResultCount:       2,
			SelectedCount:     1,
			ScoreSummary:      "max=0.900 avg=0.500",
			RetrievalStrategy: "keyword",
			Metadata: map[string]any{
				"b": "two",
				"a": "one",
			},
		}); err != nil {
			t.Fatalf("observe: %v", err)
		}
		return observer.Reports()[0]
	}
	leftJSON, err := json.Marshal(build().Advisory)
	if err != nil {
		t.Fatalf("marshal left: %v", err)
	}
	rightJSON, err := json.Marshal(build().Advisory)
	if err != nil {
		t.Fatalf("marshal right: %v", err)
	}
	if string(leftJSON) != string(rightJSON) {
		t.Fatalf("advisory serialization should be deterministic\nleft=%s\nright=%s", leftJSON, rightJSON)
	}
}
