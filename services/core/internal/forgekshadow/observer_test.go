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
		{"set cookie", map[string]any{"set_cookie": "redacted"}},
		{"x api key", map[string]any{"x_api_key": "redacted"}},
		{"auth", map[string]any{"auth": "redacted"}},
		{"jwt", map[string]any{"jwt": "redacted"}},
		{"refresh token", map[string]any{"refresh_token": "redacted"}},
		{"access token", map[string]any{"access_token": "redacted"}},
		{"request body", map[string]any{"body": "raw request"}},
		{"response body", map[string]any{"response_body": "raw response"}},
		{"prompt", map[string]any{"prompt": "raw prompt"}},
		{"raw content", map[string]any{"raw_content": "raw content"}},
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
		{"search execution", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowSearchExecution = true }},
		{"embedding call", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowEmbeddingCalls = true }},
		{"memory write", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowMemoryWrites = true }},
		{"controllane mutation", func(p *shadowharness.ShadowHarnessPolicy) { p.AllowControllaneMutations = true }},
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
		{"api", "/api/meta?token=ignored", "/api/meta", RouteClassAPI},
		{"api query without pattern", "/api/meta?token=ignored", "", RouteClassAPI},
		{"forge", "/forge/models?api_key=ignored", "/forge/models", RouteClassForge},
		{"openai", "/v1/models?secret=ignored", "/v1/models", RouteClassOpenAICompat},
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

func TestRouteEnvelopePrefersPatternAndRejectsUnsafePattern(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	if err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
		WorkspaceID:  "workspace-a",
		RequestID:    "request-a",
		Method:       "GET",
		Path:         "/api/chat/threads/123/assistant-stream?token=should-not-store",
		RoutePattern: "/api/chat/threads/{id}/assistant-stream?token=should-not-store",
		Duration:     15 * time.Millisecond,
	}); err != nil {
		t.Fatalf("observe route envelope: %v", err)
	}
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one report, got %d", len(reports))
	}
	envelope := reports[0].RouteEnvelope
	if envelope == nil {
		t.Fatalf("expected route envelope")
	}
	if envelope.Path != "/api/chat/threads/{id}/assistant-stream" || envelope.RoutePattern != "/api/chat/threads/{id}/assistant-stream" {
		t.Fatalf("expected sanitized route pattern, got %#v", envelope)
	}
	serialized := strings.ToLower(toTestString(reports[0].Observation.Metadata) + " " + reports[0].Observation.LivePath)
	for _, forbidden := range []string{"123", "token", "should-not-store"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("route envelope leaked %q in %q", forbidden, serialized)
		}
	}
}

func TestRouteEnvelopeNormalizesProvidedRouteClassToKnownClasses(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	if err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
		WorkspaceID:  "workspace-a",
		RequestID:    "request-a",
		Method:       "GET",
		Path:         "/api/meta",
		RoutePattern: "/api/meta",
		RouteClass:   "custom-or-secret-class",
	}); err != nil {
		t.Fatalf("observe route envelope: %v", err)
	}
	report := observer.Reports()[0]
	if report.RouteEnvelope == nil || report.RouteEnvelope.RouteClass != RouteClassAPI {
		t.Fatalf("expected canonical api route class, got %#v", report.RouteEnvelope)
	}
}

func TestRouteEnvelopeDropsRawDynamicRoutePatternWhenPatternUnavailable(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	if err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
		WorkspaceID:  "workspace-a",
		RequestID:    "request-a",
		Method:       "GET",
		Path:         "/api/chat/threads/123456789/assistant-stream",
		RoutePattern: "/api/chat/threads/123456789/assistant-stream",
	}); err != nil {
		t.Fatalf("observe route envelope: %v", err)
	}
	report := observer.Reports()[0]
	if report.RouteEnvelope == nil {
		t.Fatalf("expected route envelope")
	}
	if report.RouteEnvelope.Path != "" || report.RouteEnvelope.RoutePattern != "" {
		t.Fatalf("raw dynamic route pattern should be dropped, got %#v", report.RouteEnvelope)
	}
	serialized := strings.ToLower(toTestString(report.Observation.Metadata) + " " + report.Observation.LivePath)
	if strings.Contains(serialized, "123456789") {
		t.Fatalf("raw dynamic route pattern leaked: %q", serialized)
	}
}

