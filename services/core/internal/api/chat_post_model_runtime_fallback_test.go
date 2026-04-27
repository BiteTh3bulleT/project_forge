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

func TestChatPostSyncUsesRequestedModelRuntimeModel(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.models["qwen-override"] = ModelRuntimeModel{
		ID:           "qwen-override",
		DisplayName:  "Qwen Override",
		Backend:      "fake",
		Format:       "gguf",
		Status:       "available",
		Capabilities: []string{"chat", "completion"},
	}
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime requested model", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"use requested runtime model","requestAssistant":true,"syncAssistant":true,"modelId":"qwen-override"}`)
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
	if got := strings.TrimSpace(asString(payload.AssistantMessage.Metadata["modelRuntimeRequestedModelId"])); got != "qwen-override" {
		t.Fatalf("expected modelRuntimeRequestedModelId=qwen-override, got=%q", got)
	}
	if got := strings.TrimSpace(fakeRuntime.lastChat.ModelID); got != "qwen-override" {
		t.Fatalf("expected runtime chat to use requested model qwen-override, got=%q", got)
	}

	detail, err := srv.chat.GetThread(context.Background(), thread.ID)
	if err != nil {
		t.Fatalf("load thread: %v", err)
	}
	var userMeta map[string]any
	for i := len(detail.Messages) - 1; i >= 0; i-- {
		if detail.Messages[i].Role == "user" {
			userMeta = detail.Messages[i].Metadata
			break
		}
	}
	if userMeta == nil {
		t.Fatalf("expected user message in thread")
	}
	if got := strings.TrimSpace(readRequestedModelID(userMeta)); got != "qwen-override" {
		t.Fatalf("expected user metadata requestedModelId=qwen-override, got=%q", got)
	}
}

func TestChatPostSyncBoundsPlainModelRuntimePrompt(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime
	t.Setenv("OLLAMA_MODEL", "qwen2.5-coder")
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:1")

	thread, err := srv.chat.CreateThread(context.Background(), "runtime bounded prompt", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	large := strings.Repeat("x", modelRuntimePlainChatMessageMax*3)
	for i := 0; i < modelRuntimePlainChatMessages+6; i++ {
		if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", large, nil); err != nil {
			t.Fatalf("append user message %d: %v", i, err)
		}
		if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "assistant", large, nil); err != nil {
			t.Fatalf("append assistant message %d: %v", i, err)
		}
	}

	raw := []byte(`{"content":"bounded prompt check","requestAssistant":true,"syncAssistant":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	if fakeRuntime.chatCalls == 0 {
		t.Fatalf("expected model runtime chat call")
	}
	if got := len(fakeRuntime.lastChat.Messages); got > modelRuntimePlainChatMessages+1 {
		t.Fatalf("expected bounded system+recent messages, got=%d", got)
	}
	joinedPrompt := ""
	for _, msg := range fakeRuntime.lastChat.Messages {
		joinedPrompt += msg.Role + ":" + msg.Content + "\n"
		if msg.Role == "user" && len(msg.Content) > modelRuntimePlainChatUserMax {
			t.Fatalf("expected user message <= %d chars, got=%d", modelRuntimePlainChatUserMax, len(msg.Content))
		}
	}
	if !strings.Contains(joinedPrompt, "Recent chat context was compacted") {
		t.Fatalf("expected compaction notice in model runtime prompt")
	}
	if strings.Contains(joinedPrompt, "USER:") || strings.Contains(joinedPrompt, "ASSISTANT:") {
		t.Fatalf("expected structured chat messages without transcript labels, got=%q", joinedPrompt)
	}
	if strings.Count(joinedPrompt, strings.Repeat("x", modelRuntimePlainChatMessageMax+1)) > 0 {
		t.Fatalf("expected oversized message content to be truncated")
	}
	if fakeRuntime.lastChat.MaxTokens != modelRuntimePlainChatMaxOutputToken {
		t.Fatalf("expected max tokens=%d, got=%d", modelRuntimePlainChatMaxOutputToken, fakeRuntime.lastChat.MaxTokens)
	}
	if fakeRuntime.lastChat.TimeoutMs != modelRuntimePlainChatTimeoutMs {
		t.Fatalf("expected timeoutMs=%d, got=%d", modelRuntimePlainChatTimeoutMs, fakeRuntime.lastChat.TimeoutMs)
	}
	if fakeRuntime.lastChat.MaxAttempts != modelRuntimePlainChatMaxAttempts {
		t.Fatalf("expected maxAttempts=%d, got=%d", modelRuntimePlainChatMaxAttempts, fakeRuntime.lastChat.MaxAttempts)
	}
}

func TestChatPostStreamRequestUsesSSEWhenOnlyModelRuntimeCanAnswer(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime stream pending", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"stream via model runtime","requestAssistant":true,"stream":true}`)
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
	if !payload.AssistantPending {
		t.Fatalf("expected assistant pending")
	}
	if !payload.Stream {
		t.Fatalf("expected stream=true so client opens SSE downgrade path")
	}
	if payload.AssistantMessage != nil {
		t.Fatalf("expected no assistant message in initial stream post response")
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected model runtime not called before SSE connection, got %d calls", fakeRuntime.chatCalls)
	}
}

