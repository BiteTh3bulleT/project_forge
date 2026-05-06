package forgekshadow

import (
	"fmt"
	"math"
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
	"auth",
	"cookie",
	"set_cookie",
	"session",
	"x_api_key",
	"jwt",
	"refresh_token",
	"access_token",
}

var rawContentMetadataKeys = []string{
	"body",
	"request_body",
	"response_body",
	"raw_content",
	"prompt",
	"completion",
	"assistant_response",
	"system_prompt",
	"model_output",
	"message",
	"message_body",
	"tool_output",
	"tool_payload",
	"retrieval_content",
	"retrieval_query",
	"retrieval_result_body",
	"memory_content",
	"source_chunk",
	"source_text",
	"source_content",
	"chunk",
	"chunk_text",
	"document",
	"document_content",
	"document_text",
	"file_contents",
	"file_content",
	"search_query",
	"query_text",
	"search_snippet",
	"snippet",
	"embedding",
	"embeddings",
	"embedding_input",
	"embedding_vector",
	"vector",
	"vectors",
	"request_payload",
	"content",
	"query",
	"raw_query",
	"query_string",
	"request_uri",
	"url",
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
		case float32:
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return nil, fmt.Errorf("%w: non-finite value for %q", ErrUnsafeMetadata, trimmedKey)
			}
			out[trimmedKey] = v
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, fmt.Errorf("%w: non-finite value for %q", ErrUnsafeMetadata, trimmedKey)
			}
			out[trimmedKey] = v
		case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			out[trimmedKey] = v
		default:
			return nil, fmt.Errorf("%w: non-deterministic value for %q", ErrUnsafeMetadata, trimmedKey)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func containsUnsafeTerm(value string) bool {
	normalized := normalizeMetadataToken(value)
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

func containsRawContentTerm(value string) bool {
	normalized := normalizeMetadataToken(value)
	if normalized == "" {
		return false
	}
	for _, key := range rawContentMetadataKeys {
		if normalized == key {
			return true
		}
		if strings.Contains(key, "_") && strings.Contains(normalized, key) {
			return true
		}
	}
	return false
}

func isRawContentKey(value string) bool {
	normalized := normalizeMetadataToken(value)
	for _, key := range rawContentMetadataKeys {
		if normalized == key {
			return true
		}
	}
	return false
}

func normalizeMetadataToken(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	return normalized
}
