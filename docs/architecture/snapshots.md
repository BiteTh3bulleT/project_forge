# Context-Shape Snapshots

Status: Phase 6 implemented in the FORGE-K simulator only.

Snapshots preserve semantic shape, not canonical truth. A snapshot records restorable structure through source references, hashes, operation records, summaries, and provenance links. It may seed future restoration or review, but it does not admit evidence, create canonical truth, compile context, execute restoration, or authorize runtime behavior.

ADR 0005 defines the authority boundary: the Phase 6 implementation lives under `services/core/internal/forgek` and is not wired into the live daemon. Live AI-OS snapshot/restore behavior remains separate.

## Implemented Scope

Phase 6 adds an in-memory simulator package:

- `services/core/internal/forgek/snapshots`: snapshot models, lifecycle status, service, diff, restore seed, validation, and deterministic hashing.
- `services/core/internal/forgek/snapshot_syscalls.go`: simulator syscalls for snapshot create/read/list/seal/supersede/expire/diff/restore-seed creation.
- `services/core/internal/forgek/snapshot_syscalls_test.go` and `services/core/internal/forgek/snapshots/*_test.go`: shape-not-truth, provenance, capability, journal, diff, restore-seed, and by-reference integration tests.

No API routes, gateway paths, model runtime paths, live memory mutation paths, server wiring, Context Compiler, token hashing, or KV cache implementation are part of Phase 6.

## Snapshot Types

The model supports these snapshot types:

- `SEMANTIC_SNAPSHOT`
- `CASE_SNAPSHOT`
- `CONTEXT_RESTORE_SNAPSHOT`
- `PALACE_ROUTE_SNAPSHOT`
- `WORKSPACE_SNAPSHOT`
- `DECISION_SNAPSHOT`
- `KV_SHAPE_SNAPSHOT`
- `RUNTIME_SNAPSHOT`

Phase 6 implements shared model behavior for all types and focused tests for semantic, case, context-restore, palace-route, and KV-shape snapshots.

## Snapshot Model

A snapshot stores identifiers and references, not full canonical content. Important fields include:

- `snapshot_id`, `snapshot_type`, `workspace_id`, optional `case_id`
- `status`
- source, palace route, submitted, admitted, rejected, semantic operation, contradiction, supersession, derived, context block, token hash, and KV manifest refs
- `summary`
- `shape_hash` and `source_hash`
- creator and lifecycle timestamps
- supersession links
- journal refs
- metadata

The model rejects missing workspaces, invalid snapshot types, duplicate refs in normalized output, large raw content metadata keys, and empty reference sets except for explicitly allowed workspace snapshots.

## Lifecycle

Snapshot statuses are:

- `DRAFT`
- `SEALED`
- `SUPERSEDED`
- `EXPIRED`
- `RESTORE_SEED_CREATED`

New snapshots may be created as `DRAFT` or `SEALED`. Sealed snapshots are immutable through public service APIs. Superseded and expired snapshots remain inspectable. Restore-seed creation records a proposal artifact for future context restoration and does not mutate canonical truth.

Canonical simulator lifecycle changes happen through snapshot syscalls. The service owns deterministic in-memory storage, but mutating canonical operations are exposed through Kernel dispatch.

## Syscalls

Phase 6 registers these simulator syscalls:

- `snapshot.create`
- `snapshot.get`
- `snapshot.list`
- `snapshot.seal`
- `snapshot.supersede`
- `snapshot.expire`
- `snapshot.diff`
- `snapshot.restore_seed`

Mutating syscalls require matching snapshot capabilities and workspace scope. Read-only syscalls return snapshots or diffs without mutation capability. Mutating syscalls journal lifecycle events:

- `SNAPSHOT_CREATED`
- `SNAPSHOT_SEALED`
- `SNAPSHOT_SUPERSEDED`
- `SNAPSHOT_EXPIRED`
- `SNAPSHOT_RESTORE_SEED_CREATED`

`snapshot.diff` is read-only in Phase 6 and does not mutate or journal.

## SnapshotDiff

`SnapshotDiff` compares two snapshots by shape, reference sets, selected fields, status, and metadata. It returns deterministic added, removed, and unchanged refs plus changed field names. Diff creation does not mutate either snapshot.

## RestoreSeed

`RestoreSeed` cites the source snapshot id and source shape hash. It carries recommended source refs, operation refs, case refs, summary, and metadata.

A restore seed is not canonical truth, not a `ContextBlock`, and not compiled context. It is a future input proposal for Phase 7 Context Compiler work and does not execute restoration.

## Hash Rules

`shape_hash` is deterministic for stable semantic shape:

- reference lists are normalized, deduplicated, and sorted
- unstable fields such as snapshot id and timestamps are excluded
- shape-affecting refs, summary, source hash, and metadata are included
- `created_at` changes do not alter `shape_hash`
- semantic shape changes alter `shape_hash`

Token hashing and KV cache identity validation are intentionally deferred to later phases.

## Reference Integrations

Snapshots integrate with FORGE-K systems by reference only:

- Courthouse: snapshots can cite case ids, submitted exhibit refs, admitted exhibit refs, rejected exhibit refs, rulings, contradictions, and supersessions. Snapshot creation does not admit or reject evidence.
- Memory Palace: snapshots can cite route, room, anchor, and candidate refs. Candidate objects remain candidates until Courthouse admission.
- Semantic Algebra: snapshots can cite semantic object refs, operation refs, and derived object refs. Derived objects keep their existing authority and provenance.

Snapshots do not execute syscall requests produced by semantic transforms, do not promote proposals to truth, and do not change CasePacket, Courthouse, Palace, or Semantic Algebra state.

## Current Limitations

- Storage is in-memory and simulator-only.
- No live daemon integration exists.
- No live AI-OS snapshot/restore behavior was modified.
- No Context Compiler, ContextBlock, token hashing, deterministic KV cache, or runtime driver integration is implemented.
- Snapshot metadata is intentionally small and should not carry large canonical content blobs.
