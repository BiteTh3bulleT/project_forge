package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeCapabilitySearchQueryIsBounded(t *testing.T) {
	t.Parallel()
	query, err := normalizeCapabilitySearchQuery("  needle  ", "code.search_code")
	if err != nil {
		t.Fatalf("expected valid search query, got %v", err)
	}
	if query != "needle" {
		t.Fatalf("unexpected normalized query %q", query)
	}
	if _, err := normalizeCapabilitySearchQuery(strings.Repeat("q", maxCapabilitySearchQueryBytes+1), "code.search_code"); !errors.Is(err, errCapabilitySearchQueryTooLarge) {
		t.Fatalf("expected search query size rejection, got %v", err)
	}
}
