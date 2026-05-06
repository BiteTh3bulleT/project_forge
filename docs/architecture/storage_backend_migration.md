# Storage Backend Migration

Phase 13A adds the application-side storage backend boundary. It does not migrate live data.

## Current Live State

- Live storage remains SQLite at `${FORGE_DATA_DIR}/forge.sqlite`.
- `services/core/internal/store.Open` is still the live entry point.
- Existing memory, retrieval, jobs, approvals, audit-adjacent records, settings, and shadow diagnostics continue to use the existing SQLite-backed service paths.
- Docker-managed Postgres, Redis, and Qdrant are infrastructure-ready but not live authority.

## Backend Selection

`FORGE_STORE_BACKEND` is parsed as:

- `sqlite`: default and current live backend.
- `postgres`: future durable relational backend; requires `FORGE_POSTGRES_DSN` in backend config validation.

Unset `FORGE_STORE_BACKEND` means `sqlite`. Redis and Qdrant endpoint variables do not imply a backend switch.

Phase 13A intentionally keeps the live runtime on SQLite. A later cutover phase must add explicit adapter tests, migration tests, rollback tests, and operator runbooks before Postgres can become live.

## Target Roles

Postgres:
- Future durable relational store for canonical memory, jobs, approvals, settings, audit-adjacent records, retrieval metadata, and later FORGE-K persistence.
- May become canonical-capable only after parity and cutover gates pass.

Redis:
- Ephemeral coordination only: cache, queue, pub/sub, locks, streams, and progress metadata.
- Must be recoverable from durable records.
- Must not become canonical truth, memory, admissibility, or provenance authority.

Qdrant:
- Vector retrieval acceleration only.
- Vectors, nearest-neighbor hits, and scores are retrieval signals, not evidence admission or truth.
- Qdrant indexes must be rebuildable from relational records.

## Migration Phases

1. Phase 13A: backend config, capability contracts, Postgres connector scaffold, migration runner scaffold, docs, and parity plan.
2. Phase 13B: Postgres schema foundation for a small non-authoritative subset.
3. Phase 13C: SQLite/Postgres parity tests for schema, read/write behavior, ordering, timestamps, transactions, and JSON fields.
4. Phase 13D: diagnostic store persistence after explicit approval.
5. Phase 13E: retrieval metadata Postgres adapter.
6. Phase 13F: Qdrant vector adapter design.
7. Phase 13G: Qdrant shadow vector index, rebuildable and non-authoritative.
8. Phase 13H: Redis queue/cache boundary with loss-safe behavior.
9. Phase 13I: store cutover readiness review.

## Parity Strategy

- Every migrated table needs SQLite/Postgres schema parity tests.
- Every migrated repository needs write/read/list/update/delete behavior parity tests.
- Pagination and list ordering must be deterministic across backends.
- JSON field behavior must be normalized.
- Timestamp precision must be explicit.
- Transaction semantics and foreign-key behavior must be tested.
- Dual-write phases must compare checksums before any read switch.

## Cutover Strategy

No read switch happens until:

- SQLite baseline tests pass.
- Postgres schema migrations are idempotent.
- Postgres adapter tests pass without Docker dependency for default tests.
- Integration tests pass when Docker services are available.
- Dual-write comparison is clean.
- Backup and rollback are documented.
- Operator-visible config clearly selects the backend.

## Rollback Strategy

- SQLite remains available until the cutover phase is complete and reversible.
- Postgres migrations must be versioned and idempotent.
- Redis state must be disposable.
- Qdrant indexes must be rebuildable.
- Backups must exist before any canonical table migration.

## Table Migration Priority

Group A:
- Future shadow diagnostic report persistence, only if persistence is approved later.
- Non-authoritative diagnostics only.

Group B:
- Jobs.
- Settings.
- Approvals.
- Audit-adjacent records.

Group C:
- Memory notes, observations, links, and provenance.
- Retrieval metadata.
- Embeddings metadata.

Group D:
- FORGE-K canonical persistence, later only and only after authority migration design.

## Forbidden Authority Changes

- Do not make Postgres the default backend in Phase 13A.
- Do not dual-write live data in Phase 13A.
- Do not wire Redis into live queues or caches in Phase 13A.
- Do not wire Qdrant into live retrieval in Phase 13A.
- Do not make Redis or Qdrant canonical truth.
- Do not make vector hits admissible evidence.
- Do not change public APIs, routes, gateway behavior, modelruntime behavior, retrieval behavior, or memory semantics.
