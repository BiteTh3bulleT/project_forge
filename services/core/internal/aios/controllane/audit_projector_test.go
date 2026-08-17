package controllane

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/audit"
)

func TestAuditProjectorDeliversCommittedOutboxExactlyOnce(t *testing.T) {
	ctx := context.Background()
	processor, _, st := newSQLiteKernel(t, nil)
	req := validSQLiteRequest(domain.ActionCreateNote, "audit-project-1", "ws-audit")
	req.IdempotencyKey = "idem-audit-project-1"
	req.Payload = map[string]any{
		"id": "note-audit-project-1", "type": string(domain.NoteFact),
		"title": "projection", "content": "deliver exactly once",
	}
	result, err := processContextThroughForgeK(ctx, processor, req)
	if err != nil || !result.Success {
		t.Fatalf("commit through FORGE-K: result=%#v err=%v", result, err)
	}
	if result.AuditID != "" || result.StateSummary["auditProjection"] != "queued" {
		t.Fatalf("commit performed synchronous audit delivery: %#v", result)
	}

	sink := NewCoreAuditSink(audit.New(st.DB))
	projector, err := NewAuditProjector(AuditProjectorOptions{
		DB: st.DB, Sink: sink, NowMillis: func() int64 { return req.RequestedAt + 1000 },
	})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	report, err := projector.RunOnce(ctx)
	if err != nil || report.Delivered != 1 || report.Retried != 0 || report.Quarantined != 0 {
		t.Fatalf("first delivery: report=%#v err=%v", report, err)
	}
	report, err = projector.RunOnce(ctx)
	if err != nil || report.Scanned != 0 {
		t.Fatalf("terminal outbox was redelivered: report=%#v err=%v", report, err)
	}
	assertAuditProjectionCounts(t, st.DB, req.ID+":audit_outbox", 1, 1, 1)
	if _, err := st.DB.Exec(`UPDATE forge_k_audit_delivery_attempts SET status='retry' WHERE outbox_id=?`, req.ID+":audit_outbox"); err == nil {
		t.Fatal("delivery attempt update was not rejected")
	}
	if _, err := st.DB.Exec(`DELETE FROM forge_k_audit_delivery_attempts WHERE outbox_id=?`, req.ID+":audit_outbox"); err == nil {
		t.Fatal("delivery attempt delete was not rejected")
	}
}

func TestAuditProjectorRetriesAcrossRestartWithoutDuplicateSinkRecord(t *testing.T) {
	ctx := context.Background()
	processor, _, st := newSQLiteKernel(t, nil)
	req := validSQLiteRequest(domain.ActionCreateNote, "audit-retry-1", "ws-audit")
	req.IdempotencyKey = "idem-audit-retry-1"
	req.Payload = map[string]any{
		"id": "note-audit-retry-1", "type": string(domain.NoteFact),
		"title": "retry", "content": "resume after restart",
	}
	if result, err := processContextThroughForgeK(ctx, processor, req); err != nil || !result.Success {
		t.Fatalf("commit through FORGE-K: result=%#v err=%v", result, err)
	}

	now := req.RequestedAt + 1000
	flaky := &flakyAuditOutboxSink{delegate: NewCoreAuditSink(audit.New(st.DB)), failures: 1}
	first, err := NewAuditProjector(AuditProjectorOptions{
		DB: st.DB, Sink: flaky, NowMillis: func() int64 { return now },
		BaseBackoff: time.Second, MaxBackoff: time.Minute,
	})
	if err != nil {
		t.Fatalf("new first projector: %v", err)
	}
	report, err := first.RunOnce(ctx)
	if err != nil || report.Retried != 1 || report.Delivered != 0 {
		t.Fatalf("record retry: report=%#v err=%v", report, err)
	}
	assertAuditProjectionCounts(t, st.DB, req.ID+":audit_outbox", 1, 0, 0)

	// Reconstructing the projector models a daemon restart. The durable retry
	// attempt controls backoff; no in-memory retry state is required.
	now += 2 * time.Second.Milliseconds()
	restarted, err := NewAuditProjector(AuditProjectorOptions{
		DB: st.DB, Sink: flaky, NowMillis: func() int64 { return now },
		BaseBackoff: time.Second, MaxBackoff: time.Minute,
	})
	if err != nil {
		t.Fatalf("new restarted projector: %v", err)
	}
	report, err = restarted.RunOnce(ctx)
	if err != nil || report.Delivered != 1 {
		t.Fatalf("restart delivery: report=%#v err=%v", report, err)
	}
	assertAuditProjectionCounts(t, st.DB, req.ID+":audit_outbox", 2, 1, 1)
	if flaky.calls != 2 {
		t.Fatalf("unexpected sink call count: %d", flaky.calls)
	}
}

