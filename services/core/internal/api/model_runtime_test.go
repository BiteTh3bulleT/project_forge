package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

type fakeModelRuntime struct {
	mu          sync.Mutex
	models      map[string]ModelRuntimeModel
	loaded      map[string]bool
	chatErr     error
	healthErr   error
	importCalls int
	listCalls   int
	getCalls    int
	loadCalls   int
	unloadCalls int
	chatCalls   int
	healthCalls int
	queueCalls  int
	loadedCalls int
	lastMeta    ModelRuntimeRequestMeta
	lastImport  ModelRuntimeImportRequest
	lastControl ModelRuntimeControlRequest
	lastChat    ModelRuntimeChatRequest
}

func newFakeModelRuntime() *fakeModelRuntime {
	return &fakeModelRuntime{
		models: map[string]ModelRuntimeModel{
			"mistral-7b-instruct": {
				ID:           "mistral-7b-instruct",
				DisplayName:  "Mistral 7B Instruct",
				Backend:      "fake",
				Format:       "gguf",
				Status:       "available",
				Capabilities: []string{"chat", "completion"},
			},
		},
		loaded: map[string]bool{},
	}
}

func (f *fakeModelRuntime) ListModels(_ context.Context, req ModelRuntimeListRequest) ([]ModelRuntimeModel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	f.lastMeta = req.Meta
	out := make([]ModelRuntimeModel, 0, len(f.models))
	for _, model := range f.models {
		out = append(out, model)
	}
	return out, nil
}

func (f *fakeModelRuntime) GetModel(_ context.Context, modelID string, req ModelRuntimeRequestMeta) (ModelRuntimeModel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	f.lastMeta = req
	model, ok := f.models[modelID]
	if !ok {
		return ModelRuntimeModel{}, &modelRuntimeError{status: http.StatusNotFound, code: "MODEL_NOT_FOUND", message: "model not found"}
	}
	if f.loaded[modelID] {
		model.Status = "loaded"
	}
	return model, nil
}

func (f *fakeModelRuntime) ImportModel(_ context.Context, req ModelRuntimeImportRequest) (ModelRuntimeImportResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.importCalls++
	f.lastImport = req
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = "imported-model"
	}
	model := ModelRuntimeModel{
		ID:           id,
		DisplayName:  firstNonEmptyTest(req.DisplayName, "Imported Model"),
		Backend:      firstNonEmptyTest(req.Backend, "fake"),
		Format:       "gguf",
		Status:       "imported",
		Capabilities: append([]string(nil), req.Capabilities...),
		Metadata:     map[string]any{"sourcePath": req.Path},
	}
	if len(model.Capabilities) == 0 {
		model.Capabilities = []string{"chat", "completion"}
	}
	f.models[id] = model
	return ModelRuntimeImportResult{Model: model, Duplicate: false, ManagedPath: "/tmp/" + id, SourcePath: req.Path}, nil
}

func (f *fakeModelRuntime) ScanModels(_ context.Context, req ModelRuntimeControlRequest) ([]ModelRuntimeModel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastControl = req
	out := make([]ModelRuntimeModel, 0, len(f.models))
	for _, model := range f.models {
		out = append(out, model)
	}
	return out, nil
}

func (f *fakeModelRuntime) VerifyModel(_ context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error) {
	f.mu.Lock()
	f.lastControl = req
	f.mu.Unlock()
	return f.mutateStatus(modelID, "verified")
}

func (f *fakeModelRuntime) EnableModel(_ context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error) {
	f.mu.Lock()
	f.lastControl = req
	f.mu.Unlock()
	return f.mutateStatus(modelID, "verified")
}

func (f *fakeModelRuntime) DisableModel(_ context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error) {
	f.mu.Lock()
	f.lastControl = req
	f.mu.Unlock()
	return f.mutateStatus(modelID, "disabled")
}

func (f *fakeModelRuntime) ArchiveModel(_ context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error) {
	f.mu.Lock()
	f.lastControl = req
	f.mu.Unlock()
	return f.mutateStatus(modelID, "archived")
}

