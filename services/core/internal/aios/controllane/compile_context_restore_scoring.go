package controllane

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	defaultRestoreCandidateLimit = 12
	defaultRestoreMinScore       = 0.45
	restoreFreshnessHorizonMs    = int64(14 * 24 * 60 * 60 * 1000)
	restoreHardStaleHorizonMs    = int64(30 * 24 * 60 * 60 * 1000)

	restoreQueryWeight    = 0.20
	restoreScopeWeight    = 0.20
	restoreKindWeight     = 0.10
	restoreRecencyWeight  = 0.15
	restoreLineageWeight  = 0.10
	restoreStateWeight    = 0.10
	restoreLoopWeight     = 0.10
	restoreArtifactWeight = 0.05
)

type compileContextResumeHints struct {
	PreferredSnapshotID    string
	MinimumScore           float64
	FreshCompileOnly       bool
	NextAction             string
	TopBlockers            []string
	DominantStateKeys      []string
	DominantLoopIDs        []string
	RecommendedEvidenceIDs []string
	RestoreConfidence      float64
	RequiresFreshCompile   bool
	RecencyWindowMs        int64
}

type compileContextRestoreCandidate struct {
	Packet     domain.ContextPacket
	Snapshot   compiledContextSnapshot
	HeaderOnly bool
	Score      compileContextRestoreCandidateScore
}

type compileContextRestoreCandidateScore struct {
	SnapshotID           string   `json:"snapshotId"`
	ContextPacketID      string   `json:"contextPacketId,omitempty"`
	WorkspaceID          string   `json:"workspaceId,omitempty"`
	LaneID               string   `json:"laneId,omitempty"`
	SelectedPaths        []string `json:"selectedPaths,omitempty"`
	CreatedAt            int64    `json:"createdAt"`
	SnapshotKind         string   `json:"snapshotKind,omitempty"`
	Fingerprint          string   `json:"fingerprint,omitempty"`
	ParentSnapshotID     string   `json:"parentSnapshotId,omitempty"`
	HeaderOnly           bool     `json:"headerOnly"`
	QueryScore           float64  `json:"queryScore"`
	ScopeScore           float64  `json:"scopeScore"`
	KindScore            float64  `json:"kindScore"`
	LineageScore         float64  `json:"lineageScore"`
	StateOverlapScore    float64  `json:"stateOverlapScore"`
	LoopOverlapScore     float64  `json:"loopOverlapScore"`
	ArtifactOverlapScore float64  `json:"artifactOverlapScore"`
	NodeOverlapScore     float64  `json:"nodeOverlapScore"`
	EdgeOverlapScore     float64  `json:"edgeOverlapScore"`
	RecencyScore         float64  `json:"recencyScore"`
	FingerprintBonus     float64  `json:"fingerprintBonus"`
	PreferredHintBonus   float64  `json:"preferredHintBonus"`
	StalenessPenalty     float64  `json:"stalenessPenalty"`
	FreshnessPenalty     float64  `json:"freshnessPenalty"`
	ContradictionPenalty float64  `json:"contradictionPenalty"`
	HeaderOnlyPenalty    float64  `json:"headerOnlyPenalty"`
	TotalScore           float64  `json:"totalScore"`
	Confidence           float64  `json:"confidence"`
	RequiresFreshCompile bool     `json:"requiresFreshCompile"`
	Explain              []string `json:"explain"`
	Selected             bool     `json:"selected"`
}

type compileContextRestoreSelection struct {
	Decision         string
	Threshold        float64
	TopScore         float64
	SelectedIndex    int
	Candidates       []compileContextRestoreCandidate
	RequestedHints   compileContextResumeHints
	CandidatePool    int
	FilteredOut      int
	RecencyWindow    int64
	RequestQuery     string
	RequestWorkspace string
	RequestLane      string
	RequestSnapKind  string
}

func selectCompileContextRestoreCandidate(now int64, current compiledContextSnapshot, packets []domain.ContextPacket, snapshotKind string, hints compileContextResumeHints) compileContextRestoreSelection {
	selection := compileContextRestoreSelection{
		Decision:         "fresh_compile_no_candidates",
		Threshold:        clamp01(nonZero(hints.MinimumScore, defaultRestoreMinScore)),
		TopScore:         0,
		SelectedIndex:    -1,
		Candidates:       []compileContextRestoreCandidate{},
		RequestedHints:   hints,
		CandidatePool:    len(packets),
		FilteredOut:      0,
		RecencyWindow:    maxInt64(hints.RecencyWindowMs, 0),
		RequestQuery:     strings.TrimSpace(current.Header.Query),
		RequestWorkspace: strings.TrimSpace(current.Header.Scope.WorkspaceID),
		RequestLane:      strings.TrimSpace(current.Header.Scope.LaneID),
		RequestSnapKind:  strings.TrimSpace(snapshotKind),
	}
	if hints.FreshCompileOnly || hints.RequiresFreshCompile {
		selection.Decision = "fresh_compile_forced"
		return selection
	}
	for _, pkt := range packets {
		snapshot, headerOnly, ok := compiledContextSnapshotFromPacket(pkt, snapshotKind)
		if !ok {
			continue
		}
		if strings.TrimSpace(snapshot.Header.Scope.WorkspaceID) != strings.TrimSpace(current.Header.Scope.WorkspaceID) {
			selection.FilteredOut++
			continue
		}
		requestLane := strings.TrimSpace(current.Header.Scope.LaneID)
		candidateLane := strings.TrimSpace(snapshot.Header.Scope.LaneID)
		if requestLane != "" && candidateLane != "" && requestLane != candidateLane {
			selection.FilteredOut++
			continue
		}
		if !restoreCandidateWithinRecencyWindow(now, snapshot.Header.CreatedAt, selection.RecencyWindow) {
			selection.FilteredOut++
			continue
		}
		selection.Candidates = append(selection.Candidates, compileContextRestoreCandidate{
			Packet:     pkt,
			Snapshot:   snapshot,
			HeaderOnly: headerOnly,
		})
	}
	if len(selection.Candidates) == 0 {
		return selection
	}
	for idx := range selection.Candidates {
		selection.Candidates[idx].Score = scoreRestoreCandidate(now, current, selection.Candidates[idx], hints, snapshotKind)
	}
	sort.Slice(selection.Candidates, func(i, j int) bool {
		if selection.Candidates[i].Score.TotalScore == selection.Candidates[j].Score.TotalScore {
			if selection.Candidates[i].Snapshot.Header.CreatedAt == selection.Candidates[j].Snapshot.Header.CreatedAt {
				return selection.Candidates[i].Snapshot.Header.SnapshotID < selection.Candidates[j].Snapshot.Header.SnapshotID
			}
			return selection.Candidates[i].Snapshot.Header.CreatedAt > selection.Candidates[j].Snapshot.Header.CreatedAt
		}
		return selection.Candidates[i].Score.TotalScore > selection.Candidates[j].Score.TotalScore
	})
	selection.TopScore = selection.Candidates[0].Score.TotalScore
	if selection.TopScore >= selection.Threshold {
		selection.Decision = "selected"
		selection.SelectedIndex = 0
		selection.Candidates[0].Score.Selected = true
		return selection
	}
	selection.Decision = "fresh_compile_below_threshold"
	return selection
}

