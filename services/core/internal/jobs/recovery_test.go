package jobs

import (
	"context"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestRecoverInterruptedJobsRequeuesQueuedAndFailsRunning(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	insertRecoveryJob(t, st, "queued-1", StatusQueued)
	insertRecoveryJob(t, st, "preparing-1", StatusPreparing)
	insertRecoveryJob(t, st, "running-1", StatusRunning)
	insertRecoveryJob(t, st, "done-1", StatusSucceeded)

	svc := &Service{
		db:          st.DB,
		queue:       make(chan string, 256),
		stop:        make(chan struct{}),
		cancelFuncs: map[string]context.CancelFunc{},
	}
	if err := svc.recoverInterruptedJobs(context.Background()); err != nil {
		t.Fatalf("recover interrupted jobs: %v", err)
	}

	select {
	case got := <-svc.queue:
		if got != "queued-1" {
			t.Fatalf("expected queued job to be requeued, got %q", got)
		}
	default:
		t.Fatalf("expected queued job to be requeued")
	}
	running, err := svc.Get(context.Background(), "running-1")
	if err != nil {
		t.Fatalf("get running job: %v", err)
	}
	if running.Status != StatusFailed {
		t.Fatalf("running job status=%s, want failed", running.Status)
	}
	if running.LastFailureCode == nil || *running.LastFailureCode != FailInterrupted {
		t.Fatalf("running failure code=%v, want interrupted", running.LastFailureCode)
	}
	preparing, err := svc.Get(context.Background(), "preparing-1")
	if err != nil {
		t.Fatalf("get preparing job: %v", err)
	}
	if preparing.Status != StatusFailed {
		t.Fatalf("preparing job status=%s, want failed", preparing.Status)
	}
	if preparing.LastFailureCode == nil || *preparing.LastFailureCode != FailInterrupted {
		t.Fatalf("preparing failure code=%v, want interrupted", preparing.LastFailureCode)
	}
	assertEventTypes(t, svc, "preparing-1", []string{"job.status.changed", "job.failed", "job.recovered"})
	assertStatusHistory(t, svc, "preparing-1", []Status{StatusFailed})
	done, err := svc.Get(context.Background(), "done-1")
	if err != nil {
		t.Fatalf("get done job: %v", err)
	}
	if done.Status != StatusSucceeded {
		t.Fatalf("terminal job status=%s, want succeeded", done.Status)
	}
}

func insertRecoveryJob(t *testing.T, st *store.Store, id string, status Status) {
	t.Helper()
	now := time.Now().UnixMilli()
	_, err := st.DB.Exec(`
INSERT INTO jobs(
  id, created_at, updated_at, queued_at, title, requested_action, target_adapter,
  initiating_source, execution_boundary, risk_class, status, approval_status,
  write_intent, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		now,
		now,
		now,
		"Recovery job",
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
