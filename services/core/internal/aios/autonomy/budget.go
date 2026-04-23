package autonomy

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type BudgetCheckRequest struct {
	Scope        domain.ForgeScope
	BudgetID     string
	IntentID     string
	RequestedFor string
	Actions      []domain.SyscallRequest
	DryRun       bool
	Mode         domain.AutonomyMode
}

type BudgetCheckResult struct {
	Allowed  bool
	Decision domain.AutonomyDecisionType
	Reasons  []string
	Budget   *domain.FreedomBudget
	Units    int
}

type FreedomBudgetService struct {
	repo            BudgetRepository
	reservationRepo ReservationRepository
	nowMillis       func() int64
}

func NewFreedomBudgetService(repo BudgetRepository, reservationRepo ReservationRepository, nowMillis func() int64) *FreedomBudgetService {
	if nowMillis == nil {
		nowMillis = domain.NowMillis
	}
	return &FreedomBudgetService{repo: repo, reservationRepo: reservationRepo, nowMillis: nowMillis}
}

func (s *FreedomBudgetService) HasDurableBacking() bool {
	if s == nil || s.repo == nil || s.reservationRepo == nil {
		return false
	}
	if isInMemoryBudgetRepository(s.repo) || isInMemoryReservationRepository(s.reservationRepo) {
		return false
	}
	return true
}

func (s *FreedomBudgetService) CheckBudget(ctx context.Context, req BudgetCheckRequest) (BudgetCheckResult, error) {
	if strings.TrimSpace(req.Scope.WorkspaceID) == "" {
		return BudgetCheckResult{Allowed: false, Decision: domain.DecisionBlockedByScope, Reasons: []string{"missing workspace scope"}}, nil
	}
	budget, ok, err := s.resolveBudget(ctx, req)
	if err != nil {
		return BudgetCheckResult{}, err
	}
	if !ok {
		return BudgetCheckResult{Allowed: false, Decision: domain.DecisionBlockedByBudget, Reasons: []string{"no active budget for scope"}}, nil
	}
	if budget.Status != domain.BudgetStatusActive {
		return BudgetCheckResult{Allowed: false, Decision: domain.DecisionBlockedByBudget, Reasons: []string{"budget is not active"}, Budget: &budget}, nil
	}

	if s.shouldReset(budget) {
		budget.Usage = domain.FreedomBudgetUsage{}
		budget.UpdatedAt = s.nowMillis()
		budget.ResetsAt = s.nextResetAt(budget)
		if err := s.repo.Update(ctx, budget); err != nil {
			return BudgetCheckResult{}, err
		}
	}

	units := len(req.Actions)
	if units == 0 {
		units = 1
	}
	reasons := []string{}
	if overLimit(budget.MaxSelfActionsPerRun, budget.Usage.SelfActionsPerRun+units) {
		reasons = append(reasons, "maxSelfActionsPerRun exceeded")
	}
	if overLimit(budget.MaxRunsPerPeriod, budget.Usage.RunsPerPeriod+1) {
		reasons = append(reasons, "maxRunsPerPeriod exceeded")
	}
	if overLimit(budget.MaxProposedActionsPerPeriod, budget.Usage.ProposedActionsPeriod+len(req.Actions)) {
		reasons = append(reasons, "maxProposedActionsPerPeriod exceeded")
	}
	if !req.DryRun && overLimit(budget.MaxCommittedActionsPerPeriod, budget.Usage.CommittedActionsPeriod+len(req.Actions)) {
		reasons = append(reasons, "maxCommittedActionsPerPeriod exceeded")
	}
	archiveActions := countAction(req.Actions, domain.ActionArchiveNote)
	if archiveActions > 0 && overLimit(budget.MaxArchiveActions, budget.Usage.ArchiveActions+archiveActions) {
		reasons = append(reasons, "maxArchiveActions exceeded")
	}
	compileActions := countAction(req.Actions, domain.ActionCompileContext)
	if compileActions > 0 && overLimit(budget.MaxContextPrecompilations, budget.Usage.ContextPrecompilations+compileActions) {
		reasons = append(reasons, "maxContextPrecompilations exceeded")
	}
	if budget.MaxCostUnits != nil && *budget.MaxCostUnits > 0 {
		next := budget.Usage.CostUnits + units
		if next > *budget.MaxCostUnits {
			reasons = append(reasons, "maxCostUnits exceeded")
		}
	}
	if len(reasons) > 0 {
		return BudgetCheckResult{Allowed: false, Decision: domain.DecisionBlockedByBudget, Reasons: reasons, Budget: &budget, Units: units}, nil
	}
	return BudgetCheckResult{Allowed: true, Decision: domain.DecisionAllowAutoCommit, Reasons: nil, Budget: &budget, Units: units}, nil
}

