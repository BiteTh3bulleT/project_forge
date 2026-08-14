# FORGE-K Operational Cutover Design

Status: Historical Phase 14 design plus active K20A-K20B cutover. Production FORGE-K owns default syscall ingress and durable stage orchestration; Control Lane remains the temporary validation/apply/SQLite port implementation. Full FORGE-K authority is not yet complete.

Date: 2026-05-09. Updated: 2026-08-14.

## Executive Summary

FORGE-K is largely complete as a simulator architecture, but it is not yet live daemon authority. Making it operational means migrating narrow authority decisions into existing live owners with explicit gates, tests, rollback, and operator visibility.

The original Phase 14 portion of this document defined the staged cutover model. K20A now begins that migration through the production-owned `internal/forgekernel` boundary. The simulator remains isolated and is not live authority.

See `docs/architecture/forge_k_live_cutover.md` and ADR 0017 for the active completion gates and migration order.

## Current Authority Split

FORGE-K simulator packages under `services/core/internal/forgek` define target behavior for Kernel authority, Courthouse admission, Memory Palace shape, Semantic Algebra, Snapshots, Context Compiler, KV metadata, Runtime Boundary, Lymphatic Lane, Consensus Mesh, readiness contracts, and shadow reports.

Live daemon authority remains with existing systems:

- API routes: `services/core/internal/api`
- semantic mutation: `services/core/internal/aios/controllane`
- tool execution: `services/core/internal/gateway`
- policy gates: `services/core/internal/permissions` and `services/core/internal/lanes`
- audit: `services/core/internal/audit`
- model generation: `services/core/internal/modelruntime`
- retrieval/search/embeddings/memory: existing live packages and governed commit paths

Phase i1 and PhaseI2 demonstrate the only approved partial live migration pattern so far: extract a pure deterministic contract into a shared package, then call it from the live Control Lane without importing simulator services or enabling live KV reuse.

## Operational Cutover Rule

FORGE-K becomes operational one authority seam at a time.

No phase may import simulator services such as `forgek.Kernel`, `KVService`, `ContextCompilerService`, `RuntimeService`, `ConsensusService`, or simulator syscalls into live daemon paths and call that live authority. A live migration must either:

1. extract a pure deterministic contract into a shared package with no side effects, or
2. build a live-owned adapter that preserves the current live authority owner.

## Recommended First Operational Surface

Recommended first surface: live Control Lane semantic validation, not full Kernel replacement.

Reasoning:

- Control Lane already owns semantic validation and mutation boundaries.
- Phase i1/I2 already proves the shared-contract pattern with KV identity validation.
- A validation-only seam can fail closed without changing routes, gateway, modelruntime, retrieval, or memory behavior.
- It avoids creating a second Kernel or making shadow diagnostics authoritative.

The first Phase 14 implementation should choose one deterministic FORGE-K validation contract that can be extracted safely, such as:

- canonical ref normalization,
- capability predicate validation,
- journal hash-chain validation,
- semantic object shape validation,
- context block shape validation without live prompt authority.

It should not start with live Context Compiler prompt assembly, live Courthouse evidence admission, live Memory Palace retrieval ranking, live Consensus Mesh response authority, live runtime drivers, live KV reuse, or live memory writes.

## Cutover Stages

### Stage 0: Design And Readiness

- identify one live authority owner,
- identify one deterministic contract,
- document accepted inputs and outputs,
- document failure behavior,
- document rollback,
- prove no public API or route behavior changes are required.

### Stage 1: Shared Pure Contract

- create or extend a shared package outside `services/core/internal/forgek`,
- keep inputs immutable and explicit,
- forbid live side effects,
- forbid simulator service imports,
- add focused unit tests,
- keep simulator behavior passing.

### Stage 2: Live Validation-Only Integration

- call the shared package from the existing live authority owner,
- require capability checks where the live path already uses capabilities,
- fail closed on malformed or unsupported payloads,
- record audit fields or diagnostic counters,
- do not mutate memory, call modelruntime, execute tools, execute retrieval, or change response composition.

### Stage 3: Shadow Compare

- compare live decisions with simulator-style expected decisions using safe refs and metadata only,
- keep reports diagnostic-only,
- keep flags disabled by default unless explicitly approved,
- preserve no-effect guarantees.

### Stage 4: Limited Live Authority Migration

