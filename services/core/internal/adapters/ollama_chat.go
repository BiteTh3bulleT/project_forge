package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaChat calls POST /api/chat (tools-capable). Returns the decoded top-level JSON object.
func (o Ollama) OllamaChat(ctx context.Context, baseURL, model string, messages []map[string]any, tools []map[string]any, toolChoice any, timeout time.Duration) (map[string]any, error) {
	candidates := uniqueStrings([]string{
		strings.TrimSpace(model),
		normalizeOllamaModel(model),
	})
	var lastErr error
	for _, candidate := range candidates {
		out, err := o.ollamaChatWithModel(ctx, baseURL, candidate, messages, tools, toolChoice, timeout)
		if err == nil {
			return out, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ollama /api/chat did not return a response")
	}
	return nil, lastErr
}

func (o Ollama) ollamaChatWithModel(ctx context.Context, baseURL, model string, messages []map[string]any, tools []map[string]any, toolChoice any, timeout time.Duration) (map[string]any, error) {
	payload := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	if len(tools) > 0 {
		payload["tools"] = tools
	}
	if toolChoice != nil {
		payload["tool_choice"] = toolChoice
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/chat", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama /api/chat returned %s: %s", res.Status, strings.TrimSpace(string(body)))
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode /api/chat: %w", err)
	}
	return out, nil
}

func uniqueStrings(input []string) []string {
	out := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
