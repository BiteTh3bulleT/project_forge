package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestHandleGatewayCapabilityStatusUpdateRejectsUnknownStatus(t *testing.T) {
	t.Parallel()
	srv, _ := newGatewayCapabilityStatusHarness(t)

	req := withRouteParam(
		httptest.NewRequest(
			http.MethodPatch,
			"/api/gateway/capabilities/filesystem.read_file/status",
			bytes.NewBufferString(`{"status":"not_real"}`),
		),
		"id",
		"filesystem.read_file",
	)
	rr := httptest.NewRecorder()
	srv.handleGatewayCapabilityStatusUpdate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for unknown status, got %d", rr.Code)
	}
	if !strings.Contains(strings.ToLower(rr.Body.String()), "unknown capability status") {
		t.Fatalf("expected unknown status error message, got %q", rr.Body.String())
	}
}

func TestHandleGatewayCapabilityStatusUpdateRequiresReasonForDeferredDisabledStubbedDeprecated(t *testing.T) {
	t.Parallel()
	srv, _ := newGatewayCapabilityStatusHarness(t)

	for _, status := range []string{"approval_only", "deferred", "disabled", "stubbed", "deprecated"} {
		capID := "filesystem.read_file"
		req := withRouteParam(
			httptest.NewRequest(
				http.MethodPatch,
				"/api/gateway/capabilities/"+capID+"/status",
				bytes.NewBufferString(`{"status":"`+status+`","actor":"tester","actorKind":"test"}`),
			),
			"id",
			capID,
		)
		rr := httptest.NewRecorder()
		srv.handleGatewayCapabilityStatusUpdate(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request for status %q without reason, got %d", status, rr.Code)
		}
		if !strings.Contains(strings.ToLower(rr.Body.String()), "reason is required") {
			t.Fatalf("expected reason-required message for status %q, got %q", status, rr.Body.String())
		}
	}
}

func TestHandleGatewayCapabilityStatusUpdateRequiresReasonForMutatingActiveTransition(t *testing.T) {
	t.Parallel()
	srv, _ := newGatewayCapabilityStatusHarness(t)
	disableReq := withRouteParam(
		httptest.NewRequest(
			http.MethodPatch,
			"/api/gateway/capabilities/filesystem.read_file/status",
			bytes.NewBufferString(`{"status":"disabled","reason":"prepare active reason test","actor":"tester","actorKind":"test"}`),
		),
		"id",
		"filesystem.read_file",
	)
	disableRR := httptest.NewRecorder()
	srv.handleGatewayCapabilityStatusUpdate(disableRR, disableReq)
	if disableRR.Code != http.StatusOK {
		t.Fatalf("expected setup disable to succeed, got %d body=%s", disableRR.Code, disableRR.Body.String())
	}
	activeReq := withRouteParam(
		httptest.NewRequest(
			http.MethodPatch,
			"/api/gateway/capabilities/filesystem.read_file/status",
			bytes.NewBufferString(`{"status":"active","actor":"tester","actorKind":"test"}`),
		),
		"id",
		"filesystem.read_file",
	)
	activeRR := httptest.NewRecorder()
	srv.handleGatewayCapabilityStatusUpdate(activeRR, activeReq)
	if activeRR.Code != http.StatusBadRequest {
		t.Fatalf("expected active mutation without reason to fail, got %d body=%s", activeRR.Code, activeRR.Body.String())
	}
	if !strings.Contains(strings.ToLower(activeRR.Body.String()), "reason is required") {
		t.Fatalf("expected reason-required message, got %q", activeRR.Body.String())
	}
}

