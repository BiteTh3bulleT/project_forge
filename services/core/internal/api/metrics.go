package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"forge/projectforge/services/core/internal/release"
)

var (
	metricsProcessStartedAt = time.Now()
	metricsScrapesTotal     uint64
)

func (s *Server) mountMetricsRoutes(r interface {
	Get(string, http.HandlerFunc)
}) {
	if s == nil || !s.cfg.EnableMetricsEndpoint {
		return
	}
	r.Get("/metrics", s.handleMetrics)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	scrapes := atomic.AddUint64(&metricsScrapesTotal, 1)
	uptimeSeconds := time.Since(metricsProcessStartedAt).Seconds()
	if uptimeSeconds < 0 {
		uptimeSeconds = 0
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "# HELP forge_core_process_uptime_seconds Seconds since this forge-core process started.\n")
	_, _ = fmt.Fprintf(w, "# TYPE forge_core_process_uptime_seconds gauge\n")
	_, _ = fmt.Fprintf(w, "forge_core_process_uptime_seconds %.3f\n", uptimeSeconds)
	_, _ = fmt.Fprintf(w, "# HELP forge_core_build_info Static forge-core build metadata.\n")
	_, _ = fmt.Fprintf(w, "# TYPE forge_core_build_info gauge\n")
	_, _ = fmt.Fprintf(w, "forge_core_build_info{version=\"%s\"} 1\n", prometheusLabelValue(release.BuildVersion))
	_, _ = fmt.Fprintf(w, "# HELP forge_core_public_endpoint_info Static public endpoint inventory.\n")
	_, _ = fmt.Fprintf(w, "# TYPE forge_core_public_endpoint_info gauge\n")
	_, _ = fmt.Fprintf(w, "forge_core_public_endpoint_info{method=\"GET\",route=\"/health\"} 1\n")
	_, _ = fmt.Fprintf(w, "forge_core_public_endpoint_info{method=\"GET\",route=\"/metrics\"} 1\n")
	_, _ = fmt.Fprintf(w, "# HELP forge_core_metrics_scrapes_total Total /metrics scrapes handled by this process.\n")
	_, _ = fmt.Fprintf(w, "# TYPE forge_core_metrics_scrapes_total counter\n")
	_, _ = fmt.Fprintf(w, "forge_core_metrics_scrapes_total %d\n", scrapes)
}

func prometheusLabelValue(value string) string {
	return strings.NewReplacer(
		"\\", "\\\\",
		"\n", "\\n",
		"\"", "\\\"",
	).Replace(value)
}
