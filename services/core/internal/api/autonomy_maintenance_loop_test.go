package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
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
