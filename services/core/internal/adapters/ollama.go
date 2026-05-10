package adapters

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ollamaResponseBodyLimit = 4 << 20
	ollamaErrorBodyLimit    = 2048
)

type Ollama struct {
	db     *sql.DB
	client *http.Client
}

func NewOllama(db *sql.DB) Ollama {
	return Ollama{
		db: db,
		// Must be >= chat / other invoke timeouts or long /api/generate calls fail early.
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (o Ollama) Info(ctx context.Context) AdapterInfo {
	baseURL := o.setting(ctx, "ollama_base_url", envOr("OLLAMA_BASE_URL", "http://127.0.0.1:11434"))
	model := normalizeOllamaModel(o.setting(ctx, "ollama_model", envOr("OLLAMA_MODEL", "")))

	tags, err := o.fetchTags(ctx, baseURL, 1300*time.Millisecond)
	if err != nil {
		return AdapterInfo{
			ID:          "ollama",
			DisplayName: "Ollama",
			Status:      StatusMisconfig,
			Detail:      fmt.Sprintf("Ollama unavailable at %s: %v", baseURL, err),
			Capabilities: []string{
				"status",
				"generate_summary",
				"draft_plan",
				"analysis",
			},
			Config: map[string]any{
				"baseUrl": baseURL,
				"model":   model,
			},
		}
	}

	return AdapterInfo{
		ID:          "ollama",
		DisplayName: "Ollama",
		Status:      StatusReady,
		Detail:      fmt.Sprintf("Connected to local Ollama (%d models visible).", len(tags.Models)),
		Capabilities: []string{
			"status",
			"generate_summary",
			"draft_plan",
			"analysis",
		},
		Config: map[string]any{
			"baseUrl": baseURL,
			"model":   model,
			"models":  tags.ModelNames(),
		},
	}
}

func (o Ollama) Invoke(ctx context.Context, req InvokeRequest) (InvokeResult, error) {
	if req.AdapterID != "" && req.AdapterID != "ollama" {
		return InvokeResult{OK: false, FailureCode: "validation", Message: "adapterId mismatch for ollama", Data: map[string]any{}}, nil
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 45_000
	}
	baseURL := o.setting(ctx, "ollama_base_url", envOr("OLLAMA_BASE_URL", "http://127.0.0.1:11434"))

	switch strings.TrimSpace(req.Capability) {
	case "status":
		tags, err := o.fetchTags(ctx, baseURL, time.Duration(req.TimeoutMs)*time.Millisecond)
		if err != nil {
			return InvokeResult{OK: false, FailureCode: classifyNetErr(err), Message: err.Error(), Data: map[string]any{"baseUrl": baseURL}}, nil
		}
		return InvokeResult{OK: true, Message: "Ollama reachable", Data: map[string]any{"baseUrl": baseURL, "models": tags.ModelNames()}}, nil

	case "generate_summary", "draft_plan", "analysis":
		prompt := readString(req.Input, "prompt")
		if strings.TrimSpace(prompt) == "" {
			return InvokeResult{OK: false, FailureCode: "validation", Message: "prompt is required", Data: map[string]any{}}, nil
		}
		if req.DryRun {
			return InvokeResult{OK: true, Message: "Dry run: Ollama request prepared.", Data: map[string]any{
				"baseUrl":     baseURL,
				"capability":  req.Capability,
				"timeoutMs":   req.TimeoutMs,
				"correlation": req.CorrelationID,
			}}, nil
		}

		model := readString(req.Input, "model")
		if strings.TrimSpace(model) == "" {
			model = o.setting(ctx, "ollama_model", envOr("OLLAMA_MODEL", ""))
		}
		model = normalizeOllamaModel(model)
		if strings.TrimSpace(model) == "" {
			return InvokeResult{OK: false, FailureCode: "validation", Message: "ollama model is not configured", Data: map[string]any{"baseUrl": baseURL}}, nil
		}

		response, meta, err := o.generate(ctx, baseURL, model, prompt, time.Duration(req.TimeoutMs)*time.Millisecond)
		if err != nil {
			return InvokeResult{OK: false, FailureCode: classifyNetErr(err), Message: err.Error(), Data: map[string]any{"baseUrl": baseURL, "model": model}}, nil
		}
		return InvokeResult{OK: true, Message: "Ollama response received", Data: map[string]any{
			"model":      model,
			"response":   response,
			"metadata":   meta,
			"baseUrl":    baseURL,
			"capability": req.Capability,
		}}, nil
	default:
		return InvokeResult{OK: false, FailureCode: "validation", Message: fmt.Sprintf("unsupported capability %q for ollama", req.Capability), Data: map[string]any{}}, nil
	}
}

// BaseURLForChat returns the configured Ollama base URL (for streaming and direct calls).
func (o Ollama) BaseURLForChat(ctx context.Context) string {
	return o.setting(ctx, "ollama_base_url", envOr("OLLAMA_BASE_URL", "http://127.0.0.1:11434"))
}

// ModelForChat returns the configured default model name.
func (o Ollama) ModelForChat(ctx context.Context) string {
	return normalizeOllamaModel(o.setting(ctx, "ollama_model", envOr("OLLAMA_MODEL", "")))
}

func (o Ollama) FetchModels(ctx context.Context, baseURL string, timeout time.Duration) ([]string, error) {
	tags, err := o.fetchTags(ctx, baseURL, timeout)
	if err != nil {
		return nil, err
	}
	return tags.ModelNames(), nil
}

func normalizeOllamaModel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if s == "qwen-coder:30b" {
		return "qwen3-coder:30b"
	}
	return s
}

func (o Ollama) setting(ctx context.Context, key, def string) string {
	if o.db == nil {
		return def
	}
	var v string
	err := o.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil || strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

type ollamaTags struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func (t ollamaTags) ModelNames() []string {
	out := make([]string, 0, len(t.Models))
	for _, m := range t.Models {
		if strings.TrimSpace(m.Name) != "" {
			out = append(out, m.Name)
		}
	}
	return out
}

func (o Ollama) fetchTags(ctx context.Context, baseURL string, timeout time.Duration) (ollamaTags, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/tags", nil)
	if err != nil {
		return ollamaTags{}, err
	}
	res, err := o.client.Do(req)
	if err != nil {
		return ollamaTags{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return ollamaTags{}, fmt.Errorf("ollama /api/tags returned %s", res.Status)
	}
	var out ollamaTags
	if err := decodeOllamaJSONBody(res.Body, &out); err != nil {
		return ollamaTags{}, fmt.Errorf("decode /api/tags: %w", err)
	}
	return out, nil
}

func (o Ollama) generate(ctx context.Context, baseURL, model, prompt string, timeout time.Duration) (string, map[string]any, error) {
	payload := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}
	b, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/generate", bytes.NewReader(b))
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
		return "", nil, fmt.Errorf("ollama /api/generate returned %s", res.Status)
	}

	var out map[string]any
	if err := decodeOllamaJSONBody(res.Body, &out); err != nil {
		return "", nil, fmt.Errorf("decode /api/generate: %w", err)
	}
	response := readString(out, "response")
	meta := map[string]any{
		"done":               out["done"],
		"totalDuration":      out["total_duration"],
		"loadDuration":       out["load_duration"],
		"promptEvalCount":    out["prompt_eval_count"],
		"evalCount":          out["eval_count"],
		"promptEvalDuration": out["prompt_eval_duration"],
		"evalDuration":       out["eval_duration"],
	}
	return response, meta, nil
}

// StreamGenerate calls Ollama /api/generate with stream: true. Invokes onToken for each response delta; accumulates full text for the final reply.
func (o Ollama) StreamGenerate(ctx context.Context, baseURL, model, prompt string, timeout time.Duration, onToken func(string) error) (fullText string, lastMeta map[string]any, err error) {
	payload := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": true,
	}
	b, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/generate", bytes.NewReader(b))
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
			return "", nil, fmt.Errorf("read /api/generate stream error response: %w", err)
		}
		return "", nil, fmt.Errorf("ollama /api/generate stream returned %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	var acc strings.Builder
	sc := bufio.NewScanner(res.Body)
	// Ollama streams large lines; allow big tokens
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if e := json.Unmarshal([]byte(line), &obj); e != nil {
			continue
		}
		if tok := readString(obj, "response"); tok != "" {
			acc.WriteString(tok)
			if onToken != nil {
				if e := onToken(tok); e != nil {
					return acc.String(), lastMeta, e
				}
			}
		}
		if done, ok := obj["done"].(bool); ok && done {
			lastMeta = map[string]any{
				"totalDuration":      obj["total_duration"],
				"loadDuration":       obj["load_duration"],
				"promptEvalCount":    obj["prompt_eval_count"],
				"evalCount":          obj["eval_count"],
				"promptEvalDuration": obj["prompt_eval_duration"],
				"evalDuration":       obj["eval_duration"],
			}
			break
		}
	}
	if err := sc.Err(); err != nil {
		return acc.String(), lastMeta, err
	}
	return acc.String(), lastMeta, nil
}

func decodeOllamaJSONBody(body io.Reader, out any) error {
	raw, err := io.ReadAll(io.LimitReader(body, ollamaResponseBodyLimit+1))
	if err != nil {
		return err
	}
	if len(raw) > ollamaResponseBodyLimit {
		return fmt.Errorf("ollama response too large: limit %d bytes", ollamaResponseBodyLimit)
	}
	return json.Unmarshal(raw, out)
}

func readOllamaErrorBody(body io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(body, ollamaErrorBodyLimit+1))
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(raw))
	if len(raw) > ollamaErrorBodyLimit {
		text = strings.TrimSpace(string(raw[:ollamaErrorBodyLimit]))
		if text != "" {
			text += " "
		}
		text += "[truncated]"
	}
	return text, nil
}

func classifyNetErr(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return "adapter_timeout"
	}
	return "adapter_unavailable"
}

func readString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
