package approvals

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DefaultRequestTTL is the window after creation during which a pending
// approval request stays actionable. Requests past this TTL are swept into the
// "expired" terminal state by Expire.
const DefaultRequestTTL = 24 * time.Hour

type Request struct {
	ID               int64           `json:"id"`
	JobID            string          `json:"jobId"`
	CreatedAtMs      int64           `json:"createdAtMs"`
	ExpiresAtMs      int64           `json:"expiresAtMs"`
	ExpiredAtMs      int64           `json:"expiredAtMs,omitempty"`
	Status           string          `json:"status"`
	RequestedAction  string          `json:"requestedAction"`
	RiskClass        string          `json:"riskClass"`
	RequestedAdapter string          `json:"requestedAdapter"`
	WriteIntent      bool            `json:"writeIntent"`
	ScopeSnapshot    json.RawMessage `json:"scopeSnapshot"`
	TaskPacketID     *int64          `json:"taskPacketId"`
	RequestSummary   string          `json:"requestSummary"`
	Decision         *Decision       `json:"decision,omitempty"`
}

type Decision struct {
	ID          int64  `json:"id"`
	RequestID   int64  `json:"requestId"`
	CreatedAtMs int64  `json:"createdAtMs"`
	Actor       string `json:"actor"`
	Decision    string `json:"decision"`
	Note        string `json:"note"`
}

type CreateRequestInput struct {
	JobID            string
	RequestedAction  string
	RiskClass        string
	RequestedAdapter string
	WriteIntent      bool
	ScopeSnapshot    map[string]any
	TaskPacketID     *int64
	RequestSummary   string
	// TTL overrides DefaultRequestTTL for this request. Zero means use default.
	TTL time.Duration
}

type Service struct {
	db *sql.DB

	// waitersMu guards waiters. Callers subscribe to a request's decision via
	// Wait; Decide closes the channel for that request, delivering the terminal
	// status to everyone blocked on it.
	waitersMu sync.Mutex
	waiters   map[int64][]chan struct{}
}

func New(db *sql.DB) *Service {
	return &Service{db: db, waiters: map[int64][]chan struct{}{}}
}

