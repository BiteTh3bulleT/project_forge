# services/core/internal/api/server.go Architecture Review

Date: 2026-05-03

Scope: review and refactor planning only. This document does not approve route behavior changes, public API changes, runtime behavior changes, model behavior changes, gateway behavior changes, or chat behavior changes.

## Implementation Status

2026-05-03 Phase A/B status:

- Added `services/core/internal/api/server_route_inventory_test.go` with representative route inventory guardrails.
- Extracted route mounting into `services/core/internal/api/routes.go`.
- Reduced `Server.Handler()` to the route shell described in this review.
- Handler implementations remain in their current files; the inline health implementation remains in `server.go` as `handleHealth`.
- Constructor wiring, lifecycle startup/shutdown, model runtime behavior, gateway behavior, chat assistant behavior, and remote Telegram/Discord behavior were not intentionally changed.
- The assistant stream timeout exclusion is guarded by route-order inventory because chi route walking does not expose timeout middleware identity directly.
- Full `go test ./internal/api` still reports the existing Windows bounded-filesystem failures in `TestChatPostSyncRoutesDownloadSorterThroughGateway` and `TestChatPostSyncMultiSVGUsesDeterministicGatewayShortcut`; the focused route/refactor safety subset passed.

## Executive Summary

`services/core/internal/api/server.go` has become the API package shell, dependency constructor, lifecycle coordinator, route registry, settings controller, health controller, source/search controller, event endpoint, legacy memory gate, command endpoint, runtime settings hydrator, and shared JSON/settings helper file.

The most important refactor is not to move code quickly. The first implementation phase should create a route inventory test and extract route mounting mechanically, preserving exact paths, methods, middleware, ordering, and compatibility behavior. Constructor and lifecycle extraction should wait until route behavior is guarded.

Key facts from review:

- `server.go` is 1,511 lines.
- It imports 55 packages.
- `Server` currently has about 50 fields.
- `Handler()` registers 187 route methods across `/health`, `/forge`, conditional `/v1`, and `/api`.
- `NewServer()` constructs most services and starts background services before returning.
- `ShutdownWatch()` stops jobs, autonomy, Telegram, Discord, watch cancellation, and watch close, but its name no longer reflects its lifecycle scope.
- The package already has historical phase files (`phase3.go`, `phase4.go`, `phase5.go`) and some domain files, but top-level routing still lives in one large method.

## Current Responsibility Map

### Server Fields

Defined at `services/core/internal/api/server.go:63`.

Recommended grouping:

| Group | Current fields |
|---|---|
| Core | `st`, `cfg`, `log` |
| Content and knowledge | `ingest`, `search`, `embeddings`, `retrieval`, `memory`, `projectCtx`, `artifacts`, `dossiers`, `lineage`, `imports`, `insights` |
| Conversation and workspace | `chat`, `canvas`, `packets` |
| Governance | `approvals`, `strategies`, `policy`, `automation`, `packetOpt`, `reviews`, `reconcile`, `failures`, `gateway`, `lanes`, `permissions`, `auditSvc` |
| Runtime and telemetry | `modelRuntime`, `dream`, `gpuTelemetry`, `intelTelemetry` |
| Background and integrations | `jobs`, `watch`, `watchStop`, `autonomy`, `telegramGateway`, `telegramErr`, `discordGateway`, `discordErr` |
| Synchronization | `shutdownOnce`, `telegramMu`, `discordMu`, `chatAssistInflight` |
| Legacy adapters | `adapters` |
| Operations | `backup`, `release`, `dashboard` |

Risk: the field list is now an implicit dependency graph. Tests and new code can reach into too much state because everything is on `Server`.

### NewServer Responsibilities

Defined at `services/core/internal/api/server.go:118`.

Current responsibilities include:

- Hydrates runtime config from persisted settings.
- Creates the event logger.
- Loads settings such as extension lists.
- Constructs ingest, search, embeddings, memory, retrieval, artifacts, chat, canvas, packets, approvals, project context, dossiers, evaluations, lineage, imports, insights, strategies, policy, automation, packet optimization, reviews, reconciliation, failure patterns, dashboard, adapters, permissions, lanes, audit, backup, release, telemetry, model runtime, dream, gateway, and jobs.
- Ensures embedding provider config.
- Ensures permission defaults, scratch directory, chat mkdir policy, gateway tool policy, and lane defaults.
- Activates workspace-write profile under root workspace condition.
- Constructs telemetry clients.
- Initializes the model runtime service.
- Builds rule engine and dream service.
- Constructs autonomy maintenance loop.
- Creates gateway capability registry with SQLite override fallback.
- Registers the legacy adapter gateway tool.
- Ensures strategy, policy, and automation defaults.
- Constructs watch manager.
- Creates a root background context and cancel function.
- Starts approval expiry reaper.
- Starts watch manager and source sync.
- Starts autonomy loop.
- Allocates `Server`.
- Starts Telegram and Discord gateways.

