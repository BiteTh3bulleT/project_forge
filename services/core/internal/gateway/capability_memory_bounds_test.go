package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeCapabilityMemoryTextIsBounded(t *testing.T) {
	t.Parallel()
	text, err := normalizeCapabilityMemoryText("  fact  ", "content")
	if err != nil {
		t.Fatalf("expected valid memory text, got %v", err)
	}
	if text != "fact" {
		t.Fatalf("unexpected normalized text %q", text)
	}
	if _, err := normalizeCapabilityMemoryText(strings.Repeat("x", maxCapabilityMemoryTextBytes+1), "query"); !errors.Is(err, errCapabilityMemoryTextTooLarge) {
		t.Fatalf("expected memory text size rejection, got %v", err)
	}
}