func (s *Service) OpenRequestForJob(ctx context.Context, jobID string, in CreateRequestInput) (*Request, error) {
	// Reuse existing pending request when available.
	existing, err := s.LatestRequestByJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	incomingFingerprint := approvalFingerprintHashFromMap(in.ScopeSnapshot)
	if existing != nil && existing.Status == "pending" && approvalRequestMatchesFingerprint(existing, incomingFingerprint) {
		return existing, nil
	}

	scopeJSON, _ := json.Marshal(nonNilMap(in.ScopeSnapshot))
	now := time.Now().UnixMilli()
	ttl := in.TTL
	if ttl <= 0 {
		ttl = DefaultRequestTTL
	}
	expiresAt := now + ttl.Milliseconds()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO approval_requests(
  job_id, created_at, status, requested_action, risk_class, requested_adapter,
  write_intent, scope_snapshot_json, task_packet_id, request_summary, expires_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		in.JobID,
		now,
		"pending",
		in.RequestedAction,
		in.RiskClass,
		in.RequestedAdapter,
		boolToInt(in.WriteIntent),
		string(scopeJSON),
		in.TaskPacketID,
		in.RequestSummary,
		expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert approval request: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetRequest(ctx, id)
}

func approvalRequestMatchesFingerprint(req *Request, incoming string) bool {
	if strings.TrimSpace(incoming) == "" {
		return true
	}
	if req == nil {
		return false
	}
	existing := approvalFingerprintHashFromJSON(req.ScopeSnapshot)
	return existing != "" && existing == incoming
}

func approvalFingerprintHashFromMap(scope map[string]any) string {
	if scope == nil {
		return ""
	}
	if v, ok := scope["approvalShapeHash"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := scope["approvalFingerprintHash"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func approvalFingerprintHashFromJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var scope map[string]any
	if err := json.Unmarshal(raw, &scope); err != nil {
		return ""
	}
	return approvalFingerprintHashFromMap(scope)
}

func (s *Service) Decide(ctx context.Context, requestID int64, actor, decision, note string) (*Decision, error) {
	if decision != "approved" && decision != "denied" && decision != "cancelled" {
		return nil, fmt.Errorf("invalid decision %q", decision)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM approval_requests WHERE id = ?`, requestID).Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("approval request %d not found", requestID)
		}
		return nil, err
	}
	if status != "pending" {
		return nil, fmt.Errorf("approval request %d is not pending", requestID)
	}

	now := time.Now().UnixMilli()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO approval_decisions(request_id, created_at, actor, decision, note) VALUES(?,?,?,?,?)`,
		requestID,
		now,
		nonEmpty(actor, "operator"),
		decision,
		note,
	)
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE approval_requests SET status = ? WHERE id = ?`, "resolved", requestID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	did, _ := res.LastInsertId()
	s.notifyWaiters(requestID)
	return s.GetDecision(ctx, did)
}

// Wait returns a channel closed when the request reaches a terminal state
// (resolved via Decide, or expired via Expire). Callers should also respect
// ctx cancellation.
func (s *Service) Wait(ctx context.Context, requestID int64) <-chan struct{} {
	ch := make(chan struct{})
	// Fast path: if already resolved/expired, return an already-closed channel.
	status, err := s.status(ctx, requestID)
	if err == nil && status != "pending" {
		close(ch)
		return ch
	}
	s.waitersMu.Lock()
	s.waiters[requestID] = append(s.waiters[requestID], ch)
	s.waitersMu.Unlock()
	// Re-check after registering to avoid the race where Decide fires between
	// the fast-path read and the waiter registration.
	status, err = s.status(ctx, requestID)
	if err == nil && status != "pending" {
		s.notifyWaiters(requestID)
	}
	return ch
}

func (s *Service) status(ctx context.Context, requestID int64) (string, error) {
	var status string
	err := s.db.QueryRowContext(ctx, `SELECT status FROM approval_requests WHERE id = ?`, requestID).Scan(&status)
	return status, err
}

func (s *Service) notifyWaiters(requestID int64) {
	s.waitersMu.Lock()
	chans := s.waiters[requestID]
	delete(s.waiters, requestID)
	s.waitersMu.Unlock()
	for _, ch := range chans {
		select {
		case <-ch:
			// already closed
		default:
			close(ch)
		}
	}
}

// Expire sweeps pending requests whose expires_at has elapsed. It transitions
// them to status="expired" with an explicit timestamp, and notifies any
// waiters blocked on those requests. Intended to be run on a background ticker
// (every ~60s) by whichever service owns the approvals reaper.
func (s *Service) Expire(ctx context.Context) (int, error) {
	now := time.Now().UnixMilli()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM approval_requests WHERE status = 'pending' AND expires_at > 0 AND expires_at <= ?`,
		now,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, id := range ids {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE approval_requests SET status = 'expired', expired_at = ? WHERE id = ? AND status = 'pending'`,
			now, id,
		); err != nil {
			return 0, err
		}
		s.notifyWaiters(id)
	}
	return len(ids), nil
}

func (s *Service) GetRequest(ctx context.Context, id int64) (*Request, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, job_id, created_at, status, requested_action, risk_class, requested_adapter,
       write_intent, scope_snapshot_json, task_packet_id, request_summary, expires_at, expired_at
FROM approval_requests WHERE id = ?`, id)
	var r Request
	var writeInt int
	var scope string
	var packet sql.NullInt64
	if err := row.Scan(&r.ID, &r.JobID, &r.CreatedAtMs, &r.Status, &r.RequestedAction, &r.RiskClass, &r.RequestedAdapter, &writeInt, &scope, &packet, &r.RequestSummary, &r.ExpiresAtMs, &r.ExpiredAtMs); err != nil {
		return nil, err
	}
	r.WriteIntent = writeInt == 1
	r.ScopeSnapshot = json.RawMessage(scope)
	if packet.Valid {
		v := packet.Int64
		r.TaskPacketID = &v
	}
	dec, _ := s.LatestDecisionForRequest(ctx, r.ID)
	r.Decision = dec
	return &r, nil
}

func (s *Service) LatestRequestByJob(ctx context.Context, jobID string) (*Request, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id FROM approval_requests WHERE job_id = ? ORDER BY id DESC LIMIT 1`, jobID)
	var id int64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s.GetRequest(ctx, id)
}

func (s *Service) LatestDecisionForRequest(ctx context.Context, requestID int64) (*Decision, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, request_id, created_at, actor, decision, note FROM approval_decisions WHERE request_id = ? ORDER BY id DESC LIMIT 1`,
		requestID,
	)
	var d Decision
	if err := row.Scan(&d.ID, &d.RequestID, &d.CreatedAtMs, &d.Actor, &d.Decision, &d.Note); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (s *Service) GetDecision(ctx context.Context, id int64) (*Decision, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, request_id, created_at, actor, decision, note FROM approval_decisions WHERE id = ?`, id,
	)
	var d Decision
	if err := row.Scan(&d.ID, &d.RequestID, &d.CreatedAtMs, &d.Actor, &d.Decision, &d.Note); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Service) ListRequests(ctx context.Context, status string, limit int) ([]Request, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if status == "" {
		status = "pending"
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, job_id, created_at, status, requested_action, risk_class, requested_adapter,
       write_intent, scope_snapshot_json, task_packet_id, request_summary, expires_at, expired_at
FROM approval_requests WHERE status = ? ORDER BY id DESC LIMIT ?`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Request, 0, limit)
	for rows.Next() {
		var r Request
		var writeInt int
		var scope string
		var packet sql.NullInt64
		if err := rows.Scan(&r.ID, &r.JobID, &r.CreatedAtMs, &r.Status, &r.RequestedAction, &r.RiskClass, &r.RequestedAdapter, &writeInt, &scope, &packet, &r.RequestSummary, &r.ExpiresAtMs, &r.ExpiredAtMs); err != nil {
			return nil, err
		}
		r.WriteIntent = writeInt == 1
		r.ScopeSnapshot = json.RawMessage(scope)
		if packet.Valid {
			v := packet.Int64
			r.TaskPacketID = &v
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// IMPORTANT: Query decisions only after rows are closed. The store uses a
	// single SQLite connection; nested queries while iterating rows can block.
	for i := range out {
		dec, _ := s.LatestDecisionForRequest(ctx, out[i].ID)
		out[i].Decision = dec
	}
	return out, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func nonEmpty(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
