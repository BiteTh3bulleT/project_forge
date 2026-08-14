# Memory Observation Migration

Status date: 2026-08-14.

Status: `K20H / LEGACY_MEMORY_WRITERS_SEALED / HISTORY_PRESERVED / REPAIR_PROPOSAL_ONLY / PRODUCTION_FORGE_K_MATERIALIZATION`.

## Intent

K20H closes the remaining legacy observation, link, usefulness, and repair
write surfaces without deleting historical data. Existing rows remain readable
evidence. New admitted evidence and later revisions use deterministic,
authenticated production FORGE-K semantic syscalls.

## Live Owner

The live retirement gate is `services/core/internal/api`:

- `POST /api/memory/observations`
- `PATCH /api/memory/observations/{id}`
- `POST /api/memory/observations/{id}/usefulness`

Those endpoints are terminal handlers: they return `410 Gone`, write denied
audit records, do not decode request bodies, and cannot reach a writer. No
legacy observation-link mutation route is mounted. Observation and repair
history reads remain available.

## Production Owner

Production authority is `services/core/internal/forgekernel`:

- `MATERIALIZE_ADMITTED_EVIDENCE` commits admitted evidence.
- `REVISE_MEMORY_EVIDENCE` appends a superseding revision while preserving the
  original evidence.

The Control Lane SQLite implementation is a temporary durable adapter, not the
orchestration owner. The simulator under `services/core/internal/forgek` and
model/runtime proposals have no commit authority.

## Migration Rule

Legacy observation rows are evidence, not canonical truth. They may remain readable through observation and retrieval APIs for history, inspection, VSA signals, and packet alignment.

New governed memory evidence derived from an observation follows the governed path:

1. Preserve the legacy observation row or source reference as historical evidence.
2. Admit evidence through deterministic Courthouse policy and exact scoped provenance.
3. Commit accepted evidence through `MATERIALIZE_ADMITTED_EVIDENCE`.
4. Preserve the original and append later changes through `REVISE_MEMORY_EVIDENCE`.

Legacy exported service methods remain source-compatible but return
`ErrMemoryEvidenceAuthorityRequired`. Production code contains no callsite for
them and the legacy memory package contains no observation/link/usefulness or
repair SQL writer.

Repair remains proposal-only. `PreviewRepairPass` and the background ticker
select deterministic candidates and describe possible effects without
creating repair runs, rewriting evidence, or rebuilding VSA projections.
Non-dry API requests and direct `RunRepairPass` calls fail closed.

## Boundary

The retired routes do not auto-convert legacy observations, mutate usefulness,
create links, or execute repair. K20H adds no model-driven endpoint and no
batch migrator. Any future public evidence endpoint must authenticate and enter
the production Kernel; it cannot reconnect a legacy service writer.
