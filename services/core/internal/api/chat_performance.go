package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/chat"
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
)

type chatPerformanceDecision struct {
	Intent             string
	ContextBudgetClass string
	OutputMode         string
	NoModel            bool
	Confidence         float64
	Reason             string
	HyperlaneMs        int64
}

func classifyChatPerformance(content string) chatPerformanceDecision {
	start := time.Now()
	normalized := normalizeAssistantIntent(content)
	decision := chatPerformanceDecision{
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

func deterministicNoModelIntent(normalized string) bool {
	if normalized == "what is your name" || normalized == "whats your name" || normalized == "who are you" || normalized == "what are you" {
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
	switch decision.Intent {
	case "empty":
		return "No request text received."
	case "deterministic_chat_reply":
		if text, ok := deterministicNoToolChatReply(content); ok {
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

func chatLatencyTraceWithPrompt(start time.Time, decision chatPerformanceDecision, prompt modelRuntimePromptBudget, extras map[string]any) map[string]any {
	if extras == nil {
		extras = map[string]any{}
	}
	extras["tokens_estimated"] = apiMaxInt(1, prompt.TotalChars/4)
	extras["context_compile_ms"] = int64(0)
	return chatLatencyTrace(start, decision, extras)
}

func apiMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *Server) modelRuntimeChatPreflight(ctx context.Context, meta ModelRuntimeRequestMeta) (map[string]any, string) {
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
	return trace, ""
}
