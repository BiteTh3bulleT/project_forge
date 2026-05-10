package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeScheduleIDIsBounded(t *testing.T) {
	t.Parallel()
	id, err := normalizeScheduleID("  schedule-1  ")
	if err != nil {
		t.Fatalf("expected valid schedule id, got %v", err)
	}
	if id != "schedule-1" {
		t.Fatalf("unexpected normalized schedule id %q", id)
	}
	if _, err := normalizeScheduleID(strings.Repeat("s", maxScheduleIDBytes+1)); !errors.Is(err, errScheduleIDTooLarge) {
		t.Fatalf("expected schedule id size rejection, got %v", err)
	}
}

func TestMarshalSchedulePayloadIsBounded(t *testing.T) {
	t.Parallel()
	payload, err := marshalSchedulePayload(map[string]any{"note": "ok"})
	if err != nil {
		t.Fatalf("expected valid schedule payload, got %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("expected nonempty schedule payload")
	}
	if _, err := marshalSchedulePayload(map[string]any{"note": strings.Repeat("p", maxSchedulePayloadBytes+1)}); !errors.Is(err, errSchedulePayloadTooLarge) {
		t.Fatalf("expected schedule payload size rejection, got %v", err)
	}
}
