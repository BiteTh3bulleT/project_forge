package api

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/artifacts"
)

func TestResolveThreadAttachmentReadByFileName(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	ctx := context.Background()

	th, err := srv.chat.CreateThread(ctx, "attachment read thread", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	art, err := srv.artifacts.CreateTextArtifact(ctx, artifacts.CreateTextArtifactRequest{
		Type:     "chat_attachment",
		Title:    "test.txt",
		FileName: "test.txt",
		Subdir:   "chat",
		Content:  "hello attachment",
		MimeType: "text/plain",
		Metadata: map[string]any{"threadId": th.ID},
	})
	if err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	_, err = srv.chat.AppendMessage(ctx, th.ID, "user", "Can you read this?", map[string]any{
		"attachments": []any{map[string]any{"artifactId": art.ID}},
	})
	if err != nil {
		t.Fatalf("append message: %v", err)
	}

	content, meta, ok, err := srv.resolveThreadAttachmentRead(ctx, th.ID, "test.txt")
	if err != nil {
		t.Fatalf("resolve attachment read: %v", err)
	}
	if !ok {
		t.Fatalf("expected attachment read resolution")
	}
	if content == "" {
		t.Fatalf("expected content")
	}
	if meta["artifactId"] != art.ID {
		t.Fatalf("artifact mismatch: got=%v want=%d", meta["artifactId"], art.ID)
	}
}

func TestDispatchToolCallReadsThreadAttachmentWithoutWorkspacePath(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	ctx := context.Background()

	th, err := srv.chat.CreateThread(ctx, "dispatch attachment thread", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	art, err := srv.artifacts.CreateTextArtifact(ctx, artifacts.CreateTextArtifactRequest{
		Type:     "chat_attachment",
		Title:    "test.txt",
		FileName: "test.txt",
		Subdir:   "chat",
		Content:  "hello from dispatch attachment",
		MimeType: "text/plain",
		Metadata: map[string]any{"threadId": th.ID},
	})
	if err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	_, err = srv.chat.AppendMessage(ctx, th.ID, "user", "Can you read this?", map[string]any{
		"attachments": []any{map[string]any{"artifactId": art.ID}},
	})
	if err != nil {
		t.Fatalf("append message: %v", err)
	}

	res := srv.dispatchToolCall(ctx, "corr-attachment", th.ID, "filesystem_read_file", `{"path":"test.txt"}`, "Can you read this?", func(string, map[string]any) {})
	if res.state != "ok" {
		t.Fatalf("expected ok state, got %q text=%q reason=%q", res.state, res.text, res.failureReason)
	}
	data, _ := res.executionResult.(map[string]any)
	if data == nil {
		t.Fatalf("expected execution result map")
	}
	if data["source"] != "chat_attachment" {
		t.Fatalf("expected chat_attachment source, got %v", data["source"])
	}
}
