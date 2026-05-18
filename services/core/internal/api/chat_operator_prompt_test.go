package api

import (
	"context"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestDefaultChatOperatorSystemPromptRequiresSourceTruthBootstrap(t *testing.T) {
	prompt := defaultChatOperatorSystemPrompt()
	required := []string{
		"capability probe",
		"README.md",
		"AGENTS.md",
		"docs/reviews/current_phase_status.md",
		"Do not summarize project status from persona",
		"report the exact tool or filesystem error",
	}
	for _, want := range required {
		if !strings.Contains(prompt, want) {
			t.Fatalf("default chat prompt missing %q\nprompt:\n%s", want, prompt)
		}
	}
}

func TestChatOperatorSystemPromptAppendsGroundingGuardToCustomPrompt(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := upsertSetting(context.Background(), st.DB, "chat_personality_prompt", "custom operator prompt"); err != nil {
		t.Fatalf("upsert prompt: %v", err)
	}

	srv := &Server{st: st, cfg: config.Config{}}
	prompt := srv.chatOperatorSystemPrompt()
	required := []string{
		"custom operator prompt",
		"capability probe",
		"README.md",
		"AGENTS.md",
		"docs/reviews/current_phase_status.md",
		"Do not summarize project status from persona",
	}
	for _, want := range required {
		if !strings.Contains(prompt, want) {
			t.Fatalf("custom chat prompt missing %q\nprompt:\n%s", want, prompt)
		}
	}
}

func TestChatOperatorSystemPromptDoesNotDuplicateGuardsWhenDefaultIsSaved(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := upsertSetting(context.Background(), st.DB, "chat_personality_prompt", defaultChatOperatorSystemPrompt()); err != nil {
		t.Fatalf("upsert prompt: %v", err)
	}

	srv := &Server{st: st, cfg: config.Config{}}
	prompt := srv.chatOperatorSystemPrompt()
	if got := strings.Count(prompt, "Operational grounding:"); got != 1 {
		t.Fatalf("expected one operational grounding guard, got %d\nprompt:\n%s", got, prompt)
	}
	if got := strings.Count(prompt, "Visibility boundary:"); got != 1 {
		t.Fatalf("expected one visibility boundary guard, got %d\nprompt:\n%s", got, prompt)
	}
}

func TestModelRuntimeSystemPromptKeepsVisibilityGuardWhenDefaultIsSaved(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := upsertSetting(context.Background(), st.DB, "chat_personality_prompt", defaultChatOperatorSystemPrompt()); err != nil {
		t.Fatalf("upsert prompt: %v", err)
	}

	srv := &Server{st: st, cfg: config.Config{}}
	otherThread, err := chat.New(st.DB).CreateThread(context.Background(), "related memory", nil)
	if err != nil {
		t.Fatalf("create related thread: %v", err)
	}
	if _, err := chat.New(st.DB).AppendMessage(context.Background(), otherThread.ID, "assistant", strings.Repeat("related memory pressure ", 80), nil); err != nil {
		t.Fatalf("append related message: %v", err)
	}

	messages, budget := srv.buildModelRuntimePlainChatMessages(context.Background(), &chat.ThreadDetail{
		ThreadSummary: chat.ThreadSummary{Title: "personality prompt probe"},
		Messages: []chat.Message{
			{Role: "user", Content: "Who are you?"},
		},
	})
	if len(messages) == 0 {
		t.Fatal("expected at least system message")
	}
	system := messages[0].Content
	if budget.SystemChars > modelRuntimePlainChatSystemMax {
		t.Fatalf("system prompt exceeded cap: got %d max %d", budget.SystemChars, modelRuntimePlainChatSystemMax)
	}
	required := []string{
		"Your visible name is FORGE",
		"Visibility boundary:",
		"Never identify as Phi",
	}
	for _, want := range required {
		if !strings.Contains(system, want) {
			t.Fatalf("bounded system prompt missing %q\nsystem:\n%s", want, system)
		}
	}
}
