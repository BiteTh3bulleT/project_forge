package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/forgekshadow"
)

func TestServerRouteInventoryRepresentativeCoverage(t *testing.T) {
	srv := &Server{}
	routes := collectServerRoutes(t, srv.Handler())

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{http.MethodGet, "/health/detailed"},

		{http.MethodGet, "/forge/models"},
		{http.MethodPost, "/forge/models/{id}/load"},
		{http.MethodPost, "/forge/models/{id}/unload"},
		{http.MethodPost, "/forge/models/{id}/delete-file"},
		{http.MethodPost, "/forge/models/{id}/chat"},
		{http.MethodGet, "/forge/model-runtime/health"},
		{http.MethodGet, "/forge/model-runtime/queue"},
		{http.MethodGet, "/forge/model-runtime/loaded"},
		{http.MethodGet, "/forge/kernel/status"},
		{http.MethodGet, "/forge/system/status"},
		{http.MethodGet, "/forge/system/host"},

		{http.MethodGet, "/api/meta"},
		{http.MethodGet, "/api/settings"},
		{http.MethodPatch, "/api/settings"},
		{http.MethodPost, "/api/remote/telegram"},
		{http.MethodPost, "/api/remote/discord"},
		{http.MethodGet, "/api/telegram/status"},
		{http.MethodGet, "/api/discord/status"},
		{http.MethodGet, "/api/sources"},
		{http.MethodPost, "/api/sources"},
		{http.MethodGet, "/api/search"},
		{http.MethodGet, "/api/chunks/{id}"},
		{http.MethodGet, "/api/events"},
		{http.MethodGet, "/api/adapters"},
		{http.MethodPost, "/api/commands/execute"},
		{http.MethodGet, "/api/providers/capabilities"},
		{http.MethodGet, "/api/autonomy/status"},
		{http.MethodPost, "/api/autonomy/maintenance/sweep"},
		{http.MethodPost, "/api/dream/run"},
		{http.MethodGet, "/api/dream/reports/{id}"},
		{http.MethodGet, "/api/jobs"},
		{http.MethodPost, "/api/jobs"},
		{http.MethodPost, "/api/jobs/{id}/replay"},
		{http.MethodGet, "/api/chat/threads"},
		{http.MethodPost, "/api/chat/threads/{id}/messages"},
		{http.MethodGet, "/api/chat/threads/{id}/assistant-stream"},
		{http.MethodGet, "/api/canvas/boards"},
		{http.MethodPost, "/api/canvas/boards/{id}/notes"},
		{http.MethodGet, "/api/artifacts/{id}/content"},
		{http.MethodGet, "/api/approvals"},
		{http.MethodPost, "/api/approvals/{id}/approve"},
		{http.MethodGet, "/api/context-inspector/snapshots"},
		{http.MethodGet, "/api/context/restore/recent"},
		{http.MethodPost, "/api/context/restore/outcomes/{id}/feedback"},
		{http.MethodGet, "/api/process/health"},
		{http.MethodPost, "/api/project-context/import"},
		{http.MethodGet, "/api/embeddings/status"},
		{http.MethodPost, "/api/retrieval/runs"},
		{http.MethodGet, "/api/memory/observations"},
		{http.MethodGet, "/api/memory/vsa/reindex-runs"},
		{http.MethodGet, "/api/memory/vsa/reindex-runs/{id}"},
		{http.MethodPost, "/api/memory/repair/run"},
		{http.MethodGet, "/api/dossiers"},
		{http.MethodPost, "/api/evaluations"},
		{http.MethodGet, "/api/lineage/jobs/{id}"},
		{http.MethodPost, "/api/imports/executions"},
		{http.MethodPost, "/api/insights/generate"},
		{http.MethodGet, "/api/dashboard"},
		{http.MethodGet, "/api/strategies"},
		{http.MethodPost, "/api/policy/recommend"},
		{http.MethodPost, "/api/automation/run"},
		{http.MethodPost, "/api/packet-guidance/analyze"},
		{http.MethodGet, "/api/reconciliation"},
		{http.MethodPost, "/api/reviews"},
		{http.MethodPost, "/api/failure-patterns/analyze"},
		{http.MethodGet, "/api/gateway/tools"},
		{http.MethodPatch, "/api/gateway/capabilities/{id}/status"},
		{http.MethodPost, "/api/gateway/invoke"},
		{http.MethodGet, "/api/action-lanes"},
		{http.MethodPost, "/api/permissions/profiles/{id}/activate"},
		{http.MethodGet, "/api/audit/trace"},
		{http.MethodPost, "/api/backup/restore"},
		{http.MethodGet, "/api/release/readiness"},
	} {
		assertRouteMounted(t, routes, tc.method, tc.path)
	}
}

