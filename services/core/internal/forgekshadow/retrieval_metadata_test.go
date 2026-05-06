package forgekshadow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/forgek/shadowharness"
)

func TestRetrievalMetadataRequiresGlobalAndRetrievalFlags(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want int
	}{
		{"both disabled", Config{Enabled: false, RetrievalMetadataEnabled: false}, 0},
		{"global disabled retrieval enabled", Config{Enabled: false, RetrievalMetadataEnabled: true}, 0},
		{"global enabled retrieval disabled", Config{Enabled: true, RetrievalMetadataEnabled: false}, 0},
		{"both enabled", Config{Enabled: true, RetrievalMetadataEnabled: true}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserverWithSink(tc.cfg, nil, fixedNow)
			if err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
				WorkspaceID:       "workspace-a",
				RequestID:         "request-a",
				RetrievalRunID:    "run-1",
				ResultCount:       2,
				SelectedCount:     1,
				ScoreSummary:      "max=0.900 avg=0.500",
				RetrievalStrategy: "keyword",
				IndexType:         "fts",
			}); err != nil {
				t.Fatalf("observe retrieval metadata: %v", err)
			}
			if reports := observer.Reports(); len(reports) != tc.want {
				t.Fatalf("retrieval metadata reports=%d, want %d; reports=%#v", len(reports), tc.want, reports)
			}
		})
	}
}

func TestRetrievalMetadataEnabledStoresDiagnosticReportWithoutContent(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true, MaxReports: 2}, nil, fixedNow)
	if err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
		WorkspaceID:       "workspace-a",
		RequestID:         "request-a",
		CorrelationID:     "correlation-a",
		RetrievalRunID:    "run-42",
		RetrievalResultID: "result-101",
		SourceType:        "chunk",
		SourceRefID:       "chunk-7",
		SourceHash:        "sha256:abcdef",
		ResultCount:       3,
		SelectedCount:     1,
		ScoreSummary:      "max=0.900 avg=0.500",
		RankingPosition:   1,
		RetrievalStrategy: "hybrid",
		IndexType:         "fts",
		EmbeddingModelID:  "local-hash",
		FreshnessStatus:   "fresh",
		Duration:          14 * time.Millisecond,
		Warnings:          []string{"diagnostic only"},
		Metadata: map[string]any{
			"touchpoint": "retrieval_run_created",
		},
	}); err != nil {
		t.Fatalf("observe retrieval metadata: %v", err)
	}
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one retrieval metadata report, got %d", len(reports))
	}
	report := reports[0]
	if report.RetrievalMetadata == nil {
		t.Fatalf("expected typed retrieval metadata observation")
	}
	retrieval := report.RetrievalMetadata
	if retrieval.RetrievalRunID != "run-42" || retrieval.RetrievalResultID != "result-101" || retrieval.SourceRefID != "chunk-7" {
		t.Fatalf("unexpected retrieval refs: %#v", retrieval)
	}
	if retrieval.ResultCount != 3 || retrieval.SelectedCount != 1 || retrieval.RankingPosition != 1 || retrieval.DurationMS != 14 {
		t.Fatalf("unexpected retrieval counts/timing: %#v", retrieval)
	}
	if retrieval.RetrievalStrategy != "hybrid" || retrieval.IndexType != "fts" || retrieval.EmbeddingModelID != "local-hash" {
		t.Fatalf("unexpected retrieval classes: %#v", retrieval)
	}
	if report.Observation.Metadata["observation_type"] != retrievalMetadataObservationType {
		t.Fatalf("expected retrieval metadata marker, got %#v", report.Observation.Metadata)
	}
	serialized := strings.ToLower(fmt.Sprint(report))
	for _, forbidden := range []string{
		"source text", "chunk text", "document content", "raw query", "embedding:", "vector:",
		"prompt", "completion", "model output", "request body", "response body", "memory content",
	} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("retrieval metadata leaked forbidden fragment %q in %q", forbidden, serialized)
		}
	}
	if !report.Observation.IsDiagnosticOnly() || !report.Comparison.NoEffectVerified || report.Comparison.RAGShadow.NoExecutionVerified != true {
		t.Fatalf("retrieval metadata report must remain diagnostic-only/no-execution: %#v", report)
	}
}

