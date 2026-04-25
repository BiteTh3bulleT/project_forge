package api

import (
	"testing"

	"forge/projectforge/services/core/internal/chat"
)

func TestChatThreadDetailForAPIOmitsRepeatedToolManifestPayloads(t *testing.T) {
	thread := &chat.ThreadDetail{
		ThreadSummary: chat.ThreadSummary{ID: 7, Title: "older thread"},
		Messages: []chat.Message{{
			ID:       11,
			ThreadID: 7,
			Role:     "assistant",
			Content:  "done",
			Metadata: map[string]any{
				"correlationId": "corr-1",
				"toolManifest":  []any{map[string]any{"id": "one"}, map[string]any{"id": "two"}},
				"toolGatewayActivity": map[string]any{
					"executionState": "completed",
					"toolManifest":   []any{map[string]any{"id": "nested"}},
				},
			},
		}},
	}

	projected := chatThreadDetailForAPI(thread)
	if projected == nil || len(projected.Messages) != 1 {
		t.Fatalf("expected projected thread message")
	}
	meta := projected.Messages[0].Metadata
	if _, exists := meta["toolManifest"]; exists {
		t.Fatalf("top-level toolManifest should be omitted from API projection")
	}
	if _, exists := thread.Messages[0].Metadata["toolManifest"]; !exists {
		t.Fatalf("source metadata should not be mutated")
	}
	summary, ok := meta["toolManifestSummary"].(map[string]any)
	if !ok || summary["omitted"] != true || summary["count"] != 2 {
		t.Fatalf("expected top-level manifest summary, got %#v", meta["toolManifestSummary"])
	}
	activity, ok := meta["toolGatewayActivity"].(map[string]any)
	if !ok {
		t.Fatalf("expected activity metadata map, got %#v", meta["toolGatewayActivity"])
	}
	if _, exists := activity["toolManifest"]; exists {
		t.Fatalf("nested toolManifest should be omitted from API projection")
	}
	nestedSummary, ok := activity["toolManifestSummary"].(map[string]any)
	if !ok || nestedSummary["omitted"] != true || nestedSummary["count"] != 1 {
		t.Fatalf("expected nested manifest summary, got %#v", activity["toolManifestSummary"])
	}
}
