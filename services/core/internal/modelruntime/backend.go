package modelruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Compatibility aliases for M1 naming used by backend/service code.
const (
	ModelFormatSafeTensor = ModelFormatSafeTensors
)

const (
	BackendLlamaCpp     = ModelBackendLlamaCPP
	BackendVLLM         = ModelBackendVLLM
	BackendOllamaCompat = ModelBackendOllamaCompat
	BackendOpenAICompat = ModelBackendOpenAICompat
	BackendForgeNative  = ModelBackendForgeNative
	BackendFake         = ModelBackendFake
)

const (
	CapabilityChat             = ModelCapabilityChat
	CapabilityCompletion       = ModelCapabilityCompletion
	CapabilityEmbedding        = ModelCapabilityEmbedding
	CapabilityRerank           = ModelCapabilityRerank
	CapabilityVision           = ModelCapabilityVision
	CapabilityToolCalling      = ModelCapabilityToolCalling
	CapabilityStructuredOutput = ModelCapabilityStructuredOutput
	CapabilityCode             = ModelCapabilityCode
)

const (
	StatusImported    = ModelStatusImported
	StatusVerified    = ModelStatusVerified
	StatusAvailable   = ModelStatusAvailable
	StatusLoading     = ModelStatusLoading
	StatusLoaded      = ModelStatusLoaded
	StatusUnloading   = ModelStatusUnloading
	StatusUnavailable = ModelStatusUnavailable
	StatusError       = ModelStatusError
	StatusDisabled    = ModelStatusDisabled
	StatusArchived    = ModelStatusArchived
)

type RuntimeDefaults = ModelRuntimeDefaults
type ChatMessage = GenerateMessage

type BackendHealth struct {
	Name        string                     `json:"name"`
	Kind        ModelBackendKind           `json:"kind"`
	Healthy     bool                       `json:"healthy"`
	Detail      string                     `json:"detail,omitempty"`
	Meta        map[string]any             `json:"meta,omitempty"`
	Supervision BackendSupervisionSnapshot `json:"supervision,omitempty"`
}

type BackendSupervisionSnapshot struct {
	LastProbeAt         time.Time `json:"lastProbeAt,omitempty"`
	ConsecutiveFailures int       `json:"consecutiveFailures"`
	LastError           string    `json:"lastError,omitempty"`
	RestartSupported    bool      `json:"restartSupported"`
	RestartAttempted    bool      `json:"restartAttempted"`
	RestartReason       string    `json:"restartReason,omitempty"`
}

type BackendInspectResult struct {
	ModelID string           `json:"modelId"`
	Backend ModelBackendKind `json:"backend"`
	Found   bool             `json:"found"`
	Meta    map[string]any   `json:"meta,omitempty"`
}

// ModelBackend provides pluggable low-level inference implementations.
// Backends perform inference only. Governance, policy, and audit are handled above this layer.
type ModelBackend interface {
	Name() string
	Kind() ModelBackendKind
	Supports(format ModelFormat, capabilities []ModelCapability) bool
	Load(ctx context.Context, manifest ModelManifest) (LoadedModel, error)
	Unload(ctx context.Context, modelID string) error
	Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error)
	Health(ctx context.Context) (BackendHealth, error)
	Inspect(ctx context.Context, modelID string) (BackendInspectResult, error)
}

// StreamingModelBackend is implemented by backends that can return generation
// deltas before the final response. The service layer still owns governance,
// queueing, audit, and output bounding.
type StreamingModelBackend interface {
	GenerateStream(ctx context.Context, req GenerateRequest, onToken func(TokenEvent) error) (GenerateResult, error)
}

var (
	ErrBackendUnavailable          = errors.New("model backend unavailable")
	ErrUnsupportedSpawn            = errors.New("backend spawn mode unsupported in m1")
	ErrModelNotLoaded              = errors.New("model not loaded")
	ErrOutputBound                 = errors.New("output exceeded configured bound")
	ErrOutputBytesBound            = errors.New("output exceeded configured byte bound")
	ErrRequestQueueFull            = errors.New("model scheduler queue full")
	ErrModelLifecycleBusy          = errors.New("model lifecycle busy")
	ErrModelUnavailable            = errors.New("model unavailable")
	ErrActorRequired               = errors.New("actor is required")
	ErrSourceRequired              = errors.New("source is required")
	ErrWorkspaceRequired           = errors.New("workspace scope is required")
	ErrModelCapabilityUnsupported  = errors.New("model capability unsupported")
	ErrStreamingUnsupported        = errors.New("streaming is unsupported")
	ErrPolicyDenied                = errors.New("model policy denied")
	ErrLoadedModelLimit            = errors.New("max loaded models exceeded")
	ErrManagementUnavailable       = errors.New("model management unavailable")
	ErrModelAlreadyExists          = errors.New("model already exists")
	ErrModelArchived               = errors.New("model is archived")
	ErrModelSelectionAmbiguous     = errors.New("model selection is ambiguous")
	ErrImportPathInvalid           = errors.New("model import path invalid")
	ErrUnsupportedBackendOverride  = errors.New("requested backend override is unsupported")
	ErrProviderCooldownActive      = errors.New("model provider cooldown active")
	ErrChatRetryExhausted          = errors.New("chat execution retry exhausted")
	ErrGPUNotAllowedForInteractive = errors.New("gpu policy blocks interactive inference")
	ErrBackgroundJobsDisabled      = errors.New("background GPU jobs are disabled")
	ErrBackgroundWorkloadDeferred  = errors.New("background GPU work deferred for interactive priority")
)

func ValidateGenerateRequest(req GenerateRequest) error {
	if strings.TrimSpace(req.ModelID) == "" {
		return errors.New("modelId is required")
	}
	if len(req.Messages) == 0 && strings.TrimSpace(req.Prompt) == "" {
		return errors.New("messages or prompt is required")
	}
	if req.TimeoutMs < 0 {
		return errors.New("timeoutMs must be >= 0")
	}
	if req.MaxTokens < 0 {
		return errors.New("maxTokens must be >= 0")
	}
	return nil
}

func BoundTextBytes(input string, maxBytes int) (string, bool) {
	if maxBytes <= 0 {
		return input, false
	}
	if len([]byte(input)) <= maxBytes {
		return input, false
	}
	if maxBytes <= 0 {
		return "", true
	}
	out := make([]rune, 0, len(input))
	size := 0
	for _, r := range input {
		rb := len(string(r))
		if size+rb > maxBytes {
			break
		}
		out = append(out, r)
		size += rb
	}
	return string(out), true
}

func EffectiveTimeout(req GenerateRequest, fallback time.Duration) time.Duration {
	if req.TimeoutMs > 0 {
		return time.Duration(req.TimeoutMs) * time.Millisecond
	}
	if fallback > 0 {
		return fallback
	}
	return 30 * time.Second
}

func EffectiveMaxOutputTokens(req GenerateRequest, fallback int) int {
	effective := fallback
	if req.MaxTokens > 0 {
		if effective <= 0 || req.MaxTokens < effective {
			effective = req.MaxTokens
		}
	}
	return effective
}

func BoundTextApproxTokens(input string, maxTokens int) (string, bool) {
	if maxTokens <= 0 {
		return input, false
	}
	tokens := strings.Fields(input)
	if len(tokens) <= maxTokens {
		return input, false
	}
	return strings.Join(tokens[:maxTokens], " "), true
}

func WrapBackendUnavailable(err error, detail string) error {
	if err == nil {
		return nil
	}
	if detail == "" {
		return fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
	}
	return fmt.Errorf("%w: %s: %v", ErrBackendUnavailable, detail, err)
}
