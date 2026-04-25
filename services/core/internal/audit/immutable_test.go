package audit

import (
	"context"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

// TestAuditRecordsAreAppendOnly confirms the SQLite triggers installed by the
// core migration block UPDATE and DELETE on audit_records. The guarantee is
// that a compromised writer cannot rewrite history after the fact.
func TestAuditRecordsAreAppendOnly(t *testing.T) {
	t.Parallel()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	svc := New(st.DB)
	rec, err := svc.Record(context.Background(), CreateRequest{
		CorrelationID: "corr-append-only",
		Category:      "gateway",
		Action:        "tool.executed",
		Actor:         "tester",
		Outcome:       "ok",
		Summary:       "baseline",
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	// Attempting to mutate the row must fail at trigger level.
	if _, err := st.DB.ExecContext(context.Background(),
		`UPDATE audit_records SET summary = ? WHERE id = ?`,
		"tampered", rec.ID,
	); err == nil {
		t.Fatalf("expected UPDATE on audit_records to be rejected by trigger")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("expected append-only error, got %v", err)
	}

	if _, err := st.DB.ExecContext(context.Background(),
		`DELETE FROM audit_records WHERE id = ?`, rec.ID,
	); err == nil {
		t.Fatalf("expected DELETE on audit_records to be rejected by trigger")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("expected append-only error, got %v", err)
	}

	// Original row should still be intact.
	fetched, err := svc.Get(context.Background(), rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fetched.Summary != "baseline" {
		t.Fatalf("expected summary to be preserved, got %q", fetched.Summary)
	}
}
