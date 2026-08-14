package controllane

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/memory/vsaprojection"
)

var (
	ErrMemoryAccelerationTransactionRequired = errors.New("memory acceleration rebuild requires caller transaction")
	ErrMemoryAccelerationHeadConflict        = errors.New("memory acceleration projection head conflict")
	ErrMemoryAccelerationManifestActive      = errors.New("memory acceleration manifest is already active")
	ErrMemoryAccelerationNoGovernedSources   = errors.New("memory acceleration scope has no governed sources")
)

// MemoryAccelerationRebuildRequest binds a projection rebuild to an exact
// workspace/lane source set, deterministic algorithm identity, and prior head.
// ExpectedManifestHash is computed during proposal/preflight and recomputed
// from canonical evidence inside the commit transaction.
type MemoryAccelerationRebuildRequest struct {
	Scope                     vsaprojection.Scope
	Algorithm                 vsaprojection.Algorithm
	ExpectedManifestHash      string
	ExpectedPriorManifestHash string
	RequestedAtMs             int64
}

type MemoryAccelerationCommit struct {
	Manifest          vsaprojection.Manifest `json:"manifest"`
	PriorManifestHash string                 `json:"priorManifestHash"`
	PointerCount      int                    `json:"pointerCount"`
	BindingCount      int                    `json:"bindingCount"`
	AssociationCount  int                    `json:"associationCount"`
}

// PlanMemoryAcceleration is a read-only deterministic proposal. It does not
// authorize a commit; the commit path recomputes the same manifest inside the
// caller transaction and verifies its expected identity.
func (s *SQLiteSemanticStore) PlanMemoryAcceleration(ctx context.Context, scope vsaprojection.Scope, algorithm vsaprojection.Algorithm) (vsaprojection.Projection, error) {
	if s == nil || s.exec == nil {
		return vsaprojection.Projection{}, ErrMemoryAccelerationTransactionRequired
	}
	sources, err := s.loadMemoryAccelerationSources(ctx, scope)
	if err != nil {
		return vsaprojection.Projection{}, err
	}
	if len(sources) == 0 {
		return vsaprojection.Projection{}, ErrMemoryAccelerationNoGovernedSources
	}
	links, err := s.loadMemoryAccelerationLinks(ctx, scope)
	if err != nil {
		return vsaprojection.Projection{}, err
	}
	return vsaprojection.Build(scope, algorithm, sources, links)
}

func (s *SQLiteSemanticStore) MemoryAccelerationHead(ctx context.Context, scope vsaprojection.Scope) (string, bool, error) {
	return s.memoryAccelerationHead(ctx, scope)
}

