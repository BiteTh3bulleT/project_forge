# Authority Boundary Review

## Summary

GOOD: The core authority boundary is real. `controllane.Processor` performs registry lookup, validation, capability checks, approval checks, idempotency, transactional apply, journal append, audit, and linkage.

GOOD: `journal_events` append-only behavior is enforced with SQLite triggers and tested.

GOOD: Legacy memory mutation endpoints return `410 Gone` and audit the retired attempt.

GOOD: Legacy adapter direct invoke route is removed; adapter execution uses `legacy.adapter.invoke` through `/api/gateway/invoke`.

## Risks and Gaps

RISK: Public semantic syscall facade is missing.
- Files: `services/core/internal/api/server.go`, `services/core/internal/aios/controllane`.
- Why it matters: users/operators need a governed path to create semantic memory/state writes without reintroducing legacy mutation endpoints.
- Fix: add `/api/aios/syscalls` or equivalent with dry-run default, approval propagation, idempotency key, correlation, trace, and tests.

RISK: Authority/config mutation APIs are inconsistently approval-gated.
- Files/symbols: `handleSaveLane`, `handleDeleteLane`, `handleSavePermissionProfile`, `handleDeletePermissionProfile`, source add/delete handlers.
- Why it matters: permission and lane changes affect gateway authority.
- Fix: audit all changes and require approval for destructive or privilege-expanding transitions.

RISK: Model management bypasses gateway-equivalent approval.
- Files/symbols: `/forge/models/import`, `/archive`, `/remove`, `/load`, `/unload`, `model_runtime.go`, `model_runtime_bridge.go`, `modelruntime/store_management.go`.
- Why it matters: registry and filesystem metadata are durable runtime authority state.
- Fix: route management through `model.*` gateway capabilities or add equivalent approval policy.

RISK: Approval grants can be reused without a job binding.
- File/symbol: `gateway/service.go`, `approvalGrantPresent`.
- Why it matters: an approval should authorize one request fingerprint, not a reusable token.
- Fix: bind approval to tool id, lane id, capability id, paths, risk, write intent, job id, and actor.

PARTIAL: Rule agents and Dream Mode are proposal-only today.
- Good safety property.
- Missing: durable proposal/report journal and operator review workflow.

