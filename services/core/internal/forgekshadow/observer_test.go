package forgekshadow

import (
	"context"
	"errors"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/forgek/shadowharness"
)

func TestObserverDisabledStoresNoReports(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: false, MaxReports: 2}, nil, fixedNow)
	if err := observer.Observe(context.Background(), ObservationInput{
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		LivePath:       "GET /health",
		RequestSummary: "health metadata",
	}); err != nil {
		t.Fatalf("disabled observe should not fail: %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("disabled observer stored %d reports", len(reports))
	}
}

func TestObserverEnabledStoresDiagnosticReport(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, MaxReports: 2}, nil, fixedNow)
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
	if len(reports) != 1 {
		t.Fatalf("expected one report, got %d", len(reports))
	}
	report := reports[0]
	if !report.Observation.IsDiagnosticOnly() || report.Observation.CanMutateLiveState() || report.Observation.CanAffectUserVisibleOutput() {
		t.Fatalf("observation violates diagnostic-only boundary: %#v", report.Observation)
	}
	if !report.Comparison.IsDiagnosticOnly() || report.Comparison.CanMutateLiveState() || report.Comparison.CanExecuteActions() || report.Comparison.CanWriteMemory() {
		t.Fatalf("comparison violates diagnostic-only boundary: %#v", report.Comparison)
	}
	if err := shadowharness.ValidateNoEffect(shadowharness.DefaultShadowHarnessPolicy(), report.Comparison); err != nil {
		t.Fatalf("comparison should satisfy no-effect policy: %v", err)
	}
}

func TestMemorySinkBoundedRetentionDropsOldest(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, MaxReports: 2}, nil, fixedNow)
	for _, id := range []string{"request-a", "request-b", "request-c"} {
		if err := observer.Observe(context.Background(), ObservationInput{
			WorkspaceID:    "workspace-a",
			RequestID:      id,
			LivePath:       "GET /health",
			RequestSummary: "health metadata",
		}); err != nil {
			t.Fatalf("observe %s: %v", id, err)
		}
	}
	reports := observer.Reports()
	if len(reports) != 2 {
		t.Fatalf("expected bounded report count 2, got %d", len(reports))
	}
	if reports[0].Comparison.RequestID != "request-b" || reports[1].Comparison.RequestID != "request-c" {
		t.Fatalf("expected oldest report dropped, got %#v", reports)
	}
}

func TestObserverRejectsUnsafeMetadataWithoutStoringReport(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	err := observer.Observe(context.Background(), ObservationInput{
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		LivePath:       "GET /health",
		RequestSummary: "health metadata",
		Metadata: map[string]any{
			"api_key": "redacted",
		},
	})
	if !errors.Is(err, ErrUnsafeMetadata) {
		t.Fatalf("expected unsafe metadata error, got %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("unsafe metadata stored %d reports", len(reports))
	}
}

func TestObserverRefusesSideEffectPolicy(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	observer.policy.AllowToolExecution = true
	err := observer.Observe(context.Background(), ObservationInput{
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		LivePath:       "GET /health",
		RequestSummary: "health metadata",
	})
	if !errors.Is(err, ErrPolicyRejected) {
		t.Fatalf("expected policy rejection, got %v", err)
	}
}

func TestObserveBestEffortIgnoresSinkFailure(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, failingSink{}, fixedNow)
	observer.ObserveBestEffort(context.Background(), ObservationInput{
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		LivePath:       "GET /health",
		RequestSummary: "health metadata",
	})
}

func fixedNow() time.Time {
	return time.Unix(1700000000, 0).UTC()
}

type failingSink struct{}

func (failingSink) Store(context.Context, DiagnosticReport) error { return errors.New("sink failed") }
func (failingSink) List() []DiagnosticReport                      { return nil }
