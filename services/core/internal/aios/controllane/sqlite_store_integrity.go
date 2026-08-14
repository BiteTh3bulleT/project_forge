package controllane

import (
	"encoding/json"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

// CreateAuditOutbox inserts the immutable audit intent. When called on the
// transaction-scoped SQLiteSemanticStore it participates in the same UnitOfWork
// as the canonical mutation, journal record, provenance, and idempotency row.
func (s *SQLiteSemanticStore) CreateAuditOutbox(rec AuditOutboxRecord) error {
	rec = normalizeAuditOutboxRecord(rec)
	if rec.ID == "" || rec.SyscallID == "" {
		return ErrInvalidAuditOutboxRecord
	}
	if existing, ok := s.GetAuditOutbox(rec.ID); ok {
		if auditOutboxRecordsEqual(existing, rec) {
			return nil
		}
		return fmt.Errorf("%w: id %q", ErrAuditOutboxConflict, rec.ID)
	}
	if _, ok := s.findAuditOutboxBySyscall(rec.SyscallID); ok {
		return fmt.Errorf("%w: syscall %q", ErrAuditOutboxConflict, rec.SyscallID)
	}
	if err := validateAuditOutboxRecord(rec); err != nil {
		return err
	}
	result, err := s.exec.ExecContext(s.background, `
INSERT INTO forge_k_audit_outbox(
  id, syscall_id, request_fingerprint, action, workspace_id, lane_id,
  correlation_id, trace_id, success, result_json, request_json, receipt_json, authproof_json, created_at, committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT DO NOTHING`,
		rec.ID, rec.SyscallID, rec.RequestFingerprint, string(rec.Action), rec.WorkspaceID, rec.LaneID,
		rec.CorrelationID, rec.TraceID, boolToInt(rec.Success), encodeJSON(rec.Result), encodeJSON(rec.Request), encodeJSON(rec.Receipt),
		encodeJSON(rec.AuthorizationProof), rec.CreatedAt,
		nonEmpty(rec.CommittedBy, "forge_k.kernel"),
	)
	if err != nil {
		return err
	}
	if inserted, _ := result.RowsAffected(); inserted == 1 {
		return nil
	}
	existing, ok := s.GetAuditOutbox(rec.ID)
	if !ok {
		return fmt.Errorf("%w: syscall %q", ErrAuditOutboxConflict, rec.SyscallID)
	}
	if auditOutboxRecordsEqual(existing, rec) {
		return nil
	}
	return fmt.Errorf("%w: id %q", ErrAuditOutboxConflict, rec.ID)
}

func (s *SQLiteSemanticStore) findAuditOutboxBySyscall(syscallID string) (AuditOutboxRecord, bool) {
	row := s.exec.QueryRowContext(s.background, auditOutboxSelect+` WHERE syscall_id = ?`, strings.TrimSpace(syscallID))
	rec, err := scanAuditOutbox(row)
	return rec, err == nil
}

func (s *SQLiteSemanticStore) GetAuditOutbox(id string) (AuditOutboxRecord, bool) {
	row := s.exec.QueryRowContext(s.background, auditOutboxSelect+` WHERE id = ?`, strings.TrimSpace(id))
	rec, err := scanAuditOutbox(row)
	return rec, err == nil
}

func (s *SQLiteSemanticStore) ListAuditOutbox(limit int) []AuditOutboxRecord {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.exec.QueryContext(s.background, auditOutboxSelect+` ORDER BY created_at ASC, id ASC LIMIT ?`, limit)
	if err != nil {
		return []AuditOutboxRecord{}
	}
	defer rows.Close()
	out := make([]AuditOutboxRecord, 0)
	for rows.Next() {
		rec, err := scanAuditOutbox(rows)
		if err == nil {
			out = append(out, rec)
		}
	}
	return out
}

const auditOutboxSelect = `
SELECT id, syscall_id, request_fingerprint, action, workspace_id, lane_id,
       correlation_id, trace_id, success, result_json, request_json, receipt_json, authproof_json, created_at, committed_by
FROM forge_k_audit_outbox`

func scanAuditOutbox(row rowScanner) (AuditOutboxRecord, error) {
	var rec AuditOutboxRecord
	var action, resultJSON, requestJSON, receiptJSON, authproofJSON string
	var success int
	err := row.Scan(
		&rec.ID, &rec.SyscallID, &rec.RequestFingerprint, &action, &rec.WorkspaceID, &rec.LaneID,
		&rec.CorrelationID, &rec.TraceID, &success, &resultJSON, &requestJSON, &receiptJSON, &authproofJSON, &rec.CreatedAt, &rec.CommittedBy,
	)
	if err != nil {
		return AuditOutboxRecord{}, err
	}
	rec.Action = domain.SemanticActionType(action)
	rec.Success = success != 0
	if err := json.Unmarshal([]byte(resultJSON), &rec.Result); err != nil {
		return AuditOutboxRecord{}, err
	}
	if err := json.Unmarshal([]byte(requestJSON), &rec.Request); err != nil {
		return AuditOutboxRecord{}, err
	}
	if err := json.Unmarshal([]byte(receiptJSON), &rec.Receipt); err != nil {
		return AuditOutboxRecord{}, err
	}
	if err := json.Unmarshal([]byte(authproofJSON), &rec.AuthorizationProof); err != nil {
		return AuditOutboxRecord{}, err
	}
	return rec, nil
}

func auditOutboxRecordsEqual(left, right AuditOutboxRecord) bool {
	if !auditOutboxSameIdentity(left, right) || left.WorkspaceID != right.WorkspaceID || left.LaneID != right.LaneID ||
		left.CorrelationID != right.CorrelationID || left.TraceID != right.TraceID || left.Success != right.Success ||
		left.CreatedAt != right.CreatedAt || nonEmpty(left.CommittedBy, "forge_k.kernel") != nonEmpty(right.CommittedBy, "forge_k.kernel") {
		return false
	}
	return encodeJSON(left.Result) == encodeJSON(right.Result) && encodeJSON(left.Request) == encodeJSON(right.Request) &&
		encodeJSON(left.Receipt) == encodeJSON(right.Receipt) &&
		encodeJSON(left.AuthorizationProof) == encodeJSON(right.AuthorizationProof)
}
