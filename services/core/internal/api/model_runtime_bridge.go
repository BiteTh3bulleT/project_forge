package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/gpu"
	"forge/projectforge/services/core/internal/modelruntime"
)

type modelRuntimeBridge struct {
	runtime                              *modelruntime.Service
	maxPromptTokens                      int
	safeModeForceCPUOnly                 bool
	gpuEnabledConfigured                 bool
	modelruntimeDegradedOnUnavailableGPU bool
	dreamModeGPUOnlyInDeepIdle           bool
	modelOpenAICompatEndpoint            string
	modelOpenAICompatAPIKey              string
	modelVLLMEndpoint                    string
	modelVLLMAPIKey                      string
	ollamaEndpoint                       string
	ollamaDiscoveryEnabled               bool
	allowOllamaCloudModels               bool
}

const modelRuntimeDiscoveryResponseLimit = 8 << 20

func readModelRuntimeDiscoveryResponse(body io.Reader, label string) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(body, modelRuntimeDiscoveryResponseLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > modelRuntimeDiscoveryResponseLimit {
		return nil, fmt.Errorf("%s response too large: limit %d bytes", label, modelRuntimeDiscoveryResponseLimit)
	}
	return raw, nil
}

func initModelRuntimeService(cfg config.Config, auditSvc *audit.Service, telemetry ...any) modelRuntimeService {
	var gpuTelemetry *gpu.Service
	if len(telemetry) > 0 {
		gpuTelemetry, _ = telemetry[0].(*gpu.Service)
	}
	var intelTelemetry *gpu.IntelService
	if len(telemetry) > 1 {
		intelTelemetry, _ = telemetry[1].(*gpu.IntelService)
	}
	runtimeEnabled := cfg.EnableModelRuntime
	if !runtimeEnabled && cfg.EnableOpenAICompatAPI {
		log.Printf("model runtime auto-enabled because FORGE_ENABLE_OPENAI_COMPAT_API is true")
		runtimeEnabled = true
	}
	if !runtimeEnabled && strings.TrimSpace(cfg.ModelOpenAICompatEndpoint) != "" {
		log.Printf("model runtime auto-enabled because FORGE_MODEL_OPENAI_COMPAT_ENDPOINT is configured")
		runtimeEnabled = true
	}
	if !runtimeEnabled && strings.TrimSpace(cfg.ModelVLLMEndpoint) != "" {
		log.Printf("model runtime auto-enabled because FORGE_MODEL_VLLM_ENDPOINT is configured")
		runtimeEnabled = true
	}
	if !runtimeEnabled {
		return nil
	}

	modelStore := modelruntime.NewModelStore(cfg.ModelHome, modelruntime.ModelStoreOptions{StrictChecksum: false})
	if err := os.MkdirAll(filepath.Join(cfg.ModelHome, "models"), 0o755); err != nil {
		log.Printf("model runtime model home init warning: %v", err)
	}
	registry := modelruntime.NewModelRegistry(modelStore)
	models := []modelruntime.ModelManifest{}
	registered, err := registry.Scan(context.Background())
	if err != nil {
		log.Printf("model runtime startup scan warning: %v", err)
	} else {
		models = make([]modelruntime.ModelManifest, 0, len(registered))
		for _, rec := range registered {
			models = append(models, rec.Manifest)
		}
	}

	modelIDs := map[string]struct{}{}
	for _, rec := range models {
		modelIDs[strings.ToLower(strings.TrimSpace(rec.ID))] = struct{}{}
	}

	if discovered, err := discoverOpenAICompatModels(context.Background(), cfg); err != nil {
		log.Printf("model runtime remote model discovery warning: %v", err)
	} else if len(discovered) > 0 {
		for _, model := range discovered {
			id := strings.ToLower(strings.TrimSpace(model.ID))
			if id == "" {
				continue
			}
			if _, exists := modelIDs[id]; exists {
				continue
			}
			models = append(models, model)
			modelIDs[id] = struct{}{}
		}
		registerDiscoveredModels(registry, discovered)
	}
	ollamaEndpoint := localOllamaEndpoint()
	if discovered, err := discoverLocalOllamaModels(context.Background(), ollamaEndpoint, cfg.ModelRuntimeAllowOllamaCloudModels); err != nil {
		log.Printf("model runtime local ollama discovery warning: %v", err)
	} else if len(discovered) > 0 {
		for _, model := range discovered {
			id := strings.ToLower(strings.TrimSpace(model.ID))
			if id == "" {
				continue
			}
			if _, exists := modelIDs[id]; exists {
				continue
			}
			models = append(models, model)
			modelIDs[id] = struct{}{}
		}
		registerDiscoveredModels(registry, discovered)
	}

	requiredBackends := map[modelruntime.ModelBackendKind]struct{}{}
	backendKind := modelruntime.ParseModelBackendKind(cfg.ModelDefaultBackend)
	if backendKind == "" {
		switch {
		case strings.TrimSpace(cfg.ModelOpenAICompatEndpoint) != "":
			backendKind = modelruntime.BackendOpenAICompat
		case strings.TrimSpace(cfg.ModelVLLMEndpoint) != "":
			backendKind = modelruntime.BackendVLLM
		case strings.TrimSpace(cfg.ModelLlamaCppEndpoint) != "" || strings.TrimSpace(cfg.ModelLlamaCppBinary) != "":
			backendKind = modelruntime.BackendLlamaCpp
		}
	}
	if backendKind != "" {
		requiredBackends[backendKind] = struct{}{}
	}
	for _, model := range models {
		requiredBackends[model.Backend] = struct{}{}
	}
	if len(requiredBackends) == 0 {
		requiredBackends[modelruntime.BackendLlamaCpp] = struct{}{}
	}

	backends := make([]modelruntime.ModelBackend, 0, len(requiredBackends))
	for kind := range requiredBackends {
		switch kind {
		case modelruntime.BackendLlamaCpp:
			backends = append(backends, modelruntime.NewLlamaCppBackend(modelruntime.LlamaCppOptions{
				Endpoint:        cfg.ModelLlamaCppEndpoint,
				RequestTimeout:  time.Duration(cfg.ModelRequestTimeoutMs) * time.Millisecond,
				MaxOutputTokens: cfg.ModelMaxOutputTokens,
				AllowSpawn:      cfg.AllowLlamaCppSpawn,
				BinaryPath:      cfg.ModelLlamaCppBinary,
			}))
		case modelruntime.BackendFake:
			// Production runtime refuses to spin up the fake backend. The fake
			// variant exists only to drive unit tests; surfacing it from the
			// HTTP path would let a caller import a model that never calls
			// out to a real engine, which produces misleading UI state. Test
			// suites construct FakeBackend directly.
			continue
		case modelruntime.BackendOpenAICompat:
			backends = append(backends, modelruntime.NewOpenAICompatBackend(modelruntime.OpenAICompatOptions{
				Name:            "openai-compatible",
				Kind:            modelruntime.BackendOpenAICompat,
				Endpoint:        cfg.ModelOpenAICompatEndpoint,
				APIKey:          cfg.ModelOpenAICompatAPIKey,
				RequestTimeout:  time.Duration(cfg.ModelRequestTimeoutMs) * time.Millisecond,
				MaxOutputTokens: cfg.ModelMaxOutputTokens,
			}))
		case modelruntime.BackendOllamaCompat:
			backends = append(backends, modelruntime.NewOpenAICompatBackend(modelruntime.OpenAICompatOptions{
				Name:            "ollama-local",
				Kind:            modelruntime.BackendOllamaCompat,
				Endpoint:        ollamaEndpoint,
				RequestTimeout:  time.Duration(cfg.ModelRequestTimeoutMs) * time.Millisecond,
				MaxOutputTokens: cfg.ModelMaxOutputTokens,
			}))
		case modelruntime.BackendVLLM:
			backends = append(backends, modelruntime.NewOpenAICompatBackend(modelruntime.OpenAICompatOptions{
				Name:            "vllm-compatible",
				Kind:            modelruntime.BackendVLLM,
				Endpoint:        cfg.ModelVLLMEndpoint,
				APIKey:          cfg.ModelVLLMAPIKey,
				Profile:         "interactive_vllm",
				RequestTimeout:  time.Duration(cfg.ModelRequestTimeoutMs) * time.Millisecond,
				MaxOutputTokens: cfg.ModelMaxOutputTokens,
			}))
		default:
			log.Printf("model runtime backend kind %q is not supported in m3 init", kind)
		}
	}
	if len(backends) == 0 {
		log.Printf("model runtime disabled: no supported backends configured")
		return nil
	}

	gpuEnabled := cfg.GPUEnabled && !cfg.SafeModeForceCPUOnly
	gpuBackgroundJobsEnabled := cfg.GPUBackgroundJobsEnabled && gpuEnabled
	gpuRequiredForInteractiveInference := cfg.GPURequiredForInteractiveInference

	runtimeSvc, err := modelruntime.NewService(modelruntime.ServiceOptions{
		Backends:                           backends,
		Models:                             models,
		Registry:                           registry,
		DefaultModelID:                     cfg.ModelDefaultID,
		DefaultTimeout:                     time.Duration(cfg.ModelRequestTimeoutMs) * time.Millisecond,
		MaxPromptTokens:                    cfg.ModelMaxPromptTokens,
		MaxOutputTokens:                    cfg.ModelMaxOutputTokens,
		MaxOutputBytes:                     cfg.ModelMaxResponseBytes,
		MaxLoadedModels:                    cfg.ModelMaxLoadedModels,
		LoadTimeout:                        time.Duration(cfg.ModelLoadTimeoutMs) * time.Millisecond,
		UnloadTimeout:                      time.Duration(cfg.ModelUnloadTimeoutMs) * time.Millisecond,
		AutoLoad:                           cfg.ModelPolicyAllowAutoLoad && !cfg.ModelPolicyRequireExplicitLoad,
		Audit:                              &modelRuntimeAuditAdapter{auditSvc: auditSvc},
		MaxQueueDepth:                      cfg.ModelSchedulerQueueCapacity,
		MaxConcurrentRequests:              cfg.ModelSchedulerMaxConcurrentRequests,
		ChatMaxAttempts:                    cfg.ModelChatMaxAttempts,
		ChatRetryBackoff:                   time.Duration(cfg.ModelChatRetryBackoffMs) * time.Millisecond,
		ChatProviderCooldown:               time.Duration(cfg.ModelChatProviderCooldownMs) * time.Millisecond,
		ChatModelCooldown:                  time.Duration(cfg.ModelChatModelCooldownMs) * time.Millisecond,
		ChatCheckpointLimit:                cfg.ModelChatCheckpointLimit,
		GPUEnabled:                         gpuEnabled,
		GPURequiredForInteractiveInference: gpuRequiredForInteractiveInference,
		GPUVRAMHeadroomFraction:            cfg.GPUVRAMHeadroomFraction,
		GPUBkgJobsEnabled:                  gpuBackgroundJobsEnabled,
		GPUBkgIdleThreshold:                time.Duration(cfg.GPUBackgroundIdleThresholdSeconds) * time.Second,
		GPUMaxBackgroundJobs:               cfg.GPUMaxBackgroundJobs,
		DegradeOnUnavailableGPU:            cfg.ModelRuntimeDegradedOnUnavailableGPU,
		SchedulingInteractivePriorityOverBackground: cfg.SchedulingInteractivePriorityOverBackground,
		DreamModeAllowGPUClassify:                   cfg.DreamModeAllowGPUSubjobs,
		GPUTelemetry:                                modelRuntimeTelemetryAdapter(gpuTelemetry, intelTelemetry),
		RequestValidator: func(_ context.Context, req modelruntime.GenerateRequest) error {
			if cfg.ModelPolicyRequireWorkspaceScope && strings.TrimSpace(req.WorkspaceID) == "" {
				return modelruntime.ErrWorkspaceRequired
			}
			if !cfg.ModelPolicyAllowCrossWorkspace && strings.TrimSpace(req.Scope) != "" && strings.TrimSpace(req.WorkspaceID) != "" && strings.TrimSpace(req.Scope) != strings.TrimSpace(req.WorkspaceID) {
				return fmt.Errorf("%w: cross-workspace scope %q blocked for workspace %q", modelruntime.ErrPolicyDenied, req.Scope, req.WorkspaceID)
			}
			if strings.EqualFold(strings.TrimSpace(req.Source), "autonomy") {
				return fmt.Errorf("%w: self-initiated autonomy inference is not enabled in the current model runtime policy", modelruntime.ErrPolicyDenied)
			}
			if selfInitiated, _ := req.Metadata["selfInitiated"].(bool); selfInitiated {
				return fmt.Errorf("%w: self-initiated inference is not enabled in the current model runtime policy", modelruntime.ErrPolicyDenied)
			}
			return nil
		},
	})
	if err != nil {
		log.Printf("model runtime disabled: %v", err)
		return nil
	}

	if defaultModel := strings.TrimSpace(cfg.ModelDefaultID); defaultModel != "" {
		if _, err := runtimeSvc.Inspect(context.Background(), defaultModel); err != nil {
			log.Printf("model runtime default model %q unavailable: %v", defaultModel, err)
		}
	}

	return &modelRuntimeBridge{
		runtime:                              runtimeSvc,
		maxPromptTokens:                      cfg.ModelMaxPromptTokens,
		safeModeForceCPUOnly:                 cfg.SafeModeForceCPUOnly,
		gpuEnabledConfigured:                 cfg.GPUEnabled,
		modelruntimeDegradedOnUnavailableGPU: cfg.ModelRuntimeDegradedOnUnavailableGPU,
		dreamModeGPUOnlyInDeepIdle:           cfg.DreamModeGPUOnlyInDeepIdle,
		modelOpenAICompatEndpoint:            strings.TrimSpace(cfg.ModelOpenAICompatEndpoint),
		modelOpenAICompatAPIKey:              strings.TrimSpace(cfg.ModelOpenAICompatAPIKey),
		modelVLLMEndpoint:                    strings.TrimSpace(cfg.ModelVLLMEndpoint),
		modelVLLMAPIKey:                      strings.TrimSpace(cfg.ModelVLLMAPIKey),
		ollamaEndpoint:                       ollamaEndpoint,
		ollamaDiscoveryEnabled:               modelruntime.ParseModelBackendKind(cfg.ModelDefaultBackend) == modelruntime.BackendOllamaCompat || strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")) != "",
		allowOllamaCloudModels:               cfg.ModelRuntimeAllowOllamaCloudModels,
	}
}