func (f *fakeModelRuntime) RemoveModel(_ context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeRemoveResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastControl = req
	delete(f.models, modelID)
	delete(f.loaded, modelID)
	return ModelRuntimeRemoveResult{ModelID: modelID, RemovedPath: "/tmp/removed/" + modelID}, nil
}

func (f *fakeModelRuntime) LoadModel(_ context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeLoadResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loadCalls++
	f.lastMeta = req.Meta
	f.lastControl = req
	if _, ok := f.models[modelID]; !ok {
		return ModelRuntimeLoadResult{}, &modelRuntimeError{status: http.StatusNotFound, code: "MODEL_NOT_FOUND", message: "model not found"}
	}
	f.loaded[modelID] = true
	return ModelRuntimeLoadResult{
		ModelID:    modelID,
		Backend:    "fake",
		Status:     "loaded",
		Loaded:     true,
		LoadedAtMs: time.Now().UnixMilli(),
	}, nil
}

func (f *fakeModelRuntime) UnloadModel(_ context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeLoadResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unloadCalls++
	f.lastMeta = req.Meta
	f.lastControl = req
	if _, ok := f.models[modelID]; !ok {
		return ModelRuntimeLoadResult{}, &modelRuntimeError{status: http.StatusNotFound, code: "MODEL_NOT_FOUND", message: "model not found"}
	}
	delete(f.loaded, modelID)
	return ModelRuntimeLoadResult{
		ModelID: modelID,
		Backend: "fake",
		Status:  "available",
		Loaded:  false,
	}, nil
}

func (f *fakeModelRuntime) Chat(_ context.Context, req ModelRuntimeChatRequest) (ModelRuntimeChatResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chatCalls++
	f.lastMeta = req.Meta
	f.lastChat = req
	if f.chatErr != nil {
		return ModelRuntimeChatResult{}, f.chatErr
	}
	if _, ok := f.models[req.ModelID]; !ok {
		return ModelRuntimeChatResult{}, &modelRuntimeError{status: http.StatusNotFound, code: "MODEL_NOT_FOUND", message: "model not found"}
	}
	content := "ok"
	if len(req.Messages) > 0 {
		content = "echo: " + strings.TrimSpace(req.Messages[len(req.Messages)-1].Content)
	}
	if strings.TrimSpace(req.Prompt) != "" {
		content = "echo: " + strings.TrimSpace(req.Prompt)
	}
	return ModelRuntimeChatResult{
		Content:      content,
		FinishReason: "stop",
		Usage: ModelRuntimeUsage{
			PromptTokens:     4,
			CompletionTokens: 6,
			TotalTokens:      10,
		},
		DurationMs: 12,
		Backend:    "fake",
		ModelID:    req.ModelID,
		AuditID:    "audit-test",
	}, nil
}

func (f *fakeModelRuntime) Compatibility(_ context.Context, modelID string, _ ModelRuntimeRequestMeta) (ModelRuntimeCompatibility, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	model, ok := f.models[modelID]
	if !ok {
		return ModelRuntimeCompatibility{}, &modelRuntimeError{status: http.StatusNotFound, code: "MODEL_NOT_FOUND", message: "model not found"}
	}
	return ModelRuntimeCompatibility{
		ModelID:            model.ID,
		Backend:            model.Backend,
		Status:             model.Status,
		BackendConfigured:  true,
		BackendHealthy:     true,
		SupportedByBackend: true,
		CanGenerate:        model.Status != "disabled" && model.Status != "archived",
		Loaded:             f.loaded[modelID],
	}, nil
}

func (f *fakeModelRuntime) Health(_ context.Context, req ModelRuntimeRequestMeta) (ModelRuntimeHealth, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.healthCalls++
	f.lastMeta = req
	if f.healthErr != nil {
		return ModelRuntimeHealth{}, f.healthErr
	}
	return ModelRuntimeHealth{
		OK:      true,
		Status:  "ready",
		Backend: "fake",
		Details: map[string]any{"loadedModels": len(f.loaded)},
	}, nil
}

