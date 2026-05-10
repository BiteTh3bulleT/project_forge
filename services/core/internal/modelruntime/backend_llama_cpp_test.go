package modelruntime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLlamaCppBackend_GenerateChatTranslation(t *testing.T) {
	var seenPath string
	var seenModel string
	var seenMessages int
	var seenMaxTokens int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		seenModel, _ = payload["model"].(string)
		if messages, ok := payload["messages"].([]any); ok {
			seenMessages = len(messages)
		}
		if maxTokens, ok := payload["max_tokens"].(float64); ok {
			seenMaxTokens = int(maxTokens)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"finish_reason": "stop",
				"message":       map[string]any{"role": "assistant", "content": "hello from llama"},
			}},
			"usage": map[string]any{"prompt_tokens": 7, "completion_tokens": 3},
		})
	}))
	defer ts.Close()

	backend := NewLlamaCppBackend(LlamaCppOptions{Endpoint: ts.URL, RequestTimeout: time.Second, MaxOutputTokens: 16})
	if _, err := backend.Load(context.Background(), ModelManifest{ID: "chat-model", Format: ModelFormatGGUF, Backend: BackendLlamaCpp}); err != nil {
		t.Fatalf("load: %v", err)
	}

	res, err := backend.Generate(context.Background(), GenerateRequest{
		ModelID:   "chat-model",
		Messages:  []ChatMessage{{Role: "user", Content: "hi"}},
		MaxTokens: 4,
	})
	if err != nil {
		t.Fatalf("generate chat: %v", err)
	}
	if seenPath != "/v1/chat/completions" {
		t.Fatalf("unexpected path: %s", seenPath)
	}
	if seenModel != "chat-model" {
		t.Fatalf("unexpected model: %s", seenModel)
	}
	if seenMessages != 1 {
		t.Fatalf("expected one message, got %d", seenMessages)
	}
	if seenMaxTokens != 4 {
		t.Fatalf("expected max_tokens=4, got %d", seenMaxTokens)
	}
	if res.Content != "hello from llama" {
		t.Fatalf("unexpected content: %q", res.Content)
	}
	if res.PromptTokens != 7 || res.CompletionTokens != 3 {
		t.Fatalf("unexpected usage: %+v", res)
	}
}

func TestLlamaCppBackend_GenerateCompletionTranslation(t *testing.T) {
	var seenPath string
	var seenPrompt string
	var seenPredict int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		seenPrompt, _ = payload["prompt"].(string)
		if np, ok := payload["n_predict"].(float64); ok {
			seenPredict = int(np)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content":          "alpha beta gamma delta epsilon",
			"tokens_evaluated": 5,
			"tokens_predicted": 5,
		})
	}))
	defer ts.Close()

	backend := NewLlamaCppBackend(LlamaCppOptions{Endpoint: ts.URL, RequestTimeout: time.Second, MaxOutputTokens: 3})
	if _, err := backend.Load(context.Background(), ModelManifest{ID: "completion-model", Format: ModelFormatGGUF, Backend: BackendLlamaCpp}); err != nil {
		t.Fatalf("load: %v", err)
	}
	res, err := backend.Generate(context.Background(), GenerateRequest{ModelID: "completion-model", Prompt: "write words", MaxTokens: 6})
	if err != nil {
		t.Fatalf("generate completion: %v", err)
	}
	if seenPath != "/completion" {
		t.Fatalf("unexpected path: %s", seenPath)
	}
	if seenPrompt != "write words" {
		t.Fatalf("unexpected prompt: %s", seenPrompt)
	}
	if seenPredict != 3 {
		t.Fatalf("expected n_predict to be bounded to 3, got %d", seenPredict)
	}
	if len(strings.Fields(res.Content)) != 3 {
		t.Fatalf("expected bounded output 3 tokens, got %q", res.Content)
	}
	if res.FinishReason != "length" {
		t.Fatalf("expected finish reason length, got %s", res.FinishReason)
	}
}

func TestLlamaCppBackend_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{"content": "slow"})
	}))
	defer ts.Close()

	backend := NewLlamaCppBackend(LlamaCppOptions{Endpoint: ts.URL, RequestTimeout: 30 * time.Millisecond})
	if _, err := backend.Load(context.Background(), ModelManifest{ID: "slow", Format: ModelFormatGGUF, Backend: BackendLlamaCpp}); err != nil {
		t.Fatalf("load: %v", err)
	}
	_, err := backend.Generate(context.Background(), GenerateRequest{ModelID: "slow", Prompt: "hi"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestLlamaCppBackend_UnavailableEndpoint(t *testing.T) {
	backend := NewLlamaCppBackend(LlamaCppOptions{Endpoint: "http://127.0.0.1:1", RequestTimeout: 50 * time.Millisecond})
	if _, err := backend.Load(context.Background(), ModelManifest{ID: "x", Format: ModelFormatGGUF, Backend: BackendLlamaCpp}); err != nil {
		t.Fatalf("load should not require active endpoint, got %v", err)
	}
	_, err := backend.Generate(context.Background(), GenerateRequest{ModelID: "x", Prompt: "hello"})
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("expected backend unavailable error, got %v", err)
	}
}

func TestLlamaCppBackendRejectsOversizeResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", llamaCppResponseBodyLimit+1)))
	}))
	defer ts.Close()

	backend := NewLlamaCppBackend(LlamaCppOptions{Endpoint: ts.URL, RequestTimeout: time.Second})
	if _, err := backend.Load(context.Background(), ModelManifest{ID: "oversize", Format: ModelFormatGGUF, Backend: BackendLlamaCpp}); err != nil {
		t.Fatalf("load: %v", err)
	}

	_, err := backend.Generate(context.Background(), GenerateRequest{ModelID: "oversize", Prompt: "hi"})
	if err == nil || !strings.Contains(err.Error(), "response too large") {
		t.Fatalf("expected response too large error, got %v", err)
	}
}

func TestLlamaCppBackend_SpawnDeferred(t *testing.T) {
	backend := NewLlamaCppBackend(LlamaCppOptions{AllowSpawn: true, Endpoint: "http://127.0.0.1:8080"})
	_, err := backend.Load(context.Background(), ModelManifest{ID: "spawn", Format: ModelFormatGGUF, Backend: BackendLlamaCpp})
	if !errors.Is(err, ErrUnsupportedSpawn) {
		t.Fatalf("expected unsupported spawn error, got %v", err)
	}
}