func modelRuntimeTelemetryAdapter(dcgm *gpu.Service, intel *gpu.IntelService) modelruntime.GPUTelemetryFunc {
	if dcgm == nil && intel == nil {
		return nil
	}
	return func(ctx context.Context) (modelruntime.GPUTelemetrySnapshot, error) {
		snap := gpu.Telemetry{Enabled: false, Healthy: true, State: "disabled", BackgroundAdmissionOK: true}
		if dcgm != nil {
			snap = dcgm.Snapshot(ctx)
		}
		if intel != nil {
			intelSnap := intel.Snapshot(ctx)
			if !snap.Enabled || (snap.State == "disabled" && intelSnap.Enabled) {
				snap = intelSnap
			} else if intelSnap.Enabled {
				snap.Warnings = append(snap.Warnings, "intel_level_zero:"+intelSnap.State)
				snap.Devices = append(snap.Devices, intelSnap.Devices...)
				if !intelSnap.Healthy && snap.Healthy {
					snap.Healthy = false
					snap.State = "degraded"
					snap.Detail = intelSnap.Detail
				}
				if !intelSnap.BackgroundAdmissionOK {
					snap.BackgroundAdmissionOK = false
				}
			}
		}
		devices := make([]map[string]any, 0, len(snap.Devices))
		for _, device := range snap.Devices {
			devices = append(devices, map[string]any{
				"index":          device.Index,
				"uuid":           device.UUID,
				"gpuUtilization": device.GPUUtilization,
				"memoryUsedMiB":  device.MemoryUsedMiB,
				"memoryFreeMiB":  device.MemoryFreeMiB,
				"memoryTotalMiB": device.MemoryTotalMiB,
				"memoryPressure": device.MemoryPressure,
				"powerWatts":     device.PowerWatts,
				"temperatureC":   device.TemperatureC,
			})
		}
		return modelruntime.GPUTelemetrySnapshot{
			Enabled:                 snap.Enabled,
			Available:               snap.Available,
			Healthy:                 snap.Healthy,
			State:                   snap.State,
			Detail:                  snap.Detail,
			MemoryPressure:          snap.MemoryPressure,
			MemoryPressureThreshold: snap.MemoryPressureThreshold,
			BackgroundAdmissionOK:   snap.BackgroundAdmissionOK,
			Devices:                 devices,
			Warnings:                append([]string(nil), snap.Warnings...),
		}, nil
	}
}

