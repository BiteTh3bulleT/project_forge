package api

import (
	"strings"
	"testing"
)

func TestSanitizeLogTextRedactsSecretsAndURLDetails(t *testing.T) {
	raw := `request failed for https://user:pass@example.test/v1/models?token=abc123&x=1 with Authorization=Bearer abc.def token=abc123 api_key=secret`

	got := sanitizeLogText(raw)

	for _, forbidden := range []string{"user:pass", "token=abc123", "api_key=secret", "abc.def", "?token=", "&x=1"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("sanitized log text leaked %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "https://example.test/v1/models") {
		t.Fatalf("sanitized log text removed safe URL origin/path: %q", got)
	}
}
