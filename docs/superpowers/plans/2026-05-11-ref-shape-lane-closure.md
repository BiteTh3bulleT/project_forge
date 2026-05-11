# Phase 14I Ref Shape Lane Closure Implementation Plan

Status: Approved for implementation on 2026-05-11.

Design: `docs/superpowers/specs/2026-05-11-ref-shape-lane-closure-design.md`

## Objective

Close `VALIDATE_REF_SHAPE` as a deterministic, validation-only Control Lane surface. The lane must expose canonical ref-shape gates, fail closed with explicit no-effect metadata, and remain free of object truth lookup, evidence admission, semantic memory writes, context compilation, retrieval/search/embedding execution, modelruntime calls, gateway/tool execution, public route changes, and FORGE-K simulator authority.

## Task 1: Shared Ref Validator Contract

Files:

- `services/core/internal/refvalidation/models.go`
- `services/core/internal/refvalidation/models_test.go`

Work:

- Replace the inline allowed-type switch with a canonical allowed ref type list plus lookup set.
- Expose `AllowedRefTypes()` as a copied list for tests and operator/status consumers.
- Add tests proving all allowed types pass validation.
- Add fail-closed tests for empty refs, missing ref id, unknown ref type, unsafe ref id, and workspace mismatch.
- Preserve deterministic normalization, deduplication, and sorting.

Verify:

- `cd services/core && go test ./internal/refvalidation -run 'AllowedRefTypes|ValidateRefs' -count=1`

Commit:

- `test: close ref shape validation contract`

## Task 2: Live Control Lane Rejected-Path Contract

Files:

- `services/core/internal/aios/controllane/processor.go`
- `services/core/internal/aios/controllane/ref_validation_test.go`

Work:

- Enforce `VALIDATE_REF_SHAPE` before commit, mirroring the closed semantic-operation lane pattern.
- On rejected ref-shape validation, return `refShapeValidation`, `forgeKActivation`, and `forgeKNoEffect` in state summary and audit summary.
- Prove rejected invalid refs do not commit semantic objects and do not persist idempotency state.
- Preserve dry-run and successful validation behavior.

Verify:

- `cd services/core && go test ./internal/aios/controllane -run 'RefShape|ForgeKActivationContract' -count=1`

Commit:

- `test: close ref shape control lane`

## Task 3: Status Documentation

Files:

- `docs/architecture/control_lane_kernel.md`
- `docs/architecture/simulator_to_live_migration.md`
- `docs/reviews/current_phase_status.md`

Work:

- Record Phase 14I as `LANE_CLOSED / VALIDATION_ONLY / NO_AUTHORITY_EXPANSION`.
- State that ref-shape closure validates only shape/safety/scope and does not prove object truth.
- State no evidence admission, context compilation, retrieval/search/embeddings, semantic memory writes, modelruntime calls, gateway/tool execution, public route changes, or FORGE-K simulator authority were introduced.

Commit:

- `docs: record ref shape lane closure`

## Task 4: Final Verification

Run:

- `cd services/core && go test ./internal/refvalidation -count=1`
- `cd services/core && go test ./internal/aios/controllane -run 'RefShape|ForgeKActivationContract' -count=1`
- `cd services/core && go test ./internal/aios/controllane ./internal/refvalidation -count=1`
- `cd services/core && go test ./...`
- `git status --short`

Report:

- Commits made
- Verification results
- Any unrelated dirty files still present
