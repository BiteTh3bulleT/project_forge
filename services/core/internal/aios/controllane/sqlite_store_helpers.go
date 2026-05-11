package controllane

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

// internal helpers
func (s *SQLiteSemanticStore) ensureProvenance(ctx context.Context, scope domain.ForgeScope, prov domain.Provenance, metadata map[string]any, createdAt int64) (string, error) {
	id := provenanceID(scope, prov)
	_, err := s.exec.ExecContext(ctx, `
INSERT INTO provenance_records(
  id, actor, actor_type, source, trace_id, workspace_id, lane_id, selected_paths_json, metadata_json, created_at,
  proposed_by, committed_by, syscall_id, correlation_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO NOTHING`,
		id, prov.Actor, prov.ActorType, prov.Source, nonEmpty(prov.TraceID, s.meta.TraceID), scope.WorkspaceID, scope.LaneID, encodeStringSlice(scope.SelectedPaths), encodeJSON(nonNilMap(metadata)), createdAt,
		string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, "",
	)
	return id, err
}

func provenanceID(scope domain.ForgeScope, prov domain.Provenance) string {
	key := strings.Join([]string{
		scope.WorkspaceID,
		scope.LaneID,
		prov.Actor,
		prov.ActorType,
		prov.Source,
		prov.TraceID,
	}, "|")
	sum := sha1.Sum([]byte(key))
	return "prov-" + hex.EncodeToString(sum[:12])
}

func toScopeFilter(scope domain.ForgeScope) ScopeFilter {
	return ScopeFilter{WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID}
}

func encodeJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func encodeStringSlice(v []string) string {
	if v == nil {
		v = []string{}
	}
	return encodeJSON(v)
}

func decodeStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	if out == nil {
		return []string{}
	}
	return out
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func (s *SQLiteSemanticStore) detectObjectKind(ctx context.Context, id string) string {
	if id == "" {
		return "unknown"
	}
	check := func(table, kind string) (string, bool) {
		var one int
		err := s.exec.QueryRowContext(ctx, fmt.Sprintf("SELECT 1 FROM %s WHERE id = ? LIMIT 1", table), id).Scan(&one)
		return kind, err == nil
	}
	if kind, ok := check("memory_notes", "memory_note"); ok {
		return kind
	}
	if kind, ok := check("state_items", "state_item"); ok {
		return kind
	}
	if kind, ok := check("open_loops", "open_loop"); ok {
		return kind
	}
	if kind, ok := check("derived_models", "derived_model"); ok {
		return kind
	}
	if kind, ok := check("artifact_refs", "artifact_ref"); ok {
		return kind
	}
	if kind, ok := check("journal_events", "journal_event"); ok {
		return kind
	}
	return "object"
}

