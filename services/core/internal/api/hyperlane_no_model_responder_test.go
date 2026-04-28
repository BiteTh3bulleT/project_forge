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
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestHyperlaneNoModelStatusQuery(t *testing.T) {
	srv, st, fake := newHyperlaneNoModelHarness(t)
	before := canonicalCounts(t, st)
	resp := postHyperlaneChat(t, srv, "what is forge core status?")

	assertHyperlaneNoModelResponse(t, resp, "status_query", "structured.status")
	if !strings.Contains(resp.AssistantMessage.Content, "FORGE status") {
		t.Fatalf("unexpected status response: %s", resp.AssistantMessage.Content)
	}
	assertNoModelRuntimeCalls(t, fake)
	assertGatewayInvocationCount(t, st, 0)
	assertCanonicalCounts(t, st, before)
}

func TestHyperlaneNoModelDiagnosticsQuery(t *testing.T) {
	srv, st, fake := newHyperlaneNoModelHarness(t)
	resp := postHyperlaneChat(t, srv, "show diagnostics summary")

	assertHyperlaneNoModelResponse(t, resp, "diagnostics_query", "structured.diagnostics")
	if !strings.Contains(resp.AssistantMessage.Content, "Diagnostics summary") {
		t.Fatalf("unexpected diagnostics response: %s", resp.AssistantMessage.Content)
	}
	assertNoModelRuntimeCalls(t, fake)
	assertGatewayInvocationCount(t, st, 0)
}

func TestHyperlaneNoModelChatMemoryInspectionQuery(t *testing.T) {
	srv, st, fake := newHyperlaneNoModelHarness(t)
	before := canonicalCounts(t, st)
	if err := upsertSetting(context.Background(), st.DB, remoteCrossChatContextKey, "true"); err != nil {
		t.Fatalf("set remote cross chat: %v", err)
	}
	thread, err := srv.chat.CreateThread(context.Background(), "memory status seed", nil)
	if err != nil {
		t.Fatalf("create seed thread: %v", err)
	}
	if _, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "seed memory question", nil); err != nil {
		t.Fatalf("append seed message: %v", err)
	}

	resp := postHyperlaneChat(t, srv, "Do we have cross chat context and memory?")

	assertHyperlaneNoModelResponse(t, resp, "chat_memory_inspection", "structured.chat_memory")
	if !strings.Contains(resp.AssistantMessage.Content, "local chat persistence is active") || !strings.Contains(resp.AssistantMessage.Content, "Remote cross-chat context is true") {
		t.Fatalf("unexpected chat memory response: %s", resp.AssistantMessage.Content)
	}
	assertNoModelRuntimeCalls(t, fake)
	assertGatewayInvocationCount(t, st, 0)
	assertCanonicalCounts(t, st, before)
}

func TestHyperlaneNoModelChatHistoryLookupByTimestamp(t *testing.T) {
	srv, st, fake := newHyperlaneNoModelHarness(t)
	before := canonicalCounts(t, st)
	target := time.Date(2026, time.April, 16, 15, 54, 49, 0, time.Local)
	thread, err := srv.chat.CreateThread(context.Background(), "E2E Tool Audit", nil)
	if err != nil {
		t.Fatalf("create seed thread: %v", err)
	}
	mustExecHyperlane(t, st, `INSERT INTO chat_messages(thread_id, role, content, created_at, metadata_json) VALUES(?,?,?,?,?)`,
		thread.ID, "user", "Create a directory in Downloads called PeanutButterJellyTime. Inside that folder create an svg file of a flower.", target.UnixMilli(), `{}`)

	resp := postHyperlaneChat(t, srv, "On 4/16/2026 at 3:54:49 PM, what did I ask you to do?")

	assertHyperlaneNoModelResponse(t, resp, "chat_history_lookup", "structured.chat_history_lookup")
	if !strings.Contains(resp.AssistantMessage.Content, "PeanutButterJellyTime") || !strings.Contains(resp.AssistantMessage.Content, "E2E Tool Audit") {
		t.Fatalf("unexpected chat history lookup response: %s", resp.AssistantMessage.Content)
	}
	assertNoModelRuntimeCalls(t, fake)
	assertGatewayInvocationCount(t, st, 0)
	assertCanonicalCounts(t, st, before)
}

