package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/forgekshadow"
)

func TestForgeKShadowChatMetadataRequiresBothFlagsAtChatRoute(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	ctx := context.Background()
	thread, err := srv.chat.CreateThread(ctx, "chat metadata disabled", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true, ChatMetadataEnabled: false})
	srv.cfg.ForgeKShadowModeEnabled = true
	srv.cfg.ForgeKShadowChatMetadataEnabled = false
	srv.forgeKShadow = observer

	rr := postChatMessageForShadow(t, srv, thread.ID, `{"content":"hello metadata disabled","requestAssistant":false}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("chat post status=%d body=%s", rr.Code, rr.Body.String())
	}
	for _, report := range observer.Reports() {
		if report.ChatMetadata != nil {
			t.Fatalf("chat metadata stored when chat flag disabled: %#v", report.ChatMetadata)
		}
	}
}

func TestForgeKShadowChatMetadataObservedWithoutChangingResponseShape(t *testing.T) {
	body := `{"content":"DO-NOT-CAPTURE-CHAT-CONTENT","requestAssistant":false,"modelId":"local-model-a"}`

	baselineSrv, _ := newBackupAuditHarness(t)
	baselineThread, err := baselineSrv.chat.CreateThread(context.Background(), "baseline chat", nil)
	if err != nil {
		t.Fatalf("create baseline thread: %v", err)
	}
	baselineRR := postChatMessageForShadow(t, baselineSrv, baselineThread.ID, body)
	if baselineRR.Code != http.StatusOK {
		t.Fatalf("baseline chat post status=%d body=%s", baselineRR.Code, baselineRR.Body.String())
	}

	enabledSrv, _ := newBackupAuditHarness(t)
	enabledThread, err := enabledSrv.chat.CreateThread(context.Background(), "enabled chat", nil)
	if err != nil {
		t.Fatalf("create enabled thread: %v", err)
	}
	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true, ChatMetadataEnabled: true})
	enabledSrv.cfg.ForgeKShadowModeEnabled = true
	enabledSrv.cfg.ForgeKShadowChatMetadataEnabled = true
	enabledSrv.forgeKShadow = observer
	enabledRR := postChatMessageForShadow(t, enabledSrv, enabledThread.ID, body)
	if enabledRR.Code != http.StatusOK {
		t.Fatalf("enabled chat post status=%d body=%s", enabledRR.Code, enabledRR.Body.String())
	}

	assertChatPostResponseShape(t, baselineRR, enabledRR)
	report := findChatMetadataReport(t, observer.Reports())
	metadata := report.ChatMetadata
	if metadata.OperationKind != forgekshadow.ChatOperationMessagePost || metadata.ThreadID != strconv.FormatInt(enabledThread.ID, 10) || metadata.RoleClass != "user" {
		t.Fatalf("unexpected chat metadata: %#v", metadata)
	}
	if metadata.ModelID != "local-model-a" || metadata.StreamClass != "none" || metadata.MessageCount != 1 {
		t.Fatalf("unexpected chat metadata model/stream/count: %#v", metadata)
	}
	if report.Observation.Metadata["observation_type"] != "chat_metadata" || report.Observation.Metadata["touchpoint"] != "chat_message_post" {
		t.Fatalf("unexpected chat observation metadata: %#v", report.Observation.Metadata)
	}
	serializedReport := strings.ToLower(fmt.Sprint(report))
	for _, forbidden := range []string{"do-not-capture-chat-content", "body", "prompt", "completion", "tool_output", "retrieval_content", "memory_content"} {
		if strings.Contains(serializedReport, forbidden) {
			t.Fatalf("chat metadata report leaked forbidden fragment %q in %q", forbidden, serializedReport)
		}
	}
}

func TestForgeKShadowChatMetadataNoPublicDiagnosticRoutes(t *testing.T) {
	disabled := collectServerRoutes(t, (&Server{}).Handler())
	enabled := collectServerRoutes(t, (&Server{
		cfg: config.Config{
			ForgeKShadowModeEnabled:         true,
			ForgeKShadowChatMetadataEnabled: true,
		},
		forgeKShadow: forgekshadow.NewObserver(forgekshadow.Config{Enabled: true, ChatMetadataEnabled: true}),
	}).Handler())
	if !sameRouteSet(disabled, enabled) {
		t.Fatalf("chat metadata shadow changed route inventory\ndisabled=%#v\nenabled=%#v", routeKeys(disabled), routeKeys(enabled))
	}
	for _, route := range routeKeys(enabled) {
		normalized := strings.ToLower(route)
		for _, forbidden := range []string{"chat-shadow", "chat-metadata", "forgek-shadow", "shadow-diagnostic", "/api/shadow", "/forge/shadow"} {
			if strings.Contains(normalized, forbidden) {
				t.Fatalf("chat metadata shadow must not expose public diagnostics route: %s", route)
			}
		}
	}
}

func TestForgeKShadowChatMetadataStreamRequestCapturesOnlyMetadata(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	thread, err := srv.chat.CreateThread(context.Background(), "stream chat metadata", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true, ChatMetadataEnabled: true})
	srv.cfg.ForgeKShadowModeEnabled = true
	srv.cfg.ForgeKShadowChatMetadataEnabled = true
	srv.forgeKShadow = observer

	rr := postChatMessageForShadow(t, srv, thread.ID, `{"content":"STREAM-BODY-MUST-NOT-APPEAR","requestAssistant":true,"stream":true}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("stream chat post status=%d body=%s", rr.Code, rr.Body.String())
	}
	report := findChatMetadataReport(t, observer.Reports())
	if report.ChatMetadata.StreamClass != "stream" {
		t.Fatalf("expected stream metadata class, got %#v", report.ChatMetadata)
	}
	serializedReport := strings.ToLower(fmt.Sprint(report))
	if strings.Contains(serializedReport, "stream-body-must-not-appear") {
		t.Fatalf("stream chat metadata report captured message body: %q", serializedReport)
	}
}

