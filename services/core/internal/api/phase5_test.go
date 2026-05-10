package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestHandleBackupBundleAuditsCarryCorrelationAndTraceContext(t *testing.T) {
	srv, st := newBackupAuditHarness(t)

	createCorrelation := "corr-backup-create"
	createTrace := "trace-backup-create"
	createWorkspace := "workspace-backup-create"

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/backup/bundles?correlationId="+createCorrelation+"&traceId="+createTrace+"&workspaceId="+createWorkspace,
		strings.NewReader(`{"kind":"dossiers","label":"audit-check"}`),
	)
	createRR := httptest.NewRecorder()
	srv.handleCreateBundle(createRR, createReq)
	if createRR.Code != http.StatusOK {
		t.Fatalf("expected bundle create success, got %d body=%s", createRR.Code, strings.TrimSpace(createRR.Body.String()))
	}

	var createResp struct {
		Bundle struct {
			FilePath string `json:"filePath"`
		} `json:"bundle"`
	}
	if err := json.Unmarshal(createRR.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v body=%s", err, createRR.Body.String())
	}
	if createResp.Bundle.FilePath == "" {
		t.Fatalf("expected create response to include file path")
	}

	createAuditCorr, createOutcome, createPayload := mustAuditRecordByActionAndCorrelation(t, st, "bundle.created", createCorrelation)
	if createAuditCorr != createCorrelation {
		t.Fatalf("create audit correlation = %q want %q", createAuditCorr, createCorrelation)
	}
	if createOutcome != "ok" {
		t.Fatalf("create audit outcome = %q want ok", createOutcome)
	}
	if !strings.Contains(createPayload, `"traceId":"`+createTrace+`"`) {
		t.Fatalf("expected create audit trace id, got %s", createPayload)
	}
	if !strings.Contains(createPayload, `"workspaceId":"`+createWorkspace+`"`) {
		t.Fatalf("expected create audit workspace id, got %s", createPayload)
	}

	restoreCorrelation := "corr-backup-restore"
	restoreTrace := "trace-backup-restore"
	restoreWorkspace := "workspace-backup-restore"
	restoreBody, err := json.Marshal(map[string]any{
		"filePath": createResp.Bundle.FilePath,
		"dryRun":   true,
	})
	if err != nil {
		t.Fatalf("encode restore body: %v", err)
	}
	restoreReq := httptest.NewRequest(
		http.MethodPost,
		"/api/backup/restore?correlationId="+restoreCorrelation+"&traceId="+restoreTrace+"&workspaceId="+restoreWorkspace,
		strings.NewReader(string(restoreBody)),
	)
	restoreRR := httptest.NewRecorder()
	srv.handleRestoreBundle(restoreRR, restoreReq)
	if restoreRR.Code != http.StatusOK {
		t.Fatalf("expected bundle restore success, got %d body=%s", restoreRR.Code, strings.TrimSpace(restoreRR.Body.String()))
	}

	restoreAuditCorr, restoreOutcome, restorePayload := mustAuditRecordByActionAndCorrelation(t, st, "bundle.restored", restoreCorrelation)
	if restoreAuditCorr != restoreCorrelation {
		t.Fatalf("restore audit correlation = %q want %q", restoreAuditCorr, restoreCorrelation)
	}
	if restoreOutcome != "ok" {
		t.Fatalf("restore audit outcome = %q want ok", restoreOutcome)
	}
	if !strings.Contains(restorePayload, `"traceId":"`+restoreTrace+`"`) {
		t.Fatalf("expected restore audit trace id, got %s", restorePayload)
	}
	if !strings.Contains(restorePayload, `"workspaceId":"`+restoreWorkspace+`"`) {
		t.Fatalf("expected restore audit workspace id, got %s", restorePayload)
	}
}

