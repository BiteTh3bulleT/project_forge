package memory

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func requireStringSlice(t *testing.T, raw json.RawMessage, want []string) {
	t.Helper()
	var got []string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal raw string slice %q: %v", string(raw), err)
	}
	if len(got) != len(want) {
		t.Fatalf("slice length mismatch: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice mismatch at %d: got %v want %v", i, got, want)
		}
	}
}

func TestRecordObservationNormalizesAndUpsertsByOrigin(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()

	first, err := svc.RecordObservation(ctx, RecordObservationRequest{
		Type:              " memory_note ",
		RawContent:        " raw content ",
		Summary:           " summary ",
		EmbeddingRef:      " embedding:a ",
		ProjectKey:        " project-a ",
		SourcePath:        " /workspace/a.go ",
		Entities:          []string{" alpha ", "", "beta"},
		Tags:              []string{" tag-a ", "\t", "tag-b"},
		RelatedFiles:      []string{" /workspace/b.go "},
		Lineage:           []string{" seed "},
		TaskType:          " research ",
		Confidence:        1.8,
		VerificationState: "",
		OriginKind:        " job ",
		OriginID:          " job-1 ",
		ObservedAtMs:      1234,
	})
	if err != nil {
		t.Fatalf("record first observation: %v", err)
	}
	if first.Type != "memory_note" || first.RawContent != "raw content" || first.Summary != "summary" {
		t.Fatalf("observation text fields were not normalized: %+v", first)
	}
	if first.Confidence != 1 {
		t.Fatalf("expected confidence to clamp to 1, got %v", first.Confidence)
	}
	if first.VerificationState != "unknown" {
		t.Fatalf("expected default verification state unknown, got %q", first.VerificationState)
	}
	if first.OriginKind != "job" || first.OriginID != "job-1" {
		t.Fatalf("origin fields were not normalized: %+v", first)
	}
	requireStringSlice(t, first.Entities, []string{"alpha", "beta"})
	requireStringSlice(t, first.Tags, []string{"tag-a", "tag-b"})
	requireStringSlice(t, first.RelatedFiles, []string{"/workspace/b.go"})
	requireStringSlice(t, first.Lineage, []string{"seed"})

	second, err := svc.RecordObservation(ctx, RecordObservationRequest{
		Type:         "decision",
		RawContent:   "updated raw",
		Summary:      "updated summary",
		Tags:         []string{"updated"},
		Confidence:   -2,
		OriginKind:   "job",
		OriginID:     "job-1",
		ObservedAtMs: 5678,
	})
	if err != nil {
		t.Fatalf("record origin upsert: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected origin upsert to reuse id %d, got %d", first.ID, second.ID)
	}
	if second.Type != "decision" || second.Summary != "updated summary" || second.ObservedAtMs != 5678 {
		t.Fatalf("origin upsert did not update canonical fields: %+v", second)
	}
	if second.Confidence != 0.5 {
		t.Fatalf("expected non-positive confidence to default to 0.5, got %v", second.Confidence)
	}

	all, err := svc.ListObservations(ctx, ListObservationsRequest{Limit: 10})
	if err != nil {
		t.Fatalf("list observations: %v", err)
	}
	if len(all) != 1 || all[0].ID != first.ID {
		t.Fatalf("expected one upserted observation id %d, got %+v", first.ID, all)
	}
}

