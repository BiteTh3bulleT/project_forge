package controllane

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"forge/projectforge/services/core/internal/aios/domain"
)

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