func (s *FreedomBudgetService) ReserveBudget(ctx context.Context, budgetID, intentID string, scope domain.ForgeScope, requestedFor string, units int, metadata map[string]any) (domain.BudgetReservation, error) {
	if s.reservationRepo == nil {
		return domain.BudgetReservation{}, domain.AutonomyError{Code: domain.AutonomyErrBudgetDenied, Field: "reservation", Message: "reservation repository is not configured"}
	}
	if units <= 0 {
		units = 1
	}
	now := s.nowMillis()
	reservation := domain.BudgetReservation{
		ID:           "reserve-" + shortHash(budgetID, intentID, fmt.Sprintf("%d", now)),
		BudgetID:     strings.TrimSpace(budgetID),
		IntentID:     strings.TrimSpace(intentID),
		Scope:        scope,
		RequestedFor: strings.TrimSpace(requestedFor),
		Units:        units,
		CreatedAt:    now,
		Consumed:     false,
		Released:     false,
		Metadata:     cloneMeta(metadata),
	}
	if err := s.reservationRepo.Create(ctx, reservation); err != nil {
		return domain.BudgetReservation{}, err
	}
	return reservation, nil
}

func (s *FreedomBudgetService) ConsumeBudget(ctx context.Context, reservationID string) (domain.FreedomBudget, error) {
	if s.reservationRepo == nil || s.repo == nil {
		return domain.FreedomBudget{}, domain.AutonomyError{Code: domain.AutonomyErrBudgetDenied, Field: "reservation", Message: "budget repositories are not configured"}
	}
	reservation, ok, err := s.reservationRepo.GetByID(ctx, strings.TrimSpace(reservationID))
	if err != nil {
		return domain.FreedomBudget{}, err
	}
	if !ok {
		return domain.FreedomBudget{}, domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "reservationId", Message: "budget reservation not found"}
	}
	if reservation.Released {
		return domain.FreedomBudget{}, domain.AutonomyError{Code: domain.AutonomyErrBudgetDenied, Field: "reservationId", Message: "budget reservation already released"}
	}
	if reservation.Consumed {
		budget, ok, err := s.repo.GetByID(ctx, reservation.BudgetID)
		if err != nil {
			return domain.FreedomBudget{}, err
		}
		if !ok {
			return domain.FreedomBudget{}, domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "budgetId", Message: "budget not found"}
		}
		return budget, nil
	}
	budget, ok, err := s.repo.GetByID(ctx, reservation.BudgetID)
	if err != nil {
		return domain.FreedomBudget{}, err
	}
	if !ok {
		return domain.FreedomBudget{}, domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "budgetId", Message: "budget not found"}
	}
	budget.Usage.SelfActionsPerRun += reservation.Units
	budget.Usage.RunsPerPeriod += 1
	budget.Usage.CommittedActionsPeriod += reservation.Units
	budget.Usage.ProposedActionsPeriod += reservation.Units
	budget.Usage.CostUnits += reservation.Units
	if strings.Contains(strings.ToLower(reservation.RequestedFor), "archive") {
		budget.Usage.ArchiveActions += reservation.Units
	}
	if strings.Contains(strings.ToLower(reservation.RequestedFor), "context") {
		budget.Usage.ContextPrecompilations += reservation.Units
	}
	budget.UpdatedAt = s.nowMillis()
	if err := s.repo.Update(ctx, budget); err != nil {
		return domain.FreedomBudget{}, err
	}
	reservation.Consumed = true
	if err := s.reservationRepo.Update(ctx, reservation); err != nil {
		return domain.FreedomBudget{}, err
	}
	return budget, nil
}

