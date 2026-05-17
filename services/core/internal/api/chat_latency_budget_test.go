package api

import (
	"testing"
	"time"
)

func TestChatLatencyBudgetWarningPayloadUsesOnlyBoundedFields(t *testing.T) {
	trace := map[string]any{
		"total_request_ms":     int64(31_250),
		"modelruntime_ms":      int64(30_750),
		"gateway_execution_ms": int64(12),
		"route_intent":         "general_chat",
		"context_budget_class": "small",
		"output_mode":          "normal",
		"userRequestSummary":   "do not log this token=secret",
		"fallbackReason":       "do not log this Authorization=Bearer abc.def",
	}

	payload, ok := chatLatencyBudgetWarningPayload(42, 99, "corr-99", trace, 30*time.Second)
	if !ok {
		t.Fatalf("expected latency budget warning payload")
	}
	if got := payload["phase"]; got != "modelruntime" {
		t.Fatalf("phase=%v, want modelruntime", got)
	}
	if got := payload["threadId"]; got != int64(42) {
		t.Fatalf("threadId=%v, want 42", got)
	}
	if got := payload["userMessageId"]; got != int64(99) {
		t.Fatalf("userMessageId=%v, want 99", got)
	}
	if got := payload["correlationId"]; got != "corr-99" {
		t.Fatalf("correlationId=%v, want corr-99", got)
	}
	if got := payload["threshold_ms"]; got != int64(30_000) {
		t.Fatalf("threshold_ms=%v, want 30000", got)
	}
	if got := payload["phase_ms"]; got != int64(30_750) {
		t.Fatalf("phase_ms=%v, want 30750", got)
	}
	for _, forbidden := range []string{"userRequestSummary", "fallbackReason"} {
		if _, ok := payload[forbidden]; ok {
			t.Fatalf("payload copied secret-bearing field %q: %#v", forbidden, payload)
		}
	}
}

func TestChatLatencyBudgetWarningPayloadIgnoresFastTrace(t *testing.T) {
	trace := map[string]any{
		"total_request_ms":     int64(1_250),
		"modelruntime_ms":      int64(750),
		"gateway_execution_ms": int64(12),
	}

	if payload, ok := chatLatencyBudgetWarningPayload(42, 99, "corr-99", trace, 30*time.Second); ok {
		t.Fatalf("unexpected latency budget warning payload: %#v", payload)
	}
}
