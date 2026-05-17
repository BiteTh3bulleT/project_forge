package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/hyperlane"
	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/gateway"
)

const (
	chatBudgetTiny   = "tiny"
	chatBudgetSmall  = "small"
	chatBudgetMedium = "medium"
	chatBudgetDeep   = "deep"
	chatBudgetReport = "report"

	chatOutputBrief  = "brief"
	chatOutputNormal = "normal"
	chatOutputDeep   = "deep"
	chatOutputReport = "report"

	chatLatencyBudgetWarningThreshold = 30 * time.Second
)

type chatPerformanceDecision struct {
	Intent             string
	ContextBudgetClass string
	OutputMode         string
	NoModel            bool
	Confidence         float64
	Reason             string
	HyperlaneMs        int64
	HyperlaneIntent    hyperlane.Intent
}

func classifyChatPerformance(content string) (decision chatPerformanceDecision) {
	start := time.Now()
	normalized := normalizeAssistantIntent(content)
	decision = chatPerformanceDecision{
		Intent:             "general_chat",
		ContextBudgetClass: chatBudgetSmall,
		OutputMode:         chatOutputNormal,
		Confidence:         0.45,
		Reason:             "default small chat budget",
	}
	defer func() {
		decision.HyperlaneMs = time.Since(start).Milliseconds()
	}()

	if normalized == "" {
		decision.Intent = "empty"
		decision.ContextBudgetClass = chatBudgetTiny
		decision.OutputMode = chatOutputBrief
		decision.NoModel = true
		decision.Confidence = 1
		decision.Reason = "empty request has deterministic reply"
		return decision
	}

	if deterministicNoModelIntent(normalized) {
		decision.Intent = "deterministic_chat_reply"
		decision.ContextBudgetClass = chatBudgetTiny
		decision.OutputMode = chatOutputBrief
		decision.NoModel = true
		decision.Confidence = 1
		decision.Reason = "identity or missing live-context request"
		return decision
	}

	intent := gateway.ParseHyperlaneIntent(content)
	decision.HyperlaneIntent = intent
	if isSupportedNoModelHyperlaneIntent(intent) {
		decision.Intent = string(intent.Type)
		decision.ContextBudgetClass = chatBudgetTiny
		decision.OutputMode = chatOutputBrief
		decision.NoModel = true
		decision.Confidence = intent.Confidence
		decision.Reason = "hyperlane deterministic no-model route: " + intent.MatchedRule
		return decision
	}

	if isStatusLikeIntent(normalized) {
		decision.Intent = "status"
		decision.ContextBudgetClass = chatBudgetTiny
		decision.OutputMode = chatOutputBrief
		decision.NoModel = true
		decision.Confidence = 0.92
		decision.Reason = "structured status request"
		return decision
	}
	if isDiagnosticsLikeIntent(normalized) {
		decision.Intent = "diagnostics"
		decision.ContextBudgetClass = chatBudgetTiny
		decision.OutputMode = chatOutputBrief
		decision.NoModel = true
		decision.Confidence = 0.9
		decision.Reason = "structured diagnostics request"
		return decision
	}
	if isRestoreInspectorIntent(normalized) {
		decision.Intent = "restore_inspector"
		decision.ContextBudgetClass = chatBudgetTiny
		decision.OutputMode = chatOutputBrief
		decision.NoModel = true
		decision.Confidence = 0.88
		decision.Reason = "restore inspector summary request"
		return decision
	}

	if isExplicitReportIntent(normalized) {
		decision.Intent = "report"
		decision.ContextBudgetClass = chatBudgetReport
		decision.OutputMode = chatOutputReport
		decision.Confidence = 0.86
		decision.Reason = "explicit long-form report request"
		return decision
	}
	if isDeepReasoningIntent(normalized) {
		decision.Intent = "deep_reasoning"
		decision.ContextBudgetClass = chatBudgetDeep
		decision.OutputMode = chatOutputDeep
		decision.Confidence = 0.78
		decision.Reason = "explicit deep reasoning or review request"
		return decision
	}
	if len(normalized) > 600 {
		decision.ContextBudgetClass = chatBudgetMedium
		decision.Reason = "larger operator turn"
	}
	return decision
}

func isSupportedNoModelHyperlaneIntent(intent hyperlane.Intent) bool {
	if intent.RequiresModel || intent.RequiresGateway {
		return false
	}
	switch intent.Type {
	case hyperlane.IntentStatusQuery,
		hyperlane.IntentDiagnosticsQuery,
		hyperlane.IntentChatMemoryInspection,
		hyperlane.IntentChatHistoryLookup,
		hyperlane.IntentRestoreInspection,
		hyperlane.IntentDreamReportInspection,
		hyperlane.IntentModelruntimeStatus:
		return true
	default:
		return false
	}
}

