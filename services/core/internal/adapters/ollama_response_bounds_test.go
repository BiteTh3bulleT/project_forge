package adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOllamaNonStreamingResponsesRejectOversizeBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[]}`))
		case "/api/generate":
			_, _ = w.Write([]byte(`{"response":"ok","done":true}`))
		case "/api/chat":
			_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ok"},"done":true}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(strings.Repeat(" ", ollamaResponseBodyLimit+1)))
	}))
	defer server.Close()

	ollama := Ollama{client: server.Client()}
	ctx := context.Background()

	if _, err := ollama.fetchTags(ctx, server.URL, time.Second); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected oversized /api/tags response error, got %v", err)
	}
	if _, _, err := ollama.generate(ctx, server.URL, "model", "prompt", time.Second); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected oversized /api/generate response error, got %v", err)
	}
	if _, err := ollama.OllamaChat(ctx, server.URL, "model", []map[string]any{{"role": "user", "content": "hi"}}, nil, nil, time.Second); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected oversized /api/chat response error, got %v", err)
	}
}

func TestOllamaErrorResponsesAreBoundedAndMarked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(strings.Repeat("x", ollamaErrorBodyLimit+1)))
	}))
	defer server.Close()

	ollama := Ollama{client: server.Client()}
	ctx := context.Background()

	_, err := ollama.OllamaChat(ctx, server.URL, "model", []map[string]any{{"role": "user", "content": "hi"}}, nil, nil, time.Second)
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected truncated /api/chat error, got %v", err)
	}

	_, _, err = ollama.streamChatWithModel(ctx, server.URL, "model", []map[string]any{{"role": "user", "content": "hi"}}, time.Second, nil)
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected truncated /api/chat stream error, got %v", err)
	}

	_, _, err = ollama.StreamGenerate(ctx, server.URL, "model", "prompt", time.Second, nil)
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected truncated /api/generate stream error, got %v", err)
	}
}
