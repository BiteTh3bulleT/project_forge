package controllane

import (
	"math"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	defaultRestoreCandidateLimit = 12
	defaultRestoreMinScore       = 0.45
)

type compileContextResumeHints struct {
	PreferredSnapshotID string
	MinimumScore        float64
	FreshCompileOnly    bool
}

type compileContextRestoreCandidate struct {
	Packet     domain.ContextPacket
	Snapshot   compiledContextSnapshot
	HeaderOnly bool
	Score      compileContextRestoreCandidateScore
}

type compileContextRestoreCandidateScore struct {
	SnapshotID           string  `json:"snapshotId"`
	CreatedAt            int64   `json:"createdAt"`
	SnapshotKind         string  `json:"snapshotKind,omitempty"`
	Fingerprint          string  `json:"fingerprint,omitempty"`
	ParentSnapshotID     string  `json:"parentSnapshotId,omitempty"`
	HeaderOnly           bool    `json:"headerOnly"`
	QueryScore           float64 `json:"queryScore"`
	KindScore            float64 `json:"kindScore"`
	NodeOverlapScore     float64 `json:"nodeOverlapScore"`
	EdgeOverlapScore     float64 `json:"edgeOverlapScore"`
	RecencyScore         float64 `json:"recencyScore"`
	FingerprintBonus     float64 `json:"fingerprintBonus"`
	PreferredHintBonus   float64 `json:"preferredHintBonus"`
	StalenessPenalty     float64 `json:"stalenessPenalty"`
	ContradictionPenalty float64 `json:"contradictionPenalty"`
	HeaderOnlyPenalty    float64 `json:"headerOnlyPenalty"`
	TotalScore           float64 `json:"totalScore"`
	Selected             bool    `json:"selected"`
}

type compileContextRestoreSelection struct {
	Decision       string
	Threshold      float64
	TopScore       float64
	SelectedIndex  int
	Candidates     []compileContextRestoreCandidate
	RequestedHints compileContextResumeHints
}

func selectCompileContextRestoreCandidate(now int64, current compiledContextSnapshot, packets []domain.ContextPacket, snapshotKind string, hints compileContextResumeHints) compileContextRestoreSelection {
	selection := compileContextRestoreSelection{
		Decision:       "fresh_compile_no_candidates",
		Threshold:      clamp01(nonZero(hints.MinimumScore, defaultRestoreMinScore)),
		TopScore:       0,
		SelectedIndex:  -1,
		Candidates:     []compileContextRestoreCandidate{},
		RequestedHints: hints,
	}
	if hints.FreshCompileOnly {
		selection.Decision = "fresh_compile_forced"
		return selection
	}
	for _, pkt := range packets {
		snapshot, headerOnly, ok := compiledContextSnapshotFromPacket(pkt, snapshotKind)
		if !ok {
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
		CreatedAt:        candidate.Snapshot.Header.CreatedAt,
		SnapshotKind:     strings.TrimSpace(candidate.Snapshot.Header.SnapshotKind),
		Fingerprint:      strings.TrimSpace(candidate.Snapshot.Header.Fingerprint),
		ParentSnapshotID: strings.TrimSpace(candidate.Snapshot.Header.ParentSnapshotID),
		HeaderOnly:       candidate.HeaderOnly,
	}
	if strings.TrimSpace(candidate.Snapshot.Header.Query) == strings.TrimSpace(current.Header.Query) {
		score.QueryScore = 1
	}
	if strings.TrimSpace(expectedKind) == "" || strings.TrimSpace(candidate.Snapshot.Header.SnapshotKind) == strings.TrimSpace(expectedKind) {
		score.KindScore = 1
	}
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
	score.ContradictionPenalty = contradictionPenalty(candidate.Snapshot.Graph)
	if candidate.HeaderOnly {
		score.HeaderOnlyPenalty = 0.05
	}

	weightedBase := (0.20 * score.QueryScore) +
		(0.10 * score.KindScore) +
		(0.35 * score.NodeOverlapScore) +
		(0.15 * score.EdgeOverlapScore) +
		(0.15 * score.RecencyScore) +
		score.FingerprintBonus +
		score.PreferredHintBonus
	total := weightedBase - score.StalenessPenalty - score.ContradictionPenalty - score.HeaderOnlyPenalty
	score.TotalScore = clamp01(total)
	return score
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
	return map[string]any{
		"decision":             s.Decision,
		"threshold":            s.Threshold,
		"candidate_count":      len(s.Candidates),
		"selected_snapshot_id": s.selectedSnapshotID(),
		"top_score":            s.TopScore,
		"scores":               s.scoreRows(),
	}
}

func (s compileContextRestoreSelection) resumeHintsMetadata() map[string]any {
	topSnapshotID := ""
	if len(s.Candidates) > 0 {
		topSnapshotID = s.Candidates[0].Score.SnapshotID
	}
	preferred := s.selectedSnapshotID()
	if preferred == "" {
		preferred = topSnapshotID
	}
	return map[string]any{
		"preferredSnapshotId":     preferred,
		"minimumScore":            s.Threshold,
		"freshCompileOnly":        false,
		"decision":                s.Decision,
		"candidateCount":          len(s.Candidates),
		"freshCompileRecommended": s.Decision != "selected",
		"topCandidateSnapshotId":  topSnapshotID,
		"topCandidateScore":       s.TopScore,
	}
}

func compiledContextSnapshotFromPacket(pkt domain.ContextPacket, snapshotKind string) (compiledContextSnapshot, bool, bool) {
	if pkt.RestoreSnapshot != nil {
		if snapshot, ok := compiledContextSnapshotFromDomain(pkt.RestoreSnapshot); ok {
			headerOnly := pkt.RestoreSnapshot.Evidence == nil
			if pkt.RestoreSnapshot.Evidence != nil {
				_, hasGraph := pkt.RestoreSnapshot.Evidence["graph"]
				headerOnly = !hasGraph
			}
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

func nonZero(v, fallback float64) float64 {
	if v > 0 {
		return v
	}
	return fallback
}
