# Duplicate / Overlapping Systems (Phase 5.99)

Date: 2026-04-21. K20A authority update: 2026-08-14.
Scope: identify overlaps; do not expand duplicates

## Current overlap map

| System area | Path A | Path B | Status | Risk | Decision |
|---|---|---|---|---|---|
| Tool execution | Gateway (`/api/gateway/invoke` -> `gateway.Execute`) | none | converged | low | Keep gateway as sole ingress; direct adapter invoke route removed. |
| Semantic memory/state mutation | production `forgekernel.Kernel` plus one K-owned `DurablePort` implemented by Control Lane | read-only observation inspection APIs (`/api/memory/*` GET); rollback-only `legacy_v1` boot mode | K20B single-authority cutover | low while rollback exists | Mutation endpoints remain retired (`410 Gone`); daemon bootstrap constructs one port implementation and selects one orchestrator; no dual commit. |
| Compute lane structure | `aios/computelane` interfaces (forward-declared seam, zero production importers) | `aios/compute/librarian` runtime cells | not duplicated; intent vs runtime split | none | `computelane` is a contract seam for the future IRIS service; package docstring forbids runtime logic and production imports. `compute/librarian` is the only live runtime. |
| Backup semantics | `full_backup` export of many AI-OS tables | restore import supports only policy-shaped tables | partial duplicate semantics (export-only vs import subset) | false sense of recoverability | Keep as explicit export/restore asymmetry; do not claim full restore parity. |
| Autonomy state | Single canonical store: `aios/autonomy.SQLiteBundle` over the `settings` table with the `autonomy_repo.*` key prefix (production wires this via `NewSQLiteBundleStrict`, fail-closed when DB is nil) | `autonomy_settings` is the backup-export view name for the same rows (`SELECT key, value FROM settings WHERE key LIKE 'autonomy_repo.%'`), not a parallel store | not duplicated | none | `InMemoryBundle` is retained only for unit tests / `db == nil` diagnostic mode. |

## Non-duplicates (single authoritative implementation observed)

- Approval system: single `approvals.Service` and approval APIs.
- Audit substrate: single `audit.Service` with gateway/kernel sinks.
- Tool capability registry: single registry implementation in gateway.
- Gateway core: single gateway service instance wired in `api/server.go`.
- Semantic commit adapter: single `controllane.NewProcessor` assembly site in `api/server.go`, selected behind production FORGE-K or used as the explicit boot rollback path.

## Cutover priorities

1. Keep direct adapter execution side door removed from router wiring.
2. Autonomy repos are durable via `NewSQLiteBundleStrict`; default mode remains `observe` and maintain/mission stay explicit operator opt-ins (see `cutover_blockers.md` row "Autonomy default mode is `observe`").
3. Keep memory observation mutation endpoints retired; add only syscall-native write facades for future memory/state mutation.
4. `computelane` is now a forward-declared seam (package docstring; zero production importers); `compute/librarian` is the only runtime. Adding a production import of `computelane` requires promoting the consumer through kernel/gateway commit gates first.
5. Keep tool capability status transitions on one authoritative path (`PATCH /api/gateway/capabilities/{id}/status`) with required reasons for deferred/disabled/stubbed/deprecated transitions.
