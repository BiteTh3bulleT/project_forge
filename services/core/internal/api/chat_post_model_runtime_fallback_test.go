package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/store"
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

func setFakeHomeEnv(t *testing.T, homeDir string) {
	t.Helper()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	if volume := filepath.VolumeName(homeDir); volume != "" {
		t.Setenv("HOMEDRIVE", volume)
		t.Setenv("HOMEPATH", strings.TrimPrefix(homeDir, volume))
	}
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

func TestChatPostModelRuntimeConsensusGateWithholdsUnsupportedActionClaim(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "I deleted the workspace file."
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime consensus gate", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"did you delete the file?","requestAssistant":true,"syncAssistant":true}`)
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
	if strings.Contains(payload.AssistantMessage.Content, "I deleted the workspace file") {
		t.Fatalf("unsupported action claim was not withheld: %q", payload.AssistantMessage.Content)
	}
	gate, ok := payload.AssistantMessage.Metadata["consensusGate"].(map[string]any)
	if !ok {
		t.Fatalf("expected consensusGate metadata, got %#v", payload.AssistantMessage.Metadata["consensusGate"])
	}
	if got := strings.TrimSpace(asString(gate["status"])); got != "withheld" {
		t.Fatalf("consensus gate status=%q, gate=%#v", got, gate)
	}
	for _, key := range []string{"canonicalTruth", "memoryMutation", "evidenceAdmission", "gatewayExecution", "liveAuthorityMigration"} {
		if value, _ := gate[key].(bool); value {
			t.Fatalf("consensus gate claimed forbidden authority %s=true: %#v", key, gate)
		}
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

func TestChatPostSyncUsesSelectedLocalOllamaModelWhenSettingsModelBlank(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "")
	srv, st := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatErr = errors.New("runtime cold start timeout")
	fakeRuntime.models["gemma4:e4b"] = ModelRuntimeModel{
		ID:           "gemma4:e4b",
		DisplayName:  "Gemma 4",
		Backend:      "ollama_compat",
		Format:       "gguf",
		Status:       "available",
		Capabilities: []string{"chat", "completion"},
		Metadata: map[string]any{
			"provider": "ollama",
			"remote":   false,
		},
	}
	srv.modelRuntime = fakeRuntime

	var sawModel string
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected ollama path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode ollama body: %v", err)
		}
		sawModel = strings.TrimSpace(asString(body["model"]))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"native selected"}}`))
	}))
	defer ollama.Close()

	ctx := context.Background()
	if err := upsertSetting(ctx, st.DB, "ollama_base_url", ollama.URL); err != nil {
		t.Fatalf("set ollama base url: %v", err)
	}
	if err := upsertSetting(ctx, st.DB, "ollama_model", ""); err != nil {
		t.Fatalf("clear ollama model: %v", err)
	}

	thread, err := srv.chat.CreateThread(ctx, "selected native fallback", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"use selected ollama model","requestAssistant":true,"syncAssistant":true,"modelId":"gemma4:e4b"}`)
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
	if strings.TrimSpace(payload.AssistantMessage.Content) != "native selected" {
		t.Fatalf("expected native ollama response, got %q", payload.AssistantMessage.Content)
	}
	if sawModel != "gemma4:e4b" {
		t.Fatalf("expected native ollama call to use selected model gemma4:e4b, got %q", sawModel)
	}
	if ok, _ := payload.AssistantMessage.Metadata["ollamaOk"].(bool); !ok {
		t.Fatalf("expected ollamaOk metadata, got %#v", payload.AssistantMessage.Metadata["ollamaOk"])
	}
	if got := strings.TrimSpace(asString(payload.AssistantMessage.Metadata["ollamaModelSource"])); got != "selected_model" {
		t.Fatalf("expected ollamaModelSource=selected_model, got %q", got)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected selected local ollama model to bypass model runtime chat, got %d calls", fakeRuntime.chatCalls)
	}
}

func TestResolveNativeOllamaChatModelPrefersValidRequestOverPersistedDefault(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.models["smuxo/smuxoAI:0.8b"] = ModelRuntimeModel{
		ID:           "smuxo/smuxoAI:0.8b",
		DisplayName:  "smuxoAI",
		Backend:      "ollama_compat",
		Format:       "gguf",
		Status:       "available",
		Capabilities: []string{"chat", "completion", "tools"},
		Metadata: map[string]any{
			"provider": "ollama",
			"remote":   false,
		},
	}
	srv.modelRuntime = fakeRuntime

	ctx := context.Background()
	if err := upsertSetting(ctx, st.DB, "ollama_model", "gemma3:1b-it-q4_K_M"); err != nil {
		t.Fatalf("set stale persisted ollama model: %v", err)
	}

	model, source := srv.resolveNativeOllamaChatModel(ctx, adapters.NewOllama(st.DB), "smuxo/smuxoAI:0.8b")
	if model != "smuxo/smuxoAI:0.8b" || source != "selected_model" {
		t.Fatalf("resolved model=%q source=%q, want selected smuxo model", model, source)
	}
}

func TestChatToolPathUsesValidRequestedModelOverPersistedDefault(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.models["smuxo/smuxoAI:0.8b"] = ModelRuntimeModel{
		ID:           "smuxo/smuxoAI:0.8b",
		DisplayName:  "smuxoAI",
		Backend:      "ollama_compat",
		Format:       "gguf",
		Status:       "available",
		Capabilities: []string{"chat", "completion", "tools"},
		Metadata: map[string]any{
			"provider": "ollama",
			"remote":   false,
		},
	}
	srv.modelRuntime = fakeRuntime

	var sawModel string
	var sawToolNames []string
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected ollama path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode ollama body: %v", err)
		}
		sawModel = strings.TrimSpace(asString(body["model"]))
		if tools, ok := body["tools"].([]any); ok {
			for _, rawTool := range tools {
				tool, _ := rawTool.(map[string]any)
				fn, _ := tool["function"].(map[string]any)
				sawToolNames = append(sawToolNames, strings.TrimSpace(asString(fn["name"])))
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":""},"done":true}`))
	}))
	defer ollama.Close()

	ctx := context.Background()
	if err := upsertSetting(ctx, st.DB, "ollama_base_url", ollama.URL); err != nil {
		t.Fatalf("set ollama base url: %v", err)
	}
	if err := upsertSetting(ctx, st.DB, "ollama_model", "gemma3:1b-it-q4_K_M"); err != nil {
		t.Fatalf("set stale persisted ollama model: %v", err)
	}

	thread, err := srv.chat.CreateThread(ctx, "selected tool model", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"Show the modified files in the current workspace.","requestAssistant":true,"syncAssistant":true,"modelId":"smuxo/smuxoAI:0.8b"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if sawModel != "smuxo/smuxoAI:0.8b" {
		t.Fatalf("tool path used model=%q, want selected smuxo model", sawModel)
	}
	if strings.Join(sawToolNames, ",") != "forge_git_status" {
		t.Fatalf("tool path attached tools=%v, want only forge_git_status", sawToolNames)
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

func TestModelRuntimePromptIncludesEarlierSameThreadMemory(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)

	thread, err := srv.chat.CreateThread(context.Background(), "same thread recall", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "remember the deployment code phrase is blue lantern", nil); err != nil {
		t.Fatalf("append seed message: %v", err)
	}
	for i := 0; i < modelRuntimePlainChatMessages+2; i++ {
		if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "filler message "+strconv.Itoa(i), nil); err != nil {
			t.Fatalf("append filler message %d: %v", i, err)
		}
	}

	detail, err := srv.chat.GetThread(context.Background(), thread.ID)
	if err != nil {
		t.Fatalf("get thread: %v", err)
	}
	messages, budget := srv.buildModelRuntimePlainChatMessages(context.Background(), detail)
	if got := len(messages); got > modelRuntimePlainChatMessages+1 {
		t.Fatalf("expected bounded system+recent messages, got=%d", got)
	}
	if budget.MemoryChars == 0 {
		t.Fatalf("expected earlier thread memory budget to be recorded")
	}
	if !strings.Contains(messages[0].Content, "blue lantern") {
		t.Fatalf("expected earlier same-thread memory in system prompt, got %q", messages[0].Content)
	}
}