func TestRetrievalMetadataRejectsUnsafeMetadataWithoutStoringReport(t *testing.T) {
	for _, key := range []string{
		"source_text", "chunk_text", "document_content", "file_content", "content", "query",
		"raw_query", "query_text", "search_snippet", "snippet", "retrieval_result_body",
		"embedding", "embeddings", "vector", "vectors", "prompt", "completion", "model_output",
		"memory_content", "request_body", "response_body", "authorization", "cookie", "api_key",
		"token", "secret",
	} {
		t.Run(key, func(t *testing.T) {
			observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true}, nil, fixedNow)
			err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
				WorkspaceID:       "workspace-a",
				RequestID:         "request-" + key,
				RetrievalRunID:    "run-1",
				RetrievalStrategy: "keyword",
				Metadata: map[string]any{
					key: "should not store",
				},
			})
			if !errors.Is(err, ErrUnsafeMetadata) {
				t.Fatalf("expected unsafe metadata for %q, got %v", key, err)
			}
			if reports := observer.Reports(); len(reports) != 0 {
				t.Fatalf("unsafe retrieval metadata stored %d reports", len(reports))
			}
		})
	}
}

func TestRetrievalMetadataRejectsRawContentTermsInCallerMetadataValues(t *testing.T) {
	cases := []struct {
		name     string
		metadata map[string]any
	}{
		{"raw query value", map[string]any{"score_bucket": "raw_query"}},
		{"embedding vector value", map[string]any{"note": "embedding_vector"}},
		{"chunk text value", map[string]any{"diagnostic_note": "chunk_text"}},
		{"non finite float", map[string]any{"score": math.NaN()}},
		{"infinite float", map[string]any{"score": math.Inf(1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true}, nil, fixedNow)
			err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
				WorkspaceID:       "workspace-a",
				RequestID:         "request-a",
				RetrievalRunID:    "run-1",
				RetrievalStrategy: "keyword",
				Metadata:          tc.metadata,
			})
			if !errors.Is(err, ErrUnsafeMetadata) {
				t.Fatalf("expected unsafe metadata error, got %v", err)
			}
			if reports := observer.Reports(); len(reports) != 0 {
				t.Fatalf("unsafe retrieval metadata stored %d reports", len(reports))
			}
		})
	}
}

func TestRetrievalMetadataRejectsUnsafeRefsWarningsAndOversizedValues(t *testing.T) {
	cases := []struct {
		name  string
		input RetrievalMetadataInput
	}{
		{"run secret", RetrievalMetadataInput{RetrievalRunID: "secret-run"}},
		{"result token", RetrievalMetadataInput{RetrievalResultID: "token-result"}},
		{"source hash bearer", RetrievalMetadataInput{SourceHash: "bearer-hash"}},
		{"embedding model secret", RetrievalMetadataInput{EmbeddingModelID: "secret-model"}},
		{"unsafe warning", RetrievalMetadataInput{Warnings: []string{"Bearer value"}}},
		{"long run", RetrievalMetadataInput{RetrievalRunID: strings.Repeat("x", maxMetadataStringLength+1)}},
		{"oversized metadata", RetrievalMetadataInput{Metadata: map[string]any{"score_bucket": strings.Repeat("x", maxMetadataStringLength+1)}}},
		{"non deterministic metadata", RetrievalMetadataInput{Metadata: map[string]any{"compound": map[string]any{"a": "b"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true}, nil, fixedNow)
			tc.input.WorkspaceID = "workspace-a"
			tc.input.RequestID = "request-a"
			tc.input.RetrievalStrategy = "keyword"
			err := observer.ObserveRetrievalMetadata(context.Background(), tc.input)
			if !errors.Is(err, ErrUnsafeMetadata) {
				t.Fatalf("expected unsafe metadata error, got %v", err)
			}
			if reports := observer.Reports(); len(reports) != 0 {
				t.Fatalf("unsafe retrieval metadata stored %d reports", len(reports))
			}
		})
	}
}

func TestRetrievalMetadataNormalizesEnumsCountsAndReservedMetadata(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true}, nil, fixedNow)
	if err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
		WorkspaceID:       "workspace-a",
		RequestID:         "request-a",
		RetrievalRunID:    "run-1",
		SourceType:        "raw-user-document-title",
		ResultCount:       -5,
		SelectedCount:     -3,
		RankingPosition:   -1,
		RetrievalStrategy: "unknown strategy",
		IndexType:         "unsafe custom index",
		FreshnessStatus:   "maybe",
		Metadata: map[string]any{
			"result_count":       999,
			"retrieval_strategy": "unknown strategy",
			"index_type":         "unsafe custom index",
		},
	}); err != nil {
		t.Fatalf("observe retrieval metadata: %v", err)
	}
	report := observer.Reports()[0]
	if report.RetrievalMetadata.SourceType != "" || report.RetrievalMetadata.RetrievalStrategy != "" || report.RetrievalMetadata.IndexType != "" || report.RetrievalMetadata.FreshnessStatus != "" {
		t.Fatalf("unknown classes should be omitted, got %#v", report.RetrievalMetadata)
	}
	if report.RetrievalMetadata.ResultCount != 0 || report.RetrievalMetadata.SelectedCount != 0 || report.RetrievalMetadata.RankingPosition != 0 {
		t.Fatalf("negative counts/ranks should normalize to zero, got %#v", report.RetrievalMetadata)
	}
	if _, ok := report.Observation.Metadata["result_count"]; ok {
		t.Fatalf("caller metadata reintroduced reserved count: %#v", report.Observation.Metadata)
	}
	if _, ok := report.Observation.Metadata["retrieval_strategy"]; ok {
		t.Fatalf("caller metadata reintroduced reserved strategy: %#v", report.Observation.Metadata)
	}
}

