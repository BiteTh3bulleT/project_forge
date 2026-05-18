# FORGE-K Online Phase 13 Memory Observation Migration Status

## Phase

FORGE-K Online Phase 13 - Memory Observation Migration.

## Status marker

`MEMORY_OBSERVATION_WRITES_RETIRED / HISTORY_PRESERVED / COURTHOUSE_REVIEW_GUIDANCE / CONTROL_LANE_CANONICAL_COMMIT / NO_FORGE_K_AUTHORITY_MIGRATION`

## Summary

Legacy memory observation mutation endpoints remain retired. This phase makes the retirement path explicit: attempted writes receive structured migration guidance and an audit record that points to Courthouse admission-candidate review and Control Lane semantic syscalls for canonical memory commits. Existing `memory_observations` rows remain readable historical evidence and are not deleted or rewritten.

## Live owner

The live retirement gate is `services/core/internal/api`. Canonical replacement writes remain owned by `services/core/internal/aios/controllane`. Legacy observation storage remains under `services/core/internal/memory` and SQLite as historical evidence/retrieval data.

## Target FORGE-K owner

FORGE-K Courthouse and Kernel remain target architecture owners for future admission and commit authority. This phase does not import or invoke simulator Courthouse or Kernel services as live authority.

## Authority impact

No new canonical authority migration. The write surface for `POST/PATCH /api/memory/observations*` stays retired and audited. The migration path is guidance and audit metadata only:

- preserve existing `memory_observations` history
- validate new observation-derived evidence through `VALIDATE_ADMISSION_CANDIDATE`
- commit accepted canonical memory only through Control Lane semantic syscalls such as `CREATE_NOTE`, `UPDATE_STATE`, `OPEN_LOOP`, and `CLOSE_LOOP`

## Tests/evidence

Validation commands are recorded in `docs/reports/phase_13_memory_observation_migration.md`.

## Rollback

Revert the Phase 13 commit to restore the previous plain-text retirement response/audit payload. The retired write-surface behavior remains unchanged by this phase.

## Blockers

- FORGE-K Courthouse and Kernel simulator services are not live authority.
- There is no live evidence admission path yet; `VALIDATE_ADMISSION_CANDIDATE` remains validation-only.
- No automatic batch migration converts legacy observation rows into canonical notes/state/loops in this phase.
- Retrieval/VSA observation tables remain historical/retrieval evidence, not canonical truth.

## Next phase

Run the next phase as a separate bounded commit.
