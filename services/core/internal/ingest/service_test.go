package ingest

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/events"
	"forge/projectforge/services/core/internal/store"
)

func TestIndexSourceRejectsRootOutsideConfiguredScope(t *testing.T) {
	dataDir := t.TempDir()
	workspaceDir := filepath.Join(dataDir, "workspace")
	outsideDir := filepath.Join(dataDir, "outside")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := New(st.DB, events.New(st.DB), DefaultExtensionsCSV())
	svc.SetRootScope(workspaceDir)
	if err := svc.IndexSource(context.Background(), 1, outsideDir); err == nil {
		t.Fatal("expected outside root to be rejected")
	}
}

func TestIndexSourceSkipsFileSymlinkEscapingRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on some Windows systems")
	}
	dataDir := t.TempDir()
	workspaceDir := filepath.Join(dataDir, "workspace")
	sourceDir := filepath.Join(workspaceDir, "docs")
	outsideDir := filepath.Join(dataDir, "outside")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.md"), []byte("outside secret"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.Symlink(filepath.Join(outsideDir, "secret.md"), filepath.Join(sourceDir, "secret-link.md")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if _, err := st.DB.ExecContext(context.Background(), `INSERT INTO sources (id, path, created_at) VALUES (?,?,?)`, 1, sourceDir, int64(1)); err != nil {
		t.Fatalf("insert source: %v", err)
	}

	svc := New(st.DB, events.New(st.DB), DefaultExtensionsCSV())
	svc.SetRootScope(workspaceDir)
	if err := svc.IndexSource(context.Background(), 1, sourceDir); err != nil {
		t.Fatalf("index source: %v", err)
	}

	var count int
	if err := st.DB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM chunks WHERE content LIKE '%outside secret%'`).Scan(&count); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if count != 0 {
		t.Fatalf("indexed %d escaped symlink chunks, want 0", count)
	}
}

func TestIndexSourceAllowsScopedSourceWhenRootScopeIsFilesystemRoot(t *testing.T) {
	dataDir := t.TempDir()
	sourceDir := filepath.Join(dataDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "note.md"), []byte("root scope note"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if _, err := st.DB.ExecContext(context.Background(), `INSERT INTO sources (id, path, created_at) VALUES (?,?,?)`, 1, sourceDir, int64(1)); err != nil {
		t.Fatalf("insert source: %v", err)
	}

	svc := New(st.DB, events.New(st.DB), DefaultExtensionsCSV())
	svc.SetRootScope(string(filepath.Separator))
	if err := svc.IndexSource(context.Background(), 1, sourceDir); err != nil {
		t.Fatalf("index source with filesystem root scope: %v", err)
	}
}

func TestReadIngestFileRejectsOversizeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.md")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), maxIngestFileBytes+1), 0o644); err != nil {
		t.Fatalf("write large fixture: %v", err)
	}

	_, err := readIngestFile(path)
	if err == nil {
		t.Fatal("expected oversized ingest file to be rejected")
	}
	if !strings.Contains(err.Error(), "ingest file too large") {
		t.Fatalf("expected size error, got %v", err)
	}
}

func TestIndexSourceSkipsOversizeFileAndClearsExistingChunks(t *testing.T) {
	dataDir := t.TempDir()
	sourceDir := filepath.Join(dataDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	path := filepath.Join(sourceDir, "note.md")
	if err := os.WriteFile(path, []byte("indexed note"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if _, err := st.DB.ExecContext(context.Background(), `INSERT INTO sources (id, path, created_at) VALUES (?,?,?)`, 1, sourceDir, int64(1)); err != nil {
		t.Fatalf("insert source: %v", err)
	}

	svc := New(st.DB, events.New(st.DB), DefaultExtensionsCSV())
	svc.SetRootScope(dataDir)
	if err := svc.IndexSource(context.Background(), 1, sourceDir); err != nil {
		t.Fatalf("initial index source: %v", err)
	}

	var chunks int
	if err := st.DB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM chunks`).Scan(&chunks); err != nil {
		t.Fatalf("count initial chunks: %v", err)
	}
	if chunks == 0 {
		t.Fatal("expected initial chunks before oversize rewrite")
	}

	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), maxIngestFileBytes+1), 0o644); err != nil {
		t.Fatalf("rewrite oversized note: %v", err)
	}
	if err := svc.IndexSource(context.Background(), 1, sourceDir); err != nil {
		t.Fatalf("index source with oversize file: %v", err)
	}

	var files int
	if err := st.DB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM files WHERE rel_path = ?`, "note.md").Scan(&files); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if files != 0 {
		t.Fatalf("expected oversize file record to be cleared, got %d", files)
	}
	if err := st.DB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM chunks WHERE content LIKE '%indexed note%'`).Scan(&chunks); err != nil {
		t.Fatalf("count stale chunks: %v", err)
	}
	if chunks != 0 {
		t.Fatalf("expected stale chunks to be cleared, got %d", chunks)
	}
}
