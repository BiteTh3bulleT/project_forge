package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"forge/projectforge/services/core/internal/config"
)

func TestInitModelRuntimeServiceAutoEnablesForOpenAICompatEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "remote-qwen", "name": "Remote Qwen"},
			},
		})
	}))
	defer server.Close()

	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "false")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "false")
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", server.URL)
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_API_KEY", "")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_VLLM_API_KEY", "")
	modelHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(modelHome, "models"), 0o755); err != nil {
		t.Fatalf("create models root: %v", err)
	}
	t.Setenv("FORGE_MODEL_HOME", modelHome)
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "false")

	cfg := config.Load()
	svc := initModelRuntimeService(cfg, nil)
	if svc == nil {
		t.Fatalf("expected model runtime service to auto-enable when OpenAI-compatible endpoint is configured")
	}

	models, err := svc.ListModels(context.Background(), ModelRuntimeListRequest{})
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if !hasModelID(models, "remote-qwen") {
		t.Fatalf("expected discovered remote model in initial list, got=%#v", models)
	}

	if _, err := svc.ScanModels(context.Background(), ModelRuntimeControlRequest{}); err != nil {
		t.Fatalf("scan models: %v", err)
	}
	models, err = svc.ListModels(context.Background(), ModelRuntimeListRequest{})
	if err != nil {
		t.Fatalf("list models after scan: %v", err)
	}
	if !hasModelID(models, "remote-qwen") {
		t.Fatalf("expected discovered remote model to survive scan, got=%#v", models)
	}
}

func TestInitModelRuntimeServiceDisabledWithoutEnablement(t *testing.T) {
	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "false")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "false")
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "")

	cfg := config.Load()
	if svc := initModelRuntimeService(cfg, nil); svc != nil {
		t.Fatalf("expected model runtime service to remain disabled when no enablement path is configured")
	}
}

func TestInitModelRuntimeServiceSafeModeForcesCPUOnlyRuntimeHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "true")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "false")
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_DEFAULT_BACKEND", "llama_cpp")
	t.Setenv("FORGE_LLAMA_CPP_ENDPOINT", server.URL)
	t.Setenv("FORGE_GPU_ENABLED", "true")
	t.Setenv("FORGE_SAFE_MODE_FORCE_CPU_ONLY", "true")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "false")
	modelHome := t.TempDir()
	t.Setenv("FORGE_MODEL_HOME", modelHome)

	cfg := config.Load()
	svc := initModelRuntimeService(cfg, nil)
	if svc == nil {
		t.Fatalf("expected model runtime service when explicitly enabled")
	}

	health, err := svc.Health(context.Background(), ModelRuntimeRequestMeta{})
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if health.GPUAware {
		t.Fatalf("expected gpuAware=false when safe mode forces cpu-only")
	}
}

func TestInitModelRuntimeServicePreservesGPURequiredInteractivePolicyWhenGPUUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "remote-gpu-required", "name": "Remote GPU Required"},
				},
			})
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{
						"message":       map[string]any{"role": "assistant", "content": "unexpected success"},
						"finish_reason": "stop",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "true")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "false")
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", server.URL)
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_API_KEY", "")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "false")
	t.Setenv("FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD", "false")
	t.Setenv("FORGE_GPU_ENABLED", "false")
	t.Setenv("FORGE_GPU_REQUIRED_FOR_INTERACTIVE_INFERENCE", "true")
	modelHome := t.TempDir()
	t.Setenv("FORGE_MODEL_HOME", modelHome)

	svc := initModelRuntimeService(config.Load(), nil)
	if svc == nil {
		t.Fatalf("expected model runtime service")
	}
	if _, err := svc.LoadModel(context.Background(), "remote-gpu-required", ModelRuntimeControlRequest{Actor: "operator", Source: "test"}); err != nil {
		t.Fatalf("load model: %v", err)
	}

	_, err := svc.Chat(context.Background(), ModelRuntimeChatRequest{
		ModelID: "remote-gpu-required",
		Messages: []ModelRuntimeChatMessage{
			{Role: "user", Content: "hello"},
		},
		Actor:  "operator",
		Source: "test",
		Meta:   ModelRuntimeRequestMeta{WorkspaceID: "ws-test"},
	})
	runtimeErr, ok := err.(modelRuntimeCodeCarrier)
	if !ok || runtimeErr.ErrorCode() != "MODEL_GPU_INTERACTIVE_REQUIRED" {
		t.Fatalf("expected MODEL_GPU_INTERACTIVE_REQUIRED, got %T %v", err, err)
	}
}

func hasModelID(models []ModelRuntimeModel, id string) bool {
	for _, model := range models {
		if model.ID == id {
			return true
		}
	}
	return false
}
