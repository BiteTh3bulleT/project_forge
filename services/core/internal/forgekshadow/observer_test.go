package forgekshadow

import (
	"context"
	"errors"
	"strings"
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

func TestObserverEnabledWithDisabledSinkStoresNoReports(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, DisableSink: true}, nil, fixedNow)
	if err := observer.Observe(context.Background(), ObservationInput{
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		LivePath:       "GET /health",
		RequestSummary: "health metadata",
		Metadata: map[string]any{
			"route": "/health",
		},
	}); err != nil {
		t.Fatalf("observe with disabled sink: %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("disabled sink stored %d reports", len(reports))
	}
}

func TestObserverRejectsUnsafeMetadataWithoutStoringReport(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	for _, tc := range []struct {
		name     string
		metadata map[string]any
	}{
		{"api key", map[string]any{"api_key": "redacted"}},
		{"secret", map[string]any{"x_secret": "redacted"}},
		{"token", map[string]any{"trace": "token value"}},
		{"password", map[string]any{"password": "redacted"}},
		{"private key", map[string]any{"private_key": "redacted"}},
		{"bearer", map[string]any{"trace": "Bearer value"}},
		{"plaintext", map[string]any{"plaintext": "redacted"}},
		{"credential", map[string]any{"credential": "redacted"}},
		{"authorization", map[string]any{"authorization": "redacted"}},
		{"cookie", map[string]any{"cookie": "redacted"}},
		{"session", map[string]any{"session": "redacted"}},
		{"request body", map[string]any{"body": "raw request"}},
		{"response body", map[string]any{"response_body": "raw response"}},
		{"prompt", map[string]any{"prompt": "raw prompt"}},
		{"large content", map[string]any{"summary": strings.Repeat("x", maxMetadataStringLength+1)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := observer.Observe(context.Background(), ObservationInput{
				WorkspaceID:    "workspace-a",
				RequestID:      "request-" + tc.name,
				LivePath:       "GET /health",
				RequestSummary: "health metadata",
				Metadata:       tc.metadata,
			})
			if !errors.Is(err, ErrUnsafeMetadata) {
				t.Fatalf("expected unsafe metadata error, got %v", err)
			}
		})
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("unsafe metadata stored %d reports", len(reports))
	}
}

func TestObserverRefusesAnySideEffectPolicy(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*shadowharness.ShadowHarnessPolicy)
	}{
		{"live mutation", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowLiveMutation = true }},
		{"tool execution", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowToolExecution = true }},
		{"modelruntime call", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowModelRuntimeCalls = true }},
		{"retrieval execution", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowRetrievalExecution = true }},
		{"embedding call", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowEmbeddingCalls = true }},
		{"memory write", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowMemoryWrites = true }},
		{"user-visible output", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowUserVisibleOutput = true }},
		{"public API change", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowPublicAPIChanges = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
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
				t.Fatalf("side-effectful policy stored %d reports", len(reports))
			}
		})
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

func TestRouteEnvelopeObservationDisabledStoresNoReports(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: false, MaxReports: 2}, nil, fixedNow)
	if err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
		WorkspaceID:   "workspace-a",
		RequestID:     "request-a",
		Method:        "GET",
		Path:          "/api/meta?raw=ignored",
		RoutePattern:  "/api/meta",
		RouteClass:    RouteClassAPI,
		Duration:      25 * time.Millisecond,
		CorrelationID: "correlation-a",
	}); err != nil {
		t.Fatalf("disabled route envelope observe should not fail: %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("disabled observer stored %d route envelope reports", len(reports))
	}
}