// RebuildMemoryAcceleration stages a complete deterministic projection and
// atomically swaps the exact scoped active rows plus the CAS-protected head.
// It intentionally refuses a database-backed store: the FORGE-K caller must
// provide the surrounding apply+journal transaction.
func (s *SQLiteSemanticStore) RebuildMemoryAcceleration(ctx context.Context, req MemoryAccelerationRebuildRequest) (MemoryAccelerationCommit, error) {
	if s == nil || s.exec == nil {
		return MemoryAccelerationCommit{}, ErrMemoryAccelerationTransactionRequired
	}
	if _, ok := s.exec.(*sql.Tx); !ok {
		return MemoryAccelerationCommit{}, ErrMemoryAccelerationTransactionRequired
	}
	req.Scope.WorkspaceID = strings.TrimSpace(req.Scope.WorkspaceID)
	req.Scope.LaneID = strings.TrimSpace(req.Scope.LaneID)
	req.ExpectedManifestHash = strings.TrimSpace(req.ExpectedManifestHash)
	req.ExpectedPriorManifestHash = strings.TrimSpace(req.ExpectedPriorManifestHash)

	if req.RequestedAtMs <= 0 {
		return MemoryAccelerationCommit{}, fmt.Errorf("memory acceleration requested timestamp is required")
	}
	projection, err := s.PlanMemoryAcceleration(ctx, req.Scope, req.Algorithm)
	if err != nil {
		return MemoryAccelerationCommit{}, err
	}
	if err := vsaprojection.VerifyExpectedManifest(projection, req.ExpectedManifestHash); err != nil {
		return MemoryAccelerationCommit{}, err
	}

	current, exists, err := s.memoryAccelerationHead(ctx, req.Scope)
	if err != nil {
		return MemoryAccelerationCommit{}, err
	}
	if exists && current == projection.Manifest.ManifestHash {
		return MemoryAccelerationCommit{}, ErrMemoryAccelerationManifestActive
	}
	if (!exists && req.ExpectedPriorManifestHash != "") || (exists && current != req.ExpectedPriorManifestHash) {
		return MemoryAccelerationCommit{}, fmt.Errorf("%w: expected %q got %q", ErrMemoryAccelerationHeadConflict, req.ExpectedPriorManifestHash, current)
	}

	manifestJSON, err := json.Marshal(projection.Manifest)
	if err != nil {
		return MemoryAccelerationCommit{}, err
	}
	createdAt := req.RequestedAtMs
	meta := s.meta
	if strings.TrimSpace(meta.SyscallID) == "" || strings.TrimSpace(meta.CorrelationID) == "" || strings.TrimSpace(meta.TraceID) == "" {
		return MemoryAccelerationCommit{}, fmt.Errorf("memory acceleration commit metadata is incomplete")
	}
	committedBy := strings.TrimSpace(meta.CommittedBy)
	if committedBy == "" {
		committedBy = "forge_k.kernel"
	}
	if _, err := s.exec.ExecContext(ctx, `
INSERT OR IGNORE INTO memory_vsa_projection_manifests(
  manifest_hash, workspace_id, lane_id, source_set_hash, link_set_hash,
  algorithm_name, algorithm_version, dimensions, seed, source_count, link_count,
  manifest_json, syscall_id, correlation_id, trace_id, committed_by, created_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		projection.Manifest.ManifestHash, req.Scope.WorkspaceID, req.Scope.LaneID,
		projection.Manifest.SourceSetHash, projection.Manifest.LinkSetHash,
		projection.Manifest.Algorithm.Name, projection.Manifest.Algorithm.Version,
		projection.Manifest.Algorithm.Dimensions, projection.Manifest.Algorithm.Seed,
		projection.Manifest.SourceCount, projection.Manifest.LinkCount, string(manifestJSON),
		meta.SyscallID, meta.CorrelationID, meta.TraceID, committedBy, createdAt,
	); err != nil {
		return MemoryAccelerationCommit{}, err
	}
	var storedManifestJSON string
	if err := s.exec.QueryRowContext(ctx, `SELECT manifest_json FROM memory_vsa_projection_manifests WHERE manifest_hash=?`, projection.Manifest.ManifestHash).Scan(&storedManifestJSON); err != nil {
		return MemoryAccelerationCommit{}, err
	}
	if storedManifestJSON != string(manifestJSON) {
		return MemoryAccelerationCommit{}, fmt.Errorf("stored memory acceleration manifest identity mismatch")
	}

	if err := s.stageMemoryAccelerationProjection(ctx, projection); err != nil {
		return MemoryAccelerationCommit{}, err
	}
	if err := s.swapMemoryAccelerationProjection(ctx, req.Scope, projection.Manifest.ManifestHash, createdAt); err != nil {
		return MemoryAccelerationCommit{}, err
	}
	if exists {
		result, err := s.exec.ExecContext(ctx, `
UPDATE memory_vsa_projection_heads
SET manifest_hash=?, prior_manifest_hash=?, syscall_id=?, correlation_id=?, trace_id=?, committed_by=?, updated_at=?
WHERE workspace_id=? AND lane_id=? AND manifest_hash=?`,
			projection.Manifest.ManifestHash, current, meta.SyscallID, meta.CorrelationID, meta.TraceID, committedBy, createdAt,
			req.Scope.WorkspaceID, req.Scope.LaneID, current)
		if err != nil {
			return MemoryAccelerationCommit{}, err
		}
		changed, err := result.RowsAffected()
		if err != nil || changed != 1 {
			return MemoryAccelerationCommit{}, ErrMemoryAccelerationHeadConflict
		}
	} else {
		if _, err := s.exec.ExecContext(ctx, `
INSERT INTO memory_vsa_projection_heads(
  workspace_id, lane_id, manifest_hash, prior_manifest_hash, syscall_id,
  correlation_id, trace_id, committed_by, updated_at
) VALUES(?,?,?,?,?,?,?,?,?)`,
			req.Scope.WorkspaceID, req.Scope.LaneID, projection.Manifest.ManifestHash, "",
			meta.SyscallID, meta.CorrelationID, meta.TraceID, committedBy, createdAt,
		); err != nil {
			return MemoryAccelerationCommit{}, fmt.Errorf("%w: %v", ErrMemoryAccelerationHeadConflict, err)
		}
	}

	return MemoryAccelerationCommit{
		Manifest: projection.Manifest, PriorManifestHash: current,
		PointerCount: len(projection.Pointers), BindingCount: len(projection.Bindings), AssociationCount: len(projection.Associations),
	}, nil
}

func (s *SQLiteSemanticStore) loadMemoryAccelerationSources(ctx context.Context, scope vsaprojection.Scope) ([]vsaprojection.Source, error) {
	if strings.TrimSpace(scope.WorkspaceID) == "" || strings.TrimSpace(scope.LaneID) == "" {
		return nil, vsaprojection.ErrInvalidScope
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT mo.id, mo.workspace_id, mo.lane_id, mo.type, mo.task_type, mo.project_key,
       mo.source_path, mo.summary, mo.raw_content, mo.entities_json, mo.tags_json,
       mo.related_files_json, mo.lineage_json,
       COALESCE(SUM(CASE WHEN lower(trim(mue.signal)) IN ('useful','success','succeeded') THEN 1 ELSE 0 END),0),
       COALESCE(SUM(CASE WHEN lower(trim(mue.signal)) IN ('noisy','not_useful','misleading','failed','failure') THEN 1 ELSE 0 END),0)
FROM memory_observations mo
LEFT JOIN memory_usefulness_events mue ON mue.observation_id=mo.id
WHERE mo.workspace_id=? AND mo.lane_id=? AND mo.workspace_id<>'' AND mo.lane_id<>''
GROUP BY mo.id
ORDER BY mo.id ASC`, scope.WorkspaceID, scope.LaneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []vsaprojection.Source{}
	for rows.Next() {
		var source vsaprojection.Source
		var entities, tags, related, lineage string
		if err := rows.Scan(
			&source.ID, &source.WorkspaceID, &source.LaneID, &source.Type, &source.TaskType, &source.ProjectKey,
			&source.SourcePath, &source.Summary, &source.RawContent, &entities, &tags, &related, &lineage,
			&source.SupportCount, &source.NoiseCount,
		); err != nil {
			return nil, err
		}
		if err := decodeStringList(entities, &source.Entities); err != nil {
			return nil, fmt.Errorf("observation %d entities: %w", source.ID, err)
		}
		if err := decodeStringList(tags, &source.Tags); err != nil {
			return nil, fmt.Errorf("observation %d tags: %w", source.ID, err)
		}
		if err := decodeStringList(related, &source.RelatedFiles); err != nil {
			return nil, fmt.Errorf("observation %d related files: %w", source.ID, err)
		}
		if err := decodeStringList(lineage, &source.Lineage); err != nil {
			return nil, fmt.Errorf("observation %d lineage: %w", source.ID, err)
		}
		out = append(out, source)
	}
	return out, rows.Err()
}

func decodeStringList(raw string, target *[]string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "[]"
	}
	return json.Unmarshal([]byte(raw), target)
}

