package artifacts

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestReadBoundedArtifactTextRejectsOversizeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), maxArtifactTextReadBytes+1), 0o644); err != nil {
		t.Fatalf("write large artifact fixture: %v", err)
	}

	_, err := readBoundedArtifactText(path)
	if err == nil {
		t.Fatal("expected oversized artifact text to be rejected")
	}
	if !strings.Contains(err.Error(), "artifact text too large") {
		t.Fatalf("expected size error, got %v", err)
	}
}

func TestReadArtifactTextRejectsOversizeArtifact(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := New(st.DB, dataDir)
	art, err := svc.CreateTextArtifact(context.Background(), CreateTextArtifactRequest{
		Type:     "test",
		Title:    "large",
		FileName: "large.txt",
		Content:  string(bytes.Repeat([]byte("x"), maxArtifactTextReadBytes+1)),
		MimeType: "text/plain",
	})
	if err != nil {
		t.Fatalf("create text artifact: %v", err)
	}

	_, _, textual, err := svc.ReadArtifactText(context.Background(), art.ID)
	if err == nil {
		t.Fatal("expected oversized artifact text read to fail")
	}
	if !textual {
		t.Fatal("expected artifact to still be classified as textual")
	}
	if !strings.Contains(err.Error(), "artifact text too large") {
		t.Fatalf("expected size error, got %v", err)
	}
}
