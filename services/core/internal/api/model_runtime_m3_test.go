package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestModelRuntimeManagementEndpoints(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	srv.modelRuntime = fake

	importBody := governanceBody(map[string]any{
		"path":        "/tmp/local.gguf",
		"id":          "local-model",
		"displayName": "Local Model",
	})
	approvalID := requestAndApproveModelGovernance(t, srv, "/forge/models/import", importBody)
	importBody["approvalId"] = fmt.Sprintf("%d", approvalID)
	importRR := postModelRuntimeJSON(t, srv, "/forge/models/import", importBody)
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
		body       map[string]any
		wantStatus string
	}{
		{path: "/forge/models/local-model/verify", body: governanceBody(nil), wantStatus: "verified"},
		{path: "/forge/models/local-model/disable", body: governanceBody(nil), wantStatus: "disabled"},
		{path: "/forge/models/local-model/enable", body: governanceBody(nil), wantStatus: "verified"},
	} {
		rr := postModelRuntimeJSON(t, srv, tc.path, tc.body)
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

	for _, path := range []string{"/forge/model-runtime/usage", "/forge/model-runtime/backends"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, strings.TrimSpace(rr.Body.String()))
		}
	}

	for _, path := range []string{"/forge/models/local-model/archive", "/forge/models/local-model/remove"} {
		body := governanceBody(nil)
		approvalID := requestAndApproveModelGovernance(t, srv, path, body)
		body["approvalId"] = fmt.Sprintf("%d", approvalID)
		rr := postModelRuntimeJSON(t, srv, path, body)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, strings.TrimSpace(rr.Body.String()))
		}
	}
}

func TestModelRuntimeGovernanceReadOnlyAndMediumPolicy(t *testing.T) {
	t.Parallel()

	srv, st := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	srv.modelRuntime = fake

	for _, path := range []string{"/forge/models", "/forge/models/mistral-7b-instruct", "/forge/models/mistral-7b-instruct/compatibility"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, strings.TrimSpace(rr.Body.String()))
		}
	}

	denied := postModelRuntimeJSON(t, srv, "/forge/models/scan", map[string]any{
		"actor":         "operator-a",
		"source":        "test",
		"correlationId": "corr-model-denied",
	})
	if denied.Code != http.StatusForbidden {
		t.Fatalf("scan without capability status=%d body=%s", denied.Code, strings.TrimSpace(denied.Body.String()))
	}
	assertModelRuntimeErrorCode(t, denied, "MODEL_GOVERNANCE_CAPABILITY_REQUIRED")
	assertModelRuntimeAuditOutcome(t, st, "corr-model-denied", "model.scan", "denied")

	allowed := postModelRuntimeJSON(t, srv, "/forge/models/scan", governanceBody(map[string]any{"correlationId": "corr-model-scan"}))
	if allowed.Code != http.StatusOK {
		t.Fatalf("scan with capability status=%d body=%s", allowed.Code, strings.TrimSpace(allowed.Body.String()))
	}
	if fake.lastControl.Metadata["capabilityId"] != modelManagementCapabilityID {
		t.Fatalf("capability metadata not forwarded: %#v", fake.lastControl.Metadata)
	}
	assertModelRuntimeAuditOutcome(t, st, "corr-model-scan", "model.scan", "authorized")
}

func TestModelRuntimeEndpointsRejectOversizeRequestBodies(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	srv.modelRuntime = newFakeModelRuntime()
	oversizeBody := `{"prompt":"` + strings.Repeat("a", modelRuntimeRequestBodyLimit+1) + `"}`

	for _, path := range []string{
		"/forge/models/import",
		"/forge/models/mistral-7b-instruct/chat",
		"/v1/chat/completions",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(oversizeBody))
		rr := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("%s expected 413 for oversize body, got %d body=%s", path, rr.Code, strings.TrimSpace(rr.Body.String()))
		}
		assertModelRuntimeErrorCode(t, rr, "REQUEST_BODY_TOO_LARGE")
	}
}