func TestHyperlaneNoModelModelRuntimeStatusQuery(t *testing.T) {
	srv, st, fake := newHyperlaneNoModelHarness(t)
	mustExecHyperlane(t, st, `INSERT INTO model_manifests(id, schema_version, display_name, family, format, backend, model_path, discovered_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		"m1", "forge.model/v1", "M1", "test", "gguf", "fake", "m1.gguf", time.Now().UnixMilli(), time.Now().UnixMilli())
	mustExecHyperlane(t, st, `INSERT INTO model_registry_status(model_id, backend, status, updated_at, metadata_json) VALUES(?,?,?,?,?)`, "m1", "fake", "available", time.Now().UnixMilli(), `{}`)
	mustExecHyperlane(t, st, `INSERT INTO model_runtime_loads(model_id, backend, status, loaded_at, metadata_json) VALUES(?,?,?,?,?)`, "m1", "fake", "loaded", time.Now().UnixMilli(), `{}`)

	resp := postHyperlaneChat(t, srv, "modelruntime status and loaded models")

	assertHyperlaneNoModelResponse(t, resp, "modelruntime_status", "structured.modelruntime_status")
	if !strings.Contains(resp.AssistantMessage.Content, "latest loaded model=m1") {
		t.Fatalf("unexpected modelruntime response: %s", resp.AssistantMessage.Content)
	}
	assertNoModelRuntimeCalls(t, fake)
	assertGatewayInvocationCount(t, st, 0)
}

func TestHyperlaneNoModelRestoreInspectionQuery(t *testing.T) {
	srv, st, fake := newHyperlaneNoModelHarness(t)
	before := canonicalCounts(t, st)
	now := time.Now().UnixMilli()
	mustExecHyperlane(t, st, `INSERT INTO context_packet_snapshots(id,query,workspace_id,lane_id,snapshot_kind,restore_scores_json,resume_hints_json,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		"restore-current", "restore score", srv.cfg.WorkspaceDir, "control.semantic", "restore", `{"coverage":0.91}`, `{"requires_fresh_compile":true,"reason":"below threshold"}`, now)

	resp := postHyperlaneChat(t, srv, "latest restore score")

	assertHyperlaneNoModelResponse(t, resp, "restore_inspection", "structured.restore_inspector")
	if !strings.Contains(resp.AssistantMessage.Content, "restore-current") || !strings.Contains(resp.AssistantMessage.Content, "requires fresh compile=true") {
		t.Fatalf("unexpected restore response: %s", resp.AssistantMessage.Content)
	}
	assertNoModelRuntimeCalls(t, fake)
	assertGatewayInvocationCount(t, st, 0)
	assertCanonicalCounts(t, st, before)
}

func TestHyperlaneNoModelDreamReportQuery(t *testing.T) {
	srv, st, fake := newHyperlaneNoModelHarness(t)
	before := canonicalCounts(t, st)
	seedDreamReportNoModel(t, st, srv.cfg.WorkspaceDir, "dream-current")

	resp := postHyperlaneChat(t, srv, "latest Dream report")

	assertHyperlaneNoModelResponse(t, resp, "dream_report_inspection", "structured.dream_reports")
	if !strings.Contains(resp.AssistantMessage.Content, "dream-current") || !strings.Contains(resp.AssistantMessage.Content, "dry-run=true") {
		t.Fatalf("unexpected dream response: %s", resp.AssistantMessage.Content)
	}
	assertNoModelRuntimeCalls(t, fake)
	assertGatewayInvocationCount(t, st, 0)
	assertCanonicalCounts(t, st, before)
}

func TestHyperlaneUnknownQueryFallsThroughToModelPath(t *testing.T) {
	srv, _, fake := newHyperlaneNoModelHarness(t)
	resp := postHyperlaneChat(t, srv, "explain how this repository is structured")

	if resp.AssistantMessage == nil {
		t.Fatalf("expected assistant response")
	}
	if _, ok := resp.AssistantMessage.Metadata["hyperlane_intent_type"]; ok {
		t.Fatalf("unknown query should not use hyperlane metadata: %#v", resp.AssistantMessage.Metadata)
	}
	if fake.chatCalls == 0 {
		t.Fatalf("expected model runtime chat call for unknown query")
	}
}

func TestHyperlaneNoModelDoesNotLeakWrongWorkspaceDataAndHandlesEmptyState(t *testing.T) {
	srv, st, fake := newHyperlaneNoModelHarness(t)
	now := time.Now().UnixMilli()
	mustExecHyperlane(t, st, `INSERT INTO context_packet_snapshots(id,query,workspace_id,lane_id,snapshot_kind,restore_scores_json,resume_hints_json,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		"restore-other-workspace", "restore score", "other-workspace", "control.semantic", "restore", `{"coverage":1}`, `{}`, now)
	seedDreamReportNoModel(t, st, "other-workspace", "dream-other-workspace")

	restoreResp := postHyperlaneChat(t, srv, "latest restore inspection")
	assertHyperlaneNoModelResponse(t, restoreResp, "restore_inspection", "structured.restore_inspector")
	if strings.Contains(restoreResp.AssistantMessage.Content, "restore-other-workspace") {
		t.Fatalf("restore response leaked wrong workspace data: %s", restoreResp.AssistantMessage.Content)
	}
	if !strings.Contains(restoreResp.AssistantMessage.Content, "no restore snapshots") {
		t.Fatalf("expected graceful empty restore response, got: %s", restoreResp.AssistantMessage.Content)
	}

	dreamResp := postHyperlaneChat(t, srv, "dream reports")
	assertHyperlaneNoModelResponse(t, dreamResp, "dream_report_inspection", "structured.dream_reports")
	if strings.Contains(dreamResp.AssistantMessage.Content, "dream-other-workspace") {
		t.Fatalf("dream response leaked wrong workspace data: %s", dreamResp.AssistantMessage.Content)
	}
	if !strings.Contains(dreamResp.AssistantMessage.Content, "no Dream reports") {
		t.Fatalf("expected graceful empty dream response, got: %s", dreamResp.AssistantMessage.Content)
	}
	assertNoModelRuntimeCalls(t, fake)
	assertGatewayInvocationCount(t, st, 0)
}

func TestHyperlaneNoModelAssistantStreamReturnsStructuredResponse(t *testing.T) {
	srv, st, fake := newHyperlaneNoModelHarness(t)
	before := canonicalCounts(t, st)
	thread, err := srv.chat.CreateThread(context.Background(), "hyperlane no-model stream", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	um, err := srv.chat.AppendMessage(context.Background(), thread.ID, "user", "show diagnostics summary", map[string]any{"source": "operator"})
	if err != nil {
		t.Fatalf("append user message: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/threads/"+strconv.FormatInt(thread.ID, 10)+"/assistant-stream?userMessageId="+strconv.FormatInt(um.ID, 10), nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("stream status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if !strings.Contains(rr.Body.String(), "assistantMessage") || !strings.Contains(rr.Body.String(), "hyperlane_no_model_structured_response") {
		t.Fatalf("expected structured no-model SSE response, body=%s", rr.Body.String())
	}

	detail, err := srv.chat.GetThread(context.Background(), thread.ID)
	if err != nil {
		t.Fatalf("load thread: %v", err)
	}
	var assistant *chatPostAssistantMessage
	for i := len(detail.Messages) - 1; i >= 0; i-- {
		msg := detail.Messages[i]
		if msg.Role == "assistant" {
			assistant = &chatPostAssistantMessage{Content: msg.Content, Metadata: msg.Metadata}
			break
		}
	}
	if assistant == nil {
		t.Fatalf("expected persisted assistant message")
	}
	assertHyperlaneNoModelResponse(t, chatPostResponse{AssistantMessage: assistant}, "diagnostics_query", "structured.diagnostics")
	assertNoModelRuntimeCalls(t, fake)
	assertGatewayInvocationCount(t, st, 0)
	assertCanonicalCounts(t, st, before)
}

func newHyperlaneNoModelHarness(t *testing.T) (*Server, *store.Store, *fakeModelRuntime) {
	t.Helper()
	srv, st := newBackupAuditHarness(t)
	fake := newFakeModelRuntime()
	srv.modelRuntime = fake
	return srv, st, fake
}

func postHyperlaneChat(t *testing.T, srv *Server, content string) chatPostResponse {
	t.Helper()
	thread, err := srv.chat.CreateThread(context.Background(), "hyperlane no-model", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	raw, _ := json.Marshal(map[string]any{
		"content":          content,
		"requestAssistant": true,
		"syncAssistant":    true,
	})
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
	return payload
}

func assertHyperlaneNoModelResponse(t *testing.T, resp chatPostResponse, intentType, route string) {
	t.Helper()
	meta := resp.AssistantMessage.Metadata
	if got := strings.TrimSpace(asString(meta["hyperlane_intent_type"])); got != intentType {
		t.Fatalf("hyperlane_intent_type=%q want %q metadata=%#v", got, intentType, meta)
	}
	if got := strings.TrimSpace(asString(meta["hyperlane_route"])); got != route {
		t.Fatalf("hyperlane_route=%q want %q metadata=%#v", got, route, meta)
	}
	if got := strings.TrimSpace(asString(meta["hyperlane_matched_rule"])); got == "" {
		t.Fatalf("expected hyperlane_matched_rule metadata=%#v", meta)
	}
	if got := strings.TrimSpace(asString(meta["hyperlane_parser_version"])); got == "" {
		t.Fatalf("expected hyperlane_parser_version metadata=%#v", meta)
	}
	if htrace, ok := meta["hyperlane_trace"].(map[string]any); !ok {
		t.Fatalf("expected hyperlane_trace metadata=%#v", meta)
	} else if got := strings.TrimSpace(asString(htrace["matched_rule"])); got == "" {
		t.Fatalf("expected hyperlane_trace matched_rule metadata=%#v", meta)
	}
	if ok, _ := meta["modelruntime_avoided"].(bool); !ok {
		t.Fatalf("expected modelruntime_avoided=true metadata=%#v", meta)
	}
	if ok, _ := meta["context_compile_avoided"].(bool); !ok {
		t.Fatalf("expected context_compile_avoided=true metadata=%#v", meta)
	}
	if ok, _ := meta["gateway_avoided"].(bool); !ok {
		t.Fatalf("expected gateway_avoided=true metadata=%#v", meta)
	}
	if _, ok := meta["hyperlane_confidence"].(float64); !ok {
		t.Fatalf("expected hyperlane_confidence metadata=%#v", meta)
	}
	if _, ok := meta["latency_ms"].(float64); !ok {
		t.Fatalf("expected latency_ms metadata=%#v", meta)
	}
	trace, ok := meta["chatLatencyTrace"].(map[string]any)
	if !ok {
		t.Fatalf("expected chatLatencyTrace metadata=%#v", meta)
	}
	if got := strings.TrimSpace(asString(trace["hyperlane_intent_type"])); got != intentType {
		t.Fatalf("trace hyperlane_intent_type=%q want %q trace=%#v", got, intentType, trace)
	}
	if got := strings.TrimSpace(asString(trace["hyperlane_route"])); got != route {
		t.Fatalf("trace hyperlane_route=%q want %q trace=%#v", got, route, trace)
	}
	if ok, _ := trace["modelruntime_avoided"].(bool); !ok {
		t.Fatalf("expected trace modelruntime_avoided=true trace=%#v", trace)
	}
	if ok, _ := trace["context_compile_avoided"].(bool); !ok {
		t.Fatalf("expected trace context_compile_avoided=true trace=%#v", trace)
	}
	if ok, _ := trace["gateway_avoided"].(bool); !ok {
		t.Fatalf("expected trace gateway_avoided=true trace=%#v", trace)
	}
	if htrace, ok := trace["hyperlane_trace"].(map[string]any); !ok {
		t.Fatalf("expected trace hyperlane_trace trace=%#v", trace)
	} else if got := strings.TrimSpace(asString(htrace["matched_rule"])); got == "" {
		t.Fatalf("expected trace hyperlane_trace matched_rule trace=%#v", trace)
	}
}

func assertNoModelRuntimeCalls(t *testing.T, fake *fakeModelRuntime) {
	t.Helper()
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.chatCalls != 0 || fake.healthCalls != 0 || fake.queueCalls != 0 || fake.loadedCalls != 0 {
		t.Fatalf("expected no modelruntime calls, chat=%d health=%d queue=%d loaded=%d", fake.chatCalls, fake.healthCalls, fake.queueCalls, fake.loadedCalls)
	}
}

func assertGatewayInvocationCount(t *testing.T, st *store.Store, want int) {
	t.Helper()
	var got int
	if err := st.DB.QueryRow(`SELECT COUNT(1) FROM gateway_invocations`).Scan(&got); err != nil {
		t.Fatalf("count gateway invocations: %v", err)
	}
	if got != want {
		t.Fatalf("gateway_invocations=%d want %d", got, want)
	}
}

func canonicalCounts(t *testing.T, st *store.Store) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{"memory_notes", "journal_events", "state_items", "open_loops", "contradiction_records"} {
		out[table] = countRows(t, st, table)
	}
	return out
}

func assertCanonicalCounts(t *testing.T, st *store.Store, before map[string]int) {
	t.Helper()
	for table, want := range before {
		if got := countRows(t, st, table); got != want {
			t.Fatalf("%s count=%d want %d", table, got, want)
		}
	}
}

func seedDreamReportNoModel(t *testing.T, st *store.Store, workspaceID, id string) {
	t.Helper()
	now := time.Now().UnixMilli()
	mustExecHyperlane(t, st, `INSERT INTO dream_reports(
id, created_at, completed_at, workspace_id, lane_id, mode, dry_run, status,
time_window_start, time_window_end, candidates_considered, proposals_generated,
summary_json, candidates_json, salience_scores_json, memory_tier_proposals_json,
repair_proposals_json, snapshot_hygiene_proposals_json, warnings_json, trace_json,
metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, now-100, now, workspaceID, "control.semantic", "microdream", 1, "completed",
		now-1000, now, 2, 1,
		`{"summary":"seeded dream report"}`, `[]`, `[]`, `[]`, `[]`, `[]`, `[]`, `{}`, `{}`,
	)
}

func mustExecHyperlane(t *testing.T, st *store.Store, query string, args ...any) {
	t.Helper()
	if _, err := st.DB.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}
