package projectcontext

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadContextSourceRejectsOversizeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "FORGE_CONTEXT.md")
	body := bytes.Repeat([]byte("x"), maxContextSourceBytes+1)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write context fixture: %v", err)
	}

	_, err := readContextSource(path)
	if err == nil {
		t.Fatalf("expected oversize context source to be rejected")
	}
	if !strings.Contains(err.Error(), "context source too large") {
		t.Fatalf("expected size error, got %v", err)
	}
}

func TestReadContextSourceAllowsBoundedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "FORGE_CONTEXT.md")
	body := []byte("# FORGE\n\n- bounded context\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write context fixture: %v", err)
	}

	got, err := readContextSource(path)
	if err != nil {
		t.Fatalf("readContextSource returned error: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("unexpected body: %q", string(got))
	}
}

func TestResolveSourcePathRejectsOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	path := filepath.Join(outside, "FORGE_CONTEXT.md")
	if err := os.WriteFile(path, []byte("# outside\n"), 0o644); err != nil {
		t.Fatalf("write outside context: %v", err)
	}
	svc := New((*sql.DB)(nil), nil, workspace, t.TempDir())

	_, err := svc.resolveSourcePath(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "outside allowed roots") {
		t.Fatalf("expected outside root rejection, got %v", err)
	}
}

func TestResolveSourcePathAllowsConfiguredExtraRoot(t *testing.T) {
	workspace := t.TempDir()
	extra := t.TempDir()
	path := filepath.Join(extra, "FORGE_CONTEXT.md")
	if err := os.WriteFile(path, []byte("# extra\n"), 0o644); err != nil {
		t.Fatalf("write extra context: %v", err)
	}
	svc := New((*sql.DB)(nil), nil, workspace, t.TempDir())
	svc.SetAllowedRoots([]string{extra})

	got, err := svc.resolveSourcePath(context.Background(), path)
	if err != nil {
		t.Fatalf("expected extra root to be allowed: %v", err)
	}
	if got != path {
		t.Fatalf("expected path %q, got %q", path, got)
	}
}

func TestResolveSourcePathRejectsSymlinkEscape(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "FORGE_CONTEXT.md")
	if err := os.WriteFile(target, []byte("# outside\n"), 0o644); err != nil {
		t.Fatalf("write outside context: %v", err)
	}
	link := filepath.Join(workspace, "FORGE_CONTEXT.md")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	svc := New((*sql.DB)(nil), nil, workspace, t.TempDir())

	_, err := svc.resolveSourcePath(context.Background(), link)
	if err == nil || !strings.Contains(err.Error(), "outside allowed roots") {
		t.Fatalf("expected symlink escape rejection, got %v", err)
	}
}
