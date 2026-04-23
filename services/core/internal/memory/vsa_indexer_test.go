package memory

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func newMemoryServiceForTest(t *testing.T) (*Service, func()) {
	t.Helper()
	dbStore, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if _, err := dbStore.DB.Exec(`INSERT INTO settings(key, value) VALUES('retrieval_vsa_mode','shadow') ON CONFLICT(key) DO UPDATE SET value=excluded.value`); err != nil {
		t.Fatalf("set vsa mode: %v", err)
	}
	return New(dbStore.DB), func() {
		_ = dbStore.Close()
	}
}

func createTestObservation(t *testing.T, svc *Service, path, summary, raw string) *Observation {
	t.Helper()
	obs, err := svc.RecordObservation(context.Background(), RecordObservationRequest{
		Type:       "memory_note",
		SourcePath: path,
		TaskType:   "test",
		Summary:    summary,
		RawContent: raw,
		Tags:       []string{"test"},
	})
	if err != nil {
		t.Fatalf("record observation: %v", err)
	}
	return obs
}

func TestVSAReindexIdempotentAndStaleDetection(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()

	obs := createTestObservation(t, svc, "/tmp/project/a.go", "alpha summary", "alpha raw content")
	if _, err := svc.db.ExecContext(ctx, `DELETE FROM memory_vsa_pointers WHERE observation_id = ?`, obs.ID); err != nil {
		t.Fatalf("delete pointer row: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `DELETE FROM memory_vsa_role_bindings WHERE observation_id = ?`, obs.ID); err != nil {
		t.Fatalf("delete bindings: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `DELETE FROM memory_vsa_associations WHERE from_observation_id = ? OR to_observation_id = ?`, obs.ID, obs.ID); err != nil {
		t.Fatalf("delete associations: %v", err)
	}

	run1, err := svc.RunVSAReindex(ctx, RunVSAReindexRequest{Limit: 10, Reason: "test_backfill", Force: true})
	if err != nil {
		t.Fatalf("run reindex backfill: %v", err)
	}
	if run1.Run.Indexed != 1 {
		t.Fatalf("expected first run indexed=1, got %d (detail=%+v)", run1.Run.Indexed, run1.Run)
	}

	run2, err := svc.RunVSAReindex(ctx, RunVSAReindexRequest{Limit: 10, Reason: "test_idempotent"})
	if err != nil {
		t.Fatalf("run reindex second pass: %v", err)
	}
	if run2.Run.Skipped < 1 {
		t.Fatalf("expected idempotent second run to skip at least one item, got %+v", run2.Run)
	}

	_, err = svc.db.ExecContext(ctx,
		`UPDATE memory_observations SET summary = ?, updated_at = ? WHERE id = ?`,
		"alpha summary changed without vsa refresh",
		time.Now().UnixMilli(),
		obs.ID,
	)
	if err != nil {
		t.Fatalf("update observation directly: %v", err)
	}

	ids, err := svc.vsaReindexCandidates(ctx, RunVSAReindexRequest{Limit: 10, StaleOnly: true})
	if err != nil {
		t.Fatalf("stale candidate scan: %v", err)
	}
	if len(ids) != 1 || ids[0] != obs.ID {
		t.Fatalf("expected stale-only candidates [%d], got %v", obs.ID, ids)
	}

	run3, err := svc.RunVSAReindex(ctx, RunVSAReindexRequest{Limit: 10, StaleOnly: true, Reason: "test_stale_reindex"})
	if err != nil {
		t.Fatalf("run stale-only reindex: %v", err)
	}
	if run3.Run.Indexed != 1 {
		t.Fatalf("expected stale-only run indexed=1, got %+v", run3.Run)
	}
}

func TestUsefulnessFeedbackUpdatesVSAReliability(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()

	obsA := createTestObservation(t, svc, "/tmp/project/a.go", "summary A", "raw A")
	obsB := createTestObservation(t, svc, "/tmp/project/b.go", "summary B", "raw B")
	if err := svc.AddLink(ctx, obsA.ID, obsB.ID, "related", "test"); err != nil {
		t.Fatalf("add link: %v", err)
	}

	if err := svc.MarkObservationUsefulness(ctx, MarkUsefulnessRequest{
		ObservationID: obsA.ID,
		Signal:        "useful",
		Weight:        1,
		Note:          "helpful",
	}); err != nil {
		t.Fatalf("mark useful: %v", err)
	}

	var bindSupport, bindNoise int
	if err := svc.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(support_count), 0), COALESCE(SUM(noise_count), 0) FROM memory_vsa_role_bindings WHERE observation_id = ?`,
		obsA.ID,
	).Scan(&bindSupport, &bindNoise); err != nil {
		t.Fatalf("query binding counters: %v", err)
	}
	if bindSupport <= 0 || bindNoise != 0 {
		t.Fatalf("unexpected binding counters after useful signal: support=%d noise=%d", bindSupport, bindNoise)
	}

	var assocSupport, assocNoise int
	if err := svc.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(support_count), 0), COALESCE(SUM(noise_count), 0) FROM memory_vsa_associations WHERE from_observation_id = ? OR to_observation_id = ?`,
		obsA.ID,
		obsA.ID,
	).Scan(&assocSupport, &assocNoise); err != nil {
		t.Fatalf("query association counters: %v", err)
	}
	if assocSupport <= 0 || assocNoise != 0 {
		t.Fatalf("unexpected association counters after useful signal: support=%d noise=%d", assocSupport, assocNoise)
	}

	if err := svc.MarkObservationUsefulness(ctx, MarkUsefulnessRequest{
		ObservationID: obsA.ID,
		Signal:        "failed",
		Weight:        1,
		Note:          "misleading",
	}); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	if err := svc.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(noise_count), 0) FROM memory_vsa_role_bindings WHERE observation_id = ?`,
		obsA.ID,
	).Scan(&bindNoise); err != nil {
		t.Fatalf("query binding noise: %v", err)
	}
	if bindNoise <= 0 {
		t.Fatalf("expected binding noise > 0 after failed signal, got %d", bindNoise)
	}
	if err := svc.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(noise_count), 0) FROM memory_vsa_associations WHERE from_observation_id = ? OR to_observation_id = ?`,
		obsA.ID,
		obsA.ID,
	).Scan(&assocNoise); err != nil {
		t.Fatalf("query association noise: %v", err)
	}
	if assocNoise <= 0 {
		t.Fatalf("expected association noise > 0 after failed signal, got %d", assocNoise)
	}
}

