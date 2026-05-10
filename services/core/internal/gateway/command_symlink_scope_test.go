package gateway

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProcessRunRejectsSymlinkWorkingDirectoryEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink scope checks use Unix symlink semantics")
	}
	workspace, outside := symlinkScopeHarness(t)
	if err := os.Symlink(outside, filepath.Join(workspace, "link-dir")); err != nil {
		t.Fatalf("create symlink directory: %v", err)
	}

	_, err := (&processRunTool{workspace: workspace}).Execute(context.Background(), Request{
		Paths: []string{"link-dir"},
		Input: map[string]any{"command": "printf owned > marker.txt"},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("expected symlink cwd rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "marker.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no command output outside workspace, stat err=%v", err)
	}
}

func TestGitStatusRejectsSymlinkWorkingDirectoryEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink scope checks use Unix symlink semantics")
	}
	workspace, outside := symlinkScopeHarness(t)
	if err := os.Symlink(outside, filepath.Join(workspace, "link-dir")); err != nil {
		t.Fatalf("create symlink directory: %v", err)
	}

	_, err := (&gitStatusTool{workspace: workspace}).Execute(context.Background(), Request{
		Paths: []string{"link-dir"},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("expected symlink cwd rejection, got %v", err)
	}
}

func TestRepoInspectRejectsSymlinkPathEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink scope checks use Unix symlink semantics")
	}
	workspace, outside := symlinkScopeHarness(t)
	if err := os.WriteFile(filepath.Join(outside, "outside.txt"), []byte("outside"), 0o644); err != nil {
		t.Fatalf("seed outside file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, "link-dir")); err != nil {
		t.Fatalf("create symlink directory: %v", err)
	}

	_, err := (&repoInspectTool{workspace: workspace}).Execute(context.Background(), Request{
		Paths: []string{"link-dir"},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("expected symlink inspect rejection, got %v", err)
	}
}
