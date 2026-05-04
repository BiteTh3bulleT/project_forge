package snapshots

import (
	"reflect"
	"testing"
)

func TestSnapshotDiffDetectsRefsAndIsDeterministic(t *testing.T) {
	leftInput := validSnapshotInput()
	leftInput.SnapshotID = "left"
	leftInput.SourceObjectRefs = []string{"object-a", "object-b"}
	leftInput.SemanticOperationRefs = []string{"operation-a"}
	left, err := NewSnapshot(leftInput)
	if err != nil {
		t.Fatalf("left snapshot: %v", err)
	}
	rightInput := validSnapshotInput()
	rightInput.SnapshotID = "right"
	rightInput.SourceObjectRefs = []string{"object-b", "object-c"}
	rightInput.SemanticOperationRefs = []string{"operation-a", "operation-b"}
	right, err := NewSnapshot(rightInput)
	if err != nil {
		t.Fatalf("right snapshot: %v", err)
	}

	first, err := DiffSnapshots("diff-1", left, right, testTime(), map[string]any{"purpose": "test"})
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}
	second, err := DiffSnapshots("diff-1", left, right, testTime(), map[string]any{"purpose": "test"})
	if err != nil {
		t.Fatalf("second diff failed: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("diff is not deterministic:\n%#v\n%#v", first, second)
	}
	if !reflect.DeepEqual(first.AddedRefs, []string{"object-c", "operation-b"}) {
		t.Fatalf("unexpected added refs: %#v", first.AddedRefs)
	}
	if !reflect.DeepEqual(first.RemovedRefs, []string{"object-a"}) {
		t.Fatalf("unexpected removed refs: %#v", first.RemovedRefs)
	}
	if !reflect.DeepEqual(first.UnchangedRefs, []string{"derived-a", "doc-a", "doc-b", "object-b", "operation-a"}) {
		t.Fatalf("unexpected unchanged refs: %#v", first.UnchangedRefs)
	}
	if len(first.ChangedFields) == 0 {
		t.Fatal("expected changed fields")
	}
	if left.ShapeHash != ShapeHash(left) || right.ShapeHash != ShapeHash(right) {
		t.Fatal("diff mutated snapshots")
	}
}
