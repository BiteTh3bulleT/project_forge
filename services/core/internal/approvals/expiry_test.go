package approvals

import (
	"context"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

// TestExpireTransitionsPendingRequests verifies the reaper path: pending rows
// whose expires_at has elapsed get moved to status="expired" and any waiters
// blocked on them are released.
func TestExpireTransitionsPendingRequests(t *testing.T) {
	t.Parallel()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	// Need a job row to satisfy the FK on approval_requests.
	if _, err := st.DB.ExecContext(context.Background(), `
INSERT INTO jobs(id, created_at, updated_at, title, requested_action, target_adapter,
 initiating_source, execution_boundary, risk_class, status, approval_status)
VALUES('job-expiry-1', 0, 0, 'test', 'noop', 'test', 'user', 'reasoning', 'low', 'created', 'pending')`,
	); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	svc := New(st.DB)
	req, err := svc.OpenRequestForJob(context.Background(), "job-expiry-1", CreateRequestInput{
		JobID:            "job-expiry-1",
		RequestedAction:  "filesystem.write",
		RiskClass:        "high",
		RequestedAdapter: "gateway",
		WriteIntent:      true,
		RequestSummary:   "test expiry",
		TTL:              50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if req.Status != "pending" {
		t.Fatalf("expected pending, got %q", req.Status)
	}

	// Wait ~80ms to cross the TTL horizon, then run the reaper.
	time.Sleep(80 * time.Millisecond)
	n, err := svc.Expire(context.Background())
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 expired, got %d", n)
	}

	fresh, err := svc.GetRequest(context.Background(), req.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fresh.Status != "expired" {
		t.Fatalf("expected expired status, got %q", fresh.Status)
	}
	if fresh.ExpiredAtMs == 0 {
		t.Fatalf("expected expired_at stamp to be set")
	}
}

// TestWaitFiresOnDecide ensures Wait channels close promptly when Decide
// records the final choice, so autonomy callers don't have to poll.
func TestWaitFiresOnDecide(t *testing.T) {
	t.Parallel()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if _, err := st.DB.ExecContext(context.Background(), `
INSERT INTO jobs(id, created_at, updated_at, title, requested_action, target_adapter,
 initiating_source, execution_boundary, risk_class, status, approval_status)
VALUES('job-wait', 0, 0, 'test', 'noop', 'test', 'user', 'reasoning', 'low', 'created', 'pending')`,
	); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	svc := New(st.DB)
	req, err := svc.OpenRequestForJob(context.Background(), "job-wait", CreateRequestInput{
		JobID:            "job-wait",
		RequestedAction:  "filesystem.delete",
		RiskClass:        "critical",
		RequestedAdapter: "gateway",
		RequestSummary:   "test wait",
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	ch := svc.Wait(context.Background(), req.ID)

	go func() {
		time.Sleep(20 * time.Millisecond)
		if _, err := svc.Decide(context.Background(), req.ID, "operator", "approved", "looks good"); err != nil {
			t.Errorf("decide: %v", err)
		}
	}()

	select {
	case <-ch:
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("wait channel did not fire within 500ms of decision")
	}
}
