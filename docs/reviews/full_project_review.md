# FORGE Full Project Review and Readiness Audit

Date: 2026-05-03  
Scope: full repository readiness audit, with emphasis on FORGE-K doctrine, live core health, API shape, test health, and next safe work.  
Mode: review/planning only. No features, broad refactors, public API changes, or behavior changes were made by this audit.

## Executive Summary

FORGE currently has two important implementation tracks:

1. **The live FORGE core** under `services/core/internal/...`, exposed through the Go daemon and desktop shell. This path contains the current API, gateway, permissions, model runtime, AI-OS controllane, memory, chat, backup, release, and operational behavior.
2. **The FORGE-K simulator** under `services/core/internal/forgek/...`. This is a self-contained deterministic cognitive microkernel simulator. Phases 1-6 are implemented and tested there, but non-test live code does not import it yet.

The biggest readiness issue is not that the FORGE-K simulator is weak. It is strong and well tested for its current scope. The issue is that the live daemon still relies on the pre-FORGE-K authority paths (`aios/controllane`, `gateway`, `permissions`, `lanes`, `audit`, and API handlers). That coexistence is acceptable only if documented explicitly before further integration work.

The live core builds and vets cleanly. The FORGE-K test suite passes. Aggregate core tests currently fail in two API tests that are coupled to the host Windows home directory. Desktop readiness is blocked by an unhealthy local Node workspace (`@forge/shared`, `@forge/ui`, and `vitest` resolution failures). The API route extraction from the prior Phase A/B work is present: `Server.Handler()` is now a shell, route mounting is in `routes.go`, and representative route inventory guardrails pass.

## Current Status

- **Phase 0-6 FORGE-K simulator:** implemented and tested.
- **Phase 6 FORGE-K:** implemented as `SIMULATOR_ONLY`; no live daemon integration.
- **Phase 7+ FORGE-K:** documented, partial outside FORGE-K, research-only, or not started depending on phase.
- **Live daemon integration with FORGE-K:** not started.
- **Live API:** functional but still carries large server, gateway, chat, model runtime, backup, and controllane monoliths.
- **Build/vet:** core passes.
- **Full core test suite:** `npm test` passes in the Phase 6 implementation pass.
- **Desktop typecheck/build/tests:** blocked locally by Node workspace dependency resolution.
- **Security/safety posture:** generally governed, but there are evidence-based high-risk findings around model store path safety and secret result persistence.

## Repository Structure

Top-level shape:

| Area | Purpose |
| --- | --- |
| `apps/desktop` | Tauri/React desktop shell. |
| `services/core` | Go core daemon and live service implementation. |
| `services/core/internal/forgek` | FORGE-K simulator packages. |
| `packages/shared`, `packages/ui` | TypeScript shared contracts and UI package. |
| `docs` | Architecture, ADRs, roadmap, reviews, status, runbooks, testing, diagrams. |
| `scripts` | Build, smoke, and repository guard scripts. |
| `nix` | Optional Nix substrate and future packaging placeholders. |
| `.github` | CI workflow. |

Main Go entrypoint: `services/core/main.go`. It builds the live daemon and calls `api.NewServer`; it does not import `internal/forgek`.

Major `services/core/internal` packages include `api`, `aios`, `gateway`, `permissions`, `lanes`, `audit`, `modelruntime`, `memory`, `retrieval`, `jobs`, `backup`, `release`, `watch`, and the FORGE-K simulator package `forgek`.

No top-level `examples/` directory was found.

## Build and Test Health

Commands run during this audit and by the parallel review team:

