# FORGE AI-OS Upgrade Phases

This roadmap extends current FORGE doctrine and modules. It does not create a parallel architecture.

## Phase status snapshot

- Phase 1: implemented
- Phase 2: partially implemented (deterministic syscall kernel landed in `services/core/internal/aios/controllane`)
- Phase 3: partially implemented (durable cognitive filesystem schema + SQLite transaction/repository integration landed)
- Phase 4: implemented (FORGE-only ingest orchestration + librarian runtime cells + syscall integration + tests/docs)
- Phase 5: implemented (truth engine services, lifecycle hardening, scope-safe resolution, ingest truth integration, tests/docs)
- Phase 6+: planned

---

## Phase 1 - AI-OS foundation and tri-lane scaffold (implemented)

Goal:

- canonize FORGE as AI-OS
- define Control/I-O/Compute lane boundaries
- define cognitive filesystem baseline and IRIS boundary

Expected modules:

- `docs/architecture/forge_ai_os.md`
- `docs/adr/0001-forge-is-ai-os.md`
- `docs/roadmap/forge_ai_os_phases.md`
- initial lane scaffolds under `services/core/internal/aios/*`
- shared primitives in `packages/shared/src/aios.ts` and `services/core/internal/aios/domain/*`

Non-goals:

- persistence rewrite
- full IRIS runtime

Validation criteria:

- architecture and ADR docs are present
- doctrine mapping is traceable to repo modules/tables
- IRIS boundary rule is explicit

## Phase 2 - deterministic kernel + semantic syscalls (partially implemented)

Goal:

- implement deterministic semantic syscall processor
- enforce capability/approval/validation before durable mutation
- return typed deterministic syscall results with audit evidence

Expected modules:

- `services/core/internal/aios/controllane/registry.go`
- `services/core/internal/aios/controllane/processor.go`
- `services/core/internal/aios/controllane/validator.go`
- `services/core/internal/aios/controllane/capabilities.go`
- `services/core/internal/aios/controllane/approval.go`
- `services/core/internal/aios/controllane/store.go`
- `services/core/internal/aios/controllane/audit.go`
- Phase 2 docs under `docs/architecture/*semantic_syscalls*`

Non-goals:

- full cognitive filesystem persistence backend
- live LLM/provider integration
- full ingest/context compiler behavior

Validation criteria:

- starter syscalls are registered and executable through processor
- unsupported/invalid/unauthorized actions fail deterministically
- dry-run performs validation without commit
- audit emits records for commit/reject/dry-run
- tests pass in `services/core/internal/aios/controllane`

## Phase 3 - cognitive filesystem persistence (partially implemented)

Goal:

- replace in-memory semantic store with durable repositories
- preserve current-vs-historical truth boundaries

Expected modules:

- persistence adapters implementing `SemanticStore`/`TransactionRunner`
- schema/repository support for notes, links, state, loops, models, contradiction/supersession records
- workspace-scoped query support for context compilation inputs
- persistence inventory and cognitive data model docs (`docs/data_model/*`)

Non-goals:

- full semantic extraction
- live IRIS execution

Validation criteria:

- transaction boundary remains deterministic with real storage
- no hard-delete paths for canonical evidence objects
- supersession/contradiction/history trails are queryable

Current status:

- landed in `services/core/internal/aios/controllane/sqlite_store.go`
- landed schema in `services/core/internal/store/migrate.go`
- landed integration tests for syscall->persistence flows in `internal/aios/controllane/sqlite_integration_test.go`
- backup extraction inventory includes cognitive tables for `full_backup`

## Phase 4 - FORGE-only ingest pipeline + internal librarian cells (implemented)

Goal:

- run ingest as FORGE-owned compute cells producing candidate actions

Expected modules:

- `services/core/internal/aios/compute/librarian/*` implementations
- ingestion bridge into syscall requests

Landed:

- typed ingest contracts (`IngestRequest`, `IngestResult`, candidate batches, per-cell diagnostics)
- deterministic ingest orchestrator (journal append/virtualize, cell context, ordered cell execution)
- active runtime cells:
  - `IntakeCell`
  - `CatalogCell`
  - `LinkerCell`
  - `ContradictionCell`
  - `StateCell`
  - `PatternCell`
  - `RecallCell`
  - `CleanupCell`
- optional semantic inference adapter seam with no-op default
- action ordering/dedupe/idempotency and per-cell provenance metadata
- pipeline modes:
  - `validate_only`
  - `commit_valid`
  - deterministic unsupported `commit_all_or_fail` (no fake atomicity)
