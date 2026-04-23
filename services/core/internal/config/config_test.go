package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWorkspaceToRoot(t *testing.T) {
	t.Setenv("FORGE_WORKSPACE_DIR", "")
	cfg := Load()
	if cfg.WorkspaceDir != "/" {
		t.Fatalf("expected default workspace '/', got %q", cfg.WorkspaceDir)
	}
}

func TestLoadRespectsWorkspaceOverride(t *testing.T) {
	t.Setenv("FORGE_WORKSPACE_DIR", "/tmp")
	cfg := Load()
	if cfg.WorkspaceDir != "/tmp" {
		t.Fatalf("expected workspace override '/tmp', got %q", cfg.WorkspaceDir)
	}
}

func TestLoadModelRuntimeDefaultsSafe(t *testing.T) {
	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "")
	t.Setenv("FORGE_MODEL_HOME", "")
	t.Setenv("FORGE_MODEL_DEFAULT_BACKEND", "")
	t.Setenv("FORGE_MODEL_DEFAULT_ID", "")
	t.Setenv("FORGE_LLAMA_CPP_ENDPOINT", "")
	t.Setenv("FORGE_LLAMA_CPP_BINARY_PATH", "")
	t.Setenv("FORGE_ALLOW_LLAMA_CPP_SPAWN", "")
	t.Setenv("FORGE_MODEL_MAX_PROMPT_TOKENS", "")
	t.Setenv("FORGE_MODEL_MAX_OUTPUT_TOKENS", "")
	t.Setenv("FORGE_MODEL_MAX_RESPONSE_BYTES", "")
	t.Setenv("FORGE_MODEL_REQUEST_TIMEOUT_MS", "")
	t.Setenv("FORGE_MODEL_LOAD_TIMEOUT_MS", "")
	t.Setenv("FORGE_MODEL_UNLOAD_TIMEOUT_MS", "")
	t.Setenv("FORGE_MODEL_IDLE_UNLOAD_MS", "")
	t.Setenv("FORGE_MODEL_MAX_LOADED_MODELS", "")
	t.Setenv("FORGE_MODEL_SCHEDULER_MAX_CONCURRENT", "")
	t.Setenv("FORGE_MODEL_SCHEDULER_QUEUE_CAPACITY", "")
	t.Setenv("FORGE_MODEL_SCHEDULER_DISPATCH_TIMEOUT_MS", "")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD", "")
	t.Setenv("FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD", "")
	t.Setenv("FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE", "")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "")

	cfg := Load()
	expectedModelHome, err := filepath.Abs(filepath.Join(cfg.DataDir, "models"))
	if err != nil {
		t.Fatalf("resolve default model home: %v", err)
	}

	if cfg.EnableModelRuntime {
		t.Fatalf("expected model runtime disabled by default")
	}
	if cfg.ModelHome != expectedModelHome {
		t.Fatalf("expected default model home %q, got %q", expectedModelHome, cfg.ModelHome)
	}
	if cfg.ModelDefaultBackend != "" {
		t.Fatalf("expected empty default backend, got %q", cfg.ModelDefaultBackend)
	}
	if cfg.ModelDefaultID != "" {
		t.Fatalf("expected empty default model id, got %q", cfg.ModelDefaultID)
	}
	if cfg.ModelLlamaCppEndpoint != "" {
		t.Fatalf("expected empty llama.cpp endpoint, got %q", cfg.ModelLlamaCppEndpoint)
	}
	if cfg.ModelLlamaCppBinary != "" {
		t.Fatalf("expected empty llama.cpp binary path, got %q", cfg.ModelLlamaCppBinary)
	}
	if cfg.AllowLlamaCppSpawn {
		t.Fatalf("expected llama.cpp spawn disabled by default")
	}
	if cfg.ModelMaxPromptTokens != 8192 {
		t.Fatalf("expected default max prompt tokens 8192, got %d", cfg.ModelMaxPromptTokens)
	}
	if cfg.ModelMaxOutputTokens != 1024 {
		t.Fatalf("expected default max output tokens 1024, got %d", cfg.ModelMaxOutputTokens)
	}
	if cfg.ModelMaxResponseBytes != 262144 {
		t.Fatalf("expected default max response bytes 262144, got %d", cfg.ModelMaxResponseBytes)
	}
	if cfg.ModelRequestTimeoutMs != 30000 {
		t.Fatalf("expected default request timeout 30000, got %d", cfg.ModelRequestTimeoutMs)
	}
	if cfg.ModelLoadTimeoutMs != 120000 {
		t.Fatalf("expected default load timeout 120000, got %d", cfg.ModelLoadTimeoutMs)
	}
	if cfg.ModelUnloadTimeoutMs != 30000 {
		t.Fatalf("expected default unload timeout 30000, got %d", cfg.ModelUnloadTimeoutMs)
	}
	if cfg.ModelIdleUnloadMs != 0 {
		t.Fatalf("expected default idle unload 0, got %d", cfg.ModelIdleUnloadMs)
	}
	if cfg.ModelMaxLoadedModels != 1 {
		t.Fatalf("expected default max loaded models 1, got %d", cfg.ModelMaxLoadedModels)
	}
	if cfg.ModelSchedulerMaxConcurrentRequests != 1 {
		t.Fatalf("expected default scheduler max concurrent 1, got %d", cfg.ModelSchedulerMaxConcurrentRequests)
	}
	if cfg.ModelSchedulerQueueCapacity != 8 {
		t.Fatalf("expected default scheduler queue capacity 8, got %d", cfg.ModelSchedulerQueueCapacity)
	}
	if cfg.ModelSchedulerDispatchTimeoutMs != 5000 {
		t.Fatalf("expected default scheduler dispatch timeout 5000, got %d", cfg.ModelSchedulerDispatchTimeoutMs)
	}
	if !cfg.ModelPolicyRequireExplicitLoad {
		t.Fatalf("expected require explicit load enabled by default")
	}
	if cfg.ModelPolicyAllowAutoLoad {
		t.Fatalf("expected auto-load disabled by default")
	}
	if cfg.ModelPolicyAllowCrossWorkspace {
		t.Fatalf("expected cross-workspace disabled by default")
	}
	if !cfg.ModelPolicyRequireWorkspaceScope {
		t.Fatalf("expected workspace scope requirement enabled by default")
	}
	if cfg.EnableOpenAICompatAPI {
		t.Fatalf("expected OpenAI-compat API disabled by default")
	}
}