func registerDiscoveredModels(registry *modelruntime.ModelRegistry, discovered []modelruntime.ModelManifest) {
	if registry == nil || len(discovered) == 0 {
		return
	}
	for _, model := range discovered {
		if err := registry.Register(model); err != nil {
			log.Printf("model runtime discovered model registration warning for %q: %v", strings.TrimSpace(model.ID), err)
		}
	}
}

type openAIModelsListResponse struct {
	Data []openAIModelDescriptor `json:"data"`
}

type openAIModelDescriptor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ollamaTagsResponse struct {
	Models []ollamaModelDescriptor `json:"models"`
}

type ollamaModelDescriptor struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	RemoteHost string `json:"remote_host"`
	Size       int64  `json:"size"`
	Details    struct {
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
		Format            string   `json:"format"`
	} `json:"details"`
}

func localOllamaEndpoint() string {
	endpoint := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL"))
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434"
	}
	return strings.TrimRight(endpoint, "/")
}

func isLocalHTTPProvider(endpoint string) bool {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	host := strings.TrimSpace(u.Hostname())
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if strings.EqualFold(host, "host.docker.internal") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func discoverLocalOllamaModels(ctx context.Context, endpoint string, includeCloud bool) ([]modelruntime.ModelManifest, error) {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" || !isLocalHTTPProvider(endpoint) {
		return nil, nil
	}
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("ollama model discovery: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama model discovery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama model discovery: endpoint returned %s", resp.Status)
	}
	body, err := readModelRuntimeDiscoveryResponse(resp.Body, "ollama model discovery")
	if err != nil {
		return nil, fmt.Errorf("ollama model discovery: read body: %w", err)
	}
	var payload ollamaTagsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("ollama model discovery: unmarshal: %w", err)
	}
	out := make([]modelruntime.ModelManifest, 0, len(payload.Models))
	for _, item := range payload.Models {
		model, err := modelRuntimeOllamaManifest(item, endpoint)
		if err != nil {
			continue
		}
		if model.Metadata != nil {
			if remote, _ := model.Metadata["remote"].(bool); remote && !includeCloud {
				continue
			}
		}
		out = append(out, model)
	}
	return out, nil
}

func modelRuntimeOllamaManifest(item ollamaModelDescriptor, endpoint string) (modelruntime.ModelManifest, error) {
	id := strings.TrimSpace(firstNonEmptyTrimmed(item.Model, item.Name))
	if id == "" {
		return modelruntime.ModelManifest{}, fmt.Errorf("ollama model id required")
	}
	remoteHost := strings.TrimSpace(item.RemoteHost)
	remote := remoteHost != "" || strings.Contains(strings.ToLower(id), ":cloud") || strings.Contains(strings.ToLower(id), "-cloud")
	family := strings.TrimSpace(item.Details.Family)
	if family == "" && len(item.Details.Families) > 0 {
		family = strings.TrimSpace(item.Details.Families[0])
	}
	if family == "" {
		family = "ollama"
	}
	format := modelruntime.ParseModelFormat(item.Details.Format)
	if format == modelruntime.ModelFormatUnknown {
		format = modelruntime.ModelFormatGGUF
	}
	quantization := strings.TrimSpace(item.Details.QuantizationLevel)
	if quantization == "" {
		quantization = "ollama"
	}
	capabilities := []modelruntime.ModelCapability{modelruntime.ModelCapabilityChat, modelruntime.ModelCapabilityCompletion}
	if strings.Contains(strings.ToLower(id), "vl") {
		capabilities = append(capabilities, modelruntime.ModelCapabilityVision)
	}
	manifest := modelruntime.ModelManifest{
		SchemaVersion: "forge.model/v1",
		ID:            id,
		DisplayName:   id,
		Family:        family,
		Format:        format,
		Backend:       modelruntime.BackendOllamaCompat,
		FilePath:      "ollama://" + id,
		SHA256:        "",
		SizeBytes:     item.Size,
		Quantization:  quantization,
		ContextLength: 4096,
		Capabilities:  capabilities,
		License:       "ollama-local",
		Metadata: map[string]any{
			"source":        endpoint,
			"provider":      "ollama",
			"discovered":    true,
			"managed":       false,
			"localCloud":    map[bool]string{true: "cloud", false: "local"}[remote],
			"remote":        remote,
			"remoteHost":    remoteHost,
			"parameterSize": strings.TrimSpace(item.Details.ParameterSize),
		},
	}
	if err := modelruntime.ValidateManifest(manifest); err != nil {
		return modelruntime.ModelManifest{}, err
	}
	return manifest, nil
}

func discoverOpenAICompatModels(ctx context.Context, cfg config.Config) ([]modelruntime.ModelManifest, error) {
	discoveries := make(map[string]modelruntime.ModelManifest)
	var discoveryErr error

	discover := func(endpoint, apiKey string, backend modelruntime.ModelBackendKind) {
		models, err := discoverOpenAICompatibleEndpoint(ctx, endpoint, apiKey, backend)
		if err != nil {
			if discoveryErr == nil {
				discoveryErr = err
			}
			return
		}
		for _, model := range models {
			if _, exists := discoveries[strings.ToLower(model.ID)]; exists {
				continue
			}
			discoveries[strings.ToLower(model.ID)] = model
		}
	}

	if strings.TrimSpace(cfg.ModelOpenAICompatEndpoint) != "" {
		discover(cfg.ModelOpenAICompatEndpoint, cfg.ModelOpenAICompatAPIKey, modelruntime.BackendOpenAICompat)
	}
	if strings.TrimSpace(cfg.ModelVLLMEndpoint) != "" {
		discover(cfg.ModelVLLMEndpoint, cfg.ModelVLLMAPIKey, modelruntime.BackendVLLM)
	}

	if len(discoveries) == 0 {
		if discoveryErr != nil {
			return nil, discoveryErr
		}
		return nil, nil
	}

	result := make([]modelruntime.ModelManifest, 0, len(discoveries))
	for _, manifest := range discoveries {
		result = append(result, manifest)
	}
	return result, nil
}

