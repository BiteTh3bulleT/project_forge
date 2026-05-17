package api

import (
	"net/http"
	"path/filepath"
	"sort"
	"strconv"

	"forge/projectforge/services/core/internal/forgekshadow"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{
		"ok":               true,
		"service":          "forge-core",
		"cpuAuthoritative": true,
	}
	safeModeReasons := []string{}
	if s.cfg.SafeModeForceCPUOnly {
		safeModeReasons = append(safeModeReasons, "safe_mode.force_cpu_only is enabled")
	}
	payload["safeMode"] = map[string]any{
		"active":  s.cfg.SafeModeForceCPUOnly,
		"reasons": safeModeReasons,
	}
	modelRuntimeStatus := map[string]any{
		"available": s.modelRuntime != nil,
		"status":    "unavailable",
	}
	if s.modelRuntime != nil {
		meta := modelRuntimeMetaFromRequestAudit(requestAuditMetaForBackup(r, "", "", "", "health"))
		health, err := s.modelRuntime.Health(r.Context(), meta)
		if err != nil {
			modelRuntimeStatus["status"] = "degraded"
			modelRuntimeStatus["error"] = err.Error()
		} else {
			modelRuntimeStatus["status"] = health.Status
			modelRuntimeStatus["runtimeEnabled"] = health.RuntimeEnabled
			modelRuntimeStatus["gpuAware"] = health.GPUAware
			modelRuntimeStatus["degradedReasons"] = append([]string(nil), health.DegradedReasons...)
			modelRuntimeStatus["policyWarnings"] = append([]string(nil), health.PolicyWarnings...)
		}
	}
	payload["modelRuntime"] = modelRuntimeStatus
	if s.gpuTelemetry != nil {
		payload["gpuTelemetry"] = s.gpuTelemetry.Snapshot(r.Context())
	}
	if s.intelTelemetry != nil {
		payload["intelTelemetry"] = s.intelTelemetry.Snapshot(r.Context())
	}
	if s.embeddings != nil {
		cfg := s.embeddings.CurrentConfig(r.Context())
		payload["embeddings"] = map[string]any{
			"config":         cfg,
			"health":         s.embeddings.ProviderHealth(r.Context(), cfg.Provider, cfg.Model),
			"truthAuthority": false,
		}
	}
	writeJSON(w, http.StatusOK, payload)
	if s.forgeKShadow != nil && s.forgeKShadow.Enabled() {
		s.forgeKShadow.ObserveBestEffort(r.Context(), forgekshadow.ObservationInput{
			WorkspaceID:    s.cfg.WorkspaceDir,
			RequestID:      r.Header.Get("X-Request-ID"),
			LivePath:       "GET /health",
			Method:         r.Method,
			Path:           r.URL.Path,
			RequestSummary: "health route metadata only",
			Metadata: map[string]any{
				"route":      "/health",
				"method":     r.Method,
				"touchpoint": "health",
			},
		})
	}
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"dataDir":      s.cfg.DataDir,
		"dbPath":       filepath.Join(s.cfg.DataDir, "forge.sqlite"),
		"workspaceDir": s.cfg.WorkspaceDir,
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := s.log.Recent(ctx, limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": rows})
}

func (s *Server) handleAdapters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list := s.adapters.List(ctx)
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	writeJSON(w, http.StatusOK, map[string]any{"adapters": list})
}
