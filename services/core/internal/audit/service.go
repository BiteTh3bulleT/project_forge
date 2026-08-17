package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrForgeKOutboxConflict = errors.New("FORGE-K audit outbox projection conflict")

// Record is a single immutable audit entry. Audit records are meant to be
// append-only and carry a correlation id so related records across jobs,
// approvals, and gateway invocations can be rolled up into a trace.
type Record struct {
	ID                  int64           `json:"id"`
	ForgeKOutboxID      string          `json:"forgeKOutboxId,omitempty"`
	CreatedAtMs         int64           `json:"createdAtMs"`
	CorrelationID       string          `json:"correlationId"`
	Category            string          `json:"category"`
	Action              string          `json:"action"`
	Actor               string          `json:"actor"`
	SubjectType         string          `json:"subjectType"`
	SubjectID           string          `json:"subjectId"`
	JobID               *string         `json:"jobId,omitempty"`
	GatewayInvocationID *int64          `json:"gatewayInvocationId,omitempty"`
	ApprovalRequestID   *int64          `json:"approvalRequestId,omitempty"`
	RiskClass           string          `json:"riskClass"`
	Outcome             string          `json:"outcome"`
	Summary             string          `json:"summary"`
	Payload             json.RawMessage `json:"payload"`
}

type CreateRequest struct {
	CorrelationID       string
	Category            string
	Action              string
	Actor               string
	SubjectType         string
	SubjectID           string
	JobID               *string
	GatewayInvocationID *int64
	ApprovalRequestID   *int64
	RiskClass           string
	Outcome             string
	Summary             string
	Payload             map[string]any
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Record(ctx context.Context, req CreateRequest) (*Record, error) {
	return s.record(ctx, "", req)
}

// RecordForgeKOutbox projects one immutable FORGE-K audit intent into the
// operator audit log. outboxID is a sink-level idempotency key: retries after a
// crash return the original audit record instead of creating a duplicate.
func (s *Service) RecordForgeKOutbox(ctx context.Context, outboxID string, req CreateRequest) (*Record, error) {
	outboxID = strings.TrimSpace(outboxID)
	if outboxID == "" {
		return nil, fmt.Errorf("FORGE-K audit outbox id required")
	}
	return s.record(ctx, outboxID, req)
}

func (s *Service) record(ctx context.Context, outboxID string, req CreateRequest) (*Record, error) {
	if strings.TrimSpace(req.Category) == "" {
		return nil, fmt.Errorf("audit category required")
	}
	if strings.TrimSpace(req.Action) == "" {
		return nil, fmt.Errorf("audit action required")
	}
	payload := nonNilMap(req.Payload)
	raw, _ := json.Marshal(payload)
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO audit_records(
  forge_k_outbox_id, created_at, correlation_id, category, action, actor,
  subject_type, subject_id, job_id, gateway_invocation_id, approval_request_id,
  risk_class, outcome, summary, payload_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		outboxID,
		now,
		req.CorrelationID,
		strings.ToLower(req.Category),
		strings.ToLower(req.Action),
		nonEmpty(req.Actor, "operator"),
		req.SubjectType,
		req.SubjectID,
		req.JobID,
		req.GatewayInvocationID,
		req.ApprovalRequestID,
		req.RiskClass,
		req.Outcome,
		req.Summary,
		string(raw),
	)
	if err != nil {
		if outboxID != "" {
			existing, existingErr := s.GetByForgeKOutbox(ctx, outboxID)
			if existingErr == nil {
				if auditRecordMatchesRequest(existing, req, raw) {
					return existing, nil
				}
				return nil, fmt.Errorf("%w: outbox %q", ErrForgeKOutboxConflict, outboxID)
			}
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.Get(ctx, id)
}

// GetByForgeKOutbox resolves the stable projection identity used by the
// delivery worker. A missing row returns sql.ErrNoRows.
func (s *Service) GetByForgeKOutbox(ctx context.Context, outboxID string) (*Record, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, forge_k_outbox_id, created_at, correlation_id, category, action, actor,
       subject_type, subject_id, job_id, gateway_invocation_id, approval_request_id,
       risk_class, outcome, summary, payload_json
FROM audit_records WHERE forge_k_outbox_id = ?`, strings.TrimSpace(outboxID))
	return scanRecord(row)
}

type ListFilter struct {
	Limit         int
	Category      string
	CorrelationID string
	JobID         string
	Outcome       string
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Record, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	clauses := []string{}
	args := []any{}
	if strings.TrimSpace(filter.Category) != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, strings.ToLower(filter.Category))
	}
	if strings.TrimSpace(filter.CorrelationID) != "" {
		clauses = append(clauses, "correlation_id = ?")
		args = append(args, filter.CorrelationID)
	}
	if strings.TrimSpace(filter.JobID) != "" {
		clauses = append(clauses, "job_id = ?")
		args = append(args, filter.JobID)
	}
	if strings.TrimSpace(filter.Outcome) != "" {
		clauses = append(clauses, "outcome = ?")
		args = append(args, strings.ToLower(filter.Outcome))
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	args = append(args, limit)
	query := fmt.Sprintf(`
SELECT id, forge_k_outbox_id, created_at, correlation_id, category, action, actor,
       subject_type, subject_id, job_id, gateway_invocation_id, approval_request_id,
       risk_class, outcome, summary, payload_json
FROM audit_records %s ORDER BY id DESC LIMIT ?`, where)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// Trace returns the ordered (oldest-first) chain of audit records for a given
// correlation id. Used to reconstruct "what actually happened" for a single
// logical operation.
func (s *Service) Trace(ctx context.Context, correlationID string) ([]Record, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, forge_k_outbox_id, created_at, correlation_id, category, action, actor,
       subject_type, subject_id, job_id, gateway_invocation_id, approval_request_id,
       risk_class, outcome, summary, payload_json
FROM audit_records WHERE correlation_id = ? ORDER BY id ASC`, correlationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, forge_k_outbox_id, created_at, correlation_id, category, action, actor,
       subject_type, subject_id, job_id, gateway_invocation_id, approval_request_id,
       risk_class, outcome, summary, payload_json
FROM audit_records WHERE id = ?`, id)
	return scanRecord(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(row rowScanner) (*Record, error) {
	var r Record
	var job sql.NullString
	var gw sql.NullInt64
	var appr sql.NullInt64
	var payload string
	if err := row.Scan(
		&r.ID, &r.ForgeKOutboxID, &r.CreatedAtMs, &r.CorrelationID, &r.Category, &r.Action, &r.Actor,
		&r.SubjectType, &r.SubjectID, &job, &gw, &appr,
		&r.RiskClass, &r.Outcome, &r.Summary, &payload,
	); err != nil {
		return nil, err
	}
	if job.Valid {
		v := job.String
		r.JobID = &v
	}
	if gw.Valid {
		v := gw.Int64
		r.GatewayInvocationID = &v
	}
	if appr.Valid {
		v := appr.Int64
		r.ApprovalRequestID = &v
	}
	r.Payload = json.RawMessage(payload)
	return &r, nil
}

func auditRecordMatchesRequest(record *Record, req CreateRequest, payload []byte) bool {
	if record == nil {
		return false
	}
	return record.CorrelationID == req.CorrelationID && record.Category == strings.ToLower(req.Category) &&
		record.Action == strings.ToLower(req.Action) && record.Actor == nonEmpty(req.Actor, "operator") &&
		record.SubjectType == req.SubjectType && record.SubjectID == req.SubjectID &&
		record.RiskClass == req.RiskClass && record.Outcome == req.Outcome && record.Summary == req.Summary &&
		string(record.Payload) == string(payload)
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func nonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
