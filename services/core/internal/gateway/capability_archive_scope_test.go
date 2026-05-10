package gateway

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestCapabilityArchiveRejectsSymlinkOutputParentEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink scope checks use Unix symlink semantics")
	}
	workspace, outside := symlinkScopeHarness(t)
	if err := os.WriteFile(filepath.Join(workspace, "inside.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatalf("seed inside file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, "link-dir")); err != nil {
		t.Fatalf("create symlink directory: %v", err)
	}
	tool := archiveCapabilityTool(workspace, t.TempDir())

	_, err := tool.createTarGZ(Request{
		Paths: []string{"inside.txt"},
		Input: map[string]any{"output": "link-dir/archive.tar.gz"},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("expected symlink output rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "archive.tar.gz")); !os.IsNotExist(err) {
		t.Fatalf("expected no archive outside workspace, stat err=%v", err)
	}
}

func TestCapabilityArchiveRejectsSymlinkInputPathEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink scope checks use Unix symlink semantics")
	}
	workspace, outside := symlinkScopeHarness(t)
	outsideFile := filepath.Join(outside, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0o644); err != nil {
		t.Fatalf("seed outside file: %v", err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(workspace, "link-file")); err != nil {
		t.Fatalf("create symlink file: %v", err)
	}
	tool := archiveCapabilityTool(workspace, t.TempDir())

	_, err := tool.createTarGZ(Request{
		Paths: []string{"link-file"},
		Input: map[string]any{"output": "archive.tar.gz"},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("expected symlink input rejection, got %v", err)
	}
}

func archiveCapabilityTool(workspace, dataDir string) *capabilityBackingTool {
	return &capabilityBackingTool{
		workspace: workspace,
		dataDir:   dataDir,
		capability: domain.ToolCapability{
			ID:     "filesystem.archive",
			Domain: "filesystem",
			Name:   "archive",
		},
	}
}