func (f *fakeModelRuntime) QueueStatus(_ context.Context, req ModelRuntimeRequestMeta) (ModelRuntimeQueueStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queueCalls++
	f.lastMeta = req
	active := map[string]string{}
	for id, isLoaded := range f.loaded {
		if isLoaded {
			active["fake"] = id
			break
		}
	}
	return ModelRuntimeQueueStatus{
		Depth:       0,
		Active:      active,
		Pending:     []string{},
		Scheduler:   "single_loaded_per_backend",
		PolicyState: "gateway_governed",
	}, nil
}

func (f *fakeModelRuntime) LoadedStatus(_ context.Context, req ModelRuntimeRequestMeta) (ModelRuntimeLoadedStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loadedCalls++
	f.lastMeta = req
	models := make([]ModelRuntimeLoadedModel, 0, len(f.loaded))
	for id, isLoaded := range f.loaded {
		if !isLoaded {
			continue
		}
		models = append(models, ModelRuntimeLoadedModel{
			ModelID: id,
			Backend: "fake",
			Status:  "loaded",
		})
	}
	return ModelRuntimeLoadedStatus{Count: len(models), Models: models}, nil
}

func (f *fakeModelRuntime) Usage(_ context.Context, _ ModelRuntimeRequestMeta) (ModelRuntimeUsageSummary, error) {
	return ModelRuntimeUsageSummary{
		Registered: len(f.models),
		Available:  len(f.models),
		Loaded:     len(f.loaded),
		Backends:   map[string]map[string]any{"fake": {"healthy": true}},
	}, nil
}

func (f *fakeModelRuntime) Backends(_ context.Context, _ ModelRuntimeRequestMeta) ([]ModelRuntimeBackendStatus, error) {
	return []ModelRuntimeBackendStatus{{Kind: "fake", Name: "fake", Healthy: true}}, nil
}

func (f *fakeModelRuntime) mutateStatus(modelID, status string) (ModelRuntimeModel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	model, ok := f.models[modelID]
	if !ok {
		return ModelRuntimeModel{}, &modelRuntimeError{status: http.StatusNotFound, code: "MODEL_NOT_FOUND", message: "model not found"}
	}
	model.Status = status
	f.models[modelID] = model
	return model, nil
}