| Command | Result | Notes |
| --- | --- | --- |
| `npm run build:core` | PASS | Runs VSA preflight and `cd services/core && go build ./...`. |
| `npm run lint` | PASS | Runs VSA preflight and `cd services/core && go vet ./...`. |
| `go test ./internal/forgek/...` | PASS | FORGE-K, court, palace, semantic, and neurons all pass. |
| `go test ./internal/api -run "TestServerRouteInventory|TestLegacyAdapterInvokeRouteRemoved|TestOpenAICompatRoutesDisabledByDefault|TestOpenAICompatRoutesAreAvailableWhenAutoEnabledViaCompatFlag|TestServerShutdownWatchIsIdempotent|TestPatchSettings" -count=1` | PASS | Route inventory, OpenAI compatibility route gating, legacy adapter removal, shutdown, and settings guardrails pass. |
| `npm test` / `go test ./...` | PASS | Rerun in the Phase 6 implementation pass; the previously host-coupled API tests pass. |
| `npm run typecheck` | FAIL | Desktop TS module resolution failures for `@forge/shared`, `@forge/ui`, `vitest`, plus follow-on implicit-any errors. |
| `npm -w @forge/desktop run build` | FAIL | Vite cannot resolve `@forge/ui` from desktop pages. |
| `npm -w @forge/desktop run test -- --run` | FAIL | `vitest` not found in current local install. |
| `npm run smoke` | FAIL locally | Windows shell routes to WSL, but `/bin/bash` is unavailable. |
| `npm audit --omit=dev --audit-level=moderate` | PASS | No production dependency vulnerabilities reported. |
| `npm audit --audit-level=moderate` | FAIL | Moderate dev vulnerability chain through Vite/esbuild; forced fix would be breaking. |
| `go mod verify` | PASS | All modules verified. |
| `govulncheck ./...` | NOT RUN | Tool not installed. |

### Aggregate Core Test Status

The previous audit reported two host-coupled failures in `services/core/internal/api`:

- `TestChatPostSyncRoutesDownloadSorterThroughGateway`
- `TestChatPostSyncMultiSVGUsesDeterministicGatewayShortcut`

Both host-path issues are now marked repaired, and `npm test` passes in the Phase 6 implementation pass.

### Desktop Dependency Health

The local Node workspace is unhealthy:

- `node_modules/@forge/shared` and `node_modules/@forge/ui` are not resolving as usable workspace packages.
- `vitest` is unavailable to the desktop workspace.
- `npm ci --dry-run` reported it would add many packages and repair workspace package state.

This blocks meaningful desktop typecheck/build/test validation on this machine.

## Phase Completion Matrix

| Phase | Status | Evidence | Gaps |
| --- | --- | --- | --- |
| Phase 0 — Architecture Baseline | IMPLEMENTED | FORGE-K docs, ADRs 0001-0005, glossary, roadmap, DoD, diagrams. | Keep live-authority boundary visible in future status reports. |
| Phase 1 — Kernel Simulator | IMPLEMENTED + TESTED | `services/core/internal/forgek/{kernel,objects,syscalls,journal,capabilities,case_syscalls}.go` and tests. | In-memory only; not live daemon authority. |
| Phase 2 — Neuron Fabric | IMPLEMENTED + TESTED | `services/core/internal/forgek/neurons/...` and tests. | Runtime/model-backed neurons deferred. |
| Phase 3 — Courthouse Minimal | IMPLEMENTED + TESTED | `court` package, `court_syscalls.go`, tests. | Claim extraction and full adjudication deferred. |
| Phase 4 — Memory Palace Minimal | IMPLEMENTED + TESTED | `palace` package, `palace_syscalls.go`, tests. | Embeddings/vector retrieval deferred. |
| Phase 5 — Semantic Algebra | IMPLEMENTED + TESTED | `semantic` package, `semantic_syscalls.go`, tests. | Advanced policy/optimization deferred. |
| Phase 6 — Snapshots | IMPLEMENTED + TESTED | `services/core/internal/forgek/snapshots/*`, `snapshot_syscalls.go`, `snapshot_syscalls_test.go`, snapshot package tests, `docs/architecture/snapshots.md`, ADR 0003; roadmap scope recorded as `SIMULATOR_ONLY`. | Persistence, live daemon integration, Context Compiler, token hashing, and deterministic KV cache remain deferred. |
| Phase 7 — Context Compiler | DOCUMENTED / PARTIAL OUTSIDE FORGE-K | `docs/architecture/context_compiler_and_kv_cache.md`; live `aios/controllane/compile_context_*` exists but is not the FORGE-K deterministic ContextBlock compiler. | FORGE-K ContextBlock, token hashing, compiler loop absent. |
| Phase 8 — Deterministic KV System | DOCUMENTED ONLY | ADR 0004 and context/KV architecture doc. | No KV manifest, nine-gate validation, or tier code. |
| Phase 9 — Runtime Driver Integration | PARTIAL OUTSIDE FORGE-K | Live `modelruntime`, `gateway`, `aios/iolane` exist. | No FORGE-K runtime-driver boundary. |
| Phase 10 — Lymphatic Lane | PARTIAL OUTSIDE FORGE-K | Live dream/autonomy cleanup-style paths exist. | No FORGE-K lymphatic scheduler. |
| Phase 11 — Rust Kernel Core | NOT STARTED | None. | All work remains. |
| Phase 12 — FORGE Daemon | PARTIAL OUTSIDE FORGE-K | Existing `forge-core` daemon. | Not FORGE-K-governed. |
| Phase 13 — FORGE-1 Simulator | NOT STARTED | Concept doc only. | All work remains. |
| Phase 14 — FORGE-1 Prototype Research | DOCUMENTED CONCEPT ONLY | `docs/architecture/forge_1_cpu_concept.md`. | Research/prototype not started. |

