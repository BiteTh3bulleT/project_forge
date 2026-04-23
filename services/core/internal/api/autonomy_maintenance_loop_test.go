package api

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/events"
	"forge/projectforge/services/core/internal/store"
)

func TestAutonomyDreamLoopRunsMaintenanceAndImprovementOnlyWhenIdle(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := int64(1_800_000_000_000)
	nowFn := func() int64 { return now }
	maintenanceRuns := 0
	improvementRuns := 0
	scope := domain.ForgeScope{WorkspaceID: "ws-dream-loop", LaneID: "control.semantic"}

	loop := NewAutonomyMaintenanceLoop(AutonomyMaintenanceLoopOptions{
		DB:                  st.DB,
		Events:              events.New(st.DB),
		Scope:               scope,
		NowMillis:           nowFn,
		IdleAfter:           2 * time.Minute,
		MaintenanceCooldown: 15 * time.Minute,
		ImprovementCooldown: 15 * time.Minute,
		RunMaintenance: func(context.Context, string) error {
			maintenanceRuns++
			return nil
		},
		RunImprovement: func(context.Context, string) error {
			improvementRuns++
			return nil
		},
	})

	if err := loop.Tick(context.Background()); err != nil {
		t.Fatalf("tick idle: %v", err)
	}
	if maintenanceRuns != 1 || improvementRuns != 1 {
		t.Fatalf("expected first idle tick to run both passes once, got maintenance=%d improvement=%d", maintenanceRuns, improvementRuns)
	}
	if status := loop.Status(); !status.Active {
		t.Fatalf("expected dream state active after idle tick")
	}

	if err := loop.Tick(context.Background()); err != nil {
		t.Fatalf("second tick idle: %v", err)
	}
	if maintenanceRuns != 1 || improvementRuns != 1 {
		t.Fatalf("expected cooldown to prevent duplicate pass, got maintenance=%d improvement=%d", maintenanceRuns, improvementRuns)
	}

	threadID, err := insertChatThread(st, now)
	if err != nil {
		t.Fatalf("insert thread: %v", err)
	}
	if err := insertUserMessage(st, threadID, now, "recent task"); err != nil {
		t.Fatalf("insert user message: %v", err)
	}
	if err := loop.Tick(context.Background()); err != nil {
		t.Fatalf("tick with recent user activity: %v", err)
	}
	status := loop.Status()
	if status.Active {
		t.Fatalf("expected dream state exit when recent user activity exists")
	}
}

func TestAutonomyDreamLoopBusyJobBlocksEntry(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := int64(1_800_100_000_000)
	nowFn := func() int64 { return now }
	maintenanceRuns := 0
	improvementRuns := 0

	if err := insertJob(st, "job-running-1", "running", now); err != nil {
		t.Fatalf("insert running job: %v", err)
	}

	loop := NewAutonomyMaintenanceLoop(AutonomyMaintenanceLoopOptions{
		DB:                  st.DB,
		Events:              events.New(st.DB),
		Scope:               domain.ForgeScope{WorkspaceID: "ws-dream-loop", LaneID: "control.semantic"},
		NowMillis:           nowFn,
		IdleAfter:           1 * time.Minute,
		MaintenanceCooldown: 5 * time.Minute,
		ImprovementCooldown: 5 * time.Minute,
		RunMaintenance: func(context.Context, string) error {
			maintenanceRuns++
			return nil
		},
		RunImprovement: func(context.Context, string) error {
			improvementRuns++
			return nil
		},
	})

	if err := loop.Tick(context.Background()); err != nil {
		t.Fatalf("tick with running job: %v", err)
	}
	if maintenanceRuns != 0 || improvementRuns != 0 {
		t.Fatalf("expected no dream passes while job is running; got maintenance=%d improvement=%d", maintenanceRuns, improvementRuns)
	}
	if status := loop.Status(); status.Active {
		t.Fatalf("expected dream state inactive while job is running")
	}
}

