package controllane

import (
	"encoding/json"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
)

const memoryEvidenceSelect = `
SELECT id,evidence_id,root_evidence_id,revision,court_case_id,court_exhibit_id,court_ruling_id,
       admission_syscall_id,source_object_kind,source_object_id,source_object_version,source_object_hash,
       workspace_id,lane_id,selected_paths_json,source_type,source_refs_json,content_summary,raw_ref,content_hash,
       source_provenance_id,source_provenance_json,materialization_provenance_id,materialization_provenance_json,
       created_at,proposed_by,committed_by,syscall_id,correlation_id,trace_id,transaction_id,journal_event_id,
       audit_outbox_id,idempotency_key,authorization_fingerprint
FROM forge_k_memory_evidence`

func (s *SQLiteSemanticStore) FindMemoryEvidence(id string, scope domain.ForgeScope) (MemoryEvidence, bool) {
	row := s.exec.QueryRowContext(s.background, memoryEvidenceSelect+`
WHERE evidence_id=? AND workspace_id=? AND lane_id=?`, strings.TrimSpace(id), scope.WorkspaceID, scope.LaneID)
	evidence, err := scanMemoryEvidence(row)
	if err != nil || !exactUtilityScopeMatches(evidence.Scope, scope) {
		return MemoryEvidence{}, false
	}
	return evidence, true
}

func (s *SQLiteSemanticStore) HasMemoryEvidenceSupersession(id string) bool {
	var one int
	err := s.exec.QueryRowContext(s.background, `
SELECT 1 FROM forge_k_memory_evidence_supersessions WHERE superseded_evidence_id=? LIMIT 1`, strings.TrimSpace(id)).Scan(&one)
	return err == nil
}

