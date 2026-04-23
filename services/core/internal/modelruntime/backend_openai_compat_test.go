package modelruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
