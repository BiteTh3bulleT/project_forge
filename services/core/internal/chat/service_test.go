package chat

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"forge/projectforge/services/core/internal/store"
)

func TestServiceThreadMessageLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	thread, err := svc.CreateThread(ctx, "  ", nil)
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	if thread.Title != "Conversation" {
		t.Fatalf("default title = %q, want Conversation", thread.Title)
	}
	if thread.DossierID != nil {
		t.Fatalf("dossier id = %v, want nil", thread.DossierID)
	}

	user, err := svc.AppendMessage(ctx, thread.ID, " USER ", "  remember this thread  ", map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("AppendMessage user failed: %v", err)
	}
	if user.Role != "user" || user.Content != "remember this thread" {
		t.Fatalf("normalized user message = role %q content %q", user.Role, user.Content)
	}

	if _, err := svc.AppendMessage(ctx, thread.ID, "assistant", "Stored.", map[string]any{"replyToUserMessageId": user.ID}); err != nil {
		t.Fatalf("AppendMessage assistant failed: %v", err)
	}

	detail, err := svc.GetThread(ctx, thread.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}
	if detail.Title != "remember this thread" {
		t.Fatalf("auto title = %q, want first user message", detail.Title)
	}
	if len(detail.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(detail.Messages))
	}
	if detail.Messages[0].Metadata["source"] != "test" {
		t.Fatalf("metadata source = %v, want test", detail.Messages[0].Metadata["source"])
	}

	threads, err := svc.ListThreads(ctx, 1)
	if err != nil {
		t.Fatalf("ListThreads failed: %v", err)
	}
	if len(threads) != 1 || threads[0].ID != thread.ID {
		t.Fatalf("listed threads = %+v, want only thread %d", threads, thread.ID)
	}

	reply, err := svc.FindAssistantReplyTo(ctx, thread.ID, user.ID)
	if err != nil {
		t.Fatalf("FindAssistantReplyTo failed: %v", err)
	}
	if reply == nil || reply.Content != "Stored." {
		t.Fatalf("assistant reply = %+v, want Stored.", reply)
	}
}

func TestServiceValidationAndTitleRules(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	thread, err := svc.CreateThread(ctx, "Custom title", nil)
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	if _, err := svc.AppendMessage(ctx, thread.ID, "admin", "hello", nil); err == nil {
		t.Fatalf("AppendMessage accepted invalid role")
	}
	if _, err := svc.AppendMessage(ctx, thread.ID, "user", "   ", nil); err == nil {
		t.Fatalf("AppendMessage accepted empty content")
	}
	if _, err := svc.AppendMessage(ctx, thread.ID, "user", "bad metadata", map[string]any{"bad": func() {}}); err == nil {
		t.Fatalf("AppendMessage accepted unsupported metadata")
	}
	if _, err := svc.AppendMessage(ctx, thread.ID+999, "user", "missing thread", nil); err == nil {
		t.Fatalf("AppendMessage accepted missing thread")
	}
	if _, err := svc.UpdateThreadTitle(ctx, thread.ID, " "); err == nil {
		t.Fatalf("UpdateThreadTitle accepted empty title")
	}

	if _, err := svc.AppendMessage(ctx, thread.ID, "user", "do not replace the custom title", nil); err != nil {
		t.Fatalf("AppendMessage first user failed: %v", err)
	}
	detail, err := svc.GetThread(ctx, thread.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}
	if detail.Title != "Custom title" {
		t.Fatalf("custom title was overwritten: %q", detail.Title)
	}

	untitled, err := svc.CreateThread(ctx, "New Chat", nil)
	if err != nil {
		t.Fatalf("CreateThread New Chat failed: %v", err)
	}
	longPrompt := strings.Repeat("title ", 30)
	if _, err := svc.AppendMessage(ctx, untitled.ID, "user", longPrompt, nil); err != nil {
		t.Fatalf("AppendMessage long first user failed: %v", err)
	}
	firstTitle, err := svc.GetThread(ctx, untitled.ID)
	if err != nil {
		t.Fatalf("GetThread untitled failed: %v", err)
	}
	if !strings.HasPrefix(firstTitle.Title, "title title") {
		t.Fatalf("generated title = %q, want normalized prompt prefix", firstTitle.Title)
	}
	if utf8.RuneCountInString(firstTitle.Title) > 65 {
		t.Fatalf("generated title length = %d runes, want <= 65", utf8.RuneCountInString(firstTitle.Title))
	}
	if _, err := svc.AppendMessage(ctx, untitled.ID, "user", "second user message", nil); err != nil {
		t.Fatalf("AppendMessage second user failed: %v", err)
	}
	afterSecond, err := svc.GetThread(ctx, untitled.ID)
	if err != nil {
		t.Fatalf("GetThread after second user failed: %v", err)
	}
	if afterSecond.Title != firstTitle.Title {
		t.Fatalf("second user message changed title from %q to %q", firstTitle.Title, afterSecond.Title)
	}
}

