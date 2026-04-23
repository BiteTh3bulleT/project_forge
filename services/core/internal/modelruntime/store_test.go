package modelruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelStoreScanListAndInspect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	modelHome := t.TempDir()

	modelID := "qwen-7b-q4"
	modelFile := mustWriteModelFile(t, modelHome, modelID, "model.gguf", "model payload")
	checksum := mustSHA256(t, modelFile)
	mustWriteManifest(t, modelHome, modelID, fmt.Sprintf(`{
	  "schemaVersion": "1",
	  "id": %q,
	  "displayName": "Qwen 7B Q4",
	  "family": "qwen",
	  "format": "gguf",
	  "backend": "llama_cpp",
	  "file": "model.gguf",
	  "sha256": %q,
	  "sizeBytes": %d,
	  "quantization": "Q4_K_M",
	  "contextLength": 32768,
	  "capabilities": ["chat", "completion"],
	  "defaultRuntime": {"maxTokens": 512},
	  "license": {"name": "Apache-2.0"},
	  "metadata": {}
	}`, modelID, checksum, len("model payload")))

	store := NewModelStore(modelHome, ModelStoreOptions{})
	records, err := store.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	rec := records[0]
	if rec.Manifest.ID != modelID {
		t.Fatalf("expected model id %q, got %q", modelID, rec.Manifest.ID)
	}
	if rec.ModelFilePath != modelFile {
		t.Fatalf("expected resolved model file %q, got %q", modelFile, rec.ModelFilePath)
	}
	if len(rec.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %+v", rec.Warnings)
	}

	loaded, err := store.Load(ctx, modelID)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.ManifestPath == "" {
		t.Fatalf("expected manifest path")
	}
}

func TestModelStoreChecksumMismatchWarningAndStrictMode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	modelHome := t.TempDir()
	modelID := "mismatch-model"
	mustWriteModelFile(t, modelHome, modelID, "model.gguf", "actual data")
	mustWriteManifest(t, modelHome, modelID, `{
	  "schemaVersion": "1",
	  "id": "mismatch-model",
	  "displayName": "Mismatch",
	  "family": "test",
	  "format": "gguf",
	  "backend": "llama_cpp",
	  "file": "model.gguf",
	  "sha256": "deadbeef",
	  "sizeBytes": 11,
	  "quantization": "Q4_0",
	  "contextLength": 1024,
	  "capabilities": ["chat"],
	  "defaultRuntime": {},
	  "license": {"name": "MIT"},
	  "metadata": {}
	}`)

	warnStore := NewModelStore(modelHome, ModelStoreOptions{StrictChecksum: false})
	records, err := warnStore.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan warning mode error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if len(records[0].Warnings) == 0 {
		t.Fatal("expected checksum warning in warning mode")
	}
	if !strings.Contains(records[0].Warnings[0], "checksum mismatch") {
		t.Fatalf("expected checksum mismatch warning, got %q", records[0].Warnings[0])
	}

	strictStore := NewModelStore(modelHome, ModelStoreOptions{StrictChecksum: true})
	_, err = strictStore.Scan(ctx)
	if err == nil {
		t.Fatal("expected strict checksum scan error")
	}
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}

func TestModelStoreScanRejectsInvalidManifest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	modelHome := t.TempDir()
	modelID := "bad-format"
	mustWriteModelFile(t, modelHome, modelID, "model.bin", "x")
	mustWriteManifest(t, modelHome, modelID, `{
	  "schemaVersion": "1",
	  "id": "bad-format",
	  "displayName": "Bad",
	  "family": "bad",
	  "format": "not_real",
	  "backend": "llama_cpp",
	  "file": "model.bin",
	  "sha256": "",
	  "sizeBytes": 1,
	  "quantization": "Q4_0",
	  "contextLength": 1024,
	  "capabilities": ["chat"],
	  "defaultRuntime": {},
	  "license": {"name": "MIT"},
	  "metadata": {}
	}`)

	store := NewModelStore(modelHome, ModelStoreOptions{})
	_, err := store.Scan(ctx)
	if err == nil {
		t.Fatal("expected scan error for invalid manifest")
	}
	if !errors.Is(err, ErrUnsupportedModelFormat) {
		t.Fatalf("expected ErrUnsupportedModelFormat, got %v", err)
	}
}

func mustWriteModelFile(t *testing.T, modelHome, modelID, fileName, content string) string {
	t.Helper()
	modelDir := filepath.Join(modelHome, "models", modelID)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatalf("mkdir model dir: %v", err)
	}
	modelPath := filepath.Join(modelDir, fileName)
	if err := os.WriteFile(modelPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}
	return modelPath
}

func mustWriteManifest(t *testing.T, modelHome, modelID, body string) string {
	t.Helper()
	manifestPath := filepath.Join(modelHome, "models", modelID, ManifestFilename)
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return manifestPath
}

func mustSHA256(t *testing.T, path string) string {
	t.Helper()
	hash, err := fileSHA256(path)
	if err != nil {
		t.Fatalf("file sha256: %v", err)
	}
	return hash
}
