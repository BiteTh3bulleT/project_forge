package forgekshadow

import (
	"strings"
	"testing"
	"time"
)

func TestRetrievalMetadataRelationalAdapterMapsSafeFields(t *testing.T) {
	observation, _, err := normalizeRetrievalMetadataInput(RetrievalMetadataInput{
		WorkspaceID:       "workspace-a",
		RequestID:         "request-a",
		CorrelationID:     "correlation-a",
		RetrievalRunID:    "run-1",
		RetrievalResultID: "result-1",
		SourceType:        "file",
		SourceRefID:       "file-ref-1",
		SourceHash:        "sha256:abc",
		ResultCount:       10,
		SelectedCount:     3,
		ScoreSummary:      "high-confidence",
		RankingPosition:   2,
		RetrievalStrategy: "hybrid",
		IndexType:         "fts",
		EmbeddingModelID:  "model-ref",
		FreshnessStatus:   "fresh",
		Duration:          25 * time.Millisecond,
	}, fixedNow(), "retrieval-observation-1")
	if err != nil {
		t.Fatalf("normalize retrieval metadata: %v", err)
	}
	dto, err := NewRetrievalMetadataRelationalAdapter().MapObservation(observation)
	if err != nil {
		t.Fatalf("map retrieval metadata: %v", err)
	}
	if dto.RetrievalRunID != "run-1" || dto.RetrievalResultID != "result-1" || dto.SourceRefID != "file-ref-1" {
		t.Fatalf("unexpected DTO refs: %#v", dto)
	}
	if dto.ResultCount != 10 || dto.SelectedCount != 3 || dto.RankingPosition != 2 {
		t.Fatalf("unexpected DTO counts: %#v", dto)
	}
	if dto.RawQuery != "" || dto.SourceText != "" || dto.VectorJSON != "" {
		t.Fatalf("relational adapter must not populate raw fields: %#v", dto)
	}
}

func TestRetrievalMetadataRelationalAdapterRejectsRawContentFields(t *testing.T) {
	adapter := NewRetrievalMetadataRelationalAdapter()
	for _, tc := range []struct {
		name string
		mut  func(*RetrievalMetadataObservation)
	}{
		{"source text", func(o *RetrievalMetadataObservation) { o.Metadata["source_text"] = "raw source" }},
		{"chunk text", func(o *RetrievalMetadataObservation) { o.Metadata["chunk_text"] = "raw chunk" }},
		{"raw query", func(o *RetrievalMetadataObservation) { o.Metadata["raw_query"] = "raw query" }},
		{"vector", func(o *RetrievalMetadataObservation) { o.Metadata["vector"] = []float64{0.1, 0.2} }},
		{"embedding", func(o *RetrievalMetadataObservation) { o.Metadata["embedding"] = "0.1,0.2" }},
		{"memory content", func(o *RetrievalMetadataObservation) { o.Metadata["memory_content"] = "raw memory" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			observation := safeRetrievalObservationForRelationalTest()
			tc.mut(&observation)
			if _, err := adapter.MapObservation(observation); err == nil {
				t.Fatalf("expected raw content rejection")
			}
		})
	}
}

func TestRetrievalMetadataRelationalAdapterSerializesDeterministically(t *testing.T) {
	adapter := NewRetrievalMetadataRelationalAdapter()
	one, err := adapter.MapObservation(safeRetrievalObservationForRelationalTest())
	if err != nil {
		t.Fatalf("map one: %v", err)
	}
	two, err := adapter.MapObservation(safeRetrievalObservationForRelationalTest())
	if err != nil {
		t.Fatalf("map two: %v", err)
	}
	if one.CanonicalJSON != two.CanonicalJSON {
		t.Fatalf("expected deterministic serialization:\n%s\n%s", one.CanonicalJSON, two.CanonicalJSON)
	}
	for _, forbidden := range []string{"source_text", "chunk_text", "raw_query", "embedding", "vector", "memory_content"} {
		if strings.Contains(strings.ToLower(one.CanonicalJSON), forbidden) {
			t.Fatalf("canonical DTO contains forbidden marker %q: %s", forbidden, one.CanonicalJSON)
		}
	}
}

func TestRetrievalMetadataRelationalAdapterDoesNotCallRetrieval(t *testing.T) {
	adapter := NewRetrievalMetadataRelationalAdapter()
	if adapter.CanExecuteRetrieval() {
		t.Fatalf("relational adapter must not execute retrieval")
	}
}

func safeRetrievalObservationForRelationalTest() RetrievalMetadataObservation {
	return RetrievalMetadataObservation{
		ObservationID:     "retrieval-observation-1",
		ObservedAt:        fixedNow(),
		WorkspaceID:       "workspace-a",
		RequestID:         "request-a",
		CorrelationID:     "correlation-a",
		RetrievalRunID:    "run-1",
		RetrievalResultID: "result-1",
		SourceType:        "file",
		SourceRefID:       "file-ref-1",
		SourceHash:        "sha256:abc",
		ResultCount:       10,
		SelectedCount:     3,
		ScoreSummary:      "high-confidence",
		RankingPosition:   2,
		RetrievalStrategy: "hybrid",
		IndexType:         "fts",
		EmbeddingModelID:  "model-ref",
		FreshnessStatus:   "fresh",
		DurationMS:        25,
		Metadata: map[string]any{
			"safe": "metadata",
		},
	}
}
