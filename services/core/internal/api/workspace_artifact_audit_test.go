package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestHandleChatAttachmentUploadAuditsTraceContext(t *testing.T) {
	srv, st := newBackupAuditHarness(t)

	thread, err := srv.chat.CreateThread(context.Background(), "Artifact audit thread", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "note.txt")
	if err != nil {
		t.Fatalf("create multipart file part: %v", err)
	}
	if _, err := io.WriteString(part, "hello from artifact audit test"); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.WriteField("title", "Audit Artifact"); err != nil {
		t.Fatalf("write title field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	correlation := "corr-artifact-upload"
	trace := "trace-artifact-upload"
	workspace := "workspace-artifact-upload"
	path := fmt.Sprintf(
		"/api/chat/threads/%d/attachments?correlationId=%s&traceId=%s&workspaceId=%s",
		thread.ID, correlation, trace, workspace,
	)
	req := withRouteParam(httptest.NewRequest(http.MethodPost, path, &body), "id", strconv.FormatInt(thread.ID, 10))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	srv.handleChatAttachmentUpload(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected upload success, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	auditCorrelation, outcome, payload := mustAuditRecordByActionAndCorrelation(t, st, "artifact.uploaded", correlation)
	if auditCorrelation != correlation {
		t.Fatalf("audit correlation = %q want %q", auditCorrelation, correlation)
	}
	if outcome != "ok" {
		t.Fatalf("audit outcome = %q want ok", outcome)
	}
	if !strings.Contains(payload, `"traceId":"`+trace+`"`) {
		t.Fatalf("expected trace id in audit payload, got %s", payload)
	}
	if !strings.Contains(payload, `"workspaceId":"`+workspace+`"`) {
		t.Fatalf("expected workspace id in audit payload, got %s", payload)
	}
	if !strings.Contains(payload, `"artifactId"`) {
		t.Fatalf("expected artifact id in audit payload, got %s", payload)
	}
}

func TestHandleChatAttachmentUploadAuditsWorkspaceFallback(t *testing.T) {
	srv, st := newBackupAuditHarness(t)

	thread, err := srv.chat.CreateThread(context.Background(), "Artifact workspace fallback thread", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "note.txt")
	if err != nil {
		t.Fatalf("create multipart file part: %v", err)
	}
	if _, err := io.WriteString(part, "workspace fallback coverage"); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	correlation := "corr-artifact-upload-fallback"
	trace := "trace-artifact-upload-fallback"
	path := fmt.Sprintf(
		"/api/chat/threads/%d/attachments?correlationId=%s&traceId=%s",
		thread.ID, correlation, trace,
	)
	req := withRouteParam(httptest.NewRequest(http.MethodPost, path, &body), "id", strconv.FormatInt(thread.ID, 10))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	srv.handleChatAttachmentUpload(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected upload success, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	_, _, payload := mustAuditRecordByActionAndCorrelation(t, st, "artifact.uploaded", correlation)
	expectedWorkspace := "workspace:" + strings.TrimSpace(filepath.Base(strings.TrimSpace(srv.cfg.WorkspaceDir)))
	if !strings.Contains(payload, `"workspaceId":"`+expectedWorkspace+`"`) {
		t.Fatalf("expected workspace fallback in payload, got %s", payload)
	}
}