## Architecture Boundary Findings

| ID | Severity | Finding | Evidence | Why It Matters | Recommended Fix |
| --- | --- | --- | --- | --- | --- |
| AB-01 | HIGH | FORGE-K simulator is not live authority. | `rg` found `forgek` imports only inside `internal/forgek` and its tests. | Doctrine says FORGE-K owns truth, but the live daemon does not route through FORGE-K yet. | ADR 0005 records the boundary; keep Phase 6 simulator-only unless a later integration phase is explicitly scoped. |
| AB-02 | HIGH | Live memory mutation remains callable in-process outside semantic syscalls. | `services/core/internal/memory/service.go`, `retrieval.go`; HTTP legacy mutation routes are gated but service mutators remain exported. | A future caller could bypass the intended semantic syscall path. | Move canonical memory writes behind controllane/FORGE-K boundaries or add static/import tests. |
| AB-03 | HIGH | Gateway secret retrieval can persist plaintext in invocation results. | `gateway/capability_backing_tool.go` returns plaintext; `gateway/service.go` persists result JSON. | Secrets can become durable records. | Return handles or redact sensitive fields before persistence. |
| AB-04 | HIGH | Model store paths do not appear to validate model IDs as safe path segments. | `modelruntime/store_management.go`, `store.go`, `manifest.go`. | Path traversal or unsafe directory names could affect managed model storage. | Add one path-safe model ID validator and safe-join tests. |
| AB-05 | MEDIUM | Live API relies on CORS/localhost posture rather than visible auth on broad route groups. | `api/routes.go` CORS and `/forge`, `/api`, `/v1` mounts. | CORS is not authentication if core binds beyond trusted desktop use. | Add local token/auth plan before remote/broader binding. |
| AB-06 | MEDIUM | Two authority systems coexist: FORGE-K simulator and live AI-OS/gateway/permissions. | `AGENTS.md`, `internal/aios`, `internal/gateway`, `internal/forgek`. | New contributors can mistake docs for live enforcement. | Document current authority split and reconciliation phase. |
| AB-07 | MEDIUM | SQLite controllane transaction internals use a background context in store methods. | `aios/controllane/sqlite_store.go`. | Long DB work can ignore request cancellation after transaction start. | Thread request context through semantic store methods. |
| AB-08 | LOW | Some internal FORGE-K constructors panic on internal manifest/registration errors. | `forgek/kernel.go`, `neurons/neural.go`, `neurons/rule.go`. | Acceptable for current simulator but not for future runtime driver inputs. | Revisit before Phase 9 runtime integration. |

Positive boundary evidence:

- FORGE-K `Kernel.DispatchSyscall` capability-gates side-effect syscalls and verifies journal growth for journal-required syscalls.
- FORGE-K object registry and journal return defensive copies.
- Neuron outputs are envelopes or syscall requests; they do not mutate kernel state directly.
- Memory Palace candidates do not become Exhibits automatically.
- Semantic operators preserve provenance and do not execute Courthouse requests themselves.

## Monolith and Complexity Findings