func TestChatPostSyncStripsModelRuntimeReasoningScaffold(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "Thinking Process:\n1. Analyze the request.\n2. Draft options.\n\nFinal Answer: Certainly. What's on the agenda?"
	srv.modelRuntime = fakeRuntime
	t.Setenv("OLLAMA_MODEL", "qwen2.5-coder")
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:1")

	thread, err := srv.chat.CreateThread(context.Background(), "runtime scaffold strip", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"Lets conversate some more.","requestAssistant":true,"syncAssistant":true}`)
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
	if got := strings.TrimSpace(payload.AssistantMessage.Content); got != "Certainly. What's on the agenda?" {
		t.Fatalf("expected final answer only, got=%q", got)
	}
	if strings.Contains(payload.AssistantMessage.Content, "Thinking Process") || strings.Contains(payload.AssistantMessage.Content, "Final Answer") {
		t.Fatalf("assistant content leaked scaffold: %q", payload.AssistantMessage.Content)
	}
	warnings, ok := payload.AssistantMessage.Metadata["assistantContentWarnings"].([]any)
	if !ok {
		t.Fatalf("expected assistantContentWarnings metadata, got %#v", payload.AssistantMessage.Metadata["assistantContentWarnings"])
	}
	if !containsAnyString(warnings, "stripped_reasoning_scaffold") {
		t.Fatalf("expected stripped_reasoning_scaffold warning, got %#v", warnings)
	}
}

func TestChatPostSyncFallsBackWhenModelRuntimeOnlyReturnsReasoningScaffold(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "Thinking Process:\n1. Analyze the request.\n2. Draft options."
	srv.modelRuntime = fakeRuntime
	t.Setenv("OLLAMA_MODEL", "qwen2.5-coder")
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:1")

	thread, err := srv.chat.CreateThread(context.Background(), "runtime scaffold fallback", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"Lets conversate some more.","requestAssistant":true,"syncAssistant":true}`)
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
	if got := strings.TrimSpace(payload.AssistantMessage.Content); got != assistantContentFallback {
		t.Fatalf("expected fallback assistant content, got=%q", got)
	}
	if strings.Contains(payload.AssistantMessage.Content, "Thinking Process") {
		t.Fatalf("assistant content leaked scaffold: %q", payload.AssistantMessage.Content)
	}
}

func TestSanitizeAssistantVisibleContentStripsThinkingBlocksAndTraceability(t *testing.T) {
	content, warnings := sanitizeAssistantVisibleContent("<think>hidden plan</think>\nVisible answer.\n\nTRACEABILITY\nCorrelation chat-tools-339")
	if content != "Visible answer." {
		t.Fatalf("expected visible answer only, got=%q", content)
	}
	if !containsString(warnings, "stripped_hidden_thinking_block") {
		t.Fatalf("expected hidden thinking warning, got=%#v", warnings)
	}
	if !containsString(warnings, "stripped_traceability_scaffold") {
		t.Fatalf("expected traceability warning, got=%#v", warnings)
	}
}

func TestSanitizeAssistantVisibleContentStripsSyntheticUserTurnAndNormalizesIdentity(t *testing.T) {
	content, warnings := sanitizeAssistantVisibleContent("I am Phi, the AI conversational partner.\nUSER: Can we implement a feature?")
	if content != "I am FORGE." {
		t.Fatalf("expected FORGE identity without synthetic user turn, got=%q", content)
	}
	if !containsString(warnings, "normalized_model_identity") {
		t.Fatalf("expected identity normalization warning, got=%#v", warnings)
	}
	if !containsString(warnings, "stripped_synthetic_transcript_turn") {
		t.Fatalf("expected synthetic transcript warning, got=%#v", warnings)
	}
}

func TestStripSyntheticTranscriptContinuation(t *testing.T) {
	content, cut := stripSyntheticTranscriptContinuation("The answer is 42.\nAssistant: Let me ask myself another question.")
	if !cut {
		t.Fatalf("expected synthetic assistant turn to be cut")
	}
	if content != "The answer is 42." {
		t.Fatalf("expected only first answer, got=%q", content)
	}
}

