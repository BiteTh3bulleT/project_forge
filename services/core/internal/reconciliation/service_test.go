package reconciliation

import (
	"context"
	"encoding/json"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestSaveUpsertsAndNormalizesReconciliation(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	importID := seedImportedExecution(t, st)
	svc := New(st.DB)

	first, err := svc.Save(ctx, SaveRequest{
		ImportID:           importID,
		ChangedFiles:       []string{" src/main.go ", "", "docs/report.md"},
		FailureReasons:     nil,
		UnresolvedIssues:   []string{" needs review "},
		SuggestedNextSteps: []string{" rerun tests "},
		AgentNotes:         " notes ",
		PatchSummary:       " patch ",
	})
	if err != nil {
		t.Fatalf("save first reconciliation: %v", err)
	}
	if first.ReviewStatus != "pending" {
		t.Fatalf("default review status = %q, want pending", first.ReviewStatus)
	}
	assertJSONStrings(t, first.ChangedFiles, []string{"src/main.go", "docs/report.md"})
	assertJSONStrings(t, first.FailureReasons, []string{})
	assertJSONStrings(t, first.UnresolvedIssues, []string{"needs review"})
	assertJSONStrings(t, first.SuggestedNextSteps, []string{"rerun tests"})
	if first.AgentNotes != "notes" || first.PatchSummary != "patch" {
		t.Fatalf("trimmed text mismatch: notes=%q patch=%q", first.AgentNotes, first.PatchSummary)
	}

	second, err := svc.Save(ctx, SaveRequest{
		ImportID:     importID,
		ChangedFiles: []string{"src/updated.go"},
		ReviewStatus: " APPROVED ",
	})
	if err != nil {
		t.Fatalf("save updated reconciliation: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("Save should upsert by import id: first=%d second=%d", first.ID, second.ID)
	}
	if second.ReviewStatus != "approved" {
		t.Fatalf("normalized review status = %q, want approved", second.ReviewStatus)
	}
	assertJSONStrings(t, second.ChangedFiles, []string{"src/updated.go"})

	filtered, err := svc.List(ctx, 10, "APPROVED")
	if err != nil {
		t.Fatalf("list approved reconciliations: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != first.ID {
		t.Fatalf("filtered reconciliations = %#v, want one updated record", filtered)
	}
}

func seedImportedExecution(t *testing.T, st *store.Store) int64 {
	t.Helper()
	res, err := st.DB.Exec(`
INSERT INTO imported_executions(created_at, adapter_id, external_run_id, summary)
VALUES(?,?,?,?)`, int64(1000), "test-adapter", "external-1", "seed import")
	if err != nil {
		t.Fatalf("seed imported execution: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("import id: %v", err)
	}
	return id
}

func assertJSONStrings(t *testing.T, raw json.RawMessage, want []string) {
	t.Helper()
	var got []string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", string(raw), err)
	}
	if len(got) != len(want) {
		t.Fatalf("json strings = %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("json strings = %#v, want %#v", got, want)
		}
	}
}
