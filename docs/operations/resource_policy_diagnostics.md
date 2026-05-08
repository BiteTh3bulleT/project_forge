# FORGE-H Resource Policy Diagnostics

Phase N4 adds an internal advisory resource policy package at `services/core/internal/forgeh`.
Phase N5 adds advisory resource action proposals in the same package.
Phase N6 adds a bounded executor for approved resource action proposals.

It consumes a Host Kernel Bridge diagnostic snapshot and returns a `ResourcePolicySnapshot`. Phase N5 can convert that policy snapshot into `ResourceActionProposal` records. Phase N6 can produce `ResourceActionExecution` records through explicit adapters. It does not collect host diagnostics directly and does not mutate host or runtime state.

## What It Evaluates

- RAM pressure
- swap pressure
- disk pressure for the FORGE storage root
- VRAM pressure when GPU diagnostics are available
- thermal pressure when sensor diagnostics are available
- overall machine posture
- workload lane decisions
- model-load recommendation
- background-work recommendation
- bounded warnings and operator actions
- reviewable resource action proposals for background ingest, embedding, model-load posture, degraded-mode warning, and operator warning paths
- bounded execution records for approved proposals

## How To Test

```bash
cd services/core && go test ./internal/forgeh -count=1
```

Run the broader validation before changing policy behavior:

```bash
cd services/core && go test ./internal/hostbridge ./internal/forgeh -count=1
npm run test:core
npm run build:core
npm run lint
```

## Governed Boundary

`ResourcePolicySnapshot.advisory_only` is always true in Phase N4. `ResourceActionProposal.advisory_only` is always true in Phase N5.

Proposal records may move from `proposed` to `approved`, `rejected`, `expired`, or `superseded`. Approval is necessary but not sufficient. Phase N6 may only record bounded internal operational preferences through explicit adapters.

N6 execution records must keep:

- `bounded=true`
- `host_mutation=false`
- `semantic_memory_write=false`
- `modelruntime_mutation=false`

Execution is idempotent by proposal ID. Re-running the same approved proposal returns the existing execution record.

FORGE-H does not:

- pause workers
- delete files
- run cleanup
- restart services
- run package managers
- load or unload kernel modules
- load or unload models
- call modelruntime
- write semantic memory
- add public routes

Operator actions are recommendations written in bounded, non-destructive language.
Resource action proposals are governance records written in bounded, non-destructive language.
Resource action executions are bounded internal operational records, not host commands.
