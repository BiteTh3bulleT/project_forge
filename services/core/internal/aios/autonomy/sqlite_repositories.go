package autonomy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	sqliteAutonomyPrefix       = "autonomy_repo."
	sqliteAutonomyCharterPref  = sqliteAutonomyPrefix + "charter."
	sqliteAutonomyIntentPref   = sqliteAutonomyPrefix + "intent."
	sqliteAutonomyBudgetPref   = sqliteAutonomyPrefix + "budget."
	sqliteAutonomyDecisionPref = sqliteAutonomyPrefix + "decision."
	sqliteAutonomyReservePref  = sqliteAutonomyPrefix + "reservation."
	sqliteAutonomyCuriosityPref = sqliteAutonomyPrefix + "curiosity."
)

type SQLiteBundle struct {
	Charters     CharterRepository
	Intents      IntentRepository
	Budgets      BudgetRepository
	Decisions    DecisionRepository
	Reservations ReservationRepository
	Curiosity    CuriosityRepository
}

type sqliteKVRepository struct {
	db *sql.DB
}

type SQLiteCharterRepository struct{ kv *sqliteKVRepository }
type SQLiteIntentRepository struct{ kv *sqliteKVRepository }
type SQLiteBudgetRepository struct{ kv *sqliteKVRepository }
type SQLiteDecisionRepository struct{ kv *sqliteKVRepository }
type SQLiteReservationRepository struct{ kv *sqliteKVRepository }
type SQLiteCuriosityRepository struct{ kv *sqliteKVRepository }

var (
	_ CharterRepository     = (*SQLiteCharterRepository)(nil)
	_ IntentRepository      = (*SQLiteIntentRepository)(nil)
	_ BudgetRepository      = (*SQLiteBudgetRepository)(nil)
	_ DecisionRepository    = (*SQLiteDecisionRepository)(nil)
	_ ReservationRepository = (*SQLiteReservationRepository)(nil)
	_ CuriosityRepository   = (*SQLiteCuriosityRepository)(nil)
)

func NewSQLiteBundle(db *sql.DB) SQLiteBundle {
	if db == nil {
		mem := NewInMemoryBundle()
		return SQLiteBundle{
			Charters:     mem.Charters,
			Intents:      mem.Intents,
			Budgets:      mem.Budgets,
			Decisions:    mem.Decisions,
			Reservations: mem.Reservations,
			Curiosity:    mem.Curiosity,
		}
	}
	kv := &sqliteKVRepository{db: db}
	return SQLiteBundle{
		Charters:     &SQLiteCharterRepository{kv: kv},
		Intents:      &SQLiteIntentRepository{kv: kv},
		Budgets:      &SQLiteBudgetRepository{kv: kv},
		Decisions:    &SQLiteDecisionRepository{kv: kv},
		Reservations: &SQLiteReservationRepository{kv: kv},
		Curiosity:    &SQLiteCuriosityRepository{kv: kv},
	}
}

func (r *sqliteKVRepository) get(ctx context.Context, key string) (string, bool, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&raw)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return raw, true, nil
}

func (r *sqliteKVRepository) upsert(ctx context.Context, key string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO settings(key, value) VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value
`, key, string(payload))
	return err
}

func (r *sqliteKVRepository) create(ctx context.Context, key string, value any) error {
	_, exists, err := r.get(ctx, key)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("record %q already exists", key)
	}
	return r.upsert(ctx, key, value)
}

func (r *sqliteKVRepository) listByPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM settings WHERE key LIKE ?`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var key string
		var raw string
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		out[key] = raw
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func keyFor(prefix, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("id is required")
	}
	return prefix + id, nil
}

func decodeRow[T any](raw string) (T, error) {
	var out T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return out, err
	}
	return out, nil
}

func (r *SQLiteCharterRepository) Create(ctx context.Context, charter domain.AutonomyCharter) error {
	key, err := keyFor(sqliteAutonomyCharterPref, charter.ID)
	if err != nil {
		return fmt.Errorf("charter id is required")
	}
	return r.kv.create(ctx, key, charter)
}