func discoverOpenAICompatibleEndpoint(ctx context.Context, endpoint, apiKey string, backend modelruntime.ModelBackendKind) ([]modelruntime.ModelManifest, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, nil
	}
	client := &http.Client{Timeout: 5 * time.Second}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint, "/")+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("openai compat model discovery: %w", err)
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai compat model discovery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai compat model discovery: endpoint returned %s", resp.Status)
	}

	body, err := readModelRuntimeDiscoveryResponse(resp.Body, "openai compat model discovery")
	if err != nil {
		return nil, fmt.Errorf("openai compat model discovery: read body: %w", err)
	}
	var payload openAIModelsListResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("openai compat model discovery: unmarshal: %w", err)
	}

	out := make([]modelruntime.ModelManifest, 0, len(payload.Data))
	for _, item := range payload.Data {
		model, err := modelRuntimeDiscoveredManifest(strings.TrimSpace(item.ID), strings.TrimSpace(item.Name), backend, endpoint)
		if err != nil {
			continue
		}
		out = append(out, model)
	}
	return out, nil
}

func modelRuntimeDiscoveredManifest(rawID, rawName string, backend modelruntime.ModelBackendKind, source string) (modelruntime.ModelManifest, error) {
	id := strings.TrimSpace(rawID)
	if id == "" {
		return modelruntime.ModelManifest{}, fmt.Errorf("model id required")
	}
	display := strings.TrimSpace(rawName)
	if display == "" {
		display = id
	}
	manifest := modelruntime.ModelManifest{
		SchemaVersion:  "forge.model/v1",
		ID:             id,
		DisplayName:    display,
		Family:         strings.TrimSpace(string(backend)) + "-remote",
		Format:         modelruntime.ModelFormatGGUF,
		Backend:        backend,
		FilePath:       "remote/" + id,
		SHA256:         "",
		SizeBytes:      0,
		Quantization:   "remote",
		ContextLength:  4096,
		Capabilities:   []modelruntime.ModelCapability{modelruntime.ModelCapabilityChat, modelruntime.ModelCapabilityCompletion},
		License:        "remote",
		DefaultRuntime: modelruntime.ModelRuntimeDefaults{},
		Metadata:       map[string]any{"source": source, "discovered": true},
	}
	if err := modelruntime.ValidateManifest(manifest); err != nil {
		return modelruntime.ModelManifest{}, err
	}
	return manifest, nil
}

func (b *modelRuntimeBridge) ListModels(ctx context.Context, req ModelRuntimeListRequest) ([]ModelRuntimeModel, error) {
	b.refreshDiscoveredModels(ctx, modelruntime.ManagementRequestMeta{
		WorkspaceID:   strings.TrimSpace(req.Meta.WorkspaceID),
		Actor:         "api",
		Source:        "model_runtime_list",
		CorrelationID: strings.TrimSpace(req.Meta.CorrelationID),
		TraceID:       strings.TrimSpace(req.Meta.TraceID),
		Metadata:      map[string]any{"trigger": "list"},
	})
	infos, err := b.runtime.List(ctx)
	if err != nil {
		return nil, mapModelRuntimeBridgeError(err)
	}
	out := make([]ModelRuntimeModel, 0, len(infos))
	for _, info := range infos {
		out = append(out, toModelRuntimeModel(info))
	}
	return out, nil
}

func (b *modelRuntimeBridge) refreshDiscoveredModels(ctx context.Context, meta modelruntime.ManagementRequestMeta) {
	if b == nil || b.runtime == nil {
		return
	}
	discovered := make([]modelruntime.ModelManifest, 0)
	if strings.TrimSpace(b.modelOpenAICompatEndpoint) != "" || strings.TrimSpace(b.modelVLLMEndpoint) != "" {
		cfg := config.Config{
			ModelOpenAICompatEndpoint: strings.TrimSpace(b.modelOpenAICompatEndpoint),
			ModelOpenAICompatAPIKey:   strings.TrimSpace(b.modelOpenAICompatAPIKey),
			ModelVLLMEndpoint:         strings.TrimSpace(b.modelVLLMEndpoint),
			ModelVLLMAPIKey:           strings.TrimSpace(b.modelVLLMAPIKey),
		}
		if models, err := discoverOpenAICompatModels(ctx, cfg); err == nil {
			discovered = append(discovered, models...)
		}
	}
	if b.ollamaDiscoveryEnabled {
		if models, err := discoverLocalOllamaModels(ctx, b.ollamaEndpoint, b.allowOllamaCloudModels); err == nil {
			discovered = append(discovered, models...)
		}
	}
	if len(discovered) == 0 {
		return
	}
	_, _ = b.runtime.RegisterDiscoveredModels(ctx, discovered, meta)
}

func (b *modelRuntimeBridge) GetModel(ctx context.Context, modelID string, _ ModelRuntimeRequestMeta) (ModelRuntimeModel, error) {
	info, err := b.runtime.Inspect(ctx, modelID)
	if err != nil {
		return ModelRuntimeModel{}, mapModelRuntimeBridgeError(err)
	}
	return toModelRuntimeModel(info.ModelInfo), nil
}

func (b *modelRuntimeBridge) ImportModel(ctx context.Context, req ModelRuntimeImportRequest) (ModelRuntimeImportResult, error) {
	result, err := b.runtime.ImportModel(ctx, modelruntime.ImportRequest{
		Path: req.Path,
		ImportModelOptions: modelruntime.ImportModelOptions{
			ID:            req.ID,
			DisplayName:   req.DisplayName,
			Family:        req.Family,
			Backend:       modelruntime.ParseModelBackendKind(req.Backend),
			Capabilities:  parseCapabilities(req.Capabilities),
			License:       req.License,
			Quantization:  req.Quantization,
			ContextLength: req.ContextLength,
			Preferred:     req.Preferred,
			Metadata:      cloneAnyMap(req.Metadata),
		},
		ManagementRequestMeta: modelruntime.ManagementRequestMeta{
			WorkspaceID:   strings.TrimSpace(req.Meta.WorkspaceID),
			Actor:         firstNonEmptyTrimmed(req.Actor, "api"),
			Source:        firstNonEmptyTrimmed(req.Source, "forge_api"),
			CorrelationID: strings.TrimSpace(req.Meta.CorrelationID),
			TraceID:       strings.TrimSpace(req.Meta.TraceID),
			Metadata:      cloneAnyMap(req.Metadata),
		},
	})
	if err != nil {
		return ModelRuntimeImportResult{}, mapModelRuntimeBridgeError(err)
	}
	return ModelRuntimeImportResult{
		Model:       toModelRuntimeModel(result.Model),
		Duplicate:   result.Duplicate,
		ManagedPath: result.ManagedPath,
		SourcePath:  result.SourcePath,
		Warnings:    append([]string(nil), result.Warnings...),
	}, nil
}

func (b *modelRuntimeBridge) ScanModels(ctx context.Context, req ModelRuntimeControlRequest) ([]ModelRuntimeModel, error) {
	b.refreshDiscoveredModels(ctx, toManagementMeta(req))
	infos, err := b.runtime.ScanModels(ctx, modelruntime.ManagementRequestMeta{
		WorkspaceID:   strings.TrimSpace(req.Meta.WorkspaceID),
		Actor:         firstNonEmptyTrimmed(req.Actor, "api"),
		Source:        firstNonEmptyTrimmed(req.Source, "forge_api"),
		CorrelationID: strings.TrimSpace(req.Meta.CorrelationID),
		TraceID:       strings.TrimSpace(req.Meta.TraceID),
		Metadata:      cloneAnyMap(req.Metadata),
	})
	if err != nil {
		return nil, mapModelRuntimeBridgeError(err)
	}
	out := make([]ModelRuntimeModel, 0, len(infos))
	for _, info := range infos {
		out = append(out, toModelRuntimeModel(info))
	}
	return out, nil
}

func (b *modelRuntimeBridge) VerifyModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error) {
	info, err := b.runtime.VerifyModel(ctx, modelID, toManagementMeta(req))
	if err != nil {
		return ModelRuntimeModel{}, mapModelRuntimeBridgeError(err)
	}
	return toModelRuntimeModel(info), nil
}

