package librarian

import (
	"context"

	"forge/projectforge/services/core/internal/aios/domain"
)

type InferenceRequest struct {
	Event         domain.JournalEvent
	Scope         domain.ForgeScope
	CorrelationID string
	TraceID       string
	Content       string
	Hints         map[string]any
}

type InferenceNeighborhood struct {
	Notes  []domain.MemoryNote
	Links  []domain.SemanticLink
	State  []domain.StateItem
	Loops  []domain.OpenLoop
	Events []domain.JournalEvent
}

// SemanticInferenceService is an optional semantic proposer adapter.
// Kernel correctness and persistence validity must not depend on this interface.
type SemanticInferenceService interface {
	ExtractCandidates(ctx context.Context, req InferenceRequest) ([]domain.SyscallRequest, error)
	ClassifyCandidate(ctx context.Context, candidate domain.SyscallRequest) (domain.SyscallRequest, error)
	SuggestLinks(ctx context.Context, candidate domain.SyscallRequest, neighborhood InferenceNeighborhood) ([]domain.SyscallRequest, error)
	DetectContradictions(ctx context.Context, candidate domain.SyscallRequest, neighborhood InferenceNeighborhood) ([]domain.SyscallRequest, error)
	ProposeModel(ctx context.Context, scope domain.ForgeScope, neighborhood InferenceNeighborhood) (*domain.SyscallRequest, error)
	SynthesizeSummary(ctx context.Context, scope domain.ForgeScope, neighborhood InferenceNeighborhood) (string, error)
}

type NoopSemanticInference struct{}

func (NoopSemanticInference) ExtractCandidates(_ context.Context, _ InferenceRequest) ([]domain.SyscallRequest, error) {
	return nil, nil
}

func (NoopSemanticInference) ClassifyCandidate(_ context.Context, candidate domain.SyscallRequest) (domain.SyscallRequest, error) {
	return candidate, nil
}

func (NoopSemanticInference) SuggestLinks(_ context.Context, _ domain.SyscallRequest, _ InferenceNeighborhood) ([]domain.SyscallRequest, error) {
	return nil, nil
}

func (NoopSemanticInference) DetectContradictions(_ context.Context, _ domain.SyscallRequest, _ InferenceNeighborhood) ([]domain.SyscallRequest, error) {
	return nil, nil
}

func (NoopSemanticInference) ProposeModel(_ context.Context, _ domain.ForgeScope, _ InferenceNeighborhood) (*domain.SyscallRequest, error) {
	return nil, nil
}

func (NoopSemanticInference) SynthesizeSummary(_ context.Context, _ domain.ForgeScope, _ InferenceNeighborhood) (string, error) {
	return "", nil
}
