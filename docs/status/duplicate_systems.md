# Duplicate / Overlapping Systems (Phase 5.99)

Date: 2026-04-21
Scope: identify overlaps; do not expand duplicates

## Current overlap map

| System area | Path A | Path B | Status | Risk | Decision |
|---|---|---|---|---|---|
| Tool execution | Gateway (`/api/gateway/invoke` -> `gateway.Execute`) | none | converged | low | Keep gateway as sole ingress; direct adapter invoke route removed. |
| Semantic memory/state mutation | syscall kernel (`aios/controllane`) | read-only observation inspection APIs (`/api/memory/*` GET) | converged for mutation | low | Mutation endpoints are retired (`410 Gone`); reads remain compatibility inspection only. |
| Compute lane structure | `aios/computelane` interfaces | `aios/compute/librarian` runtime cells | duplicated (intent vs runtime split) | architecture confusion | Keep both only with explicit boundary: `computelane` = interface seam, `compute/librarian` = current runtime. |
| Backup semantics | `full_backup` export of many AI-OS tables | restore import supports only policy-shaped tables | partial duplicate semantics (export-only vs import subset) | false sense of recoverability | Keep as explicit export/restore asymmetry; do not claim full restore parity. |
| Autonomy state | Durable SQLite-backed autonomy repos via `sqlite_repositories` | Durable across store reopen (`autonomy_settings`-backed bundles) | capability split | none |

## Non-duplicates (single authoritative implementation observed)

- Approval system: single `approvals.Service` and approval APIs.
- Audit substrate: single `audit.Service` with gateway/kernel sinks.
- Tool capability registry: single registry implementation in gateway.
- Gateway core: single gateway service instance wired in `api/server.go`.

## Cutover priorities

1. Keep direct adapter execution side door removed from router wiring.
2. Quarantine autonomy maintain/mission until autonomy repos are durable.
3. Keep memory observation mutation endpoints retired; add only syscall-native write facades for future memory/state mutation.
4. Keep `computelane` vs `compute/librarian` roles explicit in docs to avoid parallel-architecture drift.
5. Keep tool capability status transitions on one authoritative path (`PATCH /api/gateway/capabilities/{id}/status`) with required reasons for deferred/disabled/stubbed/deprecated transitions.
