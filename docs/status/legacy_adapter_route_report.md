# Legacy Adapter Route Report

Date: 2026-04-22
Scope: final disposition for legacy adapter invocation ingress.

## Current Route / Handler

- Route: `POST /api/adapters/{id}/invoke`
- Handler: removed from `services/core/internal/api/server.go`
- Current posture: route removed from API router; request now returns standard router `404 Not Found`.

## Gating Behavior

- Route is not registered.
- `FORGE_ALLOW_LEGACY_ADAPTER_INVOKE` is no longer used by runtime code paths.
- Tool execution authority is only the gateway path.

## Policy and Audit Behavior

- There is no route-specific retired-boundary audit action because the route no longer exists.
- Adapter execution occurs only through `gateway.Execute` and keeps standard gateway policy + permission + audit behavior (`tool.executed`, `tool.denied`, `tool.error`, `tool.needs_approval`).

## Known Dependencies (Pass 1 findings)

- In-repo runtime caller before cutover: desktop probe (`apps/desktop/src/pages/AdaptersPage.tsx`) via `api.adapters.invoke` in `apps/desktop/src/lib/api.ts`.
- No other in-repo runtime execution callers were found; remaining references were tests/docs/status text.
- Desktop caller has been migrated to gateway invocation (`/api/gateway/invoke`, `toolId=legacy.adapter.invoke`, `laneId=legacy.adapter.invoke`).

## Final Disposition

- Legacy adapter ingress endpoint has been removed from router wiring.
- Adapter/tool execution remains available only through `/api/gateway/invoke`.
- No direct adapter execution path remains under `/api/adapters/{id}/invoke`; requests receive `404`.

## Phase 18 Read-Only Proof

`GET /forge/system/status` now includes `legacy_retirement.entries[]` metadata for this retired surface. The entry records:

- live owner: Gateway
- target FORGE-K owner: future gateway/capability boundary
- route state: `unrouted`
- default-live replacement: `/api/gateway/invoke` with `toolId=legacy.adapter.invoke`
- rollback proof: the direct adapter route must remain absent from route inventory while the Gateway compatibility wrapper remains bounded

This status metadata is read-only and does not reintroduce the route.
