package gateway

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFilesystemWriteRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink scope checks use Unix symlink semantics")
	}
	workspace, outside := symlinkScopeHarness(t)
	outsideFile := filepath.Join(outside, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("original"), 0o644); err != nil {
		t.Fatalf("seed outside file: %v", err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(workspace, "link-file")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err := (&writeFileTool{workspace: workspace}).Execute(context.Background(), Request{
		Paths: []string{"link-file"},
		Input: map[string]any{"contents": "mutated"},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("expected symlink escape rejection, got %v", err)
	}
	body, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatalf("read outside file: %v", err)
	}
	if string(body) != "original" {
		t.Fatalf("outside file was mutated: %q", string(body))
	}
}

func TestFilesystemCopyRejectsSymlinkParentEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink scope checks use Unix symlink semantics")
	}
	workspace, outside := symlinkScopeHarness(t)
	if err := os.WriteFile(filepath.Join(workspace, "source.txt"), []byte("source"), 0o644); err != nil {
		t.Fatalf("seed source file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, "link-dir")); err != nil {
		t.Fatalf("create symlink directory: %v", err)
	}

	_, err := (&copyTool{workspace: workspace}).Execute(context.Background(), Request{
		Paths: []string{"source.txt", "link-dir/copied.txt"},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("expected symlink parent escape rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "copied.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no outside copy, stat err=%v", err)
	}
}

func TestFilesystemChmodRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink scope checks use Unix symlink semantics")
	}
	workspace, outside := symlinkScopeHarness(t)
	outsideFile := filepath.Join(outside, "mode-target.txt")
	if err := os.WriteFile(outsideFile, []byte("mode"), 0o644); err != nil {
		t.Fatalf("seed outside file: %v", err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(workspace, "mode-link")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err := (&chmodTool{workspace: workspace}).Execute(context.Background(), Request{
		Paths: []string{"mode-link"},
		Input: map[string]any{"mode": "0600"},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("expected symlink escape rejection, got %v", err)
	}
	info, err := os.Stat(outsideFile)
	if err != nil {
		t.Fatalf("stat outside file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("outside mode changed to %o", got)
	}
}

func symlinkScopeHarness(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}
	return workspace, outside
}
