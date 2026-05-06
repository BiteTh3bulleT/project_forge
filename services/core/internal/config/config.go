package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	DataDir                                     string
	Port                                        int
	WorkspaceDir                                string
	EnableModelRuntime                          bool
	GPUEnabled                                  bool
	NVIDIADCGMEnabled                           bool
	NVIDIADCGMEndpoint                          string
	NVIDIADCGMTimeoutMs                         int
	IntelLevelZeroEnabled                       bool
	IntelLevelZeroZEInfoPath                    string
	IntelGPUTopPath                             string
	IntelGPUTelemetryTimeoutMs                  int
	GPUBackgroundMemoryPressureBlockThreshold   float64
	GPURequiredForInteractiveInference          bool
	GPUVRAMHeadroomFraction                     float64
	GPUBackgroundJobsEnabled                    bool
	GPUBackgroundIdleThresholdSeconds           int
	GPUMaxBackgroundJobs                        int
	DreamModeAllowGPUSubjobs                    bool
	DreamModeGPUOnlyInDeepIdle                  bool
	SafeModeForceCPUOnly                        bool
	ModelRuntimeDegradedOnUnavailableGPU        bool
	SchedulingInteractivePriorityOverBackground bool
	ModelHome                                   string
	ModelDefaultBackend                         string
	ModelDefaultID                              string
	ModelLlamaCppEndpoint                       string
	ModelLlamaCppBinary                         string
	ModelOpenAICompatEndpoint                   string
	ModelOpenAICompatAPIKey                     string
	ModelVLLMEndpoint                           string
	ModelVLLMAPIKey                             string
	EmbeddingProvider                           string
	EmbeddingModel                              string
	EmbeddingDims                               int
	EmbeddingTEIEndpoint                        string
	EmbeddingTEIAPIKey                          string
	EmbeddingTEITimeoutMs                       int
	AllowLlamaCppSpawn                          bool
	ModelMaxPromptTokens                        int
	ModelMaxOutputTokens                        int
	ModelMaxResponseBytes                       int
	ModelRequestTimeoutMs                       int
	ModelLoadTimeoutMs                          int
	ModelUnloadTimeoutMs                        int
	ModelIdleUnloadMs                           int
	ModelMaxLoadedModels                        int
	ModelSchedulerMaxConcurrentRequests         int
	ModelSchedulerQueueCapacity                 int
	ModelSchedulerDispatchTimeoutMs             int
	ModelPolicyRequireExplicitLoad              bool
	ModelPolicyAllowAutoLoad                    bool
	ModelPolicyAllowCrossWorkspace              bool
	ModelPolicyRequireWorkspaceScope            bool
	ModelRuntimeAllowOllamaCloudModels          bool
	ModelChatMaxAttempts                        int
	ModelChatRetryBackoffMs                     int
	ModelChatProviderCooldownMs                 int
	ModelChatModelCooldownMs                    int
	ModelChatCheckpointLimit                    int
	EnableOpenAICompatAPI                       bool
	ForgeKShadowModeEnabled                     bool
	ForgeKShadowChatMetadataEnabled             bool
	ForgeKShadowRetrievalMetadataEnabled        bool
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
		DataDir:                    dataDir,
		Port:                       port,
		WorkspaceDir:               workspace,
		EnableModelRuntime:         envBool("FORGE_ENABLE_MODEL_RUNTIME", false),
		GPUEnabled:                 envBool("FORGE_GPU_ENABLED", false),
		NVIDIADCGMEnabled:          envBool("FORGE_NVIDIA_DCGM_ENABLED", false),
		NVIDIADCGMEndpoint:         strings.TrimSpace(os.Getenv("FORGE_NVIDIA_DCGM_ENDPOINT")),
		NVIDIADCGMTimeoutMs:        envInt("FORGE_NVIDIA_DCGM_TIMEOUT_MS", 1500, 1),
		IntelLevelZeroEnabled:      envBool("FORGE_INTEL_LEVEL_ZERO_ENABLED", false),
		IntelLevelZeroZEInfoPath:   strings.TrimSpace(os.Getenv("FORGE_INTEL_LEVEL_ZERO_ZE_INFO_PATH")),
		IntelGPUTopPath:            strings.TrimSpace(os.Getenv("FORGE_INTEL_GPU_TOP_PATH")),
		IntelGPUTelemetryTimeoutMs: envInt("FORGE_INTEL_GPU_TELEMETRY_TIMEOUT_MS", 1500, 1),
		GPUBackgroundMemoryPressureBlockThreshold:   envFloat("FORGE_GPU_BACKGROUND_MEMORY_PRESSURE_BLOCK_THRESHOLD", 0.90, 0, 1),
		GPURequiredForInteractiveInference:          envBool("FORGE_GPU_REQUIRED_FOR_INTERACTIVE_INFERENCE", false),
		GPUVRAMHeadroomFraction:                     envFloat("FORGE_GPU_VRAM_HEADROOM_FRACTION", 0.20, 0, 1),
		GPUBackgroundJobsEnabled:                    envBool("FORGE_GPU_BACKGROUND_JOBS_ENABLED", false),
		GPUBackgroundIdleThresholdSeconds:           envInt("FORGE_GPU_BACKGROUND_IDLE_THRESHOLD_SECONDS", 300, 0),
		GPUMaxBackgroundJobs:                        envInt("FORGE_GPU_MAX_BACKGROUND_JOBS", 1, 0),
		DreamModeAllowGPUSubjobs:                    envBool("FORGE_DREAM_MODE_ALLOW_GPU_SUBJOBS", false),
		DreamModeGPUOnlyInDeepIdle:                  envBool("FORGE_DREAM_MODE_GPU_ONLY_IN_DEEP_IDLE", true),
		SafeModeForceCPUOnly:                        envBool("FORGE_SAFE_MODE_FORCE_CPU_ONLY", false),
		ModelRuntimeDegradedOnUnavailableGPU:        envBool("FORGE_MODELRUNTIME_DEGRADED_ON_UNAVAILABLE_GPU", true),
		SchedulingInteractivePriorityOverBackground: envBool("FORGE_SCHEDULING_INTERACTIVE_PRIORITY_OVER_BACKGROUND", true),
		ModelHome:                            modelHome,
		ModelDefaultBackend:                  strings.TrimSpace(os.Getenv("FORGE_MODEL_DEFAULT_BACKEND")),
		ModelDefaultID:                       strings.TrimSpace(os.Getenv("FORGE_MODEL_DEFAULT_ID")),
		ModelLlamaCppEndpoint:                strings.TrimSpace(os.Getenv("FORGE_LLAMA_CPP_ENDPOINT")),
		ModelLlamaCppBinary:                  llamaBinary,
		ModelOpenAICompatEndpoint:            strings.TrimSpace(os.Getenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT")),
		ModelOpenAICompatAPIKey:              strings.TrimSpace(os.Getenv("FORGE_MODEL_OPENAI_COMPAT_API_KEY")),
		ModelVLLMEndpoint:                    strings.TrimSpace(os.Getenv("FORGE_MODEL_VLLM_ENDPOINT")),
		ModelVLLMAPIKey:                      strings.TrimSpace(os.Getenv("FORGE_MODEL_VLLM_API_KEY")),
		EmbeddingProvider:                    strings.TrimSpace(os.Getenv("FORGE_EMBEDDING_PROVIDER")),
		EmbeddingModel:                       strings.TrimSpace(os.Getenv("FORGE_EMBEDDING_MODEL")),
		EmbeddingDims:                        envInt("FORGE_EMBEDDING_DIMS", 128, 1),
		EmbeddingTEIEndpoint:                 strings.TrimSpace(os.Getenv("FORGE_EMBEDDING_TEI_ENDPOINT")),
		EmbeddingTEIAPIKey:                   strings.TrimSpace(os.Getenv("FORGE_EMBEDDING_TEI_API_KEY")),
		EmbeddingTEITimeoutMs:                envInt("FORGE_EMBEDDING_TEI_TIMEOUT_MS", 30000, 1),
		AllowLlamaCppSpawn:                   envBool("FORGE_ALLOW_LLAMA_CPP_SPAWN", false),
		ModelMaxPromptTokens:                 envInt("FORGE_MODEL_MAX_PROMPT_TOKENS", 8192, 1),
		ModelMaxOutputTokens:                 envInt("FORGE_MODEL_MAX_OUTPUT_TOKENS", 1024, 1),
		ModelMaxResponseBytes:                envInt("FORGE_MODEL_MAX_RESPONSE_BYTES", 262144, 1024),
		ModelRequestTimeoutMs:                envInt("FORGE_MODEL_REQUEST_TIMEOUT_MS", 30000, 1),
		ModelLoadTimeoutMs:                   envInt("FORGE_MODEL_LOAD_TIMEOUT_MS", 120000, 1),
		ModelUnloadTimeoutMs:                 envInt("FORGE_MODEL_UNLOAD_TIMEOUT_MS", 30000, 1),
		ModelIdleUnloadMs:                    envInt("FORGE_MODEL_IDLE_UNLOAD_MS", 0, 0),
		ModelMaxLoadedModels:                 envInt("FORGE_MODEL_MAX_LOADED_MODELS", 1, 1),
		ModelSchedulerMaxConcurrentRequests:  envInt("FORGE_MODEL_SCHEDULER_MAX_CONCURRENT", 1, 1),
		ModelSchedulerQueueCapacity:          envInt("FORGE_MODEL_SCHEDULER_QUEUE_CAPACITY", 8, 0),
		ModelSchedulerDispatchTimeoutMs:      envInt("FORGE_MODEL_SCHEDULER_DISPATCH_TIMEOUT_MS", 5000, 1),
		ModelPolicyRequireExplicitLoad:       envBool("FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD", true),
		ModelPolicyAllowAutoLoad:             envBool("FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD", false),
		ModelPolicyAllowCrossWorkspace:       envBool("FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE", false),
		ModelPolicyRequireWorkspaceScope:     envBool("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", true),
		ModelRuntimeAllowOllamaCloudModels:   envBool("FORGE_MODELRUNTIME_ALLOW_OLLAMA_CLOUD_MODELS", false),
		ModelChatMaxAttempts:                 envInt("FORGE_MODEL_CHAT_MAX_ATTEMPTS", 3, 1),
		ModelChatRetryBackoffMs:              envInt("FORGE_MODEL_CHAT_RETRY_BACKOFF_MS", 250, 0),
		ModelChatProviderCooldownMs:          envInt("FORGE_MODEL_CHAT_PROVIDER_COOLDOWN_MS", 5000, 0),
		ModelChatModelCooldownMs:             envInt("FORGE_MODEL_CHAT_MODEL_COOLDOWN_MS", 5000, 0),
		ModelChatCheckpointLimit:             envInt("FORGE_MODEL_CHAT_CHECKPOINT_LIMIT", 128, 1),
		EnableOpenAICompatAPI:                envBool("FORGE_ENABLE_OPENAI_COMPAT_API", false),
		ForgeKShadowModeEnabled:              envBool("FORGE_K_SHADOW_MODE_ENABLED", false),
		ForgeKShadowChatMetadataEnabled:      envBool("FORGE_K_SHADOW_CHAT_METADATA_ENABLED", false),
		ForgeKShadowRetrievalMetadataEnabled: envBool("FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED", false),
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

func envFloat(key string, defaultValue, minValue, maxValue float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return defaultValue
	}
	if value < minValue || value > maxValue {
		return defaultValue
	}
	return value
}
