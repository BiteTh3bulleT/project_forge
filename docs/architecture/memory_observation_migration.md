# Memory Observation Migration

Status date: 2026-05-18.

Status: `MEMORY_OBSERVATION_WRITES_RETIRED / HISTORY_PRESERVED / COURTHOUSE_REVIEW_GUIDANCE / CONTROL_LANE_CANONICAL_COMMIT / NO_FORGE_K_AUTHORITY_MIGRATION`.

## Intent

FORGE-K Online Phase 13 closes the legacy memory observation write surface without losing historical observation data. Existing observation rows remain retrieval/history evidence. New canonical memory must be reviewed and committed through governed semantic paths.

## Live Owner

The live retirement gate is `services/core/internal/api`:

- `POST /api/memory/observations`
- `PATCH /api/memory/observations/{id}`
- `POST /api/memory/observations/{id}/usefulness`

Those endpoints return `410 Gone` and write audit records. Canonical replacement writes remain owned by `services/core/internal/aios/controllane`.

## Target Owners

Target FORGE-K owners remain:

- Courthouse for future evidence admission semantics
- Kernel for future canonical commit authority

This phase does not import or invoke `services/core/internal/forgek/court` or `services/core/internal/forgek/kernel` as live daemon authority.

## Migration Rule

Legacy observation rows are evidence, not canonical truth. They may remain readable through observation and retrieval APIs for history, inspection, VSA signals, and packet alignment.

New canonical memory derived from an observation must follow the governed path:

1. Preserve the legacy observation row or source reference as historical evidence.
2. Validate observation-derived evidence shape through `VALIDATE_ADMISSION_CANDIDATE`.
3. Commit accepted canonical memory through existing Control Lane semantic syscalls such as `CREATE_NOTE`, `UPDATE_STATE`, `OPEN_LOOP`, or `CLOSE_LOOP`.

## Boundary

The retired endpoint does not auto-convert legacy observations into notes, state, or loops. The response and audit payload only provide migration guidance. There is no batch migrator, no evidence admission, no direct memory write, and no live FORGE-K authority migration in this phase.