func TestRepairTriggersVSAReindexEvidence(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()

	obs := createTestObservation(t, svc, "/tmp/project/repair.go", "", "repair raw seed")
	_, err := svc.db.ExecContext(ctx,
		`UPDATE memory_observations SET stale = 1, summary = ?, raw_content = ?, updated_at = ? WHERE id = ?`,
		"",
		"repair raw updated content for evidence checks",
		time.Now().UnixMilli(),
		obs.ID,
	)
	if err != nil {
		t.Fatalf("prepare stale observation: %v", err)
	}

	detail, err := svc.RunRepairPass(ctx, RunRepairRequest{
		Mode:       "manual",
		MaxAgeDays: 30,
		Limit:      10,
		Note:       "repair evidence test",
	})
	if err != nil {
		t.Fatalf("run repair: %v", err)
	}
	if detail.Run.Candidates < 1 {
		t.Fatalf("expected at least one repair candidate, got %+v", detail.Run)
	}

	var beforeRaw, afterRaw string
	err = svc.db.QueryRowContext(ctx,
		`SELECT before_json, after_json FROM memory_vsa_reindex_items WHERE reason = ? ORDER BY id DESC LIMIT 1`,
		"repair_flow",
	).Scan(&beforeRaw, &afterRaw)
	if err != nil {
		t.Fatalf("query repair-linked vsa evidence: %v", err)
	}

	before := map[string]any{}
	after := map[string]any{}
	if err := json.Unmarshal([]byte(beforeRaw), &before); err != nil {
		t.Fatalf("decode before evidence: %v", err)
	}
	if err := json.Unmarshal([]byte(afterRaw), &after); err != nil {
		t.Fatalf("decode after evidence: %v", err)
	}
	if _, ok := before["rawContent"]; !ok {
		t.Fatalf("expected before evidence to include rawContent, got %v", before)
	}
	if _, ok := after["verificationState"]; !ok {
		t.Fatalf("expected after evidence to include verificationState, got %v", after)
	}
}

func TestVSAOffModeSkipsComputation(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()

	if _, err := svc.db.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES('retrieval_vsa_mode','off') ON CONFLICT(key) DO UPDATE SET value=excluded.value`); err != nil {
		t.Fatalf("set vsa mode off: %v", err)
	}

	obs := createTestObservation(t, svc, "/tmp/project/off-mode.go", "off mode summary", "off mode raw")
	if err := svc.ReindexObservationVSA(ctx, obs.ID, "off_mode_test", nil); err != nil {
		t.Fatalf("reindex observation in off mode: %v", err)
	}

	var count int
	if err := svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_vsa_pointers WHERE observation_id = ?`, obs.ID).Scan(&count); err != nil {
		t.Fatalf("query pointer count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no pointer rows in off mode, got %d", count)
	}
}