func TestModelRuntimeForgeListEndpoint(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	srv.modelRuntime = fake

	req := httptest.NewRequest(http.MethodGet, "/forge/models?workspaceId=ws-a&traceId=trace-a", nil)
	req.Header.Set("X-Correlation-ID", "corr-a")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	var payload struct {
		Models        []ModelRuntimeModel `json:"models"`
		CorrelationID string              `json:"correlationId"`
		TraceID       string              `json:"traceId"`
		WorkspaceID   string              `json:"workspaceId"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode list response: %v body=%s", err, rr.Body.String())
	}
	if len(payload.Models) != 1 || payload.Models[0].ID != "mistral-7b-instruct" {
		t.Fatalf("unexpected models payload: %#v", payload.Models)
	}
	if payload.CorrelationID != "corr-a" || payload.TraceID != "trace-a" || payload.WorkspaceID != "ws-a" {
		t.Fatalf("unexpected request meta: correlation=%q trace=%q workspace=%q", payload.CorrelationID, payload.TraceID, payload.WorkspaceID)
	}
	if fake.listCalls != 1 {
		t.Fatalf("list calls=%d want 1", fake.listCalls)
	}

	v1Req := httptest.NewRequest(http.MethodGet, "/v1/models?workspaceId=ws-a&traceId=trace-v1-list", nil)
	v1Req.Header.Set("X-Correlation-ID", "corr-v1-list")
	v1RR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(v1RR, v1Req)
	if v1RR.Code != http.StatusOK {
		t.Fatalf("v1 models status=%d body=%s", v1RR.Code, strings.TrimSpace(v1RR.Body.String()))
	}
	var v1Payload struct {
		Object string `json:"object"`
		Data   []struct {
			ID string `json:"id"`
		} `json:"data"`
		CorrelationID string `json:"correlationId"`
		TraceID       string `json:"traceId"`
		WorkspaceID   string `json:"workspaceId"`
	}
	if err := json.Unmarshal(v1RR.Body.Bytes(), &v1Payload); err != nil {
		t.Fatalf("decode v1 models response: %v body=%s", err, v1RR.Body.String())
	}
	if v1Payload.Object != "list" || len(v1Payload.Data) != 1 || v1Payload.Data[0].ID != "mistral-7b-instruct" {
		t.Fatalf("unexpected v1 models payload: %#v", v1Payload)
	}
	if v1Payload.CorrelationID != "corr-v1-list" || v1Payload.TraceID != "trace-v1-list" || v1Payload.WorkspaceID != "ws-a" {
		t.Fatalf("unexpected v1 request meta: correlation=%q trace=%q workspace=%q", v1Payload.CorrelationID, v1Payload.TraceID, v1Payload.WorkspaceID)
	}
}

func TestModelRuntimeForgeChatAndOpenAICompat(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	fake.models["llama3.2:latest"] = ModelRuntimeModel{
		ID:           "llama3.2:latest",
		DisplayName:  "llama3.2:latest",
		Backend:      "ollama_compat",
		Format:       "gguf",
		Status:       "available",
		Capabilities: []string{"chat", "completion"},
	}
	srv.modelRuntime = fake

	forgeRaw := []byte(`{"messages":[{"role":"user","content":"hello forge"}],"correlationId":"corr-forge-chat","workspaceId":"ws-forge"}`)
	forgeReq := httptest.NewRequest(http.MethodPost, "/forge/models/mistral-7b-instruct/chat?traceId=trace-forge", bytes.NewReader(forgeRaw))
	forgeRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(forgeRR, forgeReq)

	if forgeRR.Code != http.StatusOK {
		t.Fatalf("forge chat status=%d body=%s", forgeRR.Code, strings.TrimSpace(forgeRR.Body.String()))
	}
	var forgePayload struct {
		Result struct {
			Content string `json:"content"`
			ModelID string `json:"modelId"`
		} `json:"result"`
		CorrelationID string `json:"correlationId"`
		TraceID       string `json:"traceId"`
		WorkspaceID   string `json:"workspaceId"`
	}
	if err := json.Unmarshal(forgeRR.Body.Bytes(), &forgePayload); err != nil {
		t.Fatalf("decode forge chat response: %v body=%s", err, forgeRR.Body.String())
	}
	if forgePayload.Result.ModelID != "mistral-7b-instruct" {
		t.Fatalf("forge model=%q want mistral-7b-instruct", forgePayload.Result.ModelID)
	}
	if forgePayload.CorrelationID != "corr-forge-chat" || forgePayload.TraceID != "trace-forge" || forgePayload.WorkspaceID != "ws-forge" {
		t.Fatalf("forge chat meta mismatch: %#v", forgePayload)
	}

	encodedRaw := []byte(`{"messages":[{"role":"user","content":"hello encoded"}],"workspaceId":"ws-forge"}`)
	encodedReq := httptest.NewRequest(http.MethodPost, "/forge/models/llama3.2%3Alatest/chat", bytes.NewReader(encodedRaw))
	encodedRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(encodedRR, encodedReq)
	if encodedRR.Code != http.StatusOK {
		t.Fatalf("encoded forge chat status=%d body=%s", encodedRR.Code, strings.TrimSpace(encodedRR.Body.String()))
	}
	if fake.lastChat.ModelID != "llama3.2:latest" {
		t.Fatalf("expected decoded model id, got %q", fake.lastChat.ModelID)
	}

	oaRaw := []byte(`{"model":"mistral-7b-instruct","messages":[{"role":"user","content":"hello openai"}],"correlationId":"corr-v1-chat","workspaceId":"ws-v1"}`)
	oaReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?traceId=trace-v1", bytes.NewReader(oaRaw))
	oaRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(oaRR, oaReq)

	if oaRR.Code != http.StatusOK {
		t.Fatalf("openai chat status=%d body=%s", oaRR.Code, strings.TrimSpace(oaRR.Body.String()))
	}
	var oaPayload struct {
		Object  string `json:"object"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		CorrelationID string `json:"correlationId"`
		TraceID       string `json:"traceId"`
		WorkspaceID   string `json:"workspaceId"`
	}
	if err := json.Unmarshal(oaRR.Body.Bytes(), &oaPayload); err != nil {
		t.Fatalf("decode openai chat response: %v body=%s", err, oaRR.Body.String())
	}
	if oaPayload.Object != "chat.completion" || oaPayload.Model != "mistral-7b-instruct" {
		t.Fatalf("unexpected openai payload: %#v", oaPayload)
	}
	if len(oaPayload.Choices) != 1 || oaPayload.Choices[0].Message.Role != "assistant" {
		t.Fatalf("unexpected choices payload: %#v", oaPayload.Choices)
	}
	if oaPayload.CorrelationID != "corr-v1-chat" || oaPayload.TraceID != "trace-v1" || oaPayload.WorkspaceID != "ws-v1" {
		t.Fatalf("openai chat meta mismatch: %#v", oaPayload)
	}
}

