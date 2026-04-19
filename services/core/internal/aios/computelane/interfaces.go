package computelane

import (
	"context"

	"forge/projectforge/services/core/internal/aios/compute/librarian"
	"forge/projectforge/services/core/internal/aios/domain"
)

type InferenceRequest struct {
	Event JournalEventRef
	Scope domain.ForgeScope
	Hints map[string]any
}

type JournalEventRef struct {
	EventID       string
	CorrelationID string
}

// SemanticInferenceService proposes candidate semantic actions only.
// It must not directly mutate canonical FORGE state.
type SemanticInferenceService interface {
	ProposeActions(ctx context.Context, req InferenceRequest) ([]domain.SemanticAction, error)
}

// LibrarianCell coordinates typed memory analysis and action proposals.
// Cells produce candidates; the kernel validates and commits.
type LibrarianCell interface {
	Name() string
	Propose(ctx context.Context, in librarian.CellInput) (librarian.CellOutput, error)
}

type ContextCompileRequest struct {
	Query  string
	Scope  domain.ForgeScope
	Budget domain.ContextBudget
}

// ContextCompiler materializes context packets from committed evidence.
type ContextCompiler interface {
	Compile(ctx context.Context, req ContextCompileRequest) (*domain.ContextPacket, error)
}

type PatternModelRequest struct {
	Scope       domain.ForgeScope
	EvidenceIDs []string
}

// AdaptivePolicyService derives policy/model candidates from evidence.
type AdaptivePolicyService interface {
	Derive(ctx context.Context, req PatternModelRequest) ([]domain.AdaptivePolicyModel, error)
}

type RetrievalRequest struct {
	Query string
	Scope domain.ForgeScope
	Limit int
}

// RetrievalService provides ranked evidence retrieval to compute workflows.
type RetrievalService interface {
	Retrieve(ctx context.Context, req RetrievalRequest) ([]domain.JournalEvent, error)
}

type IrisCandidateActionRequest struct {
	Scope  domain.ForgeScope
	Events []domain.JournalEvent
	Notes  []domain.MemoryNote
	Models []domain.AdaptivePolicyModel
}

type IrisContextRequest struct {
	Query  string
	Scope  domain.ForgeScope
	Budget domain.ContextBudget
}

// IrisBridge is the future IRIS seam.
// IRIS can propose actions and request context packets, but cannot commit state.
type IrisBridge interface {
	ProposeCandidateActions(ctx context.Context, req IrisCandidateActionRequest) ([]domain.SemanticAction, error)
	RequestContextPacket(ctx context.Context, req IrisContextRequest) (*domain.ContextPacket, error)
}
