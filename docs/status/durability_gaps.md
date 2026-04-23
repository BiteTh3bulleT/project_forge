# Durability Gaps (Phase 5.996)

Date: 2026-04-21
Scope: durable storage + backup/restore parity + restart safety.

## Durability matrix

| Subsystem | Durable storage | In `full_backup` export | Restore parity | Audit-linked | Lost on restart |
|---|---|---|---|---|---|
| Autonomy settings-backed repos (charters/intents/budgets/decisions/reservations/curiosity) | yes | yes | yes | partial | no |
| Approvals (`approval_requests`, `approval_decisions`) | yes | yes | yes | yes | no |
| Events (`events`) | yes | yes | yes | partial | no |
| Jobs + job timeline (`jobs`, `job_status_history`, `job_events`) | yes | yes | yes | partial | no |
| Artifacts (`artifacts`) | yes | yes | yes (DB row + path metadata) | partial (improved) | no |
| Semantic syscall journal (`journal_events`) | yes | yes | yes (append-safe idempotent import) | yes | no |
| Cognitive filesystem core tables | yes | yes | yes | yes | no |
| Audit/event execution history (`audit_records`, `gateway_invocations`) | yes | yes (limited extract window) | yes (within export window) | yes | no |
| Project-context/evaluation sections | yes | yes | yes | partial | no |
| VSA tables | yes | yes | no | partial | no |

## Remaining durability blockers

1. Export-only sections still exist for VSA-derived tables.
2. End-to-end traceability for artifact creation is improved (chat attachment uploads now audited with correlation/trace/workspace), but still partial across all producers.
3. Fresh-clone reproducibility is still blocked by VSA file commit state (`services/core/internal/memory/vsa_*.go`), though preflight checks now fail fast with actionable guidance.
4. Restore atomicity is DB-scoped; non-DB artifact file bytes and external side effects are explicitly warned, not globally rollback-managed.
   - Restore responses now make this explicit with `atomicScope=db-supported-sections-only`, `globalAtomic=false`, and `nonDbSideEffects` entries by section.
