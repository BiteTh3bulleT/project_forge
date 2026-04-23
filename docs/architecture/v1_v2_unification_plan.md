# v1/v2 Unification Plan (Phase 5.996)

Observed baseline date: 2026-04-22.

Goal: one clearer authority path per operation, with legacy routes explicitly bounded.

## Current convergence map

| Area | Authoritative path | Legacy boundary | Current state |
|---|---|---|---|
| Tool execution | `/api/gateway/invoke` -> `gateway.Execute` | none | complete (legacy `/api/adapters/{id}/invoke` route removed; execution authority is gateway-only) |
| Semantic mutation | `controllane.Processor` syscall path | none for syscall entities | mostly complete |
| v1 memory observation mutation | n/a (legacy only) | `/api/memory/observations*` mutation endpoints | partial (now default-blocked + audited) |
| Approvals | `approvals.Service` | direct decision endpoints | stable |
| Audit | `audit.Service` | none observed | stable |
| Backup/export/import | `backup.Service` | manual DB/file operations | partial (transactional restore for supported sections with explicit `atomicScope`; VSA sections are explicitly `exportOnly` by policy and require recompute after restore) |
| Desktop mutation path | desktop -> backend `/api/*` | none observed client-side | stable |

## Hard constraints

1. Do not add a second tool execution authority.
2. Do not add a second semantic mutation authority outside syscall kernel.
3. Keep legacy routes explicit, default-off where possible, and audited.
4. Keep current truth and historical truth separate.

## Next cutover priorities

1. Resolve fresh-clone VSA reproducibility blocker.
2. Maintain explicit VSA export-only restore policy and preserve recompute-only invariants.
3. Keep gateway invocation coverage/tests for `legacy.adapter.invoke` to preserve compatibility behavior without reintroducing alternate ingress.
4. Strengthen traceability chain coverage for artifact-heavy flows.
