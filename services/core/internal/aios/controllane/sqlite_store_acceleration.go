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
  manifest_hash, source_kind, workspace_id, lane_id, source_set_hash, link_set_hash,
  algorithm_name, algorithm_version, dimensions, seed, source_count, link_count,
  manifest_json, syscall_id, correlation_id, trace_id, committed_by, created_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		projection.Manifest.ManifestHash, "forge_k_memory_evidence", req.Scope.WorkspaceID, req.Scope.LaneID,
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
SELECT e.id,e.evidence_id,e.root_evidence_id,e.revision,
       e.court_case_id,e.court_exhibit_id,e.court_ruling_id,e.admission_syscall_id,
       e.source_object_kind,e.source_object_id,e.source_object_version,e.source_object_hash,
       e.workspace_id,e.lane_id,e.source_type,e.raw_ref,e.content_summary,
       e.source_refs_json,e.selected_paths_json,e.source_provenance_id,e.materialization_provenance_id,
       e.syscall_id,e.transaction_id,e.journal_event_id,e.audit_outbox_id,e.authorization_fingerprint,e.committed_by
FROM forge_k_memory_evidence e
JOIN court_exhibits x
  ON x.id=e.court_exhibit_id AND x.case_id=e.court_case_id
 AND x.workspace_id=e.workspace_id AND x.lane_id=e.lane_id
 AND x.status='admitted' AND x.current_ruling_id=e.court_ruling_id
 AND x.content_hash=e.source_object_hash AND x.committed_by='forge_k.kernel'
JOIN court_rulings r
  ON r.id=e.court_ruling_id AND r.exhibit_id=e.court_exhibit_id AND r.case_id=e.court_case_id
 AND r.workspace_id=e.workspace_id AND r.lane_id=e.lane_id
 AND r.decision='admitted' AND r.content_hash=e.source_object_hash
 AND r.syscall_id=e.admission_syscall_id AND r.committed_by='forge_k.kernel'
LEFT JOIN forge_k_memory_evidence_supersessions superseded
  ON superseded.superseded_evidence_id=e.evidence_id
WHERE e.workspace_id=? AND e.lane_id=? AND e.workspace_id<>'' AND e.lane_id<>''
  AND e.source_object_kind='court_exhibit'
  AND e.source_object_id=e.court_exhibit_id AND e.source_object_version=e.court_ruling_id
  AND e.content_hash=e.source_object_hash AND e.committed_by='forge_k.kernel'
  AND superseded.id IS NULL
