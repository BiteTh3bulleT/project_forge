package contextcompiler

import (
	"strings"
	"testing"
)

func TestCanonicalSerializationNormalizesWhitespaceRefsAndMapOrder(t *testing.T) {
	input := testBlockInput()
	input.Metadata = map[string]any{"b": "two", "a": "one"}
	left, err := NewContextBlock(input)
	if err != nil {
		t.Fatalf("left block: %v", err)
	}
	input.Metadata = map[string]any{"a": "one", "b": "two"}
	input.ContentSummary = "admitted\n\nsemantic\tshape"
	right, err := NewContextBlock(input)
	if err != nil {
		t.Fatalf("right block: %v", err)
	}
	if left.CanonicalText != right.CanonicalText {
		t.Fatalf("canonical serialization changed with map/whitespace order:\n%s\n---\n%s", left.CanonicalText, right.CanonicalText)
	}
	for _, marker := range []string{
		"[BLOCK: ADMITTED_EVIDENCE]",
		"workspace_id: workspace-a",
		"source_object_refs:",
		"- semantic-a",
		"admitted_exhibit_refs:",
		"summary:",
	} {
		if !strings.Contains(left.CanonicalText, marker) {
			t.Fatalf("canonical text missing %q:\n%s", marker, left.CanonicalText)
		}
	}
}
