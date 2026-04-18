package approvals

import (
	"context"
	"fmt"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestListRequestsDoesNotDeadlockWithSingleSQLiteConn(t *testing.T) {
	t.Parallel()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := New(st.DB)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		jobID := fmt.Sprintf("job-list-%s-%d", time.Now().Format("150405.000000000"), i)
		now := time.Now().UnixMilli()
		if _, err := st.DB.ExecContext(ctx, `
INSERT INTO jobs(
  id, created_at, updated_at, queued_at,
  title, requested_action, target_adapter, initiating_source,
  execution_boundary, risk_class, status, approval_status, write_intent, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			jobID, now, now, now,
			"approval test", "gateway.action", "forge", "test",
			"command_execution", "safe_write", "awaiting_approval", "pending", 1, "{}",
		); err != nil {
			t.Fatalf("insert job: %v", err)
		}
		req, err := svc.OpenRequestForJob(ctx, jobID, CreateRequestInput{
			JobID:            jobID,
			RequestedAction:  "gateway.action",
			RiskClass:        "medium",
			RequestedAdapter: "gateway",
			WriteIntent:      true,
			ScopeSnapshot:    map[string]any{"paths": []string{"scratch"}},
			RequestSummary:   "test request",
		})
		if err != nil {
			t.Fatalf("open request: %v", err)
		}
		if _, err := svc.Decide(ctx, req.ID, "operator", "approved", "ok"); err != nil {
			t.Fatalf("decide: %v", err)
		}
	}

	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	rows, err := svc.ListRequests(callCtx, "resolved", 20)
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected resolved requests")
	}
	for _, r := range rows {
		if r.Decision == nil {
			t.Fatalf("expected decision for request %d", r.ID)
		}
	}
}