func TestModelRuntimeInvalidModelReturnsDeterministicError(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	srv.modelRuntime = newFakeModelRuntime()

	req := httptest.NewRequest(http.MethodGet, "/forge/models/no-such-model?correlationId=corr-missing", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		CorrelationID string `json:"correlationId"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode invalid-model response: %v body=%s", err, rr.Body.String())
	}
	if payload.Error.Code != "MODEL_NOT_FOUND" {
		t.Fatalf("error code=%q want MODEL_NOT_FOUND", payload.Error.Code)
	}
	if payload.CorrelationID != "corr-missing" {
		t.Fatalf("correlation=%q want corr-missing", payload.CorrelationID)
	}
}

func TestModelRuntimeHealthEndpoint(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	srv.modelRuntime = fake

	req := httptest.NewRequest(http.MethodGet, "/forge/model-runtime/health?correlationId=corr-health", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	var payload struct {
		Health struct {
			OK      bool   `json:"ok"`
			Status  string `json:"status"`
			Backend string `json:"backend"`
		} `json:"health"`
		CorrelationID string `json:"correlationId"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode health response: %v body=%s", err, rr.Body.String())
	}
	if !payload.Health.OK || payload.Health.Status != "ready" || payload.Health.Backend != "fake" {
		t.Fatalf("unexpected health payload: %#v", payload.Health)
	}
	if payload.CorrelationID != "corr-health" {
		t.Fatalf("correlation=%q want corr-health", payload.CorrelationID)
	}
	if fake.healthCalls != 1 {
		t.Fatalf("health calls=%d want 1", fake.healthCalls)
	}
}

func TestModelRuntimeQueueAndLoadedEndpoints(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	fake.loaded["mistral-7b-instruct"] = true
	srv.modelRuntime = fake

	queueReq := httptest.NewRequest(http.MethodGet, "/forge/model-runtime/queue?correlationId=corr-queue", nil)
	queueRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(queueRR, queueReq)
	if queueRR.Code != http.StatusOK {
		t.Fatalf("queue status=%d body=%s", queueRR.Code, strings.TrimSpace(queueRR.Body.String()))
	}
	var queuePayload struct {
		Queue struct {
			Depth  int               `json:"depth"`
			Active map[string]string `json:"active"`
		} `json:"queue"`
	}
	if err := json.Unmarshal(queueRR.Body.Bytes(), &queuePayload); err != nil {
		t.Fatalf("decode queue response: %v body=%s", err, queueRR.Body.String())
	}
	if queuePayload.Queue.Depth != 0 || queuePayload.Queue.Active["fake"] != "mistral-7b-instruct" {
		t.Fatalf("unexpected queue payload: %#v", queuePayload.Queue)
	}

	loadedReq := httptest.NewRequest(http.MethodGet, "/forge/model-runtime/loaded?correlationId=corr-loaded", nil)
	loadedRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(loadedRR, loadedReq)
	if loadedRR.Code != http.StatusOK {
		t.Fatalf("loaded status=%d body=%s", loadedRR.Code, strings.TrimSpace(loadedRR.Body.String()))
	}
	var loadedPayload struct {
		Loaded struct {
			Count  int `json:"count"`
			Models []struct {
				ModelID string `json:"modelId"`
			} `json:"models"`
		} `json:"loaded"`
	}
	if err := json.Unmarshal(loadedRR.Body.Bytes(), &loadedPayload); err != nil {
		t.Fatalf("decode loaded response: %v body=%s", err, loadedRR.Body.String())
	}
	if loadedPayload.Loaded.Count != 1 || len(loadedPayload.Loaded.Models) != 1 || loadedPayload.Loaded.Models[0].ModelID != "mistral-7b-instruct" {
		t.Fatalf("unexpected loaded payload: %#v", loadedPayload.Loaded)
	}
	if fake.queueCalls != 1 || fake.loadedCalls != 1 {
		t.Fatalf("queue/loaded calls mismatch queue=%d loaded=%d", fake.queueCalls, fake.loadedCalls)
	}
}

