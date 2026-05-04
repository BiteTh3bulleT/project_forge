package snapshots

import "testing"

func TestSnapshotStoresShapeNotCanonicalContent(t *testing.T) {
	input := validSnapshotInput()
	input.SnapshotType = SnapshotTypeCaseSnapshot
	input.SubmittedObjectRefs = []string{"exhibit-submitted"}
	input.AdmittedObjectRefs = []string{"exhibit-admitted"}
	input.RejectedObjectRefs = []string{"exhibit-rejected"}
	input.PalaceRouteRefs = []string{"route-1"}
	input.KVManifestRefs = []string{"kv-manifest-future"}
	snapshot, err := NewSnapshot(input)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.IsCanonicalTruth() {
		t.Fatal("snapshot claimed canonical truth")
	}
	if !contains(snapshot.AllRefs(), "exhibit-admitted") ||
		!contains(snapshot.AllRefs(), "exhibit-rejected") ||
		!contains(snapshot.AllRefs(), "route-1") ||
		!contains(snapshot.AllRefs(), "kv-manifest-future") {
		t.Fatalf("snapshot did not preserve cited shape refs: %#v", snapshot.AllRefs())
	}
	if snapshot.ShapeHash == snapshot.SnapshotID {
		t.Fatal("shape hash is based on snapshot id")
	}
}
