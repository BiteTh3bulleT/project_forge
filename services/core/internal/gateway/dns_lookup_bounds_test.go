package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeDNSLookupHostIsBounded(t *testing.T) {
	t.Parallel()
	host, err := normalizeDNSLookupHost("  example.com  ")
	if err != nil {
		t.Fatalf("expected valid DNS host, got %v", err)
	}
	if host != "example.com" {
		t.Fatalf("unexpected normalized host %q", host)
	}
	if _, err := normalizeDNSLookupHost(strings.Repeat("x", maxDNSLookupHostBytes+1)); !errors.Is(err, errDNSLookupHostTooLarge) {
		t.Fatalf("expected DNS host size rejection, got %v", err)
	}
}
