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
