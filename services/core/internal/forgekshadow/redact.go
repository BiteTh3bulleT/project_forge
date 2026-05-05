package forgekshadow

import (
	"fmt"
	"strings"
)

var unsafeMetadataTerms = []string{
	"api_key",
	"secret",
	"token",
	"password",
	"private_key",
	"bearer",
	"plaintext",
	"credential",
}

func safeMetadata(in map[string]any) (map[string]any, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		if containsUnsafeTerm(trimmedKey) {
			return nil, fmt.Errorf("%w: key %q", ErrUnsafeMetadata, trimmedKey)
		}
		switch v := value.(type) {
		case string:
			if containsUnsafeTerm(v) {
				return nil, fmt.Errorf("%w: value for %q", ErrUnsafeMetadata, trimmedKey)
			}
			out[trimmedKey] = strings.TrimSpace(v)
		case fmt.Stringer:
			text := v.String()
			if containsUnsafeTerm(text) {
				return nil, fmt.Errorf("%w: value for %q", ErrUnsafeMetadata, trimmedKey)
			}
			out[trimmedKey] = strings.TrimSpace(text)
		case nil:
			continue
		default:
			out[trimmedKey] = v
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func containsUnsafeTerm(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}
	for _, term := range unsafeMetadataTerms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	return false
}
