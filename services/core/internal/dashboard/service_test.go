package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestSummaryRollsUpOperatorDashboardData(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	dossierID := insertDossier(t, svc.db, "Dashboard dossier")
	insertJob(t, svc.db, "active-queued", "Queued", "queued", "codex", 100)
	insertJob(t, svc.db, "active-running", "Running", "running", "ollama", 200)
	insertJob(t, svc.db, "failed-1", "Failed 1", "failed", "codex", 300)
	insertJob(t, svc.db, "failed-2", "Failed 2", "failed", "codex", 301)
	insertJob(t, svc.db, "failed-3", "Failed 3", "failed", "codex", 302)
	insertApprovalRequest(t, svc.db, "active-queued")
	insertPendingReview(t, svc.db, dossierID)
	linkJobsToDossier(t, svc.db, dossierID, []string{"failed-1", "failed-2", "failed-3"})
	insertImport(t, svc.db, "codex", "Imported run")
	ruleID := insertAutomationRule(t, svc.db, "Review import")
	insertAutomationHistory(t, svc.db, ruleID, "preview_ready", "rule matched")
	insertStrategy(t, svc.db, "strategy-1")
	insertRecommendation(t, svc.db, dossierID, "analysis.safe_local", "codex")

	summary, err := svc.Summary(ctx)
	if err != nil {
		t.Fatalf("Summary failed: %v", err)
	}
	if len(summary.ActiveJobs) != 2 || summary.ActiveJobs[0].ID != "active-running" || summary.ActiveJobs[1].ID != "active-queued" {
		t.Fatalf("active jobs = %+v, want running then queued", summary.ActiveJobs)
	}
	if summary.ApprovalsPending != 1 || summary.ReviewsPending != 1 {
		t.Fatalf("pending counts approvals=%d reviews=%d, want 1/1", summary.ApprovalsPending, summary.ReviewsPending)
	}
	if len(summary.RecentFailures) != 3 || summary.RecentFailures[0].ID != "failed-3" {
		t.Fatalf("recent failures = %+v, want latest failed first", summary.RecentFailures)
	}
	if len(summary.RecentImports) != 1 || summary.RecentImports[0].Summary != "Imported run" {
		t.Fatalf("recent imports = %+v, want imported run", summary.RecentImports)
	}
	if len(summary.DossierHealth) != 1 {
		t.Fatalf("dossier health rows = %+v, want one", summary.DossierHealth)
	}
	health := summary.DossierHealth[0]
	if health["health"] != "review_pending" || health["failureCount"] != 3 || health["reviewPending"] != 1 {
		t.Fatalf("dossier health = %+v, want review_pending with failures/review", health)
	}
	if len(summary.AutomationActivity) != 1 || summary.AutomationActivity[0].RuleID == nil || *summary.AutomationActivity[0].RuleID != ruleID {
		t.Fatalf("automation activity = %+v, want rule activity", summary.AutomationActivity)
	}
	if len(summary.RoutingRecommendations) != 1 || !json.Valid(summary.RoutingRecommendations[0].Reasons) {
		t.Fatalf("routing recommendations = %+v, want valid recommendation", summary.RoutingRecommendations)
	}
	if summary.SystemStatus["activeJobs"] != 2 || summary.SystemStatus["approvalsPending"] != 1 || summary.SystemStatus["reviewsPending"] != 1 {
		t.Fatalf("system status = %+v, want active/pending counts", summary.SystemStatus)
	}
	if summary.SystemStatus["dossierCount"] != 1 || summary.SystemStatus["strategyCount"] != 1 || summary.SystemStatus["automationRules"] != 1 {
		t.Fatalf("system status inventory = %+v, want dossier/strategy/rule counts", summary.SystemStatus)
	}
}

