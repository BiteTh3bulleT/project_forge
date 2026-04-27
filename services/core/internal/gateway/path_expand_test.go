package gateway

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsExpandsHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("home directory unavailable")
	}

	got := resolvePaths(string(filepath.Separator), []string{"~/Downloads/Python_Scripts"})
	if len(got) != 1 {
		t.Fatalf("expected one resolved path, got %d", len(got))
	}
	want := filepath.Join(home, "Downloads", "Python_Scripts")
	if got[0] != filepath.Clean(want) {
		t.Fatalf("resolved path = %q want %q", got[0], filepath.Clean(want))
	}
	if got[0] == filepath.Clean(filepath.Join(string(filepath.Separator), "~", "Downloads", "Python_Scripts")) {
		t.Fatalf("home path was treated as literal /~ path")
	}
}

func TestPathWithinScopeExpandsForbiddenHomeScope(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("home directory unavailable")
	}

	target := filepath.Join(home, ".ssh", "config")
	if !pathWithinScope(target, "~/.ssh") {
		t.Fatalf("expected ~/.ssh scope to match expanded target %q", target)
	}
}
