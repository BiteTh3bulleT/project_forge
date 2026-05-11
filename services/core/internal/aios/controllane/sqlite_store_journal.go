package controllane

import (
	"context"
	"database/sql"
	"encoding/json"

	"forge/projectforge/services/core/internal/aios/domain"
)

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
