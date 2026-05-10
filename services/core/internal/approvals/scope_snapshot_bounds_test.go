package approvals

import (
	"errors"
	"strings"
	"testing"
)

func TestMarshalScopeSnapshotRejectsOversizeSnapshot(t *testing.T) {
	t.Parallel()
	if body, err := marshalScopeSnapshot(map[string]any{"laneId": "default"}); err != nil || len(body) == 0 {
		t.Fatalf("expected valid scope snapshot, got len=%d err=%v", len(body), err)
	}
	if _, err := marshalScopeSnapshot(map[string]any{"large": strings.Repeat("x", maxApprovalScopeSnapshotBytes+1)}); !errors.Is(err, errApprovalScopeSnapshotTooLarge) {
		t.Fatalf("expected scope snapshot size rejection, got %v", err)
	}
}
