package memory

import (
	"context"
	"errors"
	"testing"
)

func TestLegacyObservationWritersFailClosed(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()
	if _, err := svc.RecordObservation(ctx, RecordObservationRequest{Type: "note"}); !errors.Is(err, ErrMemoryEvidenceAuthorityRequired) {
		t.Fatalf("record error = %v", err)
	}
	if _, err := svc.UpdateObservation(ctx, 1, UpdateObservationRequest{}); !errors.Is(err, ErrMemoryEvidenceAuthorityRequired) {
		t.Fatalf("update error = %v", err)
	}
	if err := svc.AddLink(ctx, 1, 2, "related", "legacy"); !errors.Is(err, ErrMemoryEvidenceAuthorityRequired) {
		t.Fatalf("link error = %v", err)
	}
	if err := svc.MarkObservationUsefulness(ctx, MarkUsefulnessRequest{ObservationID: 1, Signal: "useful"}); !errors.Is(err, ErrMemoryEvidenceAuthorityRequired) {
		t.Fatalf("usefulness error = %v", err)
	}
	if err := svc.LinkResultObservation(ctx, 1, 1, "legacy"); !errors.Is(err, ErrMemoryEvidenceAuthorityRequired) {
		t.Fatalf("result link error = %v", err)
	}
	if detail, err := svc.RunRepairPass(ctx, RunRepairRequest{Mode: "legacy"}); detail != nil || !errors.Is(err, ErrMemoryEvidenceAuthorityRequired) {
		t.Fatalf("repair detail=%+v error=%v", detail, err)
	}
	for _, table := range []string{
		"memory_observations",
		"memory_observation_links",
		"memory_usefulness_events",
		"retrieval_result_observations",
		"memory_repair_runs",
		"memory_repair_items",
	} {
		var count int
		if err := svc.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("legacy writer changed %s: count=%d", table, count)
		}
	}
}

func TestLegacyObservationReadsRemainAvailable(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()
	for _, statement := range []string{
		`INSERT INTO memory_observations(created_at,updated_at,observed_at,type,raw_content,summary,source_path,origin_kind,origin_id) VALUES(1,1,1,'note','alpha','alpha summary','/a','fixture','a')`,
		`INSERT INTO memory_observations(created_at,updated_at,observed_at,type,raw_content,summary,source_path,origin_kind,origin_id) VALUES(2,2,2,'other','beta','beta summary','/b','fixture','b')`,
		`INSERT INTO memory_observation_links(created_at,from_observation_id,to_observation_id,relation_type,note) VALUES(3,1,2,'related','legacy read')`,
	} {
		if _, err := svc.db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := svc.ListObservations(ctx, ListObservationsRequest{Type: "note", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Summary != "alpha summary" {
		t.Fatalf("legacy list = %+v", rows)
	}
	detail, err := svc.GetObservation(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.OutgoingLinks) != 1 || detail.OutgoingLinks[0].ToObservationID != 2 {
		t.Fatalf("legacy detail links = %+v", detail.OutgoingLinks)
	}
}
