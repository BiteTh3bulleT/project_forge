package autonomy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"forge/projectforge/services/core/internal/aios/domain"
)

type CharterRepository interface {
	Create(ctx context.Context, charter domain.AutonomyCharter) error
	GetByID(ctx context.Context, id string) (domain.AutonomyCharter, bool, error)
	ListByScope(ctx context.Context, scope domain.ForgeScope) ([]domain.AutonomyCharter, error)
	ListActiveByScope(ctx context.Context, scope domain.ForgeScope, now int64) ([]domain.AutonomyCharter, error)
	UpdateStatus(ctx context.Context, id string, status domain.CharterStatus, updatedAt int64, metadata map[string]any) error
	FindApplicable(ctx context.Context, scope domain.ForgeScope, source domain.IntentSource, action domain.SemanticActionType, now int64) ([]domain.AutonomyCharter, error)
}

type IntentRepository interface {
	Enqueue(ctx context.Context, intent domain.AutonomyIntent) error
	GetByID(ctx context.Context, id string) (domain.AutonomyIntent, bool, error)
	Update(ctx context.Context, intent domain.AutonomyIntent) error
	UpdateStatus(ctx context.Context, id string, status domain.IntentStatus, reason string, updatedAt int64) error
	ListByStatus(ctx context.Context, scope domain.ForgeScope, status domain.IntentStatus, limit int) ([]domain.AutonomyIntent, error)
	ListActive(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyIntent, error)
	ListByScope(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyIntent, error)
	ListByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.AutonomyIntent, error)
}

type BudgetRepository interface {
	Create(ctx context.Context, budget domain.FreedomBudget) error
	GetByID(ctx context.Context, id string) (domain.FreedomBudget, bool, error)
	ListByScope(ctx context.Context, scope domain.ForgeScope) ([]domain.FreedomBudget, error)
	Update(ctx context.Context, budget domain.FreedomBudget) error
	UpdateStatus(ctx context.Context, id string, status domain.FreedomBudgetStatus, updatedAt int64, metadata map[string]any) error
}

type DecisionRepository interface {
	Create(ctx context.Context, decision domain.AutonomyDecision) error
	ListByIntent(ctx context.Context, intentID string, limit int) ([]domain.AutonomyDecision, error)
	ListByScope(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyDecision, error)
	ListByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.AutonomyDecision, error)
}

type ReservationRepository interface {
	Create(ctx context.Context, reservation domain.BudgetReservation) error
	GetByID(ctx context.Context, id string) (domain.BudgetReservation, bool, error)
	Update(ctx context.Context, reservation domain.BudgetReservation) error
}

type CuriosityRepository interface {
	Create(ctx context.Context, item domain.CuriosityItem) error
	GetByID(ctx context.Context, id string) (domain.CuriosityItem, bool, error)
	Update(ctx context.Context, item domain.CuriosityItem) error
	ListByScope(ctx context.Context, scope domain.ForgeScope, status domain.CuriosityStatus, limit int) ([]domain.CuriosityItem, error)
}

type InMemoryStore struct {
	mu           sync.RWMutex
	charters     map[string]domain.AutonomyCharter
	intents      map[string]domain.AutonomyIntent
	budgets      map[string]domain.FreedomBudget
	decisions    map[string]domain.AutonomyDecision
	reservations map[string]domain.BudgetReservation
	curiosity    map[string]domain.CuriosityItem
}

type InMemoryBundle struct {
	Store        *InMemoryStore
	Charters     *InMemoryCharterRepository
	Intents      *InMemoryIntentRepository
	Budgets      *InMemoryBudgetRepository
	Decisions    *InMemoryDecisionRepository
	Reservations *InMemoryReservationRepository
	Curiosity    *InMemoryCuriosityRepository
}

type InMemoryCharterRepository struct{ store *InMemoryStore }
type InMemoryIntentRepository struct{ store *InMemoryStore }
type InMemoryBudgetRepository struct{ store *InMemoryStore }
type InMemoryDecisionRepository struct{ store *InMemoryStore }
type InMemoryReservationRepository struct{ store *InMemoryStore }
type InMemoryCuriosityRepository struct{ store *InMemoryStore }

