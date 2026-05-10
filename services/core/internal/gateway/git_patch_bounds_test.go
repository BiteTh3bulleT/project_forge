package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeGitPatchInputIsBounded(t *testing.T) {
	t.Parallel()
	patch, err := normalizeGitPatchInput("  diff --git a/a b/a\n  ", "git.apply_patch")
	if err != nil {
		t.Fatalf("expected valid patch, got %v", err)
	}
	if patch != "diff --git a/a b/a" {
		t.Fatalf("unexpected normalized patch %q", patch)
	}
	if _, err := normalizeGitPatchInput(strings.Repeat("x", maxGitPatchInputBytes+1), "git.apply_patch"); !errors.Is(err, errGitPatchTooLarge) {
		t.Fatalf("expected patch size rejection, got %v", err)
	}
}
