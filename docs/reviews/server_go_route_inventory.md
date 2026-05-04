# services/core/internal/api/server.go Route Inventory

Date: 2026-05-03

Scope: first-pass route inventory for `services/core/internal/api/server.go`. This is intended to support a safe mechanical refactor. It is not a public API change proposal.

## Implementation Status

2026-05-03 Phase A/B status:

- Added a representative route inventory test in `services/core/internal/api/server_route_inventory_test.go`.
- Extracted route registration to `services/core/internal/api/routes.go` with focused mount helpers.
- Kept handler implementations in their existing files.
- Kept `NewServer` and `ShutdownWatch` behavior untouched.
- Preserved `/forge` and `/v1` timeout groups, conditional `/v1` mounting, `/api` timeout grouping, compatibility VSA routes, retired legacy memory gates, and removed legacy adapter invoke route behavior.
- Timeout middleware identity is not directly asserted through `chi.Walk`; the test preserves the current assistant-stream-before-timed-group route order and existing SSE behavior tests remain the deeper behavioral coverage.
- Full `go test ./internal/api` still reports the existing Windows bounded-filesystem failures in two chat gateway shortcut tests; focused route/refactor safety tests passed.

## Router Structure

Current shell:

- `chi.NewRouter()` in `Server.Handler()`.
- Global middleware: request ID, real IP, logger, recoverer.
- Global CORS policy allows empty origin, localhost, 127.0.0.1, `tauri://localhost`, and `https://tauri.localhost`.
- `/health` is mounted at root and implemented inline.
- `/forge` is mounted with `middleware.Timeout(120 * time.Second)`.
- `/v1` is mounted only when `s.cfg.EnableOpenAICompatAPI` is true and uses `middleware.Timeout(120 * time.Second)`.
- `/api/chat/threads/{id}/assistant-stream` is mounted outside the `/api` timeout group.
- All other `/api` routes are mounted inside a `middleware.Timeout(120 * time.Second)` group.

## Route Group Counts

The current `Handler()` registers 187 method/path entries.

| Group | Count | Current registration file | Proposed route group | Middleware notes | Risk notes |
|---|---:|---|---|---|---|
| `/health` | 1 | `server.go` | `mountHealthRoutes` / `health.go` | global middleware only | Health payload includes model runtime, GPU, Intel, embedding status |
| `/forge/models` | 13 | `server.go` | `mountForgeRoutes` | `/forge` 120s timeout | Model runtime management behavior is high risk |
| `/forge/model-runtime` | 5 | `server.go` | `mountForgeRoutes` | `/forge` 120s timeout | Runtime health/queue/loaded behavior is high risk |
| `/v1/models`, `/v1/chat` | 2 | `server.go` | `mountOpenAICompatRoutes` | conditional mount, 120s timeout | Compatibility route availability must remain conditional |
| `/api/chat` | 9 | `server.go`, handlers in `workspace.go` and `chat_assistant_gateway.go` | `mountChatRoutes` | assistant stream outside timeout; rest inside 120s group | Highest risk due SSE, model runtime, gateway, fallback |
| `/api/settings`, `/api/meta` | 5 | `server.go` | `mountSettingsRoutes` | 120s timeout | PATCH has runtime and gateway reload side effects |
| `/api/remote`, `/api/telegram`, `/api/discord` | 4 | `server.go`, `remote.go`, gateway status files | `mountRemoteRoutes` | 120s timeout | External ingress and gateway lifecycle |
| `/api/sources`, `/api/reindex`, `/api/search`, `/api/chunks`, `/api/events`, `/api/adapters`, `/api/commands` | 9 | `server.go` | `mountSourceRoutes` / `mountAdminRoutes` | 120s timeout | Source mutation, command execution, legacy adapter route removal |
| `/api/providers` | 1 | `server.go`, handler elsewhere | `mountProviderRoutes` | 120s timeout | Provider health should not degrade core health |
| `/api/autonomy` | 8 | `server.go`, handlers elsewhere | `mountAutonomyRoutes` | 120s timeout | Maintenance loop must remain gated |
| `/api/dream` | 6 | `server.go`, `dream.go` | `mountDreamRoutes` | 120s timeout | AI-OS dream reports are diagnostic, not automatic authority |
| `/api/jobs` | 7 | `server.go`, handlers partly phase files | `mountJobRoutes` | 120s timeout | Retry/replay/cancel behavior |
| `/api/canvas` | 7 | `server.go`, `workspace.go` | `mountCanvasRoutes` | 120s timeout | Low-medium risk |
| `/api/artifacts` | 3 | `server.go`, `workspace.go` | `mountArtifactRoutes` | 120s timeout | Content serving behavior |
| `/api/approvals` | 5 | `server.go`, handlers elsewhere | `mountApprovalRoutes` | 120s timeout | Approval gate behavior |
| `/api/context-inspector`, `/api/context`, `/api/process`, `/api/project-context` | 14 | `server.go`, `operator_inspector.go`, `restore_outcomes.go` | `mountContextRoutes` | 120s timeout | Restore inspector and outcomes must remain non-canonical |
| `/api/embeddings`, `/api/retrieval`, `/api/memory` | 26 | `server.go`, `phase3.go` | `mountMemoryRoutes` | 120s timeout | Includes retired legacy memory mutation gates and compatibility VSA routes |
| `/api/dossiers`, `/api/evaluations`, `/api/lineage`, `/api/imports`, `/api/insights` | 13 | `server.go`, `phase3.go` | `mountKnowledgeRoutes` | 120s timeout | Mixed content/knowledge operations |
| `/api/dashboard` | 1 | `server.go`, `phase4.go` | `mountDashboardRoutes` | 120s timeout | Low risk |
| `/api/strategies`, `/api/policy`, `/api/automation`, `/api/packet-guidance`, `/api/reconciliation`, `/api/reviews`, `/api/failure-patterns` | 25 | `server.go`, `phase4.go` | `mountGovernanceRoutes` | 120s timeout | Governance and policy behavior |
| `/api/gateway` | 5 | `server.go`, `phase5.go` | `mountGatewayRoutes` | 120s timeout | Tool execution and capability governance are high risk |
| `/api/action-lanes`, `/api/permissions`, `/api/audit` | 10 | `server.go`, `phase5.go`, `operator_inspector.go` | `mountPermissionAuditRoutes` | 120s timeout | Permission/audit trace behavior |
| `/api/backup`, `/api/release` | 8 | `server.go`, `phase5.go` | `mountOperationsRoutes` | 120s timeout | Backup/restore and release readiness are high risk |