Risk: construction has hidden side effects. A caller cannot instantiate a lightweight server without also starting reapers, watchers, autonomy, and remote gateways.

### Handler Responsibilities

Defined at `services/core/internal/api/server.go:341`.

Current responsibilities include:

- Constructs chi router.
- Adds request ID, real IP, logger, recoverer middleware.
- Configures CORS policy.
- Implements `/health` inline.
- Mounts `/forge` with a 120-second timeout.
- Conditionally mounts `/v1` OpenAI-compatible routes with a 120-second timeout.
- Mounts `/api`.
- Excludes `/api/chat/threads/{id}/assistant-stream` from the short `/api` timeout group.
- Registers every remaining `/api` route under a 120-second timeout.

Risk: any route addition or move requires editing the central registry. The SSE timeout exception is easy to break during mechanical cleanup.

### Handlers Still Defined in server.go

Handlers and helpers defined in `server.go` after `Handler()` include:

- `handleMeta`
- `handleGetSettings`
- `handlePatchSettings`
- `handleGetOllamaModels`
- `handleListSources`
- `handleAddSource`
- `handleDeleteSource`
- `handleReindex`
- `handleSearch`
- `handleChunk`
- `handleEvents`
- `handleAdapters`
- `withLegacyMemoryMutationGate`
- `handleCommandExecute`
- `createTemplateJob`
- `writeJSON`
- settings/config helpers: `runtimeConfigFromSettings`, `runtimeControlsFromSettings`, `patchRuntimeControls`, `reloadRuntimeControls`, `loadSetting`, `upsertSetting`
- source helper: `listSourcePaths`
- misc helpers: `normalizeOllamaModel`, `firstNonEmptyTrimmed`, `profileIDOrEmpty`

Risk: route registration and route implementation are interleaved in the same file, so even small endpoint edits increase merge conflict risk in the central shell.

### Background Services and Shutdown

Startup happens in `NewServer()`:

- Approval expiry reaper starts at `server.go:252`.
- Watch manager runs at `server.go:254`.
- Source sync starts at `server.go:255`.
- Autonomy loop starts at `server.go:258`.
- Telegram and Discord gateways start at `server.go:279`.

Shutdown happens in `ShutdownWatch()` at `server.go:312`:

- Closes jobs.
- Stops autonomy.
- Stops Telegram gateway.
- Stops Discord gateway.
- Cancels watch context.
- Closes watch manager.

Risk: shutdown responsibility is broader than the method name. Startup and shutdown symmetry is implicit and easy to miss.

## Monolith Symptoms

### High Severity

1. Hidden side effects in constructor.

`NewServer()` starts background work and remote gateways before returning. Tests that only need handler routing inherit runtime side effects. This increases flake risk and makes isolated route tests harder.

2. Route registry has fragile middleware exceptions.

The SSE route at `server.go:448` is intentionally outside the `/api` timeout group. `/forge` and `/v1` route groups have their own 120-second timeout. Any extraction that moves route order or timeout placement can change streaming behavior or compatibility behavior.

3. Server struct is a service locator.

The 50-field `Server` struct gives every handler access to nearly every domain service. This makes domain boundaries advisory rather than enforced.

4. Gateway, model runtime, chat, and filesystem fallback behavior are tightly adjacent.

`server.go` wires the gateway and model runtime, while `chat_assistant_gateway.go`, `chat_fs_fallback.go`, model runtime handlers, and gateway routes depend on the same `Server` state. Refactors in this area need route and behavior tests first.

### Medium Severity

5. Phase-based files are not long-term domain boundaries.

`phase3.go`, `phase4.go`, and `phase5.go` are useful historical splits but mix multiple domains. Over time they should be replaced with domain-based route files.

6. Settings and runtime control handlers live in the server shell.

`handlePatchSettings()` persists settings, reloads runtime controls, restarts Telegram and Discord gateways, and updates service configuration. This belongs in a settings route module plus a runtime settings/lifecycle helper.

7. Route behavior is mostly guarded indirectly.

There are strong endpoint tests in specific domains, but no route inventory snapshot test that proves the full route set survives mechanical movement.

