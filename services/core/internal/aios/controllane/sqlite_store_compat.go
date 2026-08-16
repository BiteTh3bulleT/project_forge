package controllane

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/authproof"
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
)

func (s *SQLiteSemanticStore) CreateNote(note domain.MemoryNote) error {
	return s.Create(s.background, note)
}

func (s *SQLiteSemanticStore) UpdateNote(note domain.MemoryNote) error {
	ctx := s.background
	now := note.UpdatedAt
	archivedAt := sql.NullInt64{}
	if note.Status == domain.NoteArchived {
		archivedAt = sql.NullInt64{Int64: now, Valid: true}
	}
	provID, err := s.ensureProvenance(ctx, note.Scope, note.Provenance, nil, now)
	if err != nil {
		return err
	}
	scope := toScopeFilter(note.Scope)
	selected := encodeStringSlice(note.Scope.SelectedPaths)
	provJSON := encodeJSON(note.Provenance)
	metaJSON := encodeJSON(map[string]any{
		"proposedBy":  s.meta.Source,
		"committedBy": nonEmpty(s.meta.CommittedBy, "forge_kernel"),
	})
	res, err := s.exec.ExecContext(ctx, `
UPDATE memory_notes
SET type = ?, title = ?, content = ?, workspace_id = ?, lane_id = ?, selected_paths_json = ?,
    confidence = ?, status = ?, provenance_id = ?, provenance_json = ?, updated_at = ?,
    archived_at = ?, metadata_json = ?, proposed_by = ?, committed_by = ?, syscall_id = ?, correlation_id = ?, trace_id = ?
WHERE id = ?`,
		string(note.Type), note.Title, note.Content, scope.WorkspaceID, scope.LaneID, selected,
		note.Confidence, string(note.Status), provID, provJSON, now,
		archivedAt, metaJSON, string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, nonEmpty(s.meta.TraceID, note.Provenance.TraceID),
		note.ID,
	)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("note %q not found", note.ID)
	}
	return nil
}

func (s *SQLiteSemanticStore) CreateLink(link domain.SemanticLink) error {
	ctx := s.background
	now := link.CreatedAt
	provID, err := s.ensureProvenance(ctx, link.Scope, link.Provenance, nil, now)
	if err != nil {
		return err
	}
	sourceKind := s.detectObjectKind(ctx, link.SourceID)
	targetKind := s.detectObjectKind(ctx, link.TargetID)
	selected := encodeStringSlice(link.Scope.SelectedPaths)
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO semantic_links(
  id, type, source_id, source_kind, target_id, target_kind, confidence,
  provenance_id, provenance_json, workspace_id, lane_id, selected_paths_json,
  created_at, metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		link.ID, string(link.Type), link.SourceID, sourceKind, link.TargetID, targetKind, link.Confidence,
		provID, encodeJSON(link.Provenance), link.Scope.WorkspaceID, link.Scope.LaneID, selected,
		now, encodeJSON(map[string]any{}), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, nonEmpty(s.meta.TraceID, link.Provenance.TraceID), "",
	)
	return err
}