var (
	_ CharterRepository     = (*InMemoryCharterRepository)(nil)
	_ IntentRepository      = (*InMemoryIntentRepository)(nil)
	_ BudgetRepository      = (*InMemoryBudgetRepository)(nil)
	_ DecisionRepository    = (*InMemoryDecisionRepository)(nil)
	_ ReservationRepository = (*InMemoryReservationRepository)(nil)
	_ CuriosityRepository   = (*InMemoryCuriosityRepository)(nil)
)

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		charters:     map[string]domain.AutonomyCharter{},
		intents:      map[string]domain.AutonomyIntent{},
		budgets:      map[string]domain.FreedomBudget{},
		decisions:    map[string]domain.AutonomyDecision{},
		reservations: map[string]domain.BudgetReservation{},
		curiosity:    map[string]domain.CuriosityItem{},
	}
}

func NewInMemoryBundle() InMemoryBundle {
	store := NewInMemoryStore()
	return InMemoryBundle{
		Store:        store,
		Charters:     &InMemoryCharterRepository{store: store},
		Intents:      &InMemoryIntentRepository{store: store},
		Budgets:      &InMemoryBudgetRepository{store: store},
		Decisions:    &InMemoryDecisionRepository{store: store},
		Reservations: &InMemoryReservationRepository{store: store},
		Curiosity:    &InMemoryCuriosityRepository{store: store},
	}
}

func (r *InMemoryCharterRepository) Create(_ context.Context, charter domain.AutonomyCharter) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(charter.ID)
	if id == "" {
		return fmt.Errorf("charter id is required")
	}
	if _, exists := r.store.charters[id]; exists {
		return fmt.Errorf("charter %q already exists", id)
	}
	r.store.charters[id] = charter
	return nil
}

func (r *InMemoryCharterRepository) GetByID(_ context.Context, id string) (domain.AutonomyCharter, bool, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	row, ok := r.store.charters[strings.TrimSpace(id)]
	return row, ok, nil
}

func (r *InMemoryCharterRepository) ListByScope(_ context.Context, scope domain.ForgeScope) ([]domain.AutonomyCharter, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]domain.AutonomyCharter, 0)
	for _, row := range r.store.charters {
		if scopeMatch(scope, row.Scope) {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out, nil
}

func (r *InMemoryCharterRepository) ListActiveByScope(ctx context.Context, scope domain.ForgeScope, now int64) ([]domain.AutonomyCharter, error) {
	rows, err := r.ListByScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AutonomyCharter, 0, len(rows))
	for _, row := range rows {
		if row.IsActive(now) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (r *InMemoryCharterRepository) UpdateStatus(_ context.Context, id string, status domain.CharterStatus, updatedAt int64, metadata map[string]any) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	key := strings.TrimSpace(id)
	row, ok := r.store.charters[key]
	if !ok {
		return fmt.Errorf("charter %q not found", key)
	}
	row.Status = status
	row.UpdatedAt = updatedAt
	row.Metadata = mergeMeta(row.Metadata, metadata)
	r.store.charters[key] = row
	return nil
}

func (r *InMemoryCharterRepository) FindApplicable(ctx context.Context, scope domain.ForgeScope, source domain.IntentSource, action domain.SemanticActionType, now int64) ([]domain.AutonomyCharter, error) {
	rows, err := r.ListActiveByScope(ctx, scope, now)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AutonomyCharter, 0)
	for _, row := range rows {
		if len(row.AllowedSources) > 0 && !containsSource(row.AllowedSources, source) {
			continue
		}
		if row.DeniesAction(action) {
			continue
		}
		if len(row.AllowedActions) > 0 && !row.AllowsAction(action) {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *InMemoryIntentRepository) Enqueue(_ context.Context, intent domain.AutonomyIntent) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(intent.ID)
	if id == "" {
		return fmt.Errorf("intent id is required")
	}
	if _, exists := r.store.intents[id]; exists {
		return fmt.Errorf("intent %q already exists", id)
	}
	r.store.intents[id] = intent
	return nil
}

func (r *InMemoryIntentRepository) GetByID(_ context.Context, id string) (domain.AutonomyIntent, bool, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	row, ok := r.store.intents[strings.TrimSpace(id)]
	return row, ok, nil
}

func (r *InMemoryIntentRepository) Update(_ context.Context, intent domain.AutonomyIntent) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(intent.ID)
	if id == "" {
		return fmt.Errorf("intent id is required")
	}
	if _, exists := r.store.intents[id]; !exists {
		return fmt.Errorf("intent %q not found", id)
	}
	r.store.intents[id] = intent
	return nil
}