8. Lifecycle naming is stale.

`ShutdownWatch()` now shuts down jobs, autonomy, Telegram, Discord, and watch. The public behavior should remain stable, but the internals should gain a clearer lifecycle helper.

## Target Architecture

Keep the `api` package flat at first. Avoid nested package splits until the mechanical route extraction is stable.

Recommended target files:

| File | Responsibility |
|---|---|
| `server.go` | `Server` type, public `NewServer`, `Handler`, `ShutdownWatch` compatibility method, minimal shell only |
| `dependencies.go` | `ServerDependencies`, dependency grouping structs, dependency construction helpers |
| `lifecycle.go` | background startup and shutdown helpers; approval reaper, watch, autonomy, remote gateways |
| `routes.go` | `mountMiddleware`, `mountHealthRoutes`, `mountForgeRoutes`, `mountOpenAICompatRoutes`, `mountAPIRoutes` |
| `routes_settings.go` | meta/settings/Ollama settings routes and handlers |
| `routes_sources.go` | source, reindex, search, chunk, events, adapters, command endpoints |
| `routes_chat.go` | chat thread routes and assistant stream mount point only; keep heavy assistant behavior in existing files |
| `routes_canvas.go` | canvas routes |
| `routes_artifacts.go` | artifact routes |
| `routes_jobs.go` | jobs, job templates, retry/replay/cancel |
| `routes_context.go` | context inspector, restore, restore outcomes, process health, project context |
| `routes_memory.go` | embeddings, retrieval, memory, VSA, repair |
| `routes_dossiers.go` | dossiers, evaluations, lineage, imports, insights |
| `routes_governance.go` | approvals, policy, automation, packet guidance, reconciliation, reviews, failure patterns |
| `routes_gateway.go` | gateway tools, capabilities, invocation routes |
| `routes_permissions.go` | lanes, permissions, audit |
| `routes_operations.go` | backup and release |
| `routes_remote.go` | remote webhook and Telegram/Discord status routes |
| `routes_model_runtime.go` | `/forge` and `/v1` route mounting only; keep existing model runtime handlers where they are during first pass |
| `health.go` | `/health` handler and health payload composition |

Do not move `chat_assistant_gateway.go`, `chat_fs_fallback.go`, model runtime handlers, Telegram/Discord gateway internals, or gateway execution internals in the first route extraction phase.

## Dependency Grouping Recommendation

Introduce grouping structs without changing `NewServer(st, cfg)` public behavior:

```text
ServerDependencies
  CoreServices
  ContentServices
  ConversationServices
  GovernanceServices
  RuntimeServices
  IntegrationServices
  BackgroundServices
  OperationsServices
```

Recommended grouping:

| Group | Members |
|---|---|
| `CoreServices` | store, config, event logger |
| `ContentServices` | ingest, search, embeddings, retrieval, memory, project context, artifacts, dossiers, lineage, imports, insights |
| `ConversationServices` | chat, canvas, packets |
| `GovernanceServices` | approvals, strategies, policy, automation, packet optimization, reviews, reconciliation, failure patterns, gateway, lanes, permissions, audit |
| `RuntimeServices` | model runtime, dream, GPU telemetry, Intel telemetry |
| `IntegrationServices` | adapters, Telegram gateway state, Discord gateway state |
| `BackgroundServices` | jobs, watch manager, watch cancel, autonomy loop, approval reaper context |
| `OperationsServices` | backup, release, dashboard |

Target construction flow:

1. `NewServer(st, cfg)` remains public.
2. It calls `BuildServerDependencies(context.Background(), st, cfg)`.
3. It calls `NewServerWithDeps(deps)`.
4. It calls `srv.StartBackgroundServices(context.Background())`.
5. It returns `srv`.

This keeps behavior stable while creating test seams for future phases.

## Route Modularization Recommendation

Target shell:

```text
func (s *Server) Handler() http.Handler {
    r := chi.NewRouter()
    s.mountMiddleware(r)
    s.mountHealthRoutes(r)
    s.mountForgeRoutes(r)
    s.mountOpenAICompatRoutes(r)
    s.mountAPIRoutes(r)
    return r
}
```

Important preservation rules:

