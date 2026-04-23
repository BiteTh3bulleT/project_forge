package modelruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestServiceModelManagementAndSelection(t *testing.T) {
	modelHome := t.TempDir()
	store := NewModelStore(modelHome, ModelStoreOptions{StrictChecksum: true})
	if _, err := store.ensureNamedRoot("models"); err != nil {
		t.Fatalf("ensure models root: %v", err)
	}

	alphaFile := filepath.Join(t.TempDir(), "alpha-q4.gguf")
	betaFile := filepath.Join(t.TempDir(), "beta-q4.gguf")
	if err := os.WriteFile(alphaFile, []byte("alpha-bytes"), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}
	if err := os.WriteFile(betaFile, []byte("beta-bytes"), 0o644); err != nil {
		t.Fatalf("write beta file: %v", err)
	}
	alpha, err := store.Import(context.Background(), alphaFile, ImportModelOptions{ID: "alpha", Preferred: true, Backend: BackendFake})
	if err != nil {
		t.Fatalf("import alpha: %v", err)
	}
	_, err = store.Import(context.Background(), betaFile, ImportModelOptions{ID: "beta", Backend: BackendFake})
	if err != nil {
		t.Fatalf("import beta: %v", err)
	}

	registry := NewModelRegistry(store)
	if _, err := registry.Scan(context.Background()); err != nil {
		t.Fatalf("scan registry: %v", err)
	}
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true, Kind: BackendFake})
	svc, err := NewService(ServiceOptions{Backends: []ModelBackend{backend}, Registry: registry, DefaultModelID: "", AutoLoad: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := svc.Generate(context.Background(), GenerateRequest{Prompt: "hello", WorkspaceID: "ws-test", Actor: "tester", Source: "unit"})
	if err != nil {
		t.Fatalf("generate with preferred/default selection: %v", err)
	}
	if result.ModelID != alpha.Model.Manifest.ID {
		t.Fatalf("expected preferred alpha model to win selection, got %+v", result)
	}

	if _, err := svc.DisableModel(context.Background(), "alpha", ManagementRequestMeta{WorkspaceID: "ws-test", Actor: "tester", Source: "unit"}); err != nil {
		t.Fatalf("disable alpha: %v", err)
	}
	if _, err := svc.Generate(context.Background(), GenerateRequest{ModelID: "alpha", Prompt: "blocked", WorkspaceID: "ws-test", Actor: "tester", Source: "unit"}); !errors.Is(err, ErrModelUnavailable) {
		t.Fatalf("expected disabled model to be unavailable, got %v", err)
	}
	if _, err := svc.EnableModel(context.Background(), "alpha", ManagementRequestMeta{WorkspaceID: "ws-test", Actor: "tester", Source: "unit"}); err != nil {
		t.Fatalf("enable alpha: %v", err)
	}
	if _, err := svc.VerifyModel(context.Background(), "alpha", ManagementRequestMeta{WorkspaceID: "ws-test", Actor: "tester", Source: "unit"}); err != nil {
		t.Fatalf("verify alpha: %v", err)
	}
	if _, err := svc.ArchiveModel(context.Background(), "beta", ManagementRequestMeta{WorkspaceID: "ws-test", Actor: "tester", Source: "unit"}); err != nil {
		t.Fatalf("archive beta: %v", err)
	}
	if _, err := svc.Generate(context.Background(), GenerateRequest{ModelID: "beta", Prompt: "archived", WorkspaceID: "ws-test", Actor: "tester", Source: "unit"}); !errors.Is(err, ErrModelUnavailable) {
		t.Fatalf("expected archived model to be unavailable, got %v", err)
	}
	removed, err := svc.RemoveModelRegistration(context.Background(), "beta", ManagementRequestMeta{WorkspaceID: "ws-test", Actor: "tester", Source: "unit"})
	if err != nil {
		t.Fatalf("remove beta registration: %v", err)
	}
	if removed.RemovedPath == "" {
		t.Fatalf("expected removed path, got %+v", removed)
	}
	if _, err := svc.getModelInfo("beta"); !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected beta to be removed from registry, got %v", err)
	}

	usage, err := svc.Usage(context.Background())
	if err != nil {
		t.Fatalf("usage: %v", err)
	}
	if usage.Registered != 1 || usage.Archived != 0 {
		t.Fatalf("unexpected usage summary after removal: %+v", usage)
	}

	compat, err := svc.Compatibility(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("compatibility: %v", err)
	}
	if !compat.CanGenerate || !compat.BackendConfigured {
		t.Fatalf("unexpected compatibility report: %+v", compat)
	}
}
