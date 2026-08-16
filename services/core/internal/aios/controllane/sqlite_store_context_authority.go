package controllane

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/contextcompile"
)

func (s *SQLiteSemanticStore) ListCurrentMemoryEvidence(scope domain.ForgeScope, limit int) ([]MemoryEvidence, error) {
	if limit <= 0 || limit > contextcompile.V1Policy().Limits.MaxSources {
		limit = contextcompile.V1Policy().Limits.MaxSources
	}
	rows, err := s.exec.QueryContext(s.background, memoryEvidenceSelect+`
WHERE workspace_id=? AND lane_id=? AND selected_paths_json=?
  AND NOT EXISTS (SELECT 1 FROM forge_k_memory_evidence_supersessions x WHERE x.superseded_evidence_id=forge_k_memory_evidence.evidence_id)
  AND EXISTS (
    SELECT 1 FROM court_exhibits e JOIN court_rulings r ON r.id=forge_k_memory_evidence.court_ruling_id
    WHERE e.id=forge_k_memory_evidence.court_exhibit_id
      AND e.current_ruling_id=r.id AND e.status='admitted' AND r.decision='admitted'
      AND e.content_hash=forge_k_memory_evidence.content_hash AND r.content_hash=forge_k_memory_evidence.content_hash
      AND e.committed_by='forge_k.kernel' AND r.committed_by='forge_k.kernel'
  )
ORDER BY evidence_id ASC LIMIT ?`, scope.WorkspaceID, scope.LaneID, encodeStringSlice(normalizedSelectedPaths(scope.SelectedPaths)), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MemoryEvidence{}
	for rows.Next() {
		evidence, scanErr := scanMemoryEvidence(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, evidence)
	}
	return out, rows.Err()
}

func (s *SQLiteSemanticStore) ListGovernedContextCandidates(scope domain.ForgeScope, query, snapshotKind string, limit int) ([]contextcompile.CandidateSnapshot, error) {
	if limit <= 0 || limit > contextcompile.V1Policy().Limits.MaxCandidates {
		limit = contextcompile.V1Policy().Limits.MaxCandidates
	}
	rows, err := s.exec.QueryContext(s.background, `
SELECT candidate_json FROM forge_k_context_bundles
WHERE workspace_id=? AND lane_id=? AND selected_paths_json=? AND query=? AND snapshot_kind=?
ORDER BY created_at DESC, packet_id ASC LIMIT ?`, scope.WorkspaceID, scope.LaneID,
		encodeStringSlice(normalizedSelectedPaths(scope.SelectedPaths)), strings.Join(strings.Fields(query), " "), snapshotKind, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []contextcompile.CandidateSnapshot{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var candidate contextcompile.CandidateSnapshot
		if err := json.Unmarshal([]byte(raw), &candidate); err != nil {
			return nil, fmt.Errorf("decode governed context candidate: %w", err)
		}
		out = append(out, candidate)
	}
	return out, rows.Err()
}

func (s *SQLiteSemanticStore) FindGovernedContextHead(scope domain.ForgeScope) (contextcompile.PriorSnapshotHead, bool, error) {
	var raw string
	err := s.exec.QueryRowContext(s.background, `SELECT head_json FROM forge_k_context_snapshot_heads WHERE scope_hash=?`, governedContextScopeKey(scope)).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contextcompile.PriorSnapshotHead{}, false, nil
		}
		return contextcompile.PriorSnapshotHead{}, false, err
	}
	var head contextcompile.PriorSnapshotHead
	if err := json.Unmarshal([]byte(raw), &head); err != nil {
		return contextcompile.PriorSnapshotHead{}, false, err
	}
	return head, true, nil
}

