package controllane

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type compiledContextSnapshot struct {
	Header compiledContextSnapshotHeader `json:"header"`
	Graph  compiledContextSnapshotGraph  `json:"graph"`
	Delta  compiledContextSnapshotDelta  `json:"delta"`
}

type compiledContextSnapshotHeader struct {
	SnapshotID             string                  `json:"snapshotId"`
	PacketID               string                  `json:"packetId"`
	SnapshotKind           string                  `json:"snapshotKind"`
	Query                  string                  `json:"query"`
	Fingerprint            string                  `json:"fingerprint"`
	ParentSnapshotID       string                  `json:"parentSnapshotId,omitempty"`
	RenderedCardArtifactID string                  `json:"renderedCardArtifactId,omitempty"`
	Scope                  domain.ForgeScope       `json:"scope"`
	Budget                 domain.ContextBudget    `json:"budget"`
	Counts                 compiledSnapshotCounts  `json:"counts"`
	Lineage                compiledSnapshotLineage `json:"lineage"`
	CreatedAt              int64                   `json:"createdAt"`
}

type compiledSnapshotCounts struct {
	Constraints int `json:"constraints"`
	Evidence    int `json:"evidence"`
	Hypotheses  int `json:"hypotheses"`
	Loops       int `json:"loops"`
	Nodes       int `json:"nodes"`
	Edges       int `json:"edges"`
}

type compiledSnapshotLineage struct {
	Scope         domain.ForgeScope `json:"scope"`
	CorrelationID string            `json:"correlationId,omitempty"`
	TraceID       string            `json:"traceId,omitempty"`
	SyscallID     string            `json:"syscallId,omitempty"`
	AuditID       string            `json:"auditId,omitempty"`
	ProposedBy    string            `json:"proposedBy,omitempty"`
	CommittedBy   string            `json:"committedBy,omitempty"`
}

type compiledContextSnapshotGraph struct {
	Objective compiledContextSnapshotNode   `json:"objective"`
	Rails     []compiledContextSnapshotRail `json:"rails"`
	Nodes     []compiledContextSnapshotNode `json:"nodes"`
	Edges     []compiledContextSnapshotEdge `json:"edges"`
}

type compiledContextSnapshotRail struct {
	Name    string   `json:"name"`
	Label   string   `json:"label"`
	NodeIDs []string `json:"nodeIds"`
}

