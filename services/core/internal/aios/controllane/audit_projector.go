package controllane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/audit"
)

const (
	AuditDeliveryDelivered   = "delivered"
	AuditDeliveryRetry       = "retry"
	AuditDeliveryQuarantined = "quarantined"
)

// AuditDeliveryAttempt is append-only projection evidence. It never replaces
// or mutates the canonical forge_k_audit_outbox intent.
type AuditDeliveryAttempt struct {
	ID                 string `json:"id"`
	OutboxID           string `json:"outboxId"`
	RequestFingerprint string `json:"requestFingerprint"`
	AttemptNumber      int    `json:"attemptNumber"`
	Status             string `json:"status"`
	AuditID            string `json:"auditId,omitempty"`
	ErrorCode          string `json:"errorCode,omitempty"`
	ErrorMessage       string `json:"errorMessage,omitempty"`
	CreatedAt          int64  `json:"createdAt"`
}

type AuditProjectionReport struct {
	Scanned     int `json:"scanned"`
	Delivered   int `json:"delivered"`
	Retried     int `json:"retried"`
	Quarantined int `json:"quarantined"`
	Deferred    int `json:"deferred"`
}

type AuditProjectionStatus struct {
	OutboxTotal  int    `json:"outboxTotal"`
	Pending      int    `json:"pending"`
	Delivered    int    `json:"delivered"`
	Retrying     int    `json:"retrying"`
	Quarantined  int    `json:"quarantined"`
	LastAttempt  int64  `json:"lastAttempt,omitempty"`
	Healthy      bool   `json:"healthy"`
	HealthReason string `json:"healthReason"`
}