func TestRouteEnvelopeRejectsQueryAndRequestURIMetadata(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	for _, key := range []string{"query", "raw_query", "query_string", "request_uri", "url"} {
		t.Run(key, func(t *testing.T) {
			err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
				WorkspaceID:  "workspace-a",
				RequestID:    "request-" + key,
				Method:       "GET",
				RoutePattern: "/api/meta",
				RouteClass:   RouteClassAPI,
				Metadata: map[string]any{
					key: "token=should-not-store",
				},
			})
			if !errors.Is(err, ErrUnsafeMetadata) {
				t.Fatalf("expected unsafe metadata for %q, got %v", key, err)
			}
		})
	}
}

func TestRouteEnvelopeMetadataCannotReintroducePathOrRoutePattern(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	if err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
		WorkspaceID: "workspace-a",
		RequestID:   "request-a",
		Method:      "GET",
		Path:        "/api/chat/threads/123456789/assistant-stream",
		Metadata: map[string]any{
			"path":          "/api/chat/threads/123456789/assistant-stream",
			"route_pattern": "/api/chat/threads/123456789/assistant-stream",
		},
	}); err != nil {
		t.Fatalf("observe route envelope: %v", err)
	}
	report := observer.Reports()[0]
	if _, ok := report.Observation.Metadata["path"]; ok {
		t.Fatalf("caller metadata reintroduced path: %#v", report.Observation.Metadata)
	}
	if _, ok := report.Observation.Metadata["route_pattern"]; ok {
		t.Fatalf("caller metadata reintroduced route pattern: %#v", report.Observation.Metadata)
	}
}

func TestRouteEnvelopeRejectsNonDeterministicMetadataValues(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, nil, fixedNow)
	err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
		WorkspaceID:  "workspace-a",
		RequestID:    "request-a",
		Method:       "GET",
		RoutePattern: "/api/meta",
		RouteClass:   RouteClassAPI,
		Metadata: map[string]any{
			"compound": map[string]any{"a": "b"},
		},
	})
	if !errors.Is(err, ErrUnsafeMetadata) {
		t.Fatalf("expected non-deterministic metadata rejection, got %v", err)
	}
}

func TestRouteEnvelopeBoundedRetentionDropsOldest(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, MaxReports: 2}, nil, fixedNow)
	for _, id := range []string{"request-a", "request-b", "request-c"} {
		if err := observer.ObserveRouteEnvelope(context.Background(), RouteEnvelopeInput{
			WorkspaceID:  "workspace-a",
			RequestID:    id,
			Method:       "GET",
			RoutePattern: "/api/meta",
			RouteClass:   RouteClassAPI,
		}); err != nil {
			t.Fatalf("observe route envelope %s: %v", id, err)
		}
	}
	reports := observer.Reports()
	if len(reports) != 2 {
		t.Fatalf("expected bounded route envelope report count 2, got %d", len(reports))
	}
	if reports[0].Comparison.RequestID != "request-b" || reports[1].Comparison.RequestID != "request-c" {
		t.Fatalf("expected oldest route envelope report dropped, got %#v", reports)
	}
}

func TestObserveRouteEnvelopeBestEffortIgnoresSinkFailure(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true}, failingSink{}, fixedNow)
	observer.ObserveRouteEnvelopeBestEffort(context.Background(), RouteEnvelopeInput{
		WorkspaceID:  "workspace-a",
		RequestID:    "request-a",
		Method:       "GET",
		RoutePattern: "/api/meta",
		RouteClass:   RouteClassAPI,
	})
}

