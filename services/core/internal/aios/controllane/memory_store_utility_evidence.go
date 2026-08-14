package controllane

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

func (s *InMemorySemanticStore) FindRetrievalUsefulnessTarget(_ context.Context, resultID int64) (RetrievalUsefulnessTarget, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target, ok := s.state.retrievalTargets[resultID]
	return cloneIntegrityValue(target), ok, nil
}

func (s *TransactionalSemanticStore) FindRetrievalUsefulnessTarget(_ context.Context, resultID int64) (RetrievalUsefulnessTarget, bool, error) {
	target, ok := s.state.retrievalTargets[resultID]
	return cloneIntegrityValue(target), ok, nil
}

func (s *InMemorySemanticStore) FindRestoreOutcomeFeedbackTarget(_ context.Context, id string) (RestoreOutcomeFeedbackTarget, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return findMemoryRestoreFeedbackTarget(&s.state, id)
}

func (s *TransactionalSemanticStore) FindRestoreOutcomeFeedbackTarget(_ context.Context, id string) (RestoreOutcomeFeedbackTarget, bool, error) {
	return findMemoryRestoreFeedbackTarget(s.state, id)
}

func findMemoryRestoreFeedbackTarget(state *memoryState, id string) (RestoreOutcomeFeedbackTarget, bool, error) {
	event, ok := state.restoreOutcomes[strings.TrimSpace(id)]
	if !ok {
		return RestoreOutcomeFeedbackTarget{}, false, nil
	}
	snapshot, ok := state.contextSnapshots[event.ContextPacketID]
	if !ok || strings.TrimSpace(snapshot.Scope.WorkspaceID) != strings.TrimSpace(event.WorkspaceID) ||
		strings.TrimSpace(snapshot.Scope.LaneID) != strings.TrimSpace(event.LaneID) {
		return RestoreOutcomeFeedbackTarget{}, false, nil
	}
	return RestoreOutcomeFeedbackTarget{
		RestoreOutcomeID: event.ID,
		Scope: domain.ForgeScope{
			WorkspaceID:   event.WorkspaceID,
			LaneID:        event.LaneID,
			SelectedPaths: append([]string(nil), snapshot.Scope.SelectedPaths...),
		},
		OriginalOutcome: event.Outcome,
		SourceSyscall:   event.SyscallID,
		CommittedBy:     event.CommittedBy,
	}, true, nil
}

func (s *InMemorySemanticStore) GetRestoreOutcomeFeedbackProjection(_ context.Context, id string, scope domain.ForgeScope) (RestoreOutcomeFeedbackProjection, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	projection, ok := s.state.restoreFeedbackProjection[strings.TrimSpace(id)]
	if !ok || !exactUtilityScopeMatches(scope, projection.Scope) {
		return RestoreOutcomeFeedbackProjection{}, false, nil
	}
	return cloneIntegrityValue(projection), true, nil
}

func (s *TransactionalSemanticStore) GetRestoreOutcomeFeedbackProjection(_ context.Context, id string, scope domain.ForgeScope) (RestoreOutcomeFeedbackProjection, bool, error) {
	projection, ok := s.state.restoreFeedbackProjection[strings.TrimSpace(id)]
	if !ok || !exactUtilityScopeMatches(scope, projection.Scope) {
		return RestoreOutcomeFeedbackProjection{}, false, nil
	}
	return cloneIntegrityValue(projection), true, nil
}

func (s *InMemorySemanticStore) CreateRetrievalUsefulnessEvent(event RetrievalUsefulnessEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return createMemoryRetrievalUsefulnessEvent(&s.state, event)
}

func (s *TransactionalSemanticStore) CreateRetrievalUsefulnessEvent(event RetrievalUsefulnessEvent) error {
	return createMemoryRetrievalUsefulnessEvent(s.state, event)
}

