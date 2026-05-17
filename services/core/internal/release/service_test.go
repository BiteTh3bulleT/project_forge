package release

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"forge/projectforge/services/core/internal/permissions"
	"forge/projectforge/services/core/internal/store"
)

func TestCheckReadinessSeedsDefaultLanesForFirstRunStore(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	for _, dir := range []string{
		filepath.Join(dataDir, "backups"),
		filepath.Join(dataDir, "exports"),
		workspaceDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	permSvc := permissions.New(st.DB)
	if err := permSvc.EnsureDefaults(ctx, workspaceDir); err != nil {
		t.Fatalf("ensure permission defaults: %v", err)
	}

	var before int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM action_lanes WHERE enabled = 1`).Scan(&before); err != nil {
		t.Fatalf("count enabled lanes before readiness: %v", err)
	}
	if before != 0 {
		t.Fatalf("test setup expected no enabled lanes before readiness, got %d", before)
	}

	cl, err := New(st.DB, dataDir, workspaceDir).CheckReadiness(ctx)
	if err != nil {
		t.Fatalf("check readiness: %v", err)
	}
	if item := checklistItem(cl, "lanes.enabled"); item == nil || item.Status != "ok" {
		t.Fatalf("expected lanes.enabled readiness ok after first-run default seeding, got %#v", item)
	}

	var after int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM action_lanes WHERE enabled = 1`).Scan(&after); err != nil {
		t.Fatalf("count enabled lanes after readiness: %v", err)
	}
	if after == 0 {
		t.Fatal("expected readiness to seed at least one enabled default lane")
	}
}

func checklistItem(cl *Checklist, id string) *ChecklistItem {
	for i := range cl.Items {
		if cl.Items[i].ID == id {
			return &cl.Items[i]
		}
	}
	return nil
}