func TestChatMetadataRequiresGlobalAndChatFlags(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want int
	}{
		{"global disabled chat disabled", Config{Enabled: false, ChatMetadataEnabled: false}, 0},
		{"global disabled chat enabled", Config{Enabled: false, ChatMetadataEnabled: true}, 0},
		{"global enabled chat disabled", Config{Enabled: true, ChatMetadataEnabled: false}, 0},
		{"global enabled chat enabled", Config{Enabled: true, ChatMetadataEnabled: true}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserverWithSink(tc.cfg, nil, fixedNow)
			if err := observer.ObserveChatMetadata(context.Background(), ChatMetadataInput{
				WorkspaceID:   "workspace-a",
				RequestID:     "request-a",
				OperationKind: ChatOperationMessagePost,
				ThreadID:      "thread-1",
				MessageID:     "message-1",
				RoleClass:     "user",
			}); err != nil {
				t.Fatalf("disabled chat metadata observe should not fail: %v", err)
			}
			if reports := observer.Reports(); len(reports) != tc.want {
				t.Fatalf("chat metadata observer stored %d reports, want %d", len(reports), tc.want)
			}
		})
	}
}

func TestChatMetadataEnabledStoresDiagnosticReportWithoutContent(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ChatMetadataEnabled: true, MaxReports: 2}, nil, fixedNow)
	if err := observer.ObserveChatMetadata(context.Background(), ChatMetadataInput{
		WorkspaceID:   "workspace-a",
		RequestID:     "request-a",
		CorrelationID: "correlation-a",
		OperationKind: ChatOperationMessagePost,
		ThreadID:      "42",
		MessageID:     "101",
		RoleClass:     "user",
		StreamClass:   "async",
		ModelID:       "local-model-a",
		MessageCount:  1,
		Duration:      12 * time.Millisecond,
		Warnings:      []string{"diagnostic only"},
		Metadata: map[string]any{
			"request_assistant": true,
		},
	}); err != nil {
		t.Fatalf("observe chat metadata: %v", err)
	}
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one chat metadata report, got %d", len(reports))
	}
	report := reports[0]
	if report.ChatMetadata == nil {
		t.Fatalf("expected typed chat metadata observation")
	}
	chat := report.ChatMetadata
	if chat.OperationKind != ChatOperationMessagePost || chat.ThreadID != "42" || chat.MessageID != "101" || chat.RoleClass != "user" || chat.StreamClass != "async" {
		t.Fatalf("unexpected chat metadata observation: %#v", chat)
	}
	if chat.DurationMS != 12 || chat.MessageCount != 1 {
		t.Fatalf("unexpected chat metadata timing/count: %#v", chat)
	}
	if report.Observation.Metadata["observation_type"] != chatMetadataObservationType {
		t.Fatalf("expected chat metadata marker, got %#v", report.Observation.Metadata)
	}
	serialized := strings.ToLower(toTestString(report.Observation.Metadata) + " " + report.Observation.RequestSummary)
	for _, forbidden := range []string{"prompt", "completion", "body", "tool_output", "retrieval_content", "memory_content"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("chat metadata leaked forbidden content marker %q in %q", forbidden, serialized)
		}
	}
	if !report.Observation.IsDiagnosticOnly() || !report.Comparison.NoEffectVerified {
		t.Fatalf("chat metadata report must remain diagnostic-only: %#v", report)
	}
}

func TestChatMetadataRejectsUnsafeMetadataWithoutStoringReport(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ChatMetadataEnabled: true}, nil, fixedNow)
	for _, key := range []string{
		"api_key", "secret", "token", "password", "private_key", "bearer", "plaintext",
		"credential", "authorization", "cookie", "session", "set_cookie", "x_api_key",
		"auth", "jwt", "refresh_token", "access_token", "body", "request_body",
		"response_body", "raw_content", "content", "message", "message_body", "prompt",
		"completion", "assistant_response", "system_prompt", "tool_output", "tool_payload",
		"retrieval_content", "memory_content", "source_chunk", "file_contents", "request_payload",
	} {
		t.Run(key, func(t *testing.T) {
			err := observer.ObserveChatMetadata(context.Background(), ChatMetadataInput{
				WorkspaceID:   "workspace-a",
				RequestID:     "request-" + key,
				OperationKind: ChatOperationMessagePost,
				ThreadID:      "thread-1",
				MessageID:     "message-1",
				RoleClass:     "user",
				Metadata: map[string]any{
					key: "should not store",
				},
			})
			if !errors.Is(err, ErrUnsafeMetadata) {
				t.Fatalf("expected unsafe metadata for %q, got %v", key, err)
			}
		})
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("unsafe chat metadata stored %d reports", len(reports))
	}
}

