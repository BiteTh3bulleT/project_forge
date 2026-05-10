package gateway

import (
	"errors"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestMarshalDesktopBridgePayloadIsBounded(t *testing.T) {
	t.Parallel()
	tool := capabilityBackingTool{capability: domain.ToolCapability{ID: "ui.render"}}
	body, err := tool.marshalDesktopBridgePayload(Request{Input: map[string]any{"label": "ok"}})
	if err != nil {
		t.Fatalf("expected valid desktop bridge payload, got %v", err)
	}
	if len(body) == 0 {
		t.Fatal("expected nonempty desktop bridge payload")
	}
	if _, err := tool.marshalDesktopBridgePayload(Request{Input: map[string]any{"label": strings.Repeat("x", maxDesktopBridgeRequestBodyBytes+1)}}); !errors.Is(err, errDesktopBridgeRequestBodyTooLarge) {
		t.Fatalf("expected desktop bridge payload size rejection, got %v", err)
	}
}
