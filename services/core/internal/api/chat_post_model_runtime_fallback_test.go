package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type chatPostAssistantMessage struct {
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata"`
}

type chatPostResponse struct {
	AssistantMessage *chatPostAssistantMessage `json:"assistantMessage"`
	AssistantPending bool                      `json:"assistantPending"`
	Stream           bool                      `json:"stream"`
	AsyncAssistant   bool                      `json:"asyncAssistant"`
}

func TestChatPostSyncFallsBackToModelRuntimeWhenOllamaModelMissing(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime fallback sync", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"hello from chat fallback","requestAssistant":true,"syncAssistant":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	var payload chatPostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if payload.AssistantMessage == nil {
		t.Fatalf("expected assistant message in response")
	}
	if ok, _ := payload.AssistantMessage.Metadata["modelRuntimeOk"].(bool); !ok {
		t.Fatalf("expected modelRuntimeOk metadata, got %#v", payload.AssistantMessage.Metadata["modelRuntimeOk"])
	}
	if fakeRuntime.chatCalls == 0 {
		t.Fatalf("expected model runtime chat call")
	}
}

func TestChatPostSyncPrefersModelRuntimeBeforeOllamaFallbackWhenConfigured(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime
	t.Setenv("OLLAMA_MODEL", "qwen2.5-coder")
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:1")

	thread, err := srv.chat.CreateThread(context.Background(), "runtime preferred chat", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"runtime-first selection","requestAssistant":true,"syncAssistant":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	var payload chatPostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if payload.AssistantMessage == nil {
		t.Fatalf("expected assistant message in response")
	}
	if ok, _ := payload.AssistantMessage.Metadata["modelRuntimeOk"].(bool); !ok {
		t.Fatalf("expected modelRuntimeOk metadata, got %#v", payload.AssistantMessage.Metadata["modelRuntimeOk"])
	}

	activity, ok := payload.AssistantMessage.Metadata["toolGatewayActivity"].(map[string]any)
	if !ok {
		t.Fatalf("expected toolGatewayActivity map")
	}
	pipeline, ok := activity["toolPipeline"].(map[string]any)
	if !ok {
		t.Fatalf("expected toolPipeline in toolGatewayActivity")
	}
	stages, ok := pipeline["stages"].([]any)
	if !ok {
		t.Fatalf("expected stages list in toolPipeline")
	}
	foundRuntimePrimary := false
	for _, item := range stages {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(asString(entry["stage"])) == "runtime_primary" {
			foundRuntimePrimary = true
			break
		}
	}
	if !foundRuntimePrimary {
		t.Fatalf("expected runtime_primary stage, got stages=%#v", stages)
	}
	if fakeRuntime.chatCalls == 0 {
		t.Fatalf("expected model runtime chat call")
	}
}

func TestChatPostStreamRequestDowngradesWhenOllamaStreamUnavailable(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime fallback stream downgrade", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"stream fallback check","requestAssistant":true,"stream":true,"syncAssistant":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	var payload chatPostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if payload.Stream {
		t.Fatalf("expected stream=false when ollama stream capability is unavailable")
	}
	if payload.AssistantPending {
		t.Fatalf("expected assistantPending=false for sync fallback")
	}
	if payload.AssistantMessage == nil {
		t.Fatalf("expected assistant message in downgraded response")
	}
	if ok, _ := payload.AssistantMessage.Metadata["modelRuntimeOk"].(bool); !ok {
		t.Fatalf("expected modelRuntimeOk metadata, got %#v", payload.AssistantMessage.Metadata["modelRuntimeOk"])
	}
}

func TestChatAssistantStreamFallsBackToModelRuntimeWhenOllamaStreamUnavailable(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime fallback sse", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "hello via sse fallback", map[string]any{"source": "operator"})
	if err != nil {
		t.Fatalf("append user message: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/assistant-stream?userMessageId="+strconv.FormatInt(um.ID, 10),
		nil,
	)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	body := rr.Body.String()
	if !strings.Contains(body, "event: done") {
		t.Fatalf("expected done SSE event, got body=%s", body)
	}
	if !strings.Contains(body, `"modelRuntimeOk":true`) {
		t.Fatalf("expected model runtime metadata in SSE payload, got body=%s", body)
	}
	if fakeRuntime.chatCalls == 0 {
		t.Fatalf("expected model runtime chat call")
	}
}
