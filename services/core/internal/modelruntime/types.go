package modelruntime

import (
	"errors"
	"strings"
	"time"
)

// ModelFormat identifies on-disk model artifact format.
type ModelFormat string

const (
	ModelFormatGGUF        ModelFormat = "gguf"
	ModelFormatSafeTensors ModelFormat = "safetensors"
	ModelFormatONNX        ModelFormat = "onnx"
	ModelFormatUnknown     ModelFormat = "unknown"
)

// ModelBackendKind identifies the runtime backend implementation.
type ModelBackendKind string

const (
	ModelBackendLlamaCPP     ModelBackendKind = "llama_cpp"
	ModelBackendVLLM         ModelBackendKind = "vllm"
	ModelBackendOllamaCompat ModelBackendKind = "ollama_compat"
	ModelBackendOpenAICompat ModelBackendKind = "openai_compat"
	ModelBackendForgeNative  ModelBackendKind = "forge_native"
	ModelBackendFake         ModelBackendKind = "fake"
)

// ModelCapability declares supported workloads on a model.
type ModelCapability string

const (
	ModelCapabilityChat             ModelCapability = "chat"
	ModelCapabilityCompletion       ModelCapability = "completion"
	ModelCapabilityEmbedding        ModelCapability = "embedding"
	ModelCapabilityRerank           ModelCapability = "rerank"
	ModelCapabilityVision           ModelCapability = "vision"
	ModelCapabilityToolCalling      ModelCapability = "tool_calling"
	ModelCapabilityStructuredOutput ModelCapability = "structured_output"
	ModelCapabilityCode             ModelCapability = "code"
)

// ModelStatus tracks runtime lifecycle state.
type ModelStatus string

const (
	ModelStatusImported    ModelStatus = "imported"
	ModelStatusVerified    ModelStatus = "verified"
	ModelStatusAvailable   ModelStatus = "available"
	ModelStatusLoading     ModelStatus = "loading"
	ModelStatusLoaded      ModelStatus = "loaded"
	ModelStatusUnloading   ModelStatus = "unloading"
	ModelStatusUnavailable ModelStatus = "unavailable"
	ModelStatusError       ModelStatus = "error"
	ModelStatusDisabled    ModelStatus = "disabled"
	ModelStatusArchived    ModelStatus = "archived"
)

