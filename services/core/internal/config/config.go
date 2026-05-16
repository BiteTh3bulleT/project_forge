package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"forge/projectforge/services/core/internal/storagebackend"
)

type Config struct {
	DataDir                                     string
	Port                                        int
	BindHost                                    string
	AllowWildcardBind                           bool
	APIToken                                    string
	APITokenFile                                string
	APIActor                                    string
	CORSAllowedOrigins                          []string
	CORSAllowDevLocalhost                       bool
	ProjectContextAllowedRoots                  []string
	WorkspaceDir                                string
	StoreBackend                                string
	PostgresDSN                                 string
	RedisAddr                                   string
	RedisEnabled                                bool
	RedisKeyPrefix                              string
	RedisTimeoutMs                              int
	QdrantURL                                   string
	QdrantShadowIndexEnabled                    bool
	QdrantCollection                            string
	QdrantVectorSize                            int
	QdrantTimeoutMs                             int
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
	ForgeKShadowAdvisoryEnabled                 bool
	ForgeKShadowControlLaneValidationEnabled    bool
	ShadowDiagnosticPersistenceEnabled          bool
	ShadowDiagnosticRetentionDays               int
	ShadowDiagnosticMaxPayloadBytes             int
}

var (
	ErrShadowDiagnosticPostgresRequired = errors.New("shadow diagnostic persistence requires postgres configuration")
	ErrShadowDiagnosticInvalidConfig    = errors.New("invalid shadow diagnostic persistence configuration")
	ErrQdrantShadowIndexURLRequired     = errors.New("qdrant shadow index requires qdrant url")
	ErrQdrantShadowIndexInvalidConfig   = errors.New("invalid qdrant shadow index configuration")
	ErrRedisAddrRequired                = errors.New("redis enabled requires redis addr")
	ErrRedisInvalidConfig               = errors.New("invalid redis ephemeral configuration")
)

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
	bindHost := envStringDefault("FORGE_CORE_BIND_HOST", "127.0.0.1")
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
		BindHost:                   bindHost,
		AllowWildcardBind:          envBool("FORGE_ALLOW_WILDCARD_BIND", false),
		APIToken:                   loadAPIToken(dataDir),
		APITokenFile:               apiTokenFile(dataDir),
		APIActor:                   envStringDefault("FORGE_API_ACTOR", "operator"),
		CORSAllowedOrigins:         envList("FORGE_CORS_ALLOWED_ORIGINS"),
		CORSAllowDevLocalhost:      envBool("FORGE_CORS_ALLOW_DEV_LOCALHOST", false),
		ProjectContextAllowedRoots: envList("FORGE_PROJECT_CONTEXT_ALLOWED_ROOTS"),
		WorkspaceDir:               workspace,
		StoreBackend:               envStringDefault("FORGE_STORE_BACKEND", "sqlite"),
		PostgresDSN:                strings.TrimSpace(os.Getenv("FORGE_POSTGRES_DSN")),
		RedisAddr:                  strings.TrimSpace(os.Getenv("FORGE_REDIS_ADDR")),
		RedisEnabled:               envBool("FORGE_REDIS_ENABLED", false),
		RedisKeyPrefix:             envStringDefault("FORGE_REDIS_KEY_PREFIX", "forge"),
		RedisTimeoutMs:             envInt("FORGE_REDIS_TIMEOUT_MS", 1000, 1),
		QdrantURL:                  strings.TrimSpace(os.Getenv("FORGE_QDRANT_URL")),
		QdrantShadowIndexEnabled:   envBool("FORGE_QDRANT_SHADOW_INDEX_ENABLED", false),
		QdrantCollection:           envStringDefault("FORGE_QDRANT_COLLECTION", "forge_shadow_embeddings"),
		QdrantVectorSize:           envInt("FORGE_QDRANT_VECTOR_SIZE", 0, 0),
		QdrantTimeoutMs:            envInt("FORGE_QDRANT_TIMEOUT_MS", 3000, 1),
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
		ModelHome:                                modelHome,
		ModelDefaultBackend:                      strings.TrimSpace(os.Getenv("FORGE_MODEL_DEFAULT_BACKEND")),
		ModelDefaultID:                           strings.TrimSpace(os.Getenv("FORGE_MODEL_DEFAULT_ID")),
		ModelLlamaCppEndpoint:                    strings.TrimSpace(os.Getenv("FORGE_LLAMA_CPP_ENDPOINT")),
		ModelLlamaCppBinary:                      llamaBinary,
		ModelOpenAICompatEndpoint:                strings.TrimSpace(os.Getenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT")),
		ModelOpenAICompatAPIKey:                  strings.TrimSpace(os.Getenv("FORGE_MODEL_OPENAI_COMPAT_API_KEY")),
		ModelVLLMEndpoint:                        envStringFirst("FORGE_VLLM_BASE_URL", "FORGE_MODEL_VLLM_ENDPOINT"),
		ModelVLLMAPIKey:                          envStringFirst("FORGE_VLLM_API_KEY", "FORGE_MODEL_VLLM_API_KEY"),
		EmbeddingProvider:                        strings.TrimSpace(os.Getenv("FORGE_EMBEDDING_PROVIDER")),
		EmbeddingModel:                           strings.TrimSpace(os.Getenv("FORGE_EMBEDDING_MODEL")),
		EmbeddingDims:                            envInt("FORGE_EMBEDDING_DIMS", 128, 1),
		EmbeddingTEIEndpoint:                     strings.TrimSpace(os.Getenv("FORGE_EMBEDDING_TEI_ENDPOINT")),
		EmbeddingTEIAPIKey:                       strings.TrimSpace(os.Getenv("FORGE_EMBEDDING_TEI_API_KEY")),
		EmbeddingTEITimeoutMs:                    envInt("FORGE_EMBEDDING_TEI_TIMEOUT_MS", 30000, 1),
		AllowLlamaCppSpawn:                       envBool("FORGE_ALLOW_LLAMA_CPP_SPAWN", false),
		ModelMaxPromptTokens:                     envInt("FORGE_MODEL_MAX_PROMPT_TOKENS", 8192, 1),
		ModelMaxOutputTokens:                     envInt("FORGE_MODEL_MAX_OUTPUT_TOKENS", 1024, 1),
		ModelMaxResponseBytes:                    envInt("FORGE_MODEL_MAX_RESPONSE_BYTES", 262144, 1024),
		ModelRequestTimeoutMs:                    envInt("FORGE_MODEL_REQUEST_TIMEOUT_MS", 30000, 1),
		ModelLoadTimeoutMs:                       envInt("FORGE_MODEL_LOAD_TIMEOUT_MS", 120000, 1),
		ModelUnloadTimeoutMs:                     envInt("FORGE_MODEL_UNLOAD_TIMEOUT_MS", 30000, 1),
		ModelIdleUnloadMs:                        envInt("FORGE_MODEL_IDLE_UNLOAD_MS", 0, 0),
		ModelMaxLoadedModels:                     envInt("FORGE_MODEL_MAX_LOADED_MODELS", 1, 1),
		ModelSchedulerMaxConcurrentRequests:      envInt("FORGE_MODEL_SCHEDULER_MAX_CONCURRENT", 1, 1),
		ModelSchedulerQueueCapacity:              envInt("FORGE_MODEL_SCHEDULER_QUEUE_CAPACITY", 8, 0),
		ModelSchedulerDispatchTimeoutMs:          envInt("FORGE_MODEL_SCHEDULER_DISPATCH_TIMEOUT_MS", 5000, 1),
		ModelPolicyRequireExplicitLoad:           envBool("FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD", true),
		ModelPolicyAllowAutoLoad:                 envBool("FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD", false),
		ModelPolicyAllowCrossWorkspace:           envBool("FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE", false),
		ModelPolicyRequireWorkspaceScope:         envBool("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", true),
		ModelRuntimeAllowOllamaCloudModels:       envBool("FORGE_MODELRUNTIME_ALLOW_OLLAMA_CLOUD_MODELS", false),
		ModelChatMaxAttempts:                     envInt("FORGE_MODEL_CHAT_MAX_ATTEMPTS", 3, 1),
		ModelChatRetryBackoffMs:                  envInt("FORGE_MODEL_CHAT_RETRY_BACKOFF_MS", 250, 0),
		ModelChatProviderCooldownMs:              envInt("FORGE_MODEL_CHAT_PROVIDER_COOLDOWN_MS", 5000, 0),
		ModelChatModelCooldownMs:                 envInt("FORGE_MODEL_CHAT_MODEL_COOLDOWN_MS", 5000, 0),
		ModelChatCheckpointLimit:                 envInt("FORGE_MODEL_CHAT_CHECKPOINT_LIMIT", 128, 1),
		EnableOpenAICompatAPI:                    envBool("FORGE_ENABLE_OPENAI_COMPAT_API", false),
		ForgeKShadowModeEnabled:                  envBool("FORGE_K_SHADOW_MODE_ENABLED", false),
		ForgeKShadowChatMetadataEnabled:          envBool("FORGE_K_SHADOW_CHAT_METADATA_ENABLED", false),
		ForgeKShadowRetrievalMetadataEnabled:     envBool("FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED", false),
		ForgeKShadowAdvisoryEnabled:              envBool("FORGE_K_SHADOW_ADVISORY_ENABLED", false),
		ForgeKShadowControlLaneValidationEnabled: envBool("FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED", false),
		ShadowDiagnosticPersistenceEnabled:       envBool("FORGE_SHADOW_DIAGNOSTIC_PERSISTENCE_ENABLED", false),
		ShadowDiagnosticRetentionDays:            envInt("FORGE_SHADOW_DIAGNOSTIC_RETENTION_DAYS", 30, 1),
		ShadowDiagnosticMaxPayloadBytes:          envInt("FORGE_SHADOW_DIAGNOSTIC_MAX_PAYLOAD_BYTES", 65536, 1024),
	}
}