func (s *SQLiteSemanticStore) CreateState(state domain.StateItem) error {
	ctx := s.background
	scope := toScopeFilter(state.Scope)
	now := state.UpdatedAt
	var currentID string
	var currentVersion int
	var previousValueRaw string
	err := s.exec.QueryRowContext(ctx, `
SELECT id, current_version, value_json
FROM state_items
WHERE workspace_id = ? AND lane_id = ? AND key = ?`,
		scope.WorkspaceID, scope.LaneID, state.Key,
	).Scan(&currentID, &currentVersion, &previousValueRaw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	prev := map[string]any{}
	if err == nil {
		_ = json.Unmarshal([]byte(previousValueRaw), &prev)
		currentVersion++
		if _, err := s.exec.ExecContext(ctx, `
UPDATE state_items
SET value_json = ?, status = ?, derived_from_json = ?, current_version = ?, updated_at = ?,
    metadata_json = ?, proposed_by = ?, committed_by = ?, syscall_id = ?, correlation_id = ?, trace_id = ?
WHERE id = ?`,
			encodeJSON(state.Value), string(state.Status), encodeJSON(state.DerivedFrom), currentVersion, now,
			encodeJSON(map[string]any{}), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID,
			currentID,
		); err != nil {
			return err
		}
	} else {
		currentID = state.ID
		currentVersion = 1
		if _, err := s.exec.ExecContext(ctx, `
INSERT INTO state_items(
  id, key, value_json, workspace_id, lane_id, selected_paths_json, status, derived_from_json,
  current_version, updated_at, metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			currentID, state.Key, encodeJSON(state.Value), scope.WorkspaceID, scope.LaneID, encodeStringSlice(state.Scope.SelectedPaths), string(state.Status), encodeJSON(state.DerivedFrom),
			currentVersion, now, encodeJSON(map[string]any{}), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID, "",
		); err != nil {
			return err
		}
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO state_versions(
  state_item_id, state_key, workspace_id, lane_id, previous_value_json, new_value_json, changed_by,
  derived_from_json, syscall_id, audit_id, correlation_id, trace_id, created_at, metadata_json, proposed_by, committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		currentID, state.Key, scope.WorkspaceID, scope.LaneID, encodeJSON(prev), encodeJSON(state.Value), s.meta.ActorID,
		encodeJSON(state.DerivedFrom), s.meta.SyscallID, "", s.meta.CorrelationID, s.meta.TraceID, now, encodeJSON(map[string]any{}),
		string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"),
	)
	return err
}

func (s *SQLiteSemanticStore) CreateLoop(loop domain.OpenLoop) error {
	ctx := s.background
	resolvedAt := sql.NullInt64{}
	archivedAt := sql.NullInt64{}
	if loop.State == domain.LoopResolved {
		resolvedAt = sql.NullInt64{Int64: loop.UpdatedAt, Valid: true}
	}
	if loop.State == domain.LoopArchived {
		archivedAt = sql.NullInt64{Int64: loop.UpdatedAt, Valid: true}
	}
	scope := toScopeFilter(loop.Scope)
	_, err := s.exec.ExecContext(ctx, `
INSERT INTO open_loops(
  id, title, state, priority, owner, blocker, next_action, related_notes_json, created_from,
  workspace_id, lane_id, selected_paths_json, created_at, updated_at, resolved_at, archived_at,
  metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		loop.ID, loop.Title, string(loop.State), loop.Priority, loop.Owner, loop.Blocker, loop.NextAction, encodeJSON(loop.RelatedNotes), loop.CreatedFrom,
		scope.WorkspaceID, scope.LaneID, encodeStringSlice(loop.Scope.SelectedPaths), loop.CreatedAt, loop.UpdatedAt, resolvedAt, archivedAt,
		encodeJSON(map[string]any{}), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID, "",
	)
	return err
}

func (s *SQLiteSemanticStore) UpdateLoop(loop domain.OpenLoop) error {
	ctx := s.background
	resolvedAt := sql.NullInt64{}
	archivedAt := sql.NullInt64{}
	if loop.State == domain.LoopResolved {
		resolvedAt = sql.NullInt64{Int64: loop.UpdatedAt, Valid: true}
	}
	if loop.State == domain.LoopArchived {
		archivedAt = sql.NullInt64{Int64: loop.UpdatedAt, Valid: true}
	}
	res, err := s.exec.ExecContext(ctx, `
UPDATE open_loops
SET title = ?, state = ?, priority = ?, owner = ?, blocker = ?, next_action = ?, related_notes_json = ?,
    updated_at = ?, resolved_at = COALESCE(?, resolved_at), archived_at = COALESCE(?, archived_at),
    metadata_json = ?, proposed_by = ?, committed_by = ?, syscall_id = ?, correlation_id = ?, trace_id = ?
WHERE id = ?`,
		loop.Title, string(loop.State), loop.Priority, loop.Owner, loop.Blocker, loop.NextAction, encodeJSON(loop.RelatedNotes),
		loop.UpdatedAt, resolvedAt, archivedAt, encodeJSON(map[string]any{}), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID,
		loop.ID,
	)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("loop %q not found", loop.ID)
	}
	return nil
}

func (s *SQLiteSemanticStore) CreateModel(model domain.AdaptivePolicyModel) error {
	ctx := s.background
	_, err := s.exec.ExecContext(ctx, `
INSERT INTO derived_models(
  id, type, expression_json, derived_from_json, support_count, confidence, status,
  workspace_id, lane_id, selected_paths_json, last_validated_at, created_at, updated_at,
  metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		model.ID, model.Type, encodeJSON(model.Expression), encodeJSON(model.DerivedFrom), model.SupportCount, model.Confidence, string(model.Status),
		model.Scope.WorkspaceID, model.Scope.LaneID, encodeStringSlice(model.Scope.SelectedPaths), model.LastValidatedAt, model.CreatedAt, model.CreatedAt,
		encodeJSON(map[string]any{}), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID, "",
	)
	return err
}

func (s *SQLiteSemanticStore) UpdateModel(model domain.AdaptivePolicyModel) error {
	ctx := s.background
	res, err := s.exec.ExecContext(ctx, `
UPDATE derived_models
SET type = ?, expression_json = ?, derived_from_json = ?, support_count = ?, confidence = ?, status = ?,
    workspace_id = ?, lane_id = ?, selected_paths_json = ?, last_validated_at = ?, updated_at = ?,
    metadata_json = ?, proposed_by = ?, committed_by = ?, syscall_id = ?, correlation_id = ?, trace_id = ?
WHERE id = ?`,
		model.Type, encodeJSON(model.Expression), encodeJSON(model.DerivedFrom), model.SupportCount, model.Confidence, string(model.Status),
		model.Scope.WorkspaceID, model.Scope.LaneID, encodeStringSlice(model.Scope.SelectedPaths), model.LastValidatedAt, model.CreatedAt,
		encodeJSON(map[string]any{}), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID,
		model.ID,
	)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("model %q not found", model.ID)
	}
	return nil
}

func (s *SQLiteSemanticStore) CreateContradiction(record ContradictionRecord) error {
	scope := ScopeFilter{WorkspaceID: record.WorkspaceID, LaneID: record.LaneID}
	return s.CreateContradictionWithKinds(context.Background(), record, nonEmpty(record.LeftKind, "object"), nonEmpty(record.RightKind, "object"), scope)
}

func (s *SQLiteSemanticStore) CreateSupersession(record SupersessionRecord) error {
	scope := ScopeFilter{WorkspaceID: record.WorkspaceID, LaneID: record.LaneID}
	return s.CreateSupersessionWithKinds(context.Background(), record, nonEmpty(record.OldKind, "object"), nonEmpty(record.NewKind, "object"), scope)
}

func (s *SQLiteSemanticStore) SetIdempotency(key string, rec IdempotencyRecord) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return ErrInvalidIdempotencyRecord
	}
	if existing, ok := s.GetIdempotency(key); ok {
		if idempotencyRecordsMatch(existing, rec) {
			return nil
		}
		return fmt.Errorf("%w: key %q", ErrIdempotencyConflict, key)
	}
	if err := validateIdempotencyRecord(key, rec); err != nil {
		return err
	}
	result, err := s.exec.ExecContext(s.background, `
INSERT INTO semantic_idempotency_keys(
  idempotency_key, action, request_fingerprint, idempotency_fingerprint, result_json,
  request_json, plan_json, seal_json, receipt_json, authproof_json, created_at, correlation_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(idempotency_key) DO NOTHING`,
		key, string(rec.Action), rec.RequestFingerprint, rec.IdempotencyFingerprint, encodeJSON(rec.Result),
		encodeJSON(rec.Request), encodeJSON(rec.Plan), encodeJSON(rec.Seal), encodeJSON(rec.Receipt), encodeJSON(rec.AuthorizationProof),
		rec.CreatedAt, rec.CorrelationID,
	)
	if err != nil {
		return err
	}
	if inserted, _ := result.RowsAffected(); inserted == 1 {
		return nil
	}
	existing, ok := s.GetIdempotency(key)
	if ok && idempotencyRecordsMatch(existing, rec) {
		return nil
	}
	return fmt.Errorf("%w: key %q", ErrIdempotencyConflict, key)
}

func (s *SQLiteSemanticStore) FindNote(id string) (domain.MemoryNote, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id, type, title, content, workspace_id, lane_id, selected_paths_json, confidence, status, created_at, updated_at, provenance_json
FROM memory_notes WHERE id = ?`, id)
	var note domain.MemoryNote
	var typ, status, selected, provRaw string
	if err := row.Scan(&note.ID, &typ, &note.Title, &note.Content, &note.Scope.WorkspaceID, &note.Scope.LaneID, &selected, &note.Confidence, &status, &note.CreatedAt, &note.UpdatedAt, &provRaw); err != nil {
		return domain.MemoryNote{}, false
	}
	note.Type = domain.MemoryNoteType(typ)
	note.Status = domain.MemoryNoteStatus(status)
	note.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(provRaw), &note.Provenance)
	return note, true
}

func (s *SQLiteSemanticStore) FindLink(id string) (domain.SemanticLink, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id, type, source_id, target_id, workspace_id, lane_id, selected_paths_json, confidence, provenance_json, created_at
FROM semantic_links WHERE id = ?`, id)
	var link domain.SemanticLink
	var typ, selected, provRaw string
	if err := row.Scan(&link.ID, &typ, &link.SourceID, &link.TargetID, &link.Scope.WorkspaceID, &link.Scope.LaneID, &selected, &link.Confidence, &provRaw, &link.CreatedAt); err != nil {
		return domain.SemanticLink{}, false
	}
	link.Type = domain.SemanticLinkType(typ)
	link.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(provRaw), &link.Provenance)
	return link, true
}

func (s *SQLiteSemanticStore) FindState(id string) (domain.StateItem, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id, key, value_json, workspace_id, lane_id, selected_paths_json, status, derived_from_json, updated_at
FROM state_items
WHERE id = ?`, id)
	var item domain.StateItem
	var valueRaw, status, derivedRaw, selected string
	if err := row.Scan(&item.ID, &item.Key, &valueRaw, &item.Scope.WorkspaceID, &item.Scope.LaneID, &selected, &status, &derivedRaw, &item.UpdatedAt); err != nil {
		return domain.StateItem{}, false
	}
	item.Status = domain.StateItemStatus(status)
	item.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(valueRaw), &item.Value)
	_ = json.Unmarshal([]byte(derivedRaw), &item.DerivedFrom)
	return item, true
}

