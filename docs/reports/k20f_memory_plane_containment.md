# K20F Memory-Plane Containment Report

Date: 2026-08-14  
Status: behavior-affecting legacy writers contained; broader evidence cutover pending

## Outcome

Default scheduled maintenance and manual sweeps no longer execute live memory
repair. Repair and VSA reindex endpoints require explicit `dryRun: true` and
return deterministic proposal/preview reports without creating repair runs,
reindex runs, pointers, bindings, associations, or stale-flag updates.

Retrieval continues to persist its run, ranked result, selection-reason, and
optional VSA signal evidence, but it no longer creates or updates legacy
`memory_observations` or `retrieval_result_observations`. A usefulness event
updates a linked legacy observation's VSA reliability projection exactly once;
the prior duplicate update is removed.

Static callsite guards reject new production calls to mutating repair, reindex,
or retrieval-observation linking outside a future governed adapter.

## Authority posture

These tables are non-canonical evidence or rebuildable acceleration, not
canonical truth. Containment prevents autonomous or incidental rewrites while
later stages replace them with atomic append-only evidence contracts and
identity-bound projection rebuilds. Canonical notes, state, loops, semantic
links, Court records, snapshots, and their journal/provenance path remain under
the production Kernel transaction.

## Validation

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/forgekernel/... ./internal/aios/controllane ./internal/store ./internal/api ./internal/memory ./internal/retrieval ./internal/backup`
- Focused tests prove omitted/false maintenance dry-run fails closed, explicit
  preview writes zero rows, retrieval leaves legacy observation counts
  unchanged, and one usefulness event changes reliability once.
- `git diff --check`

## Remaining blockers

- Retrieval evidence is not yet one atomic `RECORD_RETRIEVAL_EVIDENCE` batch.
- Usefulness and restore feedback need append-only utility-event syscalls rather
  than mutable projections; restore feedback is disabled meanwhile.
- VSA rebuild needs an identity-bound staging-and-atomic-swap contract.
- Legacy write-capable repository constructors remain to be narrowed or
  privatized after their consumers migrate.
