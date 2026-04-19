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

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type SQLiteSemanticStore struct {
	exec       sqlExecutor
	meta       CommitMetadata
	background context.Context
}

func NewSQLiteSemanticStore(db *sql.DB) *SQLiteSemanticStore {
	return &SQLiteSemanticStore{exec: db, background: context.Background()}
}

func newSQLiteSemanticStore(exec sqlExecutor) *SQLiteSemanticStore {
	return &SQLiteSemanticStore{exec: exec, background: context.Background()}
}

func (s *SQLiteSemanticStore) SetCommitMetadata(meta CommitMetadata) {
	s.meta = meta
}

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
	if err != nil && err != sql.ErrNoRows {
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

func (s *SQLiteSemanticStore) SetIdempotency(key string, rec IdempotencyRecord) {
	_, _ = s.exec.ExecContext(s.background, `
INSERT INTO semantic_idempotency_keys(idempotency_key, action, result_json, created_at, correlation_id)
VALUES(?,?,?,?,?)
ON CONFLICT(idempotency_key) DO UPDATE SET
 action = excluded.action,
 result_json = excluded.result_json,
 correlation_id = excluded.correlation_id`,
		key, string(rec.Action), encodeJSON(rec.Result), domain.NowMillis(), rec.Result.CorrelationID,
	)
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
	row := s.exec.QueryRowContext(s.background, `SELECT action, result_json FROM semantic_idempotency_keys WHERE idempotency_key = ?`, key)
	var action, raw string
	if err := row.Scan(&action, &raw); err != nil {
		return IdempotencyRecord{}, false
	}
	var result domain.SyscallResult
	_ = json.Unmarshal([]byte(raw), &result)
	return IdempotencyRecord{Action: domain.SemanticActionType(action), Result: result}, true
}

func (s *SQLiteSemanticStore) BuildContext(query string, scope domain.ForgeScope, budget domain.ContextBudget, now int64) domain.ContextPacket {
	notes, _ := s.ListActive(context.Background(), toScopeFilter(scope))
	if len(notes) > budget.MaxNotes {
		notes = notes[:budget.MaxNotes]
	}
	loops, _ := s.ListActiveLoops(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	states, _ := s.ListCurrent(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	links, _ := s.ListLinksByScope(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	models, _ := s.ListModelsByScope(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	artifacts, _ := s.ListByScopeArtifacts(context.Background(), toScopeFilter(scope), budget.MaxNotes)
	events, _ := s.ListByScope(context.Background(), toScopeFilter(scope), budget.MaxEvents)
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

// JournalRepository
func (s *SQLiteSemanticStore) Append(ctx context.Context, evt domain.JournalEvent) error {
	scope := toScopeFilter(evt.Scope)
	provID, err := s.ensureProvenance(ctx, evt.Scope, evt.Provenance, nil, evt.Timestamp)
	if err != nil {
		return err
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO journal_events(
  id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json,
  correlation_id, trace_id, provenance_id, provenance_json, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		evt.ID, evt.Type, evt.Source, evt.Provenance.Actor, scope.WorkspaceID, scope.LaneID, encodeStringSlice(evt.Scope.SelectedPaths), encodeJSON(evt.Payload),
		evt.CorrelationID, nonEmpty(evt.Provenance.TraceID, s.meta.TraceID), provID, encodeJSON(evt.Provenance), evt.Timestamp, encodeJSON(map[string]any{}),
		string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, "",
	)
	return err
}

func (s *SQLiteSemanticStore) GetByID(ctx context.Context, id string) (domain.JournalEvent, bool, error) {
	row := s.exec.QueryRowContext(ctx, `
SELECT id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json, correlation_id, trace_id, provenance_json, created_at
FROM journal_events WHERE id = ?`, id)
	var evt domain.JournalEvent
	var payloadRaw, provRaw, selected, actor, trace string
	if err := row.Scan(&evt.ID, &evt.Type, &evt.Source, &actor, &evt.Scope.WorkspaceID, &evt.Scope.LaneID, &selected, &payloadRaw, &evt.CorrelationID, &trace, &provRaw, &evt.Timestamp); err != nil {
		if err == sql.ErrNoRows {
			return domain.JournalEvent{}, false, nil
		}
		return domain.JournalEvent{}, false, err
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
	return evt, true, nil
}

func (s *SQLiteSemanticStore) ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.JournalEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json, correlation_id, trace_id, provenance_json, created_at
FROM journal_events
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJournalRows(rows)
}

func (s *SQLiteSemanticStore) ListByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.JournalEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json, correlation_id, trace_id, provenance_json, created_at
FROM journal_events
WHERE correlation_id = ?
ORDER BY created_at DESC
LIMIT ?`, correlationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJournalRows(rows)
}

func (s *SQLiteSemanticStore) ListRecent(ctx context.Context, filter RecentFilter) ([]domain.JournalEvent, error) {
	if filter.Scope.WorkspaceID == "" {
		return nil, nil
	}
	return s.ListByScope(ctx, filter.Scope, filter.Limit)
}

// MemoryNoteRepository
func (s *SQLiteSemanticStore) Create(ctx context.Context, note domain.MemoryNote) error {
	provID, err := s.ensureProvenance(ctx, note.Scope, note.Provenance, nil, note.CreatedAt)
	if err != nil {
		return err
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO memory_notes(
  id, type, title, content, workspace_id, lane_id, selected_paths_json,
  confidence, status, provenance_id, provenance_json, created_at, updated_at,
  archived_at, superseded_by, metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		note.ID, string(note.Type), note.Title, note.Content, note.Scope.WorkspaceID, note.Scope.LaneID, encodeStringSlice(note.Scope.SelectedPaths),
		note.Confidence, string(note.Status), provID, encodeJSON(note.Provenance), note.CreatedAt, note.UpdatedAt,
		nil, nil, encodeJSON(map[string]any{}), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, nonEmpty(s.meta.TraceID, note.Provenance.TraceID), "",
	)
	return err
}

func (s *SQLiteSemanticStore) GetByIDNote(ctx context.Context, id string) (domain.MemoryNote, bool, error) {
	n, ok := s.FindNote(id)
	return n, ok, nil
}

func (s *SQLiteSemanticStore) UpdateStatus(ctx context.Context, id string, status domain.MemoryNoteStatus, metadata map[string]any, updatedAt int64) error {
	archived := sql.NullInt64{}
	if status == domain.NoteArchived {
		archived = sql.NullInt64{Int64: updatedAt, Valid: true}
	}
	res, err := s.exec.ExecContext(ctx, `
UPDATE memory_notes
SET status = ?, updated_at = ?, archived_at = COALESCE(?, archived_at), metadata_json = ?, syscall_id = ?, correlation_id = ?, trace_id = ?
WHERE id = ?`,
		string(status), updatedAt, archived, encodeJSON(nonNilMap(metadata)), s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID, id,
	)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("note %q not found", id)
	}
	return nil
}

func (s *SQLiteSemanticStore) ListByType(ctx context.Context, typ domain.MemoryNoteType, scope ScopeFilter) ([]domain.MemoryNote, error) {
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, title, content, workspace_id, lane_id, selected_paths_json, confidence, status, created_at, updated_at, provenance_json
FROM memory_notes
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND type = ?
ORDER BY updated_at DESC`, scope.WorkspaceID, scope.LaneID, scope.LaneID, string(typ))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNoteRows(rows)
}

func (s *SQLiteSemanticStore) ListByScopeNotes(ctx context.Context, scope ScopeFilter) ([]domain.MemoryNote, error) {
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, title, content, workspace_id, lane_id, selected_paths_json, confidence, status, created_at, updated_at, provenance_json
FROM memory_notes
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY updated_at DESC`, scope.WorkspaceID, scope.LaneID, scope.LaneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNoteRows(rows)
}

func (s *SQLiteSemanticStore) ListActive(ctx context.Context, scope ScopeFilter) ([]domain.MemoryNote, error) {
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, title, content, workspace_id, lane_id, selected_paths_json, confidence, status, created_at, updated_at, provenance_json
FROM memory_notes
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND status = 'active'
ORDER BY updated_at DESC`, scope.WorkspaceID, scope.LaneID, scope.LaneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNoteRows(rows)
}

func (s *SQLiteSemanticStore) ListSuperseded(ctx context.Context, scope ScopeFilter) ([]domain.MemoryNote, error) {
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, title, content, workspace_id, lane_id, selected_paths_json, confidence, status, created_at, updated_at, provenance_json
FROM memory_notes
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND status = 'superseded'
ORDER BY updated_at DESC`, scope.WorkspaceID, scope.LaneID, scope.LaneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNoteRows(rows)
}

func (s *SQLiteSemanticStore) Archive(ctx context.Context, id, _ string, _ domain.Provenance, updatedAt int64) error {
	return s.UpdateStatus(ctx, id, domain.NoteArchived, map[string]any{}, updatedAt)
}

func (s *SQLiteSemanticStore) FindByProvenance(ctx context.Context, actor string, scope ScopeFilter, limit int) ([]domain.MemoryNote, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, title, content, workspace_id, lane_id, selected_paths_json, confidence, status, created_at, updated_at, provenance_json
FROM memory_notes
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND json_extract(provenance_json, '$.actor') = ?
ORDER BY updated_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, actor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNoteRows(rows)
}

// SemanticLinkRepository
func (s *SQLiteSemanticStore) CreateLinkWithKinds(ctx context.Context, link domain.SemanticLink, sourceKind, targetKind string) error {
	prevMeta := s.meta
	defer func() { s.meta = prevMeta }()
	return s.CreateLink(link)
}

func (s *SQLiteSemanticStore) GetByIDLink(ctx context.Context, id string) (domain.SemanticLink, bool, error) {
	row := s.exec.QueryRowContext(ctx, `
SELECT id, type, source_id, target_id, workspace_id, lane_id, selected_paths_json, confidence, provenance_json, created_at
FROM semantic_links WHERE id = ?`, id)
	var link domain.SemanticLink
	var typ, provRaw, selected string
	if err := row.Scan(&link.ID, &typ, &link.SourceID, &link.TargetID, &link.Scope.WorkspaceID, &link.Scope.LaneID, &selected, &link.Confidence, &provRaw, &link.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return domain.SemanticLink{}, false, nil
		}
		return domain.SemanticLink{}, false, err
	}
	link.Type = domain.SemanticLinkType(typ)
	link.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(provRaw), &link.Provenance)
	return link, true, nil
}

func (s *SQLiteSemanticStore) ListBySource(ctx context.Context, sourceID string, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return s.listLinks(ctx, `source_id = ?`, []any{sourceID}, scope, limit)
}

func (s *SQLiteSemanticStore) ListByTarget(ctx context.Context, targetID string, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return s.listLinks(ctx, `target_id = ?`, []any{targetID}, scope, limit)
}

func (s *SQLiteSemanticStore) ListNeighborhood(ctx context.Context, objectID string, scope ScopeFilter, depth, limit int) ([]domain.SemanticLink, error) {
	if depth <= 1 {
		return s.listLinks(ctx, `(source_id = ? OR target_id = ?)`, []any{objectID, objectID}, scope, limit)
	}
	oneHop, err := s.listLinks(ctx, `(source_id = ? OR target_id = ?)`, []any{objectID, objectID}, scope, limit)
	if err != nil {
		return nil, err
	}
	seen := map[string]domain.SemanticLink{}
	nextObjects := map[string]struct{}{}
	for _, l := range oneHop {
		seen[l.ID] = l
		nextObjects[l.SourceID] = struct{}{}
		nextObjects[l.TargetID] = struct{}{}
	}
	for oid := range nextObjects {
		links, err := s.listLinks(ctx, `(source_id = ? OR target_id = ?)`, []any{oid, oid}, scope, limit)
		if err != nil {
			return nil, err
		}
		for _, l := range links {
			seen[l.ID] = l
		}
	}
	out := make([]domain.SemanticLink, 0, len(seen))
	for _, l := range seen {
		out = append(out, l)
	}
	if len(out) > limit && limit > 0 {
		out = out[:limit]
	}
	return out, nil
}

func (s *SQLiteSemanticStore) ListByTypeLinks(ctx context.Context, typ domain.SemanticLinkType, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return s.listLinks(ctx, `type = ?`, []any{string(typ)}, scope, limit)
}

// StateRepository
func (s *SQLiteSemanticStore) GetCurrent(ctx context.Context, key string, scope ScopeFilter) (domain.StateItem, bool, error) {
	row := s.exec.QueryRowContext(ctx, `
SELECT id, key, value_json, workspace_id, lane_id, selected_paths_json, status, derived_from_json, updated_at
FROM state_items
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND key = ?`,
		scope.WorkspaceID, scope.LaneID, scope.LaneID, key)
	var item domain.StateItem
	var valueRaw, status, derivedRaw, selected string
	if err := row.Scan(&item.ID, &item.Key, &valueRaw, &item.Scope.WorkspaceID, &item.Scope.LaneID, &selected, &status, &derivedRaw, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return domain.StateItem{}, false, nil
		}
		return domain.StateItem{}, false, err
	}
	item.Scope.SelectedPaths = decodeStringSlice(selected)
	item.Status = domain.StateItemStatus(status)
	_ = json.Unmarshal([]byte(valueRaw), &item.Value)
	_ = json.Unmarshal([]byte(derivedRaw), &item.DerivedFrom)
	return item, true, nil
}

func (s *SQLiteSemanticStore) UpsertCurrent(ctx context.Context, state domain.StateItem, changedBy string, syscallID, correlationID, traceID, auditID string, metadata map[string]any) error {
	prevMeta := s.meta
	s.meta = CommitMetadata{
		SyscallID:     syscallID,
		CorrelationID: correlationID,
		TraceID:       traceID,
		ActorID:       changedBy,
		Source:        prevMeta.Source,
		CommittedBy:   nonEmpty(prevMeta.CommittedBy, "forge_kernel"),
	}
	defer func() { s.meta = prevMeta }()
	if err := s.CreateState(state); err != nil {
		return err
	}
	if auditID != "" {
		return linkAuditOnExecutor(ctx, s.exec, correlationID, syscallID, auditID)
	}
	return nil
}

func (s *SQLiteSemanticStore) AppendHistory(ctx context.Context, version StateVersionRecord) error {
	laneID := version.LaneID
	if laneID == "" {
		laneID = "control.semantic"
	}
	_, err := s.exec.ExecContext(ctx, `
INSERT INTO state_versions(
  state_item_id, state_key, workspace_id, lane_id, previous_value_json, new_value_json, changed_by,
  derived_from_json, syscall_id, audit_id, correlation_id, trace_id, created_at, metadata_json, proposed_by, committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		version.StateItemID,
		version.StateKey,
		version.WorkspaceID,
		laneID,
		encodeJSON(version.PreviousValue),
		encodeJSON(version.NewValue),
		version.ChangedBy,
		encodeJSON(version.DerivedFrom),
		version.SyscallID,
		version.AuditID,
		version.CorrelationID,
		version.TraceID,
		version.CreatedAt,
		encodeJSON(nonNilMap(version.Metadata)),
		nonEmpty(version.ProposedBy, "system"),
		nonEmpty(version.CommittedBy, "forge_kernel"),
	)
	return err
}

func (s *SQLiteSemanticStore) GetTimeline(ctx context.Context, key string, scope ScopeFilter, limit int) ([]StateVersionRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, state_item_id, state_key, workspace_id, lane_id, previous_value_json, new_value_json, changed_by, derived_from_json,
       syscall_id, audit_id, correlation_id, trace_id, proposed_by, committed_by, created_at, metadata_json
FROM state_versions
WHERE state_key = ? AND workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY id ASC
LIMIT ?`, key, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StateVersionRecord{}
	for rows.Next() {
		var r StateVersionRecord
		var prevRaw, newRaw, derivedRaw, metaRaw string
		if err := rows.Scan(&r.ID, &r.StateItemID, &r.StateKey, &r.WorkspaceID, &r.LaneID, &prevRaw, &newRaw, &r.ChangedBy, &derivedRaw, &r.SyscallID, &r.AuditID, &r.CorrelationID, &r.TraceID, &r.ProposedBy, &r.CommittedBy, &r.CreatedAt, &metaRaw); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(prevRaw), &r.PreviousValue)
		_ = json.Unmarshal([]byte(newRaw), &r.NewValue)
		_ = json.Unmarshal([]byte(derivedRaw), &r.DerivedFrom)
		_ = json.Unmarshal([]byte(metaRaw), &r.Metadata)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLiteSemanticStore) ListCurrent(ctx context.Context, scope ScopeFilter, limit int) ([]domain.StateItem, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, key, value_json, workspace_id, lane_id, selected_paths_json, status, derived_from_json, updated_at
FROM state_items
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY updated_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStateRows(rows)
}

func (s *SQLiteSemanticStore) ListRecentlyChanged(ctx context.Context, scope ScopeFilter, limit int) ([]domain.StateItem, error) {
	return s.ListCurrent(ctx, scope, limit)
}

func (s *SQLiteSemanticStore) ListHistoryKeys(ctx context.Context, scope ScopeFilter, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT DISTINCT state_key
FROM state_versions
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY state_key ASC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		key = strings.TrimSpace(key)
		if key != "" {
			out = append(out, key)
		}
	}
	return out, rows.Err()
}

// OpenLoopRepository
func (s *SQLiteSemanticStore) GetByIDLoop(ctx context.Context, id string) (domain.OpenLoop, bool, error) {
	v, ok := s.FindLoop(id)
	return v, ok, nil
}

func (s *SQLiteSemanticStore) UpdateState(ctx context.Context, id string, state domain.OpenLoopState, metadata map[string]any, updatedAt int64) error {
	loop, ok := s.FindLoop(id)
	if !ok {
		return fmt.Errorf("loop %q not found", id)
	}
	loop.State = state
	loop.UpdatedAt = updatedAt
	return s.UpdateLoop(loop)
}

func (s *SQLiteSemanticStore) ListByState(ctx context.Context, state domain.OpenLoopState, scope ScopeFilter, limit int) ([]domain.OpenLoop, error) {
	return s.listLoops(ctx, "state = ?", []any{string(state)}, scope, limit)
}

func (s *SQLiteSemanticStore) ListByPriority(ctx context.Context, priority string, scope ScopeFilter, limit int) ([]domain.OpenLoop, error) {
	return s.listLoops(ctx, "priority = ?", []any{priority}, scope, limit)
}

func (s *SQLiteSemanticStore) ListActiveLoops(ctx context.Context, scope ScopeFilter, limit int) ([]domain.OpenLoop, error) {
	return s.listLoops(ctx, "state IN ('open','in_progress','blocked')", nil, scope, limit)
}

func (s *SQLiteSemanticStore) ListStale(ctx context.Context, cutoffMillis int64, scope ScopeFilter, limit int) ([]domain.OpenLoop, error) {
	return s.listLoops(ctx, "state IN ('open','in_progress','blocked') AND updated_at < ?", []any{cutoffMillis}, scope, limit)
}

// ArtifactRefRepository
func (s *SQLiteSemanticStore) CreateArtifact(ctx context.Context, ref domain.ArtifactRef) error {
	provID, err := s.ensureProvenance(ctx, ref.Scope, ref.Provenance, nil, ref.CreatedAt)
	if err != nil {
		return err
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO artifact_refs(
  id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_id, provenance_json,
  created_at, metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ref.ID, ref.Type, ref.URI, ref.ContentHash, ref.Scope.WorkspaceID, ref.Scope.LaneID, encodeStringSlice(ref.Scope.SelectedPaths), provID, encodeJSON(ref.Provenance),
		ref.CreatedAt, encodeJSON(nonNilMap(ref.Metadata)), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID, "",
	)
	return err
}

func (s *SQLiteSemanticStore) CreateArtifactRef(ref domain.ArtifactRef) error {
	return s.CreateArtifact(s.background, ref)
}

func (s *SQLiteSemanticStore) GetArtifactByID(ctx context.Context, id string) (domain.ArtifactRef, bool, error) {
	row := s.exec.QueryRowContext(ctx, `
SELECT id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_json, created_at, metadata_json
FROM artifact_refs WHERE id = ?`, id)
	var ref domain.ArtifactRef
	var selected, provRaw, metaRaw string
	if err := row.Scan(&ref.ID, &ref.Type, &ref.URI, &ref.ContentHash, &ref.Scope.WorkspaceID, &ref.Scope.LaneID, &selected, &provRaw, &ref.CreatedAt, &metaRaw); err != nil {
		if err == sql.ErrNoRows {
			return domain.ArtifactRef{}, false, nil
		}
		return domain.ArtifactRef{}, false, err
	}
	ref.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(provRaw), &ref.Provenance)
	_ = json.Unmarshal([]byte(metaRaw), &ref.Metadata)
	return ref, true, nil
}

func (s *SQLiteSemanticStore) FindByChecksum(ctx context.Context, checksum string, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_json, created_at, metadata_json
FROM artifact_refs
WHERE content_hash = ? AND workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY created_at DESC
LIMIT ?`, checksum, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtifactRows(rows)
}

func (s *SQLiteSemanticStore) ListByScopeArtifacts(ctx context.Context, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_json, created_at, metadata_json
FROM artifact_refs
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtifactRows(rows)
}

func (s *SQLiteSemanticStore) ListByProvenanceArtifacts(ctx context.Context, actor string, scope ScopeFilter, limit int) ([]domain.ArtifactRef, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_json, created_at, metadata_json
FROM artifact_refs
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND json_extract(provenance_json, '$.actor') = ?
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, actor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtifactRows(rows)
}

// DerivedModelRepository
func (s *SQLiteSemanticStore) GetByIDModel(ctx context.Context, id string) (domain.AdaptivePolicyModel, bool, error) {
	v, ok := s.FindModel(id)
	return v, ok, nil
}

func (s *SQLiteSemanticStore) UpdateStatusModel(ctx context.Context, id string, status domain.AdaptivePolicyModelStatus, updatedAt int64) error {
	res, err := s.exec.ExecContext(ctx, `
UPDATE derived_models
SET status = ?, updated_at = ?, syscall_id = ?, correlation_id = ?, trace_id = ?
WHERE id = ?`, string(status), updatedAt, s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID, id)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("model %q not found", id)
	}
	return nil
}

func (s *SQLiteSemanticStore) ListByStatusModel(ctx context.Context, status domain.AdaptivePolicyModelStatus, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error) {
	return s.listModels(ctx, "status = ?", []any{string(status)}, scope, limit)
}

func (s *SQLiteSemanticStore) ListByTypeModel(ctx context.Context, typ string, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error) {
	return s.listModels(ctx, "type = ?", []any{typ}, scope, limit)
}

func (s *SQLiteSemanticStore) ListDerivedFrom(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, expression_json, derived_from_json, support_count, confidence, status, workspace_id, lane_id, selected_paths_json, last_validated_at, created_at
FROM derived_models
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND EXISTS (
  SELECT 1 FROM json_each(derived_from_json) WHERE json_each.value = ?
)
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, objectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanModelRows(rows)
}

// ContradictionRepository
func (s *SQLiteSemanticStore) CreateContradictionWithKinds(ctx context.Context, record ContradictionRecord, leftKind, rightKind string, scope ScopeFilter) error {
	provID, err := s.ensureProvenance(ctx, domain.ForgeScope{WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID}, record.Provenance, nil, record.CreatedAt)
	if err != nil {
		return err
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO contradiction_records(
  id, left_object_id, left_object_kind, right_object_id, right_object_kind, reason, severity, confidence,
  provenance_id, provenance_json, workspace_id, lane_id, created_at, metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		record.ID, record.LeftID, leftKind, record.RightID, rightKind, record.Reason, record.Severity, record.Confidence,
		provID, encodeJSON(record.Provenance), scope.WorkspaceID, scope.LaneID, record.CreatedAt, encodeJSON(nonNilMap(record.Metadata)),
		nonEmpty(record.ProposedBy, string(s.meta.Source)), nonEmpty(record.CommittedBy, nonEmpty(s.meta.CommittedBy, "forge_kernel")), nonEmpty(record.SyscallID, s.meta.SyscallID), nonEmpty(record.CorrelationID, s.meta.CorrelationID), nonEmpty(record.TraceID, s.meta.TraceID), "",
	)
	return err
}

func (s *SQLiteSemanticStore) GetByIDContradiction(ctx context.Context, id string) (ContradictionRecord, bool, error) {
	row := s.exec.QueryRowContext(ctx, `
SELECT id, left_object_id, left_object_kind, right_object_id, right_object_kind, reason, severity, confidence,
       workspace_id, correlation_id, trace_id, syscall_id, audit_id, proposed_by, committed_by, metadata_json, created_at, provenance_json
FROM contradiction_records WHERE id = ?`, id)
	var r ContradictionRecord
	var metaRaw, provRaw string
	if err := row.Scan(&r.ID, &r.LeftID, &r.LeftKind, &r.RightID, &r.RightKind, &r.Reason, &r.Severity, &r.Confidence, &r.WorkspaceID, &r.CorrelationID, &r.TraceID, &r.SyscallID, &r.AuditID, &r.ProposedBy, &r.CommittedBy, &metaRaw, &r.CreatedAt, &provRaw); err != nil {
		if err == sql.ErrNoRows {
			return ContradictionRecord{}, false, nil
		}
		return ContradictionRecord{}, false, err
	}
	_ = json.Unmarshal([]byte(metaRaw), &r.Metadata)
	_ = json.Unmarshal([]byte(provRaw), &r.Provenance)
	return r, true, nil
}

func (s *SQLiteSemanticStore) ListByObject(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]ContradictionRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, left_object_id, left_object_kind, right_object_id, right_object_kind, reason, severity, confidence,
       workspace_id, correlation_id, trace_id, syscall_id, audit_id, proposed_by, committed_by, metadata_json, created_at, provenance_json
FROM contradiction_records
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND (left_object_id = ? OR right_object_id = ?)
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, objectID, objectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContradictionRows(rows)
}

func (s *SQLiteSemanticStore) ListByScopeContradictions(ctx context.Context, scope ScopeFilter, limit int) ([]ContradictionRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, left_object_id, left_object_kind, right_object_id, right_object_kind, reason, severity, confidence,
       workspace_id, correlation_id, trace_id, syscall_id, audit_id, proposed_by, committed_by, metadata_json, created_at, provenance_json
FROM contradiction_records
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContradictionRows(rows)
}

// SupersessionRepository
func (s *SQLiteSemanticStore) CreateSupersessionWithKinds(ctx context.Context, record SupersessionRecord, oldKind, newKind string, scope ScopeFilter) error {
	provID, err := s.ensureProvenance(ctx, domain.ForgeScope{WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID}, record.Provenance, nil, record.CreatedAt)
	if err != nil {
		return err
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO supersession_records(
  id, old_object_id, old_object_kind, new_object_id, new_object_kind, reason,
  provenance_id, provenance_json, workspace_id, lane_id, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		record.ID, record.OldID, oldKind, record.NewID, newKind, record.Reason,
		provID, encodeJSON(record.Provenance), scope.WorkspaceID, scope.LaneID, record.CreatedAt, encodeJSON(nonNilMap(record.Metadata)),
		nonEmpty(record.ProposedBy, string(s.meta.Source)), nonEmpty(record.CommittedBy, nonEmpty(s.meta.CommittedBy, "forge_kernel")), nonEmpty(record.SyscallID, s.meta.SyscallID), nonEmpty(record.CorrelationID, s.meta.CorrelationID), nonEmpty(record.TraceID, s.meta.TraceID), "",
	)
	return err
}

func (s *SQLiteSemanticStore) GetByIDSupersession(ctx context.Context, id string) (SupersessionRecord, bool, error) {
	row := s.exec.QueryRowContext(ctx, `
SELECT id, old_object_id, old_object_kind, new_object_id, new_object_kind, reason, workspace_id, correlation_id, trace_id,
       syscall_id, audit_id, proposed_by, committed_by, metadata_json, created_at, provenance_json
FROM supersession_records WHERE id = ?`, id)
	var r SupersessionRecord
	var metaRaw, provRaw string
	if err := row.Scan(&r.ID, &r.OldID, &r.OldKind, &r.NewID, &r.NewKind, &r.Reason, &r.WorkspaceID, &r.CorrelationID, &r.TraceID, &r.SyscallID, &r.AuditID, &r.ProposedBy, &r.CommittedBy, &metaRaw, &r.CreatedAt, &provRaw); err != nil {
		if err == sql.ErrNoRows {
			return SupersessionRecord{}, false, nil
		}
		return SupersessionRecord{}, false, err
	}
	_ = json.Unmarshal([]byte(metaRaw), &r.Metadata)
	_ = json.Unmarshal([]byte(provRaw), &r.Provenance)
	return r, true, nil
}

func (s *SQLiteSemanticStore) ListByOldObject(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]SupersessionRecord, error) {
	return s.listSupersessions(ctx, "old_object_id = ?", []any{objectID}, scope, limit)
}

func (s *SQLiteSemanticStore) ListByNewObject(ctx context.Context, objectID string, scope ScopeFilter, limit int) ([]SupersessionRecord, error) {
	return s.listSupersessions(ctx, "new_object_id = ?", []any{objectID}, scope, limit)
}

func (s *SQLiteSemanticStore) ListByScopeSupersessions(ctx context.Context, scope ScopeFilter, limit int) ([]SupersessionRecord, error) {
	return s.listSupersessions(ctx, "1=1", nil, scope, limit)
}

func (s *SQLiteSemanticStore) GetCurrentSuccessor(ctx context.Context, objectID string, scope ScopeFilter) (SupersessionRecord, bool, error) {
	list, err := s.listSupersessions(ctx, "old_object_id = ?", []any{objectID}, scope, 1)
	if err != nil || len(list) == 0 {
		return SupersessionRecord{}, false, err
	}
	return list[0], true, nil
}

// ContextPacketRepository
func (s *SQLiteSemanticStore) CreateSnapshot(ctx context.Context, pkt domain.ContextPacket, syscallID, correlationID, traceID string, metadata map[string]any) error {
	_, err := s.exec.ExecContext(ctx, `
INSERT INTO context_packet_snapshots(
  id, query, workspace_id, lane_id, selected_paths_json,
  included_state_json, included_open_loops_json, included_notes_json, included_links_json, included_models_json,
  included_artifacts_json, included_events_json, budget_json, inclusion_reasons_json,
  created_at, correlation_id, trace_id, syscall_id, metadata_json, proposed_by, committed_by, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		pkt.ID, pkt.Query, pkt.Scope.WorkspaceID, pkt.Scope.LaneID, encodeStringSlice(pkt.Scope.SelectedPaths),
		encodeIDsState(pkt.ActiveState), encodeIDsLoops(pkt.OpenLoops), encodeIDsNotes(pkt.Notes), encodeIDsLinks(pkt.LinkedNotes), encodeIDsModels(pkt.Models),
		encodeIDsArtifacts(pkt.Artifacts), encodeIDsEvents(pkt.RawEvents), encodeJSON(pkt.Budget), encodeJSON(pkt.InclusionReasons),
		pkt.CreatedAt, correlationID, traceID, syscallID, encodeJSON(nonNilMap(metadata)), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), "",
	)
	return err
}

func (s *SQLiteSemanticStore) GetSnapshotByID(ctx context.Context, id string) (domain.ContextPacket, bool, error) {
	row := s.exec.QueryRowContext(ctx, `
SELECT id, query, workspace_id, lane_id, selected_paths_json, budget_json, inclusion_reasons_json, created_at
FROM context_packet_snapshots WHERE id = ?`, id)
	var pkt domain.ContextPacket
	var selected, budgetRaw, reasonsRaw string
	if err := row.Scan(&pkt.ID, &pkt.Query, &pkt.Scope.WorkspaceID, &pkt.Scope.LaneID, &selected, &budgetRaw, &reasonsRaw, &pkt.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return domain.ContextPacket{}, false, nil
		}
		return domain.ContextPacket{}, false, err
	}
	pkt.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(budgetRaw), &pkt.Budget)
	_ = json.Unmarshal([]byte(reasonsRaw), &pkt.InclusionReasons)
	return pkt, true, nil
}

func (s *SQLiteSemanticStore) ListSnapshotsByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.ContextPacket, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, selected_paths_json, budget_json, inclusion_reasons_json, created_at
FROM context_packet_snapshots
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ContextPacket{}
	for rows.Next() {
		var pkt domain.ContextPacket
		var selected, budgetRaw, reasonsRaw string
		if err := rows.Scan(&pkt.ID, &pkt.Query, &pkt.Scope.WorkspaceID, &pkt.Scope.LaneID, &selected, &budgetRaw, &reasonsRaw, &pkt.CreatedAt); err != nil {
			return nil, err
		}
		pkt.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(budgetRaw), &pkt.Budget)
		_ = json.Unmarshal([]byte(reasonsRaw), &pkt.InclusionReasons)
		out = append(out, pkt)
	}
	return out, rows.Err()
}

func (s *SQLiteSemanticStore) ListSnapshotsByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.ContextPacket, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, selected_paths_json, budget_json, inclusion_reasons_json, created_at
FROM context_packet_snapshots
WHERE correlation_id = ?
ORDER BY created_at DESC
LIMIT ?`, correlationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ContextPacket{}
	for rows.Next() {
		var pkt domain.ContextPacket
		var selected, budgetRaw, reasonsRaw string
		if err := rows.Scan(&pkt.ID, &pkt.Query, &pkt.Scope.WorkspaceID, &pkt.Scope.LaneID, &selected, &budgetRaw, &reasonsRaw, &pkt.CreatedAt); err != nil {
			return nil, err
		}
		pkt.Scope.SelectedPaths = decodeStringSlice(selected)
		_ = json.Unmarshal([]byte(budgetRaw), &pkt.Budget)
		_ = json.Unmarshal([]byte(reasonsRaw), &pkt.InclusionReasons)
		out = append(out, pkt)
	}
	return out, rows.Err()
}

// Extra read helpers used by tests and context compilation.
func (s *SQLiteSemanticStore) ListLinksByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return s.listLinks(ctx, "1=1", nil, scope, limit)
}

func (s *SQLiteSemanticStore) ListModelsByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error) {
	return s.listModels(ctx, "1=1", nil, scope, limit)
}

// SQLite Transaction runner.
type SQLiteTransactionRunner struct {
	db   *sql.DB
	read *SQLiteSemanticStore
}

func NewSQLiteTransactionRunner(db *sql.DB) *SQLiteTransactionRunner {
	return &SQLiteTransactionRunner{
		db:   db,
		read: NewSQLiteSemanticStore(db),
	}
}

func (r *SQLiteTransactionRunner) ReadStore() SemanticReadStore {
	return r.read
}

func (r *SQLiteTransactionRunner) Run(ctx context.Context, fn func(uow UnitOfWork) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txStore := newSQLiteSemanticStore(tx)
	uow := &txUnitOfWork{store: txStore}
	if err := fn(uow); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteTransactionRunner) LinkAudit(ctx context.Context, correlationID, syscallID, auditID string) error {
	return linkAuditOnExecutor(ctx, r.db, correlationID, syscallID, auditID)
}

func linkAuditOnExecutor(ctx context.Context, exec sqlExecutor, correlationID, syscallID, auditID string) error {
	if strings.TrimSpace(auditID) == "" || strings.TrimSpace(syscallID) == "" {
		return nil
	}
	tables := []string{
		"provenance_records",
		"memory_notes",
		"semantic_links",
		"state_items",
		"state_versions",
		"open_loops",
		"artifact_refs",
		"derived_models",
		"contradiction_records",
		"supersession_records",
		"journal_events",
		"context_packet_snapshots",
	}
	for _, tbl := range tables {
		query := fmt.Sprintf(`UPDATE %s SET audit_id = ? WHERE audit_id = '' AND syscall_id = ? AND correlation_id = ?`, tbl)
		if _, err := exec.ExecContext(ctx, query, auditID, syscallID, correlationID); err != nil {
			return err
		}
	}
	return nil
}

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
