package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadDefaultsWorkspaceUnderDataDir(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("FORGE_DATA_DIR", dataDir)
	t.Setenv("FORGE_WORKSPACE_DIR", "")
	cfg := Load()
	want, err := filepath.Abs(filepath.Join(dataDir, "workspace"))
	if err != nil {
		t.Fatalf("resolve default workspace: %v", err)
	}
	if cfg.WorkspaceDir != want {
		t.Fatalf("expected default workspace %q, got %q", want, cfg.WorkspaceDir)
	}
}

func TestLoadDefaultsRootWorkspaceOptInDisabled(t *testing.T) {
	t.Setenv("FORGE_ALLOW_ROOT_WORKSPACE", "")
	cfg := Load()
	if cfg.AllowRootWorkspace {
		t.Fatal("expected root workspace opt-in to default false")
	}
}

func TestLoadRespectsRootWorkspaceOptIn(t *testing.T) {
	t.Setenv("FORGE_ALLOW_ROOT_WORKSPACE", "true")
	cfg := Load()
	if !cfg.AllowRootWorkspace {
		t.Fatal("expected root workspace opt-in true")
	}
}

func TestLoadDefaultsCoreBindHostToLoopback(t *testing.T) {
	t.Setenv("FORGE_CORE_BIND_HOST", "")
	cfg := Load()
	if cfg.BindHost != "127.0.0.1" {
		t.Fatalf("expected default bind host 127.0.0.1, got %q", cfg.BindHost)
	}
}

func TestLoadUsesEnvAPITokenWithoutWritingTokenFile(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("FORGE_DATA_DIR", dataDir)
	t.Setenv("FORGE_API_TOKEN", " env-token ")
	t.Setenv("FORGE_API_TOKEN_FILE", "")

	cfg := Load()
	if cfg.APIToken != "env-token" {
		t.Fatalf("expected env API token, got %q", cfg.APIToken)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "auth", "api_token")); !os.IsNotExist(err) {
		t.Fatalf("expected env token load not to create token file, stat err=%v", err)
	}
}

func TestLoadGeneratesAPITokenFileUnderDataDir(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("FORGE_DATA_DIR", dataDir)
	t.Setenv("FORGE_API_TOKEN", "")
	t.Setenv("FORGE_API_TOKEN_FILE", "")

	cfg := Load()
	if len(cfg.APIToken) < 32 {
		t.Fatalf("expected generated API token, got %q", cfg.APIToken)
	}
	body, err := os.ReadFile(filepath.Join(dataDir, "auth", "api_token"))
	if err != nil {
		t.Fatalf("expected generated token file: %v", err)
	}
	if string(body) != cfg.APIToken+"\n" {
		t.Fatalf("token file did not match loaded token")
	}
	authDirInfo, err := os.Stat(filepath.Join(dataDir, "auth"))
	if err != nil {
		t.Fatalf("expected generated auth directory: %v", err)
	}
	assertFileMode(t, authDirInfo, 0o750, "auth directory")
	tokenInfo, err := os.Stat(filepath.Join(dataDir, "auth", "api_token"))
	if err != nil {
		t.Fatalf("expected generated token file stat: %v", err)
	}
	assertFileMode(t, tokenInfo, 0o640, "token file")
}

func TestLoadRepairsExistingAPITokenFilePermissions(t *testing.T) {
	dataDir := t.TempDir()
	tokenPath := filepath.Join(dataDir, "auth", "api_token")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
		t.Fatalf("create auth dir: %v", err)
	}
	if err := os.WriteFile(tokenPath, []byte("existing-token\n"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	t.Setenv("FORGE_DATA_DIR", dataDir)
	t.Setenv("FORGE_API_TOKEN", "")
	t.Setenv("FORGE_API_TOKEN_FILE", "")

	cfg := Load()
	if cfg.APIToken != "existing-token" {
		t.Fatalf("expected existing token, got %q", cfg.APIToken)
	}
	authDirInfo, err := os.Stat(filepath.Dir(tokenPath))
	if err != nil {
		t.Fatalf("expected auth directory stat: %v", err)
	}
	assertFileMode(t, authDirInfo, 0o750, "repaired auth directory")
	tokenInfo, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("expected token file stat: %v", err)
	}
	assertFileMode(t, tokenInfo, 0o640, "repaired token file")
}

