package controllane

import (
	"context"
	"database/sql"
	"fmt"

	"forge/projectforge/services/core/internal/aios/domain"
)

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
