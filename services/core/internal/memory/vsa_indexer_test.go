package memory

import (
	"context"
	"errors"
	"testing"

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
	return New(dbStore.DB), func() { _ = dbStore.Close() }
}

func createTestObservation(t *testing.T, svc *Service, path, summary, raw string) *Observation {
	t.Helper()
	obs, err := svc.RecordObservation(context.Background(), RecordObservationRequest{
		Type: "memory_note", SourcePath: path, TaskType: "test", Summary: summary,
		RawContent: raw, Tags: []string{"test"},
	})
	if err != nil {
		t.Fatalf("record observation: %v", err)
	}
	return obs
}

func TestLegacyVSAProjectionWritersFailClosed(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()
	obs := createTestObservation(t, svc, "/tmp/project/a.go", "alpha", "alpha body")

	if err := svc.ReindexObservationVSA(ctx, obs.ID, "legacy_direct", nil); !errors.Is(err, ErrVSAProjectionAuthorityRequired) {
		t.Fatalf("direct reindex error = %v", err)
	}
	if err := svc.TouchVSAReliabilityFromUsefulness(ctx, obs.ID, "useful", 1); !errors.Is(err, ErrVSAProjectionAuthorityRequired) {
		t.Fatalf("direct reliability error = %v", err)
	}
	run, err := svc.RunVSAReindex(ctx, RunVSAReindexRequest{Limit: 10, Force: true, Reason: "legacy_run"})
	if !errors.Is(err, ErrVSAProjectionAuthorityRequired) {
		t.Fatalf("legacy run error: %v", err)
	}
	if run != nil {
		t.Fatalf("legacy run returned detail: %+v", run)
	}
	var rows int
	if err := svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_vsa_pointers`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("legacy writers created %d pointer rows", rows)
	}
}

func TestLegacyUsefulnessWriterFailsClosedWithoutMutatingProjection(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()
	obs := createTestObservation(t, svc, "/tmp/project/a.go", "alpha", "alpha body")

	if _, err := svc.db.ExecContext(ctx, `
INSERT INTO memory_vsa_role_bindings(observation_id,role,filler,support_count,noise_count,created_at,updated_at)
VALUES(?,'type','memory_note',7,3,1,1)`, obs.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkObservationUsefulness(ctx, MarkUsefulnessRequest{ObservationID: obs.ID, Signal: "useful", Weight: 1, Note: "immutable signal"}); !errors.Is(err, ErrMemoryEvidenceAuthorityRequired) {
		t.Fatalf("legacy usefulness error = %v", err)
	}
	var events, support, noise int
	if err := svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_usefulness_events WHERE observation_id=?`, obs.ID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.QueryRowContext(ctx, `SELECT support_count,noise_count FROM memory_vsa_role_bindings WHERE observation_id=?`, obs.ID).Scan(&support, &noise); err != nil {
		t.Fatal(err)
	}
	if events != 0 || support != 7 || noise != 3 {
		t.Fatalf("events=%d projection counters=%d/%d", events, support, noise)
	}
}

func TestObservationAndLinkWritesDoNotAutoReindex(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()
	a := createTestObservation(t, svc, "/tmp/project/a.go", "a", "a")
	b := createTestObservation(t, svc, "/tmp/project/b.go", "b", "b")
	if err := svc.AddLink(ctx, a.ID, b.ID, "related", "no projection side effect"); err != nil {
		t.Fatal(err)
	}
	updated := "changed"
	if _, err := svc.UpdateObservation(ctx, a.ID, UpdateObservationRequest{Summary: &updated}); err != nil {
		t.Fatal(err)
	}
	var rows int
	if err := svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_vsa_pointers`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("observation/link writes created %d projection rows", rows)
	}
}