func scoreRestoreCandidate(now int64, current compiledContextSnapshot, candidate compileContextRestoreCandidate, hints compileContextResumeHints, expectedKind string) compileContextRestoreCandidateScore {
	score := compileContextRestoreCandidateScore{
		SnapshotID:       strings.TrimSpace(candidate.Snapshot.Header.SnapshotID),
		ContextPacketID:  strings.TrimSpace(candidate.Snapshot.Header.PacketID),
		WorkspaceID:      strings.TrimSpace(candidate.Snapshot.Header.Scope.WorkspaceID),
		LaneID:           strings.TrimSpace(candidate.Snapshot.Header.Scope.LaneID),
		SelectedPaths:    append([]string(nil), candidate.Snapshot.Header.Scope.SelectedPaths...),
		CreatedAt:        candidate.Snapshot.Header.CreatedAt,
		SnapshotKind:     strings.TrimSpace(candidate.Snapshot.Header.SnapshotKind),
		Fingerprint:      strings.TrimSpace(candidate.Snapshot.Header.Fingerprint),
		ParentSnapshotID: strings.TrimSpace(candidate.Snapshot.Header.ParentSnapshotID),
		HeaderOnly:       candidate.HeaderOnly,
	}
	score.QueryScore = lexicalQueryScore(current.Header.Query, candidate.Snapshot.Header.Query)
	if strings.TrimSpace(candidate.Snapshot.Header.Scope.WorkspaceID) == strings.TrimSpace(current.Header.Scope.WorkspaceID) {
		score.ScopeScore = scopeScore(current.Header.Scope, candidate.Snapshot.Header.Scope)
	}
	score.LineageScore = lineageOverlap(current.Header.Lineage, candidate.Snapshot.Header.Lineage)
	if strings.TrimSpace(expectedKind) == "" || strings.TrimSpace(candidate.Snapshot.Header.SnapshotKind) == strings.TrimSpace(expectedKind) {
		score.KindScore = 1
	}
	score.StateOverlapScore = graphNodePrefixOverlap(current.Graph, candidate.Snapshot.Graph, "state:")
	score.LoopOverlapScore = graphNodePrefixOverlap(current.Graph, candidate.Snapshot.Graph, "loop:")
	score.ArtifactOverlapScore = graphNodePrefixOverlap(current.Graph, candidate.Snapshot.Graph, "artifact:")
	score.NodeOverlapScore = graphNodeOverlap(current.Graph, candidate.Snapshot.Graph)
	score.EdgeOverlapScore = graphEdgeOverlap(current.Graph, candidate.Snapshot.Graph)
	score.RecencyScore = recencyScore(now, candidate.Snapshot.Header.CreatedAt)
	if strings.TrimSpace(candidate.Snapshot.Header.Fingerprint) != "" &&
		strings.TrimSpace(candidate.Snapshot.Header.Fingerprint) == strings.TrimSpace(current.Header.Fingerprint) {
		score.FingerprintBonus = 0.05
	}
	if strings.TrimSpace(hints.PreferredSnapshotID) != "" &&
		strings.TrimSpace(hints.PreferredSnapshotID) == strings.TrimSpace(candidate.Snapshot.Header.SnapshotID) {
		score.PreferredHintBonus = 0.15
	}
	score.StalenessPenalty = stalenessPenalty(now, candidate.Snapshot.Header.CreatedAt)
	score.FreshnessPenalty = freshnessPenalty(now, candidate.Snapshot.Header.CreatedAt)
	score.ContradictionPenalty = contradictionPenalty(candidate.Snapshot.Graph)
	if candidate.HeaderOnly {
		score.HeaderOnlyPenalty = 0.05
	}

	weightedBase := (restoreQueryWeight * score.QueryScore) +
		(restoreScopeWeight * score.ScopeScore) +
		(restoreKindWeight * score.KindScore) +
		(restoreRecencyWeight * score.RecencyScore) +
		(restoreLineageWeight * score.LineageScore) +
		(restoreStateWeight * score.StateOverlapScore) +
		(restoreLoopWeight * score.LoopOverlapScore) +
		(restoreArtifactWeight * score.ArtifactOverlapScore) +
		score.FingerprintBonus +
		score.PreferredHintBonus
	total := weightedBase - score.StalenessPenalty - score.FreshnessPenalty - score.ContradictionPenalty - score.HeaderOnlyPenalty
	score.TotalScore = clamp01(total)
	score.Confidence = clamp01((score.TotalScore + score.ScopeScore + score.QueryScore) / 3)
	score.RequiresFreshCompile = score.TotalScore < clamp01(nonZero(hints.MinimumScore, defaultRestoreMinScore)) || isHardStale(now, candidate.Snapshot.Header.CreatedAt)
	score.Explain = buildRestoreScoreExplain(score)
	return score
}

