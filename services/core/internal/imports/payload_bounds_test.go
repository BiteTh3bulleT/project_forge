package imports

import (
	"errors"
	"strings"
	"testing"
)

func TestMarshalImportPayloadRejectsOversizePayload(t *testing.T) {
	t.Parallel()
	if body, err := marshalImportPayload("outputRefs", []string{"artifact:1"}); err != nil || len(body) == 0 {
		t.Fatalf("expected valid import payload, got len=%d err=%v", len(body), err)
	}
	if _, err := marshalImportPayload("evaluation", map[string]any{"large": strings.Repeat("x", maxImportPayloadBytes+1)}); !errors.Is(err, errImportPayloadTooLarge) {
		t.Fatalf("expected import payload size rejection, got %v", err)
	}
}