func TestChatLLMMessagesIncludeEarlierAndRelatedChatMemory(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)

	other, err := srv.chat.CreateThread(context.Background(), "other thread", nil)
	if err != nil {
		t.Fatalf("create other thread: %v", err)
	}
	if _, err := srv.chat.AppendMessage(context.Background(), other.ID, "user", "remember the cross-thread deployment token is amber bridge", nil); err != nil {
		t.Fatalf("append other thread message: %v", err)
	}

	thread, err := srv.chat.CreateThread(context.Background(), "long current thread", nil)
	if err != nil {
		t.Fatalf("create current thread: %v", err)
	}
	if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "remember the current-thread decision is use sqlite", nil); err != nil {
		t.Fatalf("append seed message: %v", err)
	}
	for i := 0; i < chatTranscriptTurns+2; i++ {
		if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "assistant", "current filler "+strconv.Itoa(i), nil); err != nil {
			t.Fatalf("append filler message %d: %v", i, err)
		}
	}

	detail, err := srv.chat.GetThread(context.Background(), thread.ID)
	if err != nil {
		t.Fatalf("get thread: %v", err)
	}
	_, user := srv.buildChatLLMMessages(context.Background(), detail)
	if !strings.Contains(user, "EARLIER THREAD MEMORY") || !strings.Contains(user, "use sqlite") {
		t.Fatalf("expected earlier current-thread memory in user prompt, got %q", user)
	}
	if !strings.Contains(user, "RELATED CHAT MEMORY") || !strings.Contains(user, "amber bridge") {
		t.Fatalf("expected bounded related chat memory in user prompt, got %q", user)
	}
}

func TestChatLLMMessagesBoundTranscriptAfterLargeAssistantReply(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)

	thread, err := srv.chat.CreateThread(context.Background(), "large assistant transcript", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	large := strings.Repeat("z", chatPromptTranscriptMessageRunes*4)
	for i := 0; i < chatTranscriptTurns; i++ {
		if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "assistant", large, nil); err != nil {
			t.Fatalf("append large assistant message %d: %v", i, err)
		}
	}
	if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "latest operator turn", nil); err != nil {
		t.Fatalf("append latest user message: %v", err)
	}

	detail, err := srv.chat.GetThread(context.Background(), thread.ID)
	if err != nil {
		t.Fatalf("get thread: %v", err)
	}
	_, user := srv.buildChatLLMMessages(context.Background(), detail)
	if !strings.Contains(user, "latest operator turn") {
		t.Fatalf("expected latest user turn in prompt, got %q", user)
	}
	if len(user) > chatPromptTranscriptTotalRunes+chatThreadMemoryContextMaxRunes+chatCrossThreadContextMaxRunes+chatMemoryObservationMaxRunes+4096 {
		t.Fatalf("expected bounded prompt, got %d chars", len(user))
	}
	if strings.Contains(user, strings.Repeat("z", chatPromptTranscriptMessageRunes+1)) {
		t.Fatalf("expected large assistant content to be truncated")
	}
}

func TestChatLLMMessagesIncludeMemoryObservations(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	obs, err := srv.memory.RecordObservation(context.Background(), memory.RecordObservationRequest{
		Type:              "decision",
		Summary:           "The operator prefers compact context cards for memory-heavy chat.",
		RawContent:        "Memory observation fallback content",
		OriginKind:        "test",
		OriginID:          "chat-memory-observation",
		Confidence:        0.9,
		VerificationState: "observed",
	})
	if err != nil {
		t.Fatalf("record observation: %v", err)
	}
	if obs.ID == 0 {
		t.Fatalf("expected observation id")
	}

	thread, err := srv.chat.CreateThread(context.Background(), "memory observations", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "what should you remember?", nil); err != nil {
		t.Fatalf("append message: %v", err)
	}

	detail, err := srv.chat.GetThread(context.Background(), thread.ID)
	if err != nil {
		t.Fatalf("get thread: %v", err)
	}
	_, user := srv.buildChatLLMMessages(context.Background(), detail)
	if !strings.Contains(user, "MEMORY OBSERVATIONS") || !strings.Contains(user, "compact context cards") {
		t.Fatalf("expected memory observations in user prompt, got %q", user)
	}
}

