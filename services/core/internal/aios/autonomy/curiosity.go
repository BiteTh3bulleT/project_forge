package autonomy

import (
	"context"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type CuriosityQueueService struct {
	repo      CuriosityRepository
	intents   *IntentQueueService
	nowMillis func() int64
}

func NewCuriosityQueueService(repo CuriosityRepository, intents *IntentQueueService, nowMillis func() int64) *CuriosityQueueService {
	if nowMillis == nil {
		nowMillis = domain.NowMillis
	}
	return &CuriosityQueueService{repo: repo, intents: intents, nowMillis: nowMillis}
}

func (s *CuriosityQueueService) Add(ctx context.Context, item domain.CuriosityItem) (domain.CuriosityItem, error) {
	if item.CreatedAt <= 0 {
		item.CreatedAt = s.nowMillis()
	}
	item.UpdatedAt = item.CreatedAt
	if strings.TrimSpace(item.ID) == "" {
		item.ID = "curiosity-" + shortHash(item.Title, item.Question, item.Scope.WorkspaceID)
	}
	if strings.TrimSpace(string(item.Status)) == "" {
		item.Status = domain.CuriosityOpen
	}
	if strings.TrimSpace(item.Scope.WorkspaceID) == "" {
		return domain.CuriosityItem{}, domain.AutonomyError{Code: domain.AutonomyErrInvalidScope, Field: "scope.workspaceId", Message: "curiosity scope.workspaceId is required"}
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return domain.CuriosityItem{}, err
	}
	return item, nil
}

func (s *CuriosityQueueService) ListOpen(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.CuriosityItem, error) {
	return s.repo.ListByScope(ctx, scope, domain.CuriosityOpen, limit)
}

func (s *CuriosityQueueService) PromoteToIntent(ctx context.Context, itemID string, intent domain.AutonomyIntent) (domain.AutonomyIntent, error) {
	if s.intents == nil {
		return domain.AutonomyIntent{}, domain.AutonomyError{Code: domain.AutonomyErrInvalidInput, Field: "intents", Message: "intent queue service is not configured"}
	}
	item, ok, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		return domain.AutonomyIntent{}, err
	}
	if !ok {
		return domain.AutonomyIntent{}, domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "itemId", Message: "curiosity item not found"}
	}
	if item.Status != domain.CuriosityOpen {
		return domain.AutonomyIntent{}, domain.AutonomyError{Code: domain.AutonomyErrInvalidTransition, Field: "status", Message: "curiosity item is not open"}
	}
	if strings.TrimSpace(intent.ID) == "" {
		intent.ID = "intent-" + shortHash(item.ID, "promoted")
	}
	if strings.TrimSpace(intent.Title) == "" {
		intent.Title = item.Title
	}
	if strings.TrimSpace(intent.Description) == "" {
		intent.Description = item.Question
	}
	if strings.TrimSpace(string(intent.Type)) == "" {
		intent.Type = domain.IntentContextPreparation
	}
	if strings.TrimSpace(string(intent.Source)) == "" {
		intent.Source = domain.IntentSourceForge
	}
	if strings.TrimSpace(intent.ProposedBy) == "" {
		intent.ProposedBy = "curiosity_queue"
	}
	if strings.TrimSpace(intent.Scope.WorkspaceID) == "" {
		intent.Scope = item.Scope
	}
	if strings.TrimSpace(string(intent.Status)) == "" {
		intent.Status = domain.IntentStatusProposed
	}
	intent.Evidence = append(intent.Evidence, item.Evidence...)
	promoted, err := s.intents.Enqueue(ctx, intent)
	if err != nil {
		return domain.AutonomyIntent{}, err
	}
	item.Status = domain.CuriosityPromotedToIntent
	item.UpdatedAt = s.nowMillis()
	item.Metadata = mergeMeta(item.Metadata, map[string]any{"promotedIntentId": promoted.ID})
	if err := s.repo.Update(ctx, item); err != nil {
		return domain.AutonomyIntent{}, err
	}
	return promoted, nil
}

func (s *CuriosityQueueService) Dismiss(ctx context.Context, itemID, reason string) error {
	item, ok, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "itemId", Message: "curiosity item not found"}
	}
	item.Status = domain.CuriosityDismissed
	item.UpdatedAt = s.nowMillis()
	item.Metadata = mergeMeta(item.Metadata, map[string]any{"dismissReason": strings.TrimSpace(reason)})
	return s.repo.Update(ctx, item)
}

func (s *CuriosityQueueService) ExpireOld(ctx context.Context, scope domain.ForgeScope, cutoff int64) (int, error) {
	items, err := s.repo.ListByScope(ctx, scope, domain.CuriosityOpen, 500)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, item := range items {
		if item.ExpiresAt != nil {
			if *item.ExpiresAt > 0 && *item.ExpiresAt <= cutoff {
				item.Status = domain.CuriosityExpired
				item.UpdatedAt = s.nowMillis()
				if err := s.repo.Update(ctx, item); err != nil {
					return updated, err
				}
				updated++
			}
		}
	}
	return updated, nil
}

func (s *CuriosityQueueService) Explain(ctx context.Context, itemID string) (map[string]any, error) {
	item, ok, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "itemId", Message: "curiosity item not found"}
	}
	return map[string]any{
		"id":        item.ID,
		"title":     item.Title,
		"question":  item.Question,
		"source":    item.Source,
		"scope":     item.Scope,
		"evidence":  item.Evidence,
		"priority":  item.Priority,
		"status":    item.Status,
		"createdAt": item.CreatedAt,
		"updatedAt": item.UpdatedAt,
		"expiresAt": item.ExpiresAt,
		"metadata":  item.Metadata,
	}, nil
}
