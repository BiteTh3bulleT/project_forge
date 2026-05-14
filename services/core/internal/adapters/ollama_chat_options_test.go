package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaChatSendsDefaultOptions(t *testing.T) {
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_PREDICT", "")
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_CTX", "")
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_THREAD", "")

	reqBody := captureOllamaChatRequest(t, func(ollama Ollama, url string) error {
		_, err := ollama.OllamaChat(context.Background(), url, "model", []map[string]any{{"role": "user", "content": "hi"}}, nil, nil, time.Second)
		return err
	})

	options, ok := reqBody["options"].(map[string]any)
	if !ok {
		t.Fatalf("expected options object in request, got %#v", reqBody["options"])
	}
	if got := options["num_predict"]; got != float64(96) {
		t.Fatalf("expected default num_predict 96, got %#v", got)
	}
	if got := options["num_ctx"]; got != float64(1024) {
		t.Fatalf("expected default num_ctx 1024, got %#v", got)
	}
	if _, ok := options["num_thread"]; ok {
		t.Fatalf("did not expect num_thread without positive env override, got %#v", options["num_thread"])
	}
}

func TestOllamaChatHonorsEnvOptions(t *testing.T) {
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_PREDICT", "128")
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_CTX", "2048")
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_THREAD", "6")

	reqBody := captureOllamaChatRequest(t, func(ollama Ollama, url string) error {
		_, err := ollama.OllamaChat(context.Background(), url, "model", []map[string]any{{"role": "user", "content": "hi"}}, nil, nil, time.Second)
		return err
	})

	options, ok := reqBody["options"].(map[string]any)
	if !ok {
		t.Fatalf("expected options object in request, got %#v", reqBody["options"])
	}
	if got := options["num_predict"]; got != float64(128) {
		t.Fatalf("expected env num_predict 128, got %#v", got)
	}
	if got := options["num_ctx"]; got != float64(2048) {
		t.Fatalf("expected env num_ctx 2048, got %#v", got)
	}
	if got := options["num_thread"]; got != float64(6) {
		t.Fatalf("expected env num_thread 6, got %#v", got)
	}
}

func TestOllamaChatOmitsOptionsWhenDisabled(t *testing.T) {
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_PREDICT", "0")
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_CTX", "0")
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_THREAD", "0")

	reqBody := captureOllamaChatRequest(t, func(ollama Ollama, url string) error {
		_, err := ollama.OllamaChat(context.Background(), url, "model", []map[string]any{{"role": "user", "content": "hi"}}, nil, nil, time.Second)
		return err
	})

	if _, ok := reqBody["options"]; ok {
		t.Fatalf("expected options to be omitted when every option is disabled, got %#v", reqBody["options"])
	}
}

func TestOllamaStreamChatSendsOptions(t *testing.T) {
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_PREDICT", "32")
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_CTX", "512")
	t.Setenv("FORGE_OLLAMA_CHAT_NUM_THREAD", "2")

	reqBody := captureOllamaChatRequest(t, func(ollama Ollama, url string) error {
		_, _, err := ollama.StreamChat(context.Background(), url, "model", []map[string]any{{"role": "user", "content": "hi"}}, time.Second, nil)
		return err
	})

	if got := reqBody["stream"]; got != true {
		t.Fatalf("expected streaming request, got stream=%#v", got)
	}
	options, ok := reqBody["options"].(map[string]any)
	if !ok {
		t.Fatalf("expected options object in stream request, got %#v", reqBody["options"])
	}
	if got := options["num_predict"]; got != float64(32) {
		t.Fatalf("expected env num_predict 32, got %#v", got)
	}
	if got := options["num_ctx"]; got != float64(512) {
		t.Fatalf("expected env num_ctx 512, got %#v", got)
	}
	if got := options["num_thread"]; got != float64(2) {
		t.Fatalf("expected env num_thread 2, got %#v", got)
	}
}

func captureOllamaChatRequest(t *testing.T, call func(Ollama, string) error) map[string]any {
	t.Helper()
	var reqBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if stream, _ := reqBody["stream"].(bool); stream {
			_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ok"},"done":false}` + "\n"))
			_, _ = w.Write([]byte(`{"done":true}` + "\n"))
			return
		}
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ok"},"done":true}`))
	}))
	defer server.Close()

	if err := call(Ollama{client: server.Client()}, server.URL); err != nil {
		t.Fatalf("ollama chat call: %v", err)
	}
	if reqBody == nil {
		t.Fatal("expected captured request body")
	}
	return reqBody
}