func TestAuditProjectorQuarantinesInvalidProofWithoutCallingSink(t *testing.T) {
	ctx := context.Background()
	processor, runner, st := newSQLiteKernel(t, nil)
	req := validSQLiteRequest(domain.ActionCreateNote, "audit-poison-source", "ws-audit")
	req.IdempotencyKey = "idem-audit-poison-source"
	req.Payload = map[string]any{
		"id": "note-audit-poison-source", "type": string(domain.NoteFact),
		"title": "poison", "content": "proof source",
	}
	if result, err := processContextThroughForgeK(ctx, processor, req); err != nil || !result.Success {
		t.Fatalf("commit through FORGE-K: result=%#v err=%v", result, err)
	}
	rec, ok := runner.ReadStore().GetAuditOutbox(req.ID + ":audit_outbox")
	if !ok {
		t.Fatal("missing committed audit outbox")
	}
	// Insert a second immutable row whose JSON is well formed but whose receipt
	// no longer binds the row identity. The projector must quarantine it before
	// the sink sees it.
	rec.ID = "poison:audit_outbox"
	rec.SyscallID = "poison"
	rec.RequestFingerprint = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, err := st.DB.Exec(`INSERT INTO forge_k_audit_outbox(
  id,syscall_id,request_fingerprint,action,workspace_id,lane_id,correlation_id,trace_id,
  success,result_json,request_json,receipt_json,authproof_json,created_at,committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, rec.ID, rec.SyscallID, rec.RequestFingerprint, string(rec.Action),
		rec.WorkspaceID, rec.LaneID, rec.CorrelationID, rec.TraceID, 1, encodeJSON(rec.Result),
		encodeJSON(rec.Request), encodeJSON(rec.Receipt), encodeJSON(rec.AuthorizationProof), rec.CreatedAt+1, rec.CommittedBy)
	if err != nil {
		t.Fatalf("insert poison outbox fixture: %v", err)
	}
	sink := &flakyAuditOutboxSink{delegate: NewCoreAuditSink(audit.New(st.DB))}
	projector, err := NewAuditProjector(AuditProjectorOptions{DB: st.DB, Sink: sink, NowMillis: func() int64 { return rec.CreatedAt + 1000 }})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	report, err := projector.RunOnce(ctx)
	if err != nil || report.Delivered != 1 || report.Quarantined != 1 {
		t.Fatalf("poison projection: report=%#v err=%v", report, err)
	}
	if sink.calls != 1 {
		t.Fatalf("invalid outbox reached sink: calls=%d", sink.calls)
	}
	var status, code string
	if err := st.DB.QueryRow(`SELECT status,error_code FROM forge_k_audit_delivery_attempts WHERE outbox_id=?`, rec.ID).Scan(&status, &code); err != nil || status != AuditDeliveryQuarantined || code != "invalid_outbox_proof" {
		t.Fatalf("invalid quarantine evidence: status=%q code=%q err=%v", status, code, err)
	}
}

type flakyAuditOutboxSink struct {
	delegate AuditOutboxSink
	failures int
	calls    int
}

func (s *flakyAuditOutboxSink) DeliverOutbox(ctx context.Context, rec AuditOutboxRecord) (string, error) {
	s.calls++
	if s.failures > 0 {
		s.failures--
		return "", errors.New("injected sink outage")
	}
	return s.delegate.DeliverOutbox(ctx, rec)
}

func assertAuditProjectionCounts(t *testing.T, db queryRower, outboxID string, attempts, delivered, records int) {
	t.Helper()
	var gotAttempts, gotDelivered, gotRecords int
	if err := db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(CASE WHEN status='delivered' THEN 1 ELSE 0 END),0)
FROM forge_k_audit_delivery_attempts WHERE outbox_id=?`, outboxID).Scan(&gotAttempts, &gotDelivered); err != nil {
		t.Fatalf("query delivery attempts: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_records WHERE forge_k_outbox_id=?`, outboxID).Scan(&gotRecords); err != nil {
		t.Fatalf("query projected audits: %v", err)
	}
	if gotAttempts != attempts || gotDelivered != delivered || gotRecords != records {
		t.Fatalf("projection counts attempts=%d/%d delivered=%d/%d records=%d/%d",
			gotAttempts, attempts, gotDelivered, delivered, gotRecords, records)
	}
}

type queryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}