func TestHandleGatewayCapabilityStatusUpdateAllowsNoopActiveWithoutReason(t *testing.T) {
	t.Parallel()
	srv, _ := newGatewayCapabilityStatusHarness(t)

	req := withRouteParam(
		httptest.NewRequest(
			http.MethodPatch,
			"/api/gateway/capabilities/filesystem.read_file/status",
			bytes.NewBufferString(`{"status":"active"}`),
		),
		"id",
		"filesystem.read_file",
	)
	rr := httptest.NewRecorder()
	srv.handleGatewayCapabilityStatusUpdate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected active transition without reason to succeed, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGatewayCapabilityStatusUpdateUpdatesAndAudits(t *testing.T) {
	t.Parallel()
	srv, st := newGatewayCapabilityStatusHarness(t)

	req := withRouteParam(
		httptest.NewRequest(
			http.MethodPatch,
			"/api/gateway/capabilities/filesystem.read_file/status",
			bytes.NewBufferString(`{"status":"deferred","reason":"deferred until adapter readiness","actor":"operator-a","actorKind":"user","correlationId":"corr-cap-status-low","traceId":"trace-cap-status-low"}`),
		),
		"id",
		"filesystem.read_file",
	)
	rr := httptest.NewRecorder()
	srv.handleGatewayCapabilityStatusUpdate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status update ok, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Capability struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"capability"`
		PreviousStatus string `json:"previousStatus"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Capability.ID != "filesystem.read_file" {
		t.Fatalf("unexpected capability id %q", payload.Capability.ID)
	}
	if payload.Capability.Status != "deferred" {
		t.Fatalf("expected deferred status, got %q", payload.Capability.Status)
	}
	if payload.PreviousStatus == "" {
		t.Fatalf("expected previous status in response")
	}

	var auditCount int
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT COUNT(1) FROM audit_records
WHERE action = 'tool.capability.status.updated' AND subject_id = ?`,
		"filesystem.read_file",
	).Scan(&auditCount); err != nil {
		t.Fatalf("query audit count: %v", err)
	}
	if auditCount == 0 {
		t.Fatalf("expected tool capability status update audit record")
	}
	var reason, actor, actorKind, previousStatus, riskClass, transitionRisk, correlationID, traceID string
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT reason, actor, actor_kind, previous_status, risk_class, transition_risk, correlation_id, trace_id
FROM tool_capability_overrides WHERE capability_id = ?`,
		"filesystem.read_file",
	).Scan(&reason, &actor, &actorKind, &previousStatus, &riskClass, &transitionRisk, &correlationID, &traceID); err != nil {
		t.Fatalf("query override metadata: %v", err)
	}
	if reason != "deferred until adapter readiness" || actor != "operator-a" || actorKind != "user" {
		t.Fatalf("override actor/reason mismatch: reason=%q actor=%q actorKind=%q", reason, actor, actorKind)
	}
	if previousStatus == "" || riskClass == "" || transitionRisk == "" || correlationID != "corr-cap-status-low" || traceID != "trace-cap-status-low" {
		t.Fatalf("override governance metadata missing: prev=%q risk=%q transition=%q corr=%q trace=%q", previousStatus, riskClass, transitionRisk, correlationID, traceID)
	}
}

