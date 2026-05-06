package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestPatchSettingsPersistsDreamModeSettings(t *testing.T) {
	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: workspaceDir})
	t.Cleanup(func() { srv.ShutdownWatch() })

	body := []byte(`{"dreamMode":{"enabled":false,"defaultDryRun":true,"mode":"nap","windowHours":24,"maxCandidates":"12","allowLongTermPromotion":true,"requireOperatorReviewForLongTerm":false,"allowCommits":true}}`)
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader(body))
	patchRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("patch settings status=%d body=%s", patchRR.Code, patchRR.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	getRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get settings status=%d body=%s", getRR.Code, getRR.Body.String())
	}

	var payload struct {
		DreamMode map[string]any `json:"dreamMode"`
	}
	if err := json.NewDecoder(getRR.Body).Decode(&payload); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if payload.DreamMode["enabled"] != false ||
		payload.DreamMode["defaultDryRun"] != true ||
		payload.DreamMode["mode"] != "nap" ||
		payload.DreamMode["windowHours"] != "24" ||
		payload.DreamMode["maxCandidates"] != "12" ||
		payload.DreamMode["allowLongTermPromotion"] != true ||
		payload.DreamMode["requireOperatorReviewForLongTerm"] != false ||
		payload.DreamMode["allowCommits"] != true {
		t.Fatalf("unexpected dreamMode settings: %#v", payload.DreamMode)
	}
}

func TestPatchSettingsPersistsRuntimeControls(t *testing.T) {
	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{
		DataDir:               dataDir,
		WorkspaceDir:          workspaceDir,
		EnableModelRuntime:    true,
		ModelRequestTimeoutMs: 1000,
	})
	t.Cleanup(func() { srv.ShutdownWatch() })

	body := []byte(`{"runtimeControls":{"gpuEnabled":true,"nvidiaDcgmEnabled":true,"intelLevelZeroEnabled":false,"allowOllamaCloudModels":true}}`)
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader(body))
	patchRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("patch settings status=%d body=%s", patchRR.Code, patchRR.Body.String())
	}

	var payload struct {
		RuntimeControls map[string]any `json:"runtimeControls"`
	}
	if err := json.NewDecoder(patchRR.Body).Decode(&payload); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if payload.RuntimeControls["gpuEnabled"] != true ||
		payload.RuntimeControls["nvidiaDcgmEnabled"] != true ||
		payload.RuntimeControls["intelLevelZeroEnabled"] != false ||
		payload.RuntimeControls["allowOllamaCloudModels"] != true ||
		payload.RuntimeControls["effectiveGpuEnabled"] != true {
		t.Fatalf("unexpected runtimeControls settings: %#v", payload.RuntimeControls)
	}
	if !srv.cfg.GPUEnabled || !srv.cfg.NVIDIADCGMEnabled || srv.cfg.IntelLevelZeroEnabled || !srv.cfg.ModelRuntimeAllowOllamaCloudModels {
		t.Fatalf("runtime controls were not applied to server config: %#v", srv.cfg)
	}
}

func TestPatchSettingsPersistsAndAppliesShadowMode(t *testing.T) {
	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: workspaceDir})
	t.Cleanup(func() { srv.ShutdownWatch() })
	if srv.forgeKShadow != nil {
		t.Fatalf("shadow observer should be disabled by default")
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader([]byte(`{"shadowMode":{"enabled":true}}`)))
	patchRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("patch shadow settings status=%d body=%s", patchRR.Code, patchRR.Body.String())
	}
	if !srv.cfg.ForgeKShadowModeEnabled || srv.forgeKShadow == nil || !srv.forgeKShadow.Enabled() {
		t.Fatalf("shadow mode was not applied to running server: cfg=%#v observer=%#v", srv.cfg, srv.forgeKShadow)
	}

	var enabledPayload struct {
		ShadowMode map[string]any `json:"shadowMode"`
	}
	if err := json.NewDecoder(patchRR.Body).Decode(&enabledPayload); err != nil {
		t.Fatalf("decode enabled settings: %v", err)
	}
	if enabledPayload.ShadowMode["enabled"] != true {
		t.Fatalf("expected enabled shadow setting, got %#v", enabledPayload.ShadowMode)
	}

	restarted := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: workspaceDir})
	t.Cleanup(func() { restarted.ShutdownWatch() })
	if !restarted.cfg.ForgeKShadowModeEnabled || restarted.forgeKShadow == nil || !restarted.forgeKShadow.Enabled() {
		t.Fatalf("persisted shadow mode was not applied at server startup")
	}

	disableReq := httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader([]byte(`{"shadowMode":{"enabled":false}}`)))
	disableRR := httptest.NewRecorder()
	restarted.Handler().ServeHTTP(disableRR, disableReq)
	if disableRR.Code != http.StatusOK {
		t.Fatalf("disable shadow settings status=%d body=%s", disableRR.Code, disableRR.Body.String())
	}
	if restarted.cfg.ForgeKShadowModeEnabled || restarted.forgeKShadow != nil {
		t.Fatalf("shadow mode was not disabled on running server: cfg=%#v observer=%#v", restarted.cfg, restarted.forgeKShadow)
	}
}
