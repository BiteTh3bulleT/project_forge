package reviews

import (
	"context"
	"encoding/json"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestCreateUpdateAndListReviewRecords(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	svc := New(st.DB)
	rec, err := svc.Create(ctx, CreateRequest{
		TargetType:  " job ",
		TargetID:    " job-1 ",
		Summary:     " initial summary ",
		Notes:       " initial notes ",
		Annotations: []string{" first ", "", "second"},
	})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if rec.TargetType != "job" || rec.TargetID != "job-1" {
		t.Fatalf("trimmed target mismatch: %#v", rec)
	}
	if rec.Status != StatusPending {
		t.Fatalf("default status = %q, want pending", rec.Status)
	}
	if rec.Reviewer != "operator" {
		t.Fatalf("default reviewer = %q, want operator", rec.Reviewer)
	}
	assertReviewAnnotations(t, rec.Annotations, []string{"first", "second"})

	status := StatusApproved
	summary := " approved summary "
	annotations := []string{" approved "}
	reviewer := " reviewer-1 "
	updated, err := svc.Update(ctx, rec.ID, UpdateRequest{
		Status:      &status,
		Summary:     &summary,
		Annotations: &annotations,
		Reviewer:    &reviewer,
	})
	if err != nil {
		t.Fatalf("update review: %v", err)
	}
	if updated.Status != StatusApproved || updated.Summary != "approved summary" || updated.Reviewer != "reviewer-1" {
		t.Fatalf("updated review mismatch: %#v", updated)
	}
	assertReviewAnnotations(t, updated.Annotations, []string{"approved"})

	pending, err := svc.List(ctx, "pending", 10)
	if err != nil {
		t.Fatalf("list pending reviews: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending reviews = %#v, want none after approval", pending)
	}
	approved, err := svc.List(ctx, "APPROVED", 10)
	if err != nil {
		t.Fatalf("list approved reviews: %v", err)
	}
	if len(approved) != 1 || approved[0].ID != rec.ID {
		t.Fatalf("approved reviews = %#v, want updated review", approved)
	}
}

func TestCreateRequiresReviewTarget(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if _, err := New(st.DB).Create(context.Background(), CreateRequest{TargetType: "job"}); err == nil {
		t.Fatal("Create without target id succeeded")
	}
}

func assertReviewAnnotations(t *testing.T, raw json.RawMessage, want []string) {
	t.Helper()
	var got []string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal annotations %s: %v", string(raw), err)
	}
	if len(got) != len(want) {
		t.Fatalf("annotations = %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("annotations = %#v, want %#v", got, want)
		}
	}
}