// ModelRuntimeDefaults defines backend-facing defaults from the manifest.
type ModelRuntimeDefaults struct {
	MaxPromptTokens int            `json:"maxPromptTokens,omitempty"`
	MaxOutputTokens int            `json:"maxOutputTokens,omitempty"`
	MaxTokens       int            `json:"maxTokens,omitempty"`
	TimeoutMs       int            `json:"timeoutMs,omitempty"`
	Temperature     float64        `json:"temperature,omitempty"`
	TopP            float64        `json:"topP,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// ModelManifest is the FORGE-native model manifest.
type ModelManifest struct {
	SchemaVersion  string               `json:"schemaVersion"`
	ID             string               `json:"id"`
	DisplayName    string               `json:"displayName"`
	Family         string               `json:"family"`
	Format         ModelFormat          `json:"format"`
	Backend        ModelBackendKind     `json:"backend"`
	FilePath       string               `json:"filePath,omitempty"`
	SHA256         string               `json:"sha256"`
	SizeBytes      int64                `json:"sizeBytes"`
	Quantization   string               `json:"quantization"`
	ContextLength  int                  `json:"contextLength"`
	Capabilities   []ModelCapability    `json:"capabilities"`
	DefaultRuntime ModelRuntimeDefaults `json:"defaultRuntime"`
	License        string               `json:"license"`
	Metadata       map[string]any       `json:"metadata"`
}

// LoadedModel tracks loaded model process/endpoint state.
type LoadedModel struct {
	ModelID       string           `json:"modelId"`
	Backend       ModelBackendKind `json:"backend"`
	Status        ModelStatus      `json:"status"`
	LoadedAt      time.Time        `json:"loadedAt"`
	ProcessID     int              `json:"pid,omitempty"`
	Endpoint      string           `json:"endpoint,omitempty"`
	ResourceUsage ResourceUsage    `json:"resourceUsage"`
	Metadata      map[string]any   `json:"metadata"`
}

// ResourceUsage reports runtime resource usage snapshots.
type ResourceUsage struct {
	RAMBytes     int64 `json:"ramBytes,omitempty"`
	VRAMBytes    int64 `json:"vramBytes,omitempty"`
	Threads      int   `json:"threads,omitempty"`
	GPULayers    int   `json:"gpuLayers,omitempty"`
	ContextInUse int   `json:"contextInUse,omitempty"`
}

// GenerateMessage is a normalized chat-message payload.
type GenerateMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// GenerateRequest carries model invocation input and policy metadata.
type GenerateRequest struct {
	ModelID       string                 `json:"modelId"`
	Backend       ModelBackendKind       `json:"backend,omitempty"`
	WorkspaceID   string                 `json:"workspaceId,omitempty"`
	Scope         string                 `json:"scope,omitempty"`
	Actor         string                 `json:"actor,omitempty"`
	Source        string                 `json:"source,omitempty"`
	WorkloadClass GPUWorkloadClass       `json:"workloadClass,omitempty"`
	Messages      []GenerateMessage      `json:"messages,omitempty"`
	Prompt        string                 `json:"prompt,omitempty"`
	Parameters    map[string]any         `json:"parameters,omitempty"`
	MaxTokens     int                    `json:"maxTokens"`
	TimeoutMs     int                    `json:"timeoutMs"`
	Stream        bool                   `json:"stream"`
	StreamHandler func(TokenEvent) error `json:"-"`
	CorrelationID string                 `json:"correlationId,omitempty"`
	TraceID       string                 `json:"traceId,omitempty"`
	Provenance    map[string]any         `json:"provenance,omitempty"`
	Metadata      map[string]any         `json:"metadata,omitempty"`
}

// ArtifactReference points to evidence generated by model calls.
type ArtifactReference struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	URI  string `json:"uri,omitempty"`
}

// GenerateResult contains normalized model output metadata.
type GenerateResult struct {
	Content          string              `json:"content"`
	FinishReason     string              `json:"finishReason"`
	PromptTokens     int                 `json:"promptTokens"`
	CompletionTokens int                 `json:"completionTokens"`
	DurationMs       int64               `json:"durationMs"`
	Backend          ModelBackendKind    `json:"backend"`
	ModelID          string              `json:"modelId"`
	AuditID          string              `json:"auditId,omitempty"`
	Artifacts        []ArtifactReference `json:"artifacts,omitempty"`
	Warnings         []string            `json:"warnings,omitempty"`
	Errors           []string            `json:"errors,omitempty"`
}

// TokenEvent captures incremental streaming output.
type TokenEvent struct {
	Token         string           `json:"token"`
	Index         int              `json:"index"`
	Done          bool             `json:"done"`
	Backend       ModelBackendKind `json:"backend"`
	ModelID       string           `json:"modelId"`
	CorrelationID string           `json:"correlationId,omitempty"`
	TraceID       string           `json:"traceId,omitempty"`
	Error         string           `json:"error,omitempty"`
}

var (
	ErrModelNotFound = errors.New("model not found")
)

type GPUWorkloadClass string

const (
	GPUWorkloadInteractiveInference GPUWorkloadClass = "INTERACTIVE_INFERENCE"
	GPUWorkloadInteractiveEmbedding GPUWorkloadClass = "INTERACTIVE_EMBEDDING"
	GPUWorkloadBackgroundEmbedding  GPUWorkloadClass = "BACKGROUND_EMBEDDING"
	GPUWorkloadBackgroundRerank     GPUWorkloadClass = "BACKGROUND_RERANK"
	GPUWorkloadDreamDistillation    GPUWorkloadClass = "DREAM_DISTILLATION"
	GPUWorkloadAdapterEval          GPUWorkloadClass = "ADAPTER_EVAL"
	GPUWorkloadAdapterTraining      GPUWorkloadClass = "ADAPTER_TRAINING"
	GPUWorkloadUnknown              GPUWorkloadClass = ""
)

func ParseGPUWorkloadClass(raw string) GPUWorkloadClass {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	switch normalized {
	case string(GPUWorkloadInteractiveInference), string(GPUWorkloadInteractiveEmbedding), string(GPUWorkloadBackgroundEmbedding), string(GPUWorkloadBackgroundRerank), string(GPUWorkloadDreamDistillation), string(GPUWorkloadAdapterEval), string(GPUWorkloadAdapterTraining):
		return GPUWorkloadClass(normalized)
	case "INTERACTIVE", "INFERENCE":
		return GPUWorkloadInteractiveInference
	case "BACKGROUND":
		return GPUWorkloadBackgroundEmbedding
	default:
		return GPUWorkloadUnknown
	}
}

func (c GPUWorkloadClass) IsInteractive() bool {
	switch c {
	case GPUWorkloadInteractiveInference, GPUWorkloadInteractiveEmbedding, GPUWorkloadUnknown:
		return true
	default:
		return false
	}
}

func (c GPUWorkloadClass) IsBackground() bool {
	switch c {
	case GPUWorkloadBackgroundEmbedding, GPUWorkloadBackgroundRerank, GPUWorkloadDreamDistillation, GPUWorkloadAdapterEval, GPUWorkloadAdapterTraining:
		return true
	default:
		return false
	}
}

var validFormats = map[ModelFormat]struct{}{
	ModelFormatGGUF:        {},
	ModelFormatSafeTensors: {},
	ModelFormatONNX:        {},
	ModelFormatUnknown:     {},
}

var validBackends = map[ModelBackendKind]struct{}{
	ModelBackendLlamaCPP:     {},
	ModelBackendVLLM:         {},
	ModelBackendOllamaCompat: {},
	ModelBackendOpenAICompat: {},
	ModelBackendForgeNative:  {},
	ModelBackendFake:         {},
}

var validCapabilities = map[ModelCapability]struct{}{
	ModelCapabilityChat:             {},
	ModelCapabilityCompletion:       {},
	ModelCapabilityEmbedding:        {},
	ModelCapabilityRerank:           {},
	ModelCapabilityVision:           {},
	ModelCapabilityToolCalling:      {},
	ModelCapabilityStructuredOutput: {},
	ModelCapabilityCode:             {},
}

var validStatuses = map[ModelStatus]struct{}{
	ModelStatusImported:    {},
	ModelStatusVerified:    {},
	ModelStatusAvailable:   {},
	ModelStatusLoading:     {},
	ModelStatusLoaded:      {},
	ModelStatusUnloading:   {},
	ModelStatusUnavailable: {},
	ModelStatusError:       {},
	ModelStatusDisabled:    {},
	ModelStatusArchived:    {},
}