func TestServiceListThreadsDeterministicOrder(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	first, err := svc.CreateThread(ctx, "first", nil)
	if err != nil {
		t.Fatalf("CreateThread first failed: %v", err)
	}
	second, err := svc.CreateThread(ctx, "second", nil)
	if err != nil {
		t.Fatalf("CreateThread second failed: %v", err)
	}
	const sameUpdatedAt = int64(123456789)
	if _, err := svc.db.ExecContext(ctx, `UPDATE chat_threads SET updated_at = ? WHERE id IN (?, ?)`, sameUpdatedAt, first.ID, second.ID); err != nil {
		t.Fatalf("force same updated_at failed: %v", err)
	}

	threads, err := svc.ListThreads(ctx, 2)
	if err != nil {
		t.Fatalf("ListThreads failed: %v", err)
	}
	if len(threads) != 2 {
		t.Fatalf("threads = %d, want 2", len(threads))
	}
	if threads[0].ID != second.ID || threads[1].ID != first.ID {
		t.Fatalf("thread order = [%d %d], want newest id first [%d %d]", threads[0].ID, threads[1].ID, second.ID, first.ID)
	}
}

func TestServiceFindAssistantReplyToIgnoresMalformedAndWrongThreadMetadata(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	thread, err := svc.CreateThread(ctx, "Conversation", nil)
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	other, err := svc.CreateThread(ctx, "Other", nil)
	if err != nil {
		t.Fatalf("CreateThread other failed: %v", err)
	}
	user, err := svc.AppendMessage(ctx, thread.ID, "user", "question", nil)
	if err != nil {
		t.Fatalf("AppendMessage user failed: %v", err)
	}
	if _, err := svc.AppendMessage(ctx, other.ID, "assistant", "wrong thread", map[string]any{"replyToUserMessageId": user.ID}); err != nil {
		t.Fatalf("AppendMessage wrong-thread assistant failed: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `
INSERT INTO chat_messages(thread_id, role, content, created_at, metadata_json)
VALUES(?, 'assistant', 'malformed metadata', 1, '')`, thread.ID); err != nil {
		t.Fatalf("insert malformed metadata row failed: %v", err)
	}

	reply, err := svc.FindAssistantReplyTo(ctx, thread.ID, user.ID)
	if err != nil {
		t.Fatalf("FindAssistantReplyTo no valid match failed: %v", err)
	}
	if reply != nil {
		t.Fatalf("reply = %+v, want nil before valid same-thread assistant", reply)
	}

	if _, err := svc.AppendMessage(ctx, thread.ID, "assistant", "older valid", map[string]any{"replyToUserMessageId": user.ID}); err != nil {
		t.Fatalf("AppendMessage older valid assistant failed: %v", err)
	}
	if _, err := svc.AppendMessage(ctx, thread.ID, "assistant", "newer valid", map[string]any{"replyToUserMessageId": user.ID}); err != nil {
		t.Fatalf("AppendMessage newer valid assistant failed: %v", err)
	}
	reply, err = svc.FindAssistantReplyTo(ctx, thread.ID, user.ID)
	if err != nil {
		t.Fatalf("FindAssistantReplyTo valid match failed: %v", err)
	}
	if reply == nil || reply.Content != "newer valid" {
		t.Fatalf("reply = %+v, want latest valid same-thread assistant", reply)
	}
}

func TestServiceTranscriptBoundsAndDeleteCascade(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	thread, err := svc.CreateThread(ctx, "Conversation", nil)
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	messages := []Message{
		{Role: "system", Content: "system primer"},
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: strings.Repeat("long ", 20)},
		{Role: "user", Content: "last"},
	}
	transcript := svc.BuildTranscript(messages, 2)
	if strings.Contains(transcript, "system primer") || strings.Contains(transcript, "first") {
		t.Fatalf("BuildTranscript included messages outside max window: %q", transcript)
	}
	if !strings.Contains(transcript, "ASSISTANT:") || !strings.Contains(transcript, "USER: last") {
		t.Fatalf("BuildTranscript missing newest messages: %q", transcript)
	}

	bounded := svc.BuildBoundedTranscript(messages, 3, 18, 60)
	if utf8.RuneCountInString(bounded) > 60 {
		t.Fatalf("bounded transcript length = %d, want <= 60", utf8.RuneCountInString(bounded))
	}
	if !strings.Contains(bounded, "truncated") {
		t.Fatalf("bounded transcript did not mark truncation: %q", bounded)
	}

	if _, err := svc.AppendMessage(ctx, thread.ID, "user", "persist me", nil); err != nil {
		t.Fatalf("AppendMessage before delete failed: %v", err)
	}
	if err := svc.DeleteThread(ctx, thread.ID); err != nil {
		t.Fatalf("DeleteThread failed: %v", err)
	}
	if _, err := svc.GetThread(ctx, thread.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetThread after delete error = %v, want sql.ErrNoRows", err)
	}
	var count int
	if err := svc.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM chat_messages WHERE thread_id = ?`, thread.ID).Scan(&count); err != nil {
		t.Fatalf("count messages after delete failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("messages after thread delete = %d, want cascade delete", count)
	}
}

func newTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return New(st.DB), func() {
		if err := st.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}
}
