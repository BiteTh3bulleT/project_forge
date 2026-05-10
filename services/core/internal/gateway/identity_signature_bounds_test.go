package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeIdentitySignatureIsBounded(t *testing.T) {
	t.Parallel()
	signature, err := normalizeIdentitySignature("  dGVzdA==  ")
	if err != nil {
		t.Fatalf("expected valid identity signature, got %v", err)
	}
	if signature != "dGVzdA==" {
		t.Fatalf("unexpected normalized signature %q", signature)
	}
	if _, err := normalizeIdentitySignature(strings.Repeat("x", maxIdentitySignatureBytes+1)); !errors.Is(err, errIdentitySignatureTooLarge) {
		t.Fatalf("expected identity signature size rejection, got %v", err)
	}
}
