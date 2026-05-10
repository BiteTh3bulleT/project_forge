package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeConfiguredCommandArgsIsBounded(t *testing.T) {
	t.Parallel()
	args, err := normalizeConfiguredCommandArgs("  --flag value  ")
	if err != nil {
		t.Fatalf("expected valid configured command args, got %v", err)
	}
	if len(args) != 2 || args[0] != "--flag" || args[1] != "value" {
		t.Fatalf("unexpected args %#v", args)
	}
	if _, err := normalizeConfiguredCommandArgs(strings.Repeat("x", maxConfiguredCommandArgsBytes+1)); !errors.Is(err, errConfiguredCommandArgsTooLarge) {
		t.Fatalf("expected configured command args size rejection, got %v", err)
	}
}
