# Phase 14J Ref Shape Comparison Lane Closure Design

Status: Approved for implementation on 2026-05-11.

## Goal

Close the `COMPARE_REF_SHAPE` lane as a deterministic, diagnostic-only Control Lane surface.

This follows the Phase 14H and Phase 14I lane-closure pattern: the lane is connected through the live Control Lane owner, carries explicit no-effect metadata on success and rejection, and remains validation/comparison only. It does not make FORGE-K simulator services live authority.

## Scope

Phase 14J covers only:

- shared pure comparison package `services/core/internal/refvalidation`
- live Control Lane action `COMPARE_REF_SHAPE`
- status and architecture docs

Allowed:

- deterministic comparison-only hardening
- fail-closed validation for candidate and observed refs
- deterministic match/drift summaries
- rejected-path no-effect state/audit metadata
- tests proving rejected comparisons do not commit or persist idempotency state

Forbidden:

- object truth lookup
- evidence admission or rejection
- semantic memory writes
- context compilation
- retrieval, search, or embedding execution
- modelruntime calls or mutation
- gateway/tool execution
- public API or route changes
- importing FORGE-K simulator services into live code
- making FORGE-K Kernel, Courthouse, Context Compiler, Memory Palace, KV, Runtime, or Consensus services live authority

## Lane Closure Definition

For Phase 14J, the comparison lane is closed when:

1. `COMPARE_REF_SHAPE` remains routed through `services/core/internal/aios/controllane`.
2. `services/core/internal/refvalidation` remains pure deterministic comparison code with no simulator or live stateful imports.
3. Candidate and observed refs reuse the canonical ref-shape validator and fail closed when either side is invalid.
4. Matched and drift results remain accepted diagnostic results, not mutation requests.
5. Rejected live comparison carries `refShapeComparison`, `forgeKActivation`, and `forgeKNoEffect` in state and audit summaries.
6. Rejected live comparison does not commit semantic objects or persist idempotency state.
7. Successful comparison remains normalization/diff only and does not look up object truth.

## Authority Boundary

Closed means the diagnostic comparison lane is contractually bounded. It does not mean compared refs are true, admitted, searchable, retrievable, or safe to use as canonical memory. Comparison checks shape and set differences only. Any future source-object lookup, evidence admission, retrieval, or semantic write remains a separate authority-migration phase.

## Test Strategy

Add focused tests:

- shared comparison returns deterministic match results
- shared comparison returns deterministic drift results with stable added/removed/unchanged refs
- shared comparison fails closed for invalid candidate refs and invalid observed refs
- live Control Lane rejected invalid comparisons preserve no-effect state/audit metadata
- rejected live comparison does not persist idempotency state
- dry-run, match, and drift behavior remain unchanged

Then run:

- `cd services/core && go test ./internal/refvalidation -run 'CompareRefShapes' -count=1`
- `cd services/core && go test ./internal/aios/controllane -run 'CompareRefShape|ForgeKActivationContract' -count=1`
- `cd services/core && go test ./internal/aios/controllane ./internal/refvalidation -count=1`
- `cd services/core && go test ./...`

## Documentation

Update:

- `docs/architecture/control_lane_kernel.md`
- `docs/architecture/simulator_to_live_migration.md`
- `docs/reviews/current_phase_status.md`

The docs must record Phase 14J as `LANE_CLOSED / DIAGNOSTIC_ONLY / VALIDATION_ONLY / NO_AUTHORITY_EXPANSION` and explicitly state that ref-shape comparison does not prove object truth or perform evidence admission.