type AuditProjectorOptions struct {
	DB          *sql.DB
	Sink        AuditOutboxSink
	NowMillis   func() int64
	BatchSize   int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

// AuditProjector delivers immutable Kernel audit intents to the operator audit
// sink. Delivery is restart-safe because the sink consumes the outbox ID as an
// idempotency key and every attempt is appended durably.
type AuditProjector struct {
	db          *sql.DB
	sink        AuditOutboxSink
	nowMillis   func() int64
	batchSize   int
	baseBackoff time.Duration
	maxBackoff  time.Duration
}

func NewAuditProjector(opts AuditProjectorOptions) (*AuditProjector, error) {
	if opts.DB == nil || opts.Sink == nil {
		return nil, fmt.Errorf("audit projector database and idempotent sink required")
	}
	if opts.NowMillis == nil {
		opts.NowMillis = func() int64 { return time.Now().UnixMilli() }
	}
	if opts.BatchSize <= 0 || opts.BatchSize > 1000 {
		opts.BatchSize = 100
	}
	if opts.BaseBackoff <= 0 {
		opts.BaseBackoff = time.Second
	}
	if opts.MaxBackoff < opts.BaseBackoff {
		opts.MaxBackoff = 5 * time.Minute
	}
	return &AuditProjector{
		db: opts.DB, sink: opts.Sink, nowMillis: opts.NowMillis,
		batchSize: opts.BatchSize, baseBackoff: opts.BaseBackoff, maxBackoff: opts.MaxBackoff,
	}, nil
}

func (p *AuditProjector) Run(ctx context.Context, interval time.Duration, observe func(AuditProjectionReport, error)) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	run := func() {
		report, err := p.RunOnce(ctx)
		if observe != nil {
			observe(report, err)
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func (p *AuditProjector) RunOnce(ctx context.Context) (AuditProjectionReport, error) {
	var report AuditProjectionReport
	records, err := p.listUnfinished(ctx)
	if err != nil {
		return report, err
	}
	now := p.nowMillis()
	var runErr error
	for _, rec := range records {
		if err := ctx.Err(); err != nil {
			return report, errors.Join(runErr, err)
		}
		report.Scanned++
		last, hasLast, err := p.lastAttempt(ctx, rec.ID)
		if err != nil {
			runErr = errors.Join(runErr, err)
			continue
		}
		if hasLast && last.Status == AuditDeliveryRetry && now < last.CreatedAt+p.backoff(last.AttemptNumber).Milliseconds() {
			report.Deferred++
			continue
		}
		if err := VerifyAuditOutboxRecord(rec); err != nil {
			attemptErr := p.appendAttempt(ctx, rec, AuditDeliveryQuarantined, "", "invalid_outbox_proof", err.Error(), now)
			if attemptErr != nil {
				runErr = errors.Join(runErr, attemptErr)
				continue
			}
			report.Quarantined++
			continue
		}
		auditID, deliveryErr := p.sink.DeliverOutbox(ctx, rec)
		if deliveryErr != nil {
			if errors.Is(deliveryErr, audit.ErrForgeKOutboxConflict) {
				attemptErr := p.appendAttempt(ctx, rec, AuditDeliveryQuarantined, "", "sink_identity_conflict", deliveryErr.Error(), now)
				if attemptErr != nil {
					runErr = errors.Join(runErr, attemptErr)
					continue
				}
				report.Quarantined++
				continue
			}
			attemptErr := p.appendAttempt(ctx, rec, AuditDeliveryRetry, "", "sink_unavailable", deliveryErr.Error(), now)
			if attemptErr != nil {
				runErr = errors.Join(runErr, attemptErr)
				continue
			}
			report.Retried++
			continue
		}
		if strings.TrimSpace(auditID) == "" {
			attemptErr := p.appendAttempt(ctx, rec, AuditDeliveryRetry, "", "empty_audit_identity", "audit sink returned an empty identity", now)
			if attemptErr != nil {
				runErr = errors.Join(runErr, attemptErr)
				continue
			}
			report.Retried++
			continue
		}
		if err := p.appendAttempt(ctx, rec, AuditDeliveryDelivered, auditID, "", "", now); err != nil {
			// A concurrent projector may have won the terminal insert. Treat that
			// as success only when durable terminal evidence now exists.
			terminal, terminalErr := p.terminalAttempt(ctx, rec.ID)
			if terminalErr == nil && terminal.Status == AuditDeliveryDelivered && terminal.AuditID == auditID {
				report.Delivered++
				continue
			}
			runErr = errors.Join(runErr, err, terminalErr)
			continue
		}
		report.Delivered++
	}
	return report, runErr
}

// ReadAuditProjectionStatus derives current delivery health entirely from
// durable outbox and append-only attempt evidence.
func ReadAuditProjectionStatus(ctx context.Context, db *sql.DB) (AuditProjectionStatus, error) {
	var status AuditProjectionStatus
	if db == nil {
		return status, fmt.Errorf("audit projection database required")
	}
	err := db.QueryRowContext(ctx, `SELECT
  (SELECT COUNT(*) FROM forge_k_audit_outbox),
  (SELECT COUNT(*) FROM forge_k_audit_delivery_attempts WHERE status='delivered'),
  (SELECT COUNT(*) FROM forge_k_audit_delivery_attempts WHERE status='quarantined'),
  (SELECT COUNT(DISTINCT retry.outbox_id) FROM forge_k_audit_delivery_attempts retry
    WHERE retry.status='retry' AND NOT EXISTS (
      SELECT 1 FROM forge_k_audit_delivery_attempts terminal
      WHERE terminal.outbox_id=retry.outbox_id AND terminal.status IN ('delivered','quarantined')
    )),
  (SELECT COALESCE(MAX(created_at),0) FROM forge_k_audit_delivery_attempts),
  (SELECT COUNT(*) FROM forge_k_audit_outbox outbox WHERE NOT EXISTS (
    SELECT 1 FROM forge_k_audit_delivery_attempts terminal
    WHERE terminal.outbox_id=outbox.id AND terminal.status IN ('delivered','quarantined')
  ))`).Scan(&status.OutboxTotal, &status.Delivered, &status.Quarantined, &status.Retrying, &status.LastAttempt, &status.Pending)
	if err != nil {
		return AuditProjectionStatus{}, err
	}
	status.Healthy = status.Quarantined == 0
	if status.Quarantined > 0 {
		status.HealthReason = "one or more immutable audit intents failed proof verification and are quarantined"
	} else if status.Pending > 0 {
		status.HealthReason = "delivery backlog is durable and awaiting projection"
	} else {
		status.HealthReason = "all immutable audit intents have terminal delivery evidence"
	}
	return status, nil
}

func (p *AuditProjector) backoff(attempt int) time.Duration {
	if attempt < 1 {
		return 0
	}
	d := p.baseBackoff
	for i := 1; i < attempt && d < p.maxBackoff; i++ {
		if d > p.maxBackoff/2 {
			return p.maxBackoff
		}
		d *= 2
	}
	if d > p.maxBackoff {
		return p.maxBackoff
	}
	return d
}

func (p *AuditProjector) listUnfinished(ctx context.Context) ([]AuditOutboxRecord, error) {
	rows, err := p.db.QueryContext(ctx, auditOutboxSelect+`
WHERE NOT EXISTS (
  SELECT 1 FROM forge_k_audit_delivery_attempts attempt
  WHERE attempt.outbox_id = forge_k_audit_outbox.id
    AND attempt.status IN ('delivered','quarantined')
)
ORDER BY created_at ASC, id ASC LIMIT ?`, p.batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AuditOutboxRecord, 0)
	for rows.Next() {
		rec, scanErr := scanAuditOutbox(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (p *AuditProjector) appendAttempt(ctx context.Context, rec AuditOutboxRecord, status, auditID, errorCode, errorMessage string, createdAt int64) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var terminal int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM forge_k_audit_delivery_attempts
WHERE outbox_id=? AND status IN ('delivered','quarantined')`, rec.ID).Scan(&terminal); err != nil {
		return err
	}
	if terminal > 0 {
		return nil
	}
	var next int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(attempt_number),0)+1
FROM forge_k_audit_delivery_attempts WHERE outbox_id=?`, rec.ID).Scan(&next); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO forge_k_audit_delivery_attempts(
  id,outbox_id,request_fingerprint,attempt_number,status,audit_id,error_code,error_message,created_at
) VALUES(?,?,?,?,?,?,?,?,?)`, fmt.Sprintf("%s:delivery:%d", rec.ID, next), rec.ID, rec.RequestFingerprint,
		next, status, strings.TrimSpace(auditID), strings.TrimSpace(errorCode), boundedDeliveryError(errorMessage), createdAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (p *AuditProjector) lastAttempt(ctx context.Context, outboxID string) (AuditDeliveryAttempt, bool, error) {
	row := p.db.QueryRowContext(ctx, auditDeliveryAttemptSelect+`
WHERE outbox_id=? ORDER BY attempt_number DESC LIMIT 1`, outboxID)
	attempt, err := scanAuditDeliveryAttempt(row)
	if errors.Is(err, sql.ErrNoRows) {
		return AuditDeliveryAttempt{}, false, nil
	}
	return attempt, err == nil, err
}

func (p *AuditProjector) terminalAttempt(ctx context.Context, outboxID string) (AuditDeliveryAttempt, error) {
	row := p.db.QueryRowContext(ctx, auditDeliveryAttemptSelect+`
WHERE outbox_id=? AND status IN ('delivered','quarantined') ORDER BY attempt_number DESC LIMIT 1`, outboxID)
	return scanAuditDeliveryAttempt(row)
}

const auditDeliveryAttemptSelect = `SELECT id,outbox_id,request_fingerprint,attempt_number,
status,audit_id,error_code,error_message,created_at FROM forge_k_audit_delivery_attempts `

func scanAuditDeliveryAttempt(row rowScanner) (AuditDeliveryAttempt, error) {
	var out AuditDeliveryAttempt
	err := row.Scan(&out.ID, &out.OutboxID, &out.RequestFingerprint, &out.AttemptNumber,
		&out.Status, &out.AuditID, &out.ErrorCode, &out.ErrorMessage, &out.CreatedAt)
	return out, err
}

func boundedDeliveryError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 2048 {
		return value[:2048]
	}
	return value
}
