package autonomy

import (
	"context"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type IntentQueueService struct {
	repo      IntentRepository
	nowMillis func() int64
}

func NewIntentQueueService(repo IntentRepository, nowMillis func() int64) *IntentQueueService {
	if nowMillis == nil {
		nowMillis = domain.NowMillis
	}
	return &IntentQueueService{repo: repo, nowMillis: nowMillis}
}

func (s *IntentQueueService) Enqueue(ctx context.Context, intent domain.AutonomyIntent) (domain.AutonomyIntent, error) {
	if intent.CreatedAt <= 0 {
		intent.CreatedAt = s.nowMillis()
	}
	intent.UpdatedAt = intent.CreatedAt
	if strings.TrimSpace(string(intent.Status)) == "" {
		intent.Status = domain.IntentStatusProposed
	}
	if errs := intent.Validate(); len(errs) > 0 {
		return domain.AutonomyIntent{}, errs[0]
	}
	if err := s.repo.Enqueue(ctx, intent); err != nil {
		return domain.AutonomyIntent{}, err
	}
	return intent, nil
}

func (s *IntentQueueService) Get(ctx context.Context, intentID string) (domain.AutonomyIntent, bool, error) {
	return s.repo.GetByID(ctx, intentID)
}

func (s *IntentQueueService) ListByStatus(ctx context.Context, scope domain.ForgeScope, status domain.IntentStatus, limit int) ([]domain.AutonomyIntent, error) {
	return s.repo.ListByStatus(ctx, scope, status, limit)
}

func (s *IntentQueueService) ListActive(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyIntent, error) {
	return s.repo.ListActive(ctx, scope, limit)
}

func (s *IntentQueueService) Approve(ctx context.Context, intentID string) error {
	return s.transition(ctx, intentID, domain.IntentStatusApproved, "")
}

func (s *IntentQueueService) Reject(ctx context.Context, intentID, reason string) error {
	return s.transition(ctx, intentID, domain.IntentStatusRejected, reason)
}

func (s *IntentQueueService) Cancel(ctx context.Context, intentID, reason string) error {
	return s.transition(ctx, intentID, domain.IntentStatusCancelled, reason)
}

func (s *IntentQueueService) MarkRunning(ctx context.Context, intentID string) error {
	return s.transition(ctx, intentID, domain.IntentStatusRunning, "")
}

func (s *IntentQueueService) MarkCompleted(ctx context.Context, intentID string, results []domain.SyscallResult) error {
	intent, ok, err := s.repo.GetByID(ctx, intentID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "intentId", Message: "intent not found"}
	}
	if !intent.CanTransition(domain.IntentStatusCompleted) {
		return domain.AutonomyError{Code: domain.AutonomyErrInvalidTransition, Field: "status", Message: "intent cannot transition to completed"}
	}
	intent.Status = domain.IntentStatusCompleted
	intent.UpdatedAt = s.nowMillis()
	intent.CommittedActions = append([]domain.SyscallResult{}, results...)
	return s.repo.Update(ctx, intent)
}

func (s *IntentQueueService) MarkBlocked(ctx context.Context, intentID, reason string) error {
	return s.transition(ctx, intentID, domain.IntentStatusBlocked, reason)
}

func (s *IntentQueueService) ExpireOld(ctx context.Context, scope domain.ForgeScope, cutoff int64) (int, error) {
	rows, err := s.repo.ListActive(ctx, scope, 500)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range rows {
		if row.ExpiresAt != nil {
			if *row.ExpiresAt > 0 && *row.ExpiresAt <= cutoff {
				if err := s.transition(ctx, row.ID, domain.IntentStatusExpired, "intent expired"); err != nil {
					return updated, err
				}
				updated++
				continue
			}
		}
		if row.CreatedAt > 0 && row.CreatedAt <= cutoff {
			if err := s.transition(ctx, row.ID, domain.IntentStatusExpired, "intent expired by cutoff"); err != nil {
				return updated, err
			}
			updated++
		}
	}
	return updated, nil
}

func (s *IntentQueueService) Explain(ctx context.Context, intentID string) (map[string]any, error) {
	row, ok, err := s.repo.GetByID(ctx, intentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "intentId", Message: "intent not found"}
	}
	return map[string]any{
		"id":               row.ID,
		"status":           row.Status,
		"type":             row.Type,
		"source":           row.Source,
		"risk":             row.Risk,
		"autonomyLevel":    row.AutonomyLevel,
		"scope":            row.Scope,
		"requiredApproval": row.RequiredApproval,
		"approvalId":       row.ApprovalID,
		"proposedActions":  row.ProposedActions,
		"committedActions": row.CommittedActions,
		"evidence":         row.Evidence,
		"blockedReasons":   row.BlockedReasons,
		"provenance":       row.Provenance,
		"correlationId":    row.CorrelationID,
		"traceId":          row.TraceID,
		"createdAt":        row.CreatedAt,
		"updatedAt":        row.UpdatedAt,
		"expiresAt":        row.ExpiresAt,
		"metadata":         row.Metadata,
	}, nil
}

func (s *IntentQueueService) transition(ctx context.Context, intentID string, next domain.IntentStatus, reason string) error {
	intent, ok, err := s.repo.GetByID(ctx, intentID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "intentId", Message: "intent not found"}
	}
	if !intent.CanTransition(next) {
		return domain.AutonomyError{Code: domain.AutonomyErrInvalidTransition, Field: "status", Message: "invalid intent status transition"}
	}
	intent.Status = next
	intent.UpdatedAt = s.nowMillis()
	if strings.TrimSpace(reason) != "" {
		intent.BlockedReasons = append(intent.BlockedReasons, strings.TrimSpace(reason))
	}
	return s.repo.Update(ctx, intent)
}
