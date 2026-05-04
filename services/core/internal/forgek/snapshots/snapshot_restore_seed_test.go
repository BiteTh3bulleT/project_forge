package snapshots

import "testing"

func TestRestoreSeedCitesSnapshotAndDoesNotBecomeContextOrTruth(t *testing.T) {
	input := validSnapshotInput()
	input.SnapshotType = SnapshotTypeContextRestoreSnapshot
	input.AdmittedObjectRefs = []string{"exhibit-a"}
	input.RejectedObjectRefs = []string{"exhibit-b"}
	input.ContradictionRefs = []string{"contradiction-a"}
	snapshot, err := NewSnapshot(input)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	seed, err := NewRestoreSeed("seed-1", snapshot, testTime(), map[string]any{"reason": "resume"})
	if err != nil {
		t.Fatalf("restore seed: %v", err)
	}
	if seed.SnapshotID != snapshot.SnapshotID || seed.SourceShapeHash != snapshot.ShapeHash {
		t.Fatalf("restore seed lost snapshot citation: %#v", seed)
	}
	if !contains(seed.RecommendedSourceRefs, "exhibit-a") || !contains(seed.RecommendedSourceRefs, "exhibit-b") {
		t.Fatalf("restore seed did not preserve source refs: %#v", seed.RecommendedSourceRefs)
	}
	if !contains(seed.RecommendedOperationRefs, "operation-a") || !contains(seed.RecommendedOperationRefs, "contradiction-a") {
		t.Fatalf("restore seed did not preserve operation refs: %#v", seed.RecommendedOperationRefs)
	}
	if seed.IsCanonicalTruth() || seed.IsContextBlock() {
		t.Fatal("restore seed claimed truth or context authority")
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
