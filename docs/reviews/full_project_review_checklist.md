# FORGE Full Project Review Checklist

Companion to `docs/reviews/full_project_review.md` (2026-05-03).

Severity legend: **B** = blocker, **H** = high, **M** = medium, **L** = low.

## Blocking Fixes

- [x] **B** Repair `TestChatPostSyncRoutesDownloadSorterThroughGateway` so it does not depend on the real Windows user Downloads directory.
- [x] **B** Repair `TestChatPostSyncMultiSVGUsesDeterministicGatewayShortcut` for the same host-path issue.
- [ ] **B** Add ADR 0005: FORGE-K simulator authority vs live AI-OS/gateway authority.

## High-Priority Stabilization

- [ ] **H** Repair local Node workspace dependency resolution so `@forge/shared`, `@forge/ui`, and `vitest` resolve for desktop validation.
- [ ] **H** Decide whether Phase 6 Snapshots is simulator-only or the first live integration step.
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

- [ ] **H** Snapshot shape-not-truth tests before Phase 6 implementation.
- [ ] **H** ContextBlock deterministic serialization tests before Phase 7.
- [ ] **H** Token hashing identity tests before Phase 7.
- [ ] **H** KV nine-gate validation tests before Phase 8.
- [ ] **M** Static/import guard for legacy memory mutation boundaries.
- [ ] **M** Shutdown/lifecycle tests after lifecycle extraction.
- [ ] **M** Local API auth tests if token auth is introduced.
- [ ] **L** Add focused tests to `[no test files]` packages when those packages are touched.

## Docs To Update

- [ ] **B** Create ADR 0005.
- [ ] **H** Update `AGENTS.md` with a short authority coexistence note once ADR 0005 exists.
- [ ] **H** Update `docs/status/implementation_matrix.md` with FORGE-K Phase 0-14 status.
- [ ] **M** Update `docs/roadmap/forge_k_build_phases.md` with the Phase 6 scope decision.
- [ ] **M** Add `services/core/internal/forgek/README.md`.
- [ ] **M** Add comments or docs that `internal/api/phase*.go` names refer to legacy FORGE feature phases, not FORGE-K phases.
- [ ] **L** Document Windows smoke limitations or add a PowerShell smoke path.

## Security / Safety Follow-Ups

- [ ] **H** Validate model IDs as path-safe segments in the model runtime managed store.
- [ ] **H** Redact or avoid persisting plaintext secret retrieval results.
- [ ] **M** Add output caps for gateway process tool stdout/stderr.
- [ ] **M** Define local API token/auth posture before binding outside trusted desktop-only contexts.
- [ ] **M** Thread request context through controllane SQLite transaction store methods.

## Safe Next Tasks

- [ ] Stabilize the two failing API tests.
- [ ] Repair Node workspace dependency state and rerun desktop validation.
- [ ] Write ADR 0005.
- [ ] Continue API refactor Phase C only after tests are green.
- [ ] Begin Phase 6 Snapshots only after the scope decision and snapshot invariant tests exist.

## Deferred Work

- [ ] FORGE-K Phase 7 Context Compiler.
- [ ] FORGE-K Phase 8 Deterministic KV System.
- [ ] FORGE-K Phase 9 Runtime Driver Integration.
- [ ] FORGE-K Phase 10 Lymphatic Lane.
- [ ] FORGE-K Phase 11 Rust Kernel Core.
- [ ] FORGE-K Phase 12 FORGE Daemon.
- [ ] FORGE-1 phases 13-14.
- [ ] Gateway service decomposition.
- [ ] Model runtime service decomposition.
- [ ] Controllane SQLite store decomposition.

## Acceptance Criteria Before New Feature Work

- [ ] `npm test` passes.
- [ ] `npm run build:core` passes.
- [ ] `npm run lint` passes.
- [ ] Desktop dependency resolution is repaired enough for meaningful typecheck/build results.
- [ ] ADR 0005 exists.
- [ ] Route inventory guardrails pass.
- [ ] Phase 6 scope decision is recorded.
