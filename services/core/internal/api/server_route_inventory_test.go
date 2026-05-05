package api

import (
	"context"
	"errors"
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

		{http.MethodGet, "/forge/models"},
		{http.MethodPost, "/forge/models/{id}/load"},
		{http.MethodPost, "/forge/models/{id}/unload"},
		{http.MethodPost, "/forge/models/{id}/chat"},
		{http.MethodGet, "/forge/model-runtime/health"},
		{http.MethodGet, "/forge/model-runtime/queue"},
		{http.MethodGet, "/forge/model-runtime/loaded"},

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
		if strings.Contains(normalized, "forgek-shadow") || strings.Contains(normalized, "shadow-diagnostic") {
			t.Fatalf("phase 12C must not expose public diagnostics route: %s", route)
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

func TestForgeKShadowDoesNotObserveNonHealthRoute(t *testing.T) {
	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true})
	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	rr := httptest.NewRecorder()
	(&Server{
		cfg:          config.Config{ForgeKShadowModeEnabled: true},
		forgeKShadow: observer,
	}).Handler().ServeHTTP(rr, req)
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("shadow observer must remain /health-only, got %d reports for /api/meta", len(reports))
	}
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