- migrate one decision class only after Stage 2 and Stage 3 are stable,
- retain the existing live owner as the commit boundary,
- add rollback to previous live behavior,
- add operator-visible status,
- document exact authority transfer.

## Required Tests

Every operational cutover phase must include:

- simulator package tests,
- shared pure package tests,
- live owner acceptance tests,
- malformed input fail-closed tests,
- capability denial tests,
- audit or diagnostic field tests,
- no route/public API change tests when applicable,
- no gateway/tool execution tests,
- no modelruntime call tests,
- no retrieval/search/embedding execution tests,
- no memory write tests unless the phase explicitly migrates memory authority,
- no simulator-service live import tests,
- rollback or disabled-mode tests.

## Rollback

Every phase must define:

- feature flag or config default,
- how to disable the new validation path,
- expected live behavior after disablement,
- data or audit records that may remain,
- tests proving old behavior remains available.

For validation-only phases, rollback should be config disablement or commit revert. For any authority migration, rollback must be tested before merge.

## Go / No-Go Gates

Go:

- deterministic contract is pure and tested,
- live owner is explicit,
- no simulator service is imported into live authority,
- failure mode is fail-closed or diagnostic-only as designed,
- rollback path is tested,
- docs/status/roadmap are updated.

No-go:

- route/API behavior changes without explicit approval,
- gateway/modelruntime/retrieval/memory behavior changes outside the chosen live owner,
- raw prompt/content/vector/secret capture,
- unbounded diagnostic persistence,
- importing FORGE-K simulator services as live authority,
- treating shadow advisory output as truth,
- treating Consensus accepted as canonical truth,
- treating Qdrant scores or KV hits as evidence.

## What Not To Do

- Do not import the in-memory simulator Kernel into the live daemon or claim the staged production ingress facade is already the full completed Kernel.
- Do not create a second live authority path.
- Do not replace live Control Lane, gateway, permissions, lanes, audit, modelruntime, retrieval, embeddings, memory, or API routing in one phase.
- Do not make the live Context Compiler prompt authority without a separate design and tests.
- Do not enable live KV reuse.
- Do not make Qdrant live retrieval authority.
- Do not make Redis canonical state.
- Do not mutate live memory through simulator services.
- Do not add public routes as part of an authority cutover unless the phase explicitly approves them.

## Phase 14B Implemented Seam

Phase 14B selected deterministic ref-shape validation as the first operational seam.

It extracts `services/core/internal/refvalidation` as a shared pure package and invokes it from live Control Lane action `VALIDATE_REF_SHAPE`. This follows the Phase i1/I2 pattern and does not migrate live memory, retrieval, gateway, modelruntime, API routes, or FORGE-K Kernel authority.

The next operational phase must continue the same rule: choose one narrow contract, keep the existing live owner, prove no unauthorized mutation, and preserve rollback before any broader live authority migration.

## Phase 14F Implemented Seam

Phase 14F marks Control Lane validation as the first explicit FORGE-K partial live enforcement mode. The live owner remains `services/core/internal/aios/controllane`; shared pure validators provide deterministic doctrine, and summaries expose activation/no-effect metadata for operators and tests.

This remains partial enforcement, not full FORGE-K live authority. It does not import simulator services, replace the Control Lane, enable live KV reuse, admit evidence through the simulator Courthouse, compile live prompts through the simulator Context Compiler, call modelruntime, execute tools, run retrieval/search/embeddings, or write semantic memory directly.

## K20A Production Ingress Cutover

K20A adds `services/core/internal/forgekernel` as a production-owned authority
boundary. The default boot mode is `forge_k`; `legacy_v1` is an explicit tested
rollback. Only one processor is selected and only one durable commit occurs.

K20A does not import `services/core/internal/forgek`, replace SQLite, duplicate
journal writes, or claim full Kernel completion. It established the temporary
Control Lane adapter boundary that K20B subsequently split into FORGE-K-owned
durable port stages.

## K20B Durable Orchestration Cutover

K20B closes that extraction gate. `forgekernel.Kernel` now owns the live stage
order through `DurablePort`: deterministic prepare, exactly one atomic
apply+journal commit, audit persistence/linkage, then best-effort observation.
Control Lane implements those port methods and retains combined orchestration
only for the explicit `legacy_v1` rollback mode.
