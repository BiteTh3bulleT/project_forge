package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModelRuntimeManagementEndpoints(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	srv.modelRuntime = fake

	importReq := httptest.NewRequest(http.MethodPost, "/forge/models/import", bytes.NewReader([]byte(`{"path":"/tmp/local.gguf","id":"local-model","displayName":"Local Model"}`)))
	importRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(importRR, importReq)
	if importRR.Code != http.StatusOK {
		t.Fatalf("import status=%d body=%s", importRR.Code, strings.TrimSpace(importRR.Body.String()))
	}
	var importPayload struct {
		Result ModelRuntimeImportResult `json:"result"`
	}
	if err := json.Unmarshal(importRR.Body.Bytes(), &importPayload); err != nil {
		t.Fatalf("decode import response: %v body=%s", err, importRR.Body.String())
	}
	if importPayload.Result.Model.ID != "local-model" || importPayload.Result.ManagedPath == "" {
		t.Fatalf("unexpected import payload: %#v", importPayload.Result)
	}

	for _, tc := range []struct {
		path       string
		wantStatus string
	}{
		{path: "/forge/models/local-model/verify", wantStatus: "verified"},
		{path: "/forge/models/local-model/disable", wantStatus: "disabled"},
		{path: "/forge/models/local-model/enable", wantStatus: "verified"},
		{path: "/forge/models/local-model/archive", wantStatus: "archived"},
	} {
		req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader([]byte(`{}`)))
		rr := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", tc.path, rr.Code, strings.TrimSpace(rr.Body.String()))
		}
		var payload struct {
			Model ModelRuntimeModel `json:"model"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode %s response: %v body=%s", tc.path, err, rr.Body.String())
		}
		if payload.Model.Status != tc.wantStatus {
			t.Fatalf("%s status=%q want %q", tc.path, payload.Model.Status, tc.wantStatus)
		}
	}

	compatReq := httptest.NewRequest(http.MethodGet, "/forge/models/local-model/compatibility", nil)
	compatRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(compatRR, compatReq)
	if compatRR.Code != http.StatusOK {
		t.Fatalf("compatibility status=%d body=%s", compatRR.Code, strings.TrimSpace(compatRR.Body.String()))
	}
	var compatPayload struct {
		Compatibility ModelRuntimeCompatibility `json:"compatibility"`
	}
	if err := json.Unmarshal(compatRR.Body.Bytes(), &compatPayload); err != nil {
		t.Fatalf("decode compatibility response: %v body=%s", err, compatRR.Body.String())
	}
	if compatPayload.Compatibility.ModelID != "local-model" {
		t.Fatalf("unexpected compatibility payload: %#v", compatPayload.Compatibility)
	}

	usageReq := httptest.NewRequest(http.MethodGet, "/forge/model-runtime/usage", nil)
	usageRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(usageRR, usageReq)
	if usageRR.Code != http.StatusOK {
		t.Fatalf("usage status=%d body=%s", usageRR.Code, strings.TrimSpace(usageRR.Body.String()))
	}
	backendsReq := httptest.NewRequest(http.MethodGet, "/forge/model-runtime/backends", nil)
	backendsRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(backendsRR, backendsReq)
	if backendsRR.Code != http.StatusOK {
		t.Fatalf("backends status=%d body=%s", backendsRR.Code, strings.TrimSpace(backendsRR.Body.String()))
	}

	removeReq := httptest.NewRequest(http.MethodPost, "/forge/models/local-model/remove", bytes.NewReader([]byte(`{}`)))
	removeRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(removeRR, removeReq)
	if removeRR.Code != http.StatusOK {
		t.Fatalf("remove status=%d body=%s", removeRR.Code, strings.TrimSpace(removeRR.Body.String()))
	}
}