func lexicalQueryScore(requested, candidate string) float64 {
	reqNorm := normalizeRestoreQuery(requested)
	candNorm := normalizeRestoreQuery(candidate)
	if reqNorm == "" || candNorm == "" {
		return 0
	}
	if reqNorm == candNorm {
		return 1
	}
	reqTokens := tokenSet(reqNorm)
	candTokens := tokenSet(candNorm)
	if len(reqTokens) == 0 || len(candTokens) == 0 {
		return 0
	}
	overlap := jaccard(reqTokens, candTokens)
	if overlap == 1 {
		return 0.9
	}
	if tokenSubset(reqTokens, candTokens) || tokenSubset(candTokens, reqTokens) {
		return clamp01(0.65 + (overlap * 0.25))
	}
	if strings.HasPrefix(reqNorm, candNorm) || strings.HasPrefix(candNorm, reqNorm) {
		return 0.7
	}
	return clamp01(overlap * 0.75)
}

func normalizeRestoreQuery(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, raw)
	return strings.Join(strings.Fields(raw), " ")
}

func tokenSet(norm string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, tok := range strings.Fields(norm) {
		if len(tok) < 2 {
			continue
		}
		out[tok] = struct{}{}
	}
	return out
}

func tokenSubset(a, b map[string]struct{}) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for tok := range a {
		if _, ok := b[tok]; !ok {
			return false
		}
	}
	return true
}

func scopeScore(request, candidate domain.ForgeScope) float64 {
	if strings.TrimSpace(request.WorkspaceID) == "" || strings.TrimSpace(request.WorkspaceID) != strings.TrimSpace(candidate.WorkspaceID) {
		return 0
	}
	score := 0.55
	reqLane := strings.TrimSpace(request.LaneID)
	candLane := strings.TrimSpace(candidate.LaneID)
	switch {
	case reqLane == "" || candLane == "":
		score += 0.20
	case reqLane == candLane:
		score += 0.30
	}
	score += 0.15 * selectedPathOverlap(request.SelectedPaths, candidate.SelectedPaths)
	return clamp01(score)
}

func selectedPathOverlap(a, b []string) float64 {
	setA := map[string]struct{}{}
	setB := map[string]struct{}{}
	for _, path := range a {
		if v := strings.TrimSpace(path); v != "" {
			setA[v] = struct{}{}
		}
	}
	for _, path := range b {
		if v := strings.TrimSpace(path); v != "" {
			setB[v] = struct{}{}
		}
	}
	if len(setA) == 0 && len(setB) == 0 {
		return 1
	}
	return jaccard(setA, setB)
}

func restoreCandidateWithinRecencyWindow(now, createdAt, windowMs int64) bool {
	if windowMs <= 0 {
		return true
	}
	if createdAt <= 0 {
		return false
	}
	if now <= createdAt {
		return true
	}
	return now-createdAt <= windowMs
}

func graphNodeOverlap(a, b compiledContextSnapshotGraph) float64 {
	setA := map[string]struct{}{}
	setB := map[string]struct{}{}
	for _, node := range a.Nodes {
		setA[node.ID] = struct{}{}
	}
	for _, node := range b.Nodes {
		setB[node.ID] = struct{}{}
	}
	return jaccard(setA, setB)
}

func graphNodeIDSet(graph compiledContextSnapshotGraph, prefix string) map[string]struct{} {
	out := map[string]struct{}{}
	if prefix == "" {
		return out
	}
	for _, node := range graph.Nodes {
		id := strings.TrimSpace(node.ID)
		if strings.HasPrefix(id, prefix) {
			out[id] = struct{}{}
		}
	}
	return out
}

func graphNodePrefixOverlap(a, b compiledContextSnapshotGraph, prefix string) float64 {
	return jaccard(graphNodeIDSet(a, prefix), graphNodeIDSet(b, prefix))
}

func lineageOverlap(current, candidate compiledSnapshotLineage) float64 {
	matches := 0.0
	if strings.TrimSpace(current.CorrelationID) != "" && strings.TrimSpace(current.CorrelationID) == strings.TrimSpace(candidate.CorrelationID) {
		matches += 0.35
	}
	if strings.TrimSpace(current.TraceID) != "" && strings.TrimSpace(current.TraceID) == strings.TrimSpace(candidate.TraceID) {
		matches += 0.25
	}
	if strings.TrimSpace(current.SyscallID) != "" && strings.TrimSpace(current.SyscallID) == strings.TrimSpace(candidate.SyscallID) {
		matches += 0.2
	}
	if strings.TrimSpace(current.AuditID) != "" && strings.TrimSpace(current.AuditID) == strings.TrimSpace(candidate.AuditID) {
		matches += 0.2
	}
	return clamp01(matches)
}