func assertFileMode(t *testing.T, info os.FileInfo, want os.FileMode, label string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("expected %s mode %04o, got %04o", label, want, got)
	}
}

func TestLoadRespectsCoreBindHostOverride(t *testing.T) {
	t.Setenv("FORGE_CORE_BIND_HOST", " 0.0.0.0 ")
	cfg := Load()
	if cfg.BindHost != "0.0.0.0" {
		t.Fatalf("expected bind host override 0.0.0.0, got %q", cfg.BindHost)
	}
}

func TestLoadDefaultsWildcardBindOptInDisabled(t *testing.T) {
	t.Setenv("FORGE_ALLOW_WILDCARD_BIND", "")
	cfg := Load()
	if cfg.AllowWildcardBind {
		t.Fatal("expected wildcard bind opt-in to default false")
	}
}

func TestLoadDefaultsMetricsEndpointDisabled(t *testing.T) {
	t.Setenv("FORGE_ENABLE_METRICS_ENDPOINT", "")
	cfg := Load()
	if cfg.EnableMetricsEndpoint {
		t.Fatal("expected metrics endpoint to default false")
	}
}

func TestLoadRespectsMetricsEndpointOptIn(t *testing.T) {
	t.Setenv("FORGE_ENABLE_METRICS_ENDPOINT", "true")
	cfg := Load()
	if !cfg.EnableMetricsEndpoint {
		t.Fatal("expected metrics endpoint opt-in true")
	}
}

func TestLoadRespectsWildcardBindOptIn(t *testing.T) {
	t.Setenv("FORGE_ALLOW_WILDCARD_BIND", "true")
	cfg := Load()
	if !cfg.AllowWildcardBind {
		t.Fatal("expected wildcard bind opt-in true")
	}
}

func TestLoadRespectsWorkspaceOverride(t *testing.T) {
	t.Setenv("FORGE_WORKSPACE_DIR", "/tmp")
	cfg := Load()
	want, err := filepath.Abs("/tmp")
	if err != nil {
		t.Fatalf("resolve workspace override: %v", err)
	}
	if cfg.WorkspaceDir != want {
		t.Fatalf("expected workspace override %q, got %q", want, cfg.WorkspaceDir)
	}
}

