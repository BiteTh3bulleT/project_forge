package computelane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/compute/librarian"
	"forge/projectforge/services/core/internal/aios/domain"
)

type stubInference struct{}

func (stubInference) ProposeActions(_ context.Context, _ InferenceRequest) ([]domain.SemanticAction, error) {
	return []domain.SemanticAction{{ID: "act-1", Action: domain.ActionCreateNote, Scope: domain.ForgeScope{WorkspaceID: "ws"}}}, nil
}

type stubCell struct{}

func (stubCell) Name() string { return "stub-cell" }
func (stubCell) Propose(_ context.Context, _ librarian.CellInput) (librarian.CellOutput, error) {
	return librarian.CellOutput{}, nil
}

type stubCompiler struct{}

func (stubCompiler) Compile(_ context.Context, req ContextCompileRequest) (*domain.ContextPacket, error) {
	return &domain.ContextPacket{ID: "ctx-1", Query: req.Query, Scope: req.Scope, Budget: req.Budget}, nil
}

type stubAdaptivePolicy struct{}

func (stubAdaptivePolicy) Derive(_ context.Context, _ PatternModelRequest) ([]domain.AdaptivePolicyModel, error) {
	return []domain.AdaptivePolicyModel{{ID: "model-1", Type: "routing"}}, nil
}

type stubRetrieval struct{}

func (stubRetrieval) Retrieve(_ context.Context, _ RetrievalRequest) ([]domain.JournalEvent, error) {
	return []domain.JournalEvent{{ID: "evt-1", Type: "event", Scope: domain.ForgeScope{WorkspaceID: "ws"}}}, nil
}

type stubIrisBridge struct{}

func (stubIrisBridge) ProposeCandidateActions(_ context.Context, _ IrisCandidateActionRequest) ([]domain.SemanticAction, error) {
	return []domain.SemanticAction{{ID: "iris-act-1", Action: domain.ActionCreateLink, Scope: domain.ForgeScope{WorkspaceID: "ws"}}}, nil
}

func (stubIrisBridge) RequestContextPacket(_ context.Context, req IrisContextRequest) (*domain.ContextPacket, error) {
	return &domain.ContextPacket{ID: "iris-ctx-1", Query: req.Query, Scope: req.Scope, Budget: req.Budget}, nil
}

func TestComputeLaneInterfacesCompile(t *testing.T) {
	var _ SemanticInferenceService = stubInference{}
	var _ LibrarianCell = stubCell{}
	var _ ContextCompiler = stubCompiler{}
	var _ AdaptivePolicyService = stubAdaptivePolicy{}
	var _ RetrievalService = stubRetrieval{}
	var _ IrisBridge = stubIrisBridge{}
}

func TestIrisBridgeDoesNotRequireLiveIris(t *testing.T) {
	bridge := stubIrisBridge{}
	res, err := bridge.ProposeCandidateActions(context.Background(), IrisCandidateActionRequest{
		Scope: domain.ForgeScope{WorkspaceID: "ws"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected candidate actions from stub bridge")
	}
}