func postChatMessageForShadow(t *testing.T, srv *Server, threadID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/chat/threads/"+strconv.FormatInt(threadID, 10)+"/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "request-chat-shadow")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	return rr
}

func findChatMetadataReport(t *testing.T, reports []forgekshadow.DiagnosticReport) forgekshadow.DiagnosticReport {
	t.Helper()
	for _, report := range reports {
		if report.ChatMetadata != nil {
			return report
		}
	}
	t.Fatalf("expected chat metadata report, got %#v", reports)
	return forgekshadow.DiagnosticReport{}
}

func assertChatPostResponseShape(t *testing.T, baselineRR, enabledRR *httptest.ResponseRecorder) {
	t.Helper()
	if baselineRR.Code != enabledRR.Code {
		t.Fatalf("chat metadata changed status baseline=%d enabled=%d", baselineRR.Code, enabledRR.Code)
	}
	if baselineRR.Header().Get("Content-Type") != enabledRR.Header().Get("Content-Type") {
		t.Fatalf("chat metadata changed content type baseline=%q enabled=%q", baselineRR.Header().Get("Content-Type"), enabledRR.Header().Get("Content-Type"))
	}
	var baseline, enabled map[string]any
	if err := json.Unmarshal(baselineRR.Body.Bytes(), &baseline); err != nil {
		t.Fatalf("decode baseline response: %v", err)
	}
	if err := json.Unmarshal(enabledRR.Body.Bytes(), &enabled); err != nil {
		t.Fatalf("decode enabled response: %v", err)
	}
	for _, key := range []string{"assistantPending", "stream", "asyncAssistant"} {
		if baseline[key] != enabled[key] {
			t.Fatalf("chat metadata changed response key %q baseline=%v enabled=%v", key, baseline[key], enabled[key])
		}
	}
	if baseline["assistantMessage"] != enabled["assistantMessage"] {
		t.Fatalf("chat metadata changed assistant message baseline=%v enabled=%v", baseline["assistantMessage"], enabled["assistantMessage"])
	}
	baseUser, _ := baseline["userMessage"].(map[string]any)
	enabledUser, _ := enabled["userMessage"].(map[string]any)
	for _, key := range []string{"role", "content"} {
		if baseUser[key] != enabledUser[key] {
			t.Fatalf("chat metadata changed user message %q baseline=%v enabled=%v", key, baseUser[key], enabledUser[key])
		}
	}
}