func TestLoadModelRuntimeOverrides(t *testing.T) {
	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "true")
	t.Setenv("FORGE_MODEL_HOME", "./test-models")
	t.Setenv("FORGE_MODEL_DEFAULT_BACKEND", "llama_cpp")
	t.Setenv("FORGE_MODEL_DEFAULT_ID", "qwen2.5-coder")
	t.Setenv("FORGE_LLAMA_CPP_ENDPOINT", "http://127.0.0.1:8089")
	t.Setenv("FORGE_LLAMA_CPP_BINARY_PATH", "./bin/llama-server")
	t.Setenv("FORGE_ALLOW_LLAMA_CPP_SPAWN", "1")
	t.Setenv("FORGE_MODEL_MAX_PROMPT_TOKENS", "4096")
	t.Setenv("FORGE_MODEL_MAX_OUTPUT_TOKENS", "512")
	t.Setenv("FORGE_MODEL_MAX_RESPONSE_BYTES", "131072")
	t.Setenv("FORGE_MODEL_REQUEST_TIMEOUT_MS", "60000")
	t.Setenv("FORGE_MODEL_LOAD_TIMEOUT_MS", "90000")
	t.Setenv("FORGE_MODEL_UNLOAD_TIMEOUT_MS", "15000")
	t.Setenv("FORGE_MODEL_IDLE_UNLOAD_MS", "300000")
	t.Setenv("FORGE_MODEL_MAX_LOADED_MODELS", "2")
	t.Setenv("FORGE_MODEL_SCHEDULER_MAX_CONCURRENT", "3")
	t.Setenv("FORGE_MODEL_SCHEDULER_QUEUE_CAPACITY", "24")
	t.Setenv("FORGE_MODEL_SCHEDULER_DISPATCH_TIMEOUT_MS", "7000")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD", "false")
	t.Setenv("FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD", "true")
	t.Setenv("FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE", "true")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "false")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "true")

	cfg := Load()
	expectedModelHome, err := filepath.Abs("./test-models")
	if err != nil {
		t.Fatalf("resolve model home: %v", err)
	}
	expectedBinary, err := filepath.Abs("./bin/llama-server")
	if err != nil {
		t.Fatalf("resolve binary path: %v", err)
	}

	if !cfg.EnableModelRuntime {
		t.Fatalf("expected model runtime enabled")
	}
	if cfg.ModelHome != expectedModelHome {
		t.Fatalf("expected model home %q, got %q", expectedModelHome, cfg.ModelHome)
	}
	if cfg.ModelDefaultBackend != "llama_cpp" {
		t.Fatalf("expected default backend llama_cpp, got %q", cfg.ModelDefaultBackend)
	}
	if cfg.ModelDefaultID != "qwen2.5-coder" {
		t.Fatalf("expected default model id qwen2.5-coder, got %q", cfg.ModelDefaultID)
	}
	if cfg.ModelLlamaCppEndpoint != "http://127.0.0.1:8089" {
		t.Fatalf("expected llama.cpp endpoint override to be applied, got %q", cfg.ModelLlamaCppEndpoint)
	}
	if cfg.ModelLlamaCppBinary != expectedBinary {
		t.Fatalf("expected llama.cpp binary path %q, got %q", expectedBinary, cfg.ModelLlamaCppBinary)
	}
	if !cfg.AllowLlamaCppSpawn {
		t.Fatalf("expected llama.cpp spawn enabled")
	}
	if cfg.ModelMaxPromptTokens != 4096 {
		t.Fatalf("expected max prompt tokens 4096, got %d", cfg.ModelMaxPromptTokens)
	}
	if cfg.ModelMaxOutputTokens != 512 {
		t.Fatalf("expected max output tokens 512, got %d", cfg.ModelMaxOutputTokens)
	}
	if cfg.ModelMaxResponseBytes != 131072 {
		t.Fatalf("expected max response bytes 131072, got %d", cfg.ModelMaxResponseBytes)
	}
	if cfg.ModelRequestTimeoutMs != 60000 {
		t.Fatalf("expected request timeout 60000, got %d", cfg.ModelRequestTimeoutMs)
	}
	if cfg.ModelLoadTimeoutMs != 90000 {
		t.Fatalf("expected load timeout 90000, got %d", cfg.ModelLoadTimeoutMs)
	}
	if cfg.ModelUnloadTimeoutMs != 15000 {
		t.Fatalf("expected unload timeout 15000, got %d", cfg.ModelUnloadTimeoutMs)
	}
	if cfg.ModelIdleUnloadMs != 300000 {
		t.Fatalf("expected idle unload 300000, got %d", cfg.ModelIdleUnloadMs)
	}
	if cfg.ModelMaxLoadedModels != 2 {
		t.Fatalf("expected max loaded models 2, got %d", cfg.ModelMaxLoadedModels)
	}
	if cfg.ModelSchedulerMaxConcurrentRequests != 3 {
		t.Fatalf("expected scheduler max concurrent 3, got %d", cfg.ModelSchedulerMaxConcurrentRequests)
	}
	if cfg.ModelSchedulerQueueCapacity != 24 {
		t.Fatalf("expected scheduler queue capacity 24, got %d", cfg.ModelSchedulerQueueCapacity)
	}
	if cfg.ModelSchedulerDispatchTimeoutMs != 7000 {
		t.Fatalf("expected scheduler dispatch timeout 7000, got %d", cfg.ModelSchedulerDispatchTimeoutMs)
	}
	if cfg.ModelPolicyRequireExplicitLoad {
		t.Fatalf("expected explicit load policy disabled")
	}
	if !cfg.ModelPolicyAllowAutoLoad {
		t.Fatalf("expected auto-load policy enabled")
	}
	if !cfg.ModelPolicyAllowCrossWorkspace {
		t.Fatalf("expected cross-workspace policy enabled")
	}
	if cfg.ModelPolicyRequireWorkspaceScope {
		t.Fatalf("expected workspace scope policy disabled")
	}
	if !cfg.EnableOpenAICompatAPI {
		t.Fatalf("expected OpenAI-compat API enabled")
	}
}