func (s *SQLiteSemanticStore) listLinks(ctx context.Context, where string, args []any, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	if limit <= 0 {
		limit = 200
	}
	base := `
SELECT id, type, source_id, target_id, workspace_id, lane_id, selected_paths_json, confidence, provenance_json, created_at
FROM semantic_links
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND ` + where + `
ORDER BY created_at DESC
LIMIT ?`
	qargs := []any{scope.WorkspaceID, scope.LaneID, scope.LaneID}
	qargs = append(qargs, args...)
	qargs = append(qargs, limit)
	rows, err := s.exec.QueryContext(ctx, base, qargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLinkRows(rows)
}

func (s *SQLiteSemanticStore) listLoops(ctx context.Context, where string, args []any, scope ScopeFilter, limit int) ([]domain.OpenLoop, error) {
	if limit <= 0 {
		limit = 200
	}
	base := `
SELECT id, title, state, workspace_id, lane_id, selected_paths_json, priority, owner, blocker, next_action, related_notes_json, created_from, created_at, updated_at
FROM open_loops
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND ` + where + `
ORDER BY updated_at DESC
LIMIT ?`
	qargs := []any{scope.WorkspaceID, scope.LaneID, scope.LaneID}
	qargs = append(qargs, args...)
	qargs = append(qargs, limit)
	rows, err := s.exec.QueryContext(ctx, base, qargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLoopRows(rows)
}

func (s *SQLiteSemanticStore) listModels(ctx context.Context, where string, args []any, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error) {
	if limit <= 0 {
		limit = 200
	}
	base := `
SELECT id, type, expression_json, derived_from_json, support_count, confidence, status, workspace_id, lane_id, selected_paths_json, last_validated_at, created_at
FROM derived_models
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND ` + where + `
ORDER BY created_at DESC
LIMIT ?`
	qargs := []any{scope.WorkspaceID, scope.LaneID, scope.LaneID}
	qargs = append(qargs, args...)
	qargs = append(qargs, limit)
	rows, err := s.exec.QueryContext(ctx, base, qargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanModelRows(rows)
}

func (s *SQLiteSemanticStore) listSupersessions(ctx context.Context, where string, args []any, scope ScopeFilter, limit int) ([]SupersessionRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	base := `
SELECT id, old_object_id, old_object_kind, new_object_id, new_object_kind, reason, workspace_id, correlation_id, trace_id,
       syscall_id, audit_id, proposed_by, committed_by, metadata_json, created_at, provenance_json
FROM supersession_records
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND ` + where + `
ORDER BY created_at DESC
LIMIT ?`
	qargs := []any{scope.WorkspaceID, scope.LaneID, scope.LaneID}
	qargs = append(qargs, args...)
	qargs = append(qargs, limit)
	rows, err := s.exec.QueryContext(ctx, base, qargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSupersessionRows(rows)
}

func contextSnapshotMetadata(pkt domain.ContextPacket, metadata map[string]any) map[string]any {
	out := cloneMap(metadata)
	if pkt.CompileOptions != nil {
		out["compileOptions"] = pkt.CompileOptions
		if strings.TrimSpace(pkt.CompileOptions.SnapshotKind) != "" {
			out["snapshot_kind"] = strings.TrimSpace(pkt.CompileOptions.SnapshotKind)
		}
	}
	if pkt.RestoreSnapshot != nil {
		out["restoreSnapshot"] = pkt.RestoreSnapshot
		if strings.TrimSpace(pkt.RestoreSnapshot.SnapshotKind) != "" {
			out["snapshot_kind"] = strings.TrimSpace(pkt.RestoreSnapshot.SnapshotKind)
		}
		if strings.TrimSpace(pkt.RestoreSnapshot.SnapshotID) != "" {
			out["snapshot_id"] = strings.TrimSpace(pkt.RestoreSnapshot.SnapshotID)
		}
		if v := readString(pkt.RestoreSnapshot.Metadata, "restore_source_snapshot_id"); v != "" {
			out["restore_source_snapshot_id"] = v
		}
		if v := pkt.RestoreSnapshot.Metadata["restore_scope_json"]; v != nil {
			out["restore_scope_json"] = v
		}
		if v := pkt.RestoreSnapshot.Metadata["restore_reason_json"]; v != nil {
			out["restore_reason_json"] = v
		}
		if v := pkt.RestoreSnapshot.Metadata["restore_trace_json"]; v != nil {
			out["restore_trace_json"] = v
		}
		if v := pkt.RestoreSnapshot.Metadata["restore_trace"]; v != nil {
			out["restore_trace"] = v
		}
		if v := pkt.RestoreSnapshot.Metadata["restore_package_json"]; v != nil {
			out["restore_package_json"] = v
		}
		if v := pkt.RestoreSnapshot.Metadata["restore_package"]; v != nil {
			out["restore_package"] = v
		}
		if v := readString(pkt.RestoreSnapshot.Metadata, "rendered_card_artifact_id"); v != "" {
			out["rendered_card_artifact_id"] = v
		}
	}
	return out
}

type contextSnapshotColumns struct {
	SnapshotKind        string
	SnapshotFingerprint string
	ParentSnapshotID    string
	HeaderJSON          string
	GraphJSON           string
	DeltaJSON           string
	RestoreScoresJSON   string
	RenderArtifactRefID string
	ResumeHintsJSON     string
}

func buildContextSnapshotColumns(pkt domain.ContextPacket, metadata map[string]any) contextSnapshotColumns {
	cols := contextSnapshotColumns{
		SnapshotKind:        strings.TrimSpace(readString(metadata, "snapshot_kind")),
		SnapshotFingerprint: strings.TrimSpace(readString(metadata, "snapshot_fingerprint")),
		ParentSnapshotID:    strings.TrimSpace(readString(metadata, "parent_snapshot_id")),
		HeaderJSON:          encodeAnyJSONOrDefault(metadata["header_json"], "{}"),
		GraphJSON:           encodeAnyJSONOrDefault(metadata["graph_json"], "{}"),
		DeltaJSON:           encodeAnyJSONOrDefault(metadata["delta_json"], "{}"),
		RestoreScoresJSON:   encodeAnyJSONOrDefault(metadata["restore_scores_json"], "{}"),
		RenderArtifactRefID: strings.TrimSpace(readString(metadata, "render_artifact_ref_id")),
		ResumeHintsJSON:     encodeAnyJSONOrDefault(metadata["resume_hints_json"], "{}"),
	}
	if pkt.CompileOptions != nil && strings.TrimSpace(pkt.CompileOptions.SnapshotKind) != "" {
		cols.SnapshotKind = strings.TrimSpace(pkt.CompileOptions.SnapshotKind)
	}
	if pkt.RestoreSnapshot != nil {
		if strings.TrimSpace(pkt.RestoreSnapshot.SnapshotKind) != "" {
			cols.SnapshotKind = strings.TrimSpace(pkt.RestoreSnapshot.SnapshotKind)
		}
		if v := strings.TrimSpace(readString(pkt.RestoreSnapshot.Metadata, "fingerprint")); v != "" {
			cols.SnapshotFingerprint = v
		}
		if v := strings.TrimSpace(readString(pkt.RestoreSnapshot.Metadata, "parent_snapshot_id")); v != "" {
			cols.ParentSnapshotID = v
		}
		if v := strings.TrimSpace(readString(pkt.RestoreSnapshot.Metadata, "restore_source_snapshot_id")); v != "" && cols.ParentSnapshotID == "" {
			cols.ParentSnapshotID = v
		}
		if v := strings.TrimSpace(readString(pkt.RestoreSnapshot.Metadata, "rendered_card_artifact_id")); v != "" {
			cols.RenderArtifactRefID = v
		}
		if v := encodeAnyJSONOrDefault(pkt.RestoreSnapshot.Evidence["header"], "{}"); v != "{}" {
			cols.HeaderJSON = v
		}
		if v := encodeAnyJSONOrDefault(pkt.RestoreSnapshot.Evidence["graph"], "{}"); v != "{}" {
			cols.GraphJSON = v
		}
		if v := encodeAnyJSONOrDefault(pkt.RestoreSnapshot.Evidence["delta"], "{}"); v != "{}" {
			cols.DeltaJSON = v
		}
		if v := encodeAnyJSONOrDefault(pkt.RestoreSnapshot.Metadata["restore_scores_json"], "{}"); v != "{}" {
			cols.RestoreScoresJSON = v
		}
		if v := encodeAnyJSONOrDefault(pkt.RestoreSnapshot.Metadata["resume_hints_json"], "{}"); v != "{}" {
			cols.ResumeHintsJSON = v
		}
	}
	if cols.RenderArtifactRefID == "" {
		cols.RenderArtifactRefID = strings.TrimSpace(readString(metadata, "rendered_card_artifact_id"))
	}
	return cols
}

func encodeAnyJSONOrDefault(v any, fallback string) string {
	switch value := v.(type) {
	case nil:
		return fallback
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fallback
		}
		if json.Valid([]byte(trimmed)) {
			return trimmed
		}
		raw, _ := json.Marshal(value)
		if string(raw) == "null" || len(raw) == 0 {
			return fallback
		}
		return string(raw)
	default:
		raw, err := json.Marshal(value)
		if err != nil || string(raw) == "null" || len(raw) == 0 {
			return fallback
		}
		return string(raw)
	}
}

func decodeJSONAny(raw string) (any, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return nil, false
	}
	var out any
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return nil, false
	}
	return out, true
}

func hasSnapshotJSONContent(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	switch trimmed {
	case "", "null", "{}", "[]":
		return false
	default:
		return true
	}
}

func applyContextSnapshotColumns(pkt *domain.ContextPacket, cols contextSnapshotColumns) {
	if pkt == nil {
		return
	}
	snapshotKind := strings.TrimSpace(cols.SnapshotKind)
	if snapshotKind != "" {
		if pkt.CompileOptions == nil {
			pkt.CompileOptions = &domain.ContextCompileOptions{}
		}
		pkt.CompileOptions.SnapshotKind = snapshotKind
	}
	hasEvidence := hasSnapshotJSONContent(cols.HeaderJSON) || hasSnapshotJSONContent(cols.GraphJSON) || hasSnapshotJSONContent(cols.DeltaJSON)
	hasMetadata := strings.TrimSpace(cols.SnapshotFingerprint) != "" || strings.TrimSpace(cols.ParentSnapshotID) != "" ||
		strings.TrimSpace(cols.RenderArtifactRefID) != "" || hasSnapshotJSONContent(cols.RestoreScoresJSON) || hasSnapshotJSONContent(cols.ResumeHintsJSON)
	if snapshotKind == "" && !hasEvidence && !hasMetadata {
		return
	}
	if pkt.RestoreSnapshot == nil {
		pkt.RestoreSnapshot = &domain.ContextRestoreSnapshot{
			SnapshotID:   pkt.ID,
			SnapshotKind: snapshotKind,
			Evidence:     map[string]any{},
			Metadata:     map[string]any{},
		}
	}
	if pkt.RestoreSnapshot.Evidence == nil {
		pkt.RestoreSnapshot.Evidence = map[string]any{}
	}
	if pkt.RestoreSnapshot.Metadata == nil {
		pkt.RestoreSnapshot.Metadata = map[string]any{}
	}
	if pkt.RestoreSnapshot.SnapshotID == "" {
		pkt.RestoreSnapshot.SnapshotID = pkt.ID
	}
	if snapshotKind != "" {
		pkt.RestoreSnapshot.SnapshotKind = snapshotKind
	}
	if hasSnapshotJSONContent(cols.HeaderJSON) {
		if v, ok := decodeJSONAny(cols.HeaderJSON); ok {
			pkt.RestoreSnapshot.Evidence["header"] = v
		}
	}
	if hasSnapshotJSONContent(cols.GraphJSON) {
		if v, ok := decodeJSONAny(cols.GraphJSON); ok {
			pkt.RestoreSnapshot.Evidence["graph"] = v
		}
	}
	if hasSnapshotJSONContent(cols.DeltaJSON) {
		if v, ok := decodeJSONAny(cols.DeltaJSON); ok {
			pkt.RestoreSnapshot.Evidence["delta"] = v
		}
	}
	if strings.TrimSpace(cols.SnapshotFingerprint) != "" {
		pkt.RestoreSnapshot.Metadata["fingerprint"] = strings.TrimSpace(cols.SnapshotFingerprint)
	}
	if strings.TrimSpace(cols.ParentSnapshotID) != "" {
		parent := strings.TrimSpace(cols.ParentSnapshotID)
		pkt.RestoreSnapshot.Metadata["parent_snapshot_id"] = parent
		pkt.RestoreSnapshot.Metadata["restore_source_snapshot_id"] = parent
	}
	if strings.TrimSpace(cols.RenderArtifactRefID) != "" {
		pkt.RestoreSnapshot.Metadata["rendered_card_artifact_id"] = strings.TrimSpace(cols.RenderArtifactRefID)
	}
	if hasSnapshotJSONContent(cols.RestoreScoresJSON) {
		if v, ok := decodeJSONAny(cols.RestoreScoresJSON); ok {
			pkt.RestoreSnapshot.Metadata["restore_scores_json"] = v
		}
	}
	if hasSnapshotJSONContent(cols.ResumeHintsJSON) {
		if v, ok := decodeJSONAny(cols.ResumeHintsJSON); ok {
			pkt.RestoreSnapshot.Metadata["resume_hints_json"] = v
		}
	}
}

func applyContextSnapshotMetadata(pkt *domain.ContextPacket, raw string) {
	if pkt == nil || strings.TrimSpace(raw) == "" {
		return
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil || metadata == nil {
		return
	}
	if v, ok := metadata["compileOptions"]; ok {
		var opts domain.ContextCompileOptions
		if decodeSnapshotValue(v, &opts) {
			pkt.CompileOptions = &opts
		}
	}
	if v, ok := metadata["restoreSnapshot"]; ok {
		var snapshot domain.ContextRestoreSnapshot
		if decodeSnapshotValue(v, &snapshot) {
			pkt.RestoreSnapshot = &snapshot
		}
	}
	if pkt.CompileOptions == nil {
		if snapshotKind := readString(metadata, "snapshot_kind"); snapshotKind != "" {
			pkt.CompileOptions = &domain.ContextCompileOptions{SnapshotKind: snapshotKind}
		}
	}
	if pkt.RestoreSnapshot == nil {
		if snapshotKind := readString(metadata, "snapshot_kind"); snapshotKind != "" {
			pkt.RestoreSnapshot = &domain.ContextRestoreSnapshot{
				SnapshotID:   readString(metadata, "snapshot_id"),
				SnapshotKind: snapshotKind,
				Evidence:     map[string]any{},
				Metadata:     map[string]any{},
			}
		}
	}
}

func restoreOutcomeSelectSQL(where string) string {
	return `SELECT id, created_at, updated_at, workspace_id, lane_id, query, context_packet_id, snapshot_id, snapshot_kind,
       restore_score, requires_fresh_compile, selected_evidence_json, selected_state_keys_json, selected_loop_ids_json,
       selected_artifact_ids_json, outcome, outcome_confidence, operator_feedback, failure_reason, correction_summary,
       downstream_action_type, downstream_object_id, correlation_id, trace_id, syscall_id, audit_id, proposed_by,
       committed_by, metadata_json
FROM restore_outcome_events WHERE ` + where
}

func scanContextPacketRows(rows *sql.Rows) ([]domain.ContextPacket, error) {
	out := []domain.ContextPacket{}
	for rows.Next() {
		var pkt domain.ContextPacket
		var selected, budgetRaw, reasonsRaw, metadataRaw string
		var cols contextSnapshotColumns
		if err := rows.Scan(
			&pkt.ID,
			&pkt.Query,
			&pkt.Scope.WorkspaceID,
			&pkt.Scope.LaneID,
			&selected,
			&budgetRaw,
			&reasonsRaw,
			&pkt.CreatedAt,
			&metadataRaw,
			&cols.SnapshotKind,
			&cols.SnapshotFingerprint,
			&cols.ParentSnapshotID,
			&cols.HeaderJSON,
			&cols.GraphJSON,
			&cols.DeltaJSON,
			&cols.RestoreScoresJSON,
			&cols.RenderArtifactRefID,
			&cols.ResumeHintsJSON,
		); err != nil {
			return nil, err
		}
		pkt.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(budgetRaw), &pkt.Budget)
		_ = json.Unmarshal([]byte(reasonsRaw), &pkt.InclusionReasons)
		applyContextSnapshotMetadata(&pkt, metadataRaw)
		applyContextSnapshotColumns(&pkt, cols)
		out = append(out, pkt)
	}
	return out, rows.Err()
}

func scanRestoreOutcomeRows(rows *sql.Rows) ([]RestoreOutcomeEvent, error) {
	out := []RestoreOutcomeEvent{}
	for rows.Next() {
		var event RestoreOutcomeEvent
		var requiresFresh int
		var selectedEvidence, selectedStateKeys, selectedLoopIDs, selectedArtifactIDs, metadataRaw string
		var outcome string
		if err := rows.Scan(
			&event.ID, &event.CreatedAt, &event.UpdatedAt, &event.WorkspaceID, &event.LaneID, &event.Query, &event.ContextPacketID, &event.SnapshotID, &event.SnapshotKind,
			&event.RestoreScore, &requiresFresh, &selectedEvidence, &selectedStateKeys, &selectedLoopIDs, &selectedArtifactIDs, &outcome, &event.OutcomeConfidence,
			&event.OperatorFeedback, &event.FailureReason, &event.CorrectionSummary, &event.DownstreamActionType, &event.DownstreamObjectID,
			&event.CorrelationID, &event.TraceID, &event.SyscallID, &event.AuditID, &event.ProposedBy, &event.CommittedBy, &metadataRaw,
		); err != nil {
			return nil, err
		}
		event.RequiresFreshCompile = requiresFresh != 0
		event.SelectedEvidence = decodeStringSlice(selectedEvidence)
		event.SelectedStateKeys = decodeStringSlice(selectedStateKeys)
		event.SelectedLoopIDs = decodeStringSlice(selectedLoopIDs)
		event.SelectedArtifactIDs = decodeStringSlice(selectedArtifactIDs)
		event.Outcome = RestoreOutcome(outcome)
		event.Metadata = map[string]any{}
		if strings.TrimSpace(metadataRaw) != "" {
			_ = json.Unmarshal([]byte(metadataRaw), &event.Metadata)
		}
		out = append(out, normalizeRestoreOutcomeEvent(event))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanJournalRows(rows *sql.Rows) ([]domain.JournalEvent, error) {
	out := []domain.JournalEvent{}
	for rows.Next() {
		var evt domain.JournalEvent
		var payloadRaw, provRaw, selected, actor, trace string
		if err := rows.Scan(&evt.ID, &evt.Type, &evt.Source, &actor, &evt.Scope.WorkspaceID, &evt.Scope.LaneID, &selected, &payloadRaw, &evt.CorrelationID, &trace, &provRaw, &evt.Timestamp); err != nil {
			return nil, err
		}
		evt.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(payloadRaw), &evt.Payload)
		_ = json.Unmarshal([]byte(provRaw), &evt.Provenance)
		if evt.Provenance.Actor == "" {
			evt.Provenance.Actor = actor
		}
		if evt.Provenance.TraceID == "" {
			evt.Provenance.TraceID = trace
		}
		out = append(out, evt)
	}
	return out, rows.Err()
}

func scanNoteRows(rows *sql.Rows) ([]domain.MemoryNote, error) {
	out := []domain.MemoryNote{}
	for rows.Next() {
		var note domain.MemoryNote
		var typ, status, selected, provRaw string
		if err := rows.Scan(&note.ID, &typ, &note.Title, &note.Content, &note.Scope.WorkspaceID, &note.Scope.LaneID, &selected, &note.Confidence, &status, &note.CreatedAt, &note.UpdatedAt, &provRaw); err != nil {
			return nil, err
		}
		note.Type = domain.MemoryNoteType(typ)
		note.Status = domain.MemoryNoteStatus(status)
		note.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(provRaw), &note.Provenance)
		out = append(out, note)
	}
	return out, rows.Err()
}

func scanLinkRows(rows *sql.Rows) ([]domain.SemanticLink, error) {
	out := []domain.SemanticLink{}
	for rows.Next() {
		var link domain.SemanticLink
		var typ, selected, provRaw string
		if err := rows.Scan(&link.ID, &typ, &link.SourceID, &link.TargetID, &link.Scope.WorkspaceID, &link.Scope.LaneID, &selected, &link.Confidence, &provRaw, &link.CreatedAt); err != nil {
			return nil, err
		}
		link.Type = domain.SemanticLinkType(typ)
		link.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(provRaw), &link.Provenance)
		out = append(out, link)
	}
	return out, rows.Err()
}