func createMemoryRetrievalUsefulnessEvent(state *memoryState, event RetrievalUsefulnessEvent) error {
	if _, exists := state.retrievalUtility[event.ID]; exists {
		return fmt.Errorf("retrieval usefulness event %q already exists", event.ID)
	}
	target, ok := state.retrievalTargets[event.ResultID]
	if !ok {
		return fmt.Errorf("retrieval result %d not found", event.ResultID)
	}
	if err := validateRetrievalUsefulnessEvent(event, target); err != nil {
		return err
	}
	prior := state.retrievalUtilityProjection[event.ResultID]
	event.Label = NormalizeRetrievalUsefulnessLabel(event.Label)
	event.PriorProjection = map[string]any{
		"label":         prior.Label,
		"note":          prior.Note,
		"latestEventId": prior.LatestEventID,
		"nonCanonical":  true,
	}
	event.ProvenanceID = provenanceID(event.Scope, event.Provenance)
	state.retrievalUtility[event.ID] = cloneIntegrityValue(event)
	state.retrievalUtilityProjection[event.ResultID] = RetrievalUsefulnessProjection{
		ResultID: event.ResultID, LatestEventID: event.ID, Label: event.Label,
		Note: strings.TrimSpace(event.Note), UpdatedAt: event.CreatedAt, NonCanonical: true,
	}
	return nil
}

func (s *InMemorySemanticStore) CreateRestoreOutcomeFeedbackEvent(event RestoreOutcomeFeedbackEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return createMemoryRestoreOutcomeFeedbackEvent(&s.state, event)
}

func (s *TransactionalSemanticStore) CreateRestoreOutcomeFeedbackEvent(event RestoreOutcomeFeedbackEvent) error {
	return createMemoryRestoreOutcomeFeedbackEvent(s.state, event)
}

func createMemoryRestoreOutcomeFeedbackEvent(state *memoryState, event RestoreOutcomeFeedbackEvent) error {
	if _, exists := state.restoreFeedbackEvents[event.ID]; exists {
		return fmt.Errorf("restore outcome feedback event %q already exists", event.ID)
	}
	target, ok, err := findMemoryRestoreFeedbackTarget(state, event.RestoreOutcomeID)
	if err != nil {
		return err
	}
	if !ok {
		return restoreOutcomeNotFound(event.RestoreOutcomeID)
	}
	if err := validateRestoreOutcomeFeedbackEvent(event, target); err != nil {
		return err
	}
	prior, priorOK := state.restoreFeedbackProjection[event.RestoreOutcomeID]
	event.PriorProjection = map[string]any{"present": priorOK, "nonCanonical": true}
	if priorOK {
		event.PriorProjection["latestEventId"] = prior.LatestEventID
		event.PriorProjection["outcome"] = prior.Outcome
		event.PriorProjection["outcomeConfidence"] = prior.OutcomeConfidence
		event.PriorProjection["updatedAt"] = prior.UpdatedAt
	}
	event.ProvenanceID = provenanceID(event.Scope, event.Provenance)
	feedback := normalizeRestoreOutcomeFeedback(RestoreOutcomeFeedback{
		Outcome:           event.Outcome,
		OutcomeConfidence: event.OutcomeConfidence,
		OperatorFeedback:  event.OperatorFeedback,
		CorrectionSummary: event.CorrectionSummary,
		Metadata:          event.Metadata,
		CorrelationID:     event.CorrelationID,
		TraceID:           event.TraceID,
		UpdatedBy:         event.Provenance.Actor,
		UpdatedAt:         event.CreatedAt,
	})
	event.ProjectionSnapshot = feedback
	state.restoreFeedbackEvents[event.ID] = cloneIntegrityValue(event)
	state.restoreFeedbackProjection[event.RestoreOutcomeID] = RestoreOutcomeFeedbackProjection{
		RestoreOutcomeID:  event.RestoreOutcomeID,
		LatestEventID:     event.ID,
		Scope:             event.Scope,
		Outcome:           feedback.Outcome,
		OutcomeConfidence: feedback.OutcomeConfidence,
		OperatorFeedback:  feedback.OperatorFeedback,
		CorrectionSummary: feedback.CorrectionSummary,
		UpdatedBy:         feedback.UpdatedBy,
		UpdatedAt:         feedback.UpdatedAt,
		Metadata:          cloneIntegrityValue(feedback.Metadata),
		NonCanonical:      true,
	}
	return nil
}
