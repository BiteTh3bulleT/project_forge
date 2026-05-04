package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/config"
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
