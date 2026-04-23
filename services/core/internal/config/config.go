package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	DataDir                             string
	Port                                int
	WorkspaceDir                        string
	EnableModelRuntime                  bool
	ModelHome                           string
	ModelDefaultBackend                 string
	ModelDefaultID                      string
	ModelLlamaCppEndpoint               string
	ModelLlamaCppBinary                 string
	ModelOpenAICompatEndpoint           string
	ModelOpenAICompatAPIKey             string
	ModelVLLMEndpoint                   string
	ModelVLLMAPIKey                     string
	AllowLlamaCppSpawn                  bool
	ModelMaxPromptTokens                int
	ModelMaxOutputTokens                int
	ModelMaxResponseBytes               int
	ModelRequestTimeoutMs               int
	ModelLoadTimeoutMs                  int
	ModelUnloadTimeoutMs                int
	ModelIdleUnloadMs                   int
	ModelMaxLoadedModels                int
	ModelSchedulerMaxConcurrentRequests int
	ModelSchedulerQueueCapacity         int
	ModelSchedulerDispatchTimeoutMs     int
	ModelPolicyRequireExplicitLoad      bool
	ModelPolicyAllowAutoLoad            bool
	ModelPolicyAllowCrossWorkspace      bool
	ModelPolicyRequireWorkspaceScope    bool
	EnableOpenAICompatAPI               bool
}

func Load() Config {
	dataDir := os.Getenv("FORGE_DATA_DIR")
	if dataDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			base = "."
		}
		dataDir = filepath.Join(base, "forge")
	}
	port := 18492
	if v := os.Getenv("FORGE_CORE_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			port = p
		}
	}
	workspace := os.Getenv("FORGE_WORKSPACE_DIR")
	if workspace == "" {
		workspace = "/"
	}
	if abs, err := filepath.Abs(workspace); err == nil {
		workspace = abs
	}

	modelHome := os.Getenv("FORGE_MODEL_HOME")
	if modelHome == "" {
		modelHome = filepath.Join(dataDir, "models")
	}
	if abs, err := filepath.Abs(modelHome); err == nil {
		modelHome = abs
	}

	llamaBinary := strings.TrimSpace(os.Getenv("FORGE_LLAMA_CPP_BINARY_PATH"))
	if llamaBinary != "" {
		if abs, err := filepath.Abs(llamaBinary); err == nil {
			llamaBinary = abs
		}
	}

	return Config{
		DataDir:                             dataDir,
		Port:                                port,
		WorkspaceDir:                        workspace,
		EnableModelRuntime:                  envBool("FORGE_ENABLE_MODEL_RUNTIME", false),
		ModelHome:                           modelHome,
		ModelDefaultBackend:                 strings.TrimSpace(os.Getenv("FORGE_MODEL_DEFAULT_BACKEND")),
		ModelDefaultID:                      strings.TrimSpace(os.Getenv("FORGE_MODEL_DEFAULT_ID")),
		ModelLlamaCppEndpoint:               strings.TrimSpace(os.Getenv("FORGE_LLAMA_CPP_ENDPOINT")),
		ModelLlamaCppBinary:                 llamaBinary,
		ModelOpenAICompatEndpoint:           strings.TrimSpace(os.Getenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT")),
		ModelOpenAICompatAPIKey:             strings.TrimSpace(os.Getenv("FORGE_MODEL_OPENAI_COMPAT_API_KEY")),
		ModelVLLMEndpoint:                   strings.TrimSpace(os.Getenv("FORGE_MODEL_VLLM_ENDPOINT")),
		ModelVLLMAPIKey:                     strings.TrimSpace(os.Getenv("FORGE_MODEL_VLLM_API_KEY")),
		AllowLlamaCppSpawn:                  envBool("FORGE_ALLOW_LLAMA_CPP_SPAWN", false),
		ModelMaxPromptTokens:                envInt("FORGE_MODEL_MAX_PROMPT_TOKENS", 8192, 1),
		ModelMaxOutputTokens:                envInt("FORGE_MODEL_MAX_OUTPUT_TOKENS", 1024, 1),
		ModelMaxResponseBytes:               envInt("FORGE_MODEL_MAX_RESPONSE_BYTES", 262144, 1024),
		ModelRequestTimeoutMs:               envInt("FORGE_MODEL_REQUEST_TIMEOUT_MS", 30000, 1),
		ModelLoadTimeoutMs:                  envInt("FORGE_MODEL_LOAD_TIMEOUT_MS", 120000, 1),
		ModelUnloadTimeoutMs:                envInt("FORGE_MODEL_UNLOAD_TIMEOUT_MS", 30000, 1),
		ModelIdleUnloadMs:                   envInt("FORGE_MODEL_IDLE_UNLOAD_MS", 0, 0),
		ModelMaxLoadedModels:                envInt("FORGE_MODEL_MAX_LOADED_MODELS", 1, 1),
		ModelSchedulerMaxConcurrentRequests: envInt("FORGE_MODEL_SCHEDULER_MAX_CONCURRENT", 1, 1),
		ModelSchedulerQueueCapacity:         envInt("FORGE_MODEL_SCHEDULER_QUEUE_CAPACITY", 8, 0),
		ModelSchedulerDispatchTimeoutMs:     envInt("FORGE_MODEL_SCHEDULER_DISPATCH_TIMEOUT_MS", 5000, 1),
		ModelPolicyRequireExplicitLoad:      envBool("FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD", true),
		ModelPolicyAllowAutoLoad:            envBool("FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD", false),
		ModelPolicyAllowCrossWorkspace:      envBool("FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE", false),
		ModelPolicyRequireWorkspaceScope:    envBool("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", true),
		EnableOpenAICompatAPI:               envBool("FORGE_ENABLE_OPENAI_COMPAT_API", false),
	}
}

func envBool(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultValue
	}
	return value
}

func envInt(key string, defaultValue, minValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue {
		return defaultValue
	}
	return value
}
