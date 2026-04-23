package modelruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestModelRegistryScanListInspectAndStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	modelHome := t.TempDir()

	mustWriteModelFile(t, modelHome, "alpha", "model.gguf", "alpha-model")
	mustWriteManifest(t, modelHome, "alpha", `{
	  "schemaVersion": "1",
	  "id": "alpha",
	  "displayName": "Alpha",
	  "family": "test",
	  "format": "gguf",
	  "backend": "llama_cpp",
	  "file": "model.gguf",
	  "sha256": "",
	  "sizeBytes": 11,
	  "quantization": "Q4_0",
	  "contextLength": 4096,
	  "capabilities": ["chat"],
	  "defaultRuntime": {},
	  "license": {"name": "MIT"},
	  "metadata": {}
	}`)

	mustWriteModelFile(t, modelHome, "beta", "model.gguf", "beta-model")
	mustWriteManifest(t, modelHome, "beta", `{
	  "schemaVersion": "1",
	  "id": "beta",
	  "displayName": "Beta",
	  "family": "test",
	  "format": "gguf",
	  "backend": "llama_cpp",
	  "file": "model.gguf",
	  "sha256": "",
	  "sizeBytes": 10,
	  "quantization": "Q4_0",
	  "contextLength": 4096,
	  "capabilities": ["chat"],
	  "defaultRuntime": {},
	  "license": {"name": "MIT"},
	  "metadata": {}
	}`)

	store := NewModelStore(modelHome, ModelStoreOptions{})
	registry := NewModelRegistry(store)

	list, err := registry.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 models, got %d", len(list))
	}
	if list[0].Manifest.ID != "alpha" || list[1].Manifest.ID != "beta" {
		t.Fatalf("expected sorted list [alpha, beta], got [%s, %s]", list[0].Manifest.ID, list[1].Manifest.ID)
	}

	model, err := registry.Inspect("beta")
	if err != nil {
		t.Fatalf("Inspect error: %v", err)
	}
	if model.Manifest.ID != "beta" {
		t.Fatalf("expected inspect beta, got %s", model.Manifest.ID)
	}
	if model.Status != ModelStatusAvailable {
		t.Fatalf("expected available status after scan, got %s", model.Status)
	}

	if err := registry.UpdateStatus("beta", ModelStatusLoaded); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	updated, ok := registry.Get("beta")
	if !ok {
		t.Fatal("expected beta in registry")
	}
	if updated.Status != ModelStatusLoaded {
		t.Fatalf("expected loaded status, got %s", updated.Status)
	}
}

func TestModelRegistryRegisterAndValidateManifest(t *testing.T) {
	t.Parallel()

	registry := NewModelRegistry(NewModelStore(t.TempDir(), ModelStoreOptions{}))
	valid := ModelManifest{
		SchemaVersion: "1",
		ID:            "gamma",
		DisplayName:   "Gamma",
		Family:        "test",
		Format:        ModelFormatGGUF,
		Backend:       ModelBackendFake,
		FilePath:      "model.gguf",
		SHA256:        "",
		SizeBytes:     0,
		Quantization:  "Q4_0",
		ContextLength: 2048,
		Capabilities:  []ModelCapability{ModelCapabilityChat},
		License:       "MIT",
		Metadata:      map[string]any{},
	}

	if err := registry.Register(valid); err != nil {
		t.Fatalf("Register valid manifest error: %v", err)
	}
	stored, ok := registry.Get("gamma")
	if !ok {
		t.Fatal("expected gamma to be registered")
	}
	if stored.Status != ModelStatusAvailable {
		t.Fatalf("expected default status available, got %s", stored.Status)
	}

	invalid := valid
	invalid.ID = "bad"
	invalid.Format = ModelFormatUnknown
	if err := registry.Register(invalid); err == nil {
		t.Fatal("expected invalid manifest registration error")
	} else if !errors.Is(err, ErrUnsupportedModelFormat) {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestModelRegistryScanPreservesDiscoveredModels(t *testing.T) {
	t.Parallel()

	modelHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(modelHome, "models"), 0o755); err != nil {
		t.Fatalf("create models root: %v", err)
	}
	registry := NewModelRegistry(NewModelStore(modelHome, ModelStoreOptions{}))
	discovered := ModelManifest{
		SchemaVersion: "forge.model/v1",
		ID:            "remote-qwen",
		DisplayName:   "Remote Qwen",
		Family:        "openai_compat-remote",
		Format:        ModelFormatGGUF,
		Backend:       BackendOpenAICompat,
		FilePath:      "remote/remote-qwen",
		Quantization:  "remote",
		ContextLength: 4096,
		Capabilities:  []ModelCapability{ModelCapabilityChat, ModelCapabilityCompletion},
		License:       "remote",
		Metadata: map[string]any{
			"source":     "http://127.0.0.1:9000",
			"discovered": true,
		},
	}
	if err := registry.Register(discovered); err != nil {
		t.Fatalf("register discovered model: %v", err)
	}

	list, err := registry.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan with discovered model: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected discovered model to survive scan, got %d entries", len(list))
	}
	if list[0].Manifest.ID != discovered.ID {
		t.Fatalf("expected discovered model id %q, got %q", discovered.ID, list[0].Manifest.ID)
	}
}