func deterministicNoModelIntent(normalized string) bool {
	if normalized == "what is your name" || normalized == "whats your name" || normalized == "who are you" || normalized == "what are you" {
		return true
	}
	if normalized == "what is my name" || normalized == "whats my name" || normalized == "who am i" {
		return true
	}
	return isWeatherWithoutLocationQuery(normalized)
}

func isStatusLikeIntent(normalized string) bool {
	statusPhrases := []string{
		"status", "health", "what mode are we in", "which mode are we in", "current mode",
		"modelruntime status", "model runtime status", "runtime status", "backend disabled",
		"backend is disabled", "safe mode", "cpu only", "cpu-only", "what is degraded", "what's degraded",
	}
	if normalized == "what mode" || normalized == "mode" || normalized == "health check" {
		return true
	}
	for _, phrase := range statusPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func isDiagnosticsLikeIntent(normalized string) bool {
	if strings.Contains(normalized, "diagnostic") || strings.Contains(normalized, "diagnostics") {
		return true
	}
	if strings.Contains(normalized, "what is broken") || strings.Contains(normalized, "what's broken") || strings.Contains(normalized, "why is forge slow") {
		return true
	}
	return false
}

func isRestoreInspectorIntent(normalized string) bool {
	if !strings.Contains(normalized, "restore") && !strings.Contains(normalized, "context snapshot") {
		return false
	}
	for _, phrase := range []string{"latest", "summary", "score", "decision", "inspector", "snapshot", "show recent"} {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func isExplicitReportIntent(normalized string) bool {
	return strings.Contains(normalized, "full report") ||
		strings.Contains(normalized, "write a report") ||
		strings.Contains(normalized, "generate a report") ||
		strings.Contains(normalized, "complete review") ||
		strings.Contains(normalized, "entire repo")
}

func isDeepReasoningIntent(normalized string) bool {
	return strings.Contains(normalized, "deep review") ||
		strings.Contains(normalized, "security review") ||
		strings.Contains(normalized, "architecture review") ||
		strings.Contains(normalized, "root cause") ||
		strings.Contains(normalized, "investigate")
}

func (s *Server) renderNoModelChatReply(ctx context.Context, decision chatPerformanceDecision, th *chat.ThreadDetail, content string) string {
	switch decision.HyperlaneIntent.Type {
	case hyperlane.IntentStatusQuery:
		return s.renderFastStatusReply()
	case hyperlane.IntentDiagnosticsQuery:
		return s.renderFastDiagnosticsReply()
	case hyperlane.IntentRestoreInspection:
		return renderFastRestoreReply(th)
	case hyperlane.IntentDreamReportInspection:
		return s.renderFastDreamReportReply(ctx)
	case hyperlane.IntentModelruntimeStatus:
		return s.renderFastModelRuntimeStatusReply(ctx)
	}

	switch decision.Intent {
	case "empty":
		return "No request text received."
	case "deterministic_chat_reply":
		if text, ok := deterministicNoToolChatReply(th, content); ok {
			return text
		}
	case "status":
		return s.renderFastStatusReply()
	case "diagnostics":
		return s.renderFastDiagnosticsReply()
	case "restore_inspector":
		return renderFastRestoreReply(th)
	}
	return ""
}

func (s *Server) renderFastStatusReply() string {
	runtime := "unregistered"
	if s.modelRuntime != nil {
		runtime = "registered"
	}
	gatewayState := "unregistered"
	if s.gateway != nil {
		gatewayState = "registered"
	}
	mode := "cpu-authoritative"
	if s.cfg.SafeModeForceCPUOnly {
		mode = "cpu-only safe mode"
	}
	return fmt.Sprintf("Core: online. Mode: %s. Gateway: %s. Modelruntime: %s. Fast path: no model call.", mode, gatewayState, runtime)
}

func (s *Server) renderFastDiagnosticsReply() string {
	parts := []string{"Diagnostics fast path:"}
	parts = append(parts, "core online")
	if s.gateway != nil {
		parts = append(parts, "gateway registered")
	} else {
		parts = append(parts, "gateway unavailable")
	}
	if s.modelRuntime != nil {
		parts = append(parts, "modelruntime registered")
	} else {
		parts = append(parts, "modelruntime unavailable")
	}
	if s.cfg.SafeModeForceCPUOnly {
		parts = append(parts, "CPU-only safe mode active")
	}
	parts = append(parts, "provider health not probed on no-model route")
	return strings.Join(parts, "; ") + "."
}

func (s *Server) renderFastModelRuntimeStatusReply(ctx context.Context) string {
	if s.modelRuntime == nil {
		return "Modelruntime: unavailable. Core remains online on the CPU-side no-model route."
	}
	meta := ModelRuntimeRequestMeta{WorkspaceID: strings.TrimSpace(s.cfg.WorkspaceDir)}
	parts := []string{"Modelruntime fast path:"}
	if health, err := s.modelRuntime.Health(ctx, meta); err == nil {
		status := strings.TrimSpace(health.Status)
		if status == "" {
			if health.OK {
				status = "ok"
			} else {
				status = "degraded"
			}
		}
		parts = append(parts, "status "+status)
		if backend := strings.TrimSpace(health.Backend); backend != "" {
			parts = append(parts, "backend "+backend)
		}
		if len(health.DegradedReasons) > 0 {
			parts = append(parts, "degraded: "+strings.Join(health.DegradedReasons, ", "))
		}
	} else {
		parts = append(parts, "health unavailable: "+err.Error())
	}
	if queue, err := s.modelRuntime.QueueStatus(ctx, meta); err == nil {
		parts = append(parts, fmt.Sprintf("queue depth %d", queue.Depth))
		if state := strings.TrimSpace(queue.PolicyState); state != "" {
			parts = append(parts, "policy "+state)
		}
	} else {
		parts = append(parts, "queue unavailable: "+err.Error())
	}
	if loaded, err := s.modelRuntime.LoadedStatus(ctx, meta); err == nil {
		parts = append(parts, fmt.Sprintf("loaded %d", loaded.Count))
	}
	if models, err := s.modelRuntime.ListModels(ctx, ModelRuntimeListRequest{Meta: meta}); err == nil {
		available := 0
		for _, model := range models {
			status := strings.ToLower(strings.TrimSpace(model.Status))
			if status == "available" || status == "loaded" {
				available++
			}
		}
		parts = append(parts, fmt.Sprintf("registered %d", len(models)), fmt.Sprintf("available %d", available))
	}
	parts = append(parts, "no model call executed")
	return strings.Join(parts, "; ") + "."
}

func (s *Server) renderFastDreamReportReply(ctx context.Context) string {
	_ = ctx
	if s.dream == nil {
		return "Dream Mode: unavailable. No persisted Dream reports are attached to this chat."
	}
	return "Dream Mode: service registered. No persisted Dream report is attached to this chat; run /api/dream/run for a fresh dry-run report."
}

func renderFastRestoreReply(th *chat.ThreadDetail) string {
	if th == nil {
		return "No restore package is attached to this chat context."
	}
	for i := len(th.Messages) - 1; i >= 0; i-- {
		msg := th.Messages[i]
		if msg.Metadata == nil {
			continue
		}
		if raw, ok := msg.Metadata["restore_package_json"]; ok {
			return fmt.Sprintf("Latest restore package: %v", raw)
		}
		if raw, ok := msg.Metadata["restore_scores_json"]; ok {
			return fmt.Sprintf("Latest restore scores: %v", raw)
		}
		if raw, ok := msg.Metadata["restoreSummary"]; ok {
			return fmt.Sprintf("Latest restore summary: %v", raw)
		}
	}
	return "No restore package is attached to this chat thread yet."
}

func chatOutputModeMaxTokens(mode string) int {
	switch mode {
	case chatOutputBrief:
		return 160
	case chatOutputDeep:
		return 768
	case chatOutputReport:
		return 1024
	default:
		return modelRuntimePlainChatMaxOutputToken
	}
}

func chatLatencyTrace(start time.Time, decision chatPerformanceDecision, extras map[string]any) map[string]any {
	trace := map[string]any{
		"total_request_ms":     time.Since(start).Milliseconds(),
		"hyperlane_ms":         decision.HyperlaneMs,
		"context_budget_class": decision.ContextBudgetClass,
		"output_mode":          decision.OutputMode,
		"route_intent":         decision.Intent,
		"route_reason":         decision.Reason,
		"route_confidence":     decision.Confidence,
	}
	for k, v := range extras {
		trace[k] = v
	}
	return trace
}

func chatHyperlaneNoModelTrace(start time.Time, decision chatPerformanceDecision) map[string]any {
	extras := map[string]any{
		"latency_ms":               time.Since(start).Milliseconds(),
		"modelruntime_avoided":     true,
		"model_calls_avoided":      1,
		"context_compile_avoided":  true,
		"fresh_compile_avoided":    1,
		"gateway_avoided":          true,
		"restore_ms":               int64(0),
		"context_compile_ms":       int64(0),
		"modelruntime_ms":          int64(0),
		"gateway_preflight_ms":     int64(0),
		"gateway_execution_ms":     int64(0),
		"hyperlane_parser_version": hyperlane.ParserVersion,
	}
	intent := decision.HyperlaneIntent
	if intent.Type != "" {
		extras["hyperlane_intent_type"] = string(intent.Type)
		extras["hyperlane_route"] = intent.Route
		extras["hyperlane_confidence"] = intent.Confidence
		extras["hyperlane_matched_rule"] = intent.MatchedRule
		extras["hyperlane_trace"] = map[string]any{
			"parser_version":  intent.Trace.ParserVersion,
			"matched_rule":    intent.Trace.MatchedRule,
			"confidence":      intent.Trace.Confidence,
			"route":           intent.Trace.Route,
			"warnings":        intent.Trace.Warnings,
			"rejected_reason": intent.Trace.RejectedReason,
		}
	}
	return chatLatencyTrace(start, decision, extras)
}

func chatLatencyTraceWithPrompt(start time.Time, decision chatPerformanceDecision, prompt modelRuntimePromptBudget, extras map[string]any) map[string]any {
	if extras == nil {
		extras = map[string]any{}
	}
	extras["tokens_estimated"] = apiMaxInt(1, prompt.TotalChars/4)
	extras["context_compile_ms"] = int64(0)
	return chatLatencyTrace(start, decision, extras)
}

func (s *Server) warnIfChatLatencyBudgetExceeded(ctx context.Context, threadID, userMessageID int64, correlationID string, trace map[string]any) {
	payload, ok := chatLatencyBudgetWarningPayload(threadID, userMessageID, correlationID, trace, chatLatencyBudgetWarningThreshold)
	if !ok {
		return
	}
	apiLogWarn("chat latency budget exceeded",
		slog.Int64("threadId", threadID),
		slog.Int64("userMessageId", userMessageID),
		slog.String("correlationId", strings.TrimSpace(correlationID)),
		slog.String("phase", asString(payload["phase"])),
		slog.Int64("threshold_ms", chatTraceInt64(payload["threshold_ms"])),
		slog.Int64("phase_ms", chatTraceInt64(payload["phase_ms"])),
		slog.Int64("total_request_ms", chatTraceInt64(payload["total_request_ms"])),
		slog.String("route_intent", asString(payload["route_intent"])),
		slog.String("context_budget_class", asString(payload["context_budget_class"])),
		slog.String("output_mode", asString(payload["output_mode"])),
	)
}

func chatLatencyBudgetWarningPayload(threadID, userMessageID int64, correlationID string, trace map[string]any, threshold time.Duration) (map[string]any, bool) {
	if len(trace) == 0 {
		return nil, false
	}
	if threshold <= 0 {
		threshold = chatLatencyBudgetWarningThreshold
	}
	thresholdMs := threshold.Milliseconds()
	phase, phaseMs := slowestCriticalChatLatencyPhase(trace)
	totalMs := chatTraceInt64(trace["total_request_ms"])
	if phaseMs < thresholdMs && totalMs < thresholdMs {
		return nil, false
	}
	if phase == "" || phaseMs < thresholdMs {
		phase = "total_request"
		phaseMs = totalMs
	}
	payload := map[string]any{
		"threadId":             threadID,
		"userMessageId":        userMessageID,
		"correlationId":        strings.TrimSpace(correlationID),
		"phase":                phase,
		"threshold_ms":         thresholdMs,
		"phase_ms":             phaseMs,
		"total_request_ms":     totalMs,
		"route_intent":         asString(trace["route_intent"]),
		"context_budget_class": asString(trace["context_budget_class"]),
		"output_mode":          asString(trace["output_mode"]),
	}
	return payload, true
}

func slowestCriticalChatLatencyPhase(trace map[string]any) (string, int64) {
	critical := []struct {
		traceKey string
		phase    string
	}{
		{traceKey: "context_compile_ms", phase: "context_compile"},
		{traceKey: "restore_ms", phase: "restore"},
		{traceKey: "gateway_preflight_ms", phase: "gateway_preflight"},
		{traceKey: "gateway_execution_ms", phase: "gateway_execution"},
		{traceKey: "modelruntime_ms", phase: "modelruntime"},
		{traceKey: "modelruntime_first_token_ms", phase: "modelruntime_first_token"},
	}
	var slowestPhase string
	var slowestMs int64
	for _, item := range critical {
		ms := chatTraceInt64(trace[item.traceKey])
		if ms > slowestMs {
			slowestMs = ms
			slowestPhase = item.phase
		}
	}
	return slowestPhase, slowestMs
}

func chatTraceInt64(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return n
		}
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err == nil {
			return n
		}
	}
	return 0
}

func apiMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *Server) modelRuntimeChatPreflight(ctx context.Context, meta ModelRuntimeRequestMeta, modelID string) (map[string]any, string) {
	if s.modelRuntime == nil {
		return nil, "model runtime is unavailable"
	}
	start := time.Now()
	queue, err := s.modelRuntime.QueueStatus(ctx, meta)
	trace := map[string]any{"gateway_preflight_ms": time.Since(start).Milliseconds()}
	if err != nil {
		_, code, message := mapModelRuntimeError(err)
		return trace, message + " (" + code + ")"
	}
	trace["runtime_queue_depth"] = queue.Depth
	if strings.Contains(strings.ToLower(queue.PolicyState), "cooldown") {
		trace["runtime_policy_state"] = queue.PolicyState
		return trace, "model runtime provider cooldown active"
	}
	if queue.Depth >= 64 {
		trace["runtime_policy_state"] = queue.PolicyState
		return trace, "model runtime queue is saturated"
	}
	if reason := s.modelRuntimeLoadedPreflight(ctx, meta, strings.TrimSpace(modelID), queue, trace); reason != "" {
		return trace, reason
	}
	return trace, ""
}