type compiledContextSnapshotNode struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Rail       string   `json:"rail,omitempty"`
	Label      string   `json:"label"`
	Detail     string   `json:"detail,omitempty"`
	Status     string   `json:"status,omitempty"`
	Salience   string   `json:"salience,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
	Blocker    bool     `json:"blocker,omitempty"`
	Conflict   bool     `json:"conflict,omitempty"`
	Markers    []string `json:"markers,omitempty"`
}

type compiledContextSnapshotEdge struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	SourceID   string  `json:"sourceId"`
	TargetID   string  `json:"targetId"`
	Confidence float64 `json:"confidence,omitempty"`
}

type compiledContextSnapshotDelta struct {
	FingerprintMatched bool     `json:"fingerprintMatched"`
	AddedNodeIDs       []string `json:"addedNodeIds"`
	RemovedNodeIDs     []string `json:"removedNodeIds"`
	ChangedNodeIDs     []string `json:"changedNodeIds"`
	AddedEdgeIDs       []string `json:"addedEdgeIds"`
	RemovedEdgeIDs     []string `json:"removedEdgeIds"`
	ChangedEdgeIDs     []string `json:"changedEdgeIds"`
}

type compileContextSnapshotOptions struct {
	PersistSnapshot                   bool
	RenderSnapshotCard                bool
	SnapshotKind                      string
	RestoreMode                       bool
	RestoreCandidateLimit             int
	RestoreMinScore                   float64
	RequireFreshCompileBelowThreshold bool
	ExpandRestoreGraph                bool
	RestoreCacheDisabled              bool
}

type compiledSnapshotBuildInput struct {
	Packet        domain.ContextPacket
	SnapshotID    string
	SnapshotKind  string
	CorrelationID string
	TraceID       string
	SyscallID     string
	AuditID       string
	ProposedBy    string
	CommittedBy   string
}

func buildCompiledContextSnapshot(input compiledSnapshotBuildInput, prior *compiledContextSnapshot) compiledContextSnapshot {
	objective := compiledContextSnapshotNode{
		ID:       objectiveNodeID(input.Packet.Query),
		Type:     "objective",
		Label:    truncateSnapshotText(input.Packet.Query, 64),
		Detail:   fmt.Sprintf("budget:%d/%d/%d", input.Packet.Budget.MaxTokens, input.Packet.Budget.MaxEvents, input.Packet.Budget.MaxNotes),
		Salience: "high",
		Markers:  []string{"salience:high"},
	}

	conflicted := conflictedNoteIDs(input.Packet.LinkedNotes)
	rails := []compiledContextSnapshotRail{
		{Name: "constraints", Label: "Constraints"},
		{Name: "evidence", Label: "Evidence"},
		{Name: "hypotheses", Label: "Hypotheses"},
		{Name: "loops", Label: "Loops"},
	}

	nodes := []compiledContextSnapshotNode{}
	edges := []compiledContextSnapshotEdge{}
	nodeKinds := map[string]string{objective.ID: objective.Type}

	appendRailNode := func(node compiledContextSnapshotNode, edgeType string, edgeConfidence float64) {
		nodes = append(nodes, node)
		nodeKinds[node.ID] = node.Type
		edges = append(edges, compiledContextSnapshotEdge{
			ID:         stableSnapshotID(edgeType, node.ID, objective.ID),
			Type:       edgeType,
			SourceID:   node.ID,
			TargetID:   objective.ID,
			Confidence: edgeConfidence,
		})
		for idx := range rails {
			if rails[idx].Name == node.Rail {
				rails[idx].NodeIDs = append(rails[idx].NodeIDs, node.ID)
				return
			}
		}
	}

	for idx, state := range input.Packet.ActiveState {
		appendRailNode(compiledContextSnapshotNode{
			ID:         "state:" + state.ID,
			Type:       "state_item",
			Rail:       "constraints",
			Label:      truncateSnapshotText(state.Key, 48),
			Detail:     truncateSnapshotText(canonicalJSONString(state.Value), 56),
			Status:     string(state.Status),
			Salience:   salienceForRank(idx),
			Confidence: 1.0,
			Markers:    nodeMarkers(salienceForRank(idx), 1.0, false, false),
		}, "constrains", 1.0)
	}

	evidenceRank := 0
	for _, note := range sortedNotesForSnapshot(input.Packet.Notes) {
		appendRailNode(compiledContextSnapshotNode{
			ID:         "note:" + note.ID,
			Type:       "memory_note",
			Rail:       "evidence",
			Label:      truncateSnapshotText(nonEmpty(note.Title, note.ID), 48),
			Detail:     truncateSnapshotText(note.Content, 56),
			Status:     string(note.Status),
			Salience:   salienceForRank(evidenceRank),
			Confidence: note.Confidence,
			Conflict:   conflicted[note.ID],
			Markers:    nodeMarkers(salienceForRank(evidenceRank), note.Confidence, false, conflicted[note.ID]),
		}, "supports", note.Confidence)
		evidenceRank++
	}
	for _, artifact := range sortedArtifactsForSnapshot(input.Packet.Artifacts) {
		if isSnapshotCardArtifact(artifact) {
			continue
		}
		appendRailNode(compiledContextSnapshotNode{
			ID:         "artifact:" + artifact.ID,
			Type:       "artifact_ref",
			Rail:       "evidence",
			Label:      truncateSnapshotText(nonEmpty(readString(artifact.Metadata, "title"), artifact.ID), 48),
			Detail:     truncateSnapshotText(artifact.URI, 56),
			Salience:   salienceForRank(evidenceRank),
			Confidence: 1.0,
			Markers:    nodeMarkers(salienceForRank(evidenceRank), 1.0, false, false),
		}, "supports", 1.0)
		evidenceRank++
	}
	for _, evt := range sortedEventsForSnapshot(input.Packet.RawEvents) {
		if isCompileContextJournalEvent(evt) {
			continue
		}
		appendRailNode(compiledContextSnapshotNode{
			ID:         "event:" + evt.ID,
			Type:       "journal_event",
			Rail:       "evidence",
			Label:      truncateSnapshotText(nonEmpty(evt.Type, evt.ID), 48),
			Detail:     truncateSnapshotText(canonicalJSONString(evt.Payload), 56),
			Salience:   salienceForRank(evidenceRank),
			Confidence: 1.0,
			Markers:    nodeMarkers(salienceForRank(evidenceRank), 1.0, false, false),
		}, "supports", 1.0)
		evidenceRank++
	}

	for idx, model := range sortedModelsForSnapshot(input.Packet.Models) {
		appendRailNode(compiledContextSnapshotNode{
			ID:         "model:" + model.ID,
			Type:       "derived_model",
			Rail:       "hypotheses",
			Label:      truncateSnapshotText(nonEmpty(model.Type, model.ID), 48),
			Detail:     truncateSnapshotText(canonicalJSONString(model.Expression), 56),
			Status:     string(model.Status),
			Salience:   salienceForRank(idx),
			Confidence: model.Confidence,
			Markers:    nodeMarkers(salienceForRank(idx), model.Confidence, false, false),
		}, "hypothesizes", model.Confidence)
	}

	for idx, loop := range sortedLoopsForSnapshot(input.Packet.OpenLoops) {
		blocker := loop.State == domain.LoopBlocked || strings.TrimSpace(loop.Blocker) != ""
		appendRailNode(compiledContextSnapshotNode{
			ID:         "loop:" + loop.ID,
			Type:       "open_loop",
			Rail:       "loops",
			Label:      truncateSnapshotText(nonEmpty(loop.Title, loop.ID), 48),
			Detail:     truncateSnapshotText(nonEmpty(loop.NextAction, loop.Blocker), 56),
			Status:     string(loop.State),
			Salience:   salienceForRank(idx),
			Confidence: 1.0,
			Blocker:    blocker,
			Markers:    nodeMarkers(salienceForRank(idx), 1.0, blocker, false),
		}, "tracks", 1.0)
	}

	for _, link := range sortedLinksForSnapshot(input.Packet.LinkedNotes) {
		sourceID := snapshotObjectNodeID(link.SourceID, nodeKinds)
		targetID := snapshotObjectNodeID(link.TargetID, nodeKinds)
		if sourceID == "" || targetID == "" {
			continue
		}
		edges = append(edges, compiledContextSnapshotEdge{
			ID:         "link:" + link.ID,
			Type:       string(link.Type),
			SourceID:   sourceID,
			TargetID:   targetID,
			Confidence: link.Confidence,
		})
	}

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	for idx := range rails {
		sort.Strings(rails[idx].NodeIDs)
	}

	current := compiledContextSnapshot{
		Header: compiledContextSnapshotHeader{
			SnapshotID:   input.SnapshotID,
			PacketID:     input.Packet.ID,
			SnapshotKind: input.SnapshotKind,
			Query:        input.Packet.Query,
			Scope:        input.Packet.Scope,
			Budget:       input.Packet.Budget,
			Counts: compiledSnapshotCounts{
				Constraints: len(rails[0].NodeIDs),
				Evidence:    len(rails[1].NodeIDs),
				Hypotheses:  len(rails[2].NodeIDs),
				Loops:       len(rails[3].NodeIDs),
				Nodes:       len(nodes) + 1,
				Edges:       len(edges),
			},
			Lineage: compiledSnapshotLineage{
				Scope:         input.Packet.Scope,
				CorrelationID: input.CorrelationID,
				TraceID:       input.TraceID,
				SyscallID:     input.SyscallID,
				AuditID:       input.AuditID,
				ProposedBy:    input.ProposedBy,
				CommittedBy:   input.CommittedBy,
			},
			CreatedAt: input.Packet.CreatedAt,
		},
		Graph: compiledContextSnapshotGraph{
			Objective: objective,
			Rails:     rails,
			Nodes:     nodes,
			Edges:     edges,
		},
		Delta: compiledContextSnapshotDelta{
			AddedNodeIDs:   []string{},
			RemovedNodeIDs: []string{},
			ChangedNodeIDs: []string{},
			AddedEdgeIDs:   []string{},
			RemovedEdgeIDs: []string{},
			ChangedEdgeIDs: []string{},
		},
	}
	current.Header.Fingerprint = compiledContextSnapshotFingerprint(current)
	if prior != nil {
		current.Header.ParentSnapshotID = strings.TrimSpace(prior.Header.SnapshotID)
		current.Delta = computeCompiledSnapshotDelta(current, *prior)
		current.Delta.FingerprintMatched = strings.TrimSpace(prior.Header.Fingerprint) != "" &&
			strings.TrimSpace(prior.Header.Fingerprint) == strings.TrimSpace(current.Header.Fingerprint)
	}
	return current
}

func compiledContextSnapshotFingerprint(snapshot compiledContextSnapshot) string {
	canonical := map[string]any{
		"snapshotKind": snapshot.Header.SnapshotKind,
		"query":        snapshot.Header.Query,
		"scope":        snapshot.Header.Scope,
		"budget":       snapshot.Header.Budget,
		"objective":    snapshot.Graph.Objective,
		"rails":        snapshot.Graph.Rails,
		"nodes":        snapshot.Graph.Nodes,
		"edges":        snapshot.Graph.Edges,
	}
	sum := sha1.Sum([]byte(canonicalJSONString(canonical)))
	return "snapfp-" + hex.EncodeToString(sum[:12])
}

func computeCompiledSnapshotDelta(current, prior compiledContextSnapshot) compiledContextSnapshotDelta {
	delta := compiledContextSnapshotDelta{
		AddedNodeIDs:   []string{},
		RemovedNodeIDs: []string{},
		ChangedNodeIDs: []string{},
		AddedEdgeIDs:   []string{},
		RemovedEdgeIDs: []string{},
		ChangedEdgeIDs: []string{},
	}
	currentNodes := map[string]string{}
	priorNodes := map[string]string{}
	for _, node := range current.Graph.Nodes {
		currentNodes[node.ID] = canonicalJSONString(node)
	}
	for _, node := range prior.Graph.Nodes {
		priorNodes[node.ID] = canonicalJSONString(node)
	}
	for id, encoded := range currentNodes {
		if prev, ok := priorNodes[id]; !ok {
			delta.AddedNodeIDs = append(delta.AddedNodeIDs, id)
		} else if prev != encoded {
			delta.ChangedNodeIDs = append(delta.ChangedNodeIDs, id)
		}
	}
	for id := range priorNodes {
		if _, ok := currentNodes[id]; !ok {
			delta.RemovedNodeIDs = append(delta.RemovedNodeIDs, id)
		}
	}

	currentEdges := map[string]string{}
	priorEdges := map[string]string{}
	for _, edge := range current.Graph.Edges {
		currentEdges[edge.ID] = canonicalJSONString(edge)
	}
	for _, edge := range prior.Graph.Edges {
		priorEdges[edge.ID] = canonicalJSONString(edge)
	}
	for id, encoded := range currentEdges {
		if prev, ok := priorEdges[id]; !ok {
			delta.AddedEdgeIDs = append(delta.AddedEdgeIDs, id)
		} else if prev != encoded {
			delta.ChangedEdgeIDs = append(delta.ChangedEdgeIDs, id)
		}
	}
	for id := range priorEdges {
		if _, ok := currentEdges[id]; !ok {
			delta.RemovedEdgeIDs = append(delta.RemovedEdgeIDs, id)
		}
	}

	sort.Strings(delta.AddedNodeIDs)
	sort.Strings(delta.RemovedNodeIDs)
	sort.Strings(delta.ChangedNodeIDs)
	sort.Strings(delta.AddedEdgeIDs)
	sort.Strings(delta.RemovedEdgeIDs)
	sort.Strings(delta.ChangedEdgeIDs)
	return delta
}

func compiledContextSnapshotToDomain(snapshot compiledContextSnapshot) *domain.ContextRestoreSnapshot {
	evidence := map[string]any{}
	metadata := map[string]any{
		"fingerprint":                snapshot.Header.Fingerprint,
		"parent_snapshot_id":         snapshot.Header.ParentSnapshotID,
		"restore_source_snapshot_id": snapshot.Header.ParentSnapshotID,
		"restore_scope_json":         snapshot.Header.Scope,
		"restore_reason_json": map[string]any{
			"mode":                "compile_context",
			"query":               snapshot.Header.Query,
			"fingerprint_matched": snapshot.Delta.FingerprintMatched,
		},
	}
	if snapshot.Header.RenderedCardArtifactID != "" {
		metadata["rendered_card_artifact_id"] = snapshot.Header.RenderedCardArtifactID
	}
	assignSnapshotMap(evidence, "header", snapshot.Header)
	assignSnapshotMap(evidence, "graph", snapshot.Graph)
	assignSnapshotMap(evidence, "delta", snapshot.Delta)
	return &domain.ContextRestoreSnapshot{
		SnapshotID:   snapshot.Header.SnapshotID,
		SnapshotKind: snapshot.Header.SnapshotKind,
		Evidence:     evidence,
		Metadata:     metadata,
	}
}

func compiledContextSnapshotFromDomain(snapshot *domain.ContextRestoreSnapshot) (compiledContextSnapshot, bool) {
	header, ok := compiledContextSnapshotHeaderFromDomain(snapshot)
	if !ok {
		return compiledContextSnapshot{}, false
	}

	graph := compiledContextSnapshotGraph{
		Objective: compiledContextSnapshotNode{
			ID:    objectiveNodeID(header.Query),
			Type:  "objective",
			Label: truncateSnapshotText(header.Query, 64),
		},
		Rails: []compiledContextSnapshotRail{
			{Name: "constraints", Label: "Constraints"},
			{Name: "evidence", Label: "Evidence"},
			{Name: "hypotheses", Label: "Hypotheses"},
			{Name: "loops", Label: "Loops"},
		},
		Nodes: []compiledContextSnapshotNode{},
		Edges: []compiledContextSnapshotEdge{},
	}
	if snapshot != nil && snapshot.Evidence != nil {
		if graphRaw, hasGraph := snapshot.Evidence["graph"]; hasGraph {
			var decoded compiledContextSnapshotGraph
			if decodeSnapshotValue(graphRaw, &decoded) {
				graph = decoded
			}
		}
	}
	delta := compiledContextSnapshotDelta{
		AddedNodeIDs:   []string{},
		RemovedNodeIDs: []string{},
		ChangedNodeIDs: []string{},
		AddedEdgeIDs:   []string{},
		RemovedEdgeIDs: []string{},
		ChangedEdgeIDs: []string{},
	}
	if snapshot != nil && snapshot.Evidence != nil {
		if deltaRaw, hasDelta := snapshot.Evidence["delta"]; hasDelta {
			_ = decodeSnapshotValue(deltaRaw, &delta)
		}
	}
	return compiledContextSnapshot{
		Header: header,
		Graph:  graph,
		Delta:  delta,
	}, true
}

func compiledContextSnapshotHeaderFromDomain(snapshot *domain.ContextRestoreSnapshot) (compiledContextSnapshotHeader, bool) {
	if snapshot == nil {
		return compiledContextSnapshotHeader{}, false
	}
	header := compiledContextSnapshotHeader{
		SnapshotID:   strings.TrimSpace(snapshot.SnapshotID),
		SnapshotKind: strings.TrimSpace(snapshot.SnapshotKind),
	}
	if snapshot.Evidence != nil {
		if headerRaw, ok := snapshot.Evidence["header"]; ok {
			var decoded compiledContextSnapshotHeader
			if decodeSnapshotValue(headerRaw, &decoded) {
				header = decoded
			}
		}
	}
	if header.SnapshotID == "" {
		header.SnapshotID = strings.TrimSpace(snapshot.SnapshotID)
	}
	if header.SnapshotKind == "" {
		header.SnapshotKind = strings.TrimSpace(snapshot.SnapshotKind)
	}
	if snapshot.Metadata != nil {
		if header.Fingerprint == "" {
			header.Fingerprint = strings.TrimSpace(readString(snapshot.Metadata, "fingerprint"))
		}
		if header.ParentSnapshotID == "" {
			header.ParentSnapshotID = strings.TrimSpace(nonEmpty(
				readString(snapshot.Metadata, "parent_snapshot_id"),
				readString(snapshot.Metadata, "restore_source_snapshot_id"),
			))
		}
		if header.Query == "" {
			header.Query = strings.TrimSpace(readString(snapshot.Metadata, "query"))
		}
		if header.CreatedAt == 0 {
			header.CreatedAt = readInt64(snapshot.Metadata, "created_at")
		}
		if header.Scope.WorkspaceID == "" {
			if scopeRaw, ok := snapshot.Metadata["restore_scope_json"]; ok {
				_ = decodeSnapshotValue(scopeRaw, &header.Scope)
			}
		}
	}
	if header.SnapshotID == "" {
		return compiledContextSnapshotHeader{}, false
	}
	return header, true
}

func mergeCompileContextOptions(payload map[string]any) compileContextSnapshotOptions {
	opts := compileContextSnapshotOptions{}
	apply := func(src map[string]any) {
		if src == nil {
			return
		}
		if v, present, valid := readOptionalBool(src, "persistSnapshot"); present && valid {
			opts.PersistSnapshot = v
		}
		if v, present, valid := readOptionalBool(src, "renderSnapshotCard"); present && valid {
			opts.RenderSnapshotCard = v
		}
		if snapshotKind := readString(src, "snapshotKind"); snapshotKind != "" {
			opts.SnapshotKind = snapshotKind
		}
		if snapshotKind := readString(src, "restoreSnapshotKind"); snapshotKind != "" {
			opts.SnapshotKind = snapshotKind
		}
		if v, present, valid := readOptionalBool(src, "restoreMode"); present && valid {
			opts.RestoreMode = v
		}
		if v := readInt(src, "restoreCandidateLimit", 0); v > 0 {
			opts.RestoreCandidateLimit = v
		}
		if v := readFloat(src, "restoreMinScore", 0); v > 0 {
			opts.RestoreMinScore = clamp01(v)
		}
		if v, present, valid := readOptionalBool(src, "requireFreshCompileBelowThreshold"); present && valid {
			opts.RequireFreshCompileBelowThreshold = v
		}
		if v, present, valid := readOptionalBool(src, "expandRestoreGraph"); present && valid {
			opts.ExpandRestoreGraph = v
		}
		if v, present, valid := readOptionalBool(src, "restoreCacheDisabled"); present && valid {
			opts.RestoreCacheDisabled = v
		}
	}
	apply(payload)
	if raw, ok := payload["restoreSnapshot"].(map[string]any); ok {
		apply(raw)
	}
	if raw, ok := payload["compileOptions"].(map[string]any); ok {
		apply(raw)
	}
	return opts
}

func contextSnapshotArtifactRef(packet domain.ContextPacket, snapshot compiledContextSnapshot, svg string) domain.ArtifactRef {
	sum := sha1.Sum([]byte(svg))
	return domain.ArtifactRef{
		ID:          packet.ID + ":snapshot_card",
		Type:        "context_snapshot_card",
		URI:         "artifact://context_snapshot/" + packet.ID + "/card.svg",
		Scope:       packet.Scope,
		ContentHash: "sha1:" + hex.EncodeToString(sum[:]),
		CreatedAt:   packet.CreatedAt,
		Provenance: domain.Provenance{
			Actor:     snapshot.Header.Lineage.ProposedBy,
			ActorType: "syscall",
			Source:    "forge_kernel",
			TraceID:   snapshot.Header.Lineage.TraceID,
		},
		Metadata: map[string]any{
			"kind":         "context_snapshot_card",
			"mimeType":     "image/svg+xml",
			"title":        "Context Snapshot Card",
			"snapshotId":   snapshot.Header.SnapshotID,
			"snapshotKind": snapshot.Header.SnapshotKind,
			"svg":          svg,
		},
	}
}

func applyCompiledSnapshotToPacket(packet *domain.ContextPacket, snapshot compiledContextSnapshot, opts compileContextSnapshotOptions) {
	if packet == nil {
		return
	}
	packet.CompileOptions = &domain.ContextCompileOptions{
		PersistSnapshot:    opts.PersistSnapshot,
		RenderSnapshotCard: opts.RenderSnapshotCard,
		SnapshotKind:       opts.SnapshotKind,
	}
	packet.RestoreSnapshot = compiledContextSnapshotToDomain(snapshot)
}

func assignSnapshotMap(dst map[string]any, key string, value any) {
	raw, _ := json.Marshal(value)
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err == nil {
		dst[key] = decoded
		return
	}
	var decodedList []any
	if err := json.Unmarshal(raw, &decodedList); err == nil {
		dst[key] = decodedList
		return
	}
	dst[key] = value
}

func decodeSnapshotValue(raw any, out any) bool {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return false
	}
	return json.Unmarshal(encoded, out) == nil
}

func canonicalJSONString(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func readInt64(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	default:
		return 0
	}
}

func objectiveNodeID(query string) string {
	return stableSnapshotID("objective", strings.TrimSpace(query))
}

func stableSnapshotID(parts ...string) string {
	normalized := strings.Join(parts, "|")
	sum := sha1.Sum([]byte(normalized))
	return hex.EncodeToString(sum[:8])
}

func truncateSnapshotText(in string, max int) string {
	in = strings.TrimSpace(in)
	if max <= 0 || len(in) <= max {
		return in
	}
	if max <= 3 {
		return in[:max]
	}
	return in[:max-3] + "..."
}

func salienceForRank(rank int) string {
	switch {
	case rank <= 1:
		return "high"
	case rank <= 4:
		return "medium"
	default:
		return "low"
	}
}

func nodeMarkers(salience string, confidence float64, blocker, conflict bool) []string {
	markers := []string{}
	if salience != "" {
		markers = append(markers, "salience:"+salience)
	}
	if confidence > 0 {
		markers = append(markers, fmt.Sprintf("confidence:%.2f", confidence))
	}
	if blocker {
		markers = append(markers, "blocker")
	}
	if conflict {
		markers = append(markers, "conflict")
	}
	return markers
}

func conflictedNoteIDs(links []domain.SemanticLink) map[string]bool {
	out := map[string]bool{}
	for _, link := range links {
		if link.Type != domain.LinkContradicts {
			continue
		}
		out[link.SourceID] = true
		out[link.TargetID] = true
	}
	return out
}

func snapshotObjectNodeID(objectID string, nodeKinds map[string]string) string {
	if objectID == "" {
		return ""
	}
	prefixes := []string{"note:", "artifact:", "event:", "model:", "loop:", "state:"}
	for _, prefix := range prefixes {
		candidate := prefix + objectID
		if _, ok := nodeKinds[candidate]; ok {
			return candidate
		}
	}
	return ""
}

func sortedNotesForSnapshot(in []domain.MemoryNote) []domain.MemoryNote {
	out := append([]domain.MemoryNote{}, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedArtifactsForSnapshot(in []domain.ArtifactRef) []domain.ArtifactRef {
	out := append([]domain.ArtifactRef{}, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedEventsForSnapshot(in []domain.JournalEvent) []domain.JournalEvent {
	out := append([]domain.JournalEvent{}, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedModelsForSnapshot(in []domain.AdaptivePolicyModel) []domain.AdaptivePolicyModel {
	out := append([]domain.AdaptivePolicyModel{}, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedLoopsForSnapshot(in []domain.OpenLoop) []domain.OpenLoop {
	out := append([]domain.OpenLoop{}, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedLinksForSnapshot(in []domain.SemanticLink) []domain.SemanticLink {
	out := append([]domain.SemanticLink{}, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func isSnapshotCardArtifact(ref domain.ArtifactRef) bool {
	return strings.TrimSpace(ref.Type) == "context_snapshot_card" || strings.TrimSpace(readString(ref.Metadata, "kind")) == "context_snapshot_card"
}

func isCompileContextJournalEvent(evt domain.JournalEvent) bool {
	return strings.EqualFold(strings.TrimSpace(evt.Type), "semantic_syscall.compile_context")
}
