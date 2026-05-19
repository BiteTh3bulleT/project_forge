package modelruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAICompatBackendLoadGenerateHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "remote-model"}}})
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{"message": map[string]any{"content": "remote ok"}, "finish_reason": "stop"}},
				"usage":   map[string]any{"prompt_tokens": 3, "completion_tokens": 2},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	backend := NewOpenAICompatBackend(OpenAICompatOptions{
		Endpoint:        server.URL,
		Kind:            BackendOpenAICompat,
		RequestTimeout:  2000000000,
		MaxOutputTokens: 32,
	})

	loaded, err := backend.Load(context.Background(), ModelManifest{
		ID:           "remote-model",
		Backend:      BackendOpenAICompat,
		Format:       ModelFormatUnknown,
		Capabilities: []ModelCapability{CapabilityChat},
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.ModelID != "remote-model" {
		t.Fatalf("unexpected loaded model: %+v", loaded)
	}

	result, err := backend.Generate(context.Background(), GenerateRequest{
		ModelID:  "remote-model",
		Messages: []GenerateMessage{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result.Content != "remote ok" || result.PromptTokens != 3 || result.CompletionTokens != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}

	health, err := backend.Health(context.Background())
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if !health.Healthy || health.Kind != BackendOpenAICompat {
		t.Fatalf("unexpected health: %+v", health)
	}
}

func TestOpenAICompatBackendHealthReportsSupervisionMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "remote-model"}}})
	}))
	defer server.Close()

	backend := NewOpenAICompatBackend(OpenAICompatOptions{
		Name:           "vllm-compatible",
		Endpoint:       server.URL,
		Kind:           BackendVLLM,
		RequestTimeout: 2300 * time.Millisecond,
		ModelsPath:     "/models",
		ChatPath:       "/chat",
		Profile:        "interactive_vllm",
	})

	health, err := backend.Health(context.Background())
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if health.Meta["supervision"] != "external_endpoint" || health.Meta["processManaged"] != false || health.Meta["spawnSupported"] != false {
		t.Fatalf("unexpected supervision metadata: %#v", health.Meta)
	}
	if health.Meta["profile"] != "interactive_vllm" || health.Meta["modelsPath"] != "/models" || health.Meta["chatPath"] != "/chat" {
		t.Fatalf("unexpected profile/path metadata: %#v", health.Meta)
	}
	if health.Meta["requestTimeoutMs"] != int64(2300) {
		t.Fatalf("unexpected timeout metadata: %#v", health.Meta)
	}
}

func TestOpenAICompatBackendGenerateExtractsContentParts(t *testing.T) {
	result := generateWithOpenAICompatResponse(t, map[string]any{
		"choices": []map[string]any{{
			"message": map[string]any{
				"content": []map[string]any{
					{"type": "reasoning", "text": "internal notes"},
					{"type": "text", "text": "part one"},
					{"type": "text", "text": "part two"},
				},
			},
			"finish_reason": "stop",
		}},
	})
	if result.Content != "part one\npart two" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", result.Warnings)
	}
}

