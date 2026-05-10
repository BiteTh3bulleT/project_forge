package modelruntime

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeJSONMetadataReadsRejectOversizeFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.json")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), maxRuntimeJSONFileBytes+1), 0o644); err != nil {
		t.Fatalf("write large json fixture: %v", err)
	}

	if _, err := ReadManifest(path); err == nil || !strings.Contains(err.Error(), "model manifest too large") {
		t.Fatalf("ReadManifest error = %v, want size error", err)
	}
	if _, err := readModelState(path); err == nil || !strings.Contains(err.Error(), "model state too large") {
		t.Fatalf("readModelState error = %v, want size error", err)
	}
	if _, err := readChecksumsFile(path); err == nil || !strings.Contains(err.Error(), "checksums file too large") {
		t.Fatalf("readChecksumsFile error = %v, want size error", err)
	}
}
