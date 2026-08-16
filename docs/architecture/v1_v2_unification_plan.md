# Runtime Authority Cutover Plan (Phase 5.997)

Observed baseline date: 2026-04-22. K20A authority update: 2026-08-14.

Goal: one authoritative path per operation. Legacy mutation boundaries are retired rather than opt-in.

## Current convergence map

| Area | Authoritative path | Legacy boundary | Current state |
|---|---|---|---|
| Tool execution | `/api/gateway/invoke` -> `gateway.Execute` | none | complete (legacy `/api/adapters/{id}/invoke` route removed; execution authority is gateway-only) |
| Semantic mutation | `forgekernel.Kernel` -> Control Lane durable adapter | offline verified recovery only | K20J sole boot authority; no live fallback or dual commit |
| Memory observation mutation | FORGE-K semantic syscall ingress | `/api/memory/observations*` mutation endpoints | complete for route cutover: legacy mutation endpoints return `410 Gone` and emit retired audit records |
| Approvals | `approvals.Service` | direct decision endpoints | stable |
| Audit | `audit.Service` | none observed | stable |
| Backup/export/import | `backup.Service` | manual DB/file operations | partial (transactional restore for supported sections with explicit `atomicScope`; VSA sections are explicitly `exportOnly` by policy and require recompute after restore) |
| Desktop mutation path | desktop -> backend `/api/*` | none observed client-side | stable |

## Hard constraints

1. Do not add a second tool execution authority.
2. Do not add a second semantic mutation authority outside syscall kernel.
3. Keep retired routes non-executable and audited.
4. Keep current truth and historical truth separate.

## Next cutover priorities

1. Preserve strict VSA source tracking and preflight checks.
2. Maintain explicit VSA export-only restore policy and preserve recompute-only invariants.
3. Keep gateway invocation coverage/tests for adapter compatibility behavior without reintroducing alternate ingress.
4. Strengthen traceability chain coverage for artifact-heavy flows.
