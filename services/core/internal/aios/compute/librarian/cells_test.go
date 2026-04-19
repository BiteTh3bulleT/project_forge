package librarian

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestStaticIntakeCellProposesCreateNote(t *testing.T) {
	cell := StaticIntakeCell{}
	out, err := cell.Run(context.Background(), IntakeInput{
		Event: domain.JournalEvent{
			ID:    "evt-1",
			Type:  "job.completed",
			Scope: domain.ForgeScope{WorkspaceID: "ws-main"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.CandidateActions) != 1 {
		t.Fatalf("expected one candidate action, got %d", len(out.CandidateActions))
	}
	if out.CandidateActions[0].Action != domain.ActionCreateNote {
		t.Fatalf("expected CREATE_NOTE action, got %s", out.CandidateActions[0].Action)
	}
}
