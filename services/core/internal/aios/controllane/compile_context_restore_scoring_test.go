package controllane

import (
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestSelectCompileContextRestoreCandidateDeterministicRanking(t *testing.T) {
	currentPacket := createTestContextPacketSnapshot("ctx-current", "ws-main", 1760004000000)
	currentPacket.Query = "summarize blockers"
	current := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        currentPacket,
		SnapshotID:    currentPacket.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-current",
		TraceID:       "trace-current",
		SyscallID:     "syscall-current",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)

	aligned := makeSnapshotCandidatePacket("ctx-candidate-aligned", currentPacket, 1760003900000, "restore")

	stalePacket := currentPacket
	stalePacket.ID = "ctx-candidate-stale"
	stalePacket.CreatedAt = 1750000000000
	stale := makeSnapshotCandidatePacket("ctx-candidate-stale", stalePacket, stalePacket.CreatedAt, "restore")

	conflictPacket := currentPacket
	conflictPacket.ID = "ctx-candidate-conflict"
	conflictPacket.CreatedAt = 1760003950000
	conflictPacket.LinkedNotes = append(conflictPacket.LinkedNotes, domain.SemanticLink{
		ID:         "link-conflict",
		Type:       domain.LinkContradicts,
		SourceID:   "note-a",
		TargetID:   "note-a",
		Scope:      conflictPacket.Scope,
		Confidence: 1.0,
		CreatedAt:  conflictPacket.CreatedAt,
	})
	conflict := makeSnapshotCandidatePacket("ctx-candidate-conflict", conflictPacket, conflictPacket.CreatedAt, "restore")

	selectionA := selectCompileContextRestoreCandidate(
		currentPacket.CreatedAt,
		current,
		[]domain.ContextPacket{stale, conflict, aligned},
		"restore",
		compileContextResumeHints{},
	)
	selectionB := selectCompileContextRestoreCandidate(
		currentPacket.CreatedAt,
		current,
		[]domain.ContextPacket{stale, conflict, aligned},
		"restore",
		compileContextResumeHints{},
	)

	if selectionA.Decision != "selected" || selectionA.SelectedIndex != 0 {
		t.Fatalf("expected selected top candidate, got decision=%q idx=%d", selectionA.Decision, selectionA.SelectedIndex)
	}
	if got := selectionA.selectedSnapshotID(); got != "ctx-candidate-aligned" {
		t.Fatalf("expected aligned snapshot to rank first, got %q", got)
	}
	if len(selectionA.Candidates) != 3 {
		t.Fatalf("expected 3 ranked candidates, got %d", len(selectionA.Candidates))
	}
	if selectionA.Candidates[1].Score.SnapshotID != selectionB.Candidates[1].Score.SnapshotID ||
		selectionA.Candidates[2].Score.SnapshotID != selectionB.Candidates[2].Score.SnapshotID {
		t.Fatalf("expected deterministic ranking order")
	}
	if selectionA.Candidates[2].Score.SnapshotID != "ctx-candidate-stale" {
		t.Fatalf("expected stale snapshot to rank last, got %q", selectionA.Candidates[2].Score.SnapshotID)
	}
}

func TestSelectCompileContextRestoreCandidateHeaderFirstFallback(t *testing.T) {
	currentPacket := createTestContextPacketSnapshot("ctx-current-header", "ws-main", 1760004100000)
	currentPacket.Query = "summarize blockers"
	current := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        currentPacket,
		SnapshotID:    currentPacket.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-current-header",
		TraceID:       "trace-current-header",
		SyscallID:     "syscall-current-header",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)

	candidate := makeSnapshotCandidatePacket("ctx-candidate-header-only", currentPacket, 1760004090000, "restore")
	if candidate.RestoreSnapshot == nil || candidate.RestoreSnapshot.Evidence == nil {
		t.Fatalf("expected candidate restore evidence")
	}
	candidate.RestoreSnapshot.Evidence = map[string]any{
		"header": candidate.RestoreSnapshot.Evidence["header"],
	}

	selection := selectCompileContextRestoreCandidate(
		currentPacket.CreatedAt,
		current,
		[]domain.ContextPacket{candidate},
		"restore",
		compileContextResumeHints{},
	)
	if len(selection.Candidates) != 1 {
		t.Fatalf("expected one candidate, got %d", len(selection.Candidates))
	}
	if !selection.Candidates[0].HeaderOnly {
		t.Fatalf("expected header-only candidate detection")
	}
	if selection.Decision != "selected" {
		t.Fatalf("expected header-first candidate to remain selectable, got %q", selection.Decision)
	}
}