func TestModelRuntimeChatValidationAndStructuredMapping(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	srv.modelRuntime = fake

	invalidRaw := []byte(`{"messages":[{"role":"","content":"x"}]}`)
	invalidReq := httptest.NewRequest(http.MethodPost, "/forge/models/mistral-7b-instruct/chat", bytes.NewReader(invalidRaw))
	invalidRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(invalidRR, invalidReq)
	if invalidRR.Code != http.StatusBadRequest {
		t.Fatalf("invalid chat status=%d body=%s", invalidRR.Code, strings.TrimSpace(invalidRR.Body.String()))
	}
	var invalidPayload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(invalidRR.Body.Bytes(), &invalidPayload); err != nil {
		t.Fatalf("decode invalid chat response: %v body=%s", err, invalidRR.Body.String())
	}
	if invalidPayload.Error.Code != "MESSAGE_ROLE_REQUIRED" {
		t.Fatalf("unexpected invalid chat code=%q", invalidPayload.Error.Code)
	}

	fake.chatErr = errors.New("scheduler busy: backend queue is full")
	busyRaw := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)
	busyReq := httptest.NewRequest(http.MethodPost, "/forge/models/mistral-7b-instruct/chat", bytes.NewReader(busyRaw))
	busyRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(busyRR, busyReq)
	if busyRR.Code != http.StatusTooManyRequests {
		t.Fatalf("scheduler error status=%d body=%s", busyRR.Code, strings.TrimSpace(busyRR.Body.String()))
	}
	var busyPayload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(busyRR.Body.Bytes(), &busyPayload); err != nil {
		t.Fatalf("decode scheduler error response: %v body=%s", err, busyRR.Body.String())
	}
	if busyPayload.Error.Code != "MODEL_SCHEDULER_BUSY" {
		t.Fatalf("unexpected scheduler error code=%q", busyPayload.Error.Code)
	}
}

