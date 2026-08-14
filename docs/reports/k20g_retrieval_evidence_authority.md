# K20G Retrieval Evidence Authority Report

Status: `PARTIAL LIVE CUTOVER / PRODUCTION FORGE-K ADMISSION / NO SIMULATOR AUTHORITY`

Date: 2026-08-14

## Outcome

`RECORD_RETRIEVAL_EVIDENCE` is the sole production writer for governed retrieval runs, ordered results, selection reasons, and an optional already-known packet association. Retrieval computation remains in the live search/embedding services; its output becomes durable evidence only after the production Kernel validates authorization, exact source-root scope, provenance, prepared-plan seal, typed receipt, journal content, audit intent, and idempotency proof.

The commit is one SQLite transaction. A result/selection, journal, audit-outbox, or idempotency failure rolls the entire bundle back. Models, adapters, future IRIS, and the simulator have no commit authority.

## Closed gaps

- Removed direct retrieval run/result/selection writers and the late job packet-join writer.
- Removed dormant `memory.SaveSelectionReason` and scope-less job outcome calls.
- Required at least one resolved source and exact sorted canonical `SelectedPaths`.
- Required every result path to be contained by a sealed source root.
- Preserved API authenticated-user attribution and system/internal service-principal attribution.
- Bound retries to immutable request/job/scope/provenance evidence; user requests cannot bypass Kernel authorization through the retry shortcut.
- Disabled legacy manifest-less VSA influence and created no memory observations, observation links, or VSA signal rows.
- Preserved original dossier/packet/job/chunk/file ids separately from detachable live foreign keys, so normal source and job cleanup cannot erase or deadlock immutable evidence.
- Kept usefulness as a separate append-only utility event with a labeled noncanonical projection.

## Validation evidence

- `cd services/core && go test ./internal/retrieval ./internal/aios/controllane ./internal/store ./internal/jobs`
- `cd services/core && go test ./internal/api`
- `cd services/core && go test ./...`
- `cd services/core && go vet ./...`
- `git diff --check`

Tests cover payload/rank/id/scope/path tampering, unknown legacy signal fields, exact commit-plan ids, real production authorization, atomic rollback, retry-before-search, authorization-bypass prevention, sole-writer guards, zero direct VSA/memory duplication, noncanonical usefulness overlay, and parent-row lifecycle detachment with original identity/content preservation.

## Remaining blockers

- Search, FTS, embedding generation, and candidate ranking execute outside the Kernel before evidence admission.
- Control Lane still supplies the temporary deterministic apply and SQLite durable-port implementation.
- The existing backup mapping does not yet constitute daemon-stopped, chain-verified whole-store recovery for K20G evidence.
- Other retrieval-adjacent and subsystem writers must complete their own staged authority cutovers.
- This does not make `services/core/internal/forgek/palace` or any other simulator package live authority.