## Critical Exact Routes

These routes need explicit inventory assertions before route extraction.

| Method | Path | Handler | Current file | Proposed group | Compatibility/risk notes |
|---|---|---|---|---|---|
| GET | `/health` | inline handler | `server.go:366` | health | Includes model runtime, telemetry, embeddings status |
| GET | `/api/chat/threads/{id}/assistant-stream` | `handleChatAssistantStream` | `server.go:448` | chat | Must remain outside short `/api` timeout |
| GET | `/v1/models` | `handleV1Models` | `server.go:441` | model runtime | Mounted only when OpenAI compatibility is enabled |
| POST | `/v1/chat/completions` | `handleV1ChatCompletions` | `server.go:442` | model runtime | Mounted only when OpenAI compatibility is enabled |
| GET | `/forge/models` | `handleForgeModelsList` | `server.go:418` | model runtime | `/forge` timeout group |
| POST | `/forge/models/{id}/load` | `handleForgeModelLoad` | `server.go:428` | model runtime | Governance and runtime lifecycle |
| POST | `/forge/models/{id}/unload` | `handleForgeModelUnload` | `server.go:429` | model runtime | Governance and runtime lifecycle |
| POST | `/forge/models/{id}/chat` | `handleForgeModelChat` | `server.go:430` | model runtime | Model chat behavior |
| GET | `/forge/model-runtime/health` | `handleForgeModelRuntimeHealth` | `server.go:433` | model runtime | Runtime health surface |
| GET | `/forge/model-runtime/queue` | `handleForgeModelRuntimeQueue` | `server.go:434` | model runtime | Runtime scheduler surface |
| GET | `/forge/model-runtime/loaded` | `handleForgeModelRuntimeLoaded` | `server.go:435` | model runtime | Runtime loaded model surface |
| GET | `/api/settings` | `handleGetSettings` | `server.go:454` | settings | Settings response contract |
| PATCH | `/api/settings` | `handlePatchSettings` | `server.go:455` | settings | Reloads runtime controls and remote gateways |
| GET | `/api/settings/ollama-models` | `handleGetOllamaModels` | `server.go:456` | settings | Also has trailing slash route |
| POST | `/api/remote/telegram` | `handleRemoteTelegram` | `server.go:458` | remote | Remote token behavior |
| POST | `/api/remote/discord` | `handleRemoteDiscord` | `server.go:459` | remote | Remote token behavior |
| GET | `/api/telegram/status` | `handleTelegramStatus` | `server.go:460` | remote | External gateway status |
| GET | `/api/discord/status` | `handleDiscordGatewayStatus` | `server.go:461` | remote | External gateway status |
| POST | `/api/gateway/invoke` | `handleGatewayInvoke` | `server.go:623` | gateway | Tool execution authority path |
| PATCH | `/api/gateway/capabilities/{id}/status` | `handleGatewayCapabilityStatusUpdate` | `server.go:622` | gateway | Capability governance |
| GET | `/api/memory/vsa/reindex/runs` | `handleListVSAReindexRuns` | `server.go:562` | memory | Compatibility route |
| GET | `/api/memory/vsa/reindex/runs/{id}` | `handleGetVSAReindexRun` | `server.go:563` | memory | Compatibility route |
| POST | `/api/memory/observations` | `withLegacyMemoryMutationGate(...)` | `server.go:551` | memory | Retired legacy mutation gate |
| PATCH | `/api/memory/observations/{id}` | `withLegacyMemoryMutationGate(...)` | `server.go:554` | memory | Retired legacy mutation gate |
| POST | `/api/memory/observations/{id}/usefulness` | `withLegacyMemoryMutationGate(...)` | `server.go:555` | memory | Retired legacy mutation gate |
| POST | `/api/backup/restore` | `handleRestoreBundle` | `server.go:642` | operations | Restore behavior |
| POST | `/api/commands/execute` | `handleCommandExecute` | `server.go:649` | admin/commands | Command execution behavior |

