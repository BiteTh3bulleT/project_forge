package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeIdentityTokenInputIsBounded(t *testing.T) {
	t.Parallel()
	token, err := normalizeIdentityTokenInput("  token-value  ")
	if err != nil {
		t.Fatalf("expected valid token input, got %v", err)
	}
	if token != "token-value" {
		t.Fatalf("unexpected normalized token %q", token)
	}
	if _, err := normalizeIdentityTokenInput(strings.Repeat("t", maxIdentityTokenInputBytes+1)); !errors.Is(err, errIdentityTokenInputTooLarge) {
		t.Fatalf("expected token input size rejection, got %v", err)
	}
}

func TestNormalizeIdentityTokenIDIsBounded(t *testing.T) {
	t.Parallel()
	tokenID, err := normalizeIdentityTokenID("  token-id  ")
	if err != nil {
		t.Fatalf("expected valid token id, got %v", err)
	}
	if tokenID != "token-id" {
		t.Fatalf("unexpected normalized token id %q", tokenID)
	}
	if _, err := normalizeIdentityTokenID(strings.Repeat("i", maxIdentityTokenIDBytes+1)); !errors.Is(err, errIdentityTokenIDTooLarge) {
		t.Fatalf("expected token id size rejection, got %v", err)
	}
}
