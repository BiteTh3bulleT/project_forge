package contextcompiler

import (
	"testing"

	"forge/projectforge/services/core/internal/forgek/snapshots"
)

func TestCompileFromSnapshotAndRestoreSeedPreservesRefs(t *testing.T) {
	service := NewService()
	snapshot, err := snapshots.NewSnapshot(snapshots.SnapshotInput{
		SnapshotID:            "snapshot-a",
		SnapshotType:          snapshots.SnapshotTypeContextRestoreSnapshot,
		WorkspaceID:           "workspace-a",
		CaseID:                "case-a",
		SourceObjectRefs:      []string{"case-a"},
		SourceRefs:            []string{"doc-a"},
		AdmittedObjectRefs:    []string{"admitted-a"},
		RejectedObjectRefs:    []string{"rejected-a"},
		PalaceRouteRefs:       []string{"route-a"},
		SemanticOperationRefs: []string{"operation-a"},
		DerivedObjectRefs:     []string{"derived-a"},
		Summary:               "restore shape",
		CreatedBy:             "operator",
		CreatedAt:             testBlockInput().CreatedAt,
	})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	fromSnapshot, err := service.CompileFromSnapshot(snapshot, ContextCompileRequest{
		RequestID:                      "request-snapshot",
		BundleID:                       "bundle-snapshot",
		WorkspaceID:                    "workspace-a",
		IncludeRejectedEvidenceSummary: true,
		CreatedBy:                      "operator",
		CreatedAt:                      testBlockInput().CreatedAt,
	})
	if err != nil {
		t.Fatalf("compile from snapshot: %v", err)
	}
	if fromSnapshot.SnapshotID != snapshot.SnapshotID ||
		!containsString(fromSnapshot.Bundle.SourceRefs, "admitted-a") ||
		!containsString(fromSnapshot.Bundle.SourceRefs, "route-a") ||
		!containsString(fromSnapshot.Bundle.SourceRefs, "operation-a") {
		t.Fatalf("snapshot refs not preserved: %#v", fromSnapshot.Bundle.SourceRefs)
	}

	seed, err := snapshots.NewRestoreSeed("restore-seed-a", snapshot, testBlockInput().CreatedAt, nil)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	fromSeed, err := service.CompileFromRestoreSeed(seed, ContextCompileRequest{
		RequestID:   "request-seed",
		BundleID:    "bundle-seed",
		WorkspaceID: "workspace-a",
		CreatedBy:   "operator",
		CreatedAt:   testBlockInput().CreatedAt,
	})
	if err != nil {
		t.Fatalf("compile from restore seed: %v", err)
	}
	if fromSeed.RestoreSeedID != seed.RestoreSeedID || !hasBlockType(fromSeed.Blocks, BlockSnapshotRestoreSeed) {
		t.Fatalf("restore seed refs not compiled: %#v", fromSeed)
	}
	if seed.IsCanonicalTruth() || seed.IsContextBlock() {
		t.Fatal("restore seed changed authority")
	}
}
