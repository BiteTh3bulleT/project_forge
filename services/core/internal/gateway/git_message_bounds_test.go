package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeGitMessageInputIsBounded(t *testing.T) {
	t.Parallel()
	message, err := normalizeGitMessageInput("  FORGE update  ", "git.commit")
	if err != nil {
		t.Fatalf("expected valid message, got %v", err)
	}
	if message != "FORGE update" {
		t.Fatalf("unexpected normalized message %q", message)
	}
	if _, err := normalizeGitMessageInput(strings.Repeat("x", maxGitMessageInputBytes+1), "git.stash"); !errors.Is(err, errGitMessageTooLarge) {
		t.Fatalf("expected git message size rejection, got %v", err)
	}
}