func (b *modelRuntimeBridge) EnableModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error) {
	info, err := b.runtime.EnableModel(ctx, modelID, toManagementMeta(req))
	if err != nil {
		return ModelRuntimeModel{}, mapModelRuntimeBridgeError(err)
	}
	return toModelRuntimeModel(info), nil
}

func (b *modelRuntimeBridge) DisableModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error) {
	info, err := b.runtime.DisableModel(ctx, modelID, toManagementMeta(req))
	if err != nil {
		return ModelRuntimeModel{}, mapModelRuntimeBridgeError(err)
	}
	return toModelRuntimeModel(info), nil
}

func (b *modelRuntimeBridge) ArchiveModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error) {
	info, err := b.runtime.ArchiveModel(ctx, modelID, toManagementMeta(req))
	if err != nil {
		return ModelRuntimeModel{}, mapModelRuntimeBridgeError(err)
	}
	return toModelRuntimeModel(info), nil
}

func (b *modelRuntimeBridge) RemoveModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeRemoveResult, error) {
	result, err := b.runtime.RemoveModelRegistration(ctx, modelID, toManagementMeta(req))
	if err != nil {
		return ModelRuntimeRemoveResult{}, mapModelRuntimeBridgeError(err)
	}
	return ModelRuntimeRemoveResult{ModelID: result.ModelID, RemovedPath: result.RemovedPath}, nil
}

func (b *modelRuntimeBridge) DeleteModelFiles(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeDeleteFilesResult, error) {
	result, err := b.runtime.DeleteModelFiles(ctx, modelID, toManagementMeta(req))
	if err != nil {
		return ModelRuntimeDeleteFilesResult{}, mapModelRuntimeBridgeError(err)
	}
	return ModelRuntimeDeleteFilesResult{ModelID: result.ModelID, DeletedPath: result.DeletedPath, Deleted: result.Deleted}, nil
}

func (b *modelRuntimeBridge) LoadModel(ctx context.Context, modelID string, _ ModelRuntimeControlRequest) (ModelRuntimeLoadResult, error) {
	loaded, err := b.runtime.Load(ctx, modelID)
	if err != nil {
		return ModelRuntimeLoadResult{}, mapModelRuntimeBridgeError(err)
	}
	return ModelRuntimeLoadResult{
		ModelID:    loaded.ModelID,
		Backend:    string(loaded.Backend),
		Status:     string(loaded.Status),
		Loaded:     true,
		Metadata:   cloneAnyMap(loaded.Metadata),
		LoadedAtMs: loaded.LoadedAt.UnixMilli(),
	}, nil
}

func (b *modelRuntimeBridge) UnloadModel(ctx context.Context, modelID string, _ ModelRuntimeControlRequest) (ModelRuntimeLoadResult, error) {
	if err := b.runtime.Unload(ctx, modelID); err != nil {
		return ModelRuntimeLoadResult{}, mapModelRuntimeBridgeError(err)
	}
	return ModelRuntimeLoadResult{
		ModelID:  modelID,
		Status:   string(modelruntime.StatusAvailable),
		Loaded:   false,
		Metadata: map[string]any{},
	}, nil
}

func (b *modelRuntimeBridge) Chat(ctx context.Context, req ModelRuntimeChatRequest) (ModelRuntimeChatResult, error) {
	if req.ModelID == "" {
		return ModelRuntimeChatResult{}, &modelRuntimeError{
			status:  400,
			code:    "MODEL_REQUIRED",
			message: "model is required",
		}
	}
	if b.maxPromptTokens > 0 {
		promptTokens := approxPromptTokens(req)
		if promptTokens > b.maxPromptTokens {
			return ModelRuntimeChatResult{}, &modelRuntimeError{
				status:  400,
				code:    "PROMPT_TOKENS_EXCEEDED",
				message: fmt.Sprintf("prompt token estimate %d exceeds max %d", promptTokens, b.maxPromptTokens),
			}
		}
	}

	messages := make([]modelruntime.GenerateMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, modelruntime.GenerateMessage{
			Role:    strings.TrimSpace(msg.Role),
			Content: strings.TrimSpace(msg.Content),
		})
	}

	result, err := b.runtime.ExecuteChatRole(ctx, modelruntime.ChatExecutionRequest{
		Role:        resolveModelRuntimeChatRole(req.Role),
		MaxAttempts: req.MaxAttempts,
		GenerateRequest: modelruntime.GenerateRequest{
			ModelID:       strings.TrimSpace(req.ModelID),
			Backend:       modelruntime.ParseModelBackendKind(req.Backend),
			WorkspaceID:   strings.TrimSpace(req.Meta.WorkspaceID),
			Scope:         strings.TrimSpace(req.Meta.WorkspaceID),
			Actor:         firstNonEmptyTrimmed(req.Actor, "api"),
			Source:        firstNonEmptyTrimmed(req.Source, "forge_api"),
			WorkloadClass: modelruntime.ParseGPUWorkloadClass(req.WorkloadClass),
			Messages:      messages,
			Prompt:        strings.TrimSpace(req.Prompt),
			Parameters:    cloneAnyMap(req.Parameters),
			MaxTokens:     req.MaxTokens,
			TimeoutMs:     req.TimeoutMs,
			Stream:        req.Stream,
			CorrelationID: strings.TrimSpace(req.Meta.CorrelationID),
			TraceID:       strings.TrimSpace(req.Meta.TraceID),
			Provenance:    cloneAnyMap(req.Provenance),
			Metadata:      cloneAnyMap(req.Metadata),
		},
	})

	if err != nil {
		return ModelRuntimeChatResult{}, mapModelRuntimeBridgeError(err)
	}

	checkpoint := result.Checkpoint
	if checkpoint.ExecutionID == "" {
		checkpoint = buildModelRuntimeChatCheckpoint(req, result)
	}
	attemptCount := result.AttemptCount
	if attemptCount < 1 {
		attemptCount = checkpoint.AttemptCount
	}
	executionID := strings.TrimSpace(result.ExecutionID)
	if executionID == "" {
		executionID = strings.TrimSpace(checkpoint.ExecutionID)
	}
	role := strings.TrimSpace(string(result.Role))
	if role == "" {
		role = strings.TrimSpace(string(resolveModelRuntimeChatRole(req.Role)))
	}

	return ModelRuntimeChatResult{
		Content:      result.Content,
		FinishReason: result.FinishReason,
		Usage: ModelRuntimeUsage{
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.CompletionTokens,
			TotalTokens:      result.PromptTokens + result.CompletionTokens,
		},
		DurationMs:   result.DurationMs,
		Backend:      string(result.Backend),
		ModelID:      result.ModelID,
		AuditID:      result.AuditID,
		ExecutionID:  executionID,
		AttemptCount: attemptCount,
		Role:         role,
		Checkpoint:   &checkpoint,
		Artifacts:    toArtifactIDs(result.Artifacts),
		Warnings:     append([]string(nil), result.Warnings...),
	}, nil
}