func TestChatLLMMessagesIncludeMemoryObservationsAfterStoreReopen(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st1, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open initial store: %v", err)
	}
	memorySvc := memory.New(st1.DB)
	if _, err := memorySvc.RecordObservation(ctx, memory.RecordObservationRequest{
		Type:              "decision",
		Summary:           "Cross-session recall marker: remember the basalt notebook.",
		RawContent:        "The basalt notebook belongs in reopened chat memory.",
		OriginKind:        "test",
		OriginID:          "cross-session-memory-observation",
		Confidence:        0.95,
		VerificationState: "observed",
	}); err != nil {
		t.Fatalf("record observation before reopen: %v", err)
	}
	if err := st1.Close(); err != nil {
		t.Fatalf("close initial store: %v", err)
	}

	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = st2.Close() })
	srv2 := NewServer(st2, config.Config{DataDir: dataDir, WorkspaceDir: workspaceDir})
	t.Cleanup(func() { srv2.ShutdownWatch() })

	thread, err := srv2.chat.CreateThread(ctx, "reopened memory observations", nil)
	if err != nil {
		t.Fatalf("create reopened thread: %v", err)
	}
	if _, err := srv2.chat.AppendMessage(ctx, thread.ID, "user", "what should you remember after reopen?", nil); err != nil {
		t.Fatalf("append reopened message: %v", err)
	}
	detail, err := srv2.chat.GetThread(ctx, thread.ID)
	if err != nil {
		t.Fatalf("get reopened thread: %v", err)
	}
	_, user := srv2.buildChatLLMMessages(ctx, detail)
	if !strings.Contains(user, "MEMORY OBSERVATIONS") || !strings.Contains(user, "basalt notebook") {
		t.Fatalf("expected reopened memory observation in user prompt, got %q", user)
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

func TestChatPostSyncStripsRawModelPlanningPreamble(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "We need to answer: \"How are you doing today?\"\n\nFinal answer: I'm operational and ready."
	srv.modelRuntime = fakeRuntime
	t.Setenv("OLLAMA_MODEL", "qwen2.5-coder")
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:1")

	thread, err := srv.chat.CreateThread(context.Background(), "runtime planning strip", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"How are you doing today?","requestAssistant":true,"syncAssistant":true}`)
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
	if got := strings.TrimSpace(payload.AssistantMessage.Content); got != "I'm operational and ready." {
		t.Fatalf("expected final answer only, got=%q", got)
	}
	if strings.Contains(payload.AssistantMessage.Content, "We need to answer") || strings.Contains(payload.AssistantMessage.Content, "Final answer") {
		t.Fatalf("assistant content leaked planning scaffold: %q", payload.AssistantMessage.Content)
	}
	warnings, ok := payload.AssistantMessage.Metadata["assistantContentWarnings"].([]any)
	if !ok {
		t.Fatalf("expected assistantContentWarnings metadata, got %#v", payload.AssistantMessage.Metadata["assistantContentWarnings"])
	}
	if !containsAnyString(warnings, "stripped_reasoning_scaffold") {
		t.Fatalf("expected stripped_reasoning_scaffold warning, got %#v", warnings)
	}
}

func TestChatPostSyncAnswersOperatorNameFromThreadMemory(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "Your name is User."
	srv.modelRuntime = fakeRuntime
	t.Setenv("OLLAMA_MODEL", "qwen2.5-coder")
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:1")

	thread, err := srv.chat.CreateThread(context.Background(), "operator name recall", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "My name is Robert.", nil); err != nil {
		t.Fatalf("append name message: %v", err)
	}

	raw := []byte(`{"content":"What is my name?","requestAssistant":true,"syncAssistant":true}`)
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
	if got := strings.TrimSpace(payload.AssistantMessage.Content); got != "Your name is Robert." {
		t.Fatalf("expected deterministic name recall, got=%q", got)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model call for deterministic name recall, got %d", fakeRuntime.chatCalls)
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

func TestSanitizeAssistantVisibleContentStripsWeNeedReasoningLeak(t *testing.T) {
	content, warnings := sanitizeAssistantVisibleContent("We need to parse the operator's statement. I should inspect tools before answering.")
	if content != "" {
		t.Fatalf("expected reasoning leak to be fully stripped, got=%q", content)
	}
	if !containsString(warnings, "stripped_reasoning_scaffold") {
		t.Fatalf("expected stripped_reasoning_scaffold warning, got=%#v", warnings)
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

func TestSanitizeAssistantVisibleContentStripsCopiedThreadTranscript(t *testing.T) {
	content, warnings := sanitizeAssistantVisibleContent("I am FORGE.\n\n---\nTHREAD TITLE: copied context\n---\nUSER: Who are you?")
	if content != "I am FORGE." {
		t.Fatalf("expected copied transcript to be stripped, got=%q", content)
	}
	if !containsString(warnings, "stripped_synthetic_transcript_turn") {
		t.Fatalf("expected synthetic transcript warning, got=%#v", warnings)
	}

	content, warnings = sanitizeAssistantVisibleContent("I am FORGE.\n\n---\nUSER: copied context\n\n---\nASSISTANT #1: copied answer")
	if content != "I am FORGE." {
		t.Fatalf("expected copied user transcript to be stripped, got=%q", content)
	}
	if !containsString(warnings, "stripped_synthetic_transcript_turn") {
		t.Fatalf("expected synthetic transcript warning for copied user block, got=%#v", warnings)
	}

	content, warnings = sanitizeAssistantVisibleContent("I AM FORGE.\n\n---")
	if content != "I am FORGE." {
		t.Fatalf("expected canonical FORGE identity without trailing fence, got=%q", content)
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

	for _, content := range []string{
		"What is your name?",
		"Who are you? Start your answer with exactly: I am FORGE.",
	} {
		raw, err := json.Marshal(map[string]any{"content": content, "requestAssistant": true, "syncAssistant": true})
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
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
			t.Fatalf("expected deterministic FORGE identity for %q, got=%q", content, got)
		}
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
}

func TestChatPostSyncAnswersOperatorNameFromCommitMemoryClaim(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "operator name deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "Commit to memory, I am Robert.", nil); err != nil {
		t.Fatalf("append name claim: %v", err)
	}

	raw := []byte(`{"content":"Who am I?","requestAssistant":true,"syncAssistant":true}`)
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
	if got := strings.TrimSpace(payload.AssistantMessage.Content); got != "Your name is Robert." {
		t.Fatalf("expected remembered operator name, got=%q", got)
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
	if trace["hyperlane_intent_type"] != "status_query" || trace["hyperlane_route"] != "structured.status" {
		t.Fatalf("expected hyperlane status trace, got %#v", trace)
	}
	if trace["modelruntime_avoided"] != true || trace["context_compile_avoided"] != true || trace["gateway_avoided"] != true {
		t.Fatalf("expected avoided flags in trace, got %#v", trace)
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

func TestChatPostSyncRoutesModelRuntimeStatusWithoutModel(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.loaded["mistral-7b-instruct"] = true
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "modelruntime status deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"modelruntime status","requestAssistant":true,"syncAssistant":true}`)
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
	if payload.AssistantMessage == nil || !strings.Contains(payload.AssistantMessage.Content, "Modelruntime fast path") {
		t.Fatalf("expected modelruntime fast path response, got %#v", payload.AssistantMessage)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime chat calls, got %d", fakeRuntime.chatCalls)
	}
	if fakeRuntime.healthCalls != 0 || fakeRuntime.queueCalls != 0 || fakeRuntime.loadedCalls != 0 || fakeRuntime.listCalls != 0 {
		t.Fatalf("expected no modelruntime status probes on no-model route, got health=%d queue=%d loaded=%d list=%d", fakeRuntime.healthCalls, fakeRuntime.queueCalls, fakeRuntime.loadedCalls, fakeRuntime.listCalls)
	}
	trace := metadataMap(payload.AssistantMessage.Metadata, "chatLatencyTrace")
	if trace["hyperlane_intent_type"] != "modelruntime_status" || trace["hyperlane_route"] != "structured.modelruntime_status" {
		t.Fatalf("expected modelruntime hyperlane trace, got %#v", trace)
	}
}

