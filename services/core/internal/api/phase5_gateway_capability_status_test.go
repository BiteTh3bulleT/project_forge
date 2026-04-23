package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	for _, status := range []string{"deferred", "disabled", "stubbed", "deprecated"} {
		req := withRouteParam(
			httptest.NewRequest(
				http.MethodPatch,
				"/api/gateway/capabilities/filesystem.read_file/status",
				bytes.NewBufferString(`{"status":"`+status+`"}`),
			),
			"id",
			"filesystem.read_file",
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

func TestHandleGatewayCapabilityStatusUpdateAllowsActiveWithoutReason(t *testing.T) {
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
			bytes.NewBufferString(`{"status":"deferred","reason":"deferred until adapter readiness"}`),
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