- Preserve exact route paths.
- Preserve HTTP methods.
- Preserve route order where order could matter.
- Preserve CORS behavior.
- Preserve request ID, real IP, logger, recoverer middleware.
- Preserve `/forge` 120-second timeout.
- Preserve conditional `/v1` mounting based on `EnableOpenAICompatAPI`.
- Preserve `/v1` 120-second timeout.
- Preserve the `/api/chat/threads/{id}/assistant-stream` timeout exclusion.
- Preserve the `/api` 120-second timeout group for all other routes.
- Preserve compatibility memory VSA routes.
- Preserve removed legacy adapter invoke route behavior as not found.

## Construction and Lifecycle Recommendation

Do not change public constructor behavior in the first implementation pass.

Recommended stages:

1. Add `ServerDependencies` as an internal struct.
2. Move pure dependency construction into `BuildServerDependencies`.
3. Keep side-effect startup in `NewServer` initially.
4. Add `StartBackgroundServices` only after tests prove constructor behavior and shutdown behavior.
5. Keep `ShutdownWatch()` as the public compatibility method, but delegate to `shutdownBackgroundServices()`.
6. Consider a later `Shutdown()` alias only after call sites are audited.

Do not split lifecycle before route tests exist. The constructor starts several services that may have hidden assumptions about ordering.

## Testing Plan

Add tests before moving code:

| Test | Purpose |
|---|---|
| Route inventory snapshot | Capture method/path presence for `/health`, `/forge`, `/v1`, and `/api` |
| `/health` response smoke | Preserve OK status and provider degradation semantics |
| `/forge` route existence | Preserve model runtime paths and timeout group |
| `/v1` disabled by default | Preserve 404 when compatibility API disabled |
| `/v1` enabled route presence | Preserve OpenAI compatibility routes when enabled |
| `/api` route existence | Preserve key route families |
| SSE timeout exclusion | Prove assistant stream route is mounted outside short `/api` timeout group |
| Settings GET/PATCH | Preserve settings persistence and runtime control reload behavior |
| NewServer smoke | Prove constructor still returns usable server |
| Shutdown idempotency | Preserve current `ShutdownWatch()` behavior |
| Duplicate/conflict route check | Catch accidental duplicate mounts or shadowing |
| Removed legacy route test | Preserve `/api/adapters/{id}/invoke` not found behavior |

Existing useful tests include:

- `server_shutdown_test.go` for idempotent shutdown.
- `settings_test.go` for settings persistence.
- `provider_capabilities_test.go` for `/health` and provider capabilities.
- `model_runtime_test.go` for `/forge` and `/v1` compatibility behavior.
- `chat_post_model_runtime_fallback_test.go` for chat, model runtime fallback, and SSE behavior.
- `server_adapters_test.go` for removed legacy adapter invoke route and gateway adapter invocation.
- `server_memory_legacy_test.go` for retired legacy memory mutation gates.
- `remote_*`, `telegram_*`, and `discord_*` tests for remote ingress behavior.

## Phased Refactor Plan

### Phase A - Review and Route Inventory

Goal: establish a guardrail before moving code.

Deliverables:

- Route inventory test or generated test fixture.
- Documentation of current route groups and middleware notes.
- No behavior changes.

Validation:

- `go test ./internal/api -run 'Route|Health|OpenAICompat|Shutdown|Settings'`
- Confirm full API package compile.

### Phase B - Extract Route Mounting

Goal: shrink `Handler()` without moving handler implementations.

Deliverables:

- `routes.go`
- `mountMiddleware`
- `mountHealthRoutes`
- `mountForgeRoutes`
- `mountOpenAICompatRoutes`
- `mountAPIRoutes`
- Domain `mount*Routes` helpers that still call existing handlers.

Validation:

- Route inventory test passes unchanged.
- Focused route behavior tests pass.

### Phase C - Extract Settings, Meta, Health, Sources

Goal: move low-risk handlers out of `server.go`.

Deliverables:

- `health.go`
- `routes_settings.go`
- `routes_sources.go`
- Settings helper functions moved with tests unchanged.

Validation:

- Settings tests.
- Provider capabilities health test.
- Source/search smoke if available.

### Phase D - Extract Dependency Construction

Goal: create constructor seams without changing public constructor behavior.

Deliverables:

- `dependencies.go`
- `ServerDependencies`
- grouped service construction helpers.
- `NewServerWithDeps` internal or exported only if needed.

Validation:

- NewServer smoke tests.
- Existing API package tests.

### Phase E - Extract Lifecycle and Background Startup

Goal: make startup and shutdown responsibilities explicit.

Deliverables:

- `lifecycle.go`
- `startBackgroundServices`
- `shutdownBackgroundServices`
- `ShutdownWatch()` delegates to lifecycle shutdown for compatibility.

Validation:

