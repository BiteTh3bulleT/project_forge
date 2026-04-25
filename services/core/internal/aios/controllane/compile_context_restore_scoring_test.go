package controllane

import (
	"strings"
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

func TestSelectCompileContextRestoreCandidateRecencyWindowFiltersCandidates(t *testing.T) {
	currentPacket := createTestContextPacketSnapshot("ctx-current-recency", "ws-main", 1760004300000)
	currentPacket.Query = "summarize blockers"
	current := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        currentPacket,
		SnapshotID:    currentPacket.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-current-recency",
		TraceID:       "trace-current-recency",
		SyscallID:     "syscall-current-recency",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)

	recent := makeSnapshotCandidatePacket("ctx-candidate-recent", currentPacket, 1760004290000, "restore")
	old := makeSnapshotCandidatePacket("ctx-candidate-old", currentPacket, 1750000000000, "restore")

	selection := selectCompileContextRestoreCandidate(
		currentPacket.CreatedAt,
		current,
		[]domain.ContextPacket{old, recent},
		"restore",
		compileContextResumeHints{RecencyWindowMs: 10 * 60 * 1000},
	)
	if selection.Decision != "selected" {
		t.Fatalf("expected selection to proceed with recency-filtered candidates, got %q", selection.Decision)
	}
	if len(selection.Candidates) != 1 || selection.Candidates[0].Score.SnapshotID != "ctx-candidate-recent" {
		t.Fatalf("expected only recent candidate to remain, got %d (%q)", len(selection.Candidates), selection.Candidates[0].Score.SnapshotID)
	}
	if selection.CandidatePool != 2 || selection.FilteredOut != 1 {
		t.Fatalf("expected candidate pool/filtering to be tracked, pool=%d filtered=%d", selection.CandidatePool, selection.FilteredOut)
	}
	if got := selection.decisionReason(); got == "" {
		t.Fatalf("expected explicit decision reason when selected, got %q", got)
	}
}

func TestRestoreScoringPrefersExactQuerySameScopeAndExcludesWrongWorkspace(t *testing.T) {
	currentPacket := createTestContextPacketSnapshot("ctx-current-query", "ws-main", 1760004300000)
	currentPacket.Query = "Summarize blockers for restore scoring"
	current := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        currentPacket,
		SnapshotID:    currentPacket.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-current-query",
		TraceID:       "trace-current-query",
		SyscallID:     "syscall-current-query",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)

	exact := makeSnapshotCandidatePacket("ctx-exact-query", currentPacket, 1760004290000, "restore")
	partialPacket := currentPacket
	partialPacket.ID = "ctx-partial-query"
	partialPacket.Query = "restore blockers"
	partial := makeSnapshotCandidatePacket(partialPacket.ID, partialPacket, 1760004295000, "restore")
	otherWorkspacePacket := currentPacket
	otherWorkspacePacket.ID = "ctx-other-workspace"
	otherWorkspacePacket.Scope.WorkspaceID = "ws-other"
	otherWorkspace := makeSnapshotCandidatePacket(otherWorkspacePacket.ID, otherWorkspacePacket, 1760004299000, "restore")

	selection := selectCompileContextRestoreCandidate(
		currentPacket.CreatedAt,
		current,
		[]domain.ContextPacket{otherWorkspace, partial, exact},
		"restore",
		compileContextResumeHints{},
	)
	if selection.Decision != "selected" {
		t.Fatalf("expected selected restore candidate, got %q", selection.Decision)
	}
	if got := selection.selectedSnapshotID(); got != "ctx-exact-query" {
		t.Fatalf("expected exact same-scope query to win, got %q", got)
	}
	if len(selection.Candidates) != 2 || selection.FilteredOut != 1 {
		t.Fatalf("expected wrong workspace excluded before scoring, candidates=%d filtered=%d", len(selection.Candidates), selection.FilteredOut)
	}
	if selection.Candidates[0].Score.QueryScore <= selection.Candidates[1].Score.QueryScore {
		t.Fatalf("expected exact lexical query score to exceed partial score")
	}
}

func TestSelectCompileContextRestoreCandidateRecencyWindowCanFilterAllCandidates(t *testing.T) {
	currentPacket := createTestContextPacketSnapshot("ctx-current-recency-empty", "ws-main", 1760004300000)
	currentPacket.Query = "summarize blockers"
	current := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        currentPacket,
		SnapshotID:    currentPacket.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-current-recency-empty",
		TraceID:       "trace-current-recency-empty",
		SyscallID:     "syscall-current-recency-empty",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)

	oldA := makeSnapshotCandidatePacket("ctx-candidate-old-a", currentPacket, 1750000000000, "restore")
	oldB := makeSnapshotCandidatePacket("ctx-candidate-old-b", currentPacket, 1750000001000, "restore")

	selection := selectCompileContextRestoreCandidate(
		currentPacket.CreatedAt,
		current,
		[]domain.ContextPacket{oldA, oldB},
		"restore",
		compileContextResumeHints{RecencyWindowMs: 60 * 1000},
	)
	if selection.Decision != "fresh_compile_no_candidates" {
		t.Fatalf("expected fresh compile fallback when all candidates filtered, got %q", selection.Decision)
	}
	if selection.CandidatePool != 2 || selection.FilteredOut != 2 || len(selection.Candidates) != 0 {
		t.Fatalf("expected all candidates filtered: pool=%d filtered=%d candidates=%d", selection.CandidatePool, selection.FilteredOut, len(selection.Candidates))
	}
	if got := selection.decisionReason(); got != "all candidates filtered by recency window" {
		t.Fatalf("expected all-candidates-recency decision reason, got %q", got)
	}
}