func (s *SQLiteSemanticStore) FindLoop(id string) (domain.OpenLoop, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id, title, state, workspace_id, lane_id, selected_paths_json, priority, owner, blocker, next_action, related_notes_json, created_from, created_at, updated_at
FROM open_loops WHERE id = ?`, id)
	var loop domain.OpenLoop
	var st, relatedRaw, selected string
	if err := row.Scan(&loop.ID, &loop.Title, &st, &loop.Scope.WorkspaceID, &loop.Scope.LaneID, &selected, &loop.Priority, &loop.Owner, &loop.Blocker, &loop.NextAction, &relatedRaw, &loop.CreatedFrom, &loop.CreatedAt, &loop.UpdatedAt); err != nil {
		return domain.OpenLoop{}, false
	}
	loop.State = domain.OpenLoopState(st)
	loop.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(relatedRaw), &loop.RelatedNotes)
	return loop, true
}

func (s *SQLiteSemanticStore) FindModel(id string) (domain.AdaptivePolicyModel, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id, type, expression_json, derived_from_json, support_count, confidence, status, workspace_id, lane_id, selected_paths_json, last_validated_at, created_at
FROM derived_models WHERE id = ?`, id)
	var m domain.AdaptivePolicyModel
	var exprRaw, derivedRaw, status, selected string
	var lv sql.NullInt64
	if err := row.Scan(&m.ID, &m.Type, &exprRaw, &derivedRaw, &m.SupportCount, &m.Confidence, &status, &m.Scope.WorkspaceID, &m.Scope.LaneID, &selected, &lv, &m.CreatedAt); err != nil {
		return domain.AdaptivePolicyModel{}, false
	}
	m.Status = domain.AdaptivePolicyModelStatus(status)
	m.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(exprRaw), &m.Expression)
	_ = json.Unmarshal([]byte(derivedRaw), &m.DerivedFrom)
	if lv.Valid {
		v := lv.Int64
		m.LastValidatedAt = &v
	}
	return m, true
}

