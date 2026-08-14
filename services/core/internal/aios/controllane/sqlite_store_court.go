package controllane

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/court"
)

func (s *SQLiteSemanticStore) FindCourtExhibit(id string, scope domain.ForgeScope) (court.Exhibit, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id, case_id, workspace_id, lane_id, selected_paths_json, source_type, source_refs_json,
       content_summary, raw_ref, content_hash, status, current_ruling_id, provenance_json,
       created_at, updated_at, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
FROM court_exhibits WHERE id = ? AND workspace_id = ? AND (? = '' OR lane_id = ?)`, strings.TrimSpace(id), scope.WorkspaceID, scope.LaneID, scope.LaneID)
	var exhibit court.Exhibit
	var selectedPathsJSON, sourceRefsJSON, provenanceJSON string
	if err := row.Scan(
		&exhibit.ID, &exhibit.CaseID, &exhibit.Scope.WorkspaceID, &exhibit.Scope.LaneID, &selectedPathsJSON,
		&exhibit.SourceType, &sourceRefsJSON, &exhibit.ContentSummary, &exhibit.RawRef, &exhibit.ContentHash,
		&exhibit.Status, &exhibit.CurrentRulingID, &provenanceJSON, &exhibit.CreatedAt, &exhibit.UpdatedAt,
		&exhibit.ProposedBy, &exhibit.CommittedBy, &exhibit.SyscallID, &exhibit.CorrelationID, &exhibit.TraceID, &exhibit.AuditID,
	); err != nil {
		return court.Exhibit{}, false
	}
	exhibit.Scope.SelectedPaths = decodeStringSlice(selectedPathsJSON)
	exhibit.SourceRefs = decodeStringSlice(sourceRefsJSON)
	_ = json.Unmarshal([]byte(provenanceJSON), &exhibit.Provenance)
	return exhibit, true
}

func (s *SQLiteSemanticStore) FindCourtRuling(id string, scope domain.ForgeScope) (court.Ruling, bool) {
	row := s.exec.QueryRowContext(s.background, courtRulingSelect+` WHERE id = ? AND workspace_id = ? AND (? = '' OR lane_id = ?)`, strings.TrimSpace(id), scope.WorkspaceID, scope.LaneID, scope.LaneID)
	ruling, err := scanCourtRuling(row)
	return ruling, err == nil
}

func (s *SQLiteSemanticStore) ListCourtRulings(scope domain.ForgeScope, caseID, exhibitID string) []court.Ruling {
	rows, err := s.exec.QueryContext(s.background, courtRulingSelect+`
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND (? = '' OR case_id = ?) AND (? = '' OR exhibit_id = ?)
ORDER BY created_at ASC, id ASC`, scope.WorkspaceID, scope.LaneID, scope.LaneID, caseID, caseID, exhibitID, exhibitID)
	if err != nil {
		return []court.Ruling{}
	}
	defer rows.Close()
	out := make([]court.Ruling, 0)
	for rows.Next() {
		ruling, err := scanCourtRuling(rows)
		if err == nil {
			out = append(out, ruling)
		}
	}
	return out
}

func (s *SQLiteSemanticStore) CreateCourtDecision(exhibit court.Exhibit, ruling court.Ruling, appeal *court.Appeal) error {
	ctx := s.background
	provenanceID, err := s.ensureProvenance(ctx, exhibit.Scope, exhibit.Provenance, map[string]any{
		"object_type": "court_exhibit", "case_id": exhibit.CaseID, "exhibit_id": exhibit.ID,
	}, exhibit.CreatedAt)
	if err != nil {
		return err
	}
	if appeal == nil {
		_, err = s.exec.ExecContext(ctx, `
INSERT INTO court_exhibits(
  id, case_id, workspace_id, lane_id, selected_paths_json, source_type, source_refs_json,
  content_summary, raw_ref, content_hash, status, current_ruling_id, provenance_id, provenance_json,
  created_at, updated_at, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			exhibit.ID, exhibit.CaseID, exhibit.Scope.WorkspaceID, exhibit.Scope.LaneID, encodeStringSlice(exhibit.Scope.SelectedPaths),
			exhibit.SourceType, encodeStringSlice(exhibit.SourceRefs), exhibit.ContentSummary, exhibit.RawRef, exhibit.ContentHash,
			exhibit.Status, exhibit.CurrentRulingID, provenanceID, encodeJSON(exhibit.Provenance), exhibit.CreatedAt, exhibit.UpdatedAt,
			exhibit.ProposedBy, exhibit.CommittedBy, exhibit.SyscallID, exhibit.CorrelationID, exhibit.TraceID, exhibit.AuditID,
		)
		if err != nil {
			return err
		}
	} else {
		if appeal.CaseID != exhibit.CaseID || appeal.ExhibitID != exhibit.ID ||
			appeal.Scope.WorkspaceID != exhibit.Scope.WorkspaceID || appeal.Scope.LaneID != exhibit.Scope.LaneID {
			return fmt.Errorf("court appeal scope/case/exhibit mismatch")
		}
		appealProvenanceID, err := s.ensureProvenance(ctx, appeal.Scope, appeal.Provenance, map[string]any{
			"object_type": "court_appeal", "case_id": appeal.CaseID, "exhibit_id": appeal.ExhibitID,
		}, appeal.CreatedAt)
		if err != nil {
			return err
		}
		_, err = s.exec.ExecContext(ctx, `
INSERT INTO court_appeals(
  id, case_id, exhibit_id, prior_ruling_id, new_ruling_id, workspace_id, lane_id, selected_paths_json,
  grounds, new_source_refs_json, new_content_hash, provenance_id, provenance_json, created_at,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			appeal.ID, appeal.CaseID, appeal.ExhibitID, appeal.PriorRulingID, appeal.NewRulingID,
			appeal.Scope.WorkspaceID, appeal.Scope.LaneID, encodeStringSlice(appeal.Scope.SelectedPaths), appeal.Grounds,
			encodeStringSlice(appeal.NewSourceRefs), appeal.NewContentHash, appealProvenanceID, encodeJSON(appeal.Provenance), appeal.CreatedAt,
			appeal.ProposedBy, appeal.CommittedBy, appeal.SyscallID, appeal.CorrelationID, appeal.TraceID, appeal.AuditID,
		)
		if err != nil {
			return err
		}
	}

	rulingProvenanceID, err := s.ensureProvenance(ctx, ruling.Scope, ruling.Provenance, map[string]any{
		"object_type": "court_ruling", "case_id": ruling.CaseID, "exhibit_id": ruling.ExhibitID,
	}, ruling.CreatedAt)
	if err != nil {
		return err
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO court_rulings(
  id, case_id, exhibit_id, appeal_id, prior_ruling_id, workspace_id, lane_id, selected_paths_json,
  decision, reason_code, reason, policy_version, policy_refs_json, input_refs_json, content_hash,
  provenance_id, provenance_json, created_at, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ruling.ID, ruling.CaseID, ruling.ExhibitID, ruling.AppealID, ruling.PriorRulingID,
		ruling.Scope.WorkspaceID, ruling.Scope.LaneID, encodeStringSlice(ruling.Scope.SelectedPaths), ruling.Decision,
		ruling.ReasonCode, ruling.Reason, ruling.PolicyVersion, encodeStringSlice(ruling.PolicyRefs), encodeStringSlice(ruling.InputRefs),
		ruling.ContentHash, rulingProvenanceID, encodeJSON(ruling.Provenance), ruling.CreatedAt, ruling.ProposedBy, ruling.CommittedBy,
		ruling.SyscallID, ruling.CorrelationID, ruling.TraceID, ruling.AuditID,
	)
	if err != nil {
		return err
	}
	if appeal != nil {
		result, err := s.exec.ExecContext(ctx, `
UPDATE court_exhibits
SET status = ?, current_ruling_id = ?, updated_at = ?, syscall_id = ?, correlation_id = ?, trace_id = ?, audit_id = ''
WHERE id = ? AND case_id = ? AND workspace_id = ? AND lane_id = ? AND current_ruling_id = ?`,
			exhibit.Status, exhibit.CurrentRulingID, exhibit.UpdatedAt, exhibit.SyscallID, exhibit.CorrelationID, exhibit.TraceID,
			exhibit.ID, exhibit.CaseID, exhibit.Scope.WorkspaceID, exhibit.Scope.LaneID, appeal.PriorRulingID,
		)
		if err != nil {
			return err
		}
		if changed, _ := result.RowsAffected(); changed != 1 {
			return fmt.Errorf("court appeal prior ruling is not current")
		}
	}
	return nil
}

const courtRulingSelect = `
SELECT id, case_id, exhibit_id, appeal_id, prior_ruling_id, workspace_id, lane_id, selected_paths_json,
       decision, reason_code, reason, policy_version, policy_refs_json, input_refs_json, content_hash,
       provenance_json, created_at, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
FROM court_rulings`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCourtRuling(row rowScanner) (court.Ruling, error) {
	var ruling court.Ruling
	var selectedPathsJSON, policyRefsJSON, inputRefsJSON, provenanceJSON string
	err := row.Scan(
		&ruling.ID, &ruling.CaseID, &ruling.ExhibitID, &ruling.AppealID, &ruling.PriorRulingID,
		&ruling.Scope.WorkspaceID, &ruling.Scope.LaneID, &selectedPathsJSON, &ruling.Decision, &ruling.ReasonCode,
		&ruling.Reason, &ruling.PolicyVersion, &policyRefsJSON, &inputRefsJSON, &ruling.ContentHash, &provenanceJSON,
		&ruling.CreatedAt, &ruling.ProposedBy, &ruling.CommittedBy, &ruling.SyscallID, &ruling.CorrelationID, &ruling.TraceID, &ruling.AuditID,
	)
	if err != nil {
		return court.Ruling{}, err
	}
	ruling.Scope.SelectedPaths = decodeStringSlice(selectedPathsJSON)
	ruling.PolicyRefs = decodeStringSlice(policyRefsJSON)
	ruling.InputRefs = decodeStringSlice(inputRefsJSON)
	_ = json.Unmarshal([]byte(provenanceJSON), &ruling.Provenance)
	return ruling, nil
}

var _ rowScanner = (*sql.Row)(nil)
var _ rowScanner = (*sql.Rows)(nil)
