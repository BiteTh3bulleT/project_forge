package api

import (
	"context"
	"strings"
	"testing"

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
