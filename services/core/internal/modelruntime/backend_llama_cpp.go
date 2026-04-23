package modelruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type LlamaCppOptions struct {
	Endpoint        string
	HTTPClient      *http.Client
	RequestTimeout  time.Duration
	MaxOutputTokens int

	AllowSpawn bool
	BinaryPath string
	ExtraArgs  []string

	ChatPath       string
	CompletionPath string
	HealthPath     string
}

type LlamaCppBackend struct {
	endpoint        string
	client          *http.Client
	requestTimeout  time.Duration
	maxOutputTokens int

	allowSpawn bool
	binaryPath string

	chatPath       string
	completionPath string
	healthPath     string

	mu     sync.RWMutex
	loaded map[string]LoadedModel
}

func NewLlamaCppBackend(opts LlamaCppOptions) *LlamaCppBackend {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	chatPath := strings.TrimSpace(opts.ChatPath)
	if chatPath == "" {
		chatPath = "/v1/chat/completions"
	}
	completionPath := strings.TrimSpace(opts.CompletionPath)
	if completionPath == "" {
		completionPath = "/completion"
	}
	healthPath := strings.TrimSpace(opts.HealthPath)
	if healthPath == "" {
		healthPath = "/health"
	}
	return &LlamaCppBackend{
		endpoint:        strings.TrimRight(strings.TrimSpace(opts.Endpoint), "/"),
		client:          client,
		requestTimeout:  opts.RequestTimeout,
		maxOutputTokens: opts.MaxOutputTokens,
		allowSpawn:      opts.AllowSpawn,
		binaryPath:      strings.TrimSpace(opts.BinaryPath),
		chatPath:        withLeadingSlash(chatPath),
		completionPath:  withLeadingSlash(completionPath),
		healthPath:      withLeadingSlash(healthPath),
		loaded:          map[string]LoadedModel{},
	}
}

func (b *LlamaCppBackend) Name() string { return "llama.cpp" }

func (b *LlamaCppBackend) Kind() ModelBackendKind { return BackendLlamaCpp }

func (b *LlamaCppBackend) Supports(format ModelFormat, _ []ModelCapability) bool {
	switch format {
	case ModelFormatGGUF, ModelFormatUnknown:
		return true
	default:
		return false
	}
}

func (b *LlamaCppBackend) Load(_ context.Context, manifest ModelManifest) (LoadedModel, error) {
	if strings.TrimSpace(manifest.ID) == "" {
		return LoadedModel{}, errors.New("manifest.id is required")
	}
	if !b.Supports(manifest.Format, manifest.Capabilities) {
		return LoadedModel{}, fmt.Errorf("backend %s does not support format %s", b.Name(), manifest.Format)
	}
	if b.allowSpawn {
		return LoadedModel{}, fmt.Errorf("%w: spawn mode not enabled in this phase", ErrUnsupportedSpawn)
	}
	if strings.TrimSpace(b.endpoint) == "" {
		return LoadedModel{}, fmt.Errorf("%w: endpoint is empty", ErrBackendUnavailable)
	}
	if _, err := url.ParseRequestURI(b.endpoint); err != nil {
		return LoadedModel{}, fmt.Errorf("%w: invalid endpoint: %v", ErrBackendUnavailable, err)
	}

	loaded := LoadedModel{
		ModelID:  manifest.ID,
		Backend:  BackendLlamaCpp,
		Status:   StatusLoaded,
		LoadedAt: time.Now(),
		Endpoint: b.endpoint,
		Metadata: map[string]any{
			"format": string(manifest.Format),
		},
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.loaded[manifest.ID] = loaded
	return loaded, nil
}

func (b *LlamaCppBackend) Unload(_ context.Context, modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return errors.New("modelID is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.loaded[modelID]; !ok {
		return ErrModelNotLoaded
	}
	delete(b.loaded, modelID)
	return nil
}

func (b *LlamaCppBackend) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	if err := ValidateGenerateRequest(req); err != nil {
		return GenerateResult{}, err
	}
	if b.allowSpawn {
		return GenerateResult{}, fmt.Errorf("%w: llama.cpp spawn mode deferred", ErrUnsupportedSpawn)
	}
	if strings.TrimSpace(b.endpoint) == "" {
		return GenerateResult{}, fmt.Errorf("%w: endpoint is empty", ErrBackendUnavailable)
	}

	b.mu.RLock()
	_, ok := b.loaded[req.ModelID]
	b.mu.RUnlock()
	if !ok {
		return GenerateResult{}, ErrModelNotLoaded
	}

	effectiveMax := EffectiveMaxOutputTokens(req, b.maxOutputTokens)
	timeout := EffectiveTimeout(req, b.requestTimeout)
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := time.Now()
	if len(req.Messages) > 0 {
		result, err := b.generateChat(callCtx, req, effectiveMax)
		if err != nil {
			return GenerateResult{}, err
		}
		result.DurationMs = time.Since(started).Milliseconds()
		result.Backend = BackendLlamaCpp
		result.ModelID = req.ModelID
		return result, nil
	}

	result, err := b.generateCompletion(callCtx, req, effectiveMax)
	if err != nil {
		return GenerateResult{}, err
	}
	result.DurationMs = time.Since(started).Milliseconds()
	result.Backend = BackendLlamaCpp
	result.ModelID = req.ModelID
	return result, nil
}