func TestChatPostSyncRoutesDreamReportInspectionWithoutModel(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "dream report deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"show latest Dream report","requestAssistant":true,"syncAssistant":true}`)
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
	if payload.AssistantMessage == nil || !strings.Contains(payload.AssistantMessage.Content, "Dream Mode") {
		t.Fatalf("expected Dream Mode fast path response, got %#v", payload.AssistantMessage)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
	trace := metadataMap(payload.AssistantMessage.Metadata, "chatLatencyTrace")
	if trace["hyperlane_intent_type"] != "dream_report_inspection" || trace["hyperlane_route"] != "structured.dream_reports" {
		t.Fatalf("expected dream hyperlane trace, got %#v", trace)
	}
}

func TestChatPostSyncNoModelStatusDoesNotRequireGateway(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime
	srv.gateway = nil

	thread, err := srv.chat.CreateThread(context.Background(), "status no gateway", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"forge status","requestAssistant":true,"syncAssistant":true}`)
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
	if payload.AssistantMessage == nil || !strings.Contains(payload.AssistantMessage.Content, "Fast path: no model call") {
		t.Fatalf("expected no-model status response without gateway, got %#v", payload.AssistantMessage)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
	trace := metadataMap(payload.AssistantMessage.Metadata, "chatLatencyTrace")
	if trace["gateway_avoided"] != true {
		t.Fatalf("expected gateway_avoided trace, got %#v", trace)
	}
}

func TestChatPostForcedToolOmissionUsesForgeGatewayNotModelCapabilityClaim(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	var sawOllama bool
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected ollama path %s", r.URL.Path)
		}
		sawOllama = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"I can't access the web or use tools from here."},"done":true}`))
	}))
	defer ollama.Close()

	ctx := context.Background()
	if err := upsertSetting(ctx, st.DB, "ollama_base_url", ollama.URL); err != nil {
		t.Fatalf("set ollama base url: %v", err)
	}
	if err := upsertSetting(ctx, st.DB, "ollama_model", "tool-omission-model"); err != nil {
		t.Fatalf("set ollama model: %v", err)
	}

	thread, err := srv.chat.CreateThread(ctx, "tool omission authority", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"search the web for FORGE docs","requestAssistant":true,"syncAssistant":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if !sawOllama {
		t.Fatalf("expected ollama request")
	}
	var payload chatPostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if payload.AssistantMessage == nil {
		t.Fatalf("expected assistant message")
	}
	content := strings.ToLower(payload.AssistantMessage.Content)
	if strings.Contains(content, "can't access") || strings.Contains(content, "cannot access") || strings.Contains(content, "use tools from here") {
		t.Fatalf("model capability claim leaked into assistant content: %q", payload.AssistantMessage.Content)
	}
	if !strings.Contains(payload.AssistantMessage.Content, "FORGE (deterministic web search):") {
		t.Fatalf("expected FORGE gateway-controlled fallback response, got %q", payload.AssistantMessage.Content)
	}
	activity := metadataMap(payload.AssistantMessage.Metadata, "toolGatewayActivity")
	if activity["modelContentDiscarded"] != true {
		t.Fatalf("expected modelContentDiscarded=true activity=%#v", activity)
	}
	if got := strings.TrimSpace(asString(activity["toolSelected"])); got != "web.search" {
		t.Fatalf("toolSelected=%q activity=%#v", got, activity)
	}
	if activity["syntheticToolExecution"] != true {
		t.Fatalf("expected syntheticToolExecution=true activity=%#v", activity)
	}
	if got := strings.TrimSpace(asString(activity["executionState"])); got == "model_omitted_tool_calls" || got == "" {
		t.Fatalf("executionState=%q activity=%#v", got, activity)
	}
	assertGatewayInvocationCount(t, st, 1)
}

func TestChatPostForcedToolMismatchUsesDeterministicFallback(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	var sawOllama bool
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected ollama path %s", r.URL.Path)
		}
		sawOllama = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","tool_calls":[{"id":"call-wrong-tool","type":"function","function":{"name":"forge_net_connectivity","arguments":{"input":{"target":"10.150.1.9:22"}}}}]},"done":true}`))
	}))
	defer ollama.Close()

	ctx := context.Background()
	if err := upsertSetting(ctx, st.DB, "ollama_base_url", ollama.URL); err != nil {
		t.Fatalf("set ollama base url: %v", err)
	}
	if err := upsertSetting(ctx, st.DB, "ollama_model", "tool-mismatch-model"); err != nil {
		t.Fatalf("set ollama model: %v", err)
	}

	thread, err := srv.chat.CreateThread(ctx, "tool mismatch authority", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"search the web for FORGE docs","requestAssistant":true,"syncAssistant":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if !sawOllama {
		t.Fatalf("expected ollama request")
	}
	var payload chatPostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if payload.AssistantMessage == nil {
		t.Fatalf("expected assistant message")
	}
	if !strings.Contains(payload.AssistantMessage.Content, "FORGE (deterministic web search):") {
		t.Fatalf("expected deterministic web search fallback, got %q", payload.AssistantMessage.Content)
	}
	activity := metadataMap(payload.AssistantMessage.Metadata, "toolGatewayActivity")
	if activity["modelToolCallsDiscarded"] != true {
		t.Fatalf("expected modelToolCallsDiscarded=true activity=%#v", activity)
	}
	if got := strings.TrimSpace(asString(activity["toolSelected"])); got != "web.search" {
		t.Fatalf("toolSelected=%q activity=%#v", got, activity)
	}
	if got := gatewayInvocationTools(t, st); strings.Join(got, ",") != "web.search" {
		t.Fatalf("expected only web.search gateway invocation, got %v", got)
	}
}

