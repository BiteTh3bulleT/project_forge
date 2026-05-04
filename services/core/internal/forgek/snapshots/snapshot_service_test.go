package snapshots

import (
	"errors"
	"testing"
)

func TestSnapshotServiceLifecycleAndFilters(t *testing.T) {
	service := NewService()
	first, err := service.CreateSnapshot(validSnapshotInput())
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	secondInput := validSnapshotInput()
	secondInput.SnapshotID = "snapshot-2"
	secondInput.SnapshotType = SnapshotTypeCaseSnapshot
	secondInput.CaseID = "case-2"
	secondInput.SourceObjectRefs = []string{"case-2"}
	second, err := service.CreateSnapshot(secondInput)
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if got := service.ListSnapshots(ListFilter{WorkspaceID: "workspace-a"}); len(got) != 2 {
		t.Fatalf("expected two workspace snapshots, got %d", len(got))
	}
	if got := service.ListSnapshots(ListFilter{WorkspaceID: "workspace-a", CaseID: "case-2"}); len(got) != 1 || got[0].SnapshotID != second.SnapshotID {
		t.Fatalf("case filter failed: %#v", got)
	}
	if got := service.ListSnapshots(ListFilter{WorkspaceID: "workspace-a", SnapshotType: SnapshotTypeSemanticSnapshot}); len(got) != 1 || got[0].SnapshotID != first.SnapshotID {
		t.Fatalf("type filter failed: %#v", got)
	}

	sealed, err := service.SealSnapshot(first.SnapshotID, testTime(), "event-seal")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if sealed.Status != StatusSealed || sealed.SealedAt == nil || len(sealed.JournalRefs) != 1 {
		t.Fatalf("seal did not update lifecycle: %#v", sealed)
	}
	if _, err := service.SealSnapshot(first.SnapshotID, testTime(), "event-reseal"); !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected reseal rejection, got %v", err)
	}

	result, err := service.SupersedeSnapshot(first.SnapshotID, second.SnapshotID, testTime(), "event-supersede")
	if err != nil {
		t.Fatalf("supersede: %v", err)
	}
	if result.Superseded.Status != StatusSuperseded || result.Superseded.SupersededBy != second.SnapshotID {
		t.Fatalf("old snapshot not superseded: %#v", result)
	}
	if !contains(result.Superseding.Supersedes, first.SnapshotID) {
		t.Fatalf("new snapshot did not cite superseded snapshot: %#v", result.Superseding)
	}
	if _, ok := service.GetSnapshot(first.SnapshotID); !ok {
		t.Fatal("superseded snapshot is not inspectable")
	}

	thirdInput := validSnapshotInput()
	thirdInput.SnapshotID = "snapshot-3"
	thirdInput.CaseID = "case-3"
	thirdInput.SourceObjectRefs = []string{"case-3"}
	third, err := service.CreateSnapshot(thirdInput)
	if err != nil {
		t.Fatalf("create third: %v", err)
	}
	expired, err := service.ExpireSnapshot(third.SnapshotID, testTime(), "event-expire")
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if expired.Status != StatusExpired || expired.ExpiredAt == nil {
		t.Fatalf("expire did not update lifecycle: %#v", expired)
	}
	if _, ok := service.GetSnapshot(third.SnapshotID); !ok {
		t.Fatal("expired snapshot is not inspectable")
	}
}

func TestSnapshotServiceDiffAndRestoreSeed(t *testing.T) {
	service := NewService()
	left, err := service.CreateSnapshot(validSnapshotInput())
	if err != nil {
		t.Fatalf("create left: %v", err)
	}
	rightInput := validSnapshotInput()
	rightInput.SnapshotID = "snapshot-2"
	rightInput.SourceObjectRefs = []string{"object-a", "object-c"}
	right, err := service.CreateSnapshot(rightInput)
	if err != nil {
		t.Fatalf("create right: %v", err)
	}
	diff, err := service.DiffSnapshots(left.SnapshotID, right.SnapshotID, "diff-1", testTime(), nil)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if !contains(diff.AddedRefs, "object-c") || !contains(diff.RemovedRefs, "object-b") {
		t.Fatalf("unexpected diff refs: %#v", diff)
	}
	seed, updated, err := service.CreateRestoreSeed(left.SnapshotID, "seed-1", testTime(), nil, "event-seed")
	if err != nil {
		t.Fatalf("restore seed: %v", err)
	}
	if seed.SnapshotID != left.SnapshotID || updated.Status != StatusRestoreSeedCreated {
		t.Fatalf("restore seed did not update shape lifecycle: seed=%#v snapshot=%#v", seed, updated)
	}
	if got, ok := service.GetRestoreSeed(seed.RestoreSeedID); !ok || got.SourceShapeHash != left.ShapeHash {
		t.Fatalf("restore seed not inspectable: %#v ok=%v", got, ok)
	}
}
