package modelruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelStoreImportVerifyArchiveAndRemove(t *testing.T) {
	modelHome := t.TempDir()
	store := NewModelStore(modelHome, ModelStoreOptions{StrictChecksum: true})
	if _, err := store.ensureNamedRoot("models"); err != nil {
		t.Fatalf("ensure models root: %v", err)
	}

	sourceFile := filepath.Join(t.TempDir(), "mistral-q4.gguf")
	if err := os.WriteFile(sourceFile, []byte("gguf-bytes"), 0o644); err != nil {
		t.Fatalf("write source gguf: %v", err)
	}

	imported, err := store.Import(context.Background(), sourceFile, ImportModelOptions{DisplayName: "Mistral Q4", Preferred: true})
	if err != nil {
		t.Fatalf("import file: %v", err)
	}
	if imported.Model.Manifest.Format != ModelFormatGGUF || imported.Model.State.Status != StatusImported {
		t.Fatalf("unexpected imported model: %+v", imported.Model)
	}
	if !strings.HasSuffix(imported.Model.ModelFilePath, ".gguf") {
		t.Fatalf("expected managed gguf path, got %s", imported.Model.ModelFilePath)
	}

	duplicate, err := store.Import(context.Background(), sourceFile, ImportModelOptions{})
	if err != nil {
		t.Fatalf("duplicate import: %v", err)
	}
	if !duplicate.Duplicate || duplicate.Model.Manifest.ID != imported.Model.Manifest.ID {
		t.Fatalf("expected deterministic duplicate import, got %+v", duplicate)
	}

	verified, err := store.Verify(context.Background(), imported.Model.Manifest.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if verified.State.Status != StatusVerified {
		t.Fatalf("expected verified status, got %+v", verified.State)
	}

	archived, err := store.Archive(context.Background(), imported.Model.Manifest.ID)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if !archived.Archived || archived.State.Status != StatusArchived {
		t.Fatalf("expected archived model, got %+v", archived)
	}

	removedPath, err := store.RemoveRegistration(context.Background(), imported.Model.Manifest.ID)
	if err != nil {
		t.Fatalf("remove registration: %v", err)
	}
	if _, err := os.Stat(removedPath); err != nil {
		t.Fatalf("expected removed path to exist, got %v", err)
	}
}

func TestModelStoreImportManifestBackedDirectory(t *testing.T) {
	modelHome := t.TempDir()
	store := NewModelStore(modelHome, ModelStoreOptions{StrictChecksum: true})
	if _, err := store.ensureNamedRoot("models"); err != nil {
		t.Fatalf("ensure models root: %v", err)
	}

	sourceDir := filepath.Join(t.TempDir(), "manifest-model")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	modelFile := filepath.Join(sourceDir, "model.gguf")
	if err := os.WriteFile(modelFile, []byte("manifest-backed"), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}
	checksum, err := fileSHA256(modelFile)
	if err != nil {
		t.Fatalf("hash model file: %v", err)
	}
	manifest := ModelManifest{
		SchemaVersion: "forge.model/v1",
		ID:            "manifest-model",
		DisplayName:   "Manifest Model",
		Family:        "manifest",
		Format:        ModelFormatGGUF,
		Backend:       BackendLlamaCpp,
		FilePath:      "model.gguf",
		SHA256:        checksum,
		SizeBytes:     int64(len("manifest-backed")),
		Quantization:  "q4",
		ContextLength: 4096,
		Capabilities:  []ModelCapability{CapabilityChat, CapabilityCompletion},
		License:       "apache-2.0",
	}
	if err := writeManifest(filepath.Join(sourceDir, ManifestFilename), manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	result, err := store.Import(context.Background(), sourceDir, ImportModelOptions{})
	if err != nil {
		t.Fatalf("import directory: %v", err)
	}
	if result.Model.Manifest.ID != "manifest-model" || result.Model.State.Status != StatusImported {
		t.Fatalf("unexpected imported directory model: %+v", result.Model)
	}
	if result.Model.Manifest.Metadata["sourcePath"] != sourceDir {
		t.Fatalf("expected sourcePath metadata to be preserved, got %+v", result.Model.Manifest.Metadata)
	}
}

func TestModelStoreImportManifestBackedDirectoryRejectsUnsafeIDBeforeCopy(t *testing.T) {
	modelHome := t.TempDir()
	store := NewModelStore(modelHome, ModelStoreOptions{StrictChecksum: true})
	if _, err := store.ensureNamedRoot("models"); err != nil {
		t.Fatalf("ensure models root: %v", err)
	}

	sourceDir := filepath.Join(t.TempDir(), "unsafe-manifest-model")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	modelFile := filepath.Join(sourceDir, "model.gguf")
	if err := os.WriteFile(modelFile, []byte("unsafe-id"), 0o644); err != nil {
		t.Fatalf("write model file: %v", err)
	}
	checksum, err := fileSHA256(modelFile)
	if err != nil {
		t.Fatalf("hash model file: %v", err)
	}
	manifest := ModelManifest{
		SchemaVersion: "forge.model/v1",
		ID:            "../escaped-model",
		DisplayName:   "Unsafe Manifest Model",
		Family:        "manifest",
		Format:        ModelFormatGGUF,
		Backend:       BackendLlamaCpp,
		FilePath:      "model.gguf",
		SHA256:        checksum,
		SizeBytes:     int64(len("unsafe-id")),
		Quantization:  "q4",
		ContextLength: 4096,
		Capabilities:  []ModelCapability{CapabilityChat, CapabilityCompletion},
		License:       "apache-2.0",
	}
	if err := writeManifest(filepath.Join(sourceDir, ManifestFilename), manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err = store.Import(context.Background(), sourceDir, ImportModelOptions{})
	if err == nil {
		t.Fatal("expected unsafe manifest id import to fail")
	}
	if !strings.Contains(err.Error(), "model id") {
		t.Fatalf("expected model id error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(modelHome, "escaped-model")); !os.IsNotExist(statErr) {
		t.Fatalf("unsafe import created escaped destination, stat err=%v", statErr)
	}
}