func scanStateRows(rows *sql.Rows) ([]domain.StateItem, error) {
	out := []domain.StateItem{}
	for rows.Next() {
		var item domain.StateItem
		var valueRaw, status, derivedRaw, selected string
		if err := rows.Scan(&item.ID, &item.Key, &valueRaw, &item.Scope.WorkspaceID, &item.Scope.LaneID, &selected, &status, &derivedRaw, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Status = domain.StateItemStatus(status)
		item.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(valueRaw), &item.Value)
		_ = json.Unmarshal([]byte(derivedRaw), &item.DerivedFrom)
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanLoopRows(rows *sql.Rows) ([]domain.OpenLoop, error) {
	out := []domain.OpenLoop{}
	for rows.Next() {
		var loop domain.OpenLoop
		var st, relatedRaw, selected string
		if err := rows.Scan(&loop.ID, &loop.Title, &st, &loop.Scope.WorkspaceID, &loop.Scope.LaneID, &selected, &loop.Priority, &loop.Owner, &loop.Blocker, &loop.NextAction, &relatedRaw, &loop.CreatedFrom, &loop.CreatedAt, &loop.UpdatedAt); err != nil {
			return nil, err
		}
		loop.State = domain.OpenLoopState(st)
		loop.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(relatedRaw), &loop.RelatedNotes)
		out = append(out, loop)
	}
	return out, rows.Err()
}

func scanModelRows(rows *sql.Rows) ([]domain.AdaptivePolicyModel, error) {
	out := []domain.AdaptivePolicyModel{}
	for rows.Next() {
		var m domain.AdaptivePolicyModel
		var exprRaw, derivedRaw, status, selected string
		var lv sql.NullInt64
		if err := rows.Scan(&m.ID, &m.Type, &exprRaw, &derivedRaw, &m.SupportCount, &m.Confidence, &status, &m.Scope.WorkspaceID, &m.Scope.LaneID, &selected, &lv, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Status = domain.AdaptivePolicyModelStatus(status)
		m.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(exprRaw), &m.Expression)
		_ = json.Unmarshal([]byte(derivedRaw), &m.DerivedFrom)
		if lv.Valid {
			v := lv.Int64
			m.LastValidatedAt = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanArtifactRows(rows *sql.Rows) ([]domain.ArtifactRef, error) {
	out := []domain.ArtifactRef{}
	for rows.Next() {
		var ref domain.ArtifactRef
		var selected, provRaw, metaRaw string
		if err := rows.Scan(&ref.ID, &ref.Type, &ref.URI, &ref.ContentHash, &ref.Scope.WorkspaceID, &ref.Scope.LaneID, &selected, &provRaw, &ref.CreatedAt, &metaRaw); err != nil {
			return nil, err
		}
		ref.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(provRaw), &ref.Provenance)
		_ = json.Unmarshal([]byte(metaRaw), &ref.Metadata)
		out = append(out, ref)
	}
	return out, rows.Err()
}

func scanContradictionRows(rows *sql.Rows) ([]ContradictionRecord, error) {
	out := []ContradictionRecord{}
	for rows.Next() {
		var r ContradictionRecord
		var metaRaw, provRaw string
		if err := rows.Scan(&r.ID, &r.LeftID, &r.LeftKind, &r.RightID, &r.RightKind, &r.Reason, &r.Severity, &r.Confidence, &r.WorkspaceID, &r.CorrelationID, &r.TraceID, &r.SyscallID, &r.AuditID, &r.ProposedBy, &r.CommittedBy, &metaRaw, &r.CreatedAt, &provRaw); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(metaRaw), &r.Metadata)
		_ = json.Unmarshal([]byte(provRaw), &r.Provenance)
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanSupersessionRows(rows *sql.Rows) ([]SupersessionRecord, error) {
	out := []SupersessionRecord{}
	for rows.Next() {
		var r SupersessionRecord
		var metaRaw, provRaw string
		if err := rows.Scan(&r.ID, &r.OldID, &r.OldKind, &r.NewID, &r.NewKind, &r.Reason, &r.WorkspaceID, &r.CorrelationID, &r.TraceID, &r.SyscallID, &r.AuditID, &r.ProposedBy, &r.CommittedBy, &metaRaw, &r.CreatedAt, &provRaw); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(metaRaw), &r.Metadata)
		_ = json.Unmarshal([]byte(provRaw), &r.Provenance)
		out = append(out, r)
	}
	return out, rows.Err()
}

func encodeIDsNotes(in []domain.MemoryNote) string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, v.ID)
	}
	return encodeJSON(out)
}

func encodeIDsLinks(in []domain.SemanticLink) string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, v.ID)
	}
	return encodeJSON(out)
}

func encodeIDsState(in []domain.StateItem) string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, v.ID)
	}
	return encodeJSON(out)
}

func encodeIDsLoops(in []domain.OpenLoop) string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, v.ID)
	}
	return encodeJSON(out)
}

func encodeIDsModels(in []domain.AdaptivePolicyModel) string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, v.ID)
	}
	return encodeJSON(out)
}

func encodeIDsArtifacts(in []domain.ArtifactRef) string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, v.ID)
	}
	return encodeJSON(out)
}

func encodeIDsEvents(in []domain.JournalEvent) string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, v.ID)
	}
	return encodeJSON(out)
}