func TestRetrievalMetadataDeterministicSerializationForStableShape(t *testing.T) {
	left, _, err := normalizeRetrievalMetadataInput(RetrievalMetadataInput{
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
	}, fixedNow(), "retrieval-meta-1")
	if err != nil {
		t.Fatalf("normalize left: %v", err)
	}
	right, _, err := normalizeRetrievalMetadataInput(RetrievalMetadataInput{
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
			"a": "one",
			"b": "two",
		},
	}, fixedNow(), "retrieval-meta-1")
	if err != nil {
		t.Fatalf("normalize right: %v", err)
	}
	leftJSON, err := json.Marshal(left)
	if err != nil {
		t.Fatalf("marshal left: %v", err)
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		t.Fatalf("marshal right: %v", err)
	}
	if string(leftJSON) != string(rightJSON) {
		t.Fatalf("retrieval metadata serialization should be deterministic\nleft=%s\nright=%s", leftJSON, rightJSON)
	}

	leftObserver := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true}, nil, fixedNow)
	rightObserver := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true}, nil, fixedNow)
	if err := leftObserver.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
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
		t.Fatalf("observe left: %v", err)
	}
	if err := rightObserver.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
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
			"a": "one",
			"b": "two",
		},
	}); err != nil {
		t.Fatalf("observe right: %v", err)
	}
	leftReportJSON, err := json.Marshal(leftObserver.Reports()[0])
	if err != nil {
		t.Fatalf("marshal left report: %v", err)
	}
	rightReportJSON, err := json.Marshal(rightObserver.Reports()[0])
	if err != nil {
		t.Fatalf("marshal right report: %v", err)
	}
	if string(leftReportJSON) != string(rightReportJSON) {
		t.Fatalf("retrieval diagnostic report serialization should be deterministic\nleft=%s\nright=%s", leftReportJSON, rightReportJSON)
	}
}

func TestRetrievalMetadataBoundedRetentionDisabledSinkAndBestEffort(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true, MaxReports: 2}, nil, fixedNow)
	for _, id := range []string{"request-a", "request-b", "request-c"} {
		if err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
			WorkspaceID:       "workspace-a",
			RequestID:         id,
			RetrievalRunID:    "run-" + id,
			RetrievalStrategy: "keyword",
		}); err != nil {
			t.Fatalf("observe retrieval metadata %s: %v", id, err)
		}
	}
	reports := observer.Reports()
	if len(reports) != 2 {
		t.Fatalf("expected bounded retrieval metadata report count 2, got %d", len(reports))
	}
	if reports[0].Comparison.RequestID != "request-b" || reports[1].Comparison.RequestID != "request-c" {
		t.Fatalf("expected oldest retrieval metadata report dropped, got %#v", reports)
	}

	disabledSink := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true, DisableSink: true}, nil, fixedNow)
	if err := disabledSink.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
		WorkspaceID:       "workspace-a",
		RequestID:         "request-disabled-sink",
		RetrievalRunID:    "run-disabled-sink",
		RetrievalStrategy: "keyword",
	}); err != nil {
		t.Fatalf("observe with disabled sink: %v", err)
	}
	if reports := disabledSink.Reports(); len(reports) != 0 {
		t.Fatalf("disabled sink stored %d retrieval metadata reports", len(reports))
	}

	failing := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true}, failingSink{}, fixedNow)
	failing.ObserveRetrievalMetadataBestEffort(context.Background(), RetrievalMetadataInput{
		WorkspaceID:       "workspace-a",
		RequestID:         "request-failing-sink",
		RetrievalRunID:    "run-failing-sink",
		RetrievalStrategy: "keyword",
	})
}

func TestRetrievalMetadataRefusesAnySideEffectPolicy(t *testing.T) {
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
			observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true}, nil, fixedNow)
			tc.mut(&observer.policy)
			err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
				WorkspaceID:       "workspace-a",
				RequestID:         "request-a",
				RetrievalRunID:    "run-1",
				RetrievalStrategy: "keyword",
			})
			if !errors.Is(err, ErrPolicyRejected) {
				t.Fatalf("expected policy rejection, got %v", err)
			}
			if reports := observer.Reports(); len(reports) != 0 {
				t.Fatalf("side-effectful policy stored %d retrieval metadata reports", len(reports))
			}
		})
	}
}