- Shutdown idempotency.
- Remote gateway status tests.
- Watch/autonomy tests.

### Phase F - Domain-Based API Modules

Goal: replace historical phase file boundaries gradually.

Deliverables:

- `routes_memory.go`, `routes_governance.go`, `routes_gateway.go`, `routes_operations.go`, and similar domain files.
- Move handlers only when tests exist for the domain.

Validation:

- Domain-specific tests after each small move.

### Phase G - Optional Package Split

Goal: reduce package coupling only if flat files stop helping.

Deliverables:

- Consider subpackages only for isolated pure helpers or large bounded subsystems.

Validation:

- No import cycles.
- No public API churn.

## Risk Register

| Risk | Severity | Why it matters | Mitigation |
|---|---:|---|---|
| SSE stream timeout changes | High | Assistant stream route is deliberately outside `/api` timeout group | Add explicit timeout-exclusion test before extraction |
| `/v1` compatibility routing changes | High | Route availability is conditional and externally visible | Preserve conditional mount and add route inventory assertions |
| `/forge` model runtime route changes | High | Model management, load/unload, chat, health, queue, loaded routes are sensitive | Keep mount order and timeout identical; run model runtime tests |
| Gateway invocation changes | High | Tool execution authority and audit path must remain gateway-only | Do not move gateway execution internals in first pass |
| Chat assistant gateway changes | High | Mixed model runtime, gateway tools, fallback, and SSE paths | Treat as do-not-touch until route shell tests exist |
| Remote Telegram/Discord ingress changes | High | External ingress, settings reload, and gateway lifecycle are side-effectful | Keep remote handlers and gateway lifecycle untouched initially |
| Constructor side effects | Medium | Tests accidentally start background work | Extract only after NewServer smoke and lifecycle tests |
| Settings reload side effects | Medium | PATCH settings can reload runtime controls and remote gateways | Extract handlers mechanically with existing settings tests |
| Phase-file movement churn | Medium | Large moves can obscure behavior changes | Move by domain only after route mounting is stable |
| Route duplicate/shadowing | Medium | Chi route order can matter | Add route inventory and duplicate checks |

## Do Not Touch Yet Without Tests

- Assistant SSE streaming.
- Chat assistant gateway execution.
- Deterministic filesystem fallback.
- Remote Telegram and Discord ingress.
- Gateway tool invocation.
- Gateway capability status governance.
- Model runtime load, unload, chat, health, queue, and `/v1` compatibility.
- Backup and restore.
- Permissions, audit, release readiness.
- Context restore inspector routes.
- Autonomy maintenance loop.
- Legacy memory mutation gate behavior.

## Acceptance Criteria for Refactor Implementation

- Route inventory remains unchanged unless a separate API change is approved.
- HTTP methods remain unchanged.
- Middleware behavior remains unchanged.
- `/api/chat/threads/{id}/assistant-stream` remains outside the short timeout group.
- `/forge` and `/v1` timeout behavior remains unchanged.
- `NewServer(st, cfg)` remains available and behavior-compatible.
- `ShutdownWatch()` remains idempotent and behavior-compatible.
- Existing focused API tests pass.
- No new framework is introduced.
- No nested package split is introduced until flat-file extraction is exhausted.

## What Not To Do

- Do not implement new routes as part of the refactor.
- Do not change public APIs.
- Do not change route paths or methods.
- Do not move SSE handling casually.
- Do not change model runtime behavior.
- Do not change chat assistant streaming behavior.
- Do not change gateway execution behavior.
- Do not remove phase files in the first implementation pass.
- Do not create a second router architecture.
- Do not turn `routes.go` into another monolith.

## Recommended Next Implementation Prompt

Implement Phase A/B of the `server.go` refactor only.

Read `docs/reviews/server_go_refactor_review.md` and `docs/reviews/server_go_route_inventory.md` first. Add a route inventory test for the current `Server.Handler()` route surface, including `/health`, `/forge`, conditional `/v1`, `/api`, compatibility routes, and the assistant SSE timeout exception. Then extract route registration from `Handler()` into `routes.go` with `mountMiddleware`, `mountHealthRoutes`, `mountForgeRoutes`, `mountOpenAICompatRoutes`, and `mountAPIRoutes`. Do not move handler implementations yet. Preserve exact paths, methods, route order, middleware, `/forge` timeout behavior, conditional `/v1` behavior, `/api` timeout behavior, and the SSE timeout exclusion. Run focused API route/model/chat/settings/shutdown tests and report any pre-existing failures separately.