| File / Module | Approx. LOC | Responsibilities | Risk | Recommended Split |
| --- | ---: | --- | --- | --- |
| `services/core/internal/gateway/service.go` | 3596 | Gateway policy, approvals, persistence, tool implementations, shell/git/fs/network execution, fingerprinting. | HIGH | Split tool execution, policy/evaluation, persistence, approval correlation, and tool adapters. |
| `services/core/internal/aios/controllane/sqlite_store.go` | 2102 | Semantic persistence, snapshots, restore outcomes, provenance, audit linking, transactions. | HIGH | Split stores by aggregate: objects/journal/snapshots/restore/provenance/transactions. |
| `services/core/internal/api/chat_assistant_gateway.go` | 1942 | Chat gateway, tool routing, deterministic fallbacks, model interaction, response formatting. | HIGH | Split routing, model call adapter, tool manifest, fallback planner, response renderer. |
| `services/core/internal/backup/service.go` | 1857 | Backup bundle creation, restore, validation, metadata, filesystem operations. | MEDIUM | Split bundle IO, manifest validation, restore planner, store adapter. |
| `services/core/internal/api/autonomy_maintenance_loop.go` | 1479 | Background autonomy maintenance, scheduling, policy, reporting. | MEDIUM | Split scheduler, sweep planner, report builder, execution guard. |
| `services/core/internal/modelruntime/service.go` | 1439 | Registry, lifecycle, scheduler, GPU policy, generation, health. | HIGH | Split registry/lifecycle/scheduler/backend/generation/health. |
| `services/core/internal/api/server.go` | 1293 | Server fields, constructor wiring, lifecycle startup, settings/source/command handlers, health. | HIGH | Continue Phase C/D/E refactor: settings/meta/health handlers, dependency construction, lifecycle startup. |
| `services/core/internal/api/routes.go` | ~340 | Route mounting only. | MEDIUM | Acceptable as Phase A/B extraction; later split by route domain once guarded. |

The API refactor has started safely: `Handler()` is a shell and route mounting is extracted. It should not become the new permanent monolith. The next API pass should move handlers and dependencies, not add more routes.

## API / Server Audit

Current route structure:

- Global middleware: request ID, real IP, logger, recoverer, CORS.
- `/health`.
- `/forge` with 120-second timeout.
- `/v1` conditionally mounted only when `EnableOpenAICompatAPI` is true, with 120-second timeout.
- `/api/chat/threads/{id}/assistant-stream` mounted outside the timed `/api` group.
- Other `/api` routes mounted inside the 120-second timeout group.
- Representative route inventory tests now cover health, `/forge`, `/v1`, assistant stream order, major `/api` route groups, compatibility VSA routes, retired memory mutation routes, and removed legacy adapter invoke route.

Remaining server risks:

- `NewServer` still constructs many services and starts background services.
- `Server` still owns too many dependencies.
- Settings, source/search, command, and health handlers still live in `server.go`.
- Background lifecycle start/stop ownership is still mixed into construction.

Before adding new API routes:

1. Keep route inventory tests passing.
2. Extract settings/meta/health handlers.
3. Introduce dependency grouping without changing `NewServer` behavior.
4. Move background startup toward explicit lifecycle helpers.

## Domain Status

