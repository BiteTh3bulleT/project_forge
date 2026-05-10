package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeCapabilityConfigKeyIsBounded(t *testing.T) {
	t.Parallel()
	key, err := normalizeCapabilityConfigKey("  forge.mode  ")
	if err != nil {
		t.Fatalf("expected valid config key, got %v", err)
	}
	if key != "forge.mode" {
		t.Fatalf("unexpected normalized key %q", key)
	}
	if _, err := normalizeCapabilityConfigKey(strings.Repeat("k", maxCapabilityConfigKeyBytes+1)); !errors.Is(err, errCapabilityConfigKeyTooLarge) {
		t.Fatalf("expected config key size rejection, got %v", err)
	}
}

func TestNormalizeCapabilityConfigValueIsBounded(t *testing.T) {
	t.Parallel()
	value, err := normalizeCapabilityConfigValue("  exact value  ")
	if err != nil {
		t.Fatalf("expected valid config value, got %v", err)
	}
	if value != "  exact value  " {
		t.Fatalf("unexpected normalized value %q", value)
	}
	if _, err := normalizeCapabilityConfigValue(strings.Repeat("v", maxCapabilityConfigValueBytes+1)); !errors.Is(err, errCapabilityConfigValueTooLarge) {
		t.Fatalf("expected config value size rejection, got %v", err)
	}
}
