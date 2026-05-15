package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/modelruntime"
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

func TestInitModelRuntimeServiceAutoEnablesForVLLMBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "vllm-qwen", "name": "vLLM Qwen"},
			},
		})
	}))
	defer server.Close()

	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "false")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "false")
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", "")
	t.Setenv("FORGE_VLLM_BASE_URL", server.URL)
	t.Setenv("FORGE_VLLM_API_KEY", "")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "false")
	t.Setenv("FORGE_MODEL_HOME", t.TempDir())

	svc := initModelRuntimeService(config.Load(), nil)
	if svc == nil {
		t.Fatalf("expected model runtime service to auto-enable when vLLM base URL is configured")
	}
	models, err := svc.ListModels(context.Background(), ModelRuntimeListRequest{})
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if !hasModelID(models, "vllm-qwen") {
		t.Fatalf("expected discovered vLLM model in initial list, got=%#v", models)
	}
	backends, err := svc.Backends(context.Background(), ModelRuntimeRequestMeta{})
	if err != nil {
		t.Fatalf("backends: %v", err)
	}
	for _, backend := range backends {
		if backend.Kind == "vllm" {
			if backend.Meta["profile"] != "interactive_vllm" {
				t.Fatalf("expected vLLM backend profile metadata, got=%#v", backend.Meta)
			}
			return
		}
	}
	t.Fatalf("expected vLLM backend status, got=%#v", backends)
}

func TestModelRuntimeDiscoveryRejectsOversizeResponses(t *testing.T) {
	t.Run("ollama", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/tags" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(make([]byte, modelRuntimeDiscoveryResponseLimit+1))
		}))
		defer server.Close()

		_, err := discoverLocalOllamaModels(context.Background(), server.URL, false)
		if err == nil || !strings.Contains(err.Error(), "response too large") {
			t.Fatalf("discoverLocalOllamaModels error = %v, want size error", err)
		}
	})

	t.Run("openai compat", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/models" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(make([]byte, modelRuntimeDiscoveryResponseLimit+1))
		}))
		defer server.Close()

		_, err := discoverOpenAICompatibleEndpoint(context.Background(), server.URL, "", modelruntime.BackendOpenAICompat)
		if err == nil || !strings.Contains(err.Error(), "response too large") {
			t.Fatalf("discoverOpenAICompatibleEndpoint error = %v, want size error", err)
		}
	})
}

func TestMapModelRuntimeBridgeErrorRejectsUnsafeModelIDAsBadRequest(t *testing.T) {
	err := mapModelRuntimeBridgeError(modelruntime.ErrModelIDInvalid)
	runtimeErr, ok := err.(*modelRuntimeError)
	if !ok {
		t.Fatalf("mapped error=%T %[1]v, want *modelRuntimeError", err)
	}
	if runtimeErr.StatusCode() != http.StatusBadRequest {
		t.Fatalf("status=%d want %d", runtimeErr.StatusCode(), http.StatusBadRequest)
	}
	if runtimeErr.ErrorCode() != "MODEL_ID_INVALID" {
		t.Fatalf("code=%q want MODEL_ID_INVALID", runtimeErr.ErrorCode())
	}
}

