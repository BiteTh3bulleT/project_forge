package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
