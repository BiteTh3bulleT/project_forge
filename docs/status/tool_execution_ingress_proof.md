# Tool Execution Ingress Proof

Date: 2026-04-22  
Scope: post-cutover verification that tool execution authority remains gateway-only.

## Result

- Authoritative tool execution ingress is `POST /api/gateway/invoke`.
- Legacy direct adapter ingress `POST /api/adapters/{id}/invoke` is not routed.
- No alternate API/chat direct tool execution path was found outside gateway execution.

## Ingress Inventory

| Surface | Current behavior | Authority |
| --- | --- | --- |
| `/api/gateway/invoke` | calls `handleGatewayInvoke` -> `gateway.Execute` | authoritative |
| `/api/adapters/{id}/invoke` | route removed; returns router `404` | none |
| Chat deterministic fallback | calls `s.gateway.Execute(...)` | authoritative |
| Chat assistant tool dispatch | calls `s.gateway.Execute(...)` from `dispatchToolCall` | authoritative |
| Chat `/tool` command | enqueues `gateway_action` job template | authoritative |

## Evidence

- `services/core/internal/api/server.go`
  - `/api/gateway/invoke` route wiring remains.
  - `/api/adapters/{id}/invoke` route wiring is absent.
- `services/core/internal/api/server_adapters_test.go`
  - `TestLegacyAdapterInvokeRouteRemoved` asserts `404` for removed route.
  - Gateway path tests assert `ok/denied/error/needs_approval` outcomes + audit coverage for `legacy.adapter.invoke`.
  - Gateway metadata coverage asserts `legacy.adapter.invoke` is `scoped_execute`, `L2`, and `usesNetwork=true`.
- `services/core/internal/api/chat_assistant_gateway.go`
  - `dispatchToolCall(...)` builds gateway request and executes through `s.gateway.Execute(...)`.
- `services/core/internal/api/chat_fs_fallback.go`
  - deterministic filesystem fallback executes via `s.gateway.Execute(...)`.
- `services/core/internal/api/chat_post.go`
  - `/tool` path maps to `gateway_action` execution template.
- `services/core/internal/api/legacy_adapter_gateway_tool.go`
  - legacy adapter invocation remains available only as deprecated gateway tool `legacy.adapter.invoke`.
  - The tool advertises networked scoped execution so it is gated by network/profile approval instead of underreported as low-risk local behavior.

## Notes / Boundaries

- Job templates may still invoke adapters directly for non-gateway legacy job flows (`services/core/internal/jobs/service.go` adapter branch).  
  This is distinct from API tool ingress and does not reintroduce `/api/adapters/{id}/invoke`.
- Current convergence claim is specifically: API/chat tool execution authority is gateway-governed.
