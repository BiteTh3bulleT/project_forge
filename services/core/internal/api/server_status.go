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

func (s *Server) handleDetailedHealth(w http.ResponseWriter, r *http.Request) {
	services := map[string]map[string]any{
		"storage":      s.storageHealth(r),
		"modelruntime": s.modelRuntimeHealth(r, "health.detailed"),
		"gateway":      s.gatewayHealth(),
		"hostbridge":   s.hostBridgeHealth(),
		"forgekshadow": s.forgeKShadowHealth(),
		"dream":        s.dreamHealth(),
		"autonomy":     s.autonomyHealth(),
	}
	ok := detailedHealthOK(services)
	healthState := "ok"
	if !ok {
		healthState = "degraded"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          ok,
		"service":     "forge-core",
		"healthState": healthState,
		"services":    services,
	})
}

func (s *Server) storageHealth(r *http.Request) map[string]any {
	if s == nil || s.st == nil || s.st.DB == nil {
		return map[string]any{
			"ok":        false,
			"available": false,
			"status":    "unavailable",
		}
	}
	if err := s.st.DB.PingContext(r.Context()); err != nil {
		return map[string]any{
			"ok":        false,
			"available": true,
			"status":    "degraded",
			"reason":    "ping_failed",
		}
	}
	return map[string]any{
		"ok":             true,
		"available":      true,
		"status":         "ok",
		"truthAuthority": true,
	}
}

func (s *Server) modelRuntimeHealth(r *http.Request, source string) map[string]any {
	if s == nil || s.modelRuntime == nil {
		return map[string]any{
			"ok":        false,
			"available": false,
			"status":    "unavailable",
		}
	}
	meta := modelRuntimeMetaFromRequestAudit(requestAuditMetaForBackup(r, "", "", "", source))
	health, err := s.modelRuntime.Health(r.Context(), meta)
	if err != nil {
		return map[string]any{
			"ok":        false,
			"available": true,
			"status":    "degraded",
			"reason":    "health_probe_failed",
		}
	}
	status := health.Status
	if status == "" {
		if health.OK {
			status = "ok"
		} else {
			status = "degraded"
		}
	}
	return map[string]any{
		"ok":              health.OK,
		"available":       true,
		"status":          status,
		"backend":         health.Backend,
		"runtimeEnabled":  health.RuntimeEnabled,
		"gpuAware":        health.GPUAware,
		"degradedReasons": append([]string(nil), health.DegradedReasons...),
		"policyWarnings":  append([]string(nil), health.PolicyWarnings...),
	}
}

func (s *Server) gatewayHealth() map[string]any {
	if s == nil || s.gateway == nil {
		return map[string]any{
			"ok":        false,
			"available": false,
			"status":    "unavailable",
		}
	}
	out := map[string]any{
		"ok":        true,
		"available": true,
		"status":    "ok",
	}
	if !s.capStoreOK {
		out["status"] = "degraded"
		out["ok"] = false
		out["reason"] = "capability_override_store_unavailable"
	}
	return out
}

func (s *Server) hostBridgeHealth() map[string]any {
	return map[string]any{
		"ok":                true,
		"available":         true,
		"status":            "read_only_available",
		"mutationAuthority": false,
	}
}

func (s *Server) forgeKShadowHealth() map[string]any {
	enabled := s != nil && s.forgeKShadow != nil && s.forgeKShadow.Enabled()
	status := "disabled"
	if enabled {
		status = "enabled"
	}
	return map[string]any{
		"ok":            true,
		"available":     enabled,
		"status":        status,
		"liveAuthority": false,
	}
}

func (s *Server) dreamHealth() map[string]any {
	available := s != nil && s.dream != nil
	status := "not_configured"
	if available {
		status = "configured"
	}
	return map[string]any{
		"ok":           true,
		"available":    available,
		"status":       status,
		"proposalOnly": true,
	}
}

func (s *Server) autonomyHealth() map[string]any {
	if s == nil || s.autonomy == nil {
		return map[string]any{
			"ok":        true,
			"available": false,
			"status":    "not_configured",
		}
	}
	status := s.autonomy.Status()
	state := "configured"
	if status.SweepActive {
		state = "sweep_active"
	} else if status.Active {
		state = "dream_active"
	}
	return map[string]any{
		"ok":          true,
		"available":   true,
		"status":      state,
		"active":      status.Active,
		"sweepActive": status.SweepActive,
	}
}

func detailedHealthOK(services map[string]map[string]any) bool {
	for _, service := range services {
		if ok, present := service["ok"].(bool); present && !ok {
			return false
		}
	}
	return true
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