func (s *SQLiteSemanticStore) CreateMemoryEvidence(evidence MemoryEvidence, edge *MemoryEvidenceSupersession) error {
	if s == nil || s.exec == nil {
		return fmt.Errorf("memory evidence store is unavailable")
	}
	if err := validateMemoryEvidenceForPersistence(evidence, edge); err != nil {
		return err
	}
	ctx := s.background
	sourceProvenanceID, err := s.ensureProvenance(ctx, evidence.Scope, evidence.SourceProvenance, map[string]any{
		"object_type": "court_exhibit", "case_id": evidence.CourtCaseID, "exhibit_id": evidence.CourtExhibitID,
	}, evidence.CreatedAt)
	if err != nil {
		return err
	}
	if sourceProvenanceID != evidence.SourceProvenanceID {
		return fmt.Errorf("source provenance identity mismatch")
	}
	materializationProvenanceID, err := s.ensureProvenance(ctx, evidence.Scope, evidence.MaterializationProvenance, map[string]any{
		"object_type": "forge_k_memory_evidence", "evidence_id": evidence.EvidenceID,
	}, evidence.CreatedAt)
	if err != nil {
		return err
	}
	if materializationProvenanceID != evidence.MaterializationProvenanceID {
		return fmt.Errorf("materialization provenance identity mismatch")
	}
	inserted, err := s.exec.ExecContext(ctx, `
INSERT INTO forge_k_memory_evidence(
  evidence_id,root_evidence_id,revision,court_case_id,court_exhibit_id,court_ruling_id,
  admission_syscall_id,source_object_kind,source_object_id,source_object_version,source_object_hash,
  workspace_id,lane_id,selected_paths_json,source_type,source_refs_json,content_summary,raw_ref,content_hash,
  source_provenance_id,source_provenance_json,materialization_provenance_id,materialization_provenance_json,
  created_at,proposed_by,committed_by,syscall_id,correlation_id,trace_id,transaction_id,journal_event_id,
  audit_outbox_id,idempotency_key,authorization_fingerprint
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		evidence.EvidenceID, evidence.RootEvidenceID, evidence.Revision, evidence.CourtCaseID,
		evidence.CourtExhibitID, evidence.CourtRulingID, evidence.AdmissionSyscallID,
		evidence.SourceObjectKind, evidence.SourceObjectID, evidence.SourceObjectVersion, evidence.SourceObjectHash,
		evidence.Scope.WorkspaceID, evidence.Scope.LaneID, encodeStringSlice(evidence.Scope.SelectedPaths),
		evidence.SourceType, encodeStringSlice(evidence.SourceRefs), evidence.ContentSummary, evidence.RawRef, evidence.ContentHash,
		evidence.SourceProvenanceID, encodeJSON(evidence.SourceProvenance), evidence.MaterializationProvenanceID,
		encodeJSON(evidence.MaterializationProvenance), evidence.CreatedAt, evidence.ProposedBy, evidence.CommittedBy,
		evidence.SyscallID, evidence.CorrelationID, evidence.TraceID, evidence.TransactionID, evidence.JournalEventID,
		evidence.AuditOutboxID, evidence.IdempotencyKey, evidence.AuthorizationFingerprint,
	)
	if err != nil {
		return err
	}
	evidence.RowID, err = inserted.LastInsertId()
	if err != nil || evidence.RowID <= 0 {
		return fmt.Errorf("resolve memory evidence row id: %w", err)
	}
	if edge == nil {
		return nil
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO forge_k_memory_evidence_supersessions(
  id,root_evidence_id,superseded_evidence_id,replacement_evidence_id,workspace_id,lane_id,
  selected_paths_json,provenance_id,provenance_json,created_at,syscall_id,correlation_id,trace_id,committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, edge.ID, edge.RootEvidenceID, edge.SupersededEvidenceID,
		edge.ReplacementEvidenceID, edge.Scope.WorkspaceID, edge.Scope.LaneID, encodeStringSlice(edge.Scope.SelectedPaths),
		edge.ProvenanceID, encodeJSON(edge.Provenance), edge.CreatedAt, edge.SyscallID, edge.CorrelationID,
		edge.TraceID, edge.CommittedBy)
	return err
}

func validateMemoryEvidenceForPersistence(evidence MemoryEvidence, edge *MemoryEvidenceSupersession) error {
	if strings.TrimSpace(evidence.EvidenceID) == "" || strings.TrimSpace(evidence.RootEvidenceID) == "" || evidence.Revision < 1 {
		return fmt.Errorf("memory evidence identity/revision is invalid")
	}
	if evidence.SourceObjectKind != "court_exhibit" || evidence.SourceObjectID != evidence.CourtExhibitID ||
		evidence.SourceObjectVersion != evidence.CourtRulingID || evidence.SourceObjectHash != evidence.ContentHash ||
		!validMemoryEvidenceHash(evidence.ContentHash) {
		return fmt.Errorf("memory evidence source identity is inconsistent")
	}
	if strings.TrimSpace(evidence.Scope.WorkspaceID) == "" || strings.TrimSpace(evidence.Scope.LaneID) == "" || len(evidence.Scope.SelectedPaths) == 0 ||
		strings.TrimSpace(evidence.SourceProvenanceID) == "" || strings.TrimSpace(evidence.MaterializationProvenanceID) == "" ||
		strings.TrimSpace(evidence.IdempotencyKey) == "" || !validMemoryEvidenceHash(evidence.AuthorizationFingerprint) {
		return fmt.Errorf("memory evidence authority evidence is incomplete")
	}
	if evidence.CommittedBy != forgekernel.AuthorityOwnerForgeK || evidence.TransactionID != evidence.SyscallID+":transaction" ||
		evidence.JournalEventID != evidence.SyscallID+":journal_event" || evidence.AuditOutboxID != evidence.SyscallID+":audit_outbox" {
		return fmt.Errorf("memory evidence commit identity is inconsistent")
	}
	if edge == nil {
		if evidence.RootEvidenceID != evidence.EvidenceID || evidence.Revision != 1 {
			return fmt.Errorf("initial memory evidence lineage is invalid")
		}
		return nil
	}
	if edge.ID != evidence.SyscallID+":memory_supersession" || edge.ReplacementEvidenceID != evidence.EvidenceID ||
		edge.RootEvidenceID != evidence.RootEvidenceID || !exactUtilityScopeMatches(edge.Scope, evidence.Scope) ||
		edge.ProvenanceID != evidence.MaterializationProvenanceID || edge.CommittedBy != forgekernel.AuthorityOwnerForgeK {
		return fmt.Errorf("memory evidence supersession is inconsistent")
	}
	return nil
}

func scanMemoryEvidence(row rowScanner) (MemoryEvidence, error) {
	var evidence MemoryEvidence
	var selectedPathsJSON, sourceRefsJSON, sourceProvenanceJSON, materializationProvenanceJSON string
	err := row.Scan(
		&evidence.RowID, &evidence.EvidenceID, &evidence.RootEvidenceID, &evidence.Revision,
		&evidence.CourtCaseID, &evidence.CourtExhibitID, &evidence.CourtRulingID, &evidence.AdmissionSyscallID,
		&evidence.SourceObjectKind, &evidence.SourceObjectID, &evidence.SourceObjectVersion, &evidence.SourceObjectHash,
		&evidence.Scope.WorkspaceID, &evidence.Scope.LaneID, &selectedPathsJSON, &evidence.SourceType, &sourceRefsJSON,
		&evidence.ContentSummary, &evidence.RawRef, &evidence.ContentHash, &evidence.SourceProvenanceID, &sourceProvenanceJSON,
		&evidence.MaterializationProvenanceID, &materializationProvenanceJSON, &evidence.CreatedAt, &evidence.ProposedBy,
		&evidence.CommittedBy, &evidence.SyscallID, &evidence.CorrelationID, &evidence.TraceID, &evidence.TransactionID,
		&evidence.JournalEventID, &evidence.AuditOutboxID, &evidence.IdempotencyKey, &evidence.AuthorizationFingerprint,
	)
	if err != nil {
		return MemoryEvidence{}, err
	}
	evidence.Scope.SelectedPaths = decodeStringSlice(selectedPathsJSON)
	evidence.SourceRefs = decodeStringSlice(sourceRefsJSON)
	if err := json.Unmarshal([]byte(sourceProvenanceJSON), &evidence.SourceProvenance); err != nil {
		return MemoryEvidence{}, err
	}
	if err := json.Unmarshal([]byte(materializationProvenanceJSON), &evidence.MaterializationProvenance); err != nil {
		return MemoryEvidence{}, err
	}
	return evidence, nil
}
