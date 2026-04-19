# Freedom Budget

Freedom budgets cap autonomous throughput and prevent runaway self-action.

## Budget model

Defined in `services/core/internal/aios/domain/autonomy.go` as `FreedomBudget`.

Key fields:

- `id`, `name`, `scope`
- `status` (`active`, `suspended`, `exhausted`, `expired`)
- `period` (`per_run`, `hourly`, `daily`, `weekly`, `mission`)
- action/tool/run limits
- optional `maxCostUnits`
- live usage counters
- `resetsAt`
- metadata/timestamps

## Reservation model

`BudgetReservation` tracks pre-commit budget reservation:

- reservation id
- budget id
- intent id
- scope
- requested purpose
- units
- consumed/released flags

## Budget service behavior

`services/core/internal/aios/autonomy/budget.go`:

- `CheckBudget`
- `ReserveBudget`
- `ConsumeBudget`
- `ReleaseBudget`
- `GetUsage`
- `ExplainBudgetDecision`

Flow:

1. policy evaluator requests `CheckBudget`
2. runner reserves before commit
3. runner consumes on successful commit
4. runner releases reservation on failure

## Denial behavior

When limits or budget status block commit:

- decision becomes `blocked_by_budget`
- intent remains proposal/blocked path
- action may still be proposal-only or approval-required

Budget denial never bypasses kernel; it prevents runner auto-commit.

## Dry-run behavior

Dry-run/validate-only evaluations do not consume commit budget.

## Reset behavior

Budget check can reset expired usage counters when current time passes `resetsAt`, then re-evaluate limits.

## Default budgets

Defined in `autonomy/defaults.go`:

- `budget_memory_maintenance`
- `budget_context_prep`

Defaults are conservative and scoped.