func (s *Server) modelRuntimeLoadedPreflight(ctx context.Context, meta ModelRuntimeRequestMeta, modelID string, queue ModelRuntimeQueueStatus, trace map[string]any) string {
	if modelID == "" {
		return ""
	}
	model, err := s.modelRuntime.GetModel(ctx, modelID, meta)
	if err != nil {
		_, code, message := mapModelRuntimeError(err)
		if strings.TrimSpace(code) != "" {
			return message + " (" + code + ")"
		}
		return message
	}
	trace["runtime_model_id"] = model.ID
	trace["runtime_model_status"] = model.Status
	trace["runtime_model_backend"] = model.Backend
	health, err := s.modelRuntime.Health(ctx, meta)
	if err != nil {
		_, code, message := mapModelRuntimeError(err)
		if strings.TrimSpace(code) != "" {
			return message + " (" + code + ")"
		}
		return message
	}
	trace["runtime_health_status"] = health.Status
	trace["runtime_health_ok"] = health.OK
	if backend := strings.TrimSpace(model.Backend); backend != "" {
		if !modelRuntimeBackendHealthy(health, backend) {
			return "model runtime backend unavailable: " + backend
		}
	}
	if strings.EqualFold(strings.TrimSpace(model.Status), "loaded") {
		return ""
	}
	backend := strings.TrimSpace(model.Backend)
	if backend != "" {
		if activeID := strings.TrimSpace(queue.Active[backend]); activeID != "" {
			trace["runtime_active_model_id"] = activeID
			if strings.EqualFold(activeID, modelID) {
				return ""
			}
		}
	}
	loaded, err := s.modelRuntime.LoadedStatus(ctx, meta)
	if err != nil {
		_, code, message := mapModelRuntimeError(err)
		if strings.TrimSpace(code) != "" {
			return message + " (" + code + ")"
		}
		return message
	}
	trace["runtime_loaded_count"] = loaded.Count
	for _, item := range loaded.Models {
		if !strings.EqualFold(strings.TrimSpace(item.ModelID), modelID) {
			continue
		}
		trace["runtime_loaded_status"] = item.Status
		if strings.EqualFold(strings.TrimSpace(item.Status), "loaded") {
			return ""
		}
	}
	return ""
}

func modelRuntimeBackendHealthy(health ModelRuntimeHealth, backend string) bool {
	if strings.TrimSpace(backend) == "" {
		return true
	}
	details := health.Details
	if details == nil {
		return true
	}
	rawBackends, ok := details["backends"].(map[string]map[string]any)
	if !ok {
		if generic, genericOK := details["backends"].(map[string]any); genericOK {
			if entry, entryOK := generic[backend].(map[string]any); entryOK {
				if healthy, present := entry["healthy"].(bool); present {
					return healthy
				}
			}
		}
		return true
	}
	entry, ok := rawBackends[backend]
	if !ok {
		return true
	}
	healthy, ok := entry["healthy"].(bool)
	if !ok {
		return true
	}
	return healthy
}
