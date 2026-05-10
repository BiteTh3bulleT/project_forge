package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeSecretPlaintextIsBounded(t *testing.T) {
	t.Parallel()
	value, err := normalizeSecretPlaintext("  secret  ")
	if err != nil {
		t.Fatalf("expected valid plaintext, got %v", err)
	}
	if value != "  secret  " {
		t.Fatalf("unexpected normalized plaintext %q", value)
	}
	if _, err := normalizeSecretPlaintext(strings.Repeat("s", maxSecretPlaintextBytes+1)); !errors.Is(err, errSecretPlaintextTooLarge) {
		t.Fatalf("expected plaintext size rejection, got %v", err)
	}
}

func TestNormalizeSecretCiphertextIsBounded(t *testing.T) {
	t.Parallel()
	value, err := normalizeSecretCiphertext("  Y2lwaGVy  ")
	if err != nil {
		t.Fatalf("expected valid ciphertext, got %v", err)
	}
	if value != "Y2lwaGVy" {
		t.Fatalf("unexpected normalized ciphertext %q", value)
	}
	if _, err := normalizeSecretCiphertext(strings.Repeat("c", maxSecretCiphertextBytes+1)); !errors.Is(err, errSecretCiphertextTooLarge) {
		t.Fatalf("expected ciphertext size rejection, got %v", err)
	}
}