func TestAutonomyDreamLoopMaintenanceRunsMoreFrequentlyThanImprovement(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := int64(1_800_200_000_000)
	nowFn := func() int64 { return now }
	maintenanceRuns := 0
	improvementRuns := 0

	loop := NewAutonomyMaintenanceLoop(AutonomyMaintenanceLoopOptions{
		DB:                  st.DB,
		Events:              events.New(st.DB),
		Scope:               domain.ForgeScope{WorkspaceID: "ws-dream-loop", LaneID: "control.semantic"},
		NowMillis:           nowFn,
		IdleAfter:           1 * time.Minute,
		MaintenanceCooldown: 1 * time.Minute,
		ImprovementCooldown: 5 * time.Minute,
		RunMaintenance: func(context.Context, string) error {
			maintenanceRuns++
			return nil
		},
		RunImprovement: func(context.Context, string) error {
			improvementRuns++
			return nil
		},
	})

	if err := loop.Tick(context.Background()); err != nil {
		t.Fatalf("first idle tick: %v", err)
	}
	now += int64((2 * time.Minute) / time.Millisecond)
	if err := loop.Tick(context.Background()); err != nil {
		t.Fatalf("second idle tick: %v", err)
	}
	if maintenanceRuns != 2 {
		t.Fatalf("expected maintenance loop to run twice, got %d", maintenanceRuns)
	}
	if improvementRuns != 1 {
		t.Fatalf("expected improvement loop to run once due longer cooldown, got %d", improvementRuns)
	}
}

func TestAutonomyMaintenanceSweepDryRunNoCommitProducesDeterministicReport(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := int64(1_900_200_000_000)
	if err := insertRepairCandidateObservation(st, now); err != nil {
		t.Fatalf("insert repair candidate: %v", err)
	}
	loop := buildOperationalAutonomyLoop(t, st, now)

	report, err := loop.RunSweep(context.Background(), AutonomyMaintenanceSweepRequest{
		DryRun: true,
		Reason: "operator_diagnostic",
	})
	if err != nil {
		t.Fatalf("run dry-run sweep: %v", err)
	}
	if report.Status != "completed" {
		t.Fatalf("expected completed dry-run report, got %+v", report)
	}
	if !report.DryRun {
		t.Fatalf("expected dry-run report")
	}
	if len(report.Improvement.Actions) != 2 {
		t.Fatalf("expected deterministic improvement action preview, got %+v", report.Improvement.Actions)
	}
	if len(report.Maintenance.Actions) == 0 {
		t.Fatalf("expected maintenance repair preview actions, got %+v", report.Maintenance)
	}
	if got := report.Maintenance.Actions[0].Metadata["status"]; got != "repaired" {
		t.Fatalf("expected repair preview status=repaired, got %#v", got)
	}

	assertSettingsPrefixCount(t, st, "autonomy_repo.intent.", 0)
	assertSettingsPrefixCount(t, st, "autonomy_repo.decision.", 0)
	assertTableCount(t, st, "memory_notes", 0)
}

func TestAutonomyMaintenanceSweepRejectsConcurrentRunWithDiagnostic(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	loop := NewAutonomyMaintenanceLoop(AutonomyMaintenanceLoopOptions{
		DB:     st.DB,
		Events: events.New(st.DB),
		Scope:  domain.ForgeScope{WorkspaceID: "ws-dream-loop", LaneID: "control.semantic"},
	})
	loop.mu.Lock()
	loop.sweepRunning = true
	loop.activeSweepID = "sweep-active"
	loop.mu.Unlock()

	report, err := loop.RunSweep(context.Background(), AutonomyMaintenanceSweepRequest{DryRun: true, Reason: "operator_diagnostic"})
	if err != nil {
		t.Fatalf("run skipped sweep: %v", err)
	}
	if report.Status != "skipped" {
		t.Fatalf("expected skipped report, got %+v", report)
	}
	if len(report.Diagnostics) == 0 || report.Diagnostics[0].Code != "SWEEP_IN_PROGRESS" {
		t.Fatalf("expected in-progress diagnostic, got %+v", report.Diagnostics)
	}
	assertEventCountByType(t, st, "autonomy.dream_loop.sweep_skipped", 1)
}

