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
	Observation shadowharness.ShadowObservation
	Comparison  shadowharness.ShadowComparisonReport
	StoredAt    time.Time
}
