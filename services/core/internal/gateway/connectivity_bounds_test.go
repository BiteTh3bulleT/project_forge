package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeConnectivityTargetIsBounded(t *testing.T) {
	t.Parallel()
	target, err := normalizeConnectivityTarget("  example.com:443  ")
	if err != nil {
		t.Fatalf("expected valid connectivity target, got %v", err)
	}
	if target != "example.com:443" {
		t.Fatalf("unexpected normalized target %q", target)
	}
	if _, err := normalizeConnectivityTarget(strings.Repeat("x", maxConnectivityTargetBytes+1)); !errors.Is(err, errConnectivityTargetTooLarge) {
		t.Fatalf("expected connectivity target size rejection, got %v", err)
	}
}
