package permissions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePathExpandsHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("home directory unavailable")
	}

	got := normalizePath("~/Downloads/Python_Scripts")
	want := filepath.Join(home, "Downloads", "Python_Scripts")
	if got != filepath.Clean(want) {
		t.Fatalf("normalized path = %q want %q", got, filepath.Clean(want))
	}
}

func TestPathScopeMatchExpandsHomeScope(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("home directory unavailable")
	}

	target := filepath.Join(home, ".ssh", "config")
	if !pathScopeMatch(target, "~/.ssh") {
		t.Fatalf("expected ~/.ssh scope to match expanded target %q", target)
	}
}