func (s *SQLiteSemanticStore) ExistsObject(id string) bool {
	if id == "" {
		return false
	}
	tables := []string{
		"memory_notes",
		"state_items",
		"open_loops",
		"derived_models",
		"semantic_links",
		"supersession_records",
		"contradiction_records",
		"artifact_refs",
		"journal_events",
	}
	for _, tbl := range tables {
		var found int
		query := fmt.Sprintf("SELECT 1 FROM %s WHERE id = ? LIMIT 1", tbl)
		err := s.exec.QueryRowContext(s.background, query, id).Scan(&found)
		if err == nil {
			return true
		}
	}
	return false
}

func (s *SQLiteSemanticStore) FindStateByKey(key string) (domain.StateItem, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id, key, value_json, workspace_id, lane_id, selected_paths_json, status, derived_from_json, updated_at
FROM state_items
WHERE key = ?
ORDER BY updated_at DESC
LIMIT 1`, key)
	var item domain.StateItem
	var valueRaw, status, derivedRaw, selected string
	if err := row.Scan(&item.ID, &item.Key, &valueRaw, &item.Scope.WorkspaceID, &item.Scope.LaneID, &selected, &status, &derivedRaw, &item.UpdatedAt); err != nil {
		return domain.StateItem{}, false
	}
	item.Status = domain.StateItemStatus(status)
	item.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(valueRaw), &item.Value)
	_ = json.Unmarshal([]byte(derivedRaw), &item.DerivedFrom)
	return item, true
}

func (s *SQLiteSemanticStore) FindStateByScopeKey(scope domain.ForgeScope, key string) (domain.StateItem, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id, key, value_json, workspace_id, lane_id, selected_paths_json, status, derived_from_json, updated_at
FROM state_items
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND key = ?
ORDER BY updated_at DESC
LIMIT 1`, scope.WorkspaceID, scope.LaneID, scope.LaneID, key)
	var item domain.StateItem
	var valueRaw, status, derivedRaw, selected string
	if err := row.Scan(&item.ID, &item.Key, &valueRaw, &item.Scope.WorkspaceID, &item.Scope.LaneID, &selected, &status, &derivedRaw, &item.UpdatedAt); err != nil {
		return domain.StateItem{}, false
	}
	item.Status = domain.StateItemStatus(status)
	item.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(valueRaw), &item.Value)
	_ = json.Unmarshal([]byte(derivedRaw), &item.DerivedFrom)
	return item, true
}

