package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestFailRecordsFailureStateHistoryAndEvent(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newFailureModeService(t)
	defer cleanup()
	insertFailureModeJob(t, svc.db, "fail-timeout", StatusRunning)

	if err := svc.fail(ctx, "fail-timeout", FailAdapterTimeout, "adapter did not answer"); err != nil {
		t.Fatalf("fail returned error: %v", err)
	}

	job, err := svc.Get(ctx, "fail-timeout")
	if err != nil {
		t.Fatalf("Get failed job: %v", err)
	}
	if job.Status != StatusFailed {
		t.Fatalf("status = %s, want failed", job.Status)
	}
	if job.LastFailureCode == nil || *job.LastFailureCode != FailAdapterTimeout {
		t.Fatalf("last failure code = %v, want adapter timeout", job.LastFailureCode)
	}
	if job.LastError == nil || *job.LastError != "adapter did not answer" {
		t.Fatalf("last error = %v, want adapter did not answer", job.LastError)
	}
	if job.CompletedAtMs == nil {
		t.Fatalf("completed_at was not set for failed job")
	}
	assertEventTypes(t, svc, "fail-timeout", []string{"job.status.changed", "job.failed"})
	assertStatusHistory(t, svc, "fail-timeout", []Status{StatusFailed})
}

func TestMarkCancelledRecordsCancellationStateHistoryAndEvent(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newFailureModeService(t)
	defer cleanup()
	insertFailureModeJob(t, svc.db, "cancel-before-exec", StatusQueued)

	if err := svc.markCancelled(ctx, "cancel-before-exec", FailUserCancellation, "operator cancelled"); err != nil {
		t.Fatalf("markCancelled returned error: %v", err)
	}

	job, err := svc.Get(ctx, "cancel-before-exec")
	if err != nil {
		t.Fatalf("Get cancelled job: %v", err)
	}
	if job.Status != StatusCancelled || !job.CancelRequested {
		t.Fatalf("job status=%s cancelRequested=%v, want cancelled with request flag", job.Status, job.CancelRequested)
	}
	if job.LastFailureCode == nil || *job.LastFailureCode != FailUserCancellation {
		t.Fatalf("last failure code = %v, want user cancellation", job.LastFailureCode)
	}
	if job.CompletedAtMs == nil {
		t.Fatalf("completed_at was not set for cancelled job")
	}
	assertEventTypes(t, svc, "cancel-before-exec", []string{"job.status.changed", "job.cancelled"})
	assertStatusHistory(t, svc, "cancel-before-exec", []Status{StatusCancelled})
}

func TestRequestCancelCancelsRunningJobAndMarksQueuedTerminal(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newFailureModeService(t)
	defer cleanup()
	insertFailureModeJob(t, svc.db, "running-cancel", StatusRunning)
	insertFailureModeJob(t, svc.db, "queued-cancel", StatusQueued)

	cancelled := make(chan struct{})
	svc.setCancel("running-cancel", func() { close(cancelled) })
	if err := svc.RequestCancel(ctx, "running-cancel", "robert"); err != nil {
		t.Fatalf("RequestCancel running failed: %v", err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatalf("running cancellation did not invoke registered cancel function")
	}
	running, err := svc.Get(ctx, "running-cancel")
	if err != nil {
		t.Fatalf("Get running job failed: %v", err)
	}
	if !running.CancelRequested || running.Status != StatusRunning {
		t.Fatalf("running job status=%s cancelRequested=%v, want running with cancel requested", running.Status, running.CancelRequested)
	}
	assertEventTypes(t, svc, "running-cancel", []string{"job.cancel.requested"})

	if err := svc.RequestCancel(ctx, "queued-cancel", "robert"); err != nil {
		t.Fatalf("RequestCancel queued failed: %v", err)
	}
	queued, err := svc.Get(ctx, "queued-cancel")
	if err != nil {
		t.Fatalf("Get queued job failed: %v", err)
	}
	if queued.Status != StatusCancelled || !queued.CancelRequested {
		t.Fatalf("queued job status=%s cancelRequested=%v, want cancelled with cancel requested", queued.Status, queued.CancelRequested)
	}
	assertEventTypes(t, svc, "queued-cancel", []string{"job.cancel.requested", "job.status.changed", "job.cancelled"})
}

func TestAppendEventIfMissingIsIdempotent(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newFailureModeService(t)
	defer cleanup()
	insertFailureModeJob(t, svc.db, "event-once", StatusQueued)

	if _, err := svc.appendEventIfMissing(ctx, "event-once", "job.recovered", "Recovered", map[string]any{"fromStatus": StatusQueued}); err != nil {
		t.Fatalf("first appendEventIfMissing failed: %v", err)
	}
	if _, err := svc.appendEventIfMissing(ctx, "event-once", "job.recovered", "Recovered again", map[string]any{"fromStatus": StatusRunning}); err != nil {
		t.Fatalf("second appendEventIfMissing failed: %v", err)
	}
	events, err := svc.Events(ctx, "event-once", 0, 10)
	if err != nil {
		t.Fatalf("Events failed: %v", err)
	}
	if got := eventTypes(events); !slices.Equal(got, []string{"job.recovered"}) {
		t.Fatalf("event types = %v, want one recovered event", got)
	}
	if !json.Valid(events[0].Payload) {
		t.Fatalf("event payload is not valid json: %q", string(events[0].Payload))
	}
}

func newFailureModeService(t *testing.T) (*Service, func()) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	svc := &Service{
		db:          st.DB,
		cancelFuncs: map[string]context.CancelFunc{},
	}
	return svc, func() {
		if err := st.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}
}

func insertFailureModeJob(t *testing.T, db *sql.DB, id string, status Status) {
	t.Helper()
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
INSERT INTO jobs(
  id, created_at, updated_at, queued_at, title, requested_action, target_adapter,
  initiating_source, execution_boundary, risk_class, status, approval_status,
  write_intent, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		now,
		now,
		now,
		"Failure mode job",
		"test.action",
		"test-adapter",
		"test",
		"test",
		string(RiskReadOnly),
		string(status),
		string(ApprovalNotRequired),
		0,
		`{"templateId":"summarize_context"}`,
	)
	if err != nil {
		t.Fatalf("insert job %s: %v", id, err)
	}
}

func assertEventTypes(t *testing.T, svc *Service, jobID string, want []string) {
	t.Helper()
	events, err := svc.Events(context.Background(), jobID, 0, 20)
	if err != nil {
		t.Fatalf("Events(%s) failed: %v", jobID, err)
	}
	if got := eventTypes(events); !slices.Equal(got, want) {
		t.Fatalf("event types for %s = %v, want %v", jobID, got, want)
	}
}

func assertStatusHistory(t *testing.T, svc *Service, jobID string, want []Status) {
	t.Helper()
	history, err := svc.StatusHistory(context.Background(), jobID)
	if err != nil {
		t.Fatalf("StatusHistory(%s) failed: %v", jobID, err)
	}
	got := make([]Status, 0, len(history))
	for _, item := range history {
		got = append(got, item.ToStatus)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("status history for %s = %v, want %v", jobID, got, want)
	}
}

func eventTypes(events []JobEvent) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, event.Type)
	}
	return out
}
