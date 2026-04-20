# FORGE AI-OS Upgrade Phases

This roadmap extends current FORGE doctrine and modules. It does not create a parallel architecture.

## Phase status snapshot

- Phase 1: implemented
- Phase 2: partially implemented (deterministic syscall kernel landed in `services/core/internal/aios/controllane`)
- Phase 3: partially implemented (durable cognitive filesystem schema + SQLite transaction/repository integration landed)
- Phase 4: implemented (FORGE-only ingest orchestration + librarian runtime cells + syscall integration + tests/docs)
- Phase 5: implemented (truth engine services, lifecycle hardening, scope-safe resolution, ingest truth integration, tests/docs)
- Phase 5.5: partially implemented (deterministic rule-agent runtime integrated through autonomy scaffolding)
- Phase 5.75: implemented (autonomy charters + intent queue + freedom budgets + policy evaluator + self-initiated syscall runner + bounded ingest hook)
- Phase 5.9: implemented (AI-OS tool capability taxonomy + registry + gateway policy integration + capability API/UI visibility)
- Discord integration layer: implemented (Discord gateway transport + canonical event normalization + intent routing + permission/audit scaffolding)
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

## Phase 5.9 - AI-OS tool surface and capability registry (implemented)

Goal:

- formalize tools as kernel-governed capabilities with typed policy metadata

Landed modules:

- tool surface domain contracts:
  - `services/core/internal/aios/domain/tool_surface.go`
- capability registry + policy evaluator:
  - `services/core/internal/gateway/tool_capability_registry.go`
  - `services/core/internal/gateway/tool_policy.go`
- gateway integration:
  - `services/core/internal/gateway/service.go`
  - `services/core/internal/api/phase5.go`
  - `services/core/internal/api/server.go`
- UI/API surface:
  - `apps/desktop/src/lib/api.ts`
  - `apps/desktop/src/pages/ToolGatewayPage.tsx`
- tests:
  - `services/core/internal/gateway/tool_surface_test.go`

Non-goals:

- enabling all dangerous primitives as active operations
- bypassing existing permission/approval/audit boundaries

Validation highlights:

- full taxonomy registered as capability metadata (active + approval_only + stubbed coverage)
- capability policy gates execution before adapter invocation
- self-initiated context hooks support intent/charter/budget approval gating semantics
- capability metadata is inspectable via API/UI

## Phase 5.75 - autonomy layer (implemented)

Goal:

- give FORGE bounded initiative without bypassing kernel sovereignty

Landed modules:

- autonomy domain contracts:
  - `services/core/internal/aios/domain/autonomy.go`
- autonomy services/runtime:
  - `services/core/internal/aios/autonomy/repositories.go`
  - `services/core/internal/aios/autonomy/risk.go`
  - `services/core/internal/aios/autonomy/charter.go`
  - `services/core/internal/aios/autonomy/intent_queue.go`
  - `services/core/internal/aios/autonomy/budget.go`
  - `services/core/internal/aios/autonomy/policy_evaluator.go`
  - `services/core/internal/aios/autonomy/runner.go`
  - `services/core/internal/aios/autonomy/rule_agents.go`
  - `services/core/internal/aios/autonomy/ingest_integration.go`
  - `services/core/internal/aios/autonomy/curiosity.go`
  - `services/core/internal/aios/autonomy/defaults.go`
  - `services/core/internal/aios/autonomy/explain.go`
- ingest contract + integration:
  - `services/core/internal/aios/domain/ingest.go`
  - `services/core/internal/aios/compute/librarian/pipeline.go`
- tests:
  - `services/core/internal/aios/autonomy/autonomy_test.go`

Non-goals:

- unrestricted autonomy mode
- direct state mutation outside syscalls
- live LLM dependency

Validation highlights:

- charter/budget/risk gating before autonomous commit
- approval escalation for high-risk/blocked categories
- runner commits only through control-lane syscall processor
- intent and decision traces are inspectable
- ingest-triggered autonomy pass is depth-capped

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

## Discord integration layer (implemented)

Goal:

- treat Discord as FORGE external operator I/O, not reasoning core

Landed modules:

- `services/core/internal/api/discord_gateway_types.go`
- `services/core/internal/api/discord_gateway_translate.go`
- `services/core/internal/api/discord_gateway_router.go`
- `services/core/internal/api/discord_gateway_permissions.go`
- `services/core/internal/api/discord_gateway_service.go`
- `services/core/internal/api/discord_gateway_server.go`
- `docs/architecture/discord_integration_layer.md`

Validation highlights:

- slash/text ingress normalizes into canonical Discord event envelopes
- routed intents reuse existing FORGE services (chat/search/dashboard/adapters)
- outbound replies are structured and bounded by response contracts
- permission-denial, gateway errors, and accepted intents are auditable and correlated
- runtime status is inspectable via `/api/discord/status`

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
- integrate autonomy decision signals (intent/charter/budget history) into context ranking and operator explainability views
