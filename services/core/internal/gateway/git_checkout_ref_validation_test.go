package gateway

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeGitCheckoutRefRejectsUnsafeValues(t *testing.T) {
	t.Parallel()
	for _, ref := range []string{
		"--detach",
		"-B",
		"main -- path",
		"main\nother",
		"feature/../main",
		"feature@{1}",
		"refs/heads/main.lock",
		`feature\main`,
		strings.Repeat("a", 257),
	} {
		if _, err := normalizeGitCheckoutRef(ref); err == nil {
			t.Fatalf("expected checkout ref %q to be rejected", ref)
		}
	}
}

func TestNormalizeGitCheckoutRefAllowsBoundedRefs(t *testing.T) {
	t.Parallel()
	for _, ref := range []string{
		"main",
		"feature/shell-surfaces",
		"release-2026.05.10",
		"abc123def456",
		"HEAD",
		"HEAD~1",
		"main^",
		"origin/main",
	} {
		got, err := normalizeGitCheckoutRef(" " + ref + " ")
		if err != nil {
			t.Fatalf("expected checkout ref %q to be accepted: %v", ref, err)
		}
		if got != ref {
			t.Fatalf("expected checkout ref %q, got %q", ref, got)
		}
	}
}

func TestGitCheckoutToolRejectsUnsafeRefBeforeExecution(t *testing.T) {
	t.Parallel()
	tool := &gitCheckoutTool{workspace: t.TempDir()}
	if _, err := tool.Execute(context.Background(), Request{
		Input: map[string]any{"ref": "--detach"},
	}); err == nil {
		t.Fatalf("expected unsafe checkout ref to be rejected")
	}
}