func TestRestoreScoresMetadataContainsDeterministicBreakdownAndTrace(t *testing.T) {
	currentPacket := createTestContextPacketSnapshot("ctx-current-trace", "ws-main", 1760004400000)
	currentPacket.Query = "summarize blockers"
	current := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        currentPacket,
		SnapshotID:    currentPacket.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-current-trace",
		TraceID:       "trace-current-trace",
		SyscallID:     "syscall-current-trace",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)

	selection := selectCompileContextRestoreCandidate(
		currentPacket.CreatedAt,
		current,
		[]domain.ContextPacket{
			makeSnapshotCandidatePacket("ctx-candidate-trace", currentPacket, 1760004390000, "restore"),
		},
		"restore",
		compileContextResumeHints{},
	)
	if selection.Decision != "selected" {
		t.Fatalf("expected selected decision, got %q", selection.Decision)
	}
	meta := selection.restoreScoresMetadata()
	traceRaw, ok := meta["restore_trace"].(map[string]any)
	if !ok || len(traceRaw) == 0 {
		t.Fatalf("expected restore_trace in metadata, got %#v", meta["restore_trace"])
	}
	candidatesRaw, ok := traceRaw["candidates"]
	if !ok {
		t.Fatalf("expected candidate traces in restore trace")
	}
	candidatesSlice, ok := candidatesRaw.([]any)
	if !ok {
		t.Fatalf("invalid candidate trace payload type")
	}
	candidates := make([]map[string]any, 0, len(candidatesSlice))
	for _, candidateRaw := range candidatesSlice {
		candidate, ok := candidateRaw.(map[string]any)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected candidate trace entries, got %d", len(candidates))
	}
	breakdownRaw, ok := meta["score_breakdown"]
	if !ok {
		t.Fatalf("expected score_breakdown in restore metadata")
	}
	breakdownsSlice, ok := breakdownRaw.([]any)
	if !ok {
		t.Fatalf("invalid score_breakdown payload type")
	}
	breakdowns := make([]map[string]any, 0, len(breakdownsSlice))
	for _, breakdownRaw := range breakdownsSlice {
		breakdown, ok := breakdownRaw.(map[string]any)
		if !ok {
			continue
		}
		breakdowns = append(breakdowns, breakdown)
	}
	if len(breakdowns) != 1 {
		t.Fatalf("expected one score breakdown row, got %d", len(breakdowns))
	}
	required := []string{
		"query_score", "scope_score", "recency_score", "lineage_score",
		"state_overlap_score", "loop_overlap_score", "artifact_overlap_score",
		"contradiction_penalty", "staleness_penalty", "freshness_penalty",
		"snapshot_kind_score", "confidence", "requires_fresh_compile", "explain", "total",
	}
	for _, key := range required {
		if _, ok := breakdowns[0][key]; !ok {
			t.Fatalf("expected breakdown key %q", key)
		}
	}
	if _, ok := traceRaw["retrieval"].(map[string]any); !ok {
		t.Fatalf("expected retrieval section in restore trace")
	}
	retrieval, ok := traceRaw["retrieval"].(map[string]any)
	if !ok {
		t.Fatalf("retrieval section must be object")
	}
	if got := strings.TrimSpace(readString(retrieval, "query")); got != "summarize blockers" {
		t.Fatalf("expected retrieval query %q, got %q", "summarize blockers", got)
	}
	if got := strings.TrimSpace(readString(retrieval, "snapshot_kind")); got != "restore" {
		t.Fatalf("expected retrieval snapshot_kind %q, got %q", "restore", got)
	}
	if got := strings.TrimSpace(readString(retrieval, "workspace_id")); got != "ws-main" {
		t.Fatalf("expected retrieval workspace_id %q, got %q", "ws-main", got)
	}
	if got := strings.TrimSpace(readString(retrieval, "lane_id")); got != "compute.context" {
		t.Fatalf("expected retrieval lane_id %q, got %q", "compute.context", got)
	}
	restorePackage, ok := meta["restore_package"].(map[string]any)
	if !ok {
		t.Fatalf("expected header-first restore_package in metadata")
	}
	if _, ok := restorePackage["header"].(map[string]any); !ok {
		t.Fatalf("expected restore package header")
	}
	summaries, ok := restorePackage["candidate_summaries"].([]map[string]any)
	if !ok || len(summaries) == 0 {
		t.Fatalf("expected candidate summaries")
	}
	if restorePackage["requires_fresh_compile"] != false {
		t.Fatalf("expected selected restore package to avoid fresh compile")
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
	if !selection.Candidates[0].Score.RequiresFreshCompile {
		t.Fatalf("expected candidate score to expose requires_fresh_compile below threshold")
	}
	if selection.Candidates[0].Score.FreshnessPenalty == 0 {
		t.Fatalf("expected stale candidate to carry freshness penalty")
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