func TestOpenAICompatStreamingUnsupported(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	srv.modelRuntime = newFakeModelRuntime()

	raw := []byte(`{"model":"mistral-7b-instruct","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode stream unsupported response: %v body=%s", err, rr.Body.String())
	}
	if payload.Error.Code != "STREAM_UNSUPPORTED" {
		t.Fatalf("error code=%q want STREAM_UNSUPPORTED", payload.Error.Code)
	}
}

func TestForgeModelRuntimeStreamingUnsupported(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	srv.modelRuntime = newFakeModelRuntime()

	raw := []byte(`{"messages":[{"role":"user","content":"hello"}],"stream":true}`)
	req := httptest.NewRequest(http.MethodPost, "/forge/models/mistral-7b-instruct/chat", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode forge stream unsupported response: %v body=%s", err, rr.Body.String())
	}
	if payload.Error.Code != "STREAM_UNSUPPORTED" {
		t.Fatalf("error code=%q want STREAM_UNSUPPORTED", payload.Error.Code)
	}
}

func TestModelRuntimeLoadUnloadDeterministic(t *testing.T) {
	t.Parallel()

	srv, _ := newModelRuntimeHarness(t)
	fake := newFakeModelRuntime()
	srv.modelRuntime = fake

	loadBody := governanceBody(map[string]any{"correlationId": "corr-load-1"})
	loadApprovalID := requestAndApproveModelGovernance(t, srv, "/forge/models/mistral-7b-instruct/load", loadBody)
	loadBody["approvalId"] = fmt.Sprintf("%d", loadApprovalID)
	loadRR1 := postModelRuntimeJSON(t, srv, "/forge/models/mistral-7b-instruct/load", loadBody)
	if loadRR1.Code != http.StatusOK {
		t.Fatalf("load1 status=%d body=%s", loadRR1.Code, strings.TrimSpace(loadRR1.Body.String()))
	}

	loadBody["correlationId"] = "corr-load-2"
	loadRR2 := postModelRuntimeJSON(t, srv, "/forge/models/mistral-7b-instruct/load", loadBody)
	if loadRR2.Code != http.StatusOK {
		t.Fatalf("load2 status=%d body=%s", loadRR2.Code, strings.TrimSpace(loadRR2.Body.String()))
	}

	unloadBody := governanceBody(map[string]any{"correlationId": "corr-unload-1"})
	unloadApprovalID := requestAndApproveModelGovernance(t, srv, "/forge/models/mistral-7b-instruct/unload", unloadBody)
	unloadBody["approvalId"] = fmt.Sprintf("%d", unloadApprovalID)
	unloadRR1 := postModelRuntimeJSON(t, srv, "/forge/models/mistral-7b-instruct/unload", unloadBody)
	if unloadRR1.Code != http.StatusOK {
		t.Fatalf("unload1 status=%d body=%s", unloadRR1.Code, strings.TrimSpace(unloadRR1.Body.String()))
	}

	unloadBody["correlationId"] = "corr-unload-2"
	unloadRR2 := postModelRuntimeJSON(t, srv, "/forge/models/mistral-7b-instruct/unload", unloadBody)
	if unloadRR2.Code != http.StatusOK {
		t.Fatalf("unload2 status=%d body=%s", unloadRR2.Code, strings.TrimSpace(unloadRR2.Body.String()))
	}

	var loadPayload1, loadPayload2, unloadPayload1, unloadPayload2 struct {
		Result struct {
			Status string `json:"status"`
			Loaded bool   `json:"loaded"`
		} `json:"result"`
	}
	_ = json.Unmarshal(loadRR1.Body.Bytes(), &loadPayload1)
	_ = json.Unmarshal(loadRR2.Body.Bytes(), &loadPayload2)
	_ = json.Unmarshal(unloadRR1.Body.Bytes(), &unloadPayload1)
	_ = json.Unmarshal(unloadRR2.Body.Bytes(), &unloadPayload2)

	if !loadPayload1.Result.Loaded || !loadPayload2.Result.Loaded {
		t.Fatalf("load results must stay loaded=true: first=%#v second=%#v", loadPayload1.Result, loadPayload2.Result)
	}
	if unloadPayload1.Result.Loaded || unloadPayload2.Result.Loaded {
		t.Fatalf("unload results must stay loaded=false: first=%#v second=%#v", unloadPayload1.Result, unloadPayload2.Result)
	}
	if fake.loadCalls != 2 || fake.unloadCalls != 2 {
		t.Fatalf("load/unload calls mismatch load=%d unload=%d", fake.loadCalls, fake.unloadCalls)
	}
}

func TestOpenAICompatRoutesDisabledByDefault(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{
		DataDir:               dataDir,
		WorkspaceDir:          workspaceDir,
		EnableOpenAICompatAPI: false,
	})
	t.Cleanup(func() { srv.ShutdownWatch() })
	srv.modelRuntime = newFakeModelRuntime()

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func TestOpenAICompatRoutesAreAvailableWhenAutoEnabledViaCompatFlag(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{
		DataDir:                   dataDir,
		WorkspaceDir:              workspaceDir,
		EnableOpenAICompatAPI:     true,
		ModelOpenAICompatEndpoint: "http://127.0.0.1:11434",
	})
	t.Cleanup(func() { srv.ShutdownWatch() })

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("openai compat /v1/models status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func newModelRuntimeHarness(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{
		DataDir:               dataDir,
		WorkspaceDir:          workspaceDir,
		EnableOpenAICompatAPI: true,
	})
	t.Cleanup(func() { srv.ShutdownWatch() })
	return srv, st
}

func firstNonEmptyTest(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
