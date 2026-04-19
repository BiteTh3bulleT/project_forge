package librarian

import (
	"context"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

// CellInput and CellOutput are shared envelopes for all librarian cells.
// Cells must not write persistence directly; they only return candidate actions.
type CellInput interface {
	CellKind() string
}

type CellOutput struct {
	CandidateActions []domain.SemanticAction
	Signals          map[string]float64
	Notes            []string
}

type IntakeInput struct {
	Event domain.JournalEvent
}

func (IntakeInput) CellKind() string { return "intake" }

type CatalogInput struct {
	Notes []domain.MemoryNote
}

func (CatalogInput) CellKind() string { return "catalog" }

type LinkerInput struct {
	Notes []domain.MemoryNote
	Links []domain.SemanticLink
}

func (LinkerInput) CellKind() string { return "linker" }

type ContradictionInput struct {
	Notes []domain.MemoryNote
	Links []domain.SemanticLink
}

func (ContradictionInput) CellKind() string { return "contradiction" }

type StateInput struct {
	StateItems []domain.StateItem
	Loops      []domain.OpenLoop
}

func (StateInput) CellKind() string { return "state" }

type PatternInput struct {
	Events []domain.JournalEvent
	Notes  []domain.MemoryNote
	Links  []domain.SemanticLink
}

func (PatternInput) CellKind() string { return "pattern" }

type RecallInput struct {
	Query  string
	Scope  domain.ForgeScope
	Events []domain.JournalEvent
	Notes  []domain.MemoryNote
}

func (RecallInput) CellKind() string { return "recall" }

type CleanupInput struct {
	Notes []domain.MemoryNote
	Loops []domain.OpenLoop
}

func (CleanupInput) CellKind() string { return "cleanup" }

type IntakeCell interface {
	Run(ctx context.Context, in IntakeInput) (CellOutput, error)
}

type CatalogCell interface {
	Run(ctx context.Context, in CatalogInput) (CellOutput, error)
}

type LinkerCell interface {
	Run(ctx context.Context, in LinkerInput) (CellOutput, error)
}

type ContradictionCell interface {
	Run(ctx context.Context, in ContradictionInput) (CellOutput, error)
}

type StateCell interface {
	Run(ctx context.Context, in StateInput) (CellOutput, error)
}

type PatternCell interface {
	Run(ctx context.Context, in PatternInput) (CellOutput, error)
}

type RecallCell interface {
	Run(ctx context.Context, in RecallInput) (CellOutput, error)
}

type CleanupCell interface {
	Run(ctx context.Context, in CleanupInput) (CellOutput, error)
}

// StaticIntakeCell is a deterministic stub used for scaffold smoke tests.
// It proposes a single CREATE_NOTE action derived from the incoming event.
type StaticIntakeCell struct{}

func (StaticIntakeCell) Run(_ context.Context, in IntakeInput) (CellOutput, error) {
	summary := strings.TrimSpace(in.Event.Type)
	if summary == "" {
		summary = "event.observed"
	}
	action := domain.SemanticAction{
		ID:     "stub-intake-create-note",
		Action: domain.ActionCreateNote,
		Actor: domain.ActorIdentity{
			ID:   "librarian.intake.stub",
			Kind: string(domain.SourceInternal),
		},
		Source: domain.SourceInternal,
		Scope:  in.Event.Scope,
		Payload: map[string]any{
			"type":    string(domain.NoteEpisode),
			"title":   "Intake observation",
			"content": summary,
			"eventId": in.Event.ID,
			"status":  string(domain.NoteActive),
		},
		RequestedAt: domain.NowMillis(),
		Provenance: domain.Provenance{
			Actor:     "librarian.intake.stub",
			ActorType: "service",
			Source:    "compute.librarian",
		},
		CorrelationID: in.Event.CorrelationID,
		TraceID:       in.Event.CorrelationID,
	}
	return CellOutput{
		CandidateActions: []domain.SemanticAction{action},
		Signals:          map[string]float64{"novelty": 0.5},
		Notes:            []string{"stub intake proposal generated"},
	}, nil
}