func TestChatPostRemoteSSHBannerUsesDesktopOpenNotLocalWrite(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	thread, err := srv.chat.CreateThread(context.Background(), "remote ssh banner", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	raw := []byte(`{"content":"I pre approve this plan. Open terminall, ssh into robert@10.150.1.2 password redacted-secret. Create a directory labled Auto_Banner. Inside that directory create a python program called hello_world.py. I want it to be a scrolling flashing banner with the words \"HELLO WORLD\".","requestAssistant":true,"syncAssistant":true}`)
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
	activity := metadataMap(payload.AssistantMessage.Metadata, "toolGatewayActivity")
	if got := strings.TrimSpace(asString(activity["toolSelected"])); got != "desktop.open" {
		t.Fatalf("toolSelected=%q activity=%#v content=%q", got, activity, payload.AssistantMessage.Content)
	}
	tools := gatewayInvocationTools(t, st)
	if strings.Join(tools, ",") != "desktop.open" {
		t.Fatalf("expected only desktop.open gateway invocation, got %v", tools)
	}
	if strings.Contains(payload.AssistantMessage.Content, "FORGE (deterministic python):") {
		t.Fatalf("expected remote desktop path, got local python response %q", payload.AssistantMessage.Content)
	}
}

func TestChatPostRepoExplorationUsesGatewayNotModelCommandSuggestions(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "bash\nCopy\ncat README.md\ncat AGENTS.md"
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "repo exploration", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"You can explore your repo. Familiarize yourself with yourself lol","requestAssistant":true,"syncAssistant":true}`)
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
	if strings.Contains(payload.AssistantMessage.Content, "cat README.md") || strings.Contains(payload.AssistantMessage.Content, "bash\nCopy") {
		t.Fatalf("model command suggestion leaked into assistant content: %q", payload.AssistantMessage.Content)
	}
	if !strings.Contains(payload.AssistantMessage.Content, "FORGE (deterministic repo):") {
		t.Fatalf("expected deterministic repo inspection response, got %q", payload.AssistantMessage.Content)
	}
	activity := metadataMap(payload.AssistantMessage.Metadata, "toolGatewayActivity")
	if got := strings.TrimSpace(asString(activity["toolSelected"])); got != "repo.inspect" {
		t.Fatalf("toolSelected=%q activity=%#v", got, activity)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime call for deterministic repo inspection, got %d", fakeRuntime.chatCalls)
	}
	if got := gatewayInvocationTools(t, st); strings.Join(got, ",") != "repo.inspect" {
		t.Fatalf("expected only repo.inspect gateway invocation, got %v", got)
	}
}

func gatewayInvocationTools(t *testing.T, st *store.Store) []string {
	t.Helper()
	rows, err := st.DB.Query(`SELECT tool_id FROM gateway_invocations ORDER BY id`)
	if err != nil {
		t.Fatalf("query gateway invocation tools: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var toolID string
		if err := rows.Scan(&toolID); err != nil {
			t.Fatalf("scan gateway invocation tool: %v", err)
		}
		out = append(out, toolID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate gateway invocation tools: %v", err)
	}
	return out
}

func TestChatPostSyncRestoreInspectorDoesNotLeakOtherThreadMetadata(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	other, err := srv.chat.CreateThread(context.Background(), "other restore thread", nil)
	if err != nil {
		t.Fatalf("create other thread: %v", err)
	}
	if _, err := srv.chat.AppendMessage(context.Background(), other.ID, "assistant", "other restore", map[string]any{"restoreSummary": "secret-other-workspace-restore"}); err != nil {
		t.Fatalf("append other restore: %v", err)
	}
	thread, err := srv.chat.CreateThread(context.Background(), "current restore thread", nil)
	if err != nil {
		t.Fatalf("create current thread: %v", err)
	}
	raw := []byte(`{"content":"show recent restore decisions","requestAssistant":true,"syncAssistant":true}`)
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
	if strings.Contains(payload.AssistantMessage.Content, "secret-other-workspace-restore") {
		t.Fatalf("restore inspector leaked other thread metadata: %q", payload.AssistantMessage.Content)
	}
	if !strings.Contains(payload.AssistantMessage.Content, "No restore package") {
		t.Fatalf("expected empty restore state response, got %q", payload.AssistantMessage.Content)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
}

func TestChatPostSyncRoutesDownloadSorterThroughGateway(t *testing.T) {
	dataDir := t.TempDir()
	homeDir := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(filepath.Join(homeDir, "Downloads"), 0o755); err != nil {
		t.Fatalf("mkdir fake home: %v", err)
	}
	setFakeHomeEnv(t, homeDir)

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	srv := NewServer(st, config.Config{
		DataDir:      dataDir,
		WorkspaceDir: homeDir,
	})
	t.Cleanup(func() { srv.ShutdownWatch() })
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "I cannot access the filesystem."
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "download sorter deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"Create a folder in the Downloads directory labled Python_Scripts/. Inside the folder create a python script that will make anything I download get sorted into a folder in the downloads folder.","requestAssistant":true,"syncAssistant":true}`)
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
	if !strings.Contains(payload.AssistantMessage.Content, "FORGE (deterministic download sorter): gateway needs_approval") {
		t.Fatalf("expected governed gateway approval result, got %q", payload.AssistantMessage.Content)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
	scriptPath := filepath.Join(homeDir, "Downloads", "Python_Scripts", "sort_downloads.py")
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Fatalf("expected no unapproved write at %s, err=%v", scriptPath, err)
	}
	activity := metadataMap(payload.AssistantMessage.Metadata, "toolGatewayActivity")
	if activity["executionState"] != "needs_approval" || activity["toolSelected"] != "fs.write" {
		t.Fatalf("expected ok fs.write activity, got %#v", activity)
	}
}