func TestRouteEnvelopeObservationEnabledStoresDiagnosticReport(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, MaxReports: 2}, nil, fixedNow)
	if err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
		WorkspaceID:   "workspace-a",
		RequestID:     "request-a",
		Method:        "GET",
		Path:          "/api/meta?raw=ignored",
		RoutePattern:  "/api/meta",
		RouteClass:    RouteClassAPI,
		Duration:      25 * time.Millisecond,
		CorrelationID: "correlation-a",
		Warnings:      []string{"diagnostic only"},
		Metadata: map[string]any{
			"owner": "api",
		},
	}); err != nil {
		t.Fatalf("observe route envelope: %v", err)
	}
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one report, got %d", len(reports))
	}
	report := reports[0]
	if report.RouteEnvelope == nil {
		t.Fatalf("expected typed route envelope observation")
	}
	envelope := report.RouteEnvelope
	if envelope.Method != "GET" || envelope.Path != "/api/meta" || envelope.RoutePattern != "/api/meta" || envelope.RouteClass != RouteClassAPI {
		t.Fatalf("unexpected route envelope: %#v", envelope)
	}
	if envelope.DurationMS != 25 {
		t.Fatalf("duration_ms=%d, want 25", envelope.DurationMS)
	}
	if envelope.CorrelationID != "correlation-a" {
		t.Fatalf("correlation_id=%q", envelope.CorrelationID)
	}
	if report.Observation.Metadata["observation_type"] != "route_envelope" {
		t.Fatalf("expected route envelope metadata marker, got %#v", report.Observation.Metadata)
	}
	if report.Observation.Metadata["path"] != "/api/meta" {
		t.Fatalf("expected safe route pattern path, got %#v", report.Observation.Metadata)
	}
	if _, ok := report.Observation.Metadata["body"]; ok {
		t.Fatalf("route envelope must not store body metadata: %#v", report.Observation.Metadata)
	}
	if !report.Observation.IsDiagnosticOnly() || !report.Comparison.NoEffectVerified {
		t.Fatalf("route envelope report must remain diagnostic-only: %#v", report)
	}
}

func TestRouteEnvelopeNormalizesRouteClass(t *testing.T) {
	cases := []struct {
		name         string
		path         string
		routePattern string
		want         string
	}{
		{"health", "/health", "/health", RouteClassHealth},
		{"api", "/api/meta", "/api/meta", RouteClassAPI},
		{"forge", "/forge/models", "/forge/models", RouteClassForge},
		{"openai", "/v1/models", "/v1/models", RouteClassOpenAICompat},
		{"unknown", "/favicon.ico", "", RouteClassStaticOrUnknown},
		{"other", "/custom", "/custom", RouteClassOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeRouteClass(tc.path, tc.routePattern); got != tc.want {
				t.Fatalf("NormalizeRouteClass(%q, %q)=%q, want %q", tc.path, tc.routePattern, got, tc.want)
			}
		})
	}
}

func TestRouteEnvelopeRejectsUnsafeMetadataWithoutStoringReport(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
		WorkspaceID:  "workspace-a",
		RequestID:    "request-a",
		Method:       "POST",
		Path:         "/api/chat/threads",
		RoutePattern: "/api/chat/threads",
		RouteClass:   RouteClassAPI,
		Metadata: map[string]any{
			"request_body": "raw prompt-like content",
		},
	})
	if !errors.Is(err, ErrUnsafeMetadata) {
		t.Fatalf("expected unsafe metadata error, got %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("unsafe route envelope metadata stored %d reports", len(reports))
	}
}

func TestRouteEnvelopeDoesNotStoreBodyOrSecretFields(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	observer.ObserveRouteEnvelopeBestEffort(context.Background(), RouteEnvelopeInput{
		WorkspaceID:  "workspace-a",
		RequestID:    "request-a",
		Method:       "POST",
		Path:         "/api/chat/threads",
		RoutePattern: "/api/chat/threads",
		RouteClass:   RouteClassAPI,
		Metadata: map[string]any{
			"body":          "raw request",
			"authorization": "Bearer value",
		},
	})
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("unsafe best-effort route envelope stored %d reports", len(reports))
	}
}

func fixedNow() time.Time {
	return time.Unix(1700000000, 0).UTC()
}

type failingSink struct{}

func (failingSink) Store(context.Context, DiagnosticReport) error { return errors.New("sink failed") }
func (failingSink) List() []DiagnosticReport                      { return nil }