func TestModelRuntimeHighRiskApprovalAndDryRun(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	srv.modelRuntime = fake

	dryRunBody := governanceBody(map[string]any{
		"path":   "/tmp/dry-run.gguf",
		"id":     "dry-run-model",
		"dryRun": true,
	})
	dryRunRR := postModelRuntimeJSON(t, srv, "/forge/models/import", dryRunBody)
	if dryRunRR.Code != http.StatusOK {
		t.Fatalf("dry-run import status=%d body=%s", dryRunRR.Code, strings.TrimSpace(dryRunRR.Body.String()))
	}
	if fake.importCalls != 0 {
		t.Fatalf("dry-run import mutated runtime, importCalls=%d", fake.importCalls)
	}
	assertGovernanceRequiresApproval(t, dryRunRR)

	importBody := governanceBody(map[string]any{
		"path": "/tmp/high-risk.gguf",
		"id":   "high-risk-model",
	})
	needsApproval := postModelRuntimeJSON(t, srv, "/forge/models/import", importBody)
	if needsApproval.Code != http.StatusAccepted {
		t.Fatalf("import without approval status=%d body=%s", needsApproval.Code, strings.TrimSpace(needsApproval.Body.String()))
	}
	if fake.importCalls != 0 {
		t.Fatalf("approval request mutated runtime, importCalls=%d", fake.importCalls)
	}
	approvalID := approvalIDFromGovernance(t, needsApproval)
	approveModelGovernance(t, srv, approvalID)

	importBody["approvalId"] = fmt.Sprintf("%d", approvalID)
	importRR := postModelRuntimeJSON(t, srv, "/forge/models/import", importBody)
	if importRR.Code != http.StatusOK {
		t.Fatalf("approved import status=%d body=%s", importRR.Code, strings.TrimSpace(importRR.Body.String()))
	}
	if fake.importCalls != 1 {
		t.Fatalf("approved import calls=%d want 1", fake.importCalls)
	}
	if fake.lastImport.Metadata["approvalId"] != fmt.Sprintf("%d", approvalID) {
		t.Fatalf("approval metadata not forwarded: %#v", fake.lastImport.Metadata)
	}
}

func TestModelRuntimeApprovalFingerprintRejectsReuse(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name         string
		approvedURL  string
		approvedBody map[string]any
		reuseURL     string
		reuseBody    map[string]any
	}{
		{
			name:         "tool",
			approvedURL:  "/forge/models/import",
			approvedBody: governanceBody(map[string]any{"path": "/tmp/tool-a.gguf", "id": "tool-a"}),
			reuseURL:     "/forge/models/mistral-7b-instruct/archive",
			reuseBody:    governanceBody(nil),
		},
		{
			name:         "path",
			approvedURL:  "/forge/models/import",
			approvedBody: governanceBody(map[string]any{"path": "/tmp/path-a.gguf", "id": "path-model"}),
			reuseURL:     "/forge/models/import",
			reuseBody:    governanceBody(map[string]any{"path": "/tmp/path-b.gguf", "id": "path-model"}),
		},
		{
			name:         "actor",
			approvedURL:  "/forge/models/import",
			approvedBody: governanceBody(map[string]any{"path": "/tmp/actor-a.gguf", "id": "actor-model"}),
			reuseURL:     "/forge/models/import",
			reuseBody:    governanceBody(map[string]any{"path": "/tmp/actor-a.gguf", "id": "actor-model", "actor": "operator-b"}),
		},
		{
			name:         "lane-workspace",
			approvedURL:  "/forge/models/import",
			approvedBody: governanceBody(map[string]any{"path": "/tmp/lane-a.gguf", "id": "lane-model", "laneId": "lane-a", "workspaceId": "ws-a"}),
			reuseURL:     "/forge/models/import",
			reuseBody:    governanceBody(map[string]any{"path": "/tmp/lane-a.gguf", "id": "lane-model", "laneId": "lane-b", "workspaceId": "ws-b"}),
		},
		{
			name:         "risk-write-intent",
			approvedURL:  "/forge/models/import",
			approvedBody: governanceBody(map[string]any{"path": "/tmp/risk-a.gguf", "id": "risk-model"}),
			reuseURL:     "/forge/models/import",
			reuseBody:    governanceBody(map[string]any{"path": "/tmp/risk-a.gguf", "id": "risk-model", "preferred": true}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv, _ := newModelRuntimeHarness(t)
			srv.modelRuntime = newFakeModelRuntime()

			approvalID := requestAndApproveModelGovernance(t, srv, tc.approvedURL, tc.approvedBody)
			tc.reuseBody["approvalId"] = fmt.Sprintf("%d", approvalID)
			rr := postModelRuntimeJSON(t, srv, tc.reuseURL, tc.reuseBody)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("reuse status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
			}
			assertModelRuntimeErrorCode(t, rr, "MODEL_GOVERNANCE_APPROVAL_FINGERPRINT_MISMATCH")
		})
	}
}