func (s *FreedomBudgetService) ReleaseBudget(ctx context.Context, reservationID string, reason string) error {
	if s.reservationRepo == nil {
		return nil
	}
	reservation, ok, err := s.reservationRepo.GetByID(ctx, strings.TrimSpace(reservationID))
	if err != nil || !ok {
		return err
	}
	if reservation.Consumed {
		return nil
	}
	reservation.Released = true
	reservation.Metadata = mergeMeta(reservation.Metadata, map[string]any{"releasedReason": strings.TrimSpace(reason), "releasedAt": s.nowMillis()})
	return s.reservationRepo.Update(ctx, reservation)
}

func (s *FreedomBudgetService) GetUsage(ctx context.Context, scope domain.ForgeScope) ([]domain.FreedomBudget, error) {
	if s.repo == nil {
		return nil, nil
	}
	rows, err := s.repo.ListByScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows, nil
}

func (s *FreedomBudgetService) ResetExpiredBudgets(ctx context.Context, now int64) ([]domain.FreedomBudget, error) {
	if s.repo == nil {
		return nil, nil
	}
	if now <= 0 {
		now = s.nowMillis()
	}
	// no global list API exists by design; rely on caller scope loops and explicit IDs for now.
	return nil, nil
}

func (s *FreedomBudgetService) ExplainBudgetDecision(req BudgetCheckRequest, res BudgetCheckResult) map[string]any {
	out := map[string]any{
		"scope":        req.Scope,
		"budgetId":     req.BudgetID,
		"intentId":     req.IntentID,
		"requestedFor": req.RequestedFor,
		"units":        res.Units,
		"decision":     res.Decision,
		"allowed":      res.Allowed,
		"reasons":      append([]string{}, res.Reasons...),
	}
	if res.Budget != nil {
		out["budgetStatus"] = res.Budget.Status
		out["budgetUsage"] = res.Budget.Usage
		out["resetsAt"] = res.Budget.ResetsAt
	}
	return out
}

func (s *FreedomBudgetService) resolveBudget(ctx context.Context, req BudgetCheckRequest) (domain.FreedomBudget, bool, error) {
	if s.repo == nil {
		return domain.FreedomBudget{}, false, nil
	}
	if strings.TrimSpace(req.BudgetID) != "" {
		return s.repo.GetByID(ctx, strings.TrimSpace(req.BudgetID))
	}
	rows, err := s.repo.ListByScope(ctx, req.Scope)
	if err != nil {
		return domain.FreedomBudget{}, false, err
	}
	now := s.nowMillis()
	for _, row := range rows {
		if row.Status != domain.BudgetStatusActive {
			continue
		}
		if row.ResetsAt > 0 && now > row.ResetsAt {
			continue
		}
		return row, true, nil
	}
	for _, row := range rows {
		if row.Status == domain.BudgetStatusActive {
			return row, true, nil
		}
	}
	return domain.FreedomBudget{}, false, nil
}

func (s *FreedomBudgetService) shouldReset(budget domain.FreedomBudget) bool {
	if budget.ResetsAt <= 0 {
		return false
	}
	return s.nowMillis() > budget.ResetsAt
}

func (s *FreedomBudgetService) nextResetAt(budget domain.FreedomBudget) int64 {
	now := s.nowMillis()
	switch budget.Period {
	case domain.BudgetPeriodPerRun:
		return now
	case domain.BudgetPeriodHourly:
		return now + int64(60*60*1000)
	case domain.BudgetPeriodDaily:
		return now + int64(24*60*60*1000)
	case domain.BudgetPeriodWeekly:
		return now + int64(7*24*60*60*1000)
	case domain.BudgetPeriodMission:
		return budget.ResetsAt
	default:
		return now + int64(24*60*60*1000)
	}
}

func isInMemoryBudgetRepository(repo BudgetRepository) bool {
	switch repo.(type) {
	case *InMemoryBudgetRepository:
		return true
	default:
		return false
	}
}

func isInMemoryReservationRepository(repo ReservationRepository) bool {
	switch repo.(type) {
	case *InMemoryReservationRepository:
		return true
	default:
		return false
	}
}

func countAction(actions []domain.SyscallRequest, action domain.SemanticActionType) int {
	n := 0
	for _, item := range actions {
		if item.Action == action {
			n++
		}
	}
	return n
}

func overLimit(limit int, usage int) bool {
	if limit <= 0 {
		return false
	}
	return usage > limit
}

func cloneMeta(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
