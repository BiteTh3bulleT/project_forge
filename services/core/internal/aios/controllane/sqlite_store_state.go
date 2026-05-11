package controllane

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

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
