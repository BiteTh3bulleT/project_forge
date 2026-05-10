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

func TestCleanupRuntimeCellSkipsPlaceholderArchiveTargets(t *testing.T) {
	t.Parallel()

	cell := CleanupRuntimeCell{}
	out, err := cell.Run(context.Background(), CellRunContext{
		Request: domain.IngestRequest{
			ID: "ingest-cleanup",
			Metadata: map[string]any{
				"archiveNoteIds": []any{
					"candidate-note",
					"candidate-123",
					"fake-archive-target",
					"placeholder-item",
					"note-real",
				},
				"archiveReason": "cleanup_review",
			},
			RequestedAt: 1761000010000,
		},
		Event: domain.JournalEvent{ID: "evt-cleanup"},
		Scope: domain.ForgeScope{
			WorkspaceID: "ws-cleanup",
			LaneID:      "control.semantic",
		},
		Actor:         domain.ActorIdentity{ID: "tester", Kind: "test"},
		Source:        domain.SourceInternal,
		CorrelationID: "corr-cleanup",
		TraceID:       "trace-cleanup",
	})
	if err != nil {
		t.Fatalf("cleanup run: %v", err)
	}
	if len(out.ProposedActions) != 1 {
		t.Fatalf("expected only one real archive proposal, got %d: %#v", len(out.ProposedActions), out.ProposedActions)
	}
	if got := out.ProposedActions[0].Payload["noteId"]; got != "note-real" {
		t.Fatalf("expected real note archive target, got %#v", got)
	}
	if skipped, ok := out.Hints["skippedPlaceholderArchiveTargets"].(int); !ok || skipped != 4 {
		t.Fatalf("expected four skipped placeholder targets, got %#v", out.Hints)
	}
}