func (r *InMemoryIntentRepository) UpdateStatus(_ context.Context, id string, status domain.IntentStatus, reason string, updatedAt int64) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	key := strings.TrimSpace(id)
	intent, ok := r.store.intents[key]
	if !ok {
		return fmt.Errorf("intent %q not found", key)
	}
	intent.Status = status
	intent.UpdatedAt = updatedAt
	if strings.TrimSpace(reason) != "" {
		intent.BlockedReasons = append(intent.BlockedReasons, strings.TrimSpace(reason))
	}
	r.store.intents[key] = intent
	return nil
}

func (r *InMemoryIntentRepository) ListByStatus(ctx context.Context, scope domain.ForgeScope, status domain.IntentStatus, limit int) ([]domain.AutonomyIntent, error) {
	rows, err := r.ListByScope(ctx, scope, limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AutonomyIntent, 0, len(rows))
	for _, row := range rows {
		if row.Status == status {
			out = append(out, row)
		}
	}
	return limitSlice(out, limit), nil
}

func (r *InMemoryIntentRepository) ListActive(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyIntent, error) {
	rows, err := r.ListByScope(ctx, scope, limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AutonomyIntent, 0, len(rows))
	for _, row := range rows {
		switch row.Status {
		case domain.IntentStatusProposed, domain.IntentStatusApproved, domain.IntentStatusRunning, domain.IntentStatusBlocked:
			out = append(out, row)
		}
	}
	return limitSlice(out, limit), nil
}

func (r *InMemoryIntentRepository) ListByScope(_ context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyIntent, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]domain.AutonomyIntent, 0, len(r.store.intents))
	for _, row := range r.store.intents {
		if scopeMatch(scope, row.Scope) {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *InMemoryIntentRepository) ListByCorrelation(_ context.Context, correlationID string, limit int) ([]domain.AutonomyIntent, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]domain.AutonomyIntent, 0)
	correlationID = strings.TrimSpace(correlationID)
	for _, row := range r.store.intents {
		if strings.TrimSpace(row.CorrelationID) == correlationID {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *InMemoryBudgetRepository) Create(_ context.Context, budget domain.FreedomBudget) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(budget.ID)
	if id == "" {
		return fmt.Errorf("budget id is required")
	}
	if _, exists := r.store.budgets[id]; exists {
		return fmt.Errorf("budget %q already exists", id)
	}
	r.store.budgets[id] = budget
	return nil
}

func (r *InMemoryBudgetRepository) GetByID(_ context.Context, id string) (domain.FreedomBudget, bool, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	row, ok := r.store.budgets[strings.TrimSpace(id)]
	return row, ok, nil
}

func (r *InMemoryBudgetRepository) ListByScope(_ context.Context, scope domain.ForgeScope) ([]domain.FreedomBudget, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]domain.FreedomBudget, 0)
	for _, row := range r.store.budgets {
		if scopeMatch(scope, row.Scope) {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out, nil
}

func (r *InMemoryBudgetRepository) Update(_ context.Context, budget domain.FreedomBudget) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(budget.ID)
	if id == "" {
		return fmt.Errorf("budget id is required")
	}
	if _, exists := r.store.budgets[id]; !exists {
		return fmt.Errorf("budget %q not found", id)
	}
	r.store.budgets[id] = budget
	return nil
}

func (r *InMemoryBudgetRepository) UpdateStatus(_ context.Context, id string, status domain.FreedomBudgetStatus, updatedAt int64, metadata map[string]any) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	key := strings.TrimSpace(id)
	row, ok := r.store.budgets[key]
	if !ok {
		return fmt.Errorf("budget %q not found", key)
	}
	row.Status = status
	row.UpdatedAt = updatedAt
	row.Metadata = mergeMeta(row.Metadata, metadata)
	r.store.budgets[key] = row
	return nil
}

func (r *InMemoryDecisionRepository) Create(_ context.Context, decision domain.AutonomyDecision) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(decision.ID)
	if id == "" {
		return fmt.Errorf("decision id is required")
	}
	if _, exists := r.store.decisions[id]; exists {
		return fmt.Errorf("decision %q already exists", id)
	}
	r.store.decisions[id] = decision
	return nil
}

