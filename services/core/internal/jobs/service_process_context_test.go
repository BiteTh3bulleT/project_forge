package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestProcessUsesServiceContextCancellation(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	insertProcessJob(t, st, "process-cancelled", StatusQueued)
	rootCtx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := &Service{
		db:          st.DB,
		rootCtx:     rootCtx,
		cancelFuncs: map[string]context.CancelFunc{},
	}

	if err := svc.process("process-cancelled"); !errors.Is(err, context.Canceled) {
		t.Fatalf("process error=%v, want context.Canceled", err)
	}

	job, err := svc.Get(context.Background(), "process-cancelled")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job.Status != StatusQueued {
		t.Fatalf("job status=%s, want queued", job.Status)
	}
}

func insertProcessJob(t *testing.T, st *store.Store, id string, status Status) {
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
		"Process job",
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
