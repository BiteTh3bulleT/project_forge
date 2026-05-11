# Phase 14I Ref Shape Lane Closure Design

Status: Approved for implementation on 2026-05-11.

## Goal

Close the `VALIDATE_REF_SHAPE` lane as a deterministic, validation-only Control Lane surface.

This follows Phase 14H's lane-closure pattern: connected through the live owner, no-effect on success and rejection, and regression-tested before moving to another lane. It does not make FORGE-K simulator services live authority.

## Scope

Phase 14I covers only:

- shared pure package `services/core/internal/refvalidation`
- live Control Lane action `VALIDATE_REF_SHAPE`
- status and architecture docs

Allowed:

- deterministic validation-only hardening
- canonical allowed ref-type exposure for tests/operators
- rejected-path no-effect state/audit metadata
- tests proving invalid refs do not commit or persist idempotency state

Forbidden:

- object lookup or truth lookup
- evidence admission or rejection
- semantic memory writes
- context compilation
- retrieval, search, or embedding execution
- modelruntime calls or mutation
- gateway/tool execution
- public API or route changes
- importing FORGE-K simulator services into live code
- making FORGE-K Kernel, Courthouse, Context Compiler, Memory Palace, KV, or Runtime services live authority

## Lane Closure Definition

For Phase 14I, the ref shape lane is closed when:

1. `VALIDATE_REF_SHAPE` remains routed through `services/core/internal/aios/controllane`.
2. `services/core/internal/refvalidation` remains a pure deterministic validator with no simulator or live stateful imports.
3. Allowed ref types are canonical and exposed through a copied list.
4. Invalid ref type, unsafe ref id, missing ref id, scope mismatch, and empty ref list cases fail closed.
5. Rejected live validation carries `refShapeValidation`, `forgeKActivation`, and `forgeKNoEffect` in state and audit summaries.
6. Rejected live validation does not commit semantic objects or persist idempotency state.
7. Successful validation remains normalization/deduplication only and does not look up object truth.

## Authority Boundary

Closed means the validation lane is contractually bounded. It does not mean a ref is true, admitted, searchable, retrievable, or safe to use as canonical memory. Ref validation checks shape and safety only. Any future object lookup, evidence admission, retrieval, or semantic write remains a separate authority-migration phase.

## Test Strategy

Add focused tests:

- shared validator exposes and enforces canonical allowed ref types
- shared validator fails closed for missing ids, workspace mismatches, unsafe ids, unknown types, and empty refs
- live Control Lane rejected invalid refs preserve no-effect state/audit metadata
- rejected live validation does not persist idempotency state
- dry-run and success behavior remain unchanged

Then run:

- `cd services/core && go test ./internal/refvalidation -count=1`
- `cd services/core && go test ./internal/aios/controllane -run 'RefShape|ForgeKActivationContract' -count=1`
- `cd services/core && go test ./internal/aios/controllane ./internal/refvalidation -count=1`
- `cd services/core && go test ./...`

## Documentation

Update:

- `docs/architecture/control_lane_kernel.md`
- `docs/architecture/simulator_to_live_migration.md`
- `docs/reviews/current_phase_status.md`

The docs must record Phase 14I as `LANE_CLOSED / VALIDATION_ONLY / NO_AUTHORITY_EXPANSION` and explicitly state that ref shape closure does not perform object truth lookup or evidence admission.
