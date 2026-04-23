package modelruntime

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFakeBackend_LoadGenerateUnload(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true, MaxOutputTokens: 8})
	manifest := ModelManifest{ID: "fake-1", Backend: BackendFake, Format: ModelFormatGGUF}

	loaded, err := backend.Load(context.Background(), manifest)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.ModelID != "fake-1" || loaded.Status != StatusLoaded {
		t.Fatalf("unexpected loaded model: %+v", loaded)
	}

	res, err := backend.Generate(context.Background(), GenerateRequest{
		ModelID: "fake-1",
		Prompt:  "alpha beta gamma delta epsilon zeta eta theta iota",
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if res.Backend != BackendFake {
		t.Fatalf("expected fake backend, got %s", res.Backend)
	}
	if res.ModelID != "fake-1" {
		t.Fatalf("unexpected model id: %s", res.ModelID)
	}
	if len(strings.Fields(res.Content)) > 8 {
		t.Fatalf("expected bounded output, got %q", res.Content)
	}
	if res.FinishReason != "length" {
		t.Fatalf("expected finish reason length, got %s", res.FinishReason)
	}

	if err := backend.Unload(context.Background(), "fake-1"); err != nil {
		t.Fatalf("unload: %v", err)
	}
	if err := backend.Unload(context.Background(), "fake-1"); !errors.Is(err, ErrModelNotLoaded) {
		t.Fatalf("expected ErrModelNotLoaded, got %v", err)
	}
}

func TestFakeBackend_InspectAndHealth(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true})
	if _, err := backend.Load(context.Background(), ModelManifest{ID: "m-1"}); err != nil {
		t.Fatalf("load: %v", err)
	}

	inspect, err := backend.Inspect(context.Background(), "m-1")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !inspect.Found {
		t.Fatalf("expected loaded model to be found")
	}

	health, err := backend.Health(context.Background())
	if err != nil {
		t.Fatalf("health returned error: %v", err)
	}
	if !health.Healthy {
		t.Fatalf("expected backend to be healthy")
	}
	if health.Meta["loaded"].(int) != 1 {
		t.Fatalf("expected loaded count 1, got %+v", health.Meta)
	}
}

func TestFakeBackend_CustomGenerate(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{
		Healthy: true,
		Generate: func(req GenerateRequest) (GenerateResult, error) {
			if req.ModelID != "m-2" {
				return GenerateResult{}, errors.New("unexpected model")
			}
			return GenerateResult{Content: "custom", FinishReason: "stop"}, nil
		},
	})
	if _, err := backend.Load(context.Background(), ModelManifest{ID: "m-2"}); err != nil {
		t.Fatalf("load: %v", err)
	}
	res, err := backend.Generate(context.Background(), GenerateRequest{ModelID: "m-2", Prompt: "hello"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if res.Content != "custom" {
		t.Fatalf("expected custom output, got %q", res.Content)
	}
}
