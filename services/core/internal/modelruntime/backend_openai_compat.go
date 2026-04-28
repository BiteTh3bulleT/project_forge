package modelruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type OpenAICompatOptions struct {
	Name            string
	Kind            ModelBackendKind
	Endpoint        string
	APIKey          string
	HTTPClient      *http.Client
	RequestTimeout  time.Duration
	MaxOutputTokens int
	ChatPath        string
	ModelsPath      string
}

type OpenAICompatBackend struct {
	name            string
	kind            ModelBackendKind
	endpoint        string
	apiKey          string
	client          *http.Client
	requestTimeout  time.Duration
	maxOutputTokens int
	chatPath        string
	modelsPath      string

	mu     sync.RWMutex
	loaded map[string]LoadedModel
}

func NewOpenAICompatBackend(opts OpenAICompatOptions) *OpenAICompatBackend {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = "openai-compatible"
	}
	kind := opts.Kind
	if kind == "" {
		kind = BackendOpenAICompat
	}
	chatPath := strings.TrimSpace(opts.ChatPath)
	if chatPath == "" {
		chatPath = "/v1/chat/completions"
	}
	modelsPath := strings.TrimSpace(opts.ModelsPath)
	if modelsPath == "" {
		modelsPath = "/v1/models"
	}
	return &OpenAICompatBackend{
		name:            name,
		kind:            kind,
		endpoint:        strings.TrimRight(strings.TrimSpace(opts.Endpoint), "/"),
		apiKey:          strings.TrimSpace(opts.APIKey),
		client:          client,
		requestTimeout:  opts.RequestTimeout,
		maxOutputTokens: opts.MaxOutputTokens,
		chatPath:        withLeadingSlash(chatPath),
		modelsPath:      withLeadingSlash(modelsPath),
		loaded:          map[string]LoadedModel{},
	}
}

func (b *OpenAICompatBackend) Name() string { return b.name }

func (b *OpenAICompatBackend) Kind() ModelBackendKind { return b.kind }

func (b *OpenAICompatBackend) Supports(_ ModelFormat, _ []ModelCapability) bool { return true }

func (b *OpenAICompatBackend) Load(_ context.Context, manifest ModelManifest) (LoadedModel, error) {
	if strings.TrimSpace(manifest.ID) == "" {
		return LoadedModel{}, errors.New("manifest.id is required")
	}
	if strings.TrimSpace(b.endpoint) == "" {
		return LoadedModel{}, fmt.Errorf("%w: endpoint is empty", ErrBackendUnavailable)
	}
	if _, err := url.ParseRequestURI(b.endpoint); err != nil {
		return LoadedModel{}, fmt.Errorf("%w: invalid endpoint: %v", ErrBackendUnavailable, err)
	}
	loaded := LoadedModel{ModelID: manifest.ID, Backend: b.kind, Status: StatusLoaded, LoadedAt: time.Now().UTC(), Endpoint: b.endpoint, Metadata: map[string]any{"format": string(manifest.Format)}}
	b.mu.Lock()
	b.loaded[manifest.ID] = loaded
	b.mu.Unlock()
	return loaded, nil
}

