# FORGE-H Resource Policy Diagnostics

Phase N4 adds an internal advisory resource policy package at `services/core/internal/forgeh`.

It consumes a Host Kernel Bridge diagnostic snapshot and returns a `ResourcePolicySnapshot`. It does not collect host diagnostics directly and does not mutate host or runtime state.

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

## Advisory Only

`ResourcePolicySnapshot.advisory_only` is always true in Phase N4.

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
