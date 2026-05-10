package gateway

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestCapabilityFilesystemReadSurfacesRejectSymlinkPathEscape(t *testing.T) {
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

	for _, name := range []string{"get_permissions", "watch_path"} {
		t.Run(name, func(t *testing.T) {
			tool := &capabilityBackingTool{
				workspace: workspace,
				capability: domain.ToolCapability{
					ID:     "filesystem." + name,
					Domain: "filesystem",
					Name:   name,
				},
			}
			_, err := tool.executeFilesystem(context.Background(), Request{Paths: []string{"link-file"}})
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
				t.Fatalf("expected symlink path rejection, got %v", err)
			}
		})
	}
}

func TestSearchWorkspaceFilesDoesNotReadSymlinkTargets(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink scope checks use Unix symlink semantics")
	}
	workspace, outside := symlinkScopeHarness(t)
	outsideFile := filepath.Join(outside, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("needle-from-outside"), 0o644); err != nil {
		t.Fatalf("seed outside file: %v", err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(workspace, "link-file")); err != nil {
		t.Fatalf("create symlink file: %v", err)
	}

	rows, err := searchWorkspaceFiles(workspace, "needle-from-outside", 20)
	if err != nil {
		t.Fatalf("search workspace files: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no symlink target content matches, got %#v", rows)
	}
}

func TestReadCapabilityFileBoundedRejectsOversizeFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "large.txt")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), gatewayWorkspaceSearchReadLimit+1), 0o644); err != nil {
		t.Fatalf("seed large file: %v", err)
	}

	if _, err := readCapabilityFileBounded(path, "workspace search file", gatewayWorkspaceSearchReadLimit); err == nil || !strings.Contains(err.Error(), "workspace search file too large") {
		t.Fatalf("readCapabilityFileBounded error = %v, want size error", err)
	}
}

func TestSearchWorkspaceFilesSkipsOversizeContent(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	path := filepath.Join(workspace, "large.txt")
	body := append(bytes.Repeat([]byte("x"), gatewayWorkspaceSearchReadLimit), []byte("needle-in-large-file")...)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("seed large workspace file: %v", err)
	}

	rows, err := searchWorkspaceFiles(workspace, "needle-in-large-file", 20)
	if err != nil {
		t.Fatalf("search workspace files: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no oversized file content matches, got %#v", rows)
	}
}

func TestEncryptionKeyRejectsOversizeExistingKeyFile(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "master.key")
	if err := os.WriteFile(keyPath, bytes.Repeat([]byte("x"), gatewayEncryptionKeyReadLimit+1), 0o600); err != nil {
		t.Fatalf("seed oversized key file: %v", err)
	}
	t.Setenv("FORGE_ENCRYPTION_KEY_PATH", keyPath)
	tool := &capabilityBackingTool{dataDir: t.TempDir()}

	if _, err := tool.encryptionKey(); err == nil || !strings.Contains(err.Error(), "encryption key file too large") {
		t.Fatalf("encryptionKey error = %v, want size error", err)
	}
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if info.Size() != int64(gatewayEncryptionKeyReadLimit+1) {
		t.Fatalf("key file size = %d, want original oversized file preserved", info.Size())
	}
}