func TestChatPostSyncMultiSVGUsesDeterministicGatewayShortcut(t *testing.T) {
	dataDir := t.TempDir()
	homeDir := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(filepath.Join(homeDir, "Downloads"), 0o755); err != nil {
		t.Fatalf("mkdir fake home: %v", err)
	}
	setFakeHomeEnv(t, homeDir)

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	srv := NewServer(st, config.Config{
		DataDir:      dataDir,
		WorkspaceDir: homeDir,
	})
	t.Cleanup(func() { srv.ShutdownWatch() })
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "I would create the files manually."
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "multi svg deterministic", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"Create a directory in Downloads called RandomSVGs. Inside that folder create an svg file of a turtle and then one of stitch.","requestAssistant":true,"syncAssistant":true}`)
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
	if !strings.Contains(payload.AssistantMessage.Content, "FORGE (deterministic svg): gateway needs_approval") {
		t.Fatalf("expected governed deterministic svg approval result, got %q", payload.AssistantMessage.Content)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
	activity := metadataMap(payload.AssistantMessage.Metadata, "toolGatewayActivity")
	if activity["executionState"] != "needs_approval" || activity["toolSelected"] != "fs.write" {
		t.Fatalf("expected needs_approval fs.write activity, got %#v", activity)
	}
	args := metadataMap(activity, "toolArgs")
	paths, ok := args["paths"].([]any)
	if !ok || len(paths) != 2 {
		t.Fatalf("expected two governed write paths, got %#v", args["paths"])
	}
	for _, rawPath := range paths {
		path := asString(rawPath)
		if !strings.HasPrefix(path, "~/Downloads/RandomSVGs/") {
			t.Fatalf("expected home Downloads path alias, got %q", path)
		}
		if strings.HasPrefix(path, "/Downloads") || strings.Contains(path, "../") {
			t.Fatalf("unsafe write path %q", path)
		}
	}
	if _, err := os.Stat(filepath.Join(homeDir, "Downloads", "RandomSVGs", "turtle.svg")); !os.IsNotExist(err) {
		t.Fatalf("expected no unapproved turtle write, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(homeDir, "Downloads", "RandomSVGs", "stitch.svg")); !os.IsNotExist(err) {
		t.Fatalf("expected no unapproved stitch write, err=%v", err)
	}
}

func TestChatPostSyncSameDirectoryWebpageUsesPriorGatewayDirectory(t *testing.T) {
	dataDir := t.TempDir()
	homeDir := filepath.Join(t.TempDir(), "home")
	priorDir := filepath.Join(homeDir, "Downloads", "PeanutButterJellyTime")
	if err := os.MkdirAll(priorDir, 0o755); err != nil {
		t.Fatalf("mkdir fake prior dir: %v", err)
	}
	setFakeHomeEnv(t, homeDir)

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	srv := NewServer(st, config.Config{
		DataDir:      dataDir,
		WorkspaceDir: homeDir,
	})
	t.Cleanup(func() { srv.ShutdownWatch() })
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.chatContent = "I would write the prior SVG again."
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "same directory webpage", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	priorPath := filepath.Join(priorDir, "flower.svg")
	if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "assistant", "Gateway job succeeded.", map[string]any{
		"correlationId": "corr-prior-write",
		"toolGatewayActivity": map[string]any{
			"executionState": "ok",
			"toolSelected":   "fs.write",
			"executionResult": map[string]any{
				"path":  priorPath,
				"bytes": 821,
			},
		},
	}); err != nil {
		t.Fatalf("append prior assistant: %v", err)
	}

	raw := []byte(`{"content":"In the same directory, create a test webpage. I would like it to look like it belongs to a video game journal site.","requestAssistant":true,"syncAssistant":true}`)
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
	if !strings.Contains(payload.AssistantMessage.Content, "FORGE (deterministic webpage): gateway needs_approval") {
		t.Fatalf("expected governed deterministic webpage approval result, got %q", payload.AssistantMessage.Content)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected no model runtime calls, got %d", fakeRuntime.chatCalls)
	}
	activity := metadataMap(payload.AssistantMessage.Metadata, "toolGatewayActivity")
	args := metadataMap(activity, "toolArgs")
	gotPath := strings.TrimSpace(asString(args["path"]))
	if !strings.HasSuffix(gotPath, "/Downloads/PeanutButterJellyTime/test-webpage.html") {
		t.Fatalf("expected test webpage path in prior directory, got %q", gotPath)
	}
	if strings.HasSuffix(gotPath, "flower.svg") {
		t.Fatalf("webpage request reused stale SVG path: %q", gotPath)
	}
	if _, err := os.Stat(filepath.Join(priorDir, "test-webpage.html")); !os.IsNotExist(err) {
		t.Fatalf("expected no unapproved webpage write, err=%v", err)
	}
}

func TestChatPostSyncAmbiguousRequestFallsThroughToModelRuntime(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.loaded["mistral-7b-instruct"] = true
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

func TestChatPostSkipsModelRuntimeWhenSelectedBackendIsUnhealthy(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.models["ollama-chat"] = ModelRuntimeModel{
		ID:           "ollama-chat",
		DisplayName:  "Ollama Chat",
		Backend:      "ollama_compat",
		Format:       "openai_compat",
		Status:       "available",
		Capabilities: []string{"chat", "completion"},
	}
	fakeRuntime.health = &ModelRuntimeHealth{
		OK:     false,
		Status: "degraded",
		Details: map[string]any{
			"backends": map[string]map[string]any{
				"ollama_compat": {
					"healthy": false,
					"detail":  "connect: connection refused",
				},
			},
		},
	}
	fakeRuntime.chatErr = &modelRuntimeError{
		status:  http.StatusServiceUnavailable,
		code:    "MODEL_CHAT_RETRY_EXHAUSTED",
		message: "chat execution retry exhausted: model backend unavailable: ollama_compat",
	}
	srv.modelRuntime = fakeRuntime
	t.Setenv("OLLAMA_MODEL", "qwen2.5-coder")
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:1")

	thread, err := srv.chat.CreateThread(context.Background(), "unloaded runtime fallback", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw := []byte(`{"content":"That took 2 minutes.","requestAssistant":true,"syncAssistant":true,"modelId":"ollama-chat"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/messages", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected preflight to skip unavailable model runtime backend, got chatCalls=%d", fakeRuntime.chatCalls)
	}
	var payload chatPostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if payload.AssistantMessage == nil {
		t.Fatalf("expected fallback assistant message")
	}
	stages := payload.AssistantMessage.Metadata["toolPipeline"].(map[string]any)["stages"].([]any)
	serialized, _ := json.Marshal(stages)
	if !strings.Contains(string(serialized), "model runtime backend unavailable: ollama_compat") {
		t.Fatalf("expected fast backend-unloaded fallback stage, got %s", serialized)
	}
}