| Domain | Status | What Works | Missing / Risk |
| --- | --- | --- | --- |
| Kernel/object registry | Implemented and tested in FORGE-K simulator. | Capability-gated dispatch, copy-on-read registry, journal-required mutation. | Not persistent; not wired into live daemon. |
| Semantic syscall registry | Implemented and tested. | Registers case/court/palace/semantic syscalls. | Lane metadata is not yet a full scheduler. |
| Capability manager | Implemented and tested. | Subject, workspace, syscall, mutation scope, expiration checks. | Delegation/audit flags are minimal. |
| Journal | Implemented and tested. | Append-only hash chain, cloned reads. | No replay/persistence tooling yet. |
| CasePacket lifecycle | Implemented and tested. | OPEN/UPDATED/CLOSED and closed update rejection. | Reopen/future policy deferred. |
| Neuron Fabric | Implemented and tested. | Manifest, envelope, scheduler, neural proposals, rule validations, syscall requests. | No runtime/model neurons. |
| Courthouse | Implemented and tested. | Exhibit submit/admit/reject, rulings, contradictions, supersession. | No full reasoning engine. |
| Memory Palace | Implemented and tested. | Rooms, anchors, routes, candidate objects, deterministic scoring. | No vector/embedding integration. |
| Semantic Algebra | Implemented and tested. | MERGE/DIFF/INTERSECT/CONTRADICT/SUPERSEDE/COMPRESS/DERIVE/PROMOTE/DEMOTE/EXPIRE and request-only boundary ops. | No advanced algebra planner. |
| Snapshots | Implemented and tested in FORGE-K simulator. | Models, lifecycle service, syscalls, deterministic shape hashing, diffs, restore seeds, capability gates, journal events, and shape-not-truth tests. | Persistence and live daemon integration deferred. |
| Context Compiler | Documented only in FORGE-K. | Live restore scoring exists outside FORGE-K. | Need ContextBlock, stable token serialization, compiler loop. |
| Deterministic KV cache | Documented only. | Doctrine and ADR exist. | No manifest or nine-gate validation. |
| Runtime drivers | Live system partial; FORGE-K not started. | Model runtime M3-like surface exists. | FORGE-K driver boundary absent. |
| Tool drivers | Live gateway exists. | Gateway policies and approvals exist. | FORGE-K Motor neuron/driver wrappers absent. |
| Lymphatic cleanup | Live dream/autonomy partial; FORGE-K not started. | Deterministic dream/report paths exist. | FORGE-K lymphatic lane absent. |
| FORGE-1 | Concept only. | Grounded docs exist. | No simulator/prototype. |
| Examples | Missing. | None found. | Phase demos should be added once structure stabilizes. |

## Test Gap Priorities

| Priority | Gap |
| --- | --- |
| BLOCKER | Rerun `go test ./...` / `npm test` after the two host-coupled API tests marked repaired in the checklist. |
| HIGH | Add model store path traversal tests for import/load/archive/remove. |
| HIGH | Add gateway secret redaction/persistence tests. |
| HIGH | Keep Phase 6 snapshot shape-not-truth tests passing during snapshot changes. |
| HIGH | Add context block deterministic serialization and token hashing tests before Phase 7. |
| HIGH | Add KV nine-gate validation tests before Phase 8. |
| HIGH | Keep API route inventory tests as guardrails for all route refactors. |
| MEDIUM | Add shutdown/lifecycle tests after lifecycle extraction. |
| MEDIUM | Add static/import tests for legacy memory mutation boundary. |
| MEDIUM | Add local API auth tests if/when token auth is introduced. |
| LOW | Add package tests opportunistically for `[no test files]` packages when they are touched. |

## Documentation Audit

Accurate/current:

- FORGE-K architecture overview, core doctrine, kernel simulator, neuron fabric, lane model, memory palace/courthouse, semantic algebra, snapshots, context/KV, FORGE-1 concept.
- ADRs 0001-0005.
- Glossary, roadmap, testing definition of done.
- Server refactor review and route inventory review, now with Phase A/B status.

Drift or missing:

- ADR 0005 now records the FORGE-K simulator vs live AI-OS authority boundary.
- `docs/status/implementation_matrix.md` is legacy-oriented and does not clearly map FORGE-K Phase 0-14.
- `docs/reviews/full_project_review.md` and companions needed refresh after route extraction; this audit resolves that.
- `services/core/internal/forgek/README.md` is missing.
- `internal/api/phase*.go` filenames can be confused with FORGE-K phases.
- Several `docs/status/*` files describe earlier AI-OS status and need cross-links to FORGE-K status.

## TODO / FIXME / Stub Summary

High-signal findings from the marker sweep:

- `internal/forgek/...` has no significant TODO/FIXME/HACK markers in runtime code.
- Existing status docs already catalog many runtime-adjacent stubs and placeholders, especially in `docs/status/placeholders_and_stubs.md` and `docs/status/stubs_placeholders_examples.md`.
- Live AI-OS still includes deterministic stub language around `COMPILE_CONTEXT`; this is documented as non-canonical evidence but must be reconciled before the FORGE-K Context Compiler phase.
- Nix tool capsules, modules, and profiles are explicit documentation-only placeholders.
- Desktop source has many UI placeholder strings; these are not architectural risks.

## Dependency and Configuration Findings