## Proposed Mount Function Map

| Function | Owns |
|---|---|
| `mountMiddleware` | global middleware and CORS |
| `mountHealthRoutes` | `/health` |
| `mountForgeRoutes` | `/forge/models*`, `/forge/model-runtime/*` |
| `mountOpenAICompatRoutes` | conditional `/v1/models`, `/v1/chat/completions` |
| `mountAPIRoutes` | `/api` root, SSE exception, timeout group |
| `mountSettingsRoutes` | `/api/meta`, `/api/settings*` |
| `mountRemoteRoutes` | `/api/remote/*`, `/api/telegram/status`, `/api/discord/status` |
| `mountSourceRoutes` | `/api/sources`, `/api/reindex`, `/api/search`, `/api/chunks`, `/api/events`, `/api/adapters` |
| `mountProviderRoutes` | `/api/providers/capabilities` |
| `mountAutonomyRoutes` | `/api/autonomy/*` |
| `mountDreamRoutes` | `/api/dream/*` |
| `mountJobRoutes` | `/api/jobs*` |
| `mountChatRoutes` | `/api/chat/*` |
| `mountCanvasRoutes` | `/api/canvas/*` |
| `mountArtifactRoutes` | `/api/artifacts*` |
| `mountApprovalRoutes` | `/api/approvals*` |
| `mountContextRoutes` | context inspector, restore, outcomes, process health, project context |
| `mountMemoryRoutes` | embeddings, retrieval, memory, VSA, repair |
| `mountKnowledgeRoutes` | dossiers, evaluations, lineage, imports, insights |
| `mountGovernanceRoutes` | strategies, policy, automation, packet guidance, reconciliation, reviews, failure patterns |
| `mountGatewayRoutes` | gateway tools, capabilities, invocation |
| `mountPermissionAuditRoutes` | action lanes, permissions, audit |
| `mountOperationsRoutes` | backup and release |
| `mountCommandRoutes` | `/api/commands/execute` |

## Route Inventory Test Shape

Recommended assertions:

- Build handler with OpenAI compatibility disabled and assert `/v1/models` is 404.
- Build handler with OpenAI compatibility enabled and assert `/v1/models` is not 404.
- Assert representative routes from every group are mounted.
- Assert removed legacy adapter invoke route remains 404.
- Assert compatibility VSA routes remain mounted.
- Assert assistant stream route remains registered outside the `/api` timeout group. If direct middleware introspection is impractical, use a slow test handler seam only after route extraction or assert the route is mounted before the timeout group in a route inventory fixture.

## First Implementation Boundary

The first code change should only extract route mounting into functions. Handler implementations should stay in their current files. Constructor wiring, lifecycle startup, model runtime behavior, gateway behavior, and chat behavior should remain untouched.