func (r *SQLiteCharterRepository) GetByID(ctx context.Context, id string) (domain.AutonomyCharter, bool, error) {
	key, err := keyFor(sqliteAutonomyCharterPref, id)
	if err != nil {
		return domain.AutonomyCharter{}, false, nil
	}
	raw, ok, err := r.kv.get(ctx, key)
	if err != nil || !ok {
		return domain.AutonomyCharter{}, ok, err
	}
	row, err := decodeRow[domain.AutonomyCharter](raw)
	if err != nil {
		return domain.AutonomyCharter{}, false, err
	}
	return row, true, nil
}

func (r *SQLiteCharterRepository) ListByScope(ctx context.Context, scope domain.ForgeScope) ([]domain.AutonomyCharter, error) {
	items, err := r.kv.listByPrefix(ctx, sqliteAutonomyCharterPref)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AutonomyCharter, 0, len(items))
	for _, raw := range items {
		row, err := decodeRow[domain.AutonomyCharter](raw)
		if err != nil {
			return nil, err
		}
		if scopeMatch(scope, row.Scope) {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out, nil
}

func (r *SQLiteCharterRepository) ListActiveByScope(ctx context.Context, scope domain.ForgeScope, now int64) ([]domain.AutonomyCharter, error) {
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

func (r *SQLiteCharterRepository) UpdateStatus(ctx context.Context, id string, status domain.CharterStatus, updatedAt int64, metadata map[string]any) error {
	row, ok, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("charter %q not found", strings.TrimSpace(id))
	}
	row.Status = status
	row.UpdatedAt = updatedAt
	row.Metadata = mergeMeta(row.Metadata, metadata)
	key, _ := keyFor(sqliteAutonomyCharterPref, row.ID)
	return r.kv.upsert(ctx, key, row)
}

func (r *SQLiteCharterRepository) FindApplicable(ctx context.Context, scope domain.ForgeScope, source domain.IntentSource, action domain.SemanticActionType, now int64) ([]domain.AutonomyCharter, error) {
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

func (r *SQLiteIntentRepository) Enqueue(ctx context.Context, intent domain.AutonomyIntent) error {
	key, err := keyFor(sqliteAutonomyIntentPref, intent.ID)
	if err != nil {
		return fmt.Errorf("intent id is required")
	}
	return r.kv.create(ctx, key, intent)
}

func (r *SQLiteIntentRepository) GetByID(ctx context.Context, id string) (domain.AutonomyIntent, bool, error) {
	key, err := keyFor(sqliteAutonomyIntentPref, id)
	if err != nil {
		return domain.AutonomyIntent{}, false, nil
	}
	raw, ok, err := r.kv.get(ctx, key)
	if err != nil || !ok {
		return domain.AutonomyIntent{}, ok, err
	}
	row, err := decodeRow[domain.AutonomyIntent](raw)
	if err != nil {
		return domain.AutonomyIntent{}, false, err
	}
	return row, true, nil
}

func (r *SQLiteIntentRepository) Update(ctx context.Context, intent domain.AutonomyIntent) error {
	key, err := keyFor(sqliteAutonomyIntentPref, intent.ID)
	if err != nil {
		return fmt.Errorf("intent id is required")
	}
	_, ok, err := r.kv.get(ctx, key)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("intent %q not found", strings.TrimSpace(intent.ID))
	}
	return r.kv.upsert(ctx, key, intent)
}

func (r *SQLiteIntentRepository) UpdateStatus(ctx context.Context, id string, status domain.IntentStatus, reason string, updatedAt int64) error {
	intent, ok, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("intent %q not found", strings.TrimSpace(id))
	}
	intent.Status = status
	intent.UpdatedAt = updatedAt
	if strings.TrimSpace(reason) != "" {
		intent.BlockedReasons = append(intent.BlockedReasons, strings.TrimSpace(reason))
	}
	key, _ := keyFor(sqliteAutonomyIntentPref, intent.ID)
	return r.kv.upsert(ctx, key, intent)
}

