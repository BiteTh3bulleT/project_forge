package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeCodeEvalInputIsBounded(t *testing.T) {
	t.Parallel()
	code, err := normalizeCodeEvalInput("  print('ok')  ")
	if err != nil {
		t.Fatalf("expected valid code, got %v", err)
	}
	if code != "print('ok')" {
		t.Fatalf("unexpected normalized code %q", code)
	}
	if _, err := normalizeCodeEvalInput(strings.Repeat("x", maxCodeEvalInputBytes+1)); !errors.Is(err, errCodeEvalTooLarge) {
		t.Fatalf("expected code size rejection, got %v", err)
	}
}