func TestLoadModelRuntimeDefaultsSafe(t *testing.T) {
	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "")
	t.Setenv("FORGE_GPU_ENABLED", "")
	t.Setenv("FORGE_NVIDIA_DCGM_ENABLED", "")
	t.Setenv("FORGE_NVIDIA_DCGM_ENDPOINT", "")
	t.Setenv("FORGE_NVIDIA_DCGM_TIMEOUT_MS", "")
	t.Setenv("FORGE_INTEL_LEVEL_ZERO_ENABLED", "")
	t.Setenv("FORGE_INTEL_LEVEL_ZERO_ZE_INFO_PATH", "")
	t.Setenv("FORGE_INTEL_GPU_TOP_PATH", "")
	t.Setenv("FORGE_INTEL_GPU_TELEMETRY_TIMEOUT_MS", "")
	t.Setenv("FORGE_GPU_BACKGROUND_MEMORY_PRESSURE_BLOCK_THRESHOLD", "")
	t.Setenv("FORGE_GPU_REQUIRED_FOR_INTERACTIVE_INFERENCE", "")
	t.Setenv("FORGE_GPU_VRAM_HEADROOM_FRACTION", "")
	t.Setenv("FORGE_GPU_BACKGROUND_JOBS_ENABLED", "")
	t.Setenv("FORGE_GPU_BACKGROUND_IDLE_THRESHOLD_SECONDS", "")
	t.Setenv("FORGE_GPU_MAX_BACKGROUND_JOBS", "")
	t.Setenv("FORGE_DREAM_MODE_ALLOW_GPU_SUBJOBS", "")
	t.Setenv("FORGE_DREAM_MODE_GPU_ONLY_IN_DEEP_IDLE", "")
	t.Setenv("FORGE_SAFE_MODE_FORCE_CPU_ONLY", "")
	t.Setenv("FORGE_MODELRUNTIME_DEGRADED_ON_UNAVAILABLE_GPU", "")
	t.Setenv("FORGE_SCHEDULING_INTERACTIVE_PRIORITY_OVER_BACKGROUND", "")
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
	t.Setenv("FORGE_MODEL_CHAT_MAX_ATTEMPTS", "")
	t.Setenv("FORGE_MODEL_CHAT_RETRY_BACKOFF_MS", "")
	t.Setenv("FORGE_MODEL_CHAT_PROVIDER_COOLDOWN_MS", "")
	t.Setenv("FORGE_MODEL_CHAT_MODEL_COOLDOWN_MS", "")
	t.Setenv("FORGE_MODEL_CHAT_CHECKPOINT_LIMIT", "")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "")
	t.Setenv("FORGE_EMBEDDING_TEI_ENDPOINT", "")
	t.Setenv("FORGE_EMBEDDING_TEI_API_KEY", "")
	t.Setenv("FORGE_EMBEDDING_TEI_TIMEOUT_MS", "")
	t.Setenv("FORGE_EMBEDDING_PROVIDER", "")
	t.Setenv("FORGE_EMBEDDING_MODEL", "")
	t.Setenv("FORGE_EMBEDDING_DIMS", "")
	t.Setenv("FORGE_K_SHADOW_MODE_ENABLED", "")
	t.Setenv("FORGE_K_SHADOW_CHAT_METADATA_ENABLED", "")
	t.Setenv("FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED", "")
	t.Setenv("FORGE_K_SHADOW_ADVISORY_ENABLED", "")
	t.Setenv("FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED", "")

	cfg := Load()
	expectedModelHome, err := filepath.Abs(filepath.Join(cfg.DataDir, "models"))
	if err != nil {
		t.Fatalf("resolve default model home: %v", err)
	}

	if cfg.EnableModelRuntime {
		t.Fatalf("expected model runtime disabled by default")
	}
	if cfg.GPUEnabled {
		t.Fatalf("expected gpu disabled by default")
	}
	if cfg.NVIDIADCGMEnabled || cfg.NVIDIADCGMEndpoint != "" {
		t.Fatalf("expected NVIDIA DCGM disabled/unconfigured by default")
	}
	if cfg.NVIDIADCGMTimeoutMs != 1500 {
		t.Fatalf("expected default DCGM timeout 1500ms, got %d", cfg.NVIDIADCGMTimeoutMs)
	}
	if cfg.IntelLevelZeroEnabled || cfg.IntelLevelZeroZEInfoPath != "" || cfg.IntelGPUTopPath != "" {
		t.Fatalf("expected Intel Level Zero disabled/unconfigured by default")
	}
	if cfg.IntelGPUTelemetryTimeoutMs != 1500 {
		t.Fatalf("expected default Intel telemetry timeout 1500ms, got %d", cfg.IntelGPUTelemetryTimeoutMs)
	}
	if cfg.GPUBackgroundMemoryPressureBlockThreshold != 0.90 {
		t.Fatalf("expected GPU background pressure threshold 0.90, got %f", cfg.GPUBackgroundMemoryPressureBlockThreshold)
	}
	if cfg.GPURequiredForInteractiveInference {
		t.Fatalf("expected gpu to be optional for interactive inference by default")
	}
	if cfg.GPUVRAMHeadroomFraction != 0.20 {
		t.Fatalf("expected default GPU VRAM headroom fraction 0.20, got %f", cfg.GPUVRAMHeadroomFraction)
	}
	if cfg.GPUBackgroundJobsEnabled {
		t.Fatalf("expected background GPU jobs disabled by default")
	}
	if cfg.GPUBackgroundIdleThresholdSeconds != 300 {
		t.Fatalf("expected default GPU background idle threshold 300s, got %d", cfg.GPUBackgroundIdleThresholdSeconds)
	}
	if cfg.GPUMaxBackgroundJobs != 1 {
		t.Fatalf("expected default max background GPU jobs 1, got %d", cfg.GPUMaxBackgroundJobs)
	}
	if cfg.DreamModeAllowGPUSubjobs {
		t.Fatalf("expected dream mode GPU subjobs disabled by default")
	}
	if cfg.ForgeKShadowModeEnabled {
		t.Fatalf("expected FORGE-K shadow mode disabled by default")
	}
	if cfg.ForgeKShadowChatMetadataEnabled {
		t.Fatalf("expected FORGE-K chat metadata shadow disabled by default")
	}
	if cfg.ForgeKShadowRetrievalMetadataEnabled {
		t.Fatalf("expected FORGE-K retrieval metadata shadow disabled by default")
	}
	if cfg.ForgeKShadowAdvisoryEnabled {
		t.Fatalf("expected FORGE-K shadow advisory disabled by default")
	}
	if cfg.ForgeKShadowControlLaneValidationEnabled {
		t.Fatalf("expected FORGE-K control lane validation shadow disabled by default")
	}
	if !cfg.DreamModeGPUOnlyInDeepIdle {
		t.Fatalf("expected dream mode GPU to be deep-idle-only by default")
	}
	if cfg.SafeModeForceCPUOnly {
		t.Fatalf("expected safe mode cpu-only override disabled by default")
	}
	if !cfg.ModelRuntimeDegradedOnUnavailableGPU {
		t.Fatalf("expected modelruntime degraded-on-unavailable-gpu enabled by default")
	}
	if !cfg.SchedulingInteractivePriorityOverBackground {
		t.Fatalf("expected interactive-priority-over-background enabled by default")
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
	if cfg.ModelChatMaxAttempts != 3 {
		t.Fatalf("expected default max attempts 3, got %d", cfg.ModelChatMaxAttempts)
	}
	if cfg.ModelChatRetryBackoffMs != 250 {
		t.Fatalf("expected default retry backoff 250, got %d", cfg.ModelChatRetryBackoffMs)
	}
	if cfg.ModelChatProviderCooldownMs != 5000 {
		t.Fatalf("expected default provider cooldown 5000, got %d", cfg.ModelChatProviderCooldownMs)
	}
	if cfg.ModelChatModelCooldownMs != 5000 {
		t.Fatalf("expected default model cooldown 5000, got %d", cfg.ModelChatModelCooldownMs)
	}
	if cfg.ModelChatCheckpointLimit != 128 {
		t.Fatalf("expected default checkpoint limit 128, got %d", cfg.ModelChatCheckpointLimit)
	}
	if cfg.EnableOpenAICompatAPI {
		t.Fatalf("expected OpenAI-compat API disabled by default")
	}
	if cfg.EmbeddingTEIEndpoint != "" || cfg.EmbeddingTEIAPIKey != "" {
		t.Fatalf("expected TEI unconfigured by default")
	}
	if cfg.EmbeddingProvider != "" || cfg.EmbeddingModel != "" || cfg.EmbeddingDims != 128 {
		t.Fatalf("expected embedding provider/model unset and dims 128 by default, got provider=%q model=%q dims=%d", cfg.EmbeddingProvider, cfg.EmbeddingModel, cfg.EmbeddingDims)
	}
	if cfg.EmbeddingTEITimeoutMs != 30000 {
		t.Fatalf("expected TEI timeout 30000, got %d", cfg.EmbeddingTEITimeoutMs)
	}
}

