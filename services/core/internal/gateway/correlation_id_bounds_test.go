package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeCorrelationIDIsBounded(t *testing.T) {
	t.Parallel()
	correlationID, err := normalizeCorrelationID("  corr-123  ")
	if err != nil {
		t.Fatalf("expected valid correlation id, got %v", err)
	}
	if correlationID != "corr-123" {
		t.Fatalf("unexpected normalized correlation id %q", correlationID)
	}
	if _, err := normalizeCorrelationID(strings.Repeat("c", maxCorrelationIDBytes+1)); !errors.Is(err, errCorrelationIDTooLarge) {
		t.Fatalf("expected correlation id size rejection, got %v", err)
	}
}
