package lineage

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestLinkForJobAndUpsertRelation(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	insertLineageJob(t, svc.db, "parent-a", "Parent A", "succeeded", 100)
	insertLineageJob(t, svc.db, "target", "Target", "running", 200)
	insertLineageJob(t, svc.db, "child-a", "Child A", "failed", 300)

	parent, err := svc.Link(ctx, " parent-a ", " target ", "", map[string]any{"reason": "retry source"})
	if err != nil {
		t.Fatalf("Link parent failed: %v", err)
	}
	if parent.RelationType != "retry" {
		t.Fatalf("default relation type = %q, want retry", parent.RelationType)
	}
	child, err := svc.Link(ctx, "target", "child-a", "  Supersedes  ", map[string]any{"delta": "new attempt"})
	if err != nil {
		t.Fatalf("Link child failed: %v", err)
	}
	if child.RelationType != "supersedes" {
		t.Fatalf("normalized relation type = %q, want supersedes", child.RelationType)
	}

	updated, err := svc.Link(ctx, "target", "child-a", "derived", map[string]any{"delta": "updated"})
	if err != nil {
		t.Fatalf("Link upsert failed: %v", err)
	}
	if updated.ID != child.ID {
		t.Fatalf("upsert id = %d, want existing id %d", updated.ID, child.ID)
	}
	var summary map[string]any
	if err := json.Unmarshal(updated.ChangeSummary, &summary); err != nil {
		t.Fatalf("change summary json invalid: %v", err)
	}
	if summary["delta"] != "updated" {
		t.Fatalf("change summary = %v, want updated delta", summary)
	}

	lineage, err := svc.ForJob(ctx, " target ")
	if err != nil {
		t.Fatalf("ForJob failed: %v", err)
	}
	if len(lineage.Parents) != 1 || lineage.Parents[0].ParentJobID != "parent-a" {
		t.Fatalf("parents = %+v, want parent-a", lineage.Parents)
	}
	if len(lineage.Children) != 1 || lineage.Children[0].ChildJobID != "child-a" || lineage.Children[0].RelationType != "derived" {
		t.Fatalf("children = %+v, want updated child-a relation", lineage.Children)
	}
	if got := jobSummaryIDs(lineage.Related); !reflect.DeepEqual(got, []string{"child-a", "parent-a"}) {
		t.Fatalf("related ids = %v, want created_at desc child-a,parent-a", got)
	}
	if lineage.Related[0].LastFailureCode == nil || *lineage.Related[0].LastFailureCode != "execution_failure" {
		t.Fatalf("child failure code = %v, want execution_failure", lineage.Related[0].LastFailureCode)
	}
	if lineage.Related[1].ResultSummary == nil || *lineage.Related[1].ResultSummary != "parent done" {
		t.Fatalf("parent result summary = %v, want parent done", lineage.Related[1].ResultSummary)
	}
}

func TestLinkRejectsInvalidInputs(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	insertLineageJob(t, svc.db, "parent", "Parent", "queued", 100)
	insertLineageJob(t, svc.db, "child", "Child", "queued", 200)

	if _, err := svc.Link(ctx, "", "child", "retry", nil); err == nil {
		t.Fatalf("Link accepted blank parent id")
	}
	if _, err := svc.ForJob(ctx, " "); err == nil {
		t.Fatalf("ForJob accepted blank job id")
	}
	if _, err := svc.Link(ctx, "parent", "missing", "retry", nil); err == nil {
		t.Fatalf("Link accepted missing child job")
	}
	if _, err := svc.Link(ctx, "parent", "child", "retry", map[string]any{"bad": func() {}}); err == nil {
		t.Fatalf("Link accepted unsupported change summary")
	}
}

func TestForJobWithNoRelationsReturnsEmptySlices(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	insertLineageJob(t, svc.db, "lonely", "Lonely", "queued", 100)

	lineage, err := svc.ForJob(ctx, "lonely")
	if err != nil {
		t.Fatalf("ForJob lonely failed: %v", err)
	}
	if len(lineage.Parents) != 0 || len(lineage.Children) != 0 || len(lineage.Related) != 0 {
		t.Fatalf("empty lineage = %+v, want no parents/children/related", lineage)
	}
}

func newTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return New(st.DB), func() {
		if err := st.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}
}

func insertLineageJob(t *testing.T, db *sql.DB, id, title, status string, createdAt int64) {
	t.Helper()
	now := time.Now().UnixMilli()
	resultSummary := sql.NullString{}
	lastFailure := sql.NullString{}
	if status == "succeeded" {
		resultSummary = sql.NullString{String: "parent done", Valid: true}
	}
	if status == "failed" {
		lastFailure = sql.NullString{String: "execution_failure", Valid: true}
	}
	_, err := db.Exec(`
INSERT INTO jobs(
  id, created_at, updated_at, queued_at, title, requested_action, target_adapter,
  initiating_source, execution_boundary, risk_class, status, approval_status,
  write_intent, result_summary, last_failure_code, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		createdAt,
		now,
		createdAt,
		title,
		"test.action",
		"test-adapter",
		"test",
		"test",
		"read_only",
		status,
		"not_required",
		0,
		nullableString(resultSummary),
		nullableString(lastFailure),
		`{"templateId":"search_packet"}`,
	)
	if err != nil {
		t.Fatalf("insert lineage job %s: %v", id, err)
	}
}

func nullableString(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}

func jobSummaryIDs(jobs []JobSummary) []string {
	out := make([]string, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, job.ID)
	}
	return out
}
