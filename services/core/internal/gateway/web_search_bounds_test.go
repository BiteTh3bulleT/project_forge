package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeWebSearchQueryIsBounded(t *testing.T) {
	t.Parallel()
	query, err := normalizeWebSearchQuery("  forge os  ")
	if err != nil {
		t.Fatalf("expected valid web search query, got %v", err)
	}
	if query != "forge os" {
		t.Fatalf("unexpected normalized query %q", query)
	}
	if _, err := normalizeWebSearchQuery(strings.Repeat("x", maxWebSearchQueryBytes+1)); !errors.Is(err, errWebSearchQueryTooLarge) {
		t.Fatalf("expected web search query size rejection, got %v", err)
	}
}