func TestOpenAICompatBackendGenerateExtractsChoiceTextFallback(t *testing.T) {
	result := generateWithOpenAICompatResponse(t, map[string]any{
		"choices": []map[string]any{{
			"text":          "completion style ok",
			"finish_reason": "stop",
		}},
	})
	if result.Content != "completion style ok" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

func TestOpenAICompatBackendGenerateExtractsReasoningFallbackWithWarning(t *testing.T) {
	result := generateWithOpenAICompatResponse(t, map[string]any{
		"choices": []map[string]any{{
			"message":       map[string]any{"content": "", "reasoning_content": "reasoning-only provider output"},
			"finish_reason": "stop",
		}},
	})
	if result.Content != "reasoning-only provider output" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "openai-compatible response used reasoning-content fallback" {
		t.Fatalf("unexpected warnings: %v", result.Warnings)
	}
}

func TestOpenAICompatBackendGenerateStreamEmitsContentDeltas(t *testing.T) {
	var sawStream bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			sawStream, _ = payload["stream"].(bool)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"cloud \"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"stream\"},\"finish_reason\":\"stop\"}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	backend := NewOpenAICompatBackend(OpenAICompatOptions{
		Endpoint:       server.URL,
		Kind:           BackendOpenAICompat,
		RequestTimeout: 2000000000,
	})
	_, err := backend.Load(context.Background(), ModelManifest{
		ID:           "remote-model",
		Backend:      BackendOpenAICompat,
		Format:       ModelFormatUnknown,
		Capabilities: []ModelCapability{CapabilityChat},
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	tokens := []string{}
	result, err := backend.GenerateStream(context.Background(), GenerateRequest{
		ModelID:  "remote-model",
		Messages: []GenerateMessage{{Role: "user", Content: "hello"}},
	}, func(event TokenEvent) error {
		if event.Token != "" {
			tokens = append(tokens, event.Token)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("stream generate: %v", err)
	}
	if !sawStream {
		t.Fatalf("expected stream=true request")
	}
	if got := strings.Join(tokens, ""); got != "cloud stream" {
		t.Fatalf("unexpected streamed tokens %q from %#v", got, tokens)
	}
	if result.Content != "cloud stream" || result.FinishReason != "stop" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestOpenAICompatBackendGenerateStreamEmitsReasoningDeltasSeparately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"checking facts\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"answer\"},\"finish_reason\":\"stop\"}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	backend := NewOpenAICompatBackend(OpenAICompatOptions{
		Endpoint:       server.URL,
		Kind:           BackendOpenAICompat,
		RequestTimeout: 2000000000,
	})
	_, err := backend.Load(context.Background(), ModelManifest{
		ID:           "remote-model",
		Backend:      BackendOpenAICompat,
		Format:       ModelFormatUnknown,
		Capabilities: []ModelCapability{CapabilityChat},
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	reasoning := []string{}
	tokens := []string{}
	result, err := backend.GenerateStream(context.Background(), GenerateRequest{
		ModelID:  "remote-model",
		Messages: []GenerateMessage{{Role: "user", Content: "hello"}},
	}, func(event TokenEvent) error {
		if event.Reasoning != "" {
			reasoning = append(reasoning, event.Reasoning)
		}
		if event.Token != "" {
			tokens = append(tokens, event.Token)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("stream generate: %v", err)
	}
	if got := strings.Join(reasoning, ""); got != "checking facts" {
		t.Fatalf("unexpected reasoning stream %q from %#v", got, reasoning)
	}
	if got := strings.Join(tokens, ""); got != "answer" {
		t.Fatalf("unexpected visible token stream %q from %#v", got, tokens)
	}
	if result.Content != "answer" {
		t.Fatalf("reasoning leaked into visible content: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("reasoning stream chunks should not create warning spam: %v", result.Warnings)
	}
}

func TestOpenAICompatBackendGenerateRejectsOversizeResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			_, _ = w.Write([]byte(strings.Repeat("x", openAICompatResponseBodyLimit+1)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	backend := NewOpenAICompatBackend(OpenAICompatOptions{
		Endpoint:       server.URL,
		Kind:           BackendOpenAICompat,
		RequestTimeout: 2000000000,
	})
	_, err := backend.Load(context.Background(), ModelManifest{
		ID:           "remote-model",
		Backend:      BackendOpenAICompat,
		Format:       ModelFormatUnknown,
		Capabilities: []ModelCapability{CapabilityChat},
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	_, err = backend.Generate(context.Background(), GenerateRequest{
		ModelID:  "remote-model",
		Messages: []GenerateMessage{{Role: "user", Content: "hello"}},
	})
	if err == nil || !strings.Contains(err.Error(), "response too large") {
		t.Fatalf("expected response too large error, got %v", err)
	}
}

func TestOpenAICompatBackendStreamRejectsOversizeErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(strings.Repeat("x", openAICompatResponseBodyLimit+1)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	backend := NewOpenAICompatBackend(OpenAICompatOptions{
		Endpoint:       server.URL,
		Kind:           BackendOpenAICompat,
		RequestTimeout: 2000000000,
	})
	_, err := backend.Load(context.Background(), ModelManifest{
		ID:           "remote-model",
		Backend:      BackendOpenAICompat,
		Format:       ModelFormatUnknown,
		Capabilities: []ModelCapability{CapabilityChat},
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	_, err = backend.GenerateStream(context.Background(), GenerateRequest{
		ModelID:  "remote-model",
		Messages: []GenerateMessage{{Role: "user", Content: "hello"}},
	}, func(TokenEvent) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "response too large") {
		t.Fatalf("expected response too large error, got %v", err)
	}
}

func generateWithOpenAICompatResponse(t *testing.T, response map[string]any) GenerateResult {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	backend := NewOpenAICompatBackend(OpenAICompatOptions{
		Endpoint:       server.URL,
		Kind:           BackendOpenAICompat,
		RequestTimeout: 2000000000,
	})
	_, err := backend.Load(context.Background(), ModelManifest{
		ID:           "remote-model",
		Backend:      BackendOpenAICompat,
		Format:       ModelFormatUnknown,
		Capabilities: []ModelCapability{CapabilityChat},
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	result, err := backend.Generate(context.Background(), GenerateRequest{
		ModelID:  "remote-model",
		Messages: []GenerateMessage{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return result
}