func TestLoadModelRuntimeOverrides(t *testing.T) {
	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "true")
	t.Setenv("FORGE_GPU_ENABLED", "true")
	t.Setenv("FORGE_NVIDIA_DCGM_ENABLED", "true")
	t.Setenv("FORGE_NVIDIA_DCGM_ENDPOINT", "http://127.0.0.1:9400/metrics")
	t.Setenv("FORGE_NVIDIA_DCGM_TIMEOUT_MS", "2500")
	t.Setenv("FORGE_INTEL_LEVEL_ZERO_ENABLED", "true")
	t.Setenv("FORGE_INTEL_LEVEL_ZERO_ZE_INFO_PATH", "/usr/bin/ze_info")
	t.Setenv("FORGE_INTEL_GPU_TOP_PATH", "/usr/bin/intel_gpu_top")
	t.Setenv("FORGE_INTEL_GPU_TELEMETRY_TIMEOUT_MS", "2200")
	t.Setenv("FORGE_GPU_BACKGROUND_MEMORY_PRESSURE_BLOCK_THRESHOLD", "0.82")
	t.Setenv("FORGE_GPU_REQUIRED_FOR_INTERACTIVE_INFERENCE", "true")
	t.Setenv("FORGE_GPU_VRAM_HEADROOM_FRACTION", "0.35")
	t.Setenv("FORGE_GPU_BACKGROUND_JOBS_ENABLED", "true")
	t.Setenv("FORGE_GPU_BACKGROUND_IDLE_THRESHOLD_SECONDS", "900")
	t.Setenv("FORGE_GPU_MAX_BACKGROUND_JOBS", "2")
	t.Setenv("FORGE_DREAM_MODE_ALLOW_GPU_SUBJOBS", "true")
	t.Setenv("FORGE_DREAM_MODE_GPU_ONLY_IN_DEEP_IDLE", "false")
	t.Setenv("FORGE_SAFE_MODE_FORCE_CPU_ONLY", "true")
	t.Setenv("FORGE_MODELRUNTIME_DEGRADED_ON_UNAVAILABLE_GPU", "false")
	t.Setenv("FORGE_SCHEDULING_INTERACTIVE_PRIORITY_OVER_BACKGROUND", "false")
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
	t.Setenv("FORGE_MODEL_CHAT_MAX_ATTEMPTS", "5")
	t.Setenv("FORGE_MODEL_CHAT_RETRY_BACKOFF_MS", "250")
	t.Setenv("FORGE_MODEL_CHAT_PROVIDER_COOLDOWN_MS", "12000")
	t.Setenv("FORGE_MODEL_CHAT_MODEL_COOLDOWN_MS", "6000")
	t.Setenv("FORGE_MODEL_CHAT_CHECKPOINT_LIMIT", "256")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "true")
	t.Setenv("FORGE_EMBEDDING_TEI_ENDPOINT", "http://127.0.0.1:8081")
	t.Setenv("FORGE_EMBEDDING_TEI_API_KEY", "secret")
	t.Setenv("FORGE_EMBEDDING_TEI_TIMEOUT_MS", "45000")
	t.Setenv("FORGE_EMBEDDING_PROVIDER", "tei")
	t.Setenv("FORGE_EMBEDDING_MODEL", "bge-large")
	t.Setenv("FORGE_EMBEDDING_DIMS", "1024")
	t.Setenv("FORGE_K_SHADOW_MODE_ENABLED", "true")
	t.Setenv("FORGE_K_SHADOW_CHAT_METADATA_ENABLED", "true")
	t.Setenv("FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED", "true")
	t.Setenv("FORGE_K_SHADOW_ADVISORY_ENABLED", "true")
	t.Setenv("FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED", "true")

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
	if !cfg.GPUEnabled {
		t.Fatalf("expected gpu enabled")
	}
	if !cfg.NVIDIADCGMEnabled || cfg.NVIDIADCGMEndpoint != "http://127.0.0.1:9400/metrics" {
		t.Fatalf("expected NVIDIA DCGM override, got enabled=%v endpoint=%q", cfg.NVIDIADCGMEnabled, cfg.NVIDIADCGMEndpoint)
	}
	if cfg.NVIDIADCGMTimeoutMs != 2500 {
		t.Fatalf("expected DCGM timeout 2500, got %d", cfg.NVIDIADCGMTimeoutMs)
	}
	if !cfg.IntelLevelZeroEnabled || cfg.IntelLevelZeroZEInfoPath != "/usr/bin/ze_info" || cfg.IntelGPUTopPath != "/usr/bin/intel_gpu_top" {
		t.Fatalf("expected Intel Level Zero overrides, got enabled=%v ze=%q top=%q", cfg.IntelLevelZeroEnabled, cfg.IntelLevelZeroZEInfoPath, cfg.IntelGPUTopPath)
	}
	if cfg.IntelGPUTelemetryTimeoutMs != 2200 {
		t.Fatalf("expected Intel telemetry timeout 2200, got %d", cfg.IntelGPUTelemetryTimeoutMs)
	}
	if cfg.GPUBackgroundMemoryPressureBlockThreshold != 0.82 {
		t.Fatalf("expected GPU pressure threshold 0.82, got %f", cfg.GPUBackgroundMemoryPressureBlockThreshold)
	}
	if !cfg.GPURequiredForInteractiveInference {
		t.Fatalf("expected gpu required for interactive inference override")
	}
	if cfg.GPUVRAMHeadroomFraction != 0.35 {
		t.Fatalf("expected GPU VRAM headroom fraction 0.35, got %f", cfg.GPUVRAMHeadroomFraction)
	}
	if !cfg.GPUBackgroundJobsEnabled {
		t.Fatalf("expected background GPU jobs enabled")
	}
	if cfg.GPUBackgroundIdleThresholdSeconds != 900 {
		t.Fatalf("expected GPU background idle threshold 900s, got %d", cfg.GPUBackgroundIdleThresholdSeconds)
	}
	if cfg.GPUMaxBackgroundJobs != 2 {
		t.Fatalf("expected max background GPU jobs 2, got %d", cfg.GPUMaxBackgroundJobs)
	}
	if !cfg.DreamModeAllowGPUSubjobs {
		t.Fatalf("expected dream mode GPU subjobs enabled")
	}
	if cfg.DreamModeGPUOnlyInDeepIdle {
		t.Fatalf("expected dream mode deep-idle-only override disabled")
	}
	if !cfg.SafeModeForceCPUOnly {
		t.Fatalf("expected safe mode force cpu-only enabled")
	}
	if cfg.ModelRuntimeDegradedOnUnavailableGPU {
		t.Fatalf("expected modelruntime degraded-on-unavailable-gpu disabled")
	}
	if cfg.SchedulingInteractivePriorityOverBackground {
		t.Fatalf("expected interactive-priority-over-background override disabled")
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
	if cfg.ModelChatMaxAttempts != 5 {
		t.Fatalf("expected max attempts 5, got %d", cfg.ModelChatMaxAttempts)
	}
	if cfg.ModelChatRetryBackoffMs != 250 {
		t.Fatalf("expected retry backoff 250ms, got %d", cfg.ModelChatRetryBackoffMs)
	}
	if cfg.ModelChatProviderCooldownMs != 12000 {
		t.Fatalf("expected provider cooldown 12000ms, got %d", cfg.ModelChatProviderCooldownMs)
	}
	if cfg.ModelChatModelCooldownMs != 6000 {
		t.Fatalf("expected model cooldown 6000ms, got %d", cfg.ModelChatModelCooldownMs)
	}
	if cfg.ModelChatCheckpointLimit != 256 {
		t.Fatalf("expected checkpoint limit 256, got %d", cfg.ModelChatCheckpointLimit)
	}
	if !cfg.EnableOpenAICompatAPI {
		t.Fatalf("expected OpenAI-compat API enabled")
	}
	if cfg.EmbeddingTEIEndpoint != "http://127.0.0.1:8081" || cfg.EmbeddingTEIAPIKey != "secret" || cfg.EmbeddingTEITimeoutMs != 45000 {
		t.Fatalf("expected TEI overrides, got endpoint=%q key=%q timeout=%d", cfg.EmbeddingTEIEndpoint, cfg.EmbeddingTEIAPIKey, cfg.EmbeddingTEITimeoutMs)
	}
	if cfg.EmbeddingProvider != "tei" || cfg.EmbeddingModel != "bge-large" || cfg.EmbeddingDims != 1024 {
		t.Fatalf("expected embedding overrides, got provider=%q model=%q dims=%d", cfg.EmbeddingProvider, cfg.EmbeddingModel, cfg.EmbeddingDims)
	}
	if !cfg.ForgeKShadowModeEnabled {
		t.Fatalf("expected FORGE-K shadow mode enabled from env")
	}
	if !cfg.ForgeKShadowChatMetadataEnabled {
		t.Fatalf("expected FORGE-K chat metadata shadow enabled from env")
	}
	if !cfg.ForgeKShadowRetrievalMetadataEnabled {
		t.Fatalf("expected FORGE-K retrieval metadata shadow enabled from env")
	}
	if !cfg.ForgeKShadowAdvisoryEnabled {
		t.Fatalf("expected FORGE-K shadow advisory enabled from env")
	}
	if !cfg.ForgeKShadowControlLaneValidationEnabled {
		t.Fatalf("expected FORGE-K control lane validation shadow enabled from env")
	}
}

