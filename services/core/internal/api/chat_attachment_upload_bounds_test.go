package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestChatAttachmentUploadRejectsOversizeMultipartBody(t *testing.T) {
	t.Parallel()

	srv, _ := newBackupAuditHarness(t)
	thread, err := srv.chat.CreateThread(context.Background(), "oversize upload", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("file", "large.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte(strings.Repeat("a", chatAttachmentUploadRequestLimit))); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := withRouteParam(
		httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/attachments", &body),
		"id",
		strconv.FormatInt(thread.ID, 10),
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	srv.handleChatAttachmentUpload(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusRequestEntityTooLarge, rr.Body.String())
	}
	if !strings.Contains(strings.ToLower(rr.Body.String()), "too large") {
		t.Fatalf("expected too-large response, got %q", rr.Body.String())
	}
}