func (c Config) StorageBackendConfig() (storagebackend.Config, error) {
	return storagebackend.NewConfig(storagebackend.ConfigInput{
		Backend:     c.StoreBackend,
		PostgresDSN: c.PostgresDSN,
		RedisAddr:   c.RedisAddr,
		QdrantURL:   c.QdrantURL,
	})
}

func (c Config) ValidateShadowDiagnosticPersistence() error {
	if !c.ShadowDiagnosticPersistenceEnabled {
		return nil
	}
	if strings.TrimSpace(c.PostgresDSN) == "" {
		return ErrShadowDiagnosticPostgresRequired
	}
	if c.ShadowDiagnosticRetentionDays <= 0 || c.ShadowDiagnosticMaxPayloadBytes <= 0 {
		return ErrShadowDiagnosticInvalidConfig
	}
	return nil
}

func (c Config) ValidateQdrantShadowIndex() error {
	if !c.QdrantShadowIndexEnabled {
		return nil
	}
	if strings.TrimSpace(c.QdrantURL) == "" {
		return ErrQdrantShadowIndexURLRequired
	}
	if strings.TrimSpace(c.QdrantCollection) == "" || c.QdrantTimeoutMs <= 0 || c.QdrantVectorSize < 0 {
		return ErrQdrantShadowIndexInvalidConfig
	}
	return nil
}

func (c Config) ValidateRedisEphemeral() error {
	if !c.RedisEnabled {
		return nil
	}
	if strings.TrimSpace(c.RedisAddr) == "" {
		return ErrRedisAddrRequired
	}
	if strings.TrimSpace(c.RedisKeyPrefix) == "" || c.RedisTimeoutMs <= 0 {
		return ErrRedisInvalidConfig
	}
	return nil
}

func envStringDefault(key, defaultValue string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	return raw
}

func envStringFirst(keys ...string) string {
	for _, key := range keys {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return raw
		}
	}
	return ""
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

func envList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func loadAPIToken(dataDir string) string {
	if token := strings.TrimSpace(os.Getenv("FORGE_API_TOKEN")); token != "" {
		return token
	}
	path := apiTokenFile(dataDir)
	if token := readTokenFile(path); token != "" {
		return token
	}
	token, err := generateAPIToken()
	if err != nil {
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return ""
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return ""
	}
	return token
}

func apiTokenFile(dataDir string) string {
	if path := strings.TrimSpace(os.Getenv("FORGE_API_TOKEN_FILE")); path != "" {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}
	return filepath.Join(dataDir, "auth", "api_token")
}

func readTokenFile(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func generateAPIToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
