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
	Observation   shadowharness.ShadowObservation
	Comparison    shadowharness.ShadowComparisonReport
	RouteEnvelope *RouteEnvelopeObservation
	StoredAt      time.Time
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