func TestLoadModelRuntimeInvalidValuesFallbackToDefaults(t *testing.T) {
	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "not-a-bool")
	t.Setenv("FORGE_GPU_ENABLED", "gpu?")
	t.Setenv("FORGE_GPU_REQUIRED_FOR_INTERACTIVE_INFERENCE", "required?")
	t.Setenv("FORGE_GPU_VRAM_HEADROOM_FRACTION", "1.7")
	t.Setenv("FORGE_GPU_BACKGROUND_JOBS_ENABLED", "background?")
	t.Setenv("FORGE_GPU_BACKGROUND_IDLE_THRESHOLD_SECONDS", "-5")
	t.Setenv("FORGE_GPU_MAX_BACKGROUND_JOBS", "-2")
	t.Setenv("FORGE_DREAM_MODE_ALLOW_GPU_SUBJOBS", "dream?")
	t.Setenv("FORGE_DREAM_MODE_GPU_ONLY_IN_DEEP_IDLE", "deep?")
	t.Setenv("FORGE_SAFE_MODE_FORCE_CPU_ONLY", "safe?")
	t.Setenv("FORGE_MODELRUNTIME_DEGRADED_ON_UNAVAILABLE_GPU", "degrade?")
	t.Setenv("FORGE_SCHEDULING_INTERACTIVE_PRIORITY_OVER_BACKGROUND", "prio?")
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
	t.Setenv("FORGE_MODEL_CHAT_MAX_ATTEMPTS", "zero")
	t.Setenv("FORGE_MODEL_CHAT_RETRY_BACKOFF_MS", "-1")
	t.Setenv("FORGE_MODEL_CHAT_PROVIDER_COOLDOWN_MS", "bad")
	t.Setenv("FORGE_MODEL_CHAT_MODEL_COOLDOWN_MS", "none")
	t.Setenv("FORGE_MODEL_CHAT_CHECKPOINT_LIMIT", "x")
	t.Setenv("FORGE_K_SHADOW_MODE_ENABLED", "shadow?")
	t.Setenv("FORGE_K_SHADOW_CHAT_METADATA_ENABLED", "chat?")
	t.Setenv("FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED", "retrieval?")
	t.Setenv("FORGE_K_SHADOW_ADVISORY_ENABLED", "advisory?")
	t.Setenv("FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED", "control?")

	cfg := Load()

	if cfg.EnableModelRuntime {
		t.Fatalf("expected invalid bool to fall back to false")
	}
	if cfg.ForgeKShadowModeEnabled {
		t.Fatalf("expected invalid FORGE-K shadow mode bool to fall back to false")
	}
	if cfg.ForgeKShadowChatMetadataEnabled {
		t.Fatalf("expected invalid FORGE-K chat metadata shadow bool to fall back to false")
	}
	if cfg.ForgeKShadowRetrievalMetadataEnabled {
		t.Fatalf("expected invalid FORGE-K retrieval metadata shadow bool to fall back to false")
	}
	if cfg.ForgeKShadowAdvisoryEnabled {
		t.Fatalf("expected invalid FORGE-K shadow advisory bool to fall back to false")
	}
	if cfg.ForgeKShadowControlLaneValidationEnabled {
		t.Fatalf("expected invalid FORGE-K control lane validation shadow bool to fall back to false")
	}
	if cfg.GPUEnabled {
		t.Fatalf("expected invalid GPU enabled bool to fall back to false")
	}
	if cfg.GPURequiredForInteractiveInference {
		t.Fatalf("expected invalid GPU required bool to fall back to false")
	}
	if cfg.GPUVRAMHeadroomFraction != 0.20 {
		t.Fatalf("expected invalid GPU VRAM headroom to fall back to 0.20, got %f", cfg.GPUVRAMHeadroomFraction)
	}
	if cfg.GPUBackgroundJobsEnabled {
		t.Fatalf("expected invalid background jobs bool to fall back to false")
	}
	if cfg.GPUBackgroundIdleThresholdSeconds != 300 {
		t.Fatalf("expected invalid background idle threshold to fall back to 300, got %d", cfg.GPUBackgroundIdleThresholdSeconds)
	}
	if cfg.GPUMaxBackgroundJobs != 1 {
		t.Fatalf("expected invalid max background jobs to fall back to 1, got %d", cfg.GPUMaxBackgroundJobs)
	}
	if cfg.DreamModeAllowGPUSubjobs {
		t.Fatalf("expected invalid dream mode gpu-subjobs bool to fall back to false")
	}
	if !cfg.DreamModeGPUOnlyInDeepIdle {
		t.Fatalf("expected invalid dream mode deep idle bool to fall back to true")
	}
	if cfg.SafeModeForceCPUOnly {
		t.Fatalf("expected invalid safe mode bool to fall back to false")
	}
	if !cfg.ModelRuntimeDegradedOnUnavailableGPU {
		t.Fatalf("expected invalid degraded-on-unavailable-gpu bool to fall back to true")
	}
	if !cfg.SchedulingInteractivePriorityOverBackground {
		t.Fatalf("expected invalid interactive-priority bool to fall back to true")
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
	if cfg.ModelChatMaxAttempts != 3 {
		t.Fatalf("expected invalid max attempts to fall back to 3, got %d", cfg.ModelChatMaxAttempts)
	}
	if cfg.ModelChatRetryBackoffMs != 250 {
		t.Fatalf("expected invalid retry backoff to fall back to 250, got %d", cfg.ModelChatRetryBackoffMs)
	}
	if cfg.ModelChatProviderCooldownMs != 5000 {
		t.Fatalf("expected invalid provider cooldown to fall back to 5000, got %d", cfg.ModelChatProviderCooldownMs)
	}
	if cfg.ModelChatModelCooldownMs != 5000 {
		t.Fatalf("expected invalid model cooldown to fall back to 5000, got %d", cfg.ModelChatModelCooldownMs)
	}
	if cfg.ModelChatCheckpointLimit != 128 {
		t.Fatalf("expected invalid checkpoint limit to fall back to 128, got %d", cfg.ModelChatCheckpointLimit)
	}
}

