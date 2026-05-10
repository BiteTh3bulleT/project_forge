package gateway

import (
	"context"
	"testing"
)

func TestNormalizeGitStashModeRejectsUnsupportedSubcommands(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"clear", "drop", "branch", "create", "store", "--help", "push --all"} {
		if _, err := normalizeGitStashMode(mode); err == nil {
			t.Fatalf("expected git stash mode %q to be rejected", mode)
		}
	}
}

func TestNormalizeGitStashModeAllowsDeclaredModes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{raw: "", want: "push"},
		{raw: " push ", want: "push"},
		{raw: "LIST", want: "list"},
		{raw: "pop", want: "pop"},
	} {
		got, err := normalizeGitStashMode(tc.raw)
		if err != nil {
			t.Fatalf("expected mode %q to be accepted: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("expected mode %q, got %q", tc.want, got)
		}
	}
}

func TestGitStashToolRejectsUnsupportedModeBeforeExecution(t *testing.T) {
	t.Parallel()
	tool := &gitStashTool{workspace: t.TempDir()}
	if _, err := tool.Execute(context.Background(), Request{
		Input: map[string]any{"mode": "clear"},
	}); err == nil {
		t.Fatalf("expected unsupported git stash mode to be rejected")
	}
}
