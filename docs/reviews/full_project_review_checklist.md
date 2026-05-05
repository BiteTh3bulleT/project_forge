# FORGE Full Project Review Checklist

Companion to `docs/reviews/full_project_review.md` (2026-05-03).

Severity legend: **B** = blocker, **H** = high, **M** = medium, **L** = low.

## Blocking Fixes

- [x] **B** Repair `TestChatPostSyncRoutesDownloadSorterThroughGateway` so it does not depend on the real Windows user Downloads directory.
- [x] **B** Repair `TestChatPostSyncMultiSVGUsesDeterministicGatewayShortcut` for the same host-path issue.
- [x] **B** Add ADR 0005: FORGE-K simulator authority vs live AI-OS/gateway authority.

## High-Priority Stabilization

- [ ] **H** Repair local Node workspace dependency resolution so `@forge/shared`, `@forge/ui`, and `vitest` resolve for desktop validation.
- [x] **H** Decide whether Phase 6 Snapshots is simulator-only or the first live integration step. Recorded as `SIMULATOR_ONLY`.
- [ ] **H** Add model runtime path safety tests for unsafe model IDs and managed-store safe joins.
- [ ] **H** Add gateway secret result persistence tests and define redaction/handle behavior.
- [ ] **H** Keep `services/core/internal/api/server_route_inventory_test.go` passing during all API refactors.

## API Refactor Continuation

- [x] **H** Add representative route inventory guardrails.
- [x] **H** Extract route mounting from `Server.Handler()` into `routes.go`.
- [ ] **H** Phase C: extract settings/meta/health handlers out of `server.go`.
- [ ] **H** Phase D: introduce `ServerDependencies` or grouped dependency structs without changing `NewServer` public behavior.
- [ ] **M** Phase E: move background service startup toward explicit lifecycle helpers.
- [ ] **M** Add route inventory expansion only when new route groups are touched.

## Tests To Add

- [x] **H** Snapshot shape-not-truth tests for Phase 6 implementation.
- [x] **H** ContextBlock deterministic serialization tests for Phase 7.
- [x] **H** Token input hashing identity tests for Phase 7.
- [x] **H** KV nine-gate validation tests for Phase 8.
- [ ] **M** Static/import guard for legacy memory mutation boundaries.
- [ ] **M** Shutdown/lifecycle tests after lifecycle extraction.
- [ ] **M** Local API auth tests if token auth is introduced.
- [ ] **L** Add focused tests to `[no test files]` packages when those packages are touched.

## Docs To Update

- [x] **B** Create ADR 0005.
- [x] **H** Update `AGENTS.md` with a short authority coexistence note once ADR 0005 exists.
- [x] **H** Update `docs/status/implementation_matrix.md` with legacy/live AI-OS boundary note and FORGE-K status cross-link.
- [x] **M** Update `docs/roadmap/forge_k_build_phases.md` with the Phase 6 scope decision.
- [x] **M** Add `services/core/internal/forgek/README.md`.
- [x] **M** Document Phase 9 Runtime Driver Boundary as `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`.
- [x] **M** Document Phase 10 Lymphatic Lane as `SIMULATOR_ONLY`, maintenance reports/proposals only, with no live daemon, dream/autonomy, cleanup, route, gateway, modelruntime, or controllane wiring.
- [x] **M** Complete Phase 11A Rust Kernel Core research/planning docs and ADR 0006 without Rust implementation.
- [x] **M** Document Phase 11C Go/Rust test corpus alignment as `RESEARCH_ONLY / SIMULATOR_ONLY`.
- [x] **M** Document Phase 12D Controlled Shadow Expansion Design as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.
- [ ] **M** Add comments or docs that `internal/api/phase*.go` names refer to legacy FORGE feature phases, not FORGE-K phases.
- [ ] **L** Document Windows smoke limitations or add a PowerShell smoke path.

## Security / Safety Follow-Ups