func TestLoadModelRuntimeRemoteBackendOverrides(t *testing.T) {
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", "https://openai-compat.example")
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_API_KEY", "sk-test")
	t.Setenv("FORGE_VLLM_BASE_URL", "http://127.0.0.1:8000")
	t.Setenv("FORGE_VLLM_API_KEY", "vllm-key")

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

func TestLoadModelRuntimeLegacyVLLMOverrides(t *testing.T) {
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "http://127.0.0.1:8001")
	t.Setenv("FORGE_MODEL_VLLM_API_KEY", "legacy-vllm-key")

	cfg := Load()
	if cfg.ModelVLLMEndpoint != "http://127.0.0.1:8001" {
		t.Fatalf("expected legacy vllm endpoint override, got %q", cfg.ModelVLLMEndpoint)
	}
	if cfg.ModelVLLMAPIKey != "legacy-vllm-key" {
		t.Fatalf("expected legacy vllm api key override, got %q", cfg.ModelVLLMAPIKey)
	}
}

func TestLoadModelRuntimeCanonicalVLLMOverridesLegacy(t *testing.T) {
	t.Setenv("FORGE_VLLM_BASE_URL", "http://127.0.0.1:8000")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "http://127.0.0.1:8001")
	t.Setenv("FORGE_VLLM_API_KEY", "canonical-key")
	t.Setenv("FORGE_MODEL_VLLM_API_KEY", "legacy-key")

	cfg := Load()
	if cfg.ModelVLLMEndpoint != "http://127.0.0.1:8000" {
		t.Fatalf("expected canonical vllm endpoint to win, got %q", cfg.ModelVLLMEndpoint)
	}
	if cfg.ModelVLLMAPIKey != "canonical-key" {
		t.Fatalf("expected canonical vllm api key to win, got %q", cfg.ModelVLLMAPIKey)
	}
}