func TestLoadModelRuntimeInvalidValuesFallbackToDefaults(t *testing.T) {
	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "not-a-bool")
	t.Setenv("FORGE_ALLOW_LLAMA_CPP_SPAWN", "nope")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "bad")
	t.Setenv("FORGE_MODEL_MAX_PROMPT_TOKENS", "abc")
	t.Setenv("FORGE_MODEL_MAX_OUTPUT_TOKENS", "-3")
	t.Setenv("FORGE_MODEL_MAX_RESPONSE_BYTES", "12")
	t.Setenv("FORGE_MODEL_REQUEST_TIMEOUT_MS", "0")
	t.Setenv("FORGE_MODEL_LOAD_TIMEOUT_MS", "0")
	t.Setenv("FORGE_MODEL_UNLOAD_TIMEOUT_MS", "-5")
	t.Setenv("FORGE_MODEL_IDLE_UNLOAD_MS", "-1")
	t.Setenv("FORGE_MODEL_MAX_LOADED_MODELS", "-1")
	t.Setenv("FORGE_MODEL_SCHEDULER_MAX_CONCURRENT", "0")
	t.Setenv("FORGE_MODEL_SCHEDULER_QUEUE_CAPACITY", "-2")
	t.Setenv("FORGE_MODEL_SCHEDULER_DISPATCH_TIMEOUT_MS", "0")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD", "maybe")
	t.Setenv("FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD", "sure")
	t.Setenv("FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE", "nah")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "idk")

	cfg := Load()

	if cfg.EnableModelRuntime {
		t.Fatalf("expected invalid bool to fall back to false")
	}
	if cfg.AllowLlamaCppSpawn {
		t.Fatalf("expected invalid bool to fall back to false for spawn")
	}
	if cfg.EnableOpenAICompatAPI {
		t.Fatalf("expected invalid bool to fall back to false for OpenAI-compat API")
	}
	if cfg.ModelMaxPromptTokens != 8192 {
		t.Fatalf("expected invalid max prompt tokens to fall back to 8192, got %d", cfg.ModelMaxPromptTokens)
	}
	if cfg.ModelMaxOutputTokens != 1024 {
		t.Fatalf("expected invalid max output tokens to fall back to 1024, got %d", cfg.ModelMaxOutputTokens)
	}
	if cfg.ModelMaxResponseBytes != 262144 {
		t.Fatalf("expected invalid max response bytes to fall back to 262144, got %d", cfg.ModelMaxResponseBytes)
	}
	if cfg.ModelRequestTimeoutMs != 30000 {
		t.Fatalf("expected invalid request timeout to fall back to 30000, got %d", cfg.ModelRequestTimeoutMs)
	}
	if cfg.ModelLoadTimeoutMs != 120000 {
		t.Fatalf("expected invalid load timeout to fall back to 120000, got %d", cfg.ModelLoadTimeoutMs)
	}
	if cfg.ModelUnloadTimeoutMs != 30000 {
		t.Fatalf("expected invalid unload timeout to fall back to 30000, got %d", cfg.ModelUnloadTimeoutMs)
	}
	if cfg.ModelIdleUnloadMs != 0 {
		t.Fatalf("expected invalid idle unload to fall back to 0, got %d", cfg.ModelIdleUnloadMs)
	}
	if cfg.ModelMaxLoadedModels != 1 {
		t.Fatalf("expected invalid max loaded models to fall back to 1, got %d", cfg.ModelMaxLoadedModels)
	}
	if cfg.ModelSchedulerMaxConcurrentRequests != 1 {
		t.Fatalf("expected invalid scheduler max concurrent to fall back to 1, got %d", cfg.ModelSchedulerMaxConcurrentRequests)
	}
	if cfg.ModelSchedulerQueueCapacity != 8 {
		t.Fatalf("expected invalid scheduler queue capacity to fall back to 8, got %d", cfg.ModelSchedulerQueueCapacity)
	}
	if cfg.ModelSchedulerDispatchTimeoutMs != 5000 {
		t.Fatalf("expected invalid scheduler dispatch timeout to fall back to 5000, got %d", cfg.ModelSchedulerDispatchTimeoutMs)
	}
	if !cfg.ModelPolicyRequireExplicitLoad {
		t.Fatalf("expected invalid explicit load policy to fall back to true")
	}
	if cfg.ModelPolicyAllowAutoLoad {
		t.Fatalf("expected invalid auto-load policy to fall back to false")
	}
	if cfg.ModelPolicyAllowCrossWorkspace {
		t.Fatalf("expected invalid cross-workspace policy to fall back to false")
	}
	if !cfg.ModelPolicyRequireWorkspaceScope {
		t.Fatalf("expected invalid workspace scope policy to fall back to true")
	}
}

func TestLoadModelRuntimeRemoteBackendOverrides(t *testing.T) {
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", "https://openai-compat.example")
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_API_KEY", "sk-test")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "http://127.0.0.1:8000")
	t.Setenv("FORGE_MODEL_VLLM_API_KEY", "vllm-key")

	cfg := Load()
	if cfg.ModelOpenAICompatEndpoint != "https://openai-compat.example" {
		t.Fatalf("expected openai-compatible endpoint override, got %q", cfg.ModelOpenAICompatEndpoint)
	}
	if cfg.ModelOpenAICompatAPIKey != "sk-test" {
		t.Fatalf("expected openai-compatible api key override, got %q", cfg.ModelOpenAICompatAPIKey)
	}
	if cfg.ModelVLLMEndpoint != "http://127.0.0.1:8000" {
		t.Fatalf("expected vllm endpoint override, got %q", cfg.ModelVLLMEndpoint)
	}
	if cfg.ModelVLLMAPIKey != "vllm-key" {
		t.Fatalf("expected vllm api key override, got %q", cfg.ModelVLLMAPIKey)
	}
}