func (b *modelRuntimeBridge) StreamChat(ctx context.Context, req ModelRuntimeChatRequest, onToken func(ModelRuntimeChatStreamToken) error) (ModelRuntimeChatResult, error) {
	if req.ModelID == "" {
		return ModelRuntimeChatResult{}, &modelRuntimeError{
			status:  400,
			code:    "MODEL_REQUIRED",
			message: "model is required",
		}
	}
	if onToken == nil {
		return ModelRuntimeChatResult{}, &modelRuntimeError{
			status:  http.StatusNotImplemented,
			code:    "STREAM_UNSUPPORTED",
			message: "streaming requires a token handler",
		}
	}
	if b.maxPromptTokens > 0 {
		promptTokens := approxPromptTokens(req)
		if promptTokens > b.maxPromptTokens {
			return ModelRuntimeChatResult{}, &modelRuntimeError{
				status:  400,
				code:    "PROMPT_TOKENS_EXCEEDED",
				message: fmt.Sprintf("prompt token estimate %d exceeds max %d", promptTokens, b.maxPromptTokens),
			}
		}
	}

	messages := make([]modelruntime.GenerateMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, modelruntime.GenerateMessage{
			Role:    strings.TrimSpace(msg.Role),
			Content: strings.TrimSpace(msg.Content),
		})
	}

	result, err := b.runtime.ExecuteChatRole(ctx, modelruntime.ChatExecutionRequest{
		Role:        resolveModelRuntimeChatRole(req.Role),
		MaxAttempts: req.MaxAttempts,
		GenerateRequest: modelruntime.GenerateRequest{
			ModelID:       strings.TrimSpace(req.ModelID),
			Backend:       modelruntime.ParseModelBackendKind(req.Backend),
			WorkspaceID:   strings.TrimSpace(req.Meta.WorkspaceID),
			Scope:         strings.TrimSpace(req.Meta.WorkspaceID),
			Actor:         firstNonEmptyTrimmed(req.Actor, "api"),
			Source:        firstNonEmptyTrimmed(req.Source, "forge_api"),
			WorkloadClass: modelruntime.ParseGPUWorkloadClass(req.WorkloadClass),
			Messages:      messages,
			Prompt:        strings.TrimSpace(req.Prompt),
			Parameters:    cloneAnyMap(req.Parameters),
			MaxTokens:     req.MaxTokens,
			TimeoutMs:     req.TimeoutMs,
			Stream:        true,
			StreamHandler: func(event modelruntime.TokenEvent) error {
				return onToken(ModelRuntimeChatStreamToken{
					Text:    event.Token,
					Index:   event.Index,
					Done:    event.Done,
					Backend: string(event.Backend),
					ModelID: event.ModelID,
				})
			},
			CorrelationID: strings.TrimSpace(req.Meta.CorrelationID),
			TraceID:       strings.TrimSpace(req.Meta.TraceID),
			Provenance:    cloneAnyMap(req.Provenance),
			Metadata:      cloneAnyMap(req.Metadata),
		},
	})
	if err != nil {
		return ModelRuntimeChatResult{}, mapModelRuntimeBridgeError(err)
	}

	checkpoint := result.Checkpoint
	if checkpoint.ExecutionID == "" {
		checkpoint = buildModelRuntimeChatCheckpoint(req, result)
	}
	attemptCount := result.AttemptCount
	if attemptCount < 1 {
		attemptCount = checkpoint.AttemptCount
	}
	executionID := strings.TrimSpace(result.ExecutionID)
	if executionID == "" {
		executionID = strings.TrimSpace(checkpoint.ExecutionID)
	}
	role := strings.TrimSpace(string(result.Role))
	if role == "" {
		role = strings.TrimSpace(string(resolveModelRuntimeChatRole(req.Role)))
	}

	return ModelRuntimeChatResult{
		Content:      result.Content,
		FinishReason: result.FinishReason,
		Usage: ModelRuntimeUsage{
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.CompletionTokens,
			TotalTokens:      result.PromptTokens + result.CompletionTokens,
		},
		DurationMs:   result.DurationMs,
		Backend:      string(result.Backend),
		ModelID:      result.ModelID,
		AuditID:      result.AuditID,
		ExecutionID:  executionID,
		AttemptCount: attemptCount,
		Role:         role,
		Checkpoint:   &checkpoint,
		Artifacts:    toArtifactIDs(result.Artifacts),
		Warnings:     append([]string(nil), result.Warnings...),
	}, nil
}

func resolveModelRuntimeChatRole(raw string) modelruntime.ChatExecutionRole {
	return modelruntime.ChatExecutionRole(strings.TrimSpace(raw))
}

func buildModelRuntimeChatCheckpoint(req ModelRuntimeChatRequest, result modelruntime.ChatExecutionResult) modelruntime.ChatExecutionCheckpoint {
	timestamp := time.Now().UTC()
	attemptCount := result.AttemptCount
	if attemptCount < 1 {
		attemptCount = 1
	}
	executionID := strings.TrimSpace(result.ExecutionID)
	if executionID == "" {
		executionID = fmt.Sprintf("chat-%d", time.Now().UnixNano())
	}

	return modelruntime.ChatExecutionCheckpoint{
		ExecutionID:   executionID,
		CorrelationID: strings.TrimSpace(req.Meta.CorrelationID),
		TraceID:       strings.TrimSpace(req.Meta.TraceID),
		WorkspaceID:   strings.TrimSpace(req.Meta.WorkspaceID),
		Role:          resolveModelRuntimeChatRole(req.Role),
		ModelID:       strings.TrimSpace(result.ModelID),
		Backend:       modelruntime.ParseModelBackendKind(strings.TrimSpace(string(result.Backend))),
		State:         modelruntime.ChatExecutionStateCompleted,
		AttemptCount:  attemptCount,
		MaxAttempts:   attemptCount,
		LastError:     "",
		StartedAt:     timestamp,
		UpdatedAt:     timestamp,
		FinishedAt:    timestamp,
		Transitions: []modelruntime.ChatExecutionTransition{{
			State:   modelruntime.ChatExecutionStateCompleted,
			At:      timestamp,
			ModelID: strings.TrimSpace(result.ModelID),
		}},
	}
}

func (b *modelRuntimeBridge) Compatibility(ctx context.Context, modelID string, _ ModelRuntimeRequestMeta) (ModelRuntimeCompatibility, error) {
	report, err := b.runtime.Compatibility(ctx, modelID)
	if err != nil {
		return ModelRuntimeCompatibility{}, mapModelRuntimeBridgeError(err)
	}
	return ModelRuntimeCompatibility{
		ModelID:            report.ModelID,
		Backend:            string(report.Backend),
		Status:             string(report.Status),
		Loaded:             report.Loaded,
		BackendConfigured:  report.BackendConfigured,
		BackendHealthy:     report.BackendHealthy,
		SupportedByBackend: report.SupportedByBackend,
		CanGenerate:        report.CanGenerate,
		Preferred:          report.Preferred,
		Warnings:           append([]string(nil), report.Warnings...),
		Details: map[string]any{
			"backendDetail": report.BackendDetail,
			"metadata":      cloneAnyMap(report.Metadata),
		},
	}, nil
}

func (b *modelRuntimeBridge) Health(ctx context.Context, _ ModelRuntimeRequestMeta) (ModelRuntimeHealth, error) {
	health, err := b.runtime.Health(ctx)
	if err != nil {
		return ModelRuntimeHealth{}, mapModelRuntimeBridgeError(err)
	}

	details := map[string]any{
		"loaded":    map[string]string{},
		"backends":  map[string]map[string]any{},
		"scheduler": map[string]any{},
		"policy": map[string]any{
			"safeModeForceCPUOnly":                 b.safeModeForceCPUOnly,
			"gpuEnabledConfigured":                 b.gpuEnabledConfigured,
			"modelruntimeDegradedOnUnavailableGPU": b.modelruntimeDegradedOnUnavailableGPU,
			"dreamModeGPUOnlyInDeepIdle":           b.dreamModeGPUOnlyInDeepIdle,
		},
	}
	loadedDetails := details["loaded"].(map[string]string)
	for kind, modelID := range health.Loaded {
		loadedDetails[string(kind)] = modelID
	}

	backendDetails := details["backends"].(map[string]map[string]any)
	for kind, backend := range health.Backends {
		backendDetails[string(kind)] = map[string]any{
			"name":        backend.Name,
			"healthy":     backend.Healthy,
			"detail":      backend.Detail,
			"meta":        cloneAnyMap(backend.Meta),
			"supervision": backend.Supervision,
		}
	}
	if health.GPUTelemetry != nil {
		details["gpuTelemetry"] = health.GPUTelemetry
	}
	schedulerDetails := details["scheduler"].(map[string]any)
	schedulerDetails["maxQueueDepth"] = health.Scheduler.MaxQueueDepth
	schedulerDetails["maxConcurrentRequests"] = health.Scheduler.MaxConcurrentRequests
	schedulerDetails["queued"] = len(health.Scheduler.Queued)
	schedulerDetails["running"] = len(health.Scheduler.Running)
	schedulerDetails["completed"] = len(health.Scheduler.Completed)
	schedulerDetails["interactiveQueued"] = health.Scheduler.InteractiveQueued
	schedulerDetails["backgroundQueued"] = health.Scheduler.BackgroundQueued
	schedulerDetails["interactiveRunning"] = health.Scheduler.InteractiveRunning
	schedulerDetails["backgroundRunning"] = health.Scheduler.BackgroundRunning
	schedulerDetails["cooldownJobs"] = health.Scheduler.CooldownJobs

	return ModelRuntimeHealth{
		OK:              health.Healthy,
		Status:          string(health.State),
		Backend:         primaryBackend(health),
		RuntimeEnabled:  health.RuntimeEnabled,
		GPUAware:        health.GPUAware,
		DegradedReasons: append([]string(nil), health.DegradedReasons...),
		PolicyWarnings:  append([]string(nil), health.PolicyWarnings...),
		Details:         details,
	}, nil
}

