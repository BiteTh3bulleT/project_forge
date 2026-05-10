package projectcontext

import (
	"bytes"
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