- [ ] **H** Validate model IDs as path-safe segments in the model runtime managed store.
- [ ] **H** Redact or avoid persisting plaintext secret retrieval results.
- [ ] **M** Add output caps for gateway process tool stdout/stderr.
- [ ] **M** Define local API token/auth posture before binding outside trusted desktop-only contexts.
- [ ] **M** Thread request context through controllane SQLite transaction store methods.

## Safe Next Tasks

- [x] Stabilize the two failing API tests.
- [ ] Repair Node workspace dependency state and rerun desktop validation.
- [x] Write ADR 0005.
- [ ] Continue API refactor Phase C only after tests are green.
- [x] Implement Phase 6 Snapshots with snapshot invariant tests. Scope is recorded as `SIMULATOR_ONLY`.
- [x] Implement Phase 7 Context Compiler with deterministic serialization, token input hash, and shape-not-truth tests. Scope is recorded as `SIMULATOR_ONLY`.

## Deferred Work

- [x] FORGE-K Phase 7 Context Compiler. Scope is recorded as `SIMULATOR_ONLY`.
- [x] FORGE-K Phase 8 Deterministic KV System. Scope is recorded as `SIMULATOR_ONLY`.
- [x] FORGE-K Phase 9 Runtime Driver Integration implementation. Boundary scope is recorded as `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`.
- [x] FORGE-K Phase 10 Lymphatic Lane simulator implementation. Scope is recorded as `SIMULATOR_ONLY`.
- [x] FORGE-K Phase 11A Rust Kernel Core research/planning. Scope is `RESEARCH_ONLY / DOCS_ONLY`.
- [x] FORGE-K Phase 11B Rust deterministic validation crate.
- [x] FORGE-K Phase 11C Go/Rust test corpus alignment.
- [x] FORGE-K Phase 12D Controlled Shadow Expansion Design. Scope is `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.
- [x] FORGE-K Phase 12A Live Integration Design. Scope is recorded as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`; no live implementation started.
- [x] FORGE-K Phase 12B Read-only Shadow Harness Implementation. Scope is recorded as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`; `/health` metadata only, disabled by default, diagnostic-only.
- [x] FORGE-K Phase 12C Shadow Diagnostics Review and Hardening. Scope is recorded as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`; no new live touchpoints or diagnostics APIs added.
- [ ] FORGE-K Phase 12 FORGE Daemon.
- [ ] FORGE-1 phases 13-14.
- [ ] Gateway service decomposition.
- [ ] Model runtime service decomposition.
- [ ] Controllane SQLite store decomposition.

## Acceptance Criteria Before New Feature Work

- [x] `npm test` passes.
- [x] `npm run build:core` passes.
- [x] `npm run lint` passes.
- [ ] Desktop dependency resolution is repaired enough for meaningful typecheck/build results.
- [x] ADR 0005 exists.
- [x] Route inventory guardrails pass.
- [x] Phase 6 scope decision is recorded.
- [x] Phase 6 snapshot simulator tests pass under `go test ./internal/forgek/...`.
- [x] Phase 7 Context Compiler simulator tests pass under `go test ./internal/forgek/...`.
- [x] Phase 8 Deterministic KV simulator tests pass under `go test ./internal/forgek/...`.
- [x] Phase 9 Runtime Driver Boundary scope is recorded before implementation.
- [x] Phase 9 Runtime Driver Boundary simulator tests pass under `go test ./internal/forgek/...`.
- [x] Phase 10 Lymphatic Lane scope is recorded as `SIMULATOR_ONLY`.
- [x] Phase 10 Lymphatic Lane simulator tests pass under `go test ./internal/forgek/...`.
- [x] Phase 11A Rust Kernel Core planning scope is recorded as `RESEARCH_ONLY / DOCS_ONLY`.
- [x] ADR 0006 records the Rust boundary without marking Rust implementation complete.
- [x] Phase 11C Go/Rust fixture parity passes under `go test ./internal/forgek/...`, `cargo test`, Rust fixture validation, and `npm run test:forgek:parity`.