func TestChatPostSyncAnswersIdentityWithoutModel(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "I am Phi."
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "identity deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"What is your name?","requestAssistant":true,"syncAssistant":true}`)
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
		t.Fatalf("expected assistant message")
	}
	if got := strings.TrimSpace(payload.AssistantMessage.Content); got != "I am FORGE." {
		t.Fatalf("expected deterministic FORGE identity, got=%q", got)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
}

func TestChatPostSyncAsksWeatherLocationWithoutModel(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "weather deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"What is the weather looking like today?","requestAssistant":true,"syncAssistant":true}`)
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
		t.Fatalf("expected assistant message")
	}
	if got := strings.TrimSpace(payload.AssistantMessage.Content); got != "What city or ZIP code should I check for the weather?" {
		t.Fatalf("expected deterministic weather clarifier, got=%q", got)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
}

func TestChatPostSyncRoutesStatusWithoutModel(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "status deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"What mode are we in?","requestAssistant":true,"syncAssistant":true}`)
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
		t.Fatalf("expected assistant message")
	}
	if !strings.Contains(payload.AssistantMessage.Content, "Fast path: no model call") {
		t.Fatalf("expected no-model status response, got=%q", payload.AssistantMessage.Content)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
	trace := metadataMap(payload.AssistantMessage.Metadata, "chatLatencyTrace")
	if trace["context_budget_class"] != "tiny" || trace["output_mode"] != "brief" {
		t.Fatalf("expected tiny/brief trace, got %#v", trace)
	}
	if intFromTrace(trace["model_calls_avoided"]) != 1 {
		t.Fatalf("expected model_calls_avoided=1, got %#v", trace["model_calls_avoided"])
	}
}

func TestChatPostSyncRoutesDiagnosticsWithoutModel(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "diagnostics deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"Show diagnostics","requestAssistant":true,"syncAssistant":true}`)
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
	if payload.AssistantMessage == nil || !strings.Contains(payload.AssistantMessage.Content, "Diagnostics fast path") {
		t.Fatalf("expected diagnostics fast path response, got %#v", payload.AssistantMessage)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
}

func TestChatPostSyncRoutesRestoreInspectorWithoutModel(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "restore deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"show latest restore decision","requestAssistant":true,"syncAssistant":true}`)
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
	if payload.AssistantMessage == nil || !strings.Contains(payload.AssistantMessage.Content, "No restore package") {
		t.Fatalf("expected restore inspector fast path response, got %#v", payload.AssistantMessage)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
}

func TestChatPostSyncAmbiguousRequestFallsThroughToModelRuntime(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "ambiguous runtime", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"Tell me something useful about this project","requestAssistant":true,"syncAssistant":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if fakeRuntime.chatCalls != 1 {
		t.Fatalf("expected one model runtime call, got %d", fakeRuntime.chatCalls)
	}
	if fakeRuntime.lastChat.Metadata["budgetClass"] != "small" || fakeRuntime.lastChat.Metadata["outputMode"] != "normal" {
		t.Fatalf("expected small/normal runtime metadata, got %#v", fakeRuntime.lastChat.Metadata)
	}
}

func TestChatPostSyncReportRequestUsesReportOutputBudget(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "report budget", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"write a full report about the current architecture","requestAssistant":true,"syncAssistant":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if fakeRuntime.chatCalls != 1 {
		t.Fatalf("expected one model runtime call, got %d", fakeRuntime.chatCalls)
	}
	if fakeRuntime.lastChat.MaxTokens != 1024 {
		t.Fatalf("expected report max tokens, got %d", fakeRuntime.lastChat.MaxTokens)
	}
	if fakeRuntime.lastChat.Metadata["budgetClass"] != "report" || fakeRuntime.lastChat.Metadata["outputMode"] != "report" {
		t.Fatalf("expected report metadata, got %#v", fakeRuntime.lastChat.Metadata)
	}
}

func TestChatPostSyncProviderCooldownBlocksModelCall(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.queueStatus = ModelRuntimeQueueStatus{Depth: 0, PolicyState: "provider_cooldown"}
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "cooldown preflight", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"Tell me something useful about this project","requestAssistant":true,"syncAssistant":true}`)
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
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected cooldown preflight to prevent chat call, got %d", fakeRuntime.chatCalls)
	}
	if payload.AssistantMessage == nil || !strings.Contains(strings.ToLower(payload.AssistantMessage.Content), "cooldown") {
		t.Fatalf("expected explicit cooldown failure, got %#v", payload.AssistantMessage)
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
	if !strings.Contains(body, "event: agent_stage") || !strings.Contains(body, `"stage":"stream_downgrade"`) {
		t.Fatalf("expected visible thinking stage events in SSE payload, got body=%s", body)
	}
	if !strings.Contains(body, `"modelRuntimeOk":true`) {
		t.Fatalf("expected model runtime metadata in SSE payload, got body=%s", body)
	}
	if fakeRuntime.chatCalls == 0 {
		t.Fatalf("expected model runtime chat call")
	}
}

func TestChatAssistantStreamUsesNativeOllamaStreamForNoToolChat(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	var sawStream bool
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected ollama path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode ollama body: %v", err)
		}
		if stream, _ := body["stream"].(bool); !stream {
			t.Fatalf("expected native ollama stream=true, got %#v", body["stream"])
		}
		sawStream = true
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"fast "},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"path"},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"done":true,"total_duration":12,"eval_count":2}` + "\n"))
	}))
	defer ollama.Close()

	ctx := context.Background()
	if err := upsertSetting(ctx, st.DB, "ollama_base_url", ollama.URL); err != nil {
		t.Fatalf("set ollama base url: %v", err)
	}
	if err := upsertSetting(ctx, st.DB, "ollama_model", "cloud-fast"); err != nil {
		t.Fatalf("set ollama model: %v", err)
	}

	thread, err := srv.chat.CreateThread(ctx, "native stream", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(ctx, thread.ID, "user", "hello there", map[string]any{"source": "operator"})
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
	if !sawStream {
		t.Fatalf("expected ollama stream request")
	}
	if !strings.Contains(body, "event: token") || !strings.Contains(body, `"text":"fast "`) || !strings.Contains(body, `"text":"path"`) {
		t.Fatalf("expected streamed token events, got body=%s", body)
	}
	if !strings.Contains(body, `"ollamaStream":true`) {
		t.Fatalf("expected persisted assistant metadata to mark ollamaStream, got body=%s", body)
	}
	if strings.Contains(body, `"modelRuntimeOk":true`) {
		t.Fatalf("expected native ollama stream to avoid modelruntime, got body=%s", body)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected model runtime not called, got %d calls", fakeRuntime.chatCalls)
	}
}

