package automation

import (
	"errors"
	"strings"
	"testing"
)

func TestMarshalRulePayloadRejectsOversizePayload(t *testing.T) {
	t.Parallel()
	if body, err := marshalRulePayload("condition", map[string]any{"always": true}); err != nil || len(body) == 0 {
		t.Fatalf("expected valid rule payload, got len=%d err=%v", len(body), err)
	}
	if _, err := marshalRulePayload("action", map[string]any{"large": strings.Repeat("x", maxRulePayloadBytes+1)}); !errors.Is(err, errRulePayloadTooLarge) {
		t.Fatalf("expected rule payload size rejection, got %v", err)
	}
}
