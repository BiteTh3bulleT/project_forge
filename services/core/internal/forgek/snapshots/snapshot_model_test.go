package snapshots

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func testTime() time.Time {
	return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
}

func validSnapshotInput() SnapshotInput {
	return SnapshotInput{
		SnapshotID:            "snapshot-1",
		SnapshotType:          SnapshotTypeSemanticSnapshot,
		WorkspaceID:           "workspace-a",
		CaseID:                "case-1",
		SourceObjectRefs:      []string{"object-b", "object-a", "object-a"},
		SourceRefs:            []string{"doc-b", "doc-a", "doc-a"},
		SemanticOperationRefs: []string{"operation-b", "operation-a"},
		DerivedObjectRefs:     []string{"derived-a"},
		Summary:               "semantic shape summary",
		CreatedBy:             "operator",
		CreatedAt:             testTime(),
		Metadata:              map[string]any{"policy_version": "v1", "tags": []string{"semantic"}},
	}
}

func TestSnapshotModelValidatesNormalizesAndSerializes(t *testing.T) {
	snapshot, err := NewSnapshot(validSnapshotInput())
	if err != nil {
		t.Fatalf("NewSnapshot failed: %v", err)
	}
	if snapshot.Status != StatusDraft {
		t.Fatalf("expected draft snapshot, got %s", snapshot.Status)
	}
	if !reflect.DeepEqual(snapshot.SourceObjectRefs, []string{"object-a", "object-b"}) {
		t.Fatalf("source object refs not normalized: %#v", snapshot.SourceObjectRefs)
	}
	if !reflect.DeepEqual(snapshot.SourceRefs, []string{"doc-a", "doc-b"}) {
		t.Fatalf("source refs not normalized: %#v", snapshot.SourceRefs)
	}
	if snapshot.ShapeHash == "" || snapshot.SourceHash == "" {
		t.Fatalf("hashes not set: %#v", snapshot)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("snapshot must serialize: %v", err)
	}
	if !json.Valid(encoded) {
		t.Fatalf("snapshot serialization is invalid JSON: %s", encoded)
	}
}

func TestSnapshotModelRejectsInvalidInputs(t *testing.T) {
	input := validSnapshotInput()
	input.SnapshotType = "UNKNOWN"
	if _, err := NewSnapshot(input); !errors.Is(err, ErrInvalidSnapshotType) {
		t.Fatalf("expected invalid type error, got %v", err)
	}

	input = validSnapshotInput()
	input.WorkspaceID = ""
	if _, err := NewSnapshot(input); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("expected invalid snapshot error for missing workspace, got %v", err)
	}

	input = validSnapshotInput()
	input.SourceObjectRefs = nil
	input.SourceRefs = nil
	input.SemanticOperationRefs = nil
	input.DerivedObjectRefs = nil
	if _, err := NewSnapshot(input); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("expected ref requirement error, got %v", err)
	}

	input = validSnapshotInput()
	input.Metadata = map[string]any{"raw_content": "full canonical text"}
	if _, err := NewSnapshot(input); !errors.Is(err, ErrSnapshotContentRejected) {
		t.Fatalf("expected raw content rejection, got %v", err)
	}
}

func TestSnapshotShapeHashIsStableForNormalizedShape(t *testing.T) {
	left, err := NewSnapshot(validSnapshotInput())
	if err != nil {
		t.Fatalf("left snapshot: %v", err)
	}
	input := validSnapshotInput()
	input.SnapshotID = "snapshot-2"
	input.CreatedAt = input.CreatedAt.Add(6 * time.Hour)
	input.SourceObjectRefs = []string{"object-a", "object-b"}
	input.SourceRefs = []string{"doc-a", "doc-b"}
	input.SemanticOperationRefs = []string{"operation-a", "operation-b"}
	right, err := NewSnapshot(input)
	if err != nil {
		t.Fatalf("right snapshot: %v", err)
	}
	if left.ShapeHash != right.ShapeHash {
		t.Fatalf("shape hash should ignore ids and created_at: left=%s right=%s", left.ShapeHash, right.ShapeHash)
	}

	input = validSnapshotInput()
	input.SnapshotID = "snapshot-3"
	input.AdmittedObjectRefs = []string{"exhibit-a"}
	changed, err := NewSnapshot(input)
	if err != nil {
		t.Fatalf("changed snapshot: %v", err)
	}
	if left.ShapeHash == changed.ShapeHash {
		t.Fatal("shape hash did not change when semantic shape changed")
	}
}

func TestSealedSnapshotStartsImmutableByLifecycleStatus(t *testing.T) {
	input := validSnapshotInput()
	input.Seal = true
	snapshot, err := NewSnapshot(input)
	if err != nil {
		t.Fatalf("sealed snapshot: %v", err)
	}
	if snapshot.Status != StatusSealed || snapshot.SealedAt == nil {
		t.Fatalf("expected sealed lifecycle, got %#v", snapshot)
	}
	if snapshot.IsCanonicalTruth() {
		t.Fatal("snapshot claimed canonical truth")
	}
}
