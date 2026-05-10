package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestAddSourceRejectsPathOutsideWorkspace(t *testing.T) {
	dataDir := t.TempDir()
	workspaceDir := filepath.Join(dataDir, "workspace")
	outsideDir := filepath.Join(dataDir, "outside")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outsideDir, "leak.md"), []byte("outside workspace"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: workspaceDir})
	t.Cleanup(func() { srv.ShutdownWatch() })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sources", bytes.NewBufferString(`{"path":`+quoteJSON(outsideDir)+`}`))
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "outside active read scope") {
		t.Fatalf("body=%q, want read scope rejection", rr.Body.String())
	}
}

func TestAddSourceRejectsSymlinkedPathEscapingWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on some Windows systems")
	}
	dataDir := t.TempDir()
	workspaceDir := filepath.Join(dataDir, "workspace")
	outsideDir := filepath.Join(dataDir, "outside")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	linkPath := filepath.Join(workspaceDir, "outside-link")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: workspaceDir})
	t.Cleanup(func() { srv.ShutdownWatch() })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sources", bytes.NewBufferString(`{"path":`+quoteJSON(linkPath)+`}`))
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rr.Code, rr.Body.String())
	}
}

func TestAddSourceAcceptsWorkspacePath(t *testing.T) {
	dataDir := t.TempDir()
	workspaceDir := filepath.Join(dataDir, "workspace")
	sourceDir := filepath.Join(workspaceDir, "docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "note.md"), []byte("workspace note"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: workspaceDir})
	t.Cleanup(func() { srv.ShutdownWatch() })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sources", bytes.NewBufferString(`{"path":`+quoteJSON(sourceDir)+`}`))
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", rr.Code, rr.Body.String())
	}
}

func quoteJSON(s string) string {
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}
