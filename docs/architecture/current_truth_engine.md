# Current Truth Engine (Phase 5)

Phase 5 hardens FORGE truth maintenance as a deterministic projection layer over validated cognitive filesystem records.

Core rule:

**Cells propose. Kernel validates. Truth engine projects current state. FORGE preserves history.**

## What "current truth" means in FORGE

- Current truth is an explicit projection, not an overwrite of history.
- Historical truth remains inspectable in append-only and versioned records.
- Contradiction keeps both sides.
- Supersession preserves old and new records while resolving current successor.

Current truth answers:

- what is true now
- what changed
- what used to be true
- what is unresolved
- what is contradicted
- what superseded what

## Implementation modules

- Truth contracts: `services/core/internal/aios/domain/truth.go`
- Truth services: `services/core/internal/aios/truth/engine.go`
- Kernel enforcement + lifecycle validation:
  - `services/core/internal/aios/controllane/validator.go`
  - `services/core/internal/aios/controllane/processor_apply.go`
  - `services/core/internal/aios/controllane/transitions.go`

## Truth engine services

`truth.Engine` implements:

- State projection service:
  - `GetCurrentState`
  - `ListCurrentState`
  - `GetStateTimeline`
  - `ExplainState`
- Open-loop lifecycle service:
  - `OpenLoop`, `TransitionLoop`, `ResolveLoop`, `BlockLoop`, `ReopenLoop`, `ArchiveLoop`
  - `ListActiveLoops`, `ListBlockedLoops`, `ListLoopsByPriority`, `ListLoopsByOwner`, `ListStaleLoops`
  - `ExplainLoop`
- Contradiction service:
  - `RegisterContradiction`
  - `ListContradictionsForObject`
  - `ListContradictionsByScope`
  - `ExplainContradiction`
- Supersession service:
  - `MarkSuperseded`
  - `GetSuccessor`
  - `GetSupersessionChain`
  - `IsCurrentObject`
  - `ExplainSupersession`
- Current-object resolver:
  - `Resolve`
  - `ResolveMany`
  - `FilterCurrent`
  - `ExplainResolution`
- Query/explanation surface:
  - `GetCurrentTruth`
  - `GetTruthTimeline`
  - `GetTruthEvidence`
  - `ExplainCurrentTruth`
- Projection rebuild/report:
  - `RebuildProjection`

All mutating truth operations submit semantic syscalls through the Phase 2 kernel; the truth engine does not bypass commit controls.

## Active state and timelines

- `UPDATE_STATE` writes current projection (`state_items`) and appends history (`state_versions`).
- Scope key lookup is workspace/lane aware (`FindStateByScopeKey`).
- Current value queries and timeline reconstruction are deterministic.
- State explanations include:
  - current value
  - previous values
  - derived-from evidence references
  - contradiction/supersession references when available

## Current object resolution

Resolver behavior:

- archived objects are non-current
- superseded objects resolve to successor and are non-current for active retrieval
- contradicted objects may remain current but carry warnings
- deprecated models are non-current
- resolved/archived loops are non-current
- scope isolation is enforced for object resolution/explanation

## Rebuild / repair behavior

`RebuildProjection(scope, dryRun)` currently provides deterministic detection and reporting for:

- missing state current rows where history exists
- state projection value mismatches against latest history
- orphan contradiction references

Dry-run performs no mutations. Non-dry rebuild currently reports and marks `applied=true` only when no differences remain; it is intentionally conservative.

## Relationship to cells and ingest

Phase 4 ingest pipeline now wires a truth engine into cell runtime context and emits truth apply diagnostics in ingest results.

- `StateCell` suppresses duplicate state updates when current value already matches
- `ContradictionCell` avoids duplicate contradiction/supersession proposals when already known
- `PatternCell` suppresses proposals from contradicted or inactive evidence
- `RecallCell` adds contradiction warnings in hints
- `CleanupCell` uses truth stale-loop queries when available

## Relationship to cognitive filesystem

Truth engine projections are built from validated durable cognitive filesystem records; it does not replace persistence.

- Current projections: `state_items`, active statuses
- Historical evidence: `state_versions`, `journal_events`, contradiction/supersession records

## Relationship to context compiler (Phase 6)

Phase 5 exposes deterministic current-truth and explanation APIs that Phase 6 context compilation can consume for packet assembly and ranking.

## Relationship to future IRIS

Future IRIS may propose truth-changing actions (`source=future_iris`) through semantic syscalls.

IRIS cannot bypass:

- kernel validation/capability/approval
- lifecycle transition rules
- scope isolation
- audit/provenance traceability