func TestSelectCompileContextRestoreCandidateBelowThresholdFallsBack(t *testing.T) {
	currentPacket := createTestContextPacketSnapshot("ctx-current-threshold", "ws-main", 1760004200000)
	currentPacket.Query = "summarize blockers"
	current := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        currentPacket,
		SnapshotID:    currentPacket.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-current-threshold",
		TraceID:       "trace-current-threshold",
		SyscallID:     "syscall-current-threshold",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)
	candidatePacket := currentPacket
	candidatePacket.ID = "ctx-candidate-threshold"
	candidatePacket.CreatedAt = 1750000000000
	candidatePacket.LinkedNotes = append(candidatePacket.LinkedNotes, domain.SemanticLink{
		ID:         "link-threshold-conflict",
		Type:       domain.LinkContradicts,
		SourceID:   "note-a",
		TargetID:   "note-a",
		Scope:      candidatePacket.Scope,
		Confidence: 1.0,
		CreatedAt:  candidatePacket.CreatedAt,
	})
	candidate := makeSnapshotCandidatePacket(candidatePacket.ID, candidatePacket, candidatePacket.CreatedAt, "restore")

	selection := selectCompileContextRestoreCandidate(
		currentPacket.CreatedAt,
		current,
		[]domain.ContextPacket{candidate},
		"restore",
		compileContextResumeHints{MinimumScore: 0.99},
	)
	if selection.Decision != "fresh_compile_below_threshold" {
		t.Fatalf("expected fresh fallback below threshold, got %q", selection.Decision)
	}
	if selection.SelectedIndex != -1 {
		t.Fatalf("expected no selected candidate when below threshold")
	}
}

func TestInMemoryListContextSnapshotsFiltersAndOrders(t *testing.T) {
	store := NewInMemorySemanticStore()
	scopeMain := domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control.semantic"}
	scopeOther := domain.ForgeScope{WorkspaceID: "ws-other", LaneID: "control.semantic"}

	mainRestoreA := createTestContextPacketSnapshot("ctx-main-a", "ws-main", 1760004300000)
	mainRestoreA.Query = "summarize blockers"
	mainRestoreA.Scope = scopeMain
	mainRestoreA.CompileOptions = &domain.ContextCompileOptions{PersistSnapshot: true, SnapshotKind: "restore"}

	mainRestoreB := createTestContextPacketSnapshot("ctx-main-b", "ws-main", 1760004301000)
	mainRestoreB.Query = "summarize blockers"
	mainRestoreB.Scope = scopeMain
	mainRestoreB.CompileOptions = &domain.ContextCompileOptions{PersistSnapshot: true, SnapshotKind: "restore"}

	mainReview := createTestContextPacketSnapshot("ctx-main-review", "ws-main", 1760004302000)
	mainReview.Query = "summarize blockers"
	mainReview.Scope = scopeMain
	mainReview.CompileOptions = &domain.ContextCompileOptions{PersistSnapshot: true, SnapshotKind: "review"}

	otherRestore := createTestContextPacketSnapshot("ctx-other", "ws-other", 1760004303000)
	otherRestore.Query = "summarize blockers"
	otherRestore.Scope = scopeOther
	otherRestore.CompileOptions = &domain.ContextCompileOptions{PersistSnapshot: true, SnapshotKind: "restore"}

	for _, pkt := range []domain.ContextPacket{mainRestoreA, mainRestoreB, mainReview, otherRestore} {
		if err := store.CreateContextSnapshot(pkt); err != nil {
			t.Fatalf("seed context snapshot %s: %v", pkt.ID, err)
		}
	}

	list := store.ListContextSnapshots(scopeMain, "summarize blockers", "restore", 10)
	if len(list) != 2 {
		t.Fatalf("expected two restore snapshots in scope/query, got %d", len(list))
	}
	if list[0].ID != "ctx-main-b" || list[1].ID != "ctx-main-a" {
		t.Fatalf("expected created_at descending order, got %s then %s", list[0].ID, list[1].ID)
	}
}

func makeSnapshotCandidatePacket(id string, packet domain.ContextPacket, ts int64, kind string) domain.ContextPacket {
	packet.ID = id
	packet.CreatedAt = ts
	packet.CompileOptions = &domain.ContextCompileOptions{
		PersistSnapshot: true,
		SnapshotKind:    kind,
	}
	snapshot := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        packet,
		SnapshotID:    id,
		SnapshotKind:  kind,
		CorrelationID: "corr-" + id,
		TraceID:       "trace-" + id,
		SyscallID:     "syscall-" + id,
		ProposedBy:    "test",
		CommittedBy:   "forge_kernel",
	}, nil)
	packet.RestoreSnapshot = compiledContextSnapshotToDomain(snapshot)
	return packet
}