func (b *LlamaCppBackend) generateChat(ctx context.Context, req GenerateRequest, maxTokens int) (GenerateResult, error) {
	msgPayload := make([]map[string]any, 0, len(req.Messages))
	for _, msg := range req.Messages {
		msgPayload = append(msgPayload, map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	payload := map[string]any{
		"model":    req.ModelID,
		"messages": msgPayload,
		"stream":   false,
	}
	if maxTokens > 0 {
		payload["max_tokens"] = maxTokens
	}
	mergeParameters(payload, req.Parameters, map[string]struct{}{
		"model":      {},
		"messages":   {},
		"stream":     {},
		"max_tokens": {},
	})

	responseBody, err := b.postJSON(ctx, b.chatPath, payload)
	if err != nil {
		return GenerateResult{}, err
	}

	var raw map[string]any
	if err := json.Unmarshal(responseBody, &raw); err != nil {
		return GenerateResult{}, fmt.Errorf("decode llama.cpp chat response: %w", err)
	}

	content := ""
	finishReason := ""
	promptTokens := 0
	completionTokens := 0

	if choices, ok := raw["choices"].([]any); ok && len(choices) > 0 {
		if first, ok := choices[0].(map[string]any); ok {
			if msg, ok := first["message"].(map[string]any); ok {
				content, _ = msg["content"].(string)
			}
			finishReason, _ = first["finish_reason"].(string)
		}
	}
	if usage, ok := raw["usage"].(map[string]any); ok {
		promptTokens = intFromAny(usage["prompt_tokens"])
		completionTokens = intFromAny(usage["completion_tokens"])
	}

	if content == "" {
		return GenerateResult{}, errors.New("llama.cpp chat response missing content")
	}

	bounded, truncated := BoundTextApproxTokens(content, maxTokens)
	result := GenerateResult{
		Content:          bounded,
		FinishReason:     finishReason,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	}
	if result.FinishReason == "" {
		result.FinishReason = "stop"
	}
	if truncated {
		result.FinishReason = "length"
		result.Warnings = append(result.Warnings, ErrOutputBound.Error())
	}
	if result.CompletionTokens == 0 {
		result.CompletionTokens = len(strings.Fields(result.Content))
	}
	if result.PromptTokens == 0 {
		result.PromptTokens = tokenCountApprox(req)
	}
	return result, nil
}

func (b *LlamaCppBackend) generateCompletion(ctx context.Context, req GenerateRequest, maxTokens int) (GenerateResult, error) {
	payload := map[string]any{
		"model":  req.ModelID,
		"prompt": req.Prompt,
		"stream": false,
	}
	if maxTokens > 0 {
		payload["n_predict"] = maxTokens
	}
	mergeParameters(payload, req.Parameters, map[string]struct{}{
		"model":     {},
		"prompt":    {},
		"stream":    {},
		"n_predict": {},
	})

	responseBody, err := b.postJSON(ctx, b.completionPath, payload)
	if err != nil {
		return GenerateResult{}, err
	}

	var raw map[string]any
	if err := json.Unmarshal(responseBody, &raw); err != nil {
		return GenerateResult{}, fmt.Errorf("decode llama.cpp completion response: %w", err)
	}

	content := ""
	finishReason := "stop"
	promptTokens := 0
	completionTokens := 0

	if v, ok := raw["content"].(string); ok {
		content = v
	}
	if v, ok := raw["finish_reason"].(string); ok && v != "" {
		finishReason = v
	}
	if choices, ok := raw["choices"].([]any); ok && len(choices) > 0 {
		if first, ok := choices[0].(map[string]any); ok {
			if text, ok := first["text"].(string); ok && strings.TrimSpace(text) != "" {
				content = text
			}
			if fr, ok := first["finish_reason"].(string); ok && fr != "" {
				finishReason = fr
			}
		}
	}
	if usage, ok := raw["usage"].(map[string]any); ok {
		promptTokens = intFromAny(usage["prompt_tokens"])
		completionTokens = intFromAny(usage["completion_tokens"])
	}
	if completionTokens == 0 {
		completionTokens = intFromAny(raw["tokens_predicted"])
	}
	if promptTokens == 0 {
		promptTokens = intFromAny(raw["tokens_evaluated"])
	}

	if content == "" {
		return GenerateResult{}, errors.New("llama.cpp completion response missing content")
	}

	bounded, truncated := BoundTextApproxTokens(content, maxTokens)
	result := GenerateResult{
		Content:          bounded,
		FinishReason:     finishReason,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	}
	if truncated {
		result.FinishReason = "length"
		result.Warnings = append(result.Warnings, ErrOutputBound.Error())
	}
	if result.PromptTokens == 0 {
		result.PromptTokens = tokenCountApprox(req)
	}
	if result.CompletionTokens == 0 {
		result.CompletionTokens = len(strings.Fields(result.Content))
	}
	return result, nil
}

func (b *LlamaCppBackend) Health(ctx context.Context) (BackendHealth, error) {
	health := BackendHealth{
		Name: b.Name(),
		Kind: b.Kind(),
		Meta: map[string]any{
			"endpoint": b.endpoint,
		},
	}
	if strings.TrimSpace(b.endpoint) == "" {
		health.Healthy = false
		health.Detail = "llama.cpp endpoint not configured"
		return health, fmt.Errorf("%w: endpoint is empty", ErrBackendUnavailable)
	}
	ctx, cancel := context.WithTimeout(ctx, defaultIfZero(b.requestTimeout, 2*time.Second))
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.endpoint+b.healthPath, nil)
	if err != nil {
		health.Healthy = false
		health.Detail = err.Error()
		return health, err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		health.Healthy = false
		health.Detail = err.Error()
		return health, WrapBackendUnavailable(err, "llama.cpp health")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		health.Healthy = false
		health.Detail = fmt.Sprintf("health endpoint returned %s", resp.Status)
		return health, fmt.Errorf("llama.cpp health returned %s", resp.Status)
	}
	health.Healthy = true
	health.Detail = "ok"
	return health, nil
}

func (b *LlamaCppBackend) Inspect(_ context.Context, modelID string) (BackendInspectResult, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return BackendInspectResult{}, errors.New("modelID is required")
	}
	b.mu.RLock()
	loaded, ok := b.loaded[modelID]
	b.mu.RUnlock()
	meta := map[string]any{
		"endpoint": b.endpoint,
		"loaded":   ok,
	}
	if ok {
		meta["loadedAt"] = loaded.LoadedAt
	}
	return BackendInspectResult{
		ModelID: modelID,
		Backend: BackendLlamaCpp,
		Found:   ok,
		Meta:    meta,
	}, nil
}

func (b *LlamaCppBackend) postJSON(ctx context.Context, path string, payload map[string]any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, context.DeadlineExceeded
		}
		if isNetUnavailable(err) {
			return nil, WrapBackendUnavailable(err, "llama.cpp request")
		}
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("llama.cpp returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

func mergeParameters(payload map[string]any, params map[string]any, blocked map[string]struct{}) {
	for k, v := range params {
		if _, ok := blocked[k]; ok {
			continue
		}
		payload[k] = v
	}
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case float32:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case int32:
		return int(t)
	default:
		return 0
	}
}

func withLeadingSlash(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func defaultIfZero(v time.Duration, fallback time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return fallback
}

func isNetUnavailable(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host")
}
