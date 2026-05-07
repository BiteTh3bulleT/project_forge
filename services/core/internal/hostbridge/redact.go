package hostbridge

import (
	"net/url"
	"strings"
)

var sensitiveBootKeys = []string{
	"api_key",
	"apikey",
	"auth",
	"authorization",
	"bearer",
	"credential",
	"credentials",
	"key",
	"password",
	"passwd",
	"private_key",
	"secret",
	"session",
	"token",
}

func redactBootParameters(raw string) ([]string, []RedactionRecord) {
	fields := strings.Fields(strings.TrimSpace(raw))
	out := make([]string, 0, len(fields))
	redactions := []RedactionRecord{}
	for _, field := range fields {
		key, value, hasValue := strings.Cut(field, "=")
		reason := bootRedactionReason(key, value, hasValue)
		if reason == "" {
			out = append(out, field)
			continue
		}
		redactions = append(redactions, RedactionRecord{
			Source: "proc.cmdline",
			Field:  key,
			Reason: reason,
		})
		if hasValue {
			out = append(out, key+"=[REDACTED]")
		} else {
			out = append(out, "[REDACTED]")
		}
	}
	return out, redactions
}

func bootRedactionReason(key, value string, hasValue bool) string {
	normalizedKey := normalizeSensitiveToken(key)
	for _, term := range sensitiveBootKeys {
		if strings.Contains(normalizedKey, term) {
			return "sensitive_key"
		}
	}
	if !hasValue {
		return ""
	}
	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(normalizedValue, "bearer ") {
		return "bearer_value"
	}
	if urlHasEmbeddedAuth(value) {
		return "embedded_url_auth"
	}
	for _, term := range sensitiveBootKeys {
		if strings.Contains(normalizeSensitiveToken(value), term+"=") {
			return "sensitive_value"
		}
	}
	return ""
}

func urlHasEmbeddedAuth(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.User == nil {
		return false
	}
	return parsed.Scheme != "" && parsed.Host != ""
}

func normalizeSensitiveToken(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return normalized
}