func TestChatPostSyncReportRequestUsesReportOutputBudget(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	fakeRuntime.loaded["mistral-7b-instruct"] = true
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

func TestChatAssistantStreamPrefersModelRuntimePlainBeforeNativeOllamaStream(t *testing.T) {
	nativeStreamCalls := 0
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)
			return
		}
		nativeStreamCalls++
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"message":{"content":"native stream"},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"done":true}` + "\n"))
	}))
	defer ollamaServer.Close()
	t.Setenv("OLLAMA_BASE_URL", ollamaServer.URL)
	t.Setenv("OLLAMA_MODEL", "native-model")

	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime plain before native stream", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "plain runtime first please", map[string]any{"source": "operator"})
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
	if strings.Contains(body, `"ollamaStream":true`) {
		t.Fatalf("did not expect native ollama stream metadata, got body=%s", body)
	}
	if nativeStreamCalls != 0 {
		t.Fatalf("expected native ollama stream to be skipped before model runtime plain chat, got %d calls", nativeStreamCalls)
	}
	if fakeRuntime.chatCalls != 1 {
		t.Fatalf("expected one model runtime chat call, got %d", fakeRuntime.chatCalls)
	}
}

type fakeStreamingModelRuntime struct {
	*fakeModelRuntime
	streamCalls     int
	reasoningTokens []string
}

func (f *fakeStreamingModelRuntime) StreamChat(_ context.Context, req ModelRuntimeChatRequest, onToken func(ModelRuntimeChatStreamToken) error) (ModelRuntimeChatResult, error) {
	f.mu.Lock()
	f.streamCalls++
	f.lastMeta = req.Meta
	f.lastChat = req
	streamErr := f.streamErr
	tokens := append([]string(nil), f.streamTokens...)
	reasoningTokens := append([]string(nil), f.reasoningTokens...)
	if len(tokens) == 0 {
		tokens = []string{"cloud ", "stream"}
	}
	f.mu.Unlock()
	if streamErr != nil {
		return ModelRuntimeChatResult{}, streamErr
	}
	for i, token := range reasoningTokens {
		if err := onToken(ModelRuntimeChatStreamToken{Reasoning: token, Index: i}); err != nil {
			return ModelRuntimeChatResult{}, err
		}
	}
	for i, token := range tokens {
		if err := onToken(ModelRuntimeChatStreamToken{Text: token, Index: i}); err != nil {
			return ModelRuntimeChatResult{}, err
		}
	}
	content := strings.Join(tokens, "")
	return ModelRuntimeChatResult{
		Content:      content,
		FinishReason: "stop",
		Usage: ModelRuntimeUsage{
			PromptTokens:     4,
			CompletionTokens: len(tokens),
			TotalTokens:      6,
		},
		DurationMs: 25,
		Backend:    "openai_compat",
		ModelID:    req.ModelID,
		AuditID:    "audit-stream",
		Proposal:   testModelRuntimeProposal(req, content, "audit-stream", len([]byte(content))),
	}, nil
}

type cancelAfterFirstTokenModelRuntime struct {
	*fakeModelRuntime
	cancel         context.CancelFunc
	callbackCalls  int
	callbackErr    error
	streamFinished bool
}

func (f *cancelAfterFirstTokenModelRuntime) StreamChat(ctx context.Context, req ModelRuntimeChatRequest, onToken func(ModelRuntimeChatStreamToken) error) (ModelRuntimeChatResult, error) {
	f.mu.Lock()
	f.lastMeta = req.Meta
	f.lastChat = req
	f.mu.Unlock()

	tokens := []string{
		strings.Repeat("a", assistantStreamFlushChars+32),
		"should-not-be-processed-after-cancel",
		"should-not-be-requested-after-cancel",
	}
	for i, token := range tokens {
		f.callbackCalls++
		err := onToken(ModelRuntimeChatStreamToken{Text: token, Index: i})
		if i == 0 {
			f.cancel()
		}
		if err != nil {
			f.callbackErr = err
			return ModelRuntimeChatResult{}, err
		}
	}
	f.streamFinished = true
	return ModelRuntimeChatResult{Content: strings.Join(tokens, ""), FinishReason: "stop", Backend: "fake", ModelID: req.ModelID}, nil
}

func TestChatAssistantModelRuntimeStreamStopsTokenCallbackAfterClientCancel(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fakeRuntime := &cancelAfterFirstTokenModelRuntime{fakeModelRuntime: newFakeModelRuntime(), cancel: cancel}
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime stream canceled", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "cancel runtime stream", map[string]any{"source": "operator"})
	if err != nil {
		t.Fatalf("append user message: %v", err)
	}
	th, err := srv.chat.GetThread(context.Background(), thread.ID)
	if err != nil {
		t.Fatalf("get thread: %v", err)
	}

	var emitted []string
	am, reason := srv.completeAssistantWithModelRuntimeStream(
		ctx,
		thread.ID,
		um.ID,
		th,
		um.Content,
		"corr-cancel-stream",
		nil,
		nil,
		"",
		time.Now(),
		classifyChatPerformance(um.Content),
		func(event string, payload map[string]any) {
			if text, _ := payload["text"].(string); text != "" {
				emitted = append(emitted, text)
			}
		},
	)

	if am != nil {
		t.Fatalf("expected canceled stream not to append assistant message, got %#v", am)
	}
	if !strings.Contains(reason, "MODEL_REQUEST_CANCELED") {
		t.Fatalf("expected canceled modelruntime reason, got %q", reason)
	}
	if fakeRuntime.callbackCalls != 2 {
		t.Fatalf("expected callback to stop on first post-cancel token, got %d callback calls", fakeRuntime.callbackCalls)
	}
	if !errors.Is(fakeRuntime.callbackErr, context.Canceled) {
		t.Fatalf("expected callback to return context.Canceled, got %v", fakeRuntime.callbackErr)
	}
	if fakeRuntime.streamFinished {
		t.Fatalf("expected runtime stream to stop after cancellation")
	}
	for _, chunk := range emitted {
		if strings.Contains(chunk, "should-not-be-processed-after-cancel") {
			t.Fatalf("emitted post-cancel token chunk: %q", chunk)
		}
	}
}

func TestChatAssistantStreamUsesModelRuntimeStreamingWhenOllamaUnavailable(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := &fakeStreamingModelRuntime{fakeModelRuntime: newFakeModelRuntime()}
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime stream", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "hello via runtime stream", map[string]any{"source": "operator"})
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
	if !strings.Contains(body, "event: token") || !strings.Contains(body, "cloud ") || !strings.Contains(body, "stream") {
		t.Fatalf("expected streamed runtime token events, got body=%s", body)
	}
	if strings.Contains(body, "stream_downgrade") {
		t.Fatalf("did not expect stream downgrade when model runtime streaming is available, got body=%s", body)
	}
	if !strings.Contains(body, `"modelRuntimeStream":true`) {
		t.Fatalf("expected modelRuntimeStream metadata, got body=%s", body)
	}
	if fakeRuntime.streamCalls != 1 {
		t.Fatalf("expected one runtime stream call, got %d", fakeRuntime.streamCalls)
	}
	if fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected streaming path to avoid sync chat call, got %d", fakeRuntime.chatCalls)
	}
}

func TestChatAssistantStreamDoesNotRefreshRemoteDiscoveryBeforeCloudToken(t *testing.T) {
	var modelListCalls int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			atomic.AddInt32(&modelListCalls, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "remote-qwen", "name": "Remote Qwen"},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"cloud \"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"stream\"},\"finish_reason\":\"stop\"}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer remote.Close()

	t.Setenv("FORGE_ENABLE_MODEL_RUNTIME", "true")
	t.Setenv("FORGE_ENABLE_OPENAI_COMPAT_API", "false")
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_ENDPOINT", remote.URL)
	t.Setenv("FORGE_MODEL_OPENAI_COMPAT_API_KEY", "")
	t.Setenv("FORGE_MODEL_VLLM_ENDPOINT", "")
	t.Setenv("FORGE_MODEL_VLLM_API_KEY", "")
	t.Setenv("FORGE_MODEL_DEFAULT_BACKEND", "openai_compat")
	t.Setenv("FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE", "false")
	t.Setenv("FORGE_MODEL_HOME", t.TempDir())

	runtimeSvc := initModelRuntimeService(config.Load(), nil)
	if runtimeSvc == nil {
		t.Fatalf("expected model runtime service")
	}
	if _, err := runtimeSvc.LoadModel(context.Background(), "remote-qwen", ModelRuntimeControlRequest{
		Actor:  "operator",
		Source: "test",
		Meta:   ModelRuntimeRequestMeta{WorkspaceID: "ws-test"},
	}); err != nil {
		t.Fatalf("load remote model: %v", err)
	}
	readyModelListCalls := atomic.LoadInt32(&modelListCalls)
	if readyModelListCalls == 0 {
		t.Fatalf("expected startup remote discovery call")
	}

	srv, _ := newBackupAuditHarness(t)
	srv.modelRuntime = runtimeSvc
	thread, err := srv.chat.CreateThread(context.Background(), "remote stream discovery budget", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "answer quickly", map[string]any{
		"requestedModelId": "remote-qwen",
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
	body := rr.Body.String()
	if !strings.Contains(body, "cloud stream") {
		t.Fatalf("expected streamed cloud response, got body=%s", body)
	}
	if got := atomic.LoadInt32(&modelListCalls); got != readyModelListCalls {
		t.Fatalf("chat stream refreshed remote discovery before streaming: calls=%d ready=%d", got, readyModelListCalls)
	}
}

func TestChatAssistantStreamAvoidsDeepModelRuntimePreflightBeforeStreaming(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := &fakeStreamingModelRuntime{fakeModelRuntime: newFakeModelRuntime()}
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime stream preflight budget", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "hello without slow routing checks", map[string]any{"source": "operator"})
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
	if fakeRuntime.streamCalls != 1 {
		t.Fatalf("expected one runtime stream call, got %d", fakeRuntime.streamCalls)
	}
	if fakeRuntime.queueCalls != 1 {
		t.Fatalf("expected one scheduler queue preflight call, got %d", fakeRuntime.queueCalls)
	}
	if fakeRuntime.getCalls != 0 || fakeRuntime.healthCalls != 0 {
		t.Fatalf("streaming route should avoid deep model/health preflight before first token, get=%d health=%d", fakeRuntime.getCalls, fakeRuntime.healthCalls)
	}
}

func TestChatAssistantStreamEmitsModelRuntimeReasoningEvents(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	fakeRuntime := &fakeStreamingModelRuntime{
		fakeModelRuntime: newFakeModelRuntime(),
		reasoningTokens:  []string{"checking facts"},
	}
	srv.modelRuntime = fakeRuntime

	thread, err := srv.chat.CreateThread(context.Background(), "runtime reasoning stream", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "hello via runtime stream", map[string]any{"source": "operator"})
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
	if !strings.Contains(body, "event: reasoning") || !strings.Contains(body, `"text":"checking facts"`) {
		t.Fatalf("expected reasoning SSE event, got body=%s", body)
	}
	if !strings.Contains(body, "event: token") || !strings.Contains(body, `"text":"cloud "`) || !strings.Contains(body, `"text":"stream"`) {
		t.Fatalf("expected visible token SSE events, got body=%s", body)
	}
	if strings.Contains(body, `"content":"checking factscloud stream"`) || strings.Contains(body, `"content":"checking facts cloud stream"`) {
		t.Fatalf("reasoning leaked into assistant message content, got body=%s", body)
	}
	if strings.Contains(body, "stream_downgrade") {
		t.Fatalf("did not expect stream downgrade for reasoning-capable runtime stream, got body=%s", body)
	}
}

func TestChatAssistantStreamPrefersModelRuntimeStreamingOverConfiguredOllama(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	fakeRuntime := &fakeStreamingModelRuntime{fakeModelRuntime: newFakeModelRuntime()}
	srv.modelRuntime = fakeRuntime

	var ollamaCalled bool
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ollamaCalled = true
		http.Error(w, "native ollama should not be first for runtime-streamable plain chat", http.StatusBadGateway)
	}))
	defer ollama.Close()

	ctx := context.Background()
	if err := upsertSetting(ctx, st.DB, "ollama_base_url", ollama.URL); err != nil {
		t.Fatalf("set ollama base url: %v", err)
	}
	if err := upsertSetting(ctx, st.DB, "ollama_model", "configured-ollama"); err != nil {
		t.Fatalf("set ollama model: %v", err)
	}

	thread, err := srv.chat.CreateThread(ctx, "runtime stream preferred", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(ctx, thread.ID, "user", "hello via preferred runtime stream", map[string]any{"source": "operator"})
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
	if !strings.Contains(body, "event: token") || !strings.Contains(body, "cloud ") || !strings.Contains(body, "stream") {
		t.Fatalf("expected runtime token events, got body=%s", body)
	}
	if ollamaCalled {
		t.Fatalf("configured Ollama should not preempt available modelruntime streaming")
	}
	if fakeRuntime.streamCalls != 1 || fakeRuntime.chatCalls != 0 {
		t.Fatalf("expected one runtime stream and no sync chat, stream=%d chat=%d", fakeRuntime.streamCalls, fakeRuntime.chatCalls)
	}
}

func TestChatAssistantStreamRoutesStatusWithoutModel(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	fakeRuntime := newFakeModelRuntime()
	srv.modelRuntime = fakeRuntime
	before := canonicalCounts(t, st)

	thread, err := srv.chat.CreateThread(context.Background(), "status deterministic sse", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "forge status", map[string]any{"source": "operator"})
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
	if !strings.Contains(body, `"hyperlaneNoModel":true`) ||
		!strings.Contains(body, `"hyperlane_intent_type":"status_query"`) ||
		!strings.Contains(body, `"hyperlane_route":"structured.status"`) ||
		!strings.Contains(body, `"modelruntime_avoided":true`) ||
		!strings.Contains(body, `"gateway_avoided":true`) ||
		!strings.Contains(body, `"context_compile_avoided":true`) {
		t.Fatalf("expected hyperlane no-model trace in SSE payload, got body=%s", body)
	}
	if strings.Contains(body, `"modelRuntimeOk":true`) {
		t.Fatalf("expected no modelruntime success metadata on no-model route, got body=%s", body)
	}
	if fakeRuntime.chatCalls != 0 || fakeRuntime.healthCalls != 0 || fakeRuntime.queueCalls != 0 || fakeRuntime.loadedCalls != 0 {
		t.Fatalf("expected no modelruntime calls, chat=%d health=%d queue=%d loaded=%d", fakeRuntime.chatCalls, fakeRuntime.healthCalls, fakeRuntime.queueCalls, fakeRuntime.loadedCalls)
	}
	assertGatewayInvocationCount(t, st, 0)
	assertCanonicalCounts(t, st, before)
}

func TestChatAssistantStreamUsesNativeOllamaStreamForNoToolChat(t *testing.T) {
	srv, st := newBackupAuditHarness(t)

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