func TestServerRouteInventoryOpenAICompatConditional(t *testing.T) {
	disabled := collectServerRoutes(t, (&Server{}).Handler())
	assertRouteNotMounted(t, disabled, http.MethodGet, "/v1/models")
	assertRouteNotMounted(t, disabled, http.MethodPost, "/v1/chat/completions")

	enabled := collectServerRoutes(t, (&Server{cfg: config.Config{EnableOpenAICompatAPI: true}}).Handler())
	assertRouteMounted(t, enabled, http.MethodGet, "/v1/models")
	assertRouteMounted(t, enabled, http.MethodPost, "/v1/chat/completions")
}

func TestServerRouteInventoryMetricsEndpointConditional(t *testing.T) {
	disabled := collectServerRoutes(t, (&Server{}).Handler())
	assertRouteNotMounted(t, disabled, http.MethodGet, "/metrics")

	enabled := collectServerRoutes(t, (&Server{cfg: config.Config{EnableMetricsEndpoint: true}}).Handler())
	assertRouteMounted(t, enabled, http.MethodGet, "/metrics")
}

func TestServerRouteInventoryCompatibilityAndRetiredRoutes(t *testing.T) {
	srv := &Server{}
	routes := collectServerRoutes(t, srv.Handler())

	assertRouteMounted(t, routes, http.MethodGet, "/api/memory/vsa/reindex/runs")
	assertRouteMounted(t, routes, http.MethodGet, "/api/memory/vsa/reindex/runs/{id}")
	assertRouteMounted(t, routes, http.MethodPost, "/api/memory/observations")
	assertRouteMounted(t, routes, http.MethodPatch, "/api/memory/observations/{id}")
	assertRouteMounted(t, routes, http.MethodPost, "/api/memory/observations/{id}/usefulness")

	req := httptest.NewRequest(http.MethodPost, "/api/adapters/legacy-fake/invoke", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("removed legacy adapter invoke route status=%d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestServerRouteInventoryHealthAndMiddlewareSmoke(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("/health status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestDetailedHealthRequiresBearerTokenWhenConfigured(t *testing.T) {
	srv := &Server{cfg: config.Config{APIToken: "secret"}}

	publicReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	publicRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(publicRR, publicReq)
	if publicRR.Code != http.StatusOK {
		t.Fatalf("/health status=%d, want %d", publicRR.Code, http.StatusOK)
	}

	detailedReq := httptest.NewRequest(http.MethodGet, "/health/detailed", nil)
	detailedRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(detailedRR, detailedReq)
	if detailedRR.Code != http.StatusUnauthorized {
		t.Fatalf("/health/detailed status=%d, want %d", detailedRR.Code, http.StatusUnauthorized)
	}
}

func TestDetailedHealthReturnsBoundedServiceRollup(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	srv.cfg.APIToken = "secret"
	srv.modelRuntime = newFakeModelRuntime()

	req := httptest.NewRequest(http.MethodGet, "/health/detailed?api_key=should-not-appear", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "session=should-not-appear")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("/health/detailed status=%d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var payload struct {
		OK          bool                      `json:"ok"`
		Service     string                    `json:"service"`
		HealthState string                    `json:"healthState"`
		Services    map[string]map[string]any `json:"services"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode detailed health: %v body=%s", err, rr.Body.String())
	}
	if !payload.OK || payload.Service != "forge-core" || payload.HealthState == "" {
		t.Fatalf("unexpected detailed health envelope: %#v", payload)
	}
	for _, name := range []string{"storage", "modelruntime", "gateway", "hostbridge", "forgekshadow", "dream", "autonomy"} {
		if payload.Services[name] == nil {
			t.Fatalf("missing service rollup %q in %#v", name, payload.Services)
		}
		if payload.Services[name]["status"] == "" {
			t.Fatalf("service %q missing status: %#v", name, payload.Services[name])
		}
	}
	if payload.Services["storage"]["status"] != "ok" {
		t.Fatalf("storage status=%v, want ok", payload.Services["storage"]["status"])
	}
	if payload.Services["modelruntime"]["status"] != "ready" {
		t.Fatalf("modelruntime status=%v, want ready", payload.Services["modelruntime"]["status"])
	}
	serialized := strings.ToLower(rr.Body.String())
	for _, forbidden := range []string{"secret", "should-not-appear", "authorization", "cookie", "api_key"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("detailed health leaked forbidden fragment %q in %s", forbidden, rr.Body.String())
		}
	}
}

func TestServerRouteInventoryUnchangedWithForgeKShadowEnabled(t *testing.T) {
	disabled := collectServerRoutes(t, (&Server{}).Handler())
	enabled := collectServerRoutes(t, (&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: forgekshadow.NewObserver(forgekshadow.Config{Enabled: true}),
	}).Handler())
	if !sameRouteSet(disabled, enabled) {
		t.Fatalf("shadow mode changed route inventory\ndisabled=%#v\nenabled=%#v", routeKeys(disabled), routeKeys(enabled))
	}
}

func TestServerRouteInventoryHasNoForgeKShadowDiagnosticRoute(t *testing.T) {
	routes := collectServerRoutes(t, (&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: forgekshadow.NewObserver(forgekshadow.Config{Enabled: true}),
	}).Handler())
	for _, route := range routeKeys(routes) {
		normalized := strings.ToLower(route)
		for _, forbidden := range []string{"forgek-shadow", "shadow-diagnostic", "/api/shadow", "/forge/shadow"} {
			if strings.Contains(normalized, forbidden) {
				t.Fatalf("shadow mode must not expose public diagnostics route: %s", route)
			}
		}
	}
	assertRouteNotMounted(t, routes, http.MethodGet, "/api/forgek-shadow")
	assertRouteNotMounted(t, routes, http.MethodGet, "/api/shadow")
	assertRouteNotMounted(t, routes, http.MethodGet, "/forge/shadow")
}

func TestServerRouteInventoryUnchangedWithForgeKShadowAdvisoryEnabled(t *testing.T) {
	disabled := collectServerRoutes(t, (&Server{}).Handler())
	enabled := collectServerRoutes(t, (&Server{
		cfg: config.Config{
			ForgeKShadowModeEnabled:     true,
			ForgeKShadowAdvisoryEnabled: true,
		},
		forgeKShadow: forgekshadow.NewObserver(forgekshadow.Config{Enabled: true, AdvisoryEnabled: true}),
	}).Handler())
	if !sameRouteSet(disabled, enabled) {
		t.Fatalf("shadow advisory changed route inventory\ndisabled=%#v\nenabled=%#v", routeKeys(disabled), routeKeys(enabled))
	}
	for _, route := range routeKeys(enabled) {
		normalized := strings.ToLower(route)
		for _, forbidden := range []string{"shadow-advisory", "forgek-shadow", "shadow-diagnostic", "/api/shadow", "/forge/shadow"} {
			if strings.Contains(normalized, forbidden) {
				t.Fatalf("shadow advisory must not expose public diagnostics route: %s", route)
			}
		}
	}
}

func TestHealthResponseUnchangedWithForgeKShadowDisabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	disabledRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRR, req.Clone(context.Background()))

	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: false})
	shadowDisabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: false},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(shadowDisabledRR, req.Clone(context.Background()))

	if disabledRR.Code != shadowDisabledRR.Code {
		t.Fatalf("disabled shadow changed /health status baseline=%d disabled=%d", disabledRR.Code, shadowDisabledRR.Code)
	}
	if disabledRR.Body.String() != shadowDisabledRR.Body.String() {
		t.Fatalf("disabled shadow changed /health body baseline=%q disabled=%q", disabledRR.Body.String(), shadowDisabledRR.Body.String())
	}
	if disabledRR.Header().Get("Content-Type") != shadowDisabledRR.Header().Get("Content-Type") {
		t.Fatalf("disabled shadow changed content type baseline=%q disabled=%q", disabledRR.Header().Get("Content-Type"), shadowDisabledRR.Header().Get("Content-Type"))
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("disabled shadow observer stored %d reports", len(reports))
	}
}

func TestHealthResponseUnchangedWithForgeKShadowEnabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "request-health-shadow")

	disabledRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRR, req.Clone(context.Background()))

	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	enabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(enabledRR, req.Clone(context.Background()))

	if disabledRR.Code != enabledRR.Code {
		t.Fatalf("shadow changed /health status disabled=%d enabled=%d", disabledRR.Code, enabledRR.Code)
	}
	if disabledRR.Body.String() != enabledRR.Body.String() {
		t.Fatalf("shadow changed /health body disabled=%q enabled=%q", disabledRR.Body.String(), enabledRR.Body.String())
	}
	if disabledRR.Header().Get("Content-Type") != enabledRR.Header().Get("Content-Type") {
		t.Fatalf("shadow changed content type disabled=%q enabled=%q", disabledRR.Header().Get("Content-Type"), enabledRR.Header().Get("Content-Type"))
	}
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one diagnostic report, got %d", len(reports))
	}
	if reports[0].Observation.Metadata["route"] != "/health" {
		t.Fatalf("expected health metadata only, got %#v", reports[0].Observation.Metadata)
	}
	if _, ok := reports[0].Observation.Metadata["body"]; ok {
		t.Fatalf("shadow report must not capture request/response bodies: %#v", reports[0].Observation.Metadata)
	}
}

func TestHealthResponseUnchangedWithForgeKShadowAdvisoryEnabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "request-health-advisory")

	disabledRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRR, req.Clone(context.Background()))

	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true, AdvisoryEnabled: true})
	enabledRR := httptest.NewRecorder()
	(&Server{
		cfg: config.Config{
			ForgeKShadowModeEnabled:     true,
			ForgeKShadowAdvisoryEnabled: true,
		},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(enabledRR, req.Clone(context.Background()))

	if disabledRR.Code != enabledRR.Code {
		t.Fatalf("shadow advisory changed /health status disabled=%d enabled=%d", disabledRR.Code, enabledRR.Code)
	}
	if disabledRR.Body.String() != enabledRR.Body.String() {
		t.Fatalf("shadow advisory changed /health body disabled=%q enabled=%q", disabledRR.Body.String(), enabledRR.Body.String())
	}
	if disabledRR.Header().Get("Content-Type") != enabledRR.Header().Get("Content-Type") {
		t.Fatalf("shadow advisory changed content type disabled=%q enabled=%q", disabledRR.Header().Get("Content-Type"), enabledRR.Header().Get("Content-Type"))
	}
	reports := observer.Reports()
	if len(reports) != 1 || reports[0].Advisory == nil {
		t.Fatalf("expected one advisory diagnostic report, got %#v", reports)
	}
	if !reports[0].Advisory.NoEffectVerified {
		t.Fatalf("advisory must be no-effect verified: %#v", reports[0].Advisory)
	}
}

func TestAPIResponseUnchangedWithForgeKShadowDisabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)

	baselineRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(baselineRR, req.Clone(context.Background()))

	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: false})
	disabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: false},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(disabledRR, req.Clone(context.Background()))

	assertSameResponse(t, baselineRR, disabledRR, "/api/meta disabled shadow")
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("disabled shadow observer stored %d reports for /api/meta", len(reports))
	}
}

func TestForgeKShadowObservesRouteEnvelopeForAPIRouteWithoutChangingResponse(t *testing.T) {
	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	req.Header.Set("X-Request-ID", "request-api-meta")

	disabledRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRR, req.Clone(context.Background()))

	enabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(enabledRR, req.Clone(context.Background()))

	assertSameResponse(t, disabledRR, enabledRR, "/api/meta enabled shadow")
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one route envelope report for /api/meta, got %d", len(reports))
	}
	envelope := reports[0].RouteEnvelope
	if envelope == nil {
		t.Fatalf("expected typed route envelope report")
	}
	if envelope.Method != http.MethodGet || envelope.RoutePattern != "/api/meta" || envelope.RouteClass != forgekshadow.RouteClassAPI {
		t.Fatalf("unexpected route envelope: %#v", envelope)
	}
	if envelope.Path != "/api/meta" {
		t.Fatalf("route envelope should store safe route pattern as path, got %q", envelope.Path)
	}
	if _, ok := reports[0].Observation.Metadata["body"]; ok {
		t.Fatalf("route envelope captured body metadata: %#v", reports[0].Observation.Metadata)
	}
	if _, ok := reports[0].Observation.Metadata["response_body"]; ok {
		t.Fatalf("route envelope captured response body metadata: %#v", reports[0].Observation.Metadata)
	}
}

func TestForgeKShadowRouteEnvelopeDoesNotCapturePOSTBody(t *testing.T) {
	body := []byte(`{"prompt":"do not capture this"`)
	req := func() *http.Request {
		return httptest.NewRequest(http.MethodPost, "/api/commands/execute", bytes.NewReader(body))
	}
	disabledRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRR, req())

	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	enabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(enabledRR, req())

	assertSameResponse(t, disabledRR, enabledRR, "POST /api/commands/execute enabled shadow")
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one route envelope report for POST route, got %d", len(reports))
	}
	metadata := reports[0].Observation.Metadata
	for _, forbidden := range []string{"body", "request_body", "response_body", "prompt", "content", "authorization", "cookie"} {
		if _, ok := metadata[forbidden]; ok {
			t.Fatalf("route envelope captured forbidden %q metadata: %#v", forbidden, metadata)
		}
	}
	if strings.Contains(strings.ToLower(reports[0].Observation.RequestSummary), "do not capture") {
		t.Fatalf("route envelope summary captured request body: %q", reports[0].Observation.RequestSummary)
	}
}

func TestForgeKShadowRouteEnvelopeDoesNotCaptureAuthHeadersOrCookies(t *testing.T) {
	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	req.Header.Set("Authorization", "Bearer should-not-appear")
	req.Header.Set("Cookie", "session=should-not-appear")

	rr := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("/api/meta status=%d, want %d", rr.Code, http.StatusOK)
	}
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one route envelope report, got %d", len(reports))
	}
	for key, value := range reports[0].Observation.Metadata {
		normalizedKey := strings.ToLower(key)
		normalizedValue := strings.ToLower(toString(value))
		if strings.Contains(normalizedKey, "authorization") || strings.Contains(normalizedKey, "cookie") ||
			strings.Contains(normalizedValue, "should-not-appear") || strings.Contains(normalizedValue, "bearer") {
			t.Fatalf("route envelope captured auth/cookie material in %q=%v", key, value)
		}
	}
}

func TestForgeKShadowRouteEnvelopePrefersPatternOverRawPathAndQuery(t *testing.T) {
	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	req := httptest.NewRequest(http.MethodGet, "/api/chat/threads/123/assistant-stream?userMessageId=abc&token=should-not-appear", nil)

	disabledRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRR, req.Clone(context.Background()))

	enabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(enabledRR, req.Clone(context.Background()))

	assertSameResponse(t, disabledRR, enabledRR, "assistant stream invalid query enabled shadow")
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one route envelope report, got %d", len(reports))
	}
	envelope := reports[0].RouteEnvelope
	if envelope == nil {
		t.Fatalf("expected route envelope report")
	}
	if envelope.RoutePattern != "/api/chat/threads/{id}/assistant-stream" || envelope.Path != "/api/chat/threads/{id}/assistant-stream" {
		t.Fatalf("expected matched route pattern, got %#v", envelope)
	}
	serialized := strings.ToLower(toString(reports[0].Observation.Metadata) + " " + reports[0].Observation.LivePath)
	for _, forbidden := range []string{"123", "usermessageid", "abc", "token", "should-not-appear"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("route envelope leaked raw path/query fragment %q in %q", forbidden, serialized)
		}
	}
}

func TestForgeKShadowRouteEnvelopeUnmatchedRouteResponseUnchangedAndNoReport(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing/123?token=should-not-appear", nil)
	disabledRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRR, req.Clone(context.Background()))

	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	enabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(enabledRR, req.Clone(context.Background()))

	assertSameResponse(t, disabledRR, enabledRR, "unmatched route enabled shadow")
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("unmatched route should not store raw path diagnostics, got %d reports", len(reports))
	}
}

func TestForgeKShadowRouteEnvelopeForgeRouteResponseUnchanged(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/forge/model-runtime/health", nil)
	req.Header.Set("X-Request-ID", "request-forge-runtime-health")
	disabledRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRR, req.Clone(context.Background()))

	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	enabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(enabledRR, req.Clone(context.Background()))

	assertSameResponse(t, disabledRR, enabledRR, "/forge/model-runtime/health enabled shadow")
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one route envelope report, got %d", len(reports))
	}
	if reports[0].RouteEnvelope == nil || reports[0].RouteEnvelope.RouteClass != forgekshadow.RouteClassForge {
		t.Fatalf("expected forge route class, got %#v", reports[0].RouteEnvelope)
	}
}

func TestForgeKShadowRouteEnvelopeOpenAICompatConditionalBehaviorUnchanged(t *testing.T) {
	disabledRouteReq := httptest.NewRequest(http.MethodGet, "/v1/models?token=should-not-appear", nil)
	disabledRouteReq.Header.Set("X-Request-ID", "request-v1-disabled")
	disabledRouteRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRouteRR, disabledRouteReq.Clone(context.Background()))

	disabledObserver := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	disabledEnabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: disabledObserver,
	}).Handler().ServeHTTP(disabledEnabledRR, disabledRouteReq.Clone(context.Background()))
	assertSameResponse(t, disabledRouteRR, disabledEnabledRR, "/v1/models compat disabled shadow")
	if reports := disabledObserver.Reports(); len(reports) != 0 {
		t.Fatalf("unmounted /v1 route should not store diagnostics, got %d reports", len(reports))
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models?api_key=should-not-appear", nil)
	req.Header.Set("X-Request-ID", "request-v1-enabled")
	compatDisabledRR := httptest.NewRecorder()
	(&Server{cfg: config.Config{EnableOpenAICompatAPI: true}}).Handler().ServeHTTP(compatDisabledRR, req.Clone(context.Background()))

	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	compatEnabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{EnableOpenAICompatAPI: true, ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(compatEnabledRR, req.Clone(context.Background()))

	assertSameResponse(t, compatDisabledRR, compatEnabledRR, "/v1/models compat enabled shadow")
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one /v1 route envelope report, got %d", len(reports))
	}
	if reports[0].RouteEnvelope == nil || reports[0].RouteEnvelope.RouteClass != forgekshadow.RouteClassOpenAICompat {
		t.Fatalf("expected openai_compat route class, got %#v", reports[0].RouteEnvelope)
	}
}

func TestForgeKShadowRouteEnvelopeSinkFailureDoesNotChangeAPIRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	disabledRR := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(disabledRR, req.Clone(context.Background()))

	enabledRR := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: forgekshadow.NewObserverWithSink(forgekshadow.Config{Enabled: true}, failingShadowSink{}, nil),
	}).Handler().ServeHTTP(enabledRR, req.Clone(context.Background()))

	assertSameResponse(t, disabledRR, enabledRR, "/api/meta shadow sink failure")
}

func TestHealthResponseUnchangedWhenForgeKShadowSinkFails(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	observer := forgekshadow.NewObserverWithSink(forgekshadow.Config{Enabled: true}, failingShadowSink{}, nil)
	rr := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("/health status=%d, want %d", rr.Code, http.StatusOK)
	}
}

func TestServerRouteInventoryAssistantStreamOrderGuardrail(t *testing.T) {
	routes := collectServerRoutes(t, (&Server{}).Handler())
	streamIndex := routeIndex(routes, http.MethodGet, "/api/chat/threads/{id}/assistant-stream")
	metaIndex := routeIndex(routes, http.MethodGet, "/api/meta")
	if streamIndex < 0 {
		t.Fatalf("assistant stream route is not mounted")
	}
	if metaIndex < 0 {
		t.Fatalf("/api/meta route is not mounted")
	}
	// chi does not expose timeout middleware identity through Walk. This order
	// guard preserves the current structure where assistant-stream is mounted
	// before the timed /api group; behavior tests cover the stream itself.
	if streamIndex > metaIndex {
		t.Fatalf("assistant stream route index=%d should stay before timed /api group index=%d", streamIndex, metaIndex)
	}
}

func assertSameResponse(t *testing.T, left, right *httptest.ResponseRecorder, label string) {
	t.Helper()
	if left.Code != right.Code {
		t.Fatalf("%s changed status left=%d right=%d", label, left.Code, right.Code)
	}
	if left.Body.String() != right.Body.String() {
		t.Fatalf("%s changed body left=%q right=%q", label, left.Body.String(), right.Body.String())
	}
	if left.Header().Get("Content-Type") != right.Header().Get("Content-Type") {
		t.Fatalf("%s changed content type left=%q right=%q", label, left.Header().Get("Content-Type"), right.Header().Get("Content-Type"))
	}
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

type routeInventory map[string]int

func collectServerRoutes(t *testing.T, h http.Handler) routeInventory {
	t.Helper()
	routes, ok := h.(chi.Routes)
	if !ok {
		t.Fatalf("handler type %T does not expose chi routes", h)
	}
	out := routeInventory{}
	index := 0
	if err := chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		out[method+" "+route] = index
		index++
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	return out
}

func assertRouteMounted(t *testing.T, routes routeInventory, method, path string) {
	t.Helper()
	if _, ok := routes[method+" "+path]; !ok {
		t.Fatalf("route %s %s is not mounted", method, path)
	}
}

func assertRouteNotMounted(t *testing.T, routes routeInventory, method, path string) {
	t.Helper()
	if _, ok := routes[method+" "+path]; ok {
		t.Fatalf("route %s %s should not be mounted", method, path)
	}
}

func routeIndex(routes routeInventory, method, path string) int {
	if idx, ok := routes[method+" "+path]; ok {
		return idx
	}
	return -1
}

func sameRouteSet(left, right routeInventory) bool {
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if _, ok := right[key]; !ok {
			return false
		}
	}
	return true
}

func routeKeys(routes routeInventory) []string {
	out := make([]string, 0, len(routes))
	for key := range routes {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

type failingShadowSink struct{}

func (failingShadowSink) Store(context.Context, forgekshadow.DiagnosticReport) error {
	return errors.New("sink failed")
}

func (failingShadowSink) List() []forgekshadow.DiagnosticReport { return nil }
