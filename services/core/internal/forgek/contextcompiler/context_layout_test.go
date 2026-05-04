package contextcompiler

import (
	"errors"
	"testing"
)

func TestPromptLayoutValidationAndOrdering(t *testing.T) {
	layout := DefaultPromptLayout("workspace-a", "", "", "", testBlockInput().CreatedAt, nil)
	if err := ValidateLayout(layout); err != nil {
		t.Fatalf("default layout invalid: %v", err)
	}
	if layout.BlockOrder[len(layout.BlockOrder)-1] != BlockUserMessage {
		t.Fatalf("user message is not last: %#v", layout.BlockOrder)
	}
	invalid := layout
	invalid.BlockOrder = []ContextBlockType{BlockUserMessage, BlockCaseSummary}
	if err := ValidateLayout(invalid); !errors.Is(err, ErrInvalidPromptLayout) {
		t.Fatalf("expected invalid layout, got %v", err)
	}
}
