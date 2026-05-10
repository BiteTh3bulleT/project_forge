package adapters

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

// StreamChat calls POST /api/chat with stream enabled. It is the path that most
// closely matches Ollama CLI/app latency because tokens are surfaced as soon as
// Ollama sends them.
func (o Ollama) StreamChat(ctx context.Context, baseURL, model string, messages []map[string]any, timeout time.Duration, onToken func(string) error) (string, map[string]any, error) {
	candidates := uniqueStrings([]string{
		strings.TrimSpace(model),
		normalizeOllamaModel(model),
	})
	var lastErr error
	for _, candidate := range candidates {
		content, meta, err := o.streamChatWithModel(ctx, baseURL, candidate, messages, timeout, onToken)
		if err == nil {
			return content, meta, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ollama /api/chat stream did not return a response")
	}
	return "", nil, lastErr
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

	if res.StatusCode >= 300 {
		body, err := readOllamaErrorBody(res.Body)
		if err != nil {
			return nil, fmt.Errorf("read /api/chat error response: %w", err)
		}
		return nil, fmt.Errorf("ollama /api/chat returned %s: %s", res.Status, body)
	}
	var out map[string]any
	if err := decodeOllamaJSONBody(res.Body, &out); err != nil {
		return nil, fmt.Errorf("decode /api/chat: %w", err)
	}
	return out, nil
}

func (o Ollama) streamChatWithModel(ctx context.Context, baseURL, model string, messages []map[string]any, timeout time.Duration, onToken func(string) error) (string, map[string]any, error) {
	payload := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/chat", bytes.NewReader(b))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := o.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		body, err := readOllamaErrorBody(res.Body)
		if err != nil {
			return "", nil, fmt.Errorf("read /api/chat stream error response: %w", err)
		}
		return "", nil, fmt.Errorf("ollama /api/chat stream returned %s: %s", res.Status, body)
	}

	var acc strings.Builder
	var lastMeta map[string]any
	sc := bufio.NewScanner(res.Body)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		if msg, _ := obj["message"].(map[string]any); msg != nil {
			if tok, _ := msg["content"].(string); tok != "" {
				acc.WriteString(tok)
				if onToken != nil {
					if err := onToken(tok); err != nil {
						return acc.String(), lastMeta, err
					}
				}
			}
		}
		if done, _ := obj["done"].(bool); done {
			lastMeta = map[string]any{
				"done":                  obj["done"],
				"totalDuration":         obj["total_duration"],
				"loadDuration":          obj["load_duration"],
				"promptEvalCount":       obj["prompt_eval_count"],
				"evalCount":             obj["eval_count"],
				"promptEvalDuration":    obj["prompt_eval_duration"],
				"evalDuration":          obj["eval_duration"],
				"promptEvalRate":        obj["prompt_eval_rate"],
				"evalRate":              obj["eval_rate"],
				"nativeOllamaStreaming": true,
			}
			break
		}
	}
	if err := sc.Err(); err != nil {
		return acc.String(), lastMeta, err
	}
	return acc.String(), lastMeta, nil
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
