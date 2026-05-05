package forgekshadow

import (
	"fmt"
	"strings"
)

const maxMetadataStringLength = 512

var unsafeMetadataTerms = []string{
	"api_key",
	"secret",
	"token",
	"password",
	"private_key",
	"bearer",
	"plaintext",
	"credential",
	"authorization",
	"cookie",
	"session",
}

var rawContentMetadataKeys = []string{
	"body",
	"request_body",
	"response_body",
	"prompt",
	"completion",
	"model_output",
	"content",
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
		if isRawContentKey(trimmedKey) {
			return nil, fmt.Errorf("%w: raw content key %q", ErrUnsafeMetadata, trimmedKey)
		}
		switch v := value.(type) {
		case string:
			text := strings.TrimSpace(v)
			if containsUnsafeTerm(text) || len(text) > maxMetadataStringLength {
				return nil, fmt.Errorf("%w: value for %q", ErrUnsafeMetadata, trimmedKey)
			}
			out[trimmedKey] = text
		case fmt.Stringer:
			text := strings.TrimSpace(v.String())
			if containsUnsafeTerm(text) || len(text) > maxMetadataStringLength {
				return nil, fmt.Errorf("%w: value for %q", ErrUnsafeMetadata, trimmedKey)
			}
			out[trimmedKey] = text
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

func isRawContentKey(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	for _, key := range rawContentMetadataKeys {
		if normalized == key {
			return true
		}
	}
	return false
}
