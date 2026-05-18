# FORGE-K Online Phase 06 Courthouse Admission Status

## Phase

FORGE-K Online Phase 06 - Courthouse Admission.

## Status marker

`ADMISSION_CANDIDATE_ONLY / CONTROL_LANE_OWNED / NO_EVIDENCE_ADMISSION / NO_CANONICAL_TRUTH_COMMIT`

## Summary

The live Control Lane now exposes `VALIDATE_ADMISSION_CANDIDATE`, a validation-only syscall that applies the pure `admissionvalidation` contract to Courthouse-shaped admission candidates.

## Live owner

`services/core/internal/aios/controllane` remains the live owner for this surface.

## Target FORGE-K owner

FORGE-K Courthouse (`services/core/internal/forgek/court`) remains the target owner for future evidence admission and ruling semantics. This phase does not import or invoke simulator Courthouse services.

## Authority impact

No canonical truth commit. No evidence admission. No evidence rejection. No ruling authority. No memory mutation. No modelruntime call. No gateway/tool execution. No retrieval/search/embedding execution. No context compilation. No route/API behavior change. No live authority migration.

## Tests/evidence

Validation commands are recorded in `docs/reports/phase_06_courthouse_admission.md`.

## Rollback

Revert the Phase 06 commit to remove `VALIDATE_ADMISSION_CANDIDATE`, its Control Lane wiring, tests, and docs. No live data or host state rollback is required.

## Blockers

- Live evidence admission and ruling authority remain disabled.
- Memory Palace evidence mirror work remains Phase 07.
- Context Compiler shadow/canary/live work remains Phase 08.
- Full governed semantic mutation routing remains a later phase and requires separate proof.

## Next phase

Run Phase 07 as a separate bounded commit. Do not combine Memory Palace mirror work into this phase.