func (s *SQLiteSemanticStore) loadMemoryAccelerationLinks(ctx context.Context, scope vsaprojection.Scope) ([]vsaprojection.Link, error) {
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, workspace_id, lane_id, from_observation_id, to_observation_id, relation_type
FROM memory_observation_links
WHERE workspace_id=? AND lane_id=? AND workspace_id<>'' AND lane_id<>''
ORDER BY id ASC`, scope.WorkspaceID, scope.LaneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []vsaprojection.Link{}
	for rows.Next() {
		var link vsaprojection.Link
		if err := rows.Scan(&link.ID, &link.WorkspaceID, &link.LaneID, &link.FromObservationID, &link.ToObservationID, &link.RelationType); err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, rows.Err()
}

func (s *SQLiteSemanticStore) memoryAccelerationHead(ctx context.Context, scope vsaprojection.Scope) (string, bool, error) {
	var head string
	err := s.exec.QueryRowContext(ctx, `
SELECT manifest_hash FROM memory_vsa_projection_heads WHERE workspace_id=? AND lane_id=?`, scope.WorkspaceID, scope.LaneID).Scan(&head)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return head, err == nil, err
}

func (s *SQLiteSemanticStore) stageMemoryAccelerationProjection(ctx context.Context, projection vsaprojection.Projection) error {
	statements := []string{
		`CREATE TEMP TABLE IF NOT EXISTS forge_vsa_pointer_stage(observation_id INTEGER PRIMARY KEY, dims INTEGER NOT NULL, pointer_json TEXT NOT NULL, norm REAL NOT NULL, source_fingerprint TEXT NOT NULL, support_count INTEGER NOT NULL, noise_count INTEGER NOT NULL)`,
		`CREATE TEMP TABLE IF NOT EXISTS forge_vsa_binding_stage(observation_id INTEGER NOT NULL, role TEXT NOT NULL, filler TEXT NOT NULL, weight REAL NOT NULL, support_count INTEGER NOT NULL, noise_count INTEGER NOT NULL, binding_json TEXT NOT NULL, PRIMARY KEY(observation_id,role,filler))`,
		`CREATE TEMP TABLE IF NOT EXISTS forge_vsa_association_stage(from_observation_id INTEGER NOT NULL, to_observation_id INTEGER NOT NULL, association_type TEXT NOT NULL, strength REAL NOT NULL, support_count INTEGER NOT NULL, noise_count INTEGER NOT NULL, evidence_json TEXT NOT NULL, PRIMARY KEY(from_observation_id,to_observation_id,association_type))`,
		`DELETE FROM forge_vsa_pointer_stage`,
		`DELETE FROM forge_vsa_binding_stage`,
		`DELETE FROM forge_vsa_association_stage`,
	}
	for _, statement := range statements {
		if _, err := s.exec.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	for _, pointer := range projection.Pointers {
		vector, err := json.Marshal(pointer.Vector)
		if err != nil {
			return err
		}
		if _, err := s.exec.ExecContext(ctx, `INSERT INTO forge_vsa_pointer_stage VALUES(?,?,?,?,?,?,?)`, pointer.ObservationID, pointer.Dimensions, string(vector), pointer.Norm, pointer.SourceFingerprint, pointer.SupportCount, pointer.NoiseCount); err != nil {
			return err
		}
	}
	for _, binding := range projection.Bindings {
		vector, err := json.Marshal(binding.Vector)
		if err != nil {
			return err
		}
		if _, err := s.exec.ExecContext(ctx, `INSERT INTO forge_vsa_binding_stage VALUES(?,?,?,?,?,?,?)`, binding.ObservationID, binding.Role, binding.Filler, binding.Weight, binding.SupportCount, binding.NoiseCount, string(vector)); err != nil {
			return err
		}
	}
	for _, association := range projection.Associations {
		evidence, err := json.Marshal(map[string]any{"manifestHash": projection.Manifest.ManifestHash})
		if err != nil {
			return err
		}
		if _, err := s.exec.ExecContext(ctx, `INSERT INTO forge_vsa_association_stage VALUES(?,?,?,?,?,?,?)`, association.FromObservationID, association.ToObservationID, association.RelationType, association.Strength, association.SupportCount, association.NoiseCount, string(evidence)); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteSemanticStore) swapMemoryAccelerationProjection(ctx context.Context, scope vsaprojection.Scope, manifestHash string, now int64) error {
	deletes := []string{
		`DELETE FROM memory_vsa_associations WHERE workspace_id=? AND lane_id=?`,
		`DELETE FROM memory_vsa_role_bindings WHERE workspace_id=? AND lane_id=?`,
		`DELETE FROM memory_vsa_pointers WHERE workspace_id=? AND lane_id=?`,
	}
	for _, statement := range deletes {
		if _, err := s.exec.ExecContext(ctx, statement, scope.WorkspaceID, scope.LaneID); err != nil {
			return err
		}
	}
	if _, err := s.exec.ExecContext(ctx, `
INSERT INTO memory_vsa_pointers(workspace_id,lane_id,manifest_hash,observation_id,dims,pointer_json,norm,source_fingerprint,support_count,noise_count,stale,metadata_json,created_at,updated_at)
SELECT ?,?,?,observation_id,dims,pointer_json,norm,source_fingerprint,support_count,noise_count,0,?, ?, ? FROM forge_vsa_pointer_stage`,
		scope.WorkspaceID, scope.LaneID, manifestHash, `{"authority":"forge_k","projection":"vsa"}`, now, now); err != nil {
		return err
	}
	if _, err := s.exec.ExecContext(ctx, `
INSERT INTO memory_vsa_role_bindings(workspace_id,lane_id,manifest_hash,observation_id,role,filler,weight,support_count,noise_count,binding_json,created_at,updated_at)
SELECT ?,?,?,observation_id,role,filler,weight,support_count,noise_count,binding_json,?,? FROM forge_vsa_binding_stage`,
		scope.WorkspaceID, scope.LaneID, manifestHash, now, now); err != nil {
		return err
	}
	if _, err := s.exec.ExecContext(ctx, `
INSERT INTO memory_vsa_associations(workspace_id,lane_id,manifest_hash,from_observation_id,to_observation_id,association_type,strength,support_count,noise_count,evidence_json,created_at,updated_at)
SELECT ?,?,?,from_observation_id,to_observation_id,association_type,strength,support_count,noise_count,evidence_json,?,? FROM forge_vsa_association_stage`,
		scope.WorkspaceID, scope.LaneID, manifestHash, now, now); err != nil {
		return err
	}
	return nil
}