func TestChatMetadataNormalizesAllowedEnumsAndRefs(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ChatMetadataEnabled: true}, nil, fixedNow)
	if err := observer.ObserveChatMetadata(context.Background(), ChatMetadataInput{
		WorkspaceID:   "workspace-a",
		RequestID:     "request-a",
		OperationKind: "unknown-operation",
		ThreadID:      "42",
		MessageID:     "101",
		RoleClass:     "owner",
		StreamClass:   "websocket",
		Metadata: map[string]any{
			"message_count": 999,
			"role_class":    "owner",
			"stream_class":  "websocket",
		},
	}); err != nil {
		t.Fatalf("observe chat metadata: %v", err)
	}
	report := observer.Reports()[0]
	if report.ChatMetadata.OperationKind != ChatOperationMessagePost {
		t.Fatalf("unknown operation should normalize to bounded default, got %#v", report.ChatMetadata)
	}
	if report.ChatMetadata.RoleClass != "" || report.ChatMetadata.StreamClass != "" {
		t.Fatalf("unknown role/stream classes should be omitted, got %#v", report.ChatMetadata)
	}
	if _, ok := report.Observation.Metadata["role_class"]; ok {
		t.Fatalf("caller metadata reintroduced invalid role class: %#v", report.Observation.Metadata)
	}
	if _, ok := report.Observation.Metadata["stream_class"]; ok {
		t.Fatalf("caller metadata reintroduced invalid stream class: %#v", report.Observation.Metadata)
	}
	if report.Observation.Metadata["message_count"] != nil {
		t.Fatalf("caller metadata reintroduced reserved message count: %#v", report.Observation.Metadata)
	}
}

func TestChatMetadataRejectsUnsafeRefsAndWarnings(t *testing.T) {
	cases := []struct {
		name  string
		input ChatMetadataInput
	}{
		{"thread secret", ChatMetadataInput{ThreadID: "secret-thread"}},
		{"message token", ChatMetadataInput{MessageID: "token-message"}},
		{"model secret", ChatMetadataInput{ModelID: "secret-model"}},
		{"provider bearer", ChatMetadataInput{ProviderID: "bearer-provider"}},
		{"long thread", ChatMetadataInput{ThreadID: strings.Repeat("x", maxMetadataStringLength+1)}},
		{"unsafe warning", ChatMetadataInput{Warnings: []string{"Bearer value"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserverWithSink(Config{Enabled: true, ChatMetadataEnabled: true}, nil, fixedNow)
			tc.input.WorkspaceID = "workspace-a"
			tc.input.RequestID = "request-a"
			tc.input.OperationKind = ChatOperationMessagePost
			err := observer.ObserveChatMetadata(context.Background(), tc.input)
			if !errors.Is(err, ErrUnsafeMetadata) {
				t.Fatalf("expected unsafe metadata error, got %v", err)
			}
			if reports := observer.Reports(); len(reports) != 0 {
				t.Fatalf("unsafe chat metadata stored %d reports", len(reports))
			}
		})
	}
}

func TestChatMetadataDeterministicSerializationForStableShape(t *testing.T) {
	left, _, err := normalizeChatMetadataInput(ChatMetadataInput{
		WorkspaceID:   "workspace-a",
		RequestID:     "request-a",
		OperationKind: ChatOperationMessagePost,
		ThreadID:      "42",
		MessageID:     "101",
		RoleClass:     "user",
		StreamClass:   "none",
		Metadata: map[string]any{
			"b": "two",
			"a": "one",
		},
	}, fixedNow(), "chat-meta-1")
	if err != nil {
		t.Fatalf("normalize left: %v", err)
	}
	right, _, err := normalizeChatMetadataInput(ChatMetadataInput{
		WorkspaceID:   "workspace-a",
		RequestID:     "request-a",
		OperationKind: ChatOperationMessagePost,
		ThreadID:      "42",
		MessageID:     "101",
		RoleClass:     "user",
		StreamClass:   "none",
		Metadata: map[string]any{
			"a": "one",
			"b": "two",
		},
	}, fixedNow(), "chat-meta-1")
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
		t.Fatalf("chat metadata serialization should be deterministic\nleft=%s\nright=%s", leftJSON, rightJSON)
	}
}

