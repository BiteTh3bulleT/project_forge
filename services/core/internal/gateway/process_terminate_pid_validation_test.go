package gateway

import (
	"os"
	"testing"
)

func TestNormalizeTerminatePIDRejectsUnsafeTargets(t *testing.T) {
	t.Parallel()
	for _, pid := range []int{0, -1, 1, os.Getpid()} {
		if _, err := normalizeTerminatePID(float64(pid)); err == nil {
			t.Fatalf("expected pid %d to be rejected", pid)
		}
	}
}

func TestNormalizeTerminatePIDAllowsPositiveNonReservedPID(t *testing.T) {
	t.Parallel()
	got, err := normalizeTerminatePID(12345)
	if err != nil {
		t.Fatalf("expected non-reserved pid to be accepted: %v", err)
	}
	if got != 12345 {
		t.Fatalf("expected pid 12345, got %d", got)
	}
}