func TestListObservationsFiltersAndDefaultLimit(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()

	res, err := svc.db.ExecContext(ctx, `
INSERT INTO dossiers(created_at, updated_at, name)
VALUES(100, 100, 'memory-service-test')`)
	if err != nil {
		t.Fatalf("insert dossier: %v", err)
	}
	dossierID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read dossier id: %v", err)
	}
	first, err := svc.RecordObservation(ctx, RecordObservationRequest{
		Type:         "decision",
		Summary:      "first",
		DossierID:    &dossierID,
		OriginKind:   "job",
		OriginID:     "one",
		ObservedAtMs: 1000,
	})
	if err != nil {
		t.Fatalf("record first observation: %v", err)
	}
	second, err := svc.RecordObservation(ctx, RecordObservationRequest{
		Type:         "decision",
		Summary:      "second",
		DossierID:    &dossierID,
		OriginKind:   "manual",
		OriginID:     "two",
		ObservedAtMs: 2000,
	})
	if err != nil {
		t.Fatalf("record second observation: %v", err)
	}
	if _, err := svc.UpdateObservation(ctx, second.ID, UpdateObservationRequest{Stale: ptrBool(true)}); err != nil {
		t.Fatalf("mark second stale: %v", err)
	}

	filtered, err := svc.ListObservations(ctx, ListObservationsRequest{
		DossierID:  &dossierID,
		Type:       " decision ",
		OriginKind: " manual ",
		StaleOnly:  true,
		Limit:      -5,
	})
	if err != nil {
		t.Fatalf("list filtered observations: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != second.ID {
		t.Fatalf("expected only stale manual observation id %d, got %+v", second.ID, filtered)
	}

	all, err := svc.ListObservations(ctx, ListObservationsRequest{Limit: 10})
	if err != nil {
		t.Fatalf("list all observations: %v", err)
	}
	if len(all) != 2 || all[0].ID != second.ID || all[1].ID != first.ID {
		t.Fatalf("expected observed_at desc order [%d %d], got %+v", second.ID, first.ID, all)
	}
}

func TestUpdateObservationPreservesUnsetFieldsAndHydratesLinksAndSignals(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()

	from := createTestObservation(t, svc, "/tmp/project/from.go", "from summary", "from raw")
	to := createTestObservation(t, svc, "/tmp/project/to.go", "to summary", "to raw")
	if err := svc.AddLink(ctx, from.ID, to.ID, "", " link note "); err != nil {
		t.Fatalf("add default link: %v", err)
	}
	lastVerified := int64(4242)
	detail, err := svc.UpdateObservation(ctx, from.ID, UpdateObservationRequest{
		VerificationState: ptrString(" verified "),
		Stale:             ptrBool(true),
		LastVerifiedAtMs:  &lastVerified,
		Tags:              []string{" keep ", "", "new"},
		RelatedFiles:      []string{" /tmp/project/related.go "},
	})
	if err != nil {
		t.Fatalf("update observation: %v", err)
	}
	if detail.Summary != "from summary" {
		t.Fatalf("expected nil summary update to preserve summary, got %q", detail.Summary)
	}
	if detail.VerificationState != "verified" || !detail.Stale {
		t.Fatalf("expected trimmed verification and stale flag, got %+v", detail.Observation)
	}
	if detail.LastVerifiedAtMs == nil || *detail.LastVerifiedAtMs != lastVerified {
		t.Fatalf("expected last verified timestamp %d, got %+v", lastVerified, detail.LastVerifiedAtMs)
	}
	requireStringSlice(t, detail.Tags, []string{"keep", "new"})
	requireStringSlice(t, detail.RelatedFiles, []string{"/tmp/project/related.go"})

	if err := svc.MarkObservationUsefulness(ctx, MarkUsefulnessRequest{
		ObservationID: from.ID,
		Signal:        "noise",
		Weight:        2.5,
		Note:          "too broad",
	}); !errors.Is(err, ErrMemoryEvidenceAuthorityRequired) {
		t.Fatalf("legacy usefulness error: %v", err)
	}
	hydrated, err := svc.GetObservation(ctx, from.ID)
	if err != nil {
		t.Fatalf("get hydrated observation: %v", err)
	}
	if len(hydrated.OutgoingLinks) != 1 {
		t.Fatalf("expected one outgoing link, got %+v", hydrated.OutgoingLinks)
	}
	if hydrated.OutgoingLinks[0].RelationType != "related" || hydrated.OutgoingLinks[0].Note != "link note" {
		t.Fatalf("expected default related link with trimmed note, got %+v", hydrated.OutgoingLinks[0])
	}
	if len(hydrated.Signals) != 0 {
		t.Fatalf("legacy usefulness writer created signals: %+v", hydrated.Signals)
	}

	target, err := svc.GetObservation(ctx, to.ID)
	if err != nil {
		t.Fatalf("get target observation: %v", err)
	}
	if len(target.IncomingLinks) != 1 || target.IncomingLinks[0].FromObservationID != from.ID {
		t.Fatalf("expected incoming link from %d, got %+v", from.ID, target.IncomingLinks)
	}
}

func ptrBool(v bool) *bool {
	return &v
}

func ptrString(v string) *string {
	return &v
}