| Severity | Finding | Evidence / Impact | Recommended Fix |
| --- | --- | --- | --- |
| HIGH | Local Node workspace install is unhealthy. | Desktop cannot resolve `@forge/shared`, `@forge/ui`, or `vitest`. | Run a clean `npm ci` path and verify workspace links; document Node/npm versions. |
| HIGH | Windows home path handling breaks two API tests. | `os.UserHomeDir()` resolves to real home, not test HOME. | Use `t.Setenv("USERPROFILE", tempHome)` or fixture paths rooted in `t.TempDir()`. |
| MEDIUM | Smoke script is Unix-assumptive. | `npm run smoke` fails locally because WSL `/bin/bash` unavailable. | Add a PowerShell smoke path or document Linux-only smoke. |
| MEDIUM | Local Node version differs from CI. | Audit host used Node 24/npm 11 while CI uses Node 22. | Add `engines`/tooling note if Node 22 is required. |
| MEDIUM | Go vulnerability scan unavailable. | `govulncheck` not installed; older indirect `golang.org/x/*` modules exist. | Add `govulncheck` to documented optional checks or CI. |
| LOW | Dev dependency audit reports Vite/esbuild moderate vulnerabilities. | `npm audit --audit-level=moderate` fails for dev deps. | Plan dependency upgrade separately; do not force breaking update during stabilization. |

## Performance / Hot Path Concerns

- Keep FORGE-K dispatch small: syscall lookup, validation, capability check, handler execution, journal verification.
- Avoid running the full architecture synchronously on every chat/API turn.
- `memory.Service` list paths appear N+1-heavy; reserve detail hydration for detail endpoints.
- Chat assistant/gateway path logs many stages synchronously; buffer or sample if it becomes hot.
- Gateway process tools should cap stdout/stderr consistently.
- Constructor-time service startup in `NewServer` should move toward explicit lifecycle control.

## Security / Safety Concerns

Evidence-based concerns:

- Model store path safety needs explicit model ID validation.
- Secret retrieval should not persist plaintext result data.
- Local API route groups are broad; CORS is not authentication.
- Legacy memory write services remain exported and callable in process.
- Gateway process execution output capture needs consistent caps.

Positive evidence:

- Gateway lane denial is working; the failing API tests prove the deny path blocks out-of-scope file writes.
- FORGE-K simulator does not call models, execute tools, use network, or write filesystem.
- Chat path discards ungoverned model prose when a forced tool route omits governed tool calls.

## Risk Register

| ID | Risk | Severity | Likelihood | Affected Area | Evidence | Mitigation |
| --- | --- | --- | --- | --- | --- | --- |
| R-01 | FORGE-K is implemented but not live authority. | HIGH | HIGH | architecture/runtime | No non-test imports outside `internal/forgek`. | ADR 0005; Phase 6 scope recorded as simulator-only. |
| R-02 | Aggregate core tests are now green. | LOW | LOW | CI/dev confidence | Prior API path-scope failures were repaired and `npm test` passes in the Phase 6 implementation pass. | Keep aggregate tests green during future changes. |
| R-03 | Desktop validation blocked by dependency state. | HIGH | HIGH | desktop | `@forge/shared`, `@forge/ui`, `vitest` resolution failures. | Repair workspace install and document Node version. |
| R-04 | Server/API still has lifecycle and dependency monolith risks. | HIGH | HIGH | API | `server.go` still owns construction/lifecycle/handlers. | Continue phased refactor after guardrails. |
| R-05 | Gateway service is too large and owns too much execution logic. | HIGH | HIGH | gateway/security | 3596 LOC. | Split execution, policy, persistence, and adapters. |
| R-06 | Model store path traversal risk. | HIGH | MEDIUM | model runtime | Unsafe path segment evidence. | Validate model IDs and safe joins. |
| R-07 | Secret plaintext can be persisted. | HIGH | MEDIUM | gateway/secrets | Secret result data persisted. | Redact or use handles. |
| R-08 | Live memory mutation can bypass future semantic syscall expectations in process. | HIGH | MEDIUM | memory/kernel boundary | Exported legacy mutators. | Restrict writes to controllane/FORGE-K boundary. |
| R-09 | Context compiler doctrine overlaps with live compile-context path. | MEDIUM | HIGH | Phase 7 | Existing `aios/controllane/compile_context_*`. | Define relationship before Phase 7. |
| R-10 | Background services start in constructor. | MEDIUM | HIGH | tests/lifecycle | `NewServer` side effects. | Extract lifecycle helpers. |
| R-11 | Local API attack surface broad if bound beyond desktop. | MEDIUM | MEDIUM | security | No visible auth middleware on broad groups. | Add token/auth plan. |
| R-12 | TODO/stub status spread across legacy docs. | MEDIUM | MEDIUM | docs/onboarding | Multiple `docs/status/*` realities. | Add reconciliation map. |