func TestModelRuntimeCloudProviderEnablementRequiresGovernance(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	fake.models["cloud-model"] = ModelRuntimeModel{
		ID:      "cloud-model",
		Backend: "openai_compat",
		Status:  "available",
		Metadata: map[string]any{
			"provider": "openai_compat",
		},
	}
	srv.modelRuntime = fake

	denied := postModelRuntimeJSON(t, srv, "/forge/models/cloud-model/enable", map[string]any{
		"actor":  "operator-a",
		"source": "test",
	})
	if denied.Code != http.StatusForbidden {
		t.Fatalf("cloud enable without capability status=%d body=%s", denied.Code, strings.TrimSpace(denied.Body.String()))
	}
	assertModelRuntimeErrorCode(t, denied, "MODEL_GOVERNANCE_CAPABILITY_REQUIRED")

	body := governanceBody(map[string]any{"correlationId": "corr-cloud-enable"})
	missingConfig := postModelRuntimeJSON(t, srv, "/forge/models/cloud-model/enable", body)
	if missingConfig.Code != http.StatusForbidden {
		t.Fatalf("cloud enable without provider config status=%d body=%s", missingConfig.Code, strings.TrimSpace(missingConfig.Body.String()))
	}
	assertModelRuntimeErrorCode(t, missingConfig, "MODEL_GOVERNANCE_PROVIDER_CONFIG_REQUIRED")

	srv.cfg.ModelOpenAICompatEndpoint = "http://127.0.0.1:11434"
	needsApproval := postModelRuntimeJSON(t, srv, "/forge/models/cloud-model/enable", body)
	if needsApproval.Code != http.StatusAccepted {
		t.Fatalf("cloud enable without approval status=%d body=%s", needsApproval.Code, strings.TrimSpace(needsApproval.Body.String()))
	}
	approvalID := approvalIDFromGovernance(t, needsApproval)
	approveModelGovernance(t, srv, approvalID)
	body["approvalId"] = fmt.Sprintf("%d", approvalID)
	approved := postModelRuntimeJSON(t, srv, "/forge/models/cloud-model/enable", body)
	if approved.Code != http.StatusOK {
		t.Fatalf("approved cloud enable status=%d body=%s", approved.Code, strings.TrimSpace(approved.Body.String()))
	}
}

func TestModelRuntimeDegradedHealthDoesNotBlockReadPaths(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	fake.healthErr = &modelRuntimeError{status: http.StatusServiceUnavailable, code: "MODEL_RUNTIME_DEGRADED", message: "degraded"}
	srv.modelRuntime = fake

	req := httptest.NewRequest(http.MethodGet, "/forge/models", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list during degraded health status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func governanceBody(extra map[string]any) map[string]any {
	body := map[string]any{
		"actor":         "operator-a",
		"source":        "test",
		"capabilityId":  modelManagementCapabilityID,
		"workspaceId":   "workspace-a",
		"laneId":        "lane-a",
		"correlationId": "corr-model-governance",
		"traceId":       "trace-model-governance",
	}
	for key, value := range extra {
		body[key] = value
	}
	return body
}

func postModelRuntimeJSON(t *testing.T, srv *Server, path string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	return rr
}

func requestAndApproveModelGovernance(t *testing.T, srv *Server, path string, body map[string]any) int64 {
	t.Helper()
	rr := postModelRuntimeJSON(t, srv, path, body)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("approval request status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	approvalID := approvalIDFromGovernance(t, rr)
	approveModelGovernance(t, srv, approvalID)
	return approvalID
}

func approvalIDFromGovernance(t *testing.T, rr *httptest.ResponseRecorder) int64 {
	t.Helper()
	var payload struct {
		Governance struct {
			ApprovalRequestID int64 `json:"approvalRequestId"`
			RequiresApproval  bool  `json:"requiresApproval"`
		} `json:"governance"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode governance response: %v body=%s", err, rr.Body.String())
	}
	if !payload.Governance.RequiresApproval || payload.Governance.ApprovalRequestID <= 0 {
		t.Fatalf("expected approval request in governance response: %#v body=%s", payload.Governance, rr.Body.String())
	}
	return payload.Governance.ApprovalRequestID
}

func assertGovernanceRequiresApproval(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	var payload struct {
		Governance struct {
			RequiresApproval bool   `json:"requiresApproval"`
			RiskClass        string `json:"riskClass"`
			DryRun           bool   `json:"dryRun"`
		} `json:"governance"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode governance response: %v body=%s", err, rr.Body.String())
	}
	if !payload.Governance.RequiresApproval || payload.Governance.RiskClass != "high" || !payload.Governance.DryRun {
		t.Fatalf("unexpected governance dry-run response: %#v", payload.Governance)
	}
}

func approveModelGovernance(t *testing.T, srv *Server, approvalID int64) {
	t.Helper()
	if _, err := srv.approvals.Decide(context.Background(), approvalID, "operator-a", "approved", "test approval"); err != nil {
		t.Fatalf("approve request %d: %v", approvalID, err)
	}
}

func assertModelRuntimeErrorCode(t *testing.T, rr *httptest.ResponseRecorder, want string) {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v body=%s", err, rr.Body.String())
	}
	if payload.Error.Code != want {
		t.Fatalf("error code=%q want %q body=%s", payload.Error.Code, want, rr.Body.String())
	}
}

func assertModelRuntimeAuditOutcome(t *testing.T, st *store.Store, correlationID, action, outcome string) {
	t.Helper()
	var count int
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT COUNT(1) FROM audit_records
WHERE correlation_id = ? AND action = ? AND outcome = ?`, correlationID, action, outcome).Scan(&count); err != nil {
		t.Fatalf("query model runtime audit: %v", err)
	}
	if count < 1 {
		t.Fatalf("expected audit action=%s correlation=%s outcome=%s", action, correlationID, outcome)
	}
}
