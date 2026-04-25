package api

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/embeddings"
)

type providerCapability struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	LocalCloud         string         `json:"localCloud"`
	GPURequired        bool           `json:"gpuRequired"`
	SupportsEmbeddings bool           `json:"supportsEmbeddings"`
	SupportsGeneration bool           `json:"supportsGeneration"`
	SupportsLoRA       bool           `json:"supportsLora"`
	SupportsGuardrails bool           `json:"supportsGuardrails"`
	SupportsStreaming  bool           `json:"supportsStreaming"`
	HealthState        string         `json:"healthState"`
	Healthy            bool           `json:"healthy"`
	Detail             string         `json:"detail,omitempty"`
	CostClass          string         `json:"costClass"`
	LatencyClass       string         `json:"latencyClass"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

func ensureEmbeddingProviderConfig(ctx context.Context, db *sql.DB, cfg config.Config) {
	if db == nil {
		return
	}
	if strings.TrimSpace(cfg.EmbeddingProvider) != "" {
		_ = upsertSetting(ctx, db, "embedding_provider", strings.TrimSpace(cfg.EmbeddingProvider))
	}
	if strings.TrimSpace(cfg.EmbeddingModel) != "" {
		_ = upsertSetting(ctx, db, "embedding_model", strings.TrimSpace(cfg.EmbeddingModel))
	}
	if cfg.EmbeddingDims > 0 {
		_ = upsertSetting(ctx, db, "embedding_dims", strconvItoa(cfg.EmbeddingDims))
	}
	if strings.TrimSpace(cfg.EmbeddingTEIEndpoint) != "" {
		_ = upsertSetting(ctx, db, "embedding_tei_endpoint", strings.TrimSpace(cfg.EmbeddingTEIEndpoint))
	}
	if strings.TrimSpace(cfg.EmbeddingTEIAPIKey) != "" {
		_ = upsertSetting(ctx, db, "embedding_tei_api_key", strings.TrimSpace(cfg.EmbeddingTEIAPIKey))
	}
	if cfg.EmbeddingTEITimeoutMs > 0 {
		_ = upsertSetting(ctx, db, "embedding_tei_timeout_ms", strconvItoa(cfg.EmbeddingTEITimeoutMs))
	}
}

func (s *Server) handleProviderCapabilities(w http.ResponseWriter, r *http.Request) {
	caps := []providerCapability{}
	if s.embeddings != nil {
		for _, provider := range []string{embeddings.ProviderLocalHash, embeddings.ProviderOllama, embeddings.ProviderTEI} {
			h := s.embeddings.ProviderHealth(r.Context(), provider, "")
			caps = append(caps, providerCapability{
				ID:                 "embeddings." + provider,
				Name:               provider,
				LocalCloud:         h.LocalCloud,
				GPURequired:        h.GPURequired,
				SupportsEmbeddings: h.SupportsEmbeddings,
				SupportsGeneration: h.SupportsGeneration,
				SupportsLoRA:       h.SupportsLoRA,
				SupportsGuardrails: h.SupportsGuardrails,
				SupportsStreaming:  h.SupportsStreaming,
				HealthState:        h.State,
				Healthy:            h.Healthy,
				Detail:             h.Detail,
				CostClass:          h.CostClass,
				LatencyClass:       h.LatencyClass,
				Metadata:           h.Metadata,
			})
		}
	}
	if s.gpuTelemetry != nil {
		snap := s.gpuTelemetry.Snapshot(r.Context())
		caps = append(caps, providerCapability{
			ID:                 "telemetry.nvidia_dcgm",
			Name:               "NVIDIA DCGM telemetry",
			LocalCloud:         "local",
			GPURequired:        false,
			SupportsEmbeddings: false,
			SupportsGeneration: false,
			SupportsLoRA:       false,
			SupportsGuardrails: false,
			SupportsStreaming:  false,
			HealthState:        snap.State,
			Healthy:            snap.Healthy,
			Detail:             snap.Detail,
			CostClass:          "free",
			LatencyClass:       "low",
			Metadata: map[string]any{
				"enabled":                 snap.Enabled,
				"available":               snap.Available,
				"memoryPressure":          snap.MemoryPressure,
				"memoryPressureThreshold": snap.MemoryPressureThreshold,
				"backgroundAdmissionOk":   snap.BackgroundAdmissionOK,
			},
		})
	}
	if s.intelTelemetry != nil {
		snap := s.intelTelemetry.Snapshot(r.Context())
		caps = append(caps, providerCapability{
			ID:                 "telemetry.intel_level_zero",
			Name:               "Intel Level Zero telemetry",
			LocalCloud:         "local",
			GPURequired:        false,
			SupportsEmbeddings: false,
			SupportsGeneration: false,
			SupportsLoRA:       false,
			SupportsGuardrails: false,
			SupportsStreaming:  false,
			HealthState:        snap.State,
			Healthy:            snap.Healthy,
			Detail:             snap.Detail,
			CostClass:          "free",
			LatencyClass:       "low",
			Metadata: map[string]any{
				"enabled":               snap.Enabled,
				"available":             snap.Available,
				"deviceCount":           len(snap.Devices),
				"backgroundAdmissionOk": snap.BackgroundAdmissionOK,
				"warnings":              snap.Warnings,
			},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": caps})
}

func strconvItoa(v int) string {
	if v == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	n := v
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
