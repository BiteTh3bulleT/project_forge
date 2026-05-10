package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeApprovalFingerprintValueHashesOversizeString(t *testing.T) {
	t.Parallel()
	large := strings.Repeat("x", maxApprovalFingerprintStringBytes+1)
	normalized := normalizeApprovalFingerprintValue(large)
	body, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("expected normalized value to marshal, got %v", err)
	}
	if strings.Contains(string(body), large) {
		t.Fatalf("expected oversized approval fingerprint value to omit raw string")
	}
	m, ok := normalized.(map[string]any)
	if !ok {
		t.Fatalf("expected oversized string summary map, got %#v", normalized)
	}
	if m["omitted"] != true || m["bytes"] != len(large) {
		t.Fatalf("expected omitted size summary, got %#v", m)
	}
	if !strings.HasPrefix(m["sha256"].(string), "sha256:") {
		t.Fatalf("expected sha256 digest, got %#v", m["sha256"])
	}
}

func TestNormalizeApprovalFingerprintValueBoundsLargeCollections(t *testing.T) {
	t.Parallel()
	input := map[string]any{}
	for i := 0; i < maxApprovalFingerprintCollectionItems+3; i++ {
		input[strings.Repeat("k", maxApprovalFingerprintFieldNameBytes+1)+string(rune('a'+i))] = i
	}
	normalized := normalizeApprovalFingerprintValue(input)
	m, ok := normalized.(map[string]any)
	if !ok {
		t.Fatalf("expected normalized map, got %#v", normalized)
	}
	if m["_truncated"] != true || m["_truncatedFieldNames"] != true {
		t.Fatalf("expected truncation markers, got %#v", m)
	}
	if len(m) > maxApprovalFingerprintCollectionItems+2 {
		t.Fatalf("expected bounded field count, got %d", len(m))
	}
}
