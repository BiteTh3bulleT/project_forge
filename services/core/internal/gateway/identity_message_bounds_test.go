package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeIdentityMessageIsBounded(t *testing.T) {
	t.Parallel()
	message, err := normalizeIdentityMessage("  payload  ")
	if err != nil {
		t.Fatalf("expected valid identity message, got %v", err)
	}
	if string(message) != "payload" {
		t.Fatalf("unexpected normalized message %q", string(message))
	}
	if _, err := normalizeIdentityMessage(strings.Repeat("x", maxIdentityMessageBytes+1)); !errors.Is(err, errIdentityMessageTooLarge) {
		t.Fatalf("expected identity message size rejection, got %v", err)
	}
}