func TestChatAssistantStreamCutsSyntheticTranscriptContinuation(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	var sawStream bool
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected ollama path %s", r.URL.Path)
		}
		sawStream = true
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"Actual answer."},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"\nUSER: fake follow-up\nASSISTANT: fake continuation"},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"done":true,"total_duration":12,"eval_count":2}` + "\n"))
	}))
	defer ollama.Close()

	ctx := context.Background()
	if err := upsertSetting(ctx, st.DB, "ollama_base_url", ollama.URL); err != nil {
		t.Fatalf("set ollama base url: %v", err)
	}
	if err := upsertSetting(ctx, st.DB, "ollama_model", "cloud-fast"); err != nil {
		t.Fatalf("set ollama model: %v", err)
	}

	thread, err := srv.chat.CreateThread(ctx, "native stream cut", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(ctx, thread.ID, "user", "one question", map[string]any{"source": "operator"})
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
	if !sawStream {
		t.Fatalf("expected ollama stream request")
	}
	if strings.Contains(body, "fake follow-up") || strings.Contains(body, "fake continuation") || strings.Contains(body, "USER:") || strings.Contains(body, "ASSISTANT:") {
		t.Fatalf("expected synthetic transcript continuation to be cut, got body=%s", body)
	}
	if !strings.Contains(body, "Actual answer.") {
		t.Fatalf("expected actual answer to remain, got body=%s", body)
	}
	if !strings.Contains(body, "stripped_synthetic_transcript_turn") {
		t.Fatalf("expected sanitizer warning in metadata, got body=%s", body)
	}
}

func TestChatAssistantStreamRespectsRequestedModelFromUserMetadata(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.models["stream-override"] = ModelRuntimeModel{
		ID:           "stream-override",
		DisplayName:  "Stream Override",
		Backend:      "fake",
		Format:       "gguf",
		Status:       "available",
		Capabilities: []string{"chat", "completion"},
	}
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime stream model override", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "hello via stream requested model", map[string]any{
		"source":           "operator",
		"requestedModelId": "stream-override",
	})
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
	if got := strings.TrimSpace(fakeRuntime.lastChat.ModelID); got != "stream-override" {
		t.Fatalf("expected runtime chat to use requested model stream-override, got=%q", got)
	}
}

func containsAnyString(values []any, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(asString(value)) == want {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func metadataMap(metadata map[string]any, key string) map[string]any {
	raw, _ := metadata[key].(map[string]any)
	return raw
}

func intFromTrace(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}
