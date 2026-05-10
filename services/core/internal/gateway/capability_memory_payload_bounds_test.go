package gateway

import (
	"context"
	"errors"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestNormalizeCapabilityMemoryIDIsBounded(t *testing.T) {
	t.Parallel()
	if got, err := normalizeCapabilityMemoryID(" fact-1 "); err != nil || got != "fact-1" {
		t.Fatalf("expected trimmed memory id, got %q err=%v", got, err)
	}
	if _, err := normalizeCapabilityMemoryID(strings.Repeat("x", maxCapabilityMemoryIDBytes+1)); !errors.Is(err, errCapabilityMemoryIDTooLarge) {
		t.Fatalf("expected memory id size rejection, got %v", err)
	}
}

func TestMarshalCapabilityMemoryPayloadIsBounded(t *testing.T) {
	t.Parallel()
	if payload, err := marshalCapabilityMemoryPayload(map[string]any{"content": "ok"}); err != nil || len(payload) == 0 {
		t.Fatalf("expected valid memory payload, got len=%d err=%v", len(payload), err)
	}
	if _, err := marshalCapabilityMemoryPayload(map[string]any{"content": strings.Repeat("x", maxCapabilityMemoryPayloadBytes+1)}); !errors.Is(err, errCapabilityMemoryPayloadTooLarge) {
		t.Fatalf("expected memory payload size rejection, got %v", err)
	}
}

func TestMemoryRememberRejectsOversizeIDBeforePersistence(t *testing.T) {
	t.Parallel()
	tool := capabilityBackingTool{capability: domain.ToolCapability{Name: "remember"}}
	_, err := tool.executeMemory(context.Background(), Request{Input: map[string]any{
		"id":      strings.Repeat("x", maxCapabilityMemoryIDBytes+1),
		"content": "bounded",
	}})
	if !errors.Is(err, errCapabilityMemoryIDTooLarge) {
		t.Fatalf("expected memory id size rejection, got %v", err)
	}
}