func TestInitModelRuntimeServiceDiscoversLocalOllamaModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{
					{
						"name":  "llama3.2:latest",
						"model": "llama3.2:latest",
						"size":  1234,
						"details": map[string]any{
							"family":             "llama",
							"format":             "gguf",
							"quantization_level": "Q4_K_M",
							"parameter_size":     "3.2B",
						},
					},
					{
						"name":        "qwen3-coder:480b-cloud",
						"model":       "qwen3-coder:480b-cloud",
						"remote_host": "https://ollama.com:443",
						"size":        382,
						"details": map[string]any{
							"family":             "qwen3moe",
							"format":             "",
							"quantization_level": "BF16",
						},
					},
				},
			})
		case "/v1/chat/completions":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["model"] != "llama3.2:latest" {
				t.Errorf("unexpected model in chat request: %#v", body["model"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{
						"message":       map[string]any{"role": "assistant", "content": "ollama ok"},
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
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "false")
	t.Setenv("OLLAMA_BASE_URL", server.URL)
	t.Setenv("FORGE_MODEL_HOME", t.TempDir())

	svc := initModelRuntimeService(config.Load(), nil)
	if svc == nil {
		t.Fatalf("expected model runtime service")
	}
	models, err := svc.ListModels(context.Background(), ModelRuntimeListRequest{})
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if !hasModelID(models, "llama3.2:latest") {
		t.Fatalf("expected local ollama model in list, got=%#v", models)
	}
	if hasModelID(models, "qwen3-coder:480b-cloud") {
		t.Fatalf("expected remote ollama cloud model to be excluded, got=%#v", models)
	}
	if _, err := svc.ScanModels(context.Background(), ModelRuntimeControlRequest{}); err != nil {
		t.Fatalf("scan models should tolerate empty model home and preserve discovered ollama models: %v", err)
	}
	result, err := svc.Chat(context.Background(), ModelRuntimeChatRequest{
		ModelID: "llama3.2:latest",
		Messages: []ModelRuntimeChatMessage{
			{Role: "user", Content: "hello"},
		},
		Actor:  "operator",
		Source: "test",
		Meta:   ModelRuntimeRequestMeta{WorkspaceID: "ws-test"},
	})
	if err != nil {
		t.Fatalf("chat through discovered ollama model: %v", err)
	}
	if result.Content != "ollama ok" || result.Backend != "ollama_compat" {
		t.Fatalf("unexpected chat result: %#v", result)
	}
}

func TestModelRuntimeListRefreshDiscoversNewLocalOllamaModels(t *testing.T) {
	ollamaReady := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			if !ollamaReady {
				_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{
					{
						"name":  "phi4-mini:latest",
						"model": "phi4-mini:latest",
						"size":  int64(2048),
						"details": map[string]any{
							"family":             "phi",
							"parameter_size":     "3.8B",
							"quantization_level": "Q4_K_M",
							"format":             "gguf",
						},
					},
				},
			})
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{
						"message":       map[string]any{"role": "assistant", "content": "local loop ok"},
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
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_DEFAULT_BACKEND", "ollama_compat")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "false")
	t.Setenv("OLLAMA_BASE_URL", server.URL)
	t.Setenv("FORGE_MODEL_HOME", t.TempDir())

	svc := initModelRuntimeService(config.Load(), nil)
	if svc == nil {
		t.Fatalf("expected model runtime service")
	}
	models, err := svc.ListModels(context.Background(), ModelRuntimeListRequest{})
	if err != nil {
		t.Fatalf("initial list models: %v", err)
	}
	if hasModelID(models, "phi4-mini:latest") {
		t.Fatalf("did not expect model before local ollama reported it, got=%#v", models)
	}

	ollamaReady = true
	models, err = svc.ListModels(context.Background(), ModelRuntimeListRequest{})
	if err != nil {
		t.Fatalf("refreshed list models: %v", err)
	}
	if !hasModelID(models, "phi4-mini:latest") {
		t.Fatalf("expected refreshed list to discover newly pulled local ollama model, got=%#v", models)
	}
	result, err := svc.Chat(context.Background(), ModelRuntimeChatRequest{
		ModelID: "phi4-mini:latest",
		Messages: []ModelRuntimeChatMessage{
			{Role: "user", Content: "ping"},
		},
		Actor:  "operator",
		Source: "test",
		Meta:   ModelRuntimeRequestMeta{WorkspaceID: "ws-test"},
	})
	if err != nil {
		t.Fatalf("chat through refreshed ollama model: %v", err)
	}
	if result.Content != "local loop ok" || result.Backend != "ollama_compat" {
		t.Fatalf("unexpected refreshed chat result: %#v", result)
	}
}

func TestDockerHostGatewayIsTreatedAsLocalOllamaProvider(t *testing.T) {
	if !isLocalHTTPProvider("http://host.docker.internal:11434") {
		t.Fatalf("expected Docker host gateway endpoint to be treated as local Ollama")
	}
}

func TestInitModelRuntimeServiceCanExposeOllamaCloudModelsWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{
					"name":        "qwen3-coder:480b-cloud",
					"model":       "qwen3-coder:480b-cloud",
					"remote_host": "https://ollama.com:443",
					"size":        382,
					"details": map[string]any{
						"family":             "qwen3moe",
						"format":             "",
						"quantization_level": "BF16",
					},
				},
			},
		})
	}))
	defer server.Close()

	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "true")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "false")
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "false")
	t.Setenv("FORGE_MODELRUNTIME_ALLOW_OLLAMA_CLOUD_MODELS", "true")
	t.Setenv("OLLAMA_BASE_URL", server.URL)
	t.Setenv("FORGE_MODEL_HOME", t.TempDir())

	svc := initModelRuntimeService(config.Load(), nil)
	if svc == nil {
		t.Fatalf("expected model runtime service")
	}
	models, err := svc.ListModels(context.Background(), ModelRuntimeListRequest{})
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if !hasModelID(models, "qwen3-coder:480b-cloud") {
		t.Fatalf("expected cloud ollama model when explicitly enabled, got=%#v", models)
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