func buildRestoreScoreExplain(score compileContextRestoreCandidateScore) []string {
	out := []string{}
	if score.QueryScore == 1 {
		out = append(out, "query matched")
	} else if score.QueryScore > 0 {
		out = append(out, "query token overlap")
	}
	if score.ScopeScore >= 0.85 {
		out = append(out, "scope matched")
	} else if score.ScopeScore > 0 {
		out = append(out, "workspace matched with partial lane/path scope")
	}
	if score.KindScore == 1 {
		out = append(out, "snapshot kind matched")
	}
	if score.LineageScore > 0 {
		out = append(out, "lineage overlap detected")
	}
	if score.StateOverlapScore > 0 {
		out = append(out, "state overlap detected")
	}
	if score.LoopOverlapScore > 0 {
		out = append(out, "loop overlap detected")
	}
	if score.ArtifactOverlapScore > 0 {
		out = append(out, "artifact overlap detected")
	}
	if score.FingerprintBonus > 0 {
		out = append(out, "fingerprint bonus")
	}
	if score.PreferredHintBonus > 0 {
		out = append(out, "preferred snapshot bonus")
	}
	if score.StalenessPenalty > 0 {
		out = append(out, "staleness penalty")
	}
	if score.FreshnessPenalty > 0 {
		out = append(out, "freshness penalty")
	}
	if score.ContradictionPenalty > 0 {
		out = append(out, "contradiction penalty")
	}
	if score.HeaderOnlyPenalty > 0 {
		out = append(out, "header-only penalty")
	}
	if score.RequiresFreshCompile {
		out = append(out, "fresh compile required below threshold or hard-stale")
	}
	if len(out) == 0 {
		out = append(out, "candidate scored deterministically with no strong positive signal")
	}
	return out
}