func (b *OpenAICompatBackend) Unload(_ context.Context, modelID string) error {
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

func (b *OpenAICompatBackend) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	if err := ValidateGenerateRequest(req); err != nil {
		return GenerateResult{}, err
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
	payloadMessages := make([]map[string]any, 0, len(req.Messages))
	if len(req.Messages) > 0 {
		for _, msg := range req.Messages {
			payloadMessages = append(payloadMessages, map[string]any{"role": msg.Role, "content": msg.Content})
		}
	} else {
		payloadMessages = append(payloadMessages, map[string]any{"role": "user", "content": req.Prompt})
	}
	payload := map[string]any{"model": req.ModelID, "messages": payloadMessages, "stream": false}
	if maxTokens := EffectiveMaxOutputTokens(req, b.maxOutputTokens); maxTokens > 0 {
		payload["max_tokens"] = maxTokens
	}
	mergeParameters(payload, req.Parameters, map[string]struct{}{"model": {}, "messages": {}, "stream": {}, "max_tokens": {}})
	callCtx, cancel := context.WithTimeout(ctx, EffectiveTimeout(req, b.requestTimeout))
	defer cancel()
	body, err := b.postJSON(callCtx, b.chatPath, payload)
	if err != nil {
		return GenerateResult{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return GenerateResult{}, fmt.Errorf("decode openai-compatible response: %w", err)
	}
	content := ""
	finishReason := ""
	warnings := []string(nil)
	promptTokens := 0
	completionTokens := 0
	content, finishReason, warnings = extractOpenAICompatGeneration(raw)
	if usage, ok := raw["usage"].(map[string]any); ok {
		promptTokens = intFromAny(usage["prompt_tokens"])
		completionTokens = intFromAny(usage["completion_tokens"])
	}
	if content == "" {
		return GenerateResult{}, errors.New("openai-compatible response missing content")
	}
	result := GenerateResult{Content: content, FinishReason: finishReason, PromptTokens: promptTokens, CompletionTokens: completionTokens, Backend: b.kind, ModelID: req.ModelID, Warnings: warnings}
	if result.FinishReason == "" {
		result.FinishReason = "stop"
	}
	if result.PromptTokens == 0 {
		result.PromptTokens = tokenCountApprox(req)
	}
	if result.CompletionTokens == 0 {
		result.CompletionTokens = len(strings.Fields(result.Content))
	}
	return result, nil
}

func extractOpenAICompatGeneration(raw map[string]any) (string, string, []string) {
	warnings := []string(nil)
	finishReason := ""
	if choices, ok := raw["choices"].([]any); ok && len(choices) > 0 {
		if first, ok := choices[0].(map[string]any); ok {
			finishReason, _ = first["finish_reason"].(string)
			if content := openAICompatContentFromChoice(first, false); strings.TrimSpace(content) != "" {
				return content, finishReason, warnings
			}
			if content := openAICompatContentFromChoice(first, true); strings.TrimSpace(content) != "" {
				warnings = append(warnings, "openai-compatible response used reasoning-content fallback")
				return content, finishReason, warnings
			}
		}
	}
	if content := openAICompatTextFromValue(raw["output_text"], false); strings.TrimSpace(content) != "" {
		return content, finishReason, warnings
	}
	if content := openAICompatTextFromValue(raw["response"], false); strings.TrimSpace(content) != "" {
		return content, finishReason, warnings
	}
	if content := openAICompatTextFromValue(raw["content"], false); strings.TrimSpace(content) != "" {
		return content, finishReason, warnings
	}
	return "", finishReason, warnings
}

func openAICompatContentFromChoice(choice map[string]any, includeReasoning bool) string {
	if message, ok := choice["message"].(map[string]any); ok {
		if content := openAICompatTextFromValue(message["content"], includeReasoning); strings.TrimSpace(content) != "" {
			return content
		}
		if includeReasoning {
			for _, key := range []string{"reasoning_content", "reasoning", "thinking"} {
				if content := openAICompatTextFromValue(message[key], true); strings.TrimSpace(content) != "" {
					return content
				}
			}
		}
	}
	for _, key := range []string{"text", "content", "output_text"} {
		if content := openAICompatTextFromValue(choice[key], includeReasoning); strings.TrimSpace(content) != "" {
			return content
		}
	}
	return ""
}

func openAICompatTextFromValue(value any, includeReasoning bool) string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return ""
		}
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, part := range typed {
			if text := openAICompatTextFromPart(part, includeReasoning); strings.TrimSpace(text) != "" {
				parts = append(parts, strings.TrimSpace(text))
			}
		}
		return strings.Join(parts, "\n")
	case []map[string]any:
		parts := make([]string, 0, len(typed))
		for _, part := range typed {
			if text := openAICompatTextFromPart(part, includeReasoning); strings.TrimSpace(text) != "" {
				parts = append(parts, strings.TrimSpace(text))
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		return openAICompatTextFromPart(typed, includeReasoning)
	default:
		return ""
	}
}

func openAICompatTextFromPart(part any, includeReasoning bool) string {
	switch typed := part.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return ""
		}
		return typed
	case map[string]any:
		partType, _ := typed["type"].(string)
		partType = strings.ToLower(strings.TrimSpace(partType))
		if !includeReasoning && strings.Contains(partType, "reason") {
			return ""
		}
		for _, key := range []string{"text", "content", "output_text"} {
			if text := openAICompatTextFromValue(typed[key], includeReasoning); strings.TrimSpace(text) != "" {
				return text
			}
		}
		if includeReasoning {
			for _, key := range []string{"reasoning_content", "reasoning", "thinking"} {
				if text := openAICompatTextFromValue(typed[key], true); strings.TrimSpace(text) != "" {
					return text
				}
			}
		}
		return ""
	default:
		return ""
	}
}

func (b *OpenAICompatBackend) Health(ctx context.Context) (BackendHealth, error) {
	if strings.TrimSpace(b.endpoint) == "" {
		return BackendHealth{Name: b.name, Kind: b.kind, Healthy: false, Detail: "endpoint not configured"}, fmt.Errorf("%w: endpoint is empty", ErrBackendUnavailable)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.endpoint+b.modelsPath, nil)
	if err != nil {
		return BackendHealth{Name: b.name, Kind: b.kind, Healthy: false, Detail: err.Error()}, err
	}
	if b.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.apiKey)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return BackendHealth{Name: b.name, Kind: b.kind, Healthy: false, Detail: err.Error()}, WrapBackendUnavailable(err, b.name)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return BackendHealth{Name: b.name, Kind: b.kind, Healthy: false, Detail: resp.Status}, fmt.Errorf("%w: %s returned %s", ErrBackendUnavailable, b.name, resp.Status)
	}
	b.mu.RLock()
	loadedCount := len(b.loaded)
	b.mu.RUnlock()
	return BackendHealth{Name: b.name, Kind: b.kind, Healthy: true, Detail: "openai-compatible backend reachable", Meta: map[string]any{"loaded": loadedCount}}, nil
}

func (b *OpenAICompatBackend) Inspect(_ context.Context, modelID string) (BackendInspectResult, error) {
	b.mu.RLock()
	_, ok := b.loaded[strings.TrimSpace(modelID)]
	b.mu.RUnlock()
	return BackendInspectResult{ModelID: strings.TrimSpace(modelID), Backend: b.kind, Found: ok, Meta: map[string]any{"loaded": ok, "endpoint": b.endpoint}}, nil
}

func (b *OpenAICompatBackend) postJSON(ctx context.Context, path string, payload map[string]any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if b.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.apiKey)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, WrapBackendUnavailable(err, b.name)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: %s returned %s: %s", ErrBackendUnavailable, b.name, resp.Status, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}