## Must Finish Before Moving Further

This list is strict. Not every risk is a blocker.

| Class | Item |
| --- | --- |
| BLOCKER | Keep `npm test` / `go test ./...` passing after the host-coupled API test repairs. |
| BLOCKER | Keep ADR 0005's FORGE-K simulator vs live AI-OS/gateway authority boundary enforced before any FORGE-K work touches live state. |
| HIGH | Repair local Node workspace dependency resolution before making desktop-facing claims. |
| HIGH | Keep Phase 6 Snapshots simulator-only unless a later live integration phase explicitly changes scope. |
| HIGH | Add model store path safety tests and a remediation plan. |
| HIGH | Add gateway secret redaction/persistence tests and a remediation plan. |
| HIGH | Continue server refactor Phase C/D only behind current route inventory tests. |
| MEDIUM | Keep `services/core/internal/forgek/README.md` current with simulator scope and test command. |
| MEDIUM | Update implementation/status docs to show FORGE-K Phase 0-14 alongside legacy AI-OS phases. |

## Recommended Next 3 Phases of Work

1. **Stabilization Phase**
   - Keep aggregate core tests passing after the two API tests marked repaired in the checklist.
   - Repair local Node workspace dependency resolution.
   - Re-run `npm test`, `npm run typecheck`, and desktop build/test.
   - Add `govulncheck` availability or document it as optional.

2. **Authority Reconciliation Phase**
   - Maintain ADR 0005 authority boundary.
   - Keep `AGENTS.md`, `docs/status/implementation_matrix.md`, and `docs/roadmap/forge_k_build_phases.md` clear that FORGE-K Phase 1-6 is simulator-only and the live daemon remains on AI-OS/gateway authority until integration.
   - Keep Phase 7 scope explicit before implementation.

3. **Refactor and Safety Phase**
   - Continue `server.go` Phase C/D/E: settings/meta/health extraction, dependency grouping, lifecycle extraction.
   - Add path safety and secret persistence tests before implementing those fixes.
   - Plan FORGE-K Phase 7 Context Compiler only after deterministic ContextBlock serialization and token hashing test design is recorded.

## Acceptance Criteria Before New Feature Work Resumes

- `npm test` passes.
- `npm run build:core` and `npm run lint` pass.
- Desktop dependency resolution is repaired enough for `npm run typecheck` to produce meaningful results.
- ADR 0005 exists.
- Route inventory tests stay green.
- The Phase 6 scope decision and implementation status are recorded.
- High-risk path safety and secret persistence findings have tests or an approved stabilization task.

## Recommended Next Cursor/Codex Prompt

```text
You are working in the FORGE repository.

Task: Rerun project readiness checks before new feature work.

This is a focused validation pass. Do not add features or change public APIs.

Read:
- docs/reviews/full_project_review.md
- docs/reviews/archive/full_project/full_project_review_checklist.md
- docs/reviews/current_phase_status.md
- AGENTS.md
- docs/adr/0005-forge-k-simulator-vs-live-authority.md

Goals:
1. Rerun `npm test` after the two host-coupled API tests marked repaired in the checklist.
2. Rerun `npm run build:core` and `npm run lint`.
3. Do not weaken gateway lane enforcement.
4. Preserve existing route behavior and public APIs.
5. Keep FORGE-K as simulator authority only; do not wire it into the live daemon.
6. Document exact commands, results, and any remaining blockers.

If validation completes, stop. Do not begin Phase 6 implementation, server refactors, modelruntime fixes, or desktop dependency changes in the same pass.
```