func graphEdgeOverlap(a, b compiledContextSnapshotGraph) float64 {
	setA := map[string]struct{}{}
	setB := map[string]struct{}{}
	for _, edge := range a.Edges {
		setA[edge.ID] = struct{}{}
	}
	for _, edge := range b.Edges {
		setB[edge.ID] = struct{}{}
	}
	return jaccard(setA, setB)
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	for key := range a {
		if _, ok := b[key]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union <= 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func recencyScore(now, createdAt int64) float64 {
	if createdAt <= 0 {
		return 0
	}
	if now <= createdAt {
		return 1
	}
	ageMs := float64(now - createdAt)
	halfLife := float64(12 * 60 * 60 * 1000) // 12 hours
	return 1 / (1 + ageMs/halfLife)
}

func stalenessPenalty(now, createdAt int64) float64 {
	if createdAt <= 0 || now <= createdAt {
		return 0
	}
	dayMs := int64(24 * 60 * 60 * 1000)
	ageMs := now - createdAt
	if ageMs <= dayMs {
		return 0
	}
	weekMs := float64(7 * dayMs)
	scale := float64(ageMs-dayMs) / weekMs
	return clamp01(scale) * 0.30
}

func freshnessPenalty(now, createdAt int64) float64 {
	if createdAt <= 0 || now <= createdAt {
		return 0
	}
	ageMs := now - createdAt
	if ageMs <= restoreFreshnessHorizonMs {
		return 0
	}
	scale := float64(ageMs-restoreFreshnessHorizonMs) / float64(restoreHardStaleHorizonMs-restoreFreshnessHorizonMs)
	return clamp01(scale) * 0.20
}

func isHardStale(now, createdAt int64) bool {
	return createdAt > 0 && now > createdAt && now-createdAt > restoreHardStaleHorizonMs
}

func contradictionPenalty(graph compiledContextSnapshotGraph) float64 {
	if len(graph.Nodes) == 0 {
		return 0
	}
	conflicts := 0
	for _, node := range graph.Nodes {
		if node.Conflict || hasMarker(node.Markers, "conflict") {
			conflicts++
		}
	}
	return clamp01(float64(conflicts)/float64(len(graph.Nodes))) * 0.20
}

func hasMarker(markers []string, needle string) bool {
	target := strings.TrimSpace(strings.ToLower(needle))
	for _, marker := range markers {
		if strings.TrimSpace(strings.ToLower(marker)) == target {
			return true
		}
	}
	return false
}

func (s compileContextRestoreSelection) selectedPrior() *compiledContextSnapshot {
	if s.SelectedIndex < 0 || s.SelectedIndex >= len(s.Candidates) {
		return nil
	}
	chosen := s.Candidates[s.SelectedIndex].Snapshot
	return &chosen
}

func (s compileContextRestoreSelection) selectedSnapshotID() string {
	if s.SelectedIndex < 0 || s.SelectedIndex >= len(s.Candidates) {
		return ""
	}
	return strings.TrimSpace(s.Candidates[s.SelectedIndex].Snapshot.Header.SnapshotID)
}

func (s compileContextRestoreSelection) scoreRows() []compileContextRestoreCandidateScore {
	rows := make([]compileContextRestoreCandidateScore, 0, len(s.Candidates))
	for _, candidate := range s.Candidates {
		rows = append(rows, candidate.Score)
	}
	return rows
}

func (s compileContextRestoreSelection) restoreScoresMetadata() map[string]any {
	scores := s.scoreRows()
	breakdowns := make([]any, 0, len(scores))
	for _, row := range scores {
		breakdowns = append(breakdowns, row.explainableBreakdown())
	}
	selected := s.selectedCandidate()
	selectedEvidence := []string{}
	selectedHeaderOnly := false
	if selected != nil {
		selectedEvidence = dominantEvidenceIDs(selected.Snapshot.Graph)
		selectedHeaderOnly = selected.HeaderOnly
	}
	trace := s.selectionTraceMetadata()
	return map[string]any{
		"decision":                s.Decision,
		"decision_reason":         s.decisionReason(),
		"threshold":               s.Threshold,
		"candidate_count":         len(s.Candidates),
		"candidate_pool_count":    s.CandidatePool,
		"candidates_filtered_out": s.FilteredOut,
		"recency_window_ms":       s.RecencyWindow,
		"selected_snapshot_id":    s.selectedSnapshotID(),
		"top_score":               s.TopScore,
		"selected_index":          s.SelectedIndex,
		"scores":                  scores,
		"score_breakdown":         breakdowns,
		"top_candidate_id":        s.topCandidateSnapshotID(),
		"selected_evidence_ids":   selectedEvidence,
		"selected_header_only":    selectedHeaderOnly,
		"restore_trace":           trace,
		"restore_package":         s.restorePackageMetadata(false),
	}
}

func (s compileContextRestoreSelection) decisionReason() string {
	switch s.Decision {
	case "fresh_compile_forced":
		return "restore disabled by resume hints"
	case "selected":
		return "selected top-ranked candidate"
	case "fresh_compile_no_candidates":
		if s.CandidatePool == 0 {
			return "no candidates found"
		}
		if s.FilteredOut > 0 && s.CandidatePool == s.FilteredOut {
			return "all candidates filtered by recency window"
		}
		return "no valid candidates were selected"
	case "fresh_compile_below_threshold":
		return "top candidate score below threshold"
	default:
		return ""
	}
}

func (s compileContextRestoreSelection) candidateTraceRows() []any {
	traces := make([]any, 0, len(s.Candidates))
	for _, candidate := range s.Candidates {
		row := candidate.Score.explainableBreakdown()
		row["snapshot_id"] = candidate.Score.SnapshotID
		row["packet_id"] = candidate.Packet.ID
		row["packet_scope_workspace"] = candidate.Snapshot.Header.Scope.WorkspaceID
		row["packet_scope_lane"] = candidate.Snapshot.Header.Scope.LaneID
		row["header_only"] = candidate.HeaderOnly
		row["selected"] = candidate.Score.Selected
		evidence := dominantEvidenceIDs(candidate.Snapshot.Graph)
		if len(evidence) > 6 {
			evidence = evidence[:6]
		}
		row["evidence_ids"] = evidence
		traces = append(traces, row)
	}
	return traces
}

func (s compileContextRestoreSelection) selectionTraceMetadata() map[string]any {
	top := s.topCandidate()
	selected := s.selectedCandidate()
	trace := map[string]any{
		"decision_reason": s.decisionReason(),
		"decision":        s.Decision,
		"retrieval": map[string]any{
			"candidate_pool_count":    len(s.Candidates) + s.FilteredOut,
			"candidate_count":         len(s.Candidates),
			"candidates_filtered_out": s.FilteredOut,
			"query":                   s.RequestQuery,
			"workspace_id":            s.RequestWorkspace,
			"lane_id":                 s.RequestLane,
			"snapshot_kind":           s.RequestSnapKind,
			"recency_window_ms":       s.RecencyWindow,
		},
		"candidates": s.candidateTraceRows(),
	}
	if selected != nil {
		trace["selected"] = map[string]any{
			"snapshot_id":     selected.Score.SnapshotID,
			"index":           s.SelectedIndex,
			"evidence_ids":    dominantEvidenceIDs(selected.Snapshot.Graph),
			"score":           selected.Score.TotalScore,
			"header_only":     selected.HeaderOnly,
			"packet_id":       selected.Packet.ID,
			"packet_query":    selected.Packet.Query,
			"breakdown":       selected.Score.explainableBreakdown(),
			"decision_reason": s.decisionReason(),
		}
	}
	if s.CandidatePool > 0 {
		trace["candidate_pool"] = map[string]any{
			"candidate_pool_count":    s.CandidatePool,
			"candidates_filtered_out": s.FilteredOut,
		}
	}
	if top != nil {
		trace["winner"] = map[string]any{
			"snapshot_id":  top.Score.SnapshotID,
			"index":        0,
			"score":        top.Score.TotalScore,
			"evidence_ids": dominantEvidenceIDs(top.Snapshot.Graph),
			"header_only":  top.HeaderOnly,
		}
	}
	return trace
}

func (s compileContextRestoreSelection) resumeHintsMetadata() map[string]any {
	topCandidate := s.topCandidate()
	topSnapshotID := ""
	if topCandidate != nil {
		topSnapshotID = topCandidate.Score.SnapshotID
	}
	preferred := s.selectedSnapshotID()
	if preferred == "" {
		preferred = topSnapshotID
	}

	nextAction := "fresh_compile"
	requiresFresh := s.Decision != "selected"
	if preferred != "" && !requiresFresh {
		nextAction = "resume_from_snapshot"
	}
	if override := strings.TrimSpace(s.RequestedHints.NextAction); override != "" {
		nextAction = override
	}
	base := topCandidate
	if selected := s.selectedCandidate(); selected != nil {
		base = selected
	}
	topBlockers := []string{}
	dominantStateKeys := []string{}
	dominantLoopIDs := []string{}
	recommendedEvidence := []string{}
	if base != nil {
		topBlockers = topLoopBlockers(base.Snapshot.Graph)
		dominantStateKeys = dominantIDsByPrefix(base.Snapshot.Graph, "state:")
		dominantLoopIDs = dominantIDsByPrefix(base.Snapshot.Graph, "loop:")
		recommendedEvidence = dominantEvidenceIDs(base.Snapshot.Graph)
	}
	restoreConfidence := s.TopScore
	if s.RequestedHints.RestoreConfidence > 0 {
		restoreConfidence = clamp01(s.RequestedHints.RestoreConfidence)
	}
	return map[string]any{
		"nextAction":                nextAction,
		"next_action":               nextAction,
		"top_blockers":              topBlockers,
		"dominant_state_keys":       dominantStateKeys,
		"dominant_loop_ids":         dominantLoopIDs,
		"recommended_evidence_ids":  recommendedEvidence,
		"restore_confidence":        restoreConfidence,
		"requires_fresh_compile":    requiresFresh || s.RequestedHints.RequiresFreshCompile || s.RequestedHints.FreshCompileOnly,
		"preferredSnapshotId":       preferred,
		"preferred_snapshot_id":     preferred,
		"minimumScore":              s.Threshold,
		"minimum_score":             s.Threshold,
		"freshCompileOnly":          requiresFresh || s.RequestedHints.RequiresFreshCompile,
		"fresh_compile_only":        requiresFresh || s.RequestedHints.RequiresFreshCompile,
		"decision":                  s.Decision,
		"candidateCount":            len(s.Candidates),
		"candidate_count":           len(s.Candidates),
		"freshCompileRecommended":   requiresFresh,
		"topCandidateSnapshotId":    topSnapshotID,
		"topCandidateScore":         s.TopScore,
		"top_candidate_snapshot_id": topSnapshotID,
		"top_candidate_score":       s.TopScore,
	}
}

func (s compileContextRestoreSelection) restorePackageMetadata(expandGraph bool) map[string]any {
	selected := s.selectedCandidate()
	if selected == nil {
		selected = s.topCandidate()
	}
	header := map[string]any{}
	resumeHints := s.resumeHintsMetadata()
	selectedEvidence := []string{}
	selectedContextPacketID := ""
	selectedScore := map[string]any{}
	selectedSnapshotID := ""
	restoreConfidence := s.TopScore
	if selected != nil {
		selectedSnapshotID = selected.Score.SnapshotID
		selectedContextPacketID = strings.TrimSpace(selected.Packet.ID)
		header = selected.Snapshot.headerMetadata()
		selectedEvidence = dominantEvidenceIDs(selected.Snapshot.Graph)
		selectedScore = selected.Score.explainableBreakdown()
		restoreConfidence = selected.Score.Confidence
		if expandGraph {
			header["graph"] = selected.Snapshot.Graph
			header["delta"] = selected.Snapshot.Delta
		}
	}
	return map[string]any{
		"selected_snapshot_id":       selectedSnapshotID,
		"selected_context_packet_id": selectedContextPacketID,
		"restore_confidence":         restoreConfidence,
		"requires_fresh_compile":     s.Decision != "selected",
		"selected_score":             selectedScore,
		"score_breakdown":            s.scoreBreakdownRows(),
		"header":                     header,
		"resume_hints":               resumeHints,
		"selected_evidence_refs":     selectedEvidence,
		"candidate_summaries":        s.candidateSummaries(),
		"trace":                      s.selectionTraceMetadata(),
	}
}

func (s compileContextRestoreSelection) scoreBreakdownRows() []map[string]any {
	rows := make([]map[string]any, 0, len(s.Candidates))
	for _, candidate := range s.Candidates {
		rows = append(rows, candidate.Score.explainableBreakdown())
	}
	return rows
}

func (s compileContextRestoreSelection) candidateSummaries() []map[string]any {
	rows := make([]map[string]any, 0, len(s.Candidates))
	for _, candidate := range s.Candidates {
		rows = append(rows, map[string]any{
			"snapshot_id":            candidate.Score.SnapshotID,
			"context_packet_id":      candidate.Score.ContextPacketID,
			"snapshot_kind":          candidate.Score.SnapshotKind,
			"query":                  candidate.Snapshot.Header.Query,
			"workspace_id":           candidate.Score.WorkspaceID,
			"lane_id":                candidate.Score.LaneID,
			"selected_paths":         candidate.Score.SelectedPaths,
			"created_at":             candidate.Score.CreatedAt,
			"snapshot_fingerprint":   candidate.Score.Fingerprint,
			"parent_snapshot_id":     candidate.Score.ParentSnapshotID,
			"header_json_available":  candidate.Snapshot.Header.SnapshotID != "",
			"graph_json_available":   !candidate.HeaderOnly && len(candidate.Snapshot.Graph.Nodes) > 0,
			"delta_json_available":   len(candidate.Snapshot.Delta.AddedNodeIDs)+len(candidate.Snapshot.Delta.RemovedNodeIDs)+len(candidate.Snapshot.Delta.ChangedNodeIDs)+len(candidate.Snapshot.Delta.AddedEdgeIDs)+len(candidate.Snapshot.Delta.RemovedEdgeIDs)+len(candidate.Snapshot.Delta.ChangedEdgeIDs) > 0,
			"render_artifact_ref_id": candidate.Snapshot.Header.RenderedCardArtifactID,
			"lineage":                candidate.Snapshot.Header.Lineage,
			"total_score":            candidate.Score.TotalScore,
			"confidence":             candidate.Score.Confidence,
			"requires_fresh_compile": candidate.Score.RequiresFreshCompile,
			"selected":               candidate.Score.Selected,
		})
	}
	return rows
}

func (s compiledContextSnapshot) headerMetadata() map[string]any {
	return map[string]any{
		"snapshot_id":            s.Header.SnapshotID,
		"context_packet_id":      s.Header.PacketID,
		"snapshot_kind":          s.Header.SnapshotKind,
		"query":                  s.Header.Query,
		"workspace_id":           s.Header.Scope.WorkspaceID,
		"lane_id":                s.Header.Scope.LaneID,
		"selected_paths":         append([]string(nil), s.Header.Scope.SelectedPaths...),
		"created_at":             s.Header.CreatedAt,
		"snapshot_fingerprint":   s.Header.Fingerprint,
		"parent_snapshot_id":     s.Header.ParentSnapshotID,
		"render_artifact_ref_id": s.Header.RenderedCardArtifactID,
		"counts":                 s.Header.Counts,
		"lineage":                s.Header.Lineage,
		"graph_json_available":   len(s.Graph.Nodes) > 0,
		"delta_json_available":   len(s.Delta.AddedNodeIDs)+len(s.Delta.RemovedNodeIDs)+len(s.Delta.ChangedNodeIDs)+len(s.Delta.AddedEdgeIDs)+len(s.Delta.RemovedEdgeIDs)+len(s.Delta.ChangedEdgeIDs) > 0,
	}
}

func (s compileContextRestoreSelection) topCandidate() *compileContextRestoreCandidate {
	if len(s.Candidates) == 0 {
		return nil
	}
	return &s.Candidates[0]
}

func (s compileContextRestoreSelection) selectedCandidate() *compileContextRestoreCandidate {
	if s.SelectedIndex < 0 || s.SelectedIndex >= len(s.Candidates) {
		return nil
	}
	return &s.Candidates[s.SelectedIndex]
}

func (s compileContextRestoreSelection) topCandidateSnapshotID() string {
	candidate := s.topCandidate()
	if candidate == nil {
		return ""
	}
	return candidate.Score.SnapshotID
}

func (r compileContextRestoreCandidateScore) explainableBreakdown() map[string]any {
	return map[string]any{
		"snapshot_id":            r.SnapshotID,
		"created_at":             r.CreatedAt,
		"snapshot_kind":          r.SnapshotKind,
		"context_packet_id":      r.ContextPacketID,
		"workspace_id":           r.WorkspaceID,
		"lane_id":                r.LaneID,
		"selected_paths":         r.SelectedPaths,
		"query_score":            r.QueryScore,
		"scope_score":            r.ScopeScore,
		"kind_score":             r.KindScore,
		"snapshot_kind_score":    r.KindScore,
		"recency_score":          r.RecencyScore,
		"lineage_score":          r.LineageScore,
		"state_overlap_score":    r.StateOverlapScore,
		"loop_overlap_score":     r.LoopOverlapScore,
		"artifact_overlap_score": r.ArtifactOverlapScore,
		"fingerprint_bonus":      r.FingerprintBonus,
		"preferred_hint_bonus":   r.PreferredHintBonus,
		"node_overlap_score":     r.NodeOverlapScore,
		"edge_overlap_score":     r.EdgeOverlapScore,
		"contradiction_penalty":  r.ContradictionPenalty,
		"staleness_penalty":      r.StalenessPenalty,
		"freshness_penalty":      r.FreshnessPenalty,
		"header_only_penalty":    r.HeaderOnlyPenalty,
		"total":                  r.TotalScore,
		"total_score":            r.TotalScore,
		"confidence":             r.Confidence,
		"requires_fresh_compile": r.RequiresFreshCompile,
		"explain":                r.Explain,
	}
}

func compiledContextSnapshotFromPacket(pkt domain.ContextPacket, snapshotKind string) (compiledContextSnapshot, bool, bool) {
	if pkt.RestoreSnapshot != nil {
		if snapshot, ok := compiledContextSnapshotFromDomain(pkt.RestoreSnapshot); ok {
			headerOnly := isHeaderOnlySnapshotEvidence(pkt.RestoreSnapshot.Evidence)
			return snapshot, headerOnly, true
		}
	}
	kind := strings.TrimSpace(snapshotKind)
	if kind == "" {
		kind = contextSnapshotKind(pkt)
	}
	if kind == "" {
		kind = "restore"
	}
	header := compiledContextSnapshotHeader{
		SnapshotID:   pkt.ID,
		PacketID:     pkt.ID,
		SnapshotKind: kind,
		Query:        pkt.Query,
		Scope:        pkt.Scope,
		Budget:       pkt.Budget,
		CreatedAt:    pkt.CreatedAt,
	}
	return compiledContextSnapshot{
		Header: header,
		Graph: compiledContextSnapshotGraph{
			Objective: compiledContextSnapshotNode{
				ID:    objectiveNodeID(pkt.Query),
				Type:  "objective",
				Label: truncateSnapshotText(pkt.Query, 64),
			},
			Rails: []compiledContextSnapshotRail{
				{Name: "constraints", Label: "Constraints"},
				{Name: "evidence", Label: "Evidence"},
				{Name: "hypotheses", Label: "Hypotheses"},
				{Name: "loops", Label: "Loops"},
			},
			Nodes: []compiledContextSnapshotNode{},
			Edges: []compiledContextSnapshotEdge{},
		},
		Delta: compiledContextSnapshotDelta{
			AddedNodeIDs:   []string{},
			RemovedNodeIDs: []string{},
			ChangedNodeIDs: []string{},
			AddedEdgeIDs:   []string{},
			RemovedEdgeIDs: []string{},
			ChangedEdgeIDs: []string{},
		},
	}, true, true
}

func isHeaderOnlySnapshotEvidence(evidence map[string]any) bool {
	if evidence == nil {
		return true
	}
	rawGraph, hasGraph := evidence["graph"]
	if !hasGraph {
		return true
	}
	switch graph := rawGraph.(type) {
	case map[string]any:
		return len(graph) == 0
	case []any:
		return len(graph) == 0
	case nil:
		return true
	default:
		return false
	}
}

func readCompileContextResumeHints(payload map[string]any) compileContextResumeHints {
	hints := compileContextResumeHints{}
	apply := func(src map[string]any) {
		if src == nil {
			return
		}
		if v := strings.TrimSpace(readString(src, "preferredSnapshotId")); v != "" {
			hints.PreferredSnapshotID = v
		}
		if v := strings.TrimSpace(readString(src, "preferred_snapshot_id")); v != "" {
			hints.PreferredSnapshotID = v
		}
		if v := readFloat(src, "minimumScore", 0); v > 0 {
			hints.MinimumScore = clamp01(v)
		}
		if v := readFloat(src, "minimum_score", 0); v > 0 {
			hints.MinimumScore = clamp01(v)
		}
		if _, ok := src["restoreConfidence"]; ok {
			value := readFloat(src, "restoreConfidence", -1)
			if value >= 0 && value <= 1 {
				hints.RestoreConfidence = clamp01(value)
			}
		}
		if _, ok := src["restore_confidence"]; ok {
			value := readFloat(src, "restore_confidence", -1)
			if value >= 0 && value <= 1 {
				hints.RestoreConfidence = clamp01(value)
			}
		}
		if v := strings.TrimSpace(readString(src, "nextAction")); v != "" {
			hints.NextAction = v
		}
		if v := strings.TrimSpace(readString(src, "next_action")); v != "" {
			hints.NextAction = v
		}
		if v := readStringSlice(src, "topBlockers"); len(v) > 0 {
			hints.TopBlockers = v
		}
		if v := readStringSlice(src, "top_blockers"); len(v) > 0 {
			hints.TopBlockers = v
		}
		if v := readStringSlice(src, "dominantStateKeys"); len(v) > 0 {
			hints.DominantStateKeys = v
		}
		if v := readStringSlice(src, "dominant_state_keys"); len(v) > 0 {
			hints.DominantStateKeys = v
		}
		if v := readStringSlice(src, "dominantLoopIds"); len(v) > 0 {
			hints.DominantLoopIDs = v
		}
		if v := readStringSlice(src, "dominant_loop_ids"); len(v) > 0 {
			hints.DominantLoopIDs = v
		}
		if v := readStringSlice(src, "recommendedEvidenceIds"); len(v) > 0 {
			hints.RecommendedEvidenceIDs = v
		}
		if v := readStringSlice(src, "recommendedEvidenceIDs"); len(v) > 0 {
			hints.RecommendedEvidenceIDs = v
		}
		if v := readStringSlice(src, "recommended_evidence_ids"); len(v) > 0 {
			hints.RecommendedEvidenceIDs = v
		}
		if v := readFloat(src, "recencyWindowMs", 0); v > 0 {
			hints.RecencyWindowMs = int64(math.Round(v))
		}
		if v := readFloat(src, "recency_window_ms", 0); v > 0 {
			hints.RecencyWindowMs = int64(math.Round(v))
		}
		if v, present, valid := readOptionalBool(src, "requiresFreshCompile"); present && valid {
			hints.RequiresFreshCompile = v
		}
		if v, present, valid := readOptionalBool(src, "requires_fresh_compile"); present && valid {
			hints.RequiresFreshCompile = v
		}
		if v, present, valid := readOptionalBool(src, "freshCompileOnly"); present && valid {
			hints.FreshCompileOnly = v
		}
		if v, present, valid := readOptionalBool(src, "fresh_compile_only"); present && valid {
			hints.FreshCompileOnly = v
		}
	}
	if raw, ok := payload["resumeHints"].(map[string]any); ok {
		apply(raw)
	}
	if raw, ok := payload["restoreSnapshot"].(map[string]any); ok {
		if hintsRaw, ok := raw["resumeHints"].(map[string]any); ok {
			apply(hintsRaw)
		}
		apply(raw)
	}
	if raw, ok := payload["compileOptions"].(map[string]any); ok {
		if hintsRaw, ok := raw["resumeHints"].(map[string]any); ok {
			apply(hintsRaw)
		}
		apply(raw)
	}
	return hints
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return math.Round(v*1000) / 1000
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func nonZero(v, fallback float64) float64 {
	if v > 0 {
		return v
	}
	return fallback
}

func topLoopBlockers(graph compiledContextSnapshotGraph) []string {
	ids := make([]string, 0)
	for _, node := range graph.Nodes {
		if strings.TrimSpace(node.Type) != "open_loop" {
			continue
		}
		if node.Blocker || hasMarker(node.Markers, "blocker") {
			ids = append(ids, strings.TrimPrefix(strings.TrimSpace(node.ID), "loop:"))
		}
	}
	sort.Strings(ids)
	if len(ids) > 6 {
		return ids[:6]
	}
	return ids
}

func dominantIDsByPrefix(graph compiledContextSnapshotGraph, prefix string) []string {
	out := make([]string, 0)
	for _, node := range graph.Nodes {
		id := strings.TrimSpace(node.ID)
		if strings.HasPrefix(id, prefix) {
			out = append(out, strings.TrimPrefix(id, prefix))
		}
	}
	sort.Strings(out)
	if len(out) > 10 {
		return out[:10]
	}
	return out
}

func dominantEvidenceIDs(graph compiledContextSnapshotGraph) []string {
	out := make([]string, 0)
	for _, node := range graph.Nodes {
		id := strings.TrimSpace(node.ID)
		switch {
		case strings.HasPrefix(id, "artifact:"):
			out = append(out, strings.TrimPrefix(id, "artifact:"))
		case strings.HasPrefix(id, "note:"):
			out = append(out, strings.TrimPrefix(id, "note:"))
		case strings.HasPrefix(id, "event:"):
			out = append(out, strings.TrimPrefix(id, "event:"))
		}
	}
	sort.Strings(out)
	if len(out) > 12 {
		return out[:12]
	}
	return out
}