func TestChatMetadataBoundedRetentionDropsOldest(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ChatMetadataEnabled: true, MaxReports: 2}, nil, fixedNow)
	for _, id := range []string{"request-a", "request-b", "request-c"} {
		if err := observer.ObserveChatMetadata(context.Background(), ChatMetadataInput{
			WorkspaceID:   "workspace-a",
			RequestID:     id,
			OperationKind: ChatOperationMessagePost,
			ThreadID:      "thread-1",
			MessageID:     id,
			RoleClass:     "user",
		}); err != nil {
			t.Fatalf("observe chat metadata %s: %v", id, err)
		}
	}
	reports := observer.Reports()
	if len(reports) != 2 {
		t.Fatalf("expected bounded chat metadata report count 2, got %d", len(reports))
	}
	if reports[0].Comparison.RequestID != "request-b" || reports[1].Comparison.RequestID != "request-c" {
		t.Fatalf("expected oldest chat metadata report dropped, got %#v", reports)
	}
}

func TestChatMetadataEnabledWithDisabledSinkStoresNoReports(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ChatMetadataEnabled: true, DisableSink: true}, nil, fixedNow)
	if err := observer.ObserveChatMetadata(context.Background(), ChatMetadataInput{
		WorkspaceID:   "workspace-a",
		RequestID:     "request-a",
		OperationKind: ChatOperationMessagePost,
		ThreadID:      "thread-1",
		MessageID:     "101",
		RoleClass:     "user",
	}); err != nil {
		t.Fatalf("observe with disabled sink: %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("disabled sink stored %d chat metadata reports", len(reports))
	}
}

func TestChatMetadataRefusesAnySideEffectPolicy(t *testing.T) {
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
			observer := NewObserverWithSink(Config{Enabled: true, ChatMetadataEnabled: true}, nil, fixedNow)
			tc.mut(&observer.policy)
			err := observer.ObserveChatMetadata(context.Background(), ChatMetadataInput{
				WorkspaceID:   "workspace-a",
				RequestID:     "request-a",
				OperationKind: ChatOperationMessagePost,
				ThreadID:      "42",
				MessageID:     "101",
				RoleClass:     "user",
			})
			if !errors.Is(err, ErrPolicyRejected) {
				t.Fatalf("expected policy rejection, got %v", err)
			}
			if reports := observer.Reports(); len(reports) != 0 {
				t.Fatalf("side-effectful policy stored %d chat metadata reports", len(reports))
			}
		})
	}
}

func TestObserveChatMetadataBestEffortIgnoresSinkFailure(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ChatMetadataEnabled: true}, failingSink{}, fixedNow)
	observer.ObserveChatMetadataBestEffort(context.Background(), ChatMetadataInput{
		WorkspaceID:   "workspace-a",
		RequestID:     "request-a",
		OperationKind: ChatOperationMessagePost,
		ThreadID:      "thread-1",
		MessageID:     "message-1",
		RoleClass:     "user",
	})
}

func fixedNow() time.Time {
	return time.Unix(1700000000, 0).UTC()
}

func toTestString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

type failingSink struct{}

func (failingSink) Store(context.Context, DiagnosticReport) error { return errors.New("sink failed") }
func (failingSink) List() []DiagnosticReport                      { return nil }