func (b *modelRuntimeBridge) QueueStatus(ctx context.Context, _ ModelRuntimeRequestMeta) (ModelRuntimeQueueStatus, error) {
	snapshot := b.runtime.SchedulerSnapshot()
	active := map[string]string{}
	for _, loaded := range b.runtime.LoadedModels() {
		if strings.TrimSpace(loaded.ModelID) == "" {
			continue
		}
		active[string(loaded.Backend)] = strings.TrimSpace(loaded.ModelID)
	}
	for _, running := range snapshot.Running {
		if strings.TrimSpace(running.ModelID) == "" {
			continue
		}
		active[string(running.Backend)] = strings.TrimSpace(running.ModelID)
	}
	pending := make([]string, 0, len(snapshot.Queued))
	for _, queued := range snapshot.Queued {
		pending = append(pending, fmt.Sprintf("%s:%s", queued.RequestID, queued.ModelID))
	}
	return ModelRuntimeQueueStatus{
		Depth:       len(snapshot.Queued),
		Active:      active,
		Pending:     pending,
		Scheduler:   "fifo_single_active_per_backend",
		PolicyState: queuePolicyState(snapshot),
	}, nil
}

func (b *modelRuntimeBridge) LoadedStatus(ctx context.Context, _ ModelRuntimeRequestMeta) (ModelRuntimeLoadedStatus, error) {
	models := b.runtime.LoadedModels()
	loaded := make([]ModelRuntimeLoadedModel, 0, len(models))
	for _, info := range models {
		loaded = append(loaded, ModelRuntimeLoadedModel{
			ModelID:    info.ModelID,
			Backend:    string(info.Backend),
			Status:     string(info.Status),
			LoadedAtMs: info.LoadedAt.UnixMilli(),
			Metadata:   cloneAnyMap(info.Metadata),
		})
	}
	return ModelRuntimeLoadedStatus{
		Count:  len(loaded),
		Models: loaded,
	}, nil
}

func (b *modelRuntimeBridge) Usage(ctx context.Context, _ ModelRuntimeRequestMeta) (ModelRuntimeUsageSummary, error) {
	usage, err := b.runtime.Usage(ctx)
	if err != nil {
		return ModelRuntimeUsageSummary{}, mapModelRuntimeBridgeError(err)
	}
	out := map[string]map[string]any{}
	for kind, meta := range usage.Backends {
		out[string(kind)] = cloneAnyMap(meta)
	}
	return ModelRuntimeUsageSummary{
		Registered: usage.Registered,
		Imported:   usage.Imported,
		Verified:   usage.Verified,
		Available:  usage.Available,
		Disabled:   usage.Disabled,
		Archived:   usage.Archived,
		Loaded:     usage.Loaded,
		QueueDepth: usage.QueueDepth,
		Running:    usage.Running,
		Completed:  usage.Completed,
		Backends:   out,
	}, nil
}

func (b *modelRuntimeBridge) Backends(ctx context.Context, _ ModelRuntimeRequestMeta) ([]ModelRuntimeBackendStatus, error) {
	backends, err := b.runtime.Backends(ctx)
	if err != nil {
		return nil, mapModelRuntimeBridgeError(err)
	}
	out := make([]ModelRuntimeBackendStatus, 0, len(backends))
	for _, backend := range backends {
		out = append(out, ModelRuntimeBackendStatus{
			Kind:        string(backend.Kind),
			Name:        backend.Name,
			Healthy:     backend.Healthy,
			Detail:      backend.Detail,
			LoadedModel: backend.LoadedModel,
			Meta:        cloneAnyMap(backend.Meta),
			Supervision: backend.Supervision,
		})
	}
	return out, nil
}

type modelRuntimeAuditAdapter struct {
	auditSvc *audit.Service
}

