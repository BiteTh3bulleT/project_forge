# K20G — Governed VSA Acceleration Cutover

Status: **PARTIAL LIVE AUTHORITY / PROJECTION-ONLY** (2026-08-14)

K20G moves VSA acceleration replacement behind the production FORGE-K semantic syscall `REBUILD_MEMORY_ACCELERATION`. It does not make VSA canonical memory and it does not complete the broader memory-plane authority cutover.

## Live contract

- The action is registered as a mutating production syscall with capability `memory.acceleration.rebuild`.
- Every rebuild is bound to an exact non-empty `workspaceId` and `laneId`, the versioned deterministic algorithm identity, dimensions, seed, expected manifest hash, expected prior head, and the sealed syscall `requestedAt` timestamp.
- The manifest hashes the normalized scoped source set, scoped link set, and algorithm identity. Model output is not an input.
- SQLite builds all pointer, binding, and association rows in transaction-local staging tables. The caller's FORGE-K apply+journal transaction then swaps only the exact scope and advances its singleton head with compare-and-swap semantics.
- A stage failure, manifest mismatch, head mismatch, or journal transaction failure rolls back the manifest, projection rows, and head together. An already-active manifest is rejected rather than creating a self-referential head.
- An exact scope with zero governed sources fails closed. Clearing a projection requires a separate future explicit action.
- Manifest history is immutable. Projection rows and heads are derived acceleration state, never canonical memory truth.

## Legacy exclusion and runtime reads

Existing observation, link, and VSA rows migrate with empty `workspace_id`, `lane_id`, and/or `manifest_hash`. Migration deliberately does not invent authority for them. Governed rebuilds and reads exclude those rows.

Runtime VSA scoring requires an explicit non-empty workspace/lane request and a matching active scoped manifest head. It uses the active manifest's dimensions and seed and only rows carrying that manifest hash. Existing retrieval callers do not yet propagate this exact scope, so they receive no VSA score or ranking influence. There is no global active-head fallback.

The legacy memory package no longer writes VSA projection tables or usefulness events/counters. Observation/link auto-reindex, observation-local reindex, direct reliability updates, reindex-run mutation, and direct usefulness/link-result writers fail closed. A syntax-level test rejects reintroduction of projection/usefulness mutation SQL under `internal/memory`; the governed SQLite adapter is the sole projection writer.

## Operator API

`POST /api/memory/vsa/reindex/run` has two explicit modes:

- `dryRun: true` preserves read-only legacy candidate inspection. When both `workspaceId` and `laneId` are supplied, it also returns the deterministic governed manifest and current expected prior head. A scoped proposal with no governed sources fails closed.
- `dryRun: false` requires exact scope, idempotency key, expected manifest hash, and an explicitly supplied expected prior manifest hash (empty is valid for the first head). The server recomputes both identities before submitting `REBUILD_MEMORY_ACCELERATION` through the selected production FORGE-K processor.
- An omitted `dryRun` is rejected. Direct legacy reindex is not a fallback.

## K20H source cutover and remaining blocker

K20H supplies the missing source contract through immutable,
Courthouse-admitted `forge_k_memory_evidence` and dedicated evidence-FK VSA
projection tables. The v2 planner and runtime exclude legacy observations and
superseded or no-longer-current Court evidence. Direct observation, link,
repair, usefulness, and legacy projection writer bodies now fail closed, with
static callsite and SQL guards preventing their reconnection.

Retrieval callers must still propagate an exact nonempty workspace/lane scope.
Callers without it receive zero VSA influence. Governed semantic relationship
edges and usefulness aggregation for immutable memory evidence also remain
future contracts; legacy links and counters are never imported into the v2
manifest.

## Verification expectations

Focused coverage includes deterministic reorder identity, algorithm/source/link divergence, duplicate and cross-scope rejection, legacy migration exclusion, immutable manifests, sealed timestamps, zero-source rejection, transaction requirement, CAS head conflicts, injected swap rollback, scoped active-head reads, unscoped/mismatched-scope zero influence, API proposal-to-syscall routing, legacy writer fail-closed behavior, and the static SQL authority guard.
