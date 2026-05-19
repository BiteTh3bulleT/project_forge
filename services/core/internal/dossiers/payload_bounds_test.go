package dossiers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestMarshalDossierPayloadRejectsOversizeList(t *testing.T) {
	if body, err := marshalDossierPayload("constraints", []string{"local only"}); err != nil || !json.Valid(body) {
		t.Fatalf("expected valid dossier payload, got len=%d err=%v", len(body), err)
	}
	if _, err := marshalDossierPayload("importantFiles", []string{strings.Repeat("x", maxDossierPayloadBytes+1)}); !errors.Is(err, errDossierPayloadTooLarge) {
		t.Fatalf("expected payload size rejection, got %v", err)
	}
}

func TestCreateRejectsOversizeDossierPayloadBeforeInsert(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.Create(ctx, CreateRequest{
		Name:        "Oversize dossier",
		Description: "must not persist",
		Constraints: []string{
			strings.Repeat("x", maxDossierPayloadBytes+1),
		},
	})
	if !errors.Is(err, errDossierPayloadTooLarge) {
		t.Fatalf("Create error=%v, want errDossierPayloadTooLarge", err)
	}
	var count int
	if err := svc.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM dossiers WHERE name = ?`, "Oversize dossier").Scan(&count); err != nil {
		t.Fatalf("count dossiers failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("oversize dossier rows = %d, want 0", count)
	}
}

func TestUpdateRejectsOversizeDossierPayloadPreservesExistingRecord(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	dossier, err := svc.Create(ctx, CreateRequest{
		Name:           "Safe dossier",
		ImportantFiles: []string{"docs/plan.md"},
		Constraints:    []string{"local only"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	oversize := []string{strings.Repeat("x", maxDossierPayloadBytes+1)}
	if _, err := svc.Update(ctx, dossier.ID, UpdateRequest{ImportantFiles: &oversize}); !errors.Is(err, errDossierPayloadTooLarge) {
		t.Fatalf("Update error=%v, want errDossierPayloadTooLarge", err)
	}
	after, err := svc.Get(ctx, dossier.ID)
	if err != nil {
		t.Fatalf("Get after failed update failed: %v", err)
	}
	var important []string
	if err := json.Unmarshal(after.ImportantFiles, &important); err != nil {
		t.Fatalf("important files json invalid: %v", err)
	}
	if len(important) != 1 || important[0] != "docs/plan.md" {
		t.Fatalf("important files after rejected update = %v, want original value", important)
	}
	var constraints []string
	if err := json.Unmarshal(after.Constraints, &constraints); err != nil {
		t.Fatalf("constraints json invalid: %v", err)
	}
	if len(constraints) != 1 || constraints[0] != "local only" {
		t.Fatalf("constraints after rejected update = %v, want original value", constraints)
	}
}

func TestDossierLifecycleLinksJobsAndBriefs(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	sourceA := insertSource(t, svc.db, "docs")
	sourceB := insertSource(t, svc.db, "src")
	insertJob(t, svc.db, "job-1", "Build plan", "succeeded")

	dossier, err := svc.Create(ctx, CreateRequest{
		Name:              "  Operator context  ",
		Description:       "  active workspace  ",
		SourceIDs:         []int64{sourceA, sourceB},
		PrimaryPaths:      []string{" docs ", "docs", " "},
		RelatedRepos:      []string{"project_forge"},
		PreferredAdapters: []string{"codex"},
		RoutingNotes:      "  route locally  ",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if dossier.Name != "Operator context" || dossier.Description != "active workspace" || dossier.RoutingNotes != "route locally" {
		t.Fatalf("trimmed dossier fields = %+v", dossier)
	}
	var primary []string
	if err := json.Unmarshal(dossier.PrimaryPaths, &primary); err != nil {
		t.Fatalf("primary paths json invalid: %v", err)
	}
	if len(primary) != 1 || primary[0] != "docs" {
		t.Fatalf("primary paths = %v, want deduped docs path", primary)
	}
	links, err := svc.SourceLinks(ctx, dossier.ID)
	if err != nil {
		t.Fatalf("SourceLinks failed: %v", err)
	}
	if len(links) != 2 || links[0].Path != "docs" || links[1].Path != "src" {
		t.Fatalf("source links = %+v, want docs/src", links)
	}

	updatedDescription := "updated description"
	updatedPaths := []string{"src"}
	updated, err := svc.Update(ctx, dossier.ID, UpdateRequest{
		Description:  &updatedDescription,
		SourceIDs:    []int64{sourceB},
		PrimaryPaths: &updatedPaths,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Description != "updated description" {
		t.Fatalf("updated description = %q", updated.Description)
	}
	links, err = svc.SourceLinks(ctx, dossier.ID)
	if err != nil {
		t.Fatalf("SourceLinks after update failed: %v", err)
	}
	if len(links) != 1 || links[0].Path != "src" {
		t.Fatalf("source links after update = %+v, want only src", links)
	}

	if err := svc.AttachJob(ctx, dossier.ID, "job-1"); err != nil {
		t.Fatalf("AttachJob failed: %v", err)
	}
	brief, err := svc.GenerateBrief(ctx, dossier.ID, "  operator note  ")
	if err != nil {
		t.Fatalf("GenerateBrief failed: %v", err)
	}
	if !strings.Contains(brief.SummaryMarkdown, "Dossier: Operator context") || !strings.Contains(brief.SummaryMarkdown, "job-1") {
		t.Fatalf("brief summary missing dossier/job context:\n%s", brief.SummaryMarkdown)
	}
	if brief.Notes != "operator note" {
		t.Fatalf("brief notes = %q, want trimmed operator note", brief.Notes)
	}
	detail, err := svc.Detail(ctx, dossier.ID)
	if err != nil {
		t.Fatalf("Detail failed: %v", err)
	}
	if len(detail.RecentJobs) != 1 || detail.RecentJobs[0].JobID != "job-1" {
		t.Fatalf("detail recent jobs = %+v, want job-1", detail.RecentJobs)
	}
	if len(detail.Briefs) != 1 || detail.Briefs[0].ID != brief.ID {
		t.Fatalf("detail briefs = %+v, want generated brief", detail.Briefs)
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

func insertSource(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO sources(path, created_at) VALUES(?, ?)`, path, timeNowMs())
	if err != nil {
		t.Fatalf("insert source %q: %v", path, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("source last insert id: %v", err)
	}
	return id
}

func insertJob(t *testing.T, db *sql.DB, id, title, status string) {
	t.Helper()
	now := timeNowMs()
	_, err := db.Exec(`
INSERT INTO jobs(
  id, created_at, updated_at, queued_at, title, requested_action, target_adapter,
  initiating_source, execution_boundary, risk_class, status, approval_status,
  write_intent, result_summary, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		now,
		now,
		now,
		title,
		"test.action",
		"test-adapter",
		"test",
		"test",
		"read_only",
		status,
		"not_required",
		0,
		"job done",
		`{"templateId":"search_packet"}`,
	)
	if err != nil {
		t.Fatalf("insert job %q: %v", id, err)
	}
}

func timeNowMs() int64 {
	return time.Now().UnixMilli()
}