func (s *SQLiteSemanticStore) CreateGovernedContextBundle(bundle GovernedContextBundle) error {
	if err := validateGovernedContextBundle(bundle, s); err != nil {
		return err
	}
	if bundle.Decision.PacketID == "" || bundle.Candidate.SnapshotID != bundle.Decision.SnapshotID || bundle.CommittedBy != "forge_k.kernel" {
		return fmt.Errorf("governed context bundle authority is incomplete")
	}
	current, present, err := s.FindGovernedContextHead(bundle.Scope)
	if err != nil {
		return err
	}
	expected := bundle.Input.PriorSnapshotHead
	if present != expected.Present || (present && current.HeadHash != expected.HeadHash) {
		return fmt.Errorf("governed context head changed")
	}
	_, err = s.exec.ExecContext(s.background, `
INSERT INTO forge_k_context_bundles(
 packet_id,snapshot_id,workspace_id,lane_id,selected_paths_json,query,snapshot_kind,
 source_manifest_hash,request_hash,packet_commitment,snapshot_commitment,outcome_commitment,
 decision_digest,policy_digest,input_json,decision_json,candidate_json,sources_json,created_at,
 provenance_id,syscall_id,correlation_id,trace_id,transaction_id,journal_event_id,audit_outbox_id,
 authorization_fingerprint,committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		bundle.Decision.PacketID, bundle.Decision.SnapshotID, bundle.Scope.WorkspaceID, bundle.Scope.LaneID,
		encodeStringSlice(normalizedSelectedPaths(bundle.Scope.SelectedPaths)), bundle.Query, bundle.Candidate.SnapshotKind,
		bundle.Decision.SourceManifestHash, bundle.Decision.RequestHash, bundle.Decision.PacketCommitment,
		bundle.Decision.SnapshotCommitment, bundle.Decision.OutcomeCommitment, bundle.Decision.DecisionDigest,
		bundle.Decision.PolicyDigest, encodeJSON(bundle.Input), encodeJSON(bundle.Decision), encodeJSON(bundle.Candidate),
		encodeJSON(bundle.Sources), bundle.CreatedAt, bundle.ProvenanceID, bundle.SyscallID, bundle.CorrelationID,
		bundle.TraceID, bundle.TransactionID, bundle.JournalEventID, bundle.AuditOutboxID,
		bundle.AuthorizationFingerprint, bundle.CommittedBy)
	if err != nil {
		return err
	}
	revision := int64(1)
	if present {
		revision = current.Revision + 1
	}
	head, err := contextcompile.SealPriorSnapshotHead(contextcompile.PriorSnapshotHead{Present: true, Scope: contextCompileScope(bundle.Scope), SnapshotID: bundle.Candidate.SnapshotID, SnapshotHash: bundle.Candidate.SnapshotHash, Revision: revision, ProvenanceID: bundle.ProvenanceID, SyscallID: bundle.SyscallID, JournalEventID: bundle.JournalEventID, CommittedBy: bundle.CommittedBy})
	if err != nil {
		return err
	}
	if !present {
		_, err = s.exec.ExecContext(s.background, `INSERT INTO forge_k_context_snapshot_heads(scope_hash,workspace_id,lane_id,selected_paths_json,revision,head_hash,head_json,updated_at) VALUES(?,?,?,?,?,?,?,?)`, governedContextScopeKey(bundle.Scope), bundle.Scope.WorkspaceID, bundle.Scope.LaneID, encodeStringSlice(normalizedSelectedPaths(bundle.Scope.SelectedPaths)), revision, head.HeadHash, encodeJSON(head), bundle.CreatedAt)
		return err
	}
	result, err := s.exec.ExecContext(s.background, `UPDATE forge_k_context_snapshot_heads SET revision=?,head_hash=?,head_json=?,updated_at=? WHERE scope_hash=? AND revision=? AND head_hash=?`, revision, head.HeadHash, encodeJSON(head), bundle.CreatedAt, governedContextScopeKey(bundle.Scope), current.Revision, current.HeadHash)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return fmt.Errorf("governed context head CAS conflict")
	}
	return nil
}

func normalizedSelectedPaths(paths []string) []string {
	out := append([]string(nil), paths...)
	sort.Strings(out)
	return out
}