func insertJob(st *store.Store, id, status string, now int64) error {
	_, err := st.DB.Exec(`
INSERT INTO jobs(
	id, created_at, updated_at, title, requested_action, target_adapter, initiating_source,
	execution_boundary, risk_class, status, approval_status, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		now,
		now,
		"test job",
		"test action",
		"test_adapter",
		"test",
		"execution_boundary",
		"read_only",
		status,
		"not_required",
		"{}",
	)
	return err
}

func insertChatThread(st *store.Store, now int64) (int64, error) {
	res, err := st.DB.Exec(`INSERT INTO chat_threads(title, created_at, updated_at, dossier_id) VALUES(?,?,?,NULL)`, "Conversation", now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func insertUserMessage(st *store.Store, threadID int64, now int64, content string) error {
	_, err := st.DB.Exec(`INSERT INTO chat_messages(thread_id, role, content, created_at, metadata_json) VALUES(?,?,?,?,?)`, threadID, "user", fmt.Sprintf("%s", content), now, "{}")
	return err
}

func buildOperationalAutonomyLoop(t *testing.T, st *store.Store, now int64) *AutonomyMaintenanceLoop {
	t.Helper()
	cfg := config.Config{
		DataDir:      t.TempDir(),
		WorkspaceDir: filepath.Join(t.TempDir(), "workspace"),
	}
	loop := newDefaultAutonomyMaintenanceLoop(st.DB, cfg, events.New(st.DB), nil)
	if loop == nil {
		t.Fatalf("expected default autonomy loop")
	}
	loop.nowMillis = func() int64 { return now }
	return loop
}

func insertRepairCandidateObservation(st *store.Store, now int64) error {
	_, err := st.DB.Exec(`
INSERT INTO memory_observations(
	created_at, updated_at, observed_at, type, raw_content, summary, embedding_ref, dossier_id, project_key, source_path,
	entities_json, tags_json, related_files_json, task_type, confidence, verification_state, lineage_json, origin_kind, origin_id,
	stale, last_verified_at, usefulness_score, usefulness_count, noise_count
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		now,
		now,
		now,
		"note",
		"repair preview content that should force a deterministic summary refresh",
		"",
		"",
		nil,
		"",
		"/tmp/example.txt",
		"[]",
		"[]",
		"[]",
		"",
		0.5,
		"unknown",
		"[]",
		"",
		"",
		1,
		nil,
		-0.7,
		0,
		1,
	)
	return err
}

func assertSettingsPrefixCount(t *testing.T, st *store.Store, prefix string, want int) {
	t.Helper()
	var got int
	if err := st.DB.QueryRow(`SELECT COUNT(1) FROM settings WHERE key LIKE ?`, prefix+"%").Scan(&got); err != nil {
		t.Fatalf("count settings prefix %q: %v", prefix, err)
	}
	if got != want {
		t.Fatalf("settings count for prefix %q = %d want %d", prefix, got, want)
	}
}

func assertTableCount(t *testing.T, st *store.Store, table string, want int) {
	t.Helper()
	var got int
	if err := st.DB.QueryRow(fmt.Sprintf(`SELECT COUNT(1) FROM %s`, table)).Scan(&got); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("row count in %s = %d want %d", table, got, want)
	}
}

func assertEventCountByType(t *testing.T, st *store.Store, typ string, want int) {
	t.Helper()
	var got int
	if err := st.DB.QueryRow(`SELECT COUNT(1) FROM events WHERE type = ?`, typ).Scan(&got); err != nil {
		t.Fatalf("count events %q: %v", typ, err)
	}
	if got != want {
		t.Fatalf("event count for %q = %d want %d", typ, got, want)
	}
}
