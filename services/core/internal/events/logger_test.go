package events

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestEmitWritesStructuredLogWithCorrelationFields(t *testing.T) {
	dbStore, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer dbStore.Close()

	var logs bytes.Buffer
	logger := NewWithStructuredOutput(dbStore.DB, &logs)
	logger.nowMillis = func() int64 { return 1760000000000 }

	payload := map[string]any{
		"correlationId": "corr-chat-1",
		"requestId":     "req-chat-1",
		"trace_id":      "trace-chat-1",
		"threadId":      int64(42),
	}
	if err := logger.Emit(context.Background(), "chat.tool.pipeline", payload); err != nil {
		t.Fatalf("emit event: %v", err)
	}

	line, err := bufio.NewReader(&logs).ReadString('\n')
	if err != nil {
		t.Fatalf("read structured log line: %v", err)
	}
	record := map[string]any{}
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("decode structured log %q: %v", line, err)
	}

	if record["event_type"] != "chat.tool.pipeline" {
		t.Fatalf("event_type = %#v", record["event_type"])
	}
	if record["correlation_id"] != "corr-chat-1" {
		t.Fatalf("correlation_id = %#v", record["correlation_id"])
	}
	if record["request_id"] != "req-chat-1" {
		t.Fatalf("request_id = %#v", record["request_id"])
	}
	if record["trace_id"] != "trace-chat-1" {
		t.Fatalf("trace_id = %#v", record["trace_id"])
	}
	if record["payload_json"] == "" {
		t.Fatalf("expected bounded payload_json in structured log")
	}
}

func TestEmitBoundsStructuredLogPayload(t *testing.T) {
	dbStore, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer dbStore.Close()

	var logs bytes.Buffer
	logger := NewWithStructuredOutput(dbStore.DB, &logs)
	if err := logger.Emit(context.Background(), "large.event", map[string]any{
		"correlation_id": "corr-large",
		"body":           strings.Repeat("x", structuredPayloadLogMaxBytes*2),
	}); err != nil {
		t.Fatalf("emit event: %v", err)
	}

	line, err := bufio.NewReader(&logs).ReadString('\n')
	if err != nil {
		t.Fatalf("read structured log line: %v", err)
	}
	record := map[string]any{}
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("decode structured log %q: %v", line, err)
	}
	payload, _ := record["payload_json"].(string)
	if len(payload) > structuredPayloadLogMaxBytes+len(structuredPayloadTruncatedSuffix) {
		t.Fatalf("payload_json length = %d, want bounded", len(payload))
	}
	if !strings.HasSuffix(payload, structuredPayloadTruncatedSuffix) {
		t.Fatalf("expected truncated payload suffix, got %q", payload[len(payload)-min(len(payload), 80):])
	}
}