func (r *InMemoryDecisionRepository) ListByIntent(_ context.Context, intentID string, limit int) ([]domain.AutonomyDecision, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]domain.AutonomyDecision, 0)
	intentID = strings.TrimSpace(intentID)
	for _, row := range r.store.decisions {
		if strings.TrimSpace(row.IntentID) == intentID {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *InMemoryDecisionRepository) ListByScope(_ context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyDecision, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]domain.AutonomyDecision, 0)
	for _, row := range r.store.decisions {
		if actionScopeMatch(scope, row.AllowedActions, row.BlockedActions) {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *InMemoryDecisionRepository) ListByCorrelation(_ context.Context, correlationID string, limit int) ([]domain.AutonomyDecision, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]domain.AutonomyDecision, 0)
	correlationID = strings.TrimSpace(correlationID)
	for _, row := range r.store.decisions {
		if strings.TrimSpace(row.CorrelationID) == correlationID {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *InMemoryReservationRepository) Create(_ context.Context, reservation domain.BudgetReservation) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(reservation.ID)
	if id == "" {
		return fmt.Errorf("reservation id is required")
	}
	if _, exists := r.store.reservations[id]; exists {
		return fmt.Errorf("reservation %q already exists", id)
	}
	r.store.reservations[id] = reservation
	return nil
}

func (r *InMemoryReservationRepository) GetByID(_ context.Context, id string) (domain.BudgetReservation, bool, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	row, ok := r.store.reservations[strings.TrimSpace(id)]
	return row, ok, nil
}

func (r *InMemoryReservationRepository) Update(_ context.Context, reservation domain.BudgetReservation) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(reservation.ID)
	if id == "" {
		return fmt.Errorf("reservation id is required")
	}
	if _, exists := r.store.reservations[id]; !exists {
		return fmt.Errorf("reservation %q not found", id)
	}
	r.store.reservations[id] = reservation
	return nil
}

func (r *InMemoryCuriosityRepository) Create(_ context.Context, item domain.CuriosityItem) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return fmt.Errorf("curiosity id is required")
	}
	if _, exists := r.store.curiosity[id]; exists {
		return fmt.Errorf("curiosity %q already exists", id)
	}
	r.store.curiosity[id] = item
	return nil
}

func (r *InMemoryCuriosityRepository) GetByID(_ context.Context, id string) (domain.CuriosityItem, bool, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	row, ok := r.store.curiosity[strings.TrimSpace(id)]
	return row, ok, nil
}

func (r *InMemoryCuriosityRepository) Update(_ context.Context, item domain.CuriosityItem) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return fmt.Errorf("curiosity id is required")
	}
	if _, exists := r.store.curiosity[id]; !exists {
		return fmt.Errorf("curiosity %q not found", id)
	}
	r.store.curiosity[id] = item
	return nil
}

func (r *InMemoryCuriosityRepository) ListByScope(_ context.Context, scope domain.ForgeScope, status domain.CuriosityStatus, limit int) ([]domain.CuriosityItem, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	out := make([]domain.CuriosityItem, 0)
	for _, row := range r.store.curiosity {
		if !scopeMatch(scope, row.Scope) {
			continue
		}
		if status != "" && row.Status != status {
			continue
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func scopeMatch(expected, actual domain.ForgeScope) bool {
	if strings.TrimSpace(expected.WorkspaceID) == "" {
		return false
	}
	if strings.TrimSpace(expected.WorkspaceID) != strings.TrimSpace(actual.WorkspaceID) {
		return false
	}
	expectedLane := strings.TrimSpace(expected.LaneID)
	actualLane := strings.TrimSpace(actual.LaneID)
	if expectedLane == "" || actualLane == "" {
		return true
	}
	return expectedLane == actualLane
}

func actionScopeMatch(scope domain.ForgeScope, allowed []domain.SyscallRequest, blocked []domain.SyscallRequest) bool {
	for _, action := range allowed {
		if scopeMatch(scope, action.Scope) {
			return true
		}
	}
	for _, action := range blocked {
		if scopeMatch(scope, action.Scope) {
			return true
		}
	}
	return false
}

func containsSource(items []domain.IntentSource, source domain.IntentSource) bool {
	for _, item := range items {
		if item == source {
			return true
		}
	}
	return false
}

func limitSlice[T any](items []T, limit int) []T {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}

func mergeMeta(base map[string]any, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
