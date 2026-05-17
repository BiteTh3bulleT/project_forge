package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
)

func TestMetricsRouteDisabledByDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	(&Server{}).Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("/metrics status=%d, want %d body=%q", rr.Code, http.StatusNotFound, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "forge_core_") {
		t.Fatalf("disabled /metrics exposed metrics body:\n%s", rr.Body.String())
	}
}

func TestMetricsRouteServesPrometheusTextWhenEnabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	(&Server{cfg: config.Config{EnableMetricsEndpoint: true}}).Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("/metrics status=%d, want %d body=%q", rr.Code, http.StatusOK, rr.Body.String())
	}
	contentType := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain; version=0.0.4") {
		t.Fatalf("/metrics content type=%q, want Prometheus text", contentType)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"# HELP forge_core_process_uptime_seconds",
		"# TYPE forge_core_process_uptime_seconds gauge",
		"forge_core_process_uptime_seconds ",
		"# HELP forge_core_build_info",
		"# TYPE forge_core_build_info gauge",
		"forge_core_build_info{version=\"",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("/metrics body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "%!") {
		t.Fatalf("/metrics body contains fmt formatting error:\n%s", body)
	}
	if !strings.Contains(body, "forge_core_metrics_scrapes_total ") {
		t.Fatalf("/metrics body missing scrape counter:\n%s", body)
	}
}

func TestMetricsRouteDoesNotExposeSecretLookingTextWhenEnabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics?token=should-not-appear", nil)
	req.Header.Set("Authorization", "Bearer should-not-appear")
	req.Header.Set("X-Forge-Remote-Token", "should-not-appear")
	rr := httptest.NewRecorder()

	(&Server{cfg: config.Config{EnableMetricsEndpoint: true}}).Handler().ServeHTTP(rr, req)

	body := strings.ToLower(rr.Body.String())
	for _, forbidden := range []string{
		"should-not-appear",
		"authorization",
		"x-forge-remote-token",
		"api_key",
		"password",
		"secret",
		"token",
		"dsn",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("/metrics body exposed forbidden text %q:\n%s", forbidden, rr.Body.String())
		}
	}
}