func TestHandleRestoreBundleRequiresApprovalForNonDryRun(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)

	createRR := httptest.NewRecorder()
	srv.handleCreateBundle(createRR, httptest.NewRequest(
		http.MethodPost,
		"/api/backup/bundles",
		strings.NewReader(`{"kind":"dossiers","label":"approval-check"}`),
	))
	if createRR.Code != http.StatusOK {
		t.Fatalf("expected bundle create success, got %d body=%s", createRR.Code, strings.TrimSpace(createRR.Body.String()))
	}
	var createResp struct {
		Bundle struct {
			FilePath string `json:"filePath"`
		} `json:"bundle"`
	}
	if err := json.Unmarshal(createRR.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"filePath": createResp.Bundle.FilePath,
		"dryRun":   false,
	})
	needsApprovalRR := httptest.NewRecorder()
	srv.handleRestoreBundle(needsApprovalRR, httptest.NewRequest(
		http.MethodPost,
		"/api/backup/restore",
		strings.NewReader(string(body)),
	))
	if needsApprovalRR.Code != http.StatusAccepted {
		t.Fatalf("expected restore approval gate, got %d body=%s", needsApprovalRR.Code, strings.TrimSpace(needsApprovalRR.Body.String()))
	}
	var needsApproval struct {
		Governance struct {
			RequiresApproval  bool  `json:"requiresApproval"`
			Approved          bool  `json:"approved"`
			ApprovalRequestID int64 `json:"approvalRequestId"`
		} `json:"governance"`
	}
	if err := json.Unmarshal(needsApprovalRR.Body.Bytes(), &needsApproval); err != nil {
		t.Fatalf("decode approval response: %v body=%s", err, needsApprovalRR.Body.String())
	}
	if !needsApproval.Governance.RequiresApproval || needsApproval.Governance.Approved || needsApproval.Governance.ApprovalRequestID <= 0 {
		t.Fatalf("expected pending approval governance, got %+v", needsApproval.Governance)
	}

	if _, err := srv.approvals.Decide(context.Background(), needsApproval.Governance.ApprovalRequestID, "operator-test", "approved", "approve restore"); err != nil {
		t.Fatalf("approve restore request: %v", err)
	}
	approvedBody, _ := json.Marshal(map[string]any{
		"filePath":   createResp.Bundle.FilePath,
		"dryRun":     false,
		"approvalId": strconv.FormatInt(needsApproval.Governance.ApprovalRequestID, 10),
	})
	approvedRR := httptest.NewRecorder()
	srv.handleRestoreBundle(approvedRR, httptest.NewRequest(
		http.MethodPost,
		"/api/backup/restore",
		strings.NewReader(string(approvedBody)),
	))
	if approvedRR.Code != http.StatusOK {
		t.Fatalf("expected approved restore success, got %d body=%s", approvedRR.Code, strings.TrimSpace(approvedRR.Body.String()))
	}
}

func TestHandleRestoreBundleRejectsApprovalReplayForDifferentBundle(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	firstPath := mustCreateBackupBundlePath(t, srv, "first")
	secondPath := mustCreateBackupBundlePath(t, srv, "second")

	body, _ := json.Marshal(map[string]any{"filePath": firstPath})
	needsApprovalRR := httptest.NewRecorder()
	srv.handleRestoreBundle(needsApprovalRR, httptest.NewRequest(http.MethodPost, "/api/backup/restore", strings.NewReader(string(body))))
	if needsApprovalRR.Code != http.StatusAccepted {
		t.Fatalf("expected approval gate, got %d body=%s", needsApprovalRR.Code, strings.TrimSpace(needsApprovalRR.Body.String()))
	}
	var needsApproval struct {
		Governance struct {
			ApprovalRequestID int64 `json:"approvalRequestId"`
		} `json:"governance"`
	}
	if err := json.Unmarshal(needsApprovalRR.Body.Bytes(), &needsApproval); err != nil {
		t.Fatalf("decode approval response: %v", err)
	}
	if _, err := srv.approvals.Decide(context.Background(), needsApproval.Governance.ApprovalRequestID, "operator-test", "approved", "approve restore"); err != nil {
		t.Fatalf("approve restore request: %v", err)
	}

	replayBody, _ := json.Marshal(map[string]any{
		"filePath":   secondPath,
		"approvalId": strconv.FormatInt(needsApproval.Governance.ApprovalRequestID, 10),
	})
	replayRR := httptest.NewRecorder()
	srv.handleRestoreBundle(replayRR, httptest.NewRequest(http.MethodPost, "/api/backup/restore", strings.NewReader(string(replayBody))))
	if replayRR.Code != http.StatusForbidden {
		t.Fatalf("expected approval replay rejection, got %d body=%s", replayRR.Code, strings.TrimSpace(replayRR.Body.String()))
	}
	if !strings.Contains(replayRR.Body.String(), "fingerprint mismatch") {
		t.Fatalf("expected fingerprint mismatch response, got %s", replayRR.Body.String())
	}
}

func TestHandleRestoreBundleRejectsOutsidePathBeforeApproval(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{"schema":1}`), 0o644); err != nil {
		t.Fatalf("write outside bundle: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"filePath": outside, "dryRun": true})
	rr := httptest.NewRecorder()
	srv.handleRestoreBundle(rr, httptest.NewRequest(http.MethodPost, "/api/backup/restore", strings.NewReader(string(body))))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected outside path rejection, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if !strings.Contains(rr.Body.String(), "backup or export directory") {
		t.Fatalf("expected bundle dir rejection, got %s", rr.Body.String())
	}
}

func newBackupAuditHarness(t *testing.T) (*Server, *store.Store) {
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

func mustCreateBackupBundlePath(t *testing.T, srv *Server, label string) string {
	t.Helper()
	rr := httptest.NewRecorder()
	srv.handleCreateBundle(rr, httptest.NewRequest(
		http.MethodPost,
		"/api/backup/bundles",
		strings.NewReader(`{"kind":"dossiers","label":"`+label+`"}`),
	))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected bundle create success, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	var resp struct {
		Bundle struct {
			FilePath string `json:"filePath"`
		} `json:"bundle"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode bundle response: %v", err)
	}
	if resp.Bundle.FilePath == "" {
		t.Fatalf("expected bundle path")
	}
	return resp.Bundle.FilePath
}
