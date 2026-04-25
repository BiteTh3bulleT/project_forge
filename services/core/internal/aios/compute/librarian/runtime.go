package librarian

import (
	"context"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/truth"
)

type CellReadRepositories struct {
	Journal        controllane.JournalRepository
	Notes          controllane.MemoryNoteRepository
	Links          controllane.SemanticLinkRepository
	State          controllane.StateRepository
	Loops          controllane.OpenLoopRepository
	Artifacts      controllane.ArtifactRefRepository
	Models         controllane.DerivedModelRepository
	Contradictions controllane.ContradictionRepository
	Supersessions  controllane.SupersessionRepository
	ContextPackets controllane.ContextPacketRepository
}

type CellRunContext struct {
	Request         domain.IngestRequest
	Event           domain.JournalEvent
	Scope           domain.ForgeScope
	Actor           domain.ActorIdentity
	Source          domain.ActionSource
	Provenance      domain.Provenance
	CorrelationID   string
	TraceID         string
	DryRun          bool
	CurrentState    []domain.StateItem
	ActiveNotes     []domain.MemoryNote
	ActiveLoops     []domain.OpenLoop
	RecentArtifacts []domain.ArtifactRef
	ExistingActions []domain.SyscallRequest
	Repositories    CellReadRepositories
	Semantic        SemanticInferenceService
	Truth           *truth.Engine
	FeatureFlags    map[string]bool
}

type CellRunResult struct {
	CellName        string
	CellVersion     string
	ProposedActions []domain.SyscallRequest
	AnalysisNotes   []string
	Warnings        []string
	Errors          []domain.IngestError
	Confidence      float64
	Duration        time.Duration
	SkippedReason   string
	Hints           map[string]any
}

func (r CellRunResult) Diagnostic() domain.CellDiagnostic {
	d := domain.CellDiagnostic{
		CellName:      r.CellName,
		CellVersion:   r.CellVersion,
		ProposedCount: len(r.ProposedActions),
		Warnings:      append([]string{}, r.Warnings...),
		Errors:        append([]domain.IngestError{}, r.Errors...),
		DurationMs:    r.Duration.Milliseconds(),
		Metadata:      map[string]any{},
	}
	if r.SkippedReason != "" {
		d.Skipped = true
		d.SkippedReason = r.SkippedReason
	}
	if len(r.Hints) > 0 {
		d.Metadata["hints"] = r.Hints
	}
	return d
}

type RuntimeCell interface {
	Name() string
	Version() string
	Lane() string
	Dependencies() []string
	CanRun(ctx context.Context, in CellRunContext) (bool, string)
	Run(ctx context.Context, in CellRunContext) (CellRunResult, error)
}