- broad Phase 4 pipeline tests in `services/core/internal/aios/compute/librarian/pipeline_phase4_test.go`
- docs:
  - `docs/architecture/ingest_pipeline.md`
  - `docs/architecture/librarian_cells.md`

Non-goals:

- direct cell writes to canonical persistence

Validation criteria:

- cells output proposals only
- kernel remains sole commit authority

## Phase 5 - active state, open loops, contradiction/supersession (implemented)

Goal:

- formalize active state and loop lifecycle projections with reconciliation

Landed modules:

- truth contracts:
  - `services/core/internal/aios/domain/truth.go`
- truth services:
  - `services/core/internal/aios/truth/engine.go`
- control-lane hardening for lifecycle/scope:
  - scoped state lookup and repositories:
    - `services/core/internal/aios/controllane/store.go`
    - `services/core/internal/aios/controllane/sqlite_store.go`
    - `services/core/internal/aios/controllane/repository_interfaces.go`
    - `services/core/internal/aios/controllane/sqlite_repositories.go`
  - validator/apply transition hardening:
    - `services/core/internal/aios/controllane/validator.go`
    - `services/core/internal/aios/controllane/processor_apply.go`
- ingest integration:
  - `services/core/internal/aios/compute/librarian/pipeline.go`
  - `services/core/internal/aios/compute/librarian/runtime.go`
  - `services/core/internal/aios/compute/librarian/cells_phase4.go`
  - `services/core/internal/aios/domain/ingest.go`
- shared TS mirror updates:
  - `packages/shared/src/aios.ts`
- tests:
  - `services/core/internal/aios/truth/engine_test.go`
  - `services/core/internal/aios/compute/librarian/pipeline_phase4_test.go`

Non-goals:

- model-only truth overwrite

Validation highlights:

- deterministic transitions and archival/supersession semantics hold under replay
- scope-aware state and object resolution
- contradiction/supersession preservation without hard-delete
- stale loop detection as non-destructive warning/query path
- ingest pipeline truth diagnostics and duplicate-update suppression

## Phase 6 - context compiler

Goal:

- compile deterministic context packets from evidence and active state

Expected modules:

- context compiler service integrating notes/links/state/loops/models/artifacts/events

Non-goals:

- unconstrained freeform prompt assembly

Validation criteria:

- packet inclusion reasons are explicit and reproducible

## Phase 7 - runtime/event bus/workspaces

Goal:

- explicit workspace runtime boundaries and event bus semantics

Expected modules:

- runtime/workspace/event-bus integrations with syscall kernel

Non-goals:

- remote multi-tenant runtime redesign

Validation criteria:

- workspace boundary enforcement across kernel + I/O + compute

## Phase 8 - adaptive policy/algebraic model layer

Goal:

- formal derived policy/model algebra over evidence

Expected modules:

- adaptive policy services tied to derived models

Non-goals:

- replacing kernel rules with heuristics

Validation criteria:

- policy outputs explainable from evidence; reversible and versioned

## Phase 9 - shell/API/observability

Goal:

- expose stable control lane APIs and operator observability

Expected modules:

- API + desktop surfaces for syscall traceability and audit inspection

Non-goals:

- complete desktop redesign

Validation criteria:

- proposal-to-commit/reject trace visible end-to-end

## Phase 10 - evals and future IRIS integration seam

Goal:

- evaluate and integrate IRIS via FORGE-controlled syscall seam

Expected modules:

- eval harnesses
- IRIS bridge integration using `source=future_iris` syscall path

Non-goals:

- IRIS direct mutation of canonical state

Validation criteria:

- IRIS proposals are validated/authorized/audited by FORGE before commit
- no bypass path exists around Control Lane

## Remaining Phase 3 work after current Phase 2

- optional: expose cognitive filesystem query APIs via HTTP/desktop inspector surfaces
- optional: broaden backup restore import mappings for cognitive tables (currently extraction-focused)
- optional: automatic context snapshot writes for `COMPILE_CONTEXT` when policy enables it

## Remaining work for Phase 5 and Phase 6

Phase 6 focus:

- full deterministic context compiler and packet inclusion-reason expansion
- richer retrieval/context assembly over notes/links/state/loops/models/artifacts/events
- use Phase 5 current-object resolution and contradiction/supersession signals for packet ranking and warnings