func (a *modelRuntimeAuditAdapter) RecordModelRuntime(ctx context.Context, record modelruntime.ModelRuntimeAuditRecord) (string, error) {
	if a == nil || a.auditSvc == nil {
		return "", nil
	}

	action := strings.TrimSpace(record.Operation)
	if action == "" {
		action = "event"
	}
	outcome := strings.TrimSpace(record.Outcome)
	if outcome == "" {
		outcome = "ok"
	}
	metadata := cloneAnyMap(record.Metadata)
	riskClass := firstNonEmptyTrimmed(metadataStringAny(metadata, "riskClass"), "low")
	approvalID := metadataStringAny(metadata, "approvalId")
	capabilityID := metadataStringAny(metadata, "capabilityId")

	payload := map[string]any{
		"modelId":       record.ModelID,
		"backend":       string(record.Backend),
		"workspaceId":   record.WorkspaceID,
		"actor":         record.Actor,
		"source":        record.Source,
		"timeoutMs":     record.TimeoutMs,
		"maxTokens":     record.MaxTokens,
		"requestId":     record.RequestID,
		"queueWaitMs":   record.QueueWaitMs,
		"queueDepth":    record.QueueDepth,
		"runningCount":  record.RunningCount,
		"durationMs":    record.DurationMs,
		"promptTokens":  record.PromptTokens,
		"outputTokens":  record.OutputTokens,
		"outputBytes":   record.OutputBytes,
		"error":         record.Error,
		"metadata":      metadata,
		"riskClass":     riskClass,
		"approvalId":    approvalID,
		"capabilityId":  capabilityID,
		"correlationId": record.CorrelationID,
		"traceId":       record.TraceID,
	}

	entry, err := a.auditSvc.Record(ctx, audit.CreateRequest{
		CorrelationID: record.CorrelationID,
		Category:      "model_runtime",
		Action:        "model." + action,
		Actor:         firstNonEmptyTrimmed(record.Actor, "api"),
		SubjectType:   "model",
		SubjectID:     record.ModelID,
		RiskClass:     riskClass,
		Outcome:       outcome,
		Summary:       fmt.Sprintf("model runtime %s %s", action, outcome),
		Payload:       payload,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", entry.ID), nil
}

func mapModelRuntimeBridgeError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, modelruntime.ErrModelIDInvalid):
		return &modelRuntimeError{status: 400, code: "MODEL_ID_INVALID", message: err.Error()}
	case errors.Is(err, modelruntime.ErrModelNotFound):
		return &modelRuntimeError{status: 404, code: "MODEL_NOT_FOUND", message: "model not found"}
	case errors.Is(err, modelruntime.ErrModelNotLoaded):
		return &modelRuntimeError{status: 409, code: "MODEL_NOT_LOADED", message: "model is not loaded"}
	case errors.Is(err, modelruntime.ErrModelLifecycleBusy):
		return &modelRuntimeError{status: 409, code: "MODEL_LIFECYCLE_BUSY", message: err.Error()}
	case errors.Is(err, modelruntime.ErrModelUnavailable):
		return &modelRuntimeError{status: 409, code: "MODEL_UNAVAILABLE", message: err.Error()}
	case errors.Is(err, modelruntime.ErrLoadedModelLimit):
		return &modelRuntimeError{status: 429, code: "MODEL_LOADED_LIMIT", message: err.Error()}
	case errors.Is(err, modelruntime.ErrManagementUnavailable):
		return &modelRuntimeError{status: 503, code: "MODEL_MANAGEMENT_UNAVAILABLE", message: err.Error()}
	case errors.Is(err, modelruntime.ErrModelAlreadyExists):
		return &modelRuntimeError{status: 409, code: "MODEL_ALREADY_EXISTS", message: err.Error()}
	case errors.Is(err, modelruntime.ErrImportPathInvalid):
		return &modelRuntimeError{status: 400, code: "MODEL_IMPORT_PATH_INVALID", message: err.Error()}
	case errors.Is(err, modelruntime.ErrModelSelectionAmbiguous):
		return &modelRuntimeError{status: 409, code: "MODEL_SELECTION_AMBIGUOUS", message: err.Error()}
	case errors.Is(err, modelruntime.ErrUnsupportedBackendOverride):
		return &modelRuntimeError{status: 400, code: "MODEL_BACKEND_OVERRIDE_UNSUPPORTED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrBackendUnavailable):
		return &modelRuntimeError{status: 503, code: "MODEL_BACKEND_UNAVAILABLE", message: err.Error()}
	case errors.Is(err, modelruntime.ErrUnsupportedSpawn):
		return &modelRuntimeError{status: 501, code: "MODEL_BACKEND_UNSUPPORTED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrRequestQueueFull):
		return &modelRuntimeError{status: 429, code: "MODEL_SCHEDULER_BUSY", message: err.Error()}
	case errors.Is(err, modelruntime.ErrBackgroundWorkloadDeferred):
		return &modelRuntimeError{status: 429, code: "MODEL_BACKGROUND_WORKLOAD_DEFERRED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrPolicyDenied):
		return &modelRuntimeError{status: 403, code: "MODEL_POLICY_DENIED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrGPUNotAllowedForInteractive):
		return &modelRuntimeError{status: 503, code: "MODEL_GPU_INTERACTIVE_REQUIRED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrBackgroundJobsDisabled):
		return &modelRuntimeError{status: 403, code: "MODEL_GPU_BACKGROUND_DISABLED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrModelCapabilityUnsupported):
		return &modelRuntimeError{status: 400, code: "MODEL_CAPABILITY_UNSUPPORTED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrActorRequired):
		return &modelRuntimeError{status: 400, code: "ACTOR_REQUIRED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrSourceRequired):
		return &modelRuntimeError{status: 400, code: "SOURCE_REQUIRED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrWorkspaceRequired):
		return &modelRuntimeError{status: 400, code: "WORKSPACE_REQUIRED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrStreamingUnsupported):
		return &modelRuntimeError{status: 501, code: "STREAM_UNSUPPORTED", message: err.Error()}
	case errors.Is(err, modelruntime.ErrProviderCooldownActive):
		return &modelRuntimeError{status: 429, code: "MODEL_PROVIDER_COOLDOWN", message: err.Error()}
	case errors.Is(err, modelruntime.ErrChatRetryExhausted):
		return &modelRuntimeError{status: 503, code: "MODEL_CHAT_RETRY_EXHAUSTED", message: err.Error()}
	case errors.Is(err, context.Canceled):
		return &modelRuntimeError{status: 408, code: "MODEL_REQUEST_CANCELED", message: "model request canceled"}
	case errors.Is(err, context.DeadlineExceeded):
		return &modelRuntimeError{status: 504, code: "MODEL_REQUEST_TIMEOUT", message: "model request timed out"}
	default:
		message := strings.TrimSpace(err.Error())
		lower := strings.ToLower(message)
		if strings.Contains(lower, "scheduler") || strings.Contains(lower, "queue") || strings.Contains(lower, "busy") {
			return &modelRuntimeError{status: 429, code: "MODEL_SCHEDULER_BUSY", message: message}
		}
		if strings.Contains(lower, "policy") || strings.Contains(lower, "permission") || strings.Contains(lower, "denied") {
			return &modelRuntimeError{status: 403, code: "MODEL_POLICY_DENIED", message: message}
		}
		return err
	}
}

func toModelRuntimeModel(info modelruntime.ModelInfo) ModelRuntimeModel {
	caps := make([]string, 0, len(info.Manifest.Capabilities))
	for _, capability := range info.Manifest.Capabilities {
		caps = append(caps, string(capability))
	}

	metadata := cloneAnyMap(info.Manifest.Metadata)
	if info.Loaded != nil {
		if metadata == nil {
			metadata = map[string]any{}
		}
		metadata["loadedAt"] = info.Loaded.LoadedAt.UnixMilli()
		if strings.TrimSpace(info.Loaded.Endpoint) != "" {
			metadata["endpoint"] = info.Loaded.Endpoint
		}
	}

	return ModelRuntimeModel{
		ID:           info.Manifest.ID,
		DisplayName:  info.Manifest.DisplayName,
		Family:       info.Manifest.Family,
		Backend:      string(info.Manifest.Backend),
		Format:       string(info.Manifest.Format),
		Status:       string(info.Status),
		Capabilities: caps,
		Metadata:     metadata,
	}
}

func toArtifactIDs(artifacts []modelruntime.ArtifactReference) []string {
	if len(artifacts) == 0 {
		return nil
	}
	out := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		switch {
		case strings.TrimSpace(artifact.URI) != "":
			out = append(out, strings.TrimSpace(artifact.URI))
		case strings.TrimSpace(artifact.ID) != "":
			out = append(out, strings.TrimSpace(artifact.ID))
		}
	}
	return out
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func approxPromptTokens(req ModelRuntimeChatRequest) int {
	total := len(strings.Fields(req.Prompt))
	for _, msg := range req.Messages {
		total += len(strings.Fields(msg.Content))
	}
	return total
}

func parseCapabilities(input []string) []modelruntime.ModelCapability {
	if len(input) == 0 {
		return nil
	}
	out := make([]modelruntime.ModelCapability, 0, len(input))
	for _, value := range input {
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed == "" {
			continue
		}
		out = append(out, modelruntime.ModelCapability(trimmed))
	}
	return out
}

func toManagementMeta(req ModelRuntimeControlRequest) modelruntime.ManagementRequestMeta {
	return modelruntime.ManagementRequestMeta{
		WorkspaceID:   strings.TrimSpace(req.Meta.WorkspaceID),
		Actor:         firstNonEmptyTrimmed(req.Actor, "api"),
		Source:        firstNonEmptyTrimmed(req.Source, "forge_api"),
		CorrelationID: strings.TrimSpace(req.Meta.CorrelationID),
		TraceID:       strings.TrimSpace(req.Meta.TraceID),
		Metadata:      cloneAnyMap(req.Metadata),
	}
}

func primaryBackend(health modelruntime.RuntimeHealth) string {
	for kind, backend := range health.Backends {
		if backend.Healthy {
			return string(kind)
		}
	}
	for kind := range health.Backends {
		return string(kind)
	}
	return ""
}

func queuePolicyState(snapshot modelruntime.SchedulerSnapshot) string {
	if snapshot.CooldownJobs > 0 {
		return "background_deferred"
	}
	if snapshot.InteractiveQueued > 0 || snapshot.InteractiveRunning > 0 {
		if snapshot.BackgroundQueued > 0 {
			return "interactive_priority_active"
		}
		return "interactive_active"
	}
	return "policy_guarded"
}