func TestSummaryOnEmptyStoreReturnsEmptyCollections(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	summary, err := svc.Summary(ctx)
	if err != nil {
		t.Fatalf("Summary empty failed: %v", err)
	}
	if len(summary.ActiveJobs) != 0 || len(summary.RecentFailures) != 0 || len(summary.DossierHealth) != 0 {
		t.Fatalf("empty summary has populated collections: %+v", summary)
	}
	if summary.SystemStatus["activeJobs"] != 0 || summary.SystemStatus["dossierCount"] != 0 {
		t.Fatalf("empty system status = %+v, want zero counts", summary.SystemStatus)
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

func insertDossier(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	now := time.Now().UnixMilli()
	res, err := db.Exec(`INSERT INTO dossiers(created_at, updated_at, name, description) VALUES(?,?,?,?)`, now, now, name, "")
	if err != nil {
		t.Fatalf("insert dossier %q: %v", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("dossier last insert id: %v", err)
	}
	return id
}

func insertJob(t *testing.T, db *sql.DB, id, title, status, adapter string, createdAt int64) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO jobs(
  id, created_at, updated_at, queued_at, title, requested_action, target_adapter,
  initiating_source, execution_boundary, risk_class, status, approval_status,
  write_intent, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		createdAt,
		createdAt,
		createdAt,
		title,
		"test.action",
		adapter,
		"test",
		"test",
		"read_only",
		status,
		"not_required",
		0,
		`{"templateId":"search_packet"}`,
	)
	if err != nil {
		t.Fatalf("insert job %q: %v", id, err)
	}
}

func insertApprovalRequest(t *testing.T, db *sql.DB, jobID string) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO approval_requests(created_at, job_id, status, requested_action, risk_class, requested_adapter, write_intent, scope_snapshot_json, request_summary)
VALUES(?,?,?,?,?,?,?,?,?)`, time.Now().UnixMilli(), jobID, "pending", "test.action", "write_files", "codex", 1, `{}`, "approval")
	if err != nil {
		t.Fatalf("insert approval request: %v", err)
	}
}

func insertPendingReview(t *testing.T, db *sql.DB, dossierID int64) {
	t.Helper()
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
INSERT INTO review_records(created_at, updated_at, target_type, target_id, dossier_id, status, summary, notes, annotations_json, reviewer)
VALUES(?,?,?,?,?,?,?,?,?,?)`, now, now, "job", "failed-1", dossierID, "pending", "review", "", "[]", "operator")
	if err != nil {
		t.Fatalf("insert review: %v", err)
	}
}

func linkJobsToDossier(t *testing.T, db *sql.DB, dossierID int64, jobIDs []string) {
	t.Helper()
	now := time.Now().UnixMilli()
	for _, jobID := range jobIDs {
		if _, err := db.Exec(`INSERT INTO dossier_jobs(dossier_id, job_id, linked_at) VALUES(?,?,?)`, dossierID, jobID, now); err != nil {
			t.Fatalf("link job %q to dossier: %v", jobID, err)
		}
	}
}

func insertImport(t *testing.T, db *sql.DB, adapterID, summary string) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO imported_executions(created_at, adapter_id, external_run_id, summary, output_refs_json, evaluation_json)
VALUES(?,?,?,?,?,?)`, time.Now().UnixMilli(), adapterID, "external-1", summary, "[]", "{}")
	if err != nil {
		t.Fatalf("insert import: %v", err)
	}
}

func insertAutomationRule(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	now := time.Now().UnixMilli()
	res, err := db.Exec(`
INSERT INTO automation_rules(created_at, updated_at, name, trigger, condition_json, action_json, scope_json, enabled, dry_run_default)
VALUES(?,?,?,?,?,?,?,?,?)`, now, now, name, "import.execution.created", `{"always":true}`, `{"type":"create_review"}`, `{}`, 1, 1)
	if err != nil {
		t.Fatalf("insert automation rule: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("automation rule last insert id: %v", err)
	}
	return id
}

func insertAutomationHistory(t *testing.T, db *sql.DB, ruleID int64, status, message string) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO automation_history(created_at, rule_id, trigger, matched, dry_run, status, message, preview_json, result_json)
VALUES(?,?,?,?,?,?,?,?,?)`, time.Now().UnixMilli(), ruleID, "import.execution.created", 1, 1, status, message, `{}`, `{}`)
	if err != nil {
		t.Fatalf("insert automation history: %v", err)
	}
}

func insertStrategy(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
INSERT INTO execution_strategies(created_at, updated_at, id, name, task_type, target_adapter, retrieval_mode, packet_rules_json, expected_artifacts_json, success_criteria_json, retry_guidance_json, enabled)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, now, now, id, id, "analysis.safe_local", "codex", "hybrid", "{}", "[]", "{}", "{}", 1)
	if err != nil {
		t.Fatalf("insert strategy: %v", err)
	}
}

func insertRecommendation(t *testing.T, db *sql.DB, dossierID int64, taskType, adapter string) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO routing_policy_recommendations(created_at, dossier_id, task_type, target_adapter, retrieval_mode, packet_shape_json, approval_required, confidence, reasons_json, evidence_json)
VALUES(?,?,?,?,?,?,?,?,?,?)`, time.Now().UnixMilli(), dossierID, taskType, adapter, "hybrid", "{}", 1, 0.8, `["stable"]`, "{}")
	if err != nil {
		t.Fatalf("insert recommendation: %v", err)
	}
}
