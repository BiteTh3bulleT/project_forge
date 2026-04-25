package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestProviderCapabilitiesAndTEIDegradedKeepCoreHealthy(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cfg := config.Config{
		DataDir:                    t.TempDir(),
		WorkspaceDir:               t.TempDir(),
		EmbeddingProvider:          "tei",
		EmbeddingTEIEndpoint:       "http://127.0.0.1:1",
		EmbeddingTEITimeoutMs:      10,
		NVIDIADCGMEnabled:          true,
		NVIDIADCGMEndpoint:         "http://127.0.0.1:1/metrics",
		NVIDIADCGMTimeoutMs:        10,
		IntelLevelZeroEnabled:      true,
		IntelLevelZeroZEInfoPath:   t.TempDir() + "/missing-ze-info",
		IntelGPUTelemetryTimeoutMs: 10,
	}
	srv := NewServer(st, cfg)
	t.Cleanup(srv.ShutdownWatch)

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(healthRR, healthReq)
	if healthRR.Code != http.StatusOK {
		t.Fatalf("expected core health OK with degraded providers, status=%d body=%s", healthRR.Code, healthRR.Body.String())
	}
	var health map[string]any
	if err := json.Unmarshal(healthRR.Body.Bytes(), &health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if health["ok"] != true {
		t.Fatalf("expected core ok=true, got %+v", health)
	}
	embeddingsHealth, ok := health["embeddings"].(map[string]any)
	if !ok {
		t.Fatalf("expected embeddings health in core health: %+v", health)
	}
	if providerHealth, ok := embeddingsHealth["health"].(map[string]any); !ok || providerHealth["state"] != "degraded" {
		t.Fatalf("expected current TEI embedding backend degraded while core stays healthy, got %+v", embeddingsHealth["health"])
	}

	req := httptest.NewRequest(http.MethodGet, "/api/providers/capabilities", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("providers status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Providers []providerCapability `json:"providers"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode providers: %v", err)
	}
	if !hasProvider(payload.Providers, "embeddings.tei") || !hasProvider(payload.Providers, "telemetry.nvidia_dcgm") || !hasProvider(payload.Providers, "telemetry.intel_level_zero") {
		t.Fatalf("expected TEI and DCGM capabilities, got %+v", payload.Providers)
	}
}

func hasProvider(providers []providerCapability, id string) bool {
	for _, provider := range providers {
		if provider.ID == id {
			return true
		}
	}
	return false
}