func (r *SQLiteIntentRepository) ListByStatus(ctx context.Context, scope domain.ForgeScope, status domain.IntentStatus, limit int) ([]domain.AutonomyIntent, error) {
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

func (r *SQLiteIntentRepository) ListActive(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyIntent, error) {
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

func (r *SQLiteIntentRepository) ListByScope(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyIntent, error) {
	items, err := r.kv.listByPrefix(ctx, sqliteAutonomyIntentPref)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AutonomyIntent, 0, len(items))
	for _, raw := range items {
		row, err := decodeRow[domain.AutonomyIntent](raw)
		if err != nil {
			return nil, err
		}
		if scopeMatch(scope, row.Scope) {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *SQLiteIntentRepository) ListByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.AutonomyIntent, error) {
	items, err := r.kv.listByPrefix(ctx, sqliteAutonomyIntentPref)
	if err != nil {
		return nil, err
	}
	correlationID = strings.TrimSpace(correlationID)
	out := make([]domain.AutonomyIntent, 0)
	for _, raw := range items {
		row, err := decodeRow[domain.AutonomyIntent](raw)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(row.CorrelationID) == correlationID {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *SQLiteBudgetRepository) Create(ctx context.Context, budget domain.FreedomBudget) error {
	key, err := keyFor(sqliteAutonomyBudgetPref, budget.ID)
	if err != nil {
		return fmt.Errorf("budget id is required")
	}
	return r.kv.create(ctx, key, budget)
}

func (r *SQLiteBudgetRepository) GetByID(ctx context.Context, id string) (domain.FreedomBudget, bool, error) {
	key, err := keyFor(sqliteAutonomyBudgetPref, id)
	if err != nil {
		return domain.FreedomBudget{}, false, nil
	}
	raw, ok, err := r.kv.get(ctx, key)
	if err != nil || !ok {
		return domain.FreedomBudget{}, ok, err
	}
	row, err := decodeRow[domain.FreedomBudget](raw)
	if err != nil {
		return domain.FreedomBudget{}, false, err
	}
	return row, true, nil
}

func (r *SQLiteBudgetRepository) ListByScope(ctx context.Context, scope domain.ForgeScope) ([]domain.FreedomBudget, error) {
	items, err := r.kv.listByPrefix(ctx, sqliteAutonomyBudgetPref)
	if err != nil {
		return nil, err
	}
	out := make([]domain.FreedomBudget, 0, len(items))
	for _, raw := range items {
		row, err := decodeRow[domain.FreedomBudget](raw)
		if err != nil {
			return nil, err
		}
		if scopeMatch(scope, row.Scope) {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out, nil
}

func (r *SQLiteBudgetRepository) Update(ctx context.Context, budget domain.FreedomBudget) error {
	key, err := keyFor(sqliteAutonomyBudgetPref, budget.ID)
	if err != nil {
		return fmt.Errorf("budget id is required")
	}
	_, ok, err := r.kv.get(ctx, key)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("budget %q not found", strings.TrimSpace(budget.ID))
	}
	return r.kv.upsert(ctx, key, budget)
}

func (r *SQLiteBudgetRepository) UpdateStatus(ctx context.Context, id string, status domain.FreedomBudgetStatus, updatedAt int64, metadata map[string]any) error {
	row, ok, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("budget %q not found", strings.TrimSpace(id))
	}
	row.Status = status
	row.UpdatedAt = updatedAt
	row.Metadata = mergeMeta(row.Metadata, metadata)
	key, _ := keyFor(sqliteAutonomyBudgetPref, row.ID)
	return r.kv.upsert(ctx, key, row)
}

func (r *SQLiteDecisionRepository) Create(ctx context.Context, decision domain.AutonomyDecision) error {
	key, err := keyFor(sqliteAutonomyDecisionPref, decision.ID)
	if err != nil {
		return fmt.Errorf("decision id is required")
	}
	return r.kv.create(ctx, key, decision)
}

func (r *SQLiteDecisionRepository) ListByIntent(ctx context.Context, intentID string, limit int) ([]domain.AutonomyDecision, error) {
	items, err := r.kv.listByPrefix(ctx, sqliteAutonomyDecisionPref)
	if err != nil {
		return nil, err
	}
	intentID = strings.TrimSpace(intentID)
	out := make([]domain.AutonomyDecision, 0)
	for _, raw := range items {
		row, err := decodeRow[domain.AutonomyDecision](raw)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(row.IntentID) == intentID {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *SQLiteDecisionRepository) ListByScope(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.AutonomyDecision, error) {
	items, err := r.kv.listByPrefix(ctx, sqliteAutonomyDecisionPref)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AutonomyDecision, 0)
	for _, raw := range items {
		row, err := decodeRow[domain.AutonomyDecision](raw)
		if err != nil {
			return nil, err
		}
		if actionScopeMatch(scope, row.AllowedActions, row.BlockedActions) {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *SQLiteDecisionRepository) ListByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.AutonomyDecision, error) {
	items, err := r.kv.listByPrefix(ctx, sqliteAutonomyDecisionPref)
	if err != nil {
		return nil, err
	}
	correlationID = strings.TrimSpace(correlationID)
	out := make([]domain.AutonomyDecision, 0)
	for _, raw := range items {
		row, err := decodeRow[domain.AutonomyDecision](raw)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(row.CorrelationID) == correlationID {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return limitSlice(out, limit), nil
}

func (r *SQLiteReservationRepository) Create(ctx context.Context, reservation domain.BudgetReservation) error {
	key, err := keyFor(sqliteAutonomyReservePref, reservation.ID)
	if err != nil {
		return fmt.Errorf("reservation id is required")
	}
	return r.kv.create(ctx, key, reservation)
}

func (r *SQLiteReservationRepository) GetByID(ctx context.Context, id string) (domain.BudgetReservation, bool, error) {
	key, err := keyFor(sqliteAutonomyReservePref, id)
	if err != nil {
		return domain.BudgetReservation{}, false, nil
	}
	raw, ok, err := r.kv.get(ctx, key)
	if err != nil || !ok {
		return domain.BudgetReservation{}, ok, err
	}
	row, err := decodeRow[domain.BudgetReservation](raw)
	if err != nil {
		return domain.BudgetReservation{}, false, err
	}
	return row, true, nil
}

func (r *SQLiteReservationRepository) Update(ctx context.Context, reservation domain.BudgetReservation) error {
	key, err := keyFor(sqliteAutonomyReservePref, reservation.ID)
	if err != nil {
		return fmt.Errorf("reservation id is required")
	}
	_, ok, err := r.kv.get(ctx, key)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("reservation %q not found", strings.TrimSpace(reservation.ID))
	}
	return r.kv.upsert(ctx, key, reservation)
}

func (r *SQLiteCuriosityRepository) Create(ctx context.Context, item domain.CuriosityItem) error {
	key, err := keyFor(sqliteAutonomyCuriosityPref, item.ID)
	if err != nil {
		return fmt.Errorf("curiosity id is required")
	}
	return r.kv.create(ctx, key, item)
}

func (r *SQLiteCuriosityRepository) GetByID(ctx context.Context, id string) (domain.CuriosityItem, bool, error) {
	key, err := keyFor(sqliteAutonomyCuriosityPref, id)
	if err != nil {
		return domain.CuriosityItem{}, false, nil
	}
	raw, ok, err := r.kv.get(ctx, key)
	if err != nil || !ok {
		return domain.CuriosityItem{}, ok, err
	}
	row, err := decodeRow[domain.CuriosityItem](raw)
	if err != nil {
		return domain.CuriosityItem{}, false, err
	}
	return row, true, nil
}

func (r *SQLiteCuriosityRepository) Update(ctx context.Context, item domain.CuriosityItem) error {
	key, err := keyFor(sqliteAutonomyCuriosityPref, item.ID)
	if err != nil {
		return fmt.Errorf("curiosity id is required")
	}
	_, ok, err := r.kv.get(ctx, key)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("curiosity %q not found", strings.TrimSpace(item.ID))
	}
	return r.kv.upsert(ctx, key, item)
}

func (r *SQLiteCuriosityRepository) ListByScope(ctx context.Context, scope domain.ForgeScope, status domain.CuriosityStatus, limit int) ([]domain.CuriosityItem, error) {
	items, err := r.kv.listByPrefix(ctx, sqliteAutonomyCuriosityPref)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CuriosityItem, 0)
	for _, raw := range items {
		row, err := decodeRow[domain.CuriosityItem](raw)
		if err != nil {
			return nil, err
		}
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