func (s *SQLiteSemanticStore) GetIdempotency(key string) (IdempotencyRecord, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT action, request_fingerprint, idempotency_fingerprint, result_json,
       request_json, plan_json, seal_json, receipt_json, authproof_json, created_at, correlation_id
FROM semantic_idempotency_keys WHERE idempotency_key = ?`, strings.TrimSpace(key))
	var action, requestFingerprint, idempotencyFingerprint, raw, requestRaw, planRaw, sealRaw, receiptRaw, authproofRaw, correlationID string
	var createdAt int64
	if err := row.Scan(&action, &requestFingerprint, &idempotencyFingerprint, &raw, &requestRaw, &planRaw, &sealRaw, &receiptRaw, &authproofRaw, &createdAt, &correlationID); err != nil {
		return IdempotencyRecord{}, false
	}
	var result domain.SyscallResult
	_ = json.Unmarshal([]byte(raw), &result)
	var request domain.SyscallRequest
	_ = json.Unmarshal([]byte(requestRaw), &request)
	var plan commitproof.PreparedPlan
	_ = json.Unmarshal([]byte(planRaw), &plan)
	var seal commitproof.PreparedPlanSeal
	_ = json.Unmarshal([]byte(sealRaw), &seal)
	var receipt commitproof.CommitReceipt
	_ = json.Unmarshal([]byte(receiptRaw), &receipt)
	var authorizationProof authproof.Proof
	_ = json.Unmarshal([]byte(authproofRaw), &authorizationProof)
	return IdempotencyRecord{
		Action: domain.SemanticActionType(action), RequestFingerprint: requestFingerprint, IdempotencyFingerprint: idempotencyFingerprint,
		Result: result, Request: request, Plan: plan, Seal: seal, Receipt: receipt,
		AuthorizationProof: authorizationProof, CreatedAt: createdAt, CorrelationID: correlationID,
	}, true
}

func (s *SQLiteSemanticStore) FindLatestContextSnapshot(scope domain.ForgeScope, query, snapshotKind string) (domain.ContextPacket, bool) {
	packets := s.ListContextSnapshots(scope, query, snapshotKind, 1)
	if len(packets) == 0 {
		return domain.ContextPacket{}, false
	}
	return packets[0], true
}

func (s *SQLiteSemanticStore) ListContextSnapshots(scope domain.ForgeScope, query, snapshotKind string, limit int) []domain.ContextPacket {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.exec.QueryContext(s.background, `
SELECT id, query, workspace_id, lane_id, selected_paths_json, budget_json, inclusion_reasons_json, created_at,
       metadata_json, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, header_json, graph_json, delta_json,
       restore_scores_json, render_artifact_ref_id, resume_hints_json
FROM context_packet_snapshots
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND (? = '' OR query = ?)
  AND (? = '' OR snapshot_kind = ? OR (snapshot_kind = '' AND json_extract(metadata_json, '$.snapshot_kind') = ?))
ORDER BY created_at DESC, id DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, query, query, snapshotKind, snapshotKind, snapshotKind, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	packets, err := scanContextPacketRows(rows)
	if err != nil {
		return nil
	}
	return packets
}

