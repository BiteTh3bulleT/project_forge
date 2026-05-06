package forgekshadow

import (
	"time"

	"forge/projectforge/services/core/internal/forgek/shadowharness"
)

type ObservationInput struct {
	WorkspaceID    string
	RequestID      string
	LivePath       string
	Method         string
	Path           string
	RequestSummary string
	Metadata       map[string]any
}

type DiagnosticReport struct {
	Observation       shadowharness.ShadowObservation
	Comparison        shadowharness.ShadowComparisonReport
	RouteEnvelope     *RouteEnvelopeObservation
	ChatMetadata      *ChatMetadataObservation
	RetrievalMetadata *RetrievalMetadataObservation
	StoredAt          time.Time
}

type RouteEnvelopeInput struct {
	WorkspaceID   string
	RequestID     string
	CorrelationID string
	Method        string
	Path          string
	RoutePattern  string
	RouteClass    string
	StatusCode    int
	Duration      time.Duration
	Warnings      []string
	Metadata      map[string]any
}

type RouteEnvelopeObservation struct {
	ObservationID string         `json:"observation_id"`
	ObservedAt    time.Time      `json:"observed_at"`
	Method        string         `json:"method"`
	Path          string         `json:"path,omitempty"`
	RoutePattern  string         `json:"route_pattern,omitempty"`
	RouteClass    string         `json:"route_class"`
	StatusCode    int            `json:"status_code,omitempty"`
	DurationMS    int64          `json:"duration_ms"`
	RequestID     string         `json:"request_id,omitempty"`
	WorkspaceID   string         `json:"workspace_id,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Warnings      []string       `json:"warnings,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type ChatMetadataInput struct {
	WorkspaceID   string
	RequestID     string
	CorrelationID string
	OperationKind string
	ThreadID      string
	MessageID     string
	RoleClass     string
	StreamClass   string
	ModelID       string
	ProviderID    string
	MessageCount  int
	Duration      time.Duration
	Warnings      []string
	Metadata      map[string]any
}

type ChatMetadataObservation struct {
	ObservationID string         `json:"observation_id"`
	ObservedAt    time.Time      `json:"observed_at"`
	OperationKind string         `json:"operation_kind"`
	ThreadID      string         `json:"thread_id,omitempty"`
	MessageID     string         `json:"message_id,omitempty"`
	RoleClass     string         `json:"role_class,omitempty"`
	StreamClass   string         `json:"stream_class,omitempty"`
	ModelID       string         `json:"model_id,omitempty"`
	ProviderID    string         `json:"provider_id,omitempty"`
	MessageCount  int            `json:"message_count,omitempty"`
	DurationMS    int64          `json:"duration_ms"`
	RequestID     string         `json:"request_id,omitempty"`
	WorkspaceID   string         `json:"workspace_id,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Warnings      []string       `json:"warnings,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type RetrievalMetadataInput struct {
	WorkspaceID       string
	RequestID         string
	CorrelationID     string
	RetrievalRunID    string
	RetrievalResultID string
	SourceType        string
	SourceRefID       string
	SourceHash        string
	ResultCount       int
	SelectedCount     int
	ScoreSummary      string
	RankingPosition   int
	RetrievalStrategy string
	IndexType         string
	EmbeddingModelID  string
	FreshnessStatus   string
	Duration          time.Duration
	Warnings          []string
	Metadata          map[string]any
}

type RetrievalMetadataObservation struct {
	ObservationID     string         `json:"observation_id"`
	ObservedAt        time.Time      `json:"observed_at"`
	WorkspaceID       string         `json:"workspace_id,omitempty"`
	RequestID         string         `json:"request_id,omitempty"`
	CorrelationID     string         `json:"correlation_id,omitempty"`
	RetrievalRunID    string         `json:"retrieval_run_id,omitempty"`
	RetrievalResultID string         `json:"retrieval_result_id,omitempty"`
	SourceType        string         `json:"source_type,omitempty"`
	SourceRefID       string         `json:"source_ref_id,omitempty"`
	SourceHash        string         `json:"source_hash,omitempty"`
	ResultCount       int            `json:"result_count,omitempty"`
	SelectedCount     int            `json:"selected_count,omitempty"`
	ScoreSummary      string         `json:"score_summary,omitempty"`
	RankingPosition   int            `json:"ranking_position,omitempty"`
	RetrievalStrategy string         `json:"retrieval_strategy,omitempty"`
	IndexType         string         `json:"index_type,omitempty"`
	EmbeddingModelID  string         `json:"embedding_model_id,omitempty"`
	FreshnessStatus   string         `json:"freshness_status,omitempty"`
	DurationMS        int64          `json:"duration_ms"`
	Warnings          []string       `json:"warnings,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}
