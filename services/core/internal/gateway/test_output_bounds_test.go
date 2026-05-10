package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeTestOutputInputIsBounded(t *testing.T) {
	t.Parallel()
	output, err := normalizeTestOutputInput("  PASS\n")
	if err != nil {
		t.Fatalf("expected valid test output, got %v", err)
	}
	if output != "PASS" {
		t.Fatalf("unexpected normalized output %q", output)
	}
	if _, err := normalizeTestOutputInput(strings.Repeat("x", maxTestOutputInputBytes+1)); !errors.Is(err, errTestOutputTooLarge) {
		t.Fatalf("expected test output size rejection, got %v", err)
	}
}