ORDER BY e.evidence_id ASC,e.id ASC`, scope.WorkspaceID, scope.LaneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []vsaprojection.Source{}
	for rows.Next() {
		var source vsaprojection.Source
		var sourceRefsJSON, selectedPathsJSON string
		if err := rows.Scan(
			&source.MemoryEvidenceRowID, &source.EvidenceID, &source.RootEvidenceID, &source.Revision,
			&source.CourtCaseID, &source.CourtExhibitID, &source.CourtRulingID, &source.AdmissionSyscallID,
			&source.SourceObjectKind, &source.SourceObjectID, &source.SourceObjectVersion, &source.SourceObjectHash,
			&source.WorkspaceID, &source.LaneID, &source.Type, &source.SourcePath, &source.Summary,
			&sourceRefsJSON, &selectedPathsJSON, &source.SourceProvenanceID, &source.MaterializationProvenanceID,
			&source.SyscallID, &source.TransactionID, &source.JournalEventID, &source.AuditOutboxID,
			&source.AuthorizationFingerprint, &source.CommittedBy,
		); err != nil {
			return nil, err
		}
		if err := decodeStringList(sourceRefsJSON, &source.Entities); err != nil {
			return nil, fmt.Errorf("memory evidence %s source refs: %w", source.EvidenceID, err)
		}
		if err := decodeStringList(selectedPathsJSON, &source.RelatedFiles); err != nil {
			return nil, fmt.Errorf("memory evidence %s selected paths: %w", source.EvidenceID, err)
		}
		source.Tags = []string{"court_admitted", source.CourtCaseID, source.SourceObjectKind}
		source.Lineage = []string{source.RootEvidenceID, source.EvidenceID, source.CourtExhibitID, source.CourtRulingID}
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
	if strings.TrimSpace(scope.WorkspaceID) == "" || strings.TrimSpace(scope.LaneID) == "" {
		return nil, vsaprojection.ErrInvalidScope
	}
	// K20H admits immutable evidence but does not yet define governed semantic
	// relationship edges. Legacy memory_observation_links are never imported.
	return []vsaprojection.Link{}, nil
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
		`CREATE TEMP TABLE IF NOT EXISTS forge_vsa_pointer_stage(memory_evidence_row_id INTEGER PRIMARY KEY, memory_evidence_id TEXT NOT NULL UNIQUE, dims INTEGER NOT NULL, pointer_json TEXT NOT NULL, norm REAL NOT NULL, source_fingerprint TEXT NOT NULL, support_count INTEGER NOT NULL, noise_count INTEGER NOT NULL)`,
		`CREATE TEMP TABLE IF NOT EXISTS forge_vsa_binding_stage(memory_evidence_row_id INTEGER NOT NULL, memory_evidence_id TEXT NOT NULL, role TEXT NOT NULL, filler TEXT NOT NULL, weight REAL NOT NULL, support_count INTEGER NOT NULL, noise_count INTEGER NOT NULL, binding_json TEXT NOT NULL, PRIMARY KEY(memory_evidence_row_id,role,filler))`,
		`CREATE TEMP TABLE IF NOT EXISTS forge_vsa_association_stage(from_memory_evidence_row_id INTEGER NOT NULL, to_memory_evidence_row_id INTEGER NOT NULL, association_type TEXT NOT NULL, strength REAL NOT NULL, support_count INTEGER NOT NULL, noise_count INTEGER NOT NULL, evidence_json TEXT NOT NULL, PRIMARY KEY(from_memory_evidence_row_id,to_memory_evidence_row_id,association_type))`,
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
		if _, err := s.exec.ExecContext(ctx, `INSERT INTO forge_vsa_pointer_stage VALUES(?,?,?,?,?,?,?,?)`, pointer.MemoryEvidenceRowID, pointer.EvidenceID, pointer.Dimensions, string(vector), pointer.Norm, pointer.SourceFingerprint, pointer.SupportCount, pointer.NoiseCount); err != nil {
			return err
		}
	}
	for _, binding := range projection.Bindings {
		vector, err := json.Marshal(binding.Vector)
		if err != nil {
			return err
		}
		if _, err := s.exec.ExecContext(ctx, `INSERT INTO forge_vsa_binding_stage VALUES(?,?,?,?,?,?,?,?)`, binding.MemoryEvidenceRowID, binding.EvidenceID, binding.Role, binding.Filler, binding.Weight, binding.SupportCount, binding.NoiseCount, string(vector)); err != nil {
			return err
		}
	}
	for _, association := range projection.Associations {
		evidence, err := json.Marshal(map[string]any{"manifestHash": projection.Manifest.ManifestHash})
		if err != nil {
			return err
		}
		if _, err := s.exec.ExecContext(ctx, `INSERT INTO forge_vsa_association_stage VALUES(?,?,?,?,?,?,?)`, association.FromMemoryEvidenceRowID, association.ToMemoryEvidenceRowID, association.RelationType, association.Strength, association.SupportCount, association.NoiseCount, string(evidence)); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteSemanticStore) swapMemoryAccelerationProjection(ctx context.Context, scope vsaprojection.Scope, manifestHash string, now int64) error {
	deletes := []string{
		`DELETE FROM forge_k_memory_vsa_associations WHERE workspace_id=? AND lane_id=?`,
		`DELETE FROM forge_k_memory_vsa_role_bindings WHERE workspace_id=? AND lane_id=?`,
		`DELETE FROM forge_k_memory_vsa_pointers WHERE workspace_id=? AND lane_id=?`,
	}
	for _, statement := range deletes {
		if _, err := s.exec.ExecContext(ctx, statement, scope.WorkspaceID, scope.LaneID); err != nil {
			return err
		}
	}
	if _, err := s.exec.ExecContext(ctx, `
INSERT INTO forge_k_memory_vsa_pointers(workspace_id,lane_id,manifest_hash,memory_evidence_row_id,memory_evidence_id,dims,pointer_json,norm,source_fingerprint,support_count,noise_count,metadata_json,created_at,updated_at)
SELECT ?,?,?,memory_evidence_row_id,memory_evidence_id,dims,pointer_json,norm,source_fingerprint,support_count,noise_count,?, ?, ? FROM forge_vsa_pointer_stage`,
		scope.WorkspaceID, scope.LaneID, manifestHash, `{"authority":"forge_k","projection":"vsa"}`, now, now); err != nil {
		return err
	}
	if _, err := s.exec.ExecContext(ctx, `
INSERT INTO forge_k_memory_vsa_role_bindings(workspace_id,lane_id,manifest_hash,memory_evidence_row_id,memory_evidence_id,role,filler,weight,support_count,noise_count,binding_json,created_at,updated_at)
SELECT ?,?,?,memory_evidence_row_id,memory_evidence_id,role,filler,weight,support_count,noise_count,binding_json,?,? FROM forge_vsa_binding_stage`,
		scope.WorkspaceID, scope.LaneID, manifestHash, now, now); err != nil {
		return err
	}
	if _, err := s.exec.ExecContext(ctx, `
INSERT INTO forge_k_memory_vsa_associations(workspace_id,lane_id,manifest_hash,from_memory_evidence_row_id,to_memory_evidence_row_id,association_type,strength,support_count,noise_count,evidence_json,created_at,updated_at)
SELECT ?,?,?,from_memory_evidence_row_id,to_memory_evidence_row_id,association_type,strength,support_count,noise_count,evidence_json,?,? FROM forge_vsa_association_stage`,
		scope.WorkspaceID, scope.LaneID, manifestHash, now, now); err != nil {
		return err
	}
	return nil
}
