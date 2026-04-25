# Autonomy And Safety Gates

## Current State

PARTIAL: FORGE has autonomy charters, intents, budgets, decisions, default seeding, bounded maintenance loops, policy evaluation, and safety tests. Default mode is `observe`.

Evidence: `services/core/internal/aios/autonomy`, `services/core/internal/api/autonomy_maintenance_loop.go`, and tests.

## Lifecycle

1. Charter defines allowed purpose.
2. Intent records proposed work.
3. Budget limits time/action/risk.
4. Policy evaluator decides allow/deny/approval-required.
5. Runner executes proposal/dry-run/maintenance path.
6. Tool calls still pass gateway.
7. Semantic writes still pass syscalls.

## What Autonomy Cannot Do

- Cannot bypass gateway for tools.
- Cannot bypass syscall kernel for semantic writes.
- Cannot escalate high-risk/external/destructive/cross-workspace actions without approval.
- Cannot rely on live model behavior for kernel correctness.
- Cannot silently switch from observe/propose to maintain/mission.

## Gaps

- PARTIAL: Budget consumption across tool calls needs more tests.
- PARTIAL: Trace visibility is not yet strong enough for operator trust.
- PARTIAL: Rule agents are safe but narrow.