func TestHandleGatewayCapabilityStatusUpdateDangerousActivationRequiresApproval(t *testing.T) {
	t.Parallel()
	srv, _ := newGatewayCapabilityStatusHarness(t)

	req := withRouteParam(
		httptest.NewRequest(
			http.MethodPatch,
			"/api/gateway/capabilities/process.spawn_process/status",
			bytes.NewBufferString(`{"status":"active","reason":"temporary operator maintenance","actor":"operator-a","actorKind":"user","correlationId":"corr-cap-status-needs-approval"}`),
		),
		"id",
		"process.spawn_process",
	)
	rr := httptest.NewRecorder()
	srv.handleGatewayCapabilityStatusUpdate(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected accepted approval-required response, got %d body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		ApprovalRequired  bool   `json:"approvalRequired"`
		ApprovalRequestID *int64 `json:"approvalRequestId"`
		RiskClass         string `json:"riskClass"`
		RejectionReason   string `json:"rejectionReason"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode approval response: %v", err)
	}
	if !payload.ApprovalRequired || payload.ApprovalRequestID == nil || *payload.ApprovalRequestID <= 0 {
		t.Fatalf("expected approval request, got %#v", payload)
	}
	if payload.RiskClass != "high" || !strings.Contains(strings.ToLower(payload.RejectionReason), "approval required") {
		t.Fatalf("unexpected governance response: %#v", payload)
	}
	capability, ok := srv.gateway.Capability("process.spawn_process")
	if !ok {
		t.Fatalf("missing capability")
	}
	if capability.Status == "active" {
		t.Fatalf("dangerous capability became active before approval")
	}
}

func TestHandleGatewayCapabilityStatusUpdateDangerousActivationWithApprovalSucceeds(t *testing.T) {
	t.Parallel()
	srv, st := newGatewayCapabilityStatusHarness(t)
	body := `{"status":"active","reason":"approved maintenance window","actor":"operator-a","actorKind":"user","correlationId":"corr-cap-status-approved","traceId":"trace-cap-status-approved"}`
	first := withRouteParam(
		httptest.NewRequest(http.MethodPatch, "/api/gateway/capabilities/process.spawn_process/status", bytes.NewBufferString(body)),
		"id",
		"process.spawn_process",
	)
	firstRR := httptest.NewRecorder()
	srv.handleGatewayCapabilityStatusUpdate(firstRR, first)
	if firstRR.Code != http.StatusAccepted {
		t.Fatalf("expected approval request, got %d body=%s", firstRR.Code, firstRR.Body.String())
	}
	var firstPayload struct {
		ApprovalRequestID *int64 `json:"approvalRequestId"`
	}
	if err := json.Unmarshal(firstRR.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode approval request: %v", err)
	}
	if firstPayload.ApprovalRequestID == nil {
		t.Fatalf("missing approval request id")
	}
	if _, err := srv.approvals.Decide(context.Background(), *firstPayload.ApprovalRequestID, "operator-a", "approved", "approved capability status transition"); err != nil {
		t.Fatalf("approve capability status request: %v", err)
	}

	secondBody := `{"status":"active","reason":"approved maintenance window","actor":"operator-a","actorKind":"user","correlationId":"corr-cap-status-approved","traceId":"trace-cap-status-approved","approvalId":"` + strconv.FormatInt(*firstPayload.ApprovalRequestID, 10) + `"}`
	second := withRouteParam(
		httptest.NewRequest(http.MethodPatch, "/api/gateway/capabilities/process.spawn_process/status", bytes.NewBufferString(secondBody)),
		"id",
		"process.spawn_process",
	)
	secondRR := httptest.NewRecorder()
	srv.handleGatewayCapabilityStatusUpdate(secondRR, second)
	if secondRR.Code != http.StatusOK {
		t.Fatalf("expected approved update ok, got %d body=%s", secondRR.Code, secondRR.Body.String())
	}
	var capabilityStatus string
	if err := st.DB.QueryRowContext(context.Background(), `SELECT status FROM tool_capability_overrides WHERE capability_id = ?`, "process.spawn_process").Scan(&capabilityStatus); err != nil {
		t.Fatalf("query override status: %v", err)
	}
	if capabilityStatus != "active" {
		t.Fatalf("expected active override, got %q", capabilityStatus)
	}
}

func TestHandleGatewayCapabilityStatusUpdateDeniedApprovalHasDeterministicReason(t *testing.T) {
	t.Parallel()
	srv, _ := newGatewayCapabilityStatusHarness(t)
	req := withRouteParam(
		httptest.NewRequest(
			http.MethodPatch,
			"/api/gateway/capabilities/process.spawn_process/status",
			bytes.NewBufferString(`{"status":"active","reason":"bad approval","actor":"operator-a","actorKind":"user","approvalId":"999999"}`),
		),
		"id",
		"process.spawn_process",
	)
	rr := httptest.NewRecorder()
	srv.handleGatewayCapabilityStatusUpdate(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden for missing approval, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(strings.ToLower(rr.Body.String()), "approval request 999999 not found") {
		t.Fatalf("expected deterministic approval rejection, got %q", rr.Body.String())
	}
}

func newGatewayCapabilityStatusHarness(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{
		DataDir:      dataDir,
		WorkspaceDir: workspaceDir,
	})
	t.Cleanup(func() { srv.ShutdownWatch() })
	return srv, st
}
