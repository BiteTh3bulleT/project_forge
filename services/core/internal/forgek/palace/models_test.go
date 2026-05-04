package palace

import (
	"testing"
	"time"
)

func TestMemoryRoomPreservesWorkspaceAndTopology(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)

	room, err := NewMemoryRoom(MemoryRoomInput{
		RoomID:         "room-1",
		WorkspaceID:    "workspace-a",
		Name:           "architecture",
		Description:    "Architecture references",
		DomainTags:     []string{"architecture", "kernel"},
		AnchorRefs:     []string{"anchor-1"},
		LinkedRoomRefs: []string{"room-2"},
		CreatedAt:      now,
		Metadata:       map[string]any{"owner": "operator"},
	})
	if err != nil {
		t.Fatalf("NewMemoryRoom failed: %v", err)
	}

	if room.WorkspaceID != "workspace-a" || room.Name != "architecture" {
		t.Fatalf("room lost workspace or name: %#v", room)
	}
	if len(room.DomainTags) != 2 || room.DomainTags[0] != "architecture" {
		t.Fatalf("room lost domain tags: %#v", room.DomainTags)
	}
	if room.RouteStats.SuccessCount != 0 || room.RouteStats.RouteCount != 0 {
		t.Fatalf("new room should start with empty route stats: %#v", room.RouteStats)
	}

	clone := room.Clone()
	clone.DomainTags[0] = "tampered"
	if room.DomainTags[0] == "tampered" {
		t.Fatal("room clone exposed mutable domain tags")
	}
}

func TestMemoryAnchorPreservesRefsAndProvenance(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)

	anchor, err := NewMemoryAnchor(MemoryAnchorInput{
		AnchorID:    "anchor-1",
		WorkspaceID: "workspace-a",
		RoomID:      "room-1",
		Label:       "courthouse doctrine",
		ObjectRefs:  []string{"exhibit-1"},
		Keywords:    []string{"courthouse", "admission"},
		Tags:        []string{"evidence"},
		SourceRefs:  []string{"doc:memory-palace"},
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatalf("NewMemoryAnchor failed: %v", err)
	}

	if anchor.RoomID != "room-1" || anchor.ObjectRefs[0] != "exhibit-1" || anchor.SourceRefs[0] != "doc:memory-palace" {
		t.Fatalf("anchor lost refs or provenance: %#v", anchor)
	}

	clone := anchor.Clone()
	clone.ObjectRefs[0] = "tampered"
	if anchor.ObjectRefs[0] == "tampered" {
		t.Fatal("anchor clone exposed mutable object refs")
	}
}

func TestPalaceRouteAndCandidatePreserveRetrievalTrace(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	candidate := CandidateObject{
		CandidateID:      "candidate-1",
		WorkspaceID:      "workspace-a",
		SourceObjectID:   "exhibit-1",
		SourceType:       SourceTypeKernelObject,
		SourceRefs:       []string{"exhibit-1"},
		AnchorID:         "anchor-1",
		RoomID:           "room-1",
		RelevanceScore:   3.5,
		RetrievalReason:  "keyword overlap: courthouse",
		CandidateSummary: "Courthouse admission doctrine",
		CreatedAt:        now,
	}

	route, err := NewPalaceRoute(PalaceRouteInput{
		RouteID:          "route-1",
		CaseID:           "case-1",
		WorkspaceID:      "workspace-a",
		QueryText:        "courthouse admission",
		StartRoomID:      "room-1",
		VisitedRoomIDs:   []string{"room-1", "room-2"},
		AnchorRefs:       []string{"anchor-1"},
		CandidateObjects: []CandidateObject{candidate},
		RouteScore:       3.5,
		RouteStrategy:    "keyword_tag_overlap",
		CreatedBy:        "operator",
		CreatedAt:        now,
	})
	if err != nil {
		t.Fatalf("NewPalaceRoute failed: %v", err)
	}

	if route.CaseID != "case-1" || route.CandidateObjects[0].SourceObjectID != "exhibit-1" {
		t.Fatalf("route lost retrieval trace: %#v", route)
	}
	if route.CandidateObjects[0].IsExhibit() {
		t.Fatal("candidate object should not be an exhibit")
	}
	if route.CandidateObjects[0].IsAdmittedEvidence() {
		t.Fatal("candidate object should not be admitted evidence")
	}
}

func TestDeterministicRouteScoringRanksKeywordAndTagMatches(t *testing.T) {
	query := RouteQuery{
		WorkspaceID: "workspace-a",
		QueryText:   "kernel courthouse evidence",
		Tags:        []string{"architecture"},
	}
	matching := MemoryAnchor{
		WorkspaceID: "workspace-a",
		RoomID:      "room-1",
		Label:       "Courthouse evidence",
		ObjectRefs:  []string{"exhibit-1"},
		Keywords:    []string{"courthouse", "evidence"},
		Tags:        []string{"architecture"},
	}
	unrelated := MemoryAnchor{
		WorkspaceID: "workspace-a",
		RoomID:      "room-1",
		Label:       "Runtime driver",
		ObjectRefs:  []string{"exhibit-2"},
		Keywords:    []string{"runtime"},
		Tags:        []string{"driver"},
	}
	otherWorkspace := matching
	otherWorkspace.WorkspaceID = "workspace-b"

	high := ScoreAnchor(query, matching, "room-1", RoomRouteStats{SuccessCount: 2})
	low := ScoreAnchor(query, unrelated, "room-1", RoomRouteStats{})
	blocked := ScoreAnchor(query, otherWorkspace, "room-1", RoomRouteStats{})
	again := ScoreAnchor(query, matching, "room-1", RoomRouteStats{SuccessCount: 2})

	if high <= low {
		t.Fatalf("expected matching score %v to exceed unrelated score %v", high, low)
	}
	if blocked != 0 {
		t.Fatalf("cross-workspace score should be zero, got %v", blocked)
	}
	if high != again {
		t.Fatalf("scoring is not deterministic: %v then %v", high, again)
	}
}