// buildLegacyContextForInspection reconstructs historical v1 context for
// package-level diagnostics only. It is not a production compile authority.
func (s *SQLiteSemanticStore) buildLegacyContextForInspection(query string, scope domain.ForgeScope, budget domain.ContextBudget, now int64) domain.ContextPacket {
	notes, _ := s.ListActive(context.Background(), toScopeFilter(scope))
	if len(notes) > budget.MaxNotes {
		notes = notes[:budget.MaxNotes]
	}
	loops, _ := s.ListActiveLoops(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	states, _ := s.ListCurrent(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	links, _ := s.ListLinksByScope(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	models, _ := s.ListModelsByScope(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	artifacts, _ := s.listContextEvidenceArtifacts(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	events, _ := s.listContextEvidenceEvents(context.Background(), toScopeFilter(scope), budget.MaxEvents)
	return domain.ContextPacket{
		ID:          "ctx-" + strings.ReplaceAll(query, " ", "_") + "-" + fmt.Sprintf("%d", now),
		Query:       query,
		Scope:       scope,
		ActiveState: states,
		OpenLoops:   loops,
		Notes:       notes,
		LinkedNotes: links,
		Models:      models,
		Artifacts:   artifacts,
		RawEvents:   events,
		Budget:      budget,
		InclusionReasons: map[string]string{
			"mode": "sqlite_phase3",
		},
		CreatedAt: now,
	}
}

func (s *SQLiteSemanticStore) listContextEvidenceArtifacts(ctx context.Context, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_json, created_at, metadata_json
FROM artifact_refs
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
  AND type != 'context_snapshot_card'
  AND COALESCE(json_extract(metadata_json, '$.kind'), '') != 'context_snapshot_card'
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtifactRows(rows)
}

func (s *SQLiteSemanticStore) listContextEvidenceEvents(ctx context.Context, scope ScopeFilter, limit int) ([]domain.JournalEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json, correlation_id, trace_id, provenance_json, created_at
FROM journal_events
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
  AND lower(type) != 'semantic_syscall.compile_context'
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJournalRows(rows)
}
