# Storage Backend Migration

Phase 13A adds the application-side storage backend boundary. Phase 13B-C adds the first Postgres schema foundation and parity tests for storage metadata plus disabled shadow diagnostic schema only. Phase 13D-E adds an opt-in diagnostic persistence repository and retrieval metadata relational adapter scaffold. Phase 13F-G adds a disabled-by-default Qdrant shadow vector adapter and shadow index scaffold. Phase 13H adds a disabled-by-default Redis ephemeral coordination boundary. Phase 13I adds the store cutover readiness review. Phase 14A adds FORGE-K operational cutover design. These phases do not migrate live data.

## Current Live State

- Live storage remains SQLite at `${FORGE_DATA_DIR}/forge.sqlite`.
- `services/core/internal/store.Open` is still the live entry point.
- Existing memory, retrieval, jobs, approvals, audit-adjacent records, and settings continue to use the existing SQLite-backed service paths.
- Shadow diagnostics remain bounded in memory by default. Postgres diagnostic persistence exists only behind `FORGE_SHADOW_DIAGNOSTIC_PERSISTENCE_ENABLED=false` and requires explicit Postgres configuration.
- Docker-managed Postgres, Redis, and Qdrant are infrastructure-ready but not live authority.
- Qdrant shadow indexing defaults disabled and is not part of live retrieval.
- Redis ephemeral coordination defaults disabled and is not part of live jobs, gateway, modelruntime, retrieval, memory, or public API behavior.

## Backend Selection

`FORGE_STORE_BACKEND` is parsed as:

- `sqlite`: default and current live backend.
- `postgres`: future durable relational backend; requires `FORGE_POSTGRES_DSN` in backend config validation.

Unset `FORGE_STORE_BACKEND` means `sqlite`. Redis and Qdrant endpoint variables do not imply a backend switch.

Phase 13D-E intentionally keeps the live runtime on SQLite. Phase 13I records that Postgres is foundation-ready but not canonical-ready. A later cutover phase must add explicit adapter tests, migration tests, rollback tests, and operator runbooks before Postgres can become live.

## Phase 13B-C Foundation Schema

The first Postgres schema target is foundation-only:

- `forge_schema_migrations`: migration version records with checksums.
- `storage_backend_metadata`: low-risk backend metadata records.
- `storage_migration_audit`: migration audit records.
- `shadow_diagnostic_reports`: disabled shadow diagnostic report persistence shape.
- `shadow_diagnostic_report_events`: disabled diagnostic event persistence shape.
- `shadow_diagnostic_redactions`: disabled diagnostic redaction persistence shape.

The shadow diagnostic tables were schema-only in Phase 13B-C. Phase 13D-E extends the schema with retention expiry, no-effect verification, and schema-version metadata used by the opt-in diagnostic repository.

## Phase 13D-E Diagnostic Persistence

Phase 13D-E adds `services/core/internal/forgekshadow` diagnostic persistence primitives:

- `FORGE_SHADOW_DIAGNOSTIC_PERSISTENCE_ENABLED` defaults to `false`.
- `FORGE_SHADOW_DIAGNOSTIC_RETENTION_DAYS` defaults to `30`.
- `FORGE_SHADOW_DIAGNOSTIC_MAX_PAYLOAD_BYTES` defaults to `65536`.
- Enabling persistence validates that a Postgres DSN is configured and does not switch `FORGE_STORE_BACKEND`.
- The persistence sink is best-effort: the existing in-memory sink still stores reports, and repository failures do not fail live diagnostic handling.
- Persisted rows contain only safe report identity, workspace/request/correlation refs, route/chat/retrieval/advisory summary counts and classes, warning summaries, retention expiry, no-effect verification, and schema version.
- Persisted rows reject unsafe metadata and size overrun. They do not contain request or response bodies, prompts, completions, message bodies, source/chunk text, raw queries, snippets, embeddings, vectors, tool payloads, memory content, auth headers, cookies, tokens, or secrets.

Phase 13D-E also adds a retrieval metadata relational adapter scaffold. It maps existing retrieval metadata observations into relational-safe DTOs and deterministic canonical JSON. It cannot execute retrieval, call embedding/search providers, compile context, admit evidence, write memory, call Qdrant, or implement live RAG.

## Phase 13F-G Qdrant Shadow Vector Adapter

Phase 13F-G adds `services/core/internal/vectorstore`:

- a generic `VectorStore` interface,
- a Qdrant HTTP adapter,
- safe vector payload validation,
- deterministic point ID generation,
- a disabled-by-default `ShadowIndexService`,
- optional Qdrant integration tests gated by `FORGE_QDRANT_TEST_URL`.

Configuration:

- `FORGE_QDRANT_SHADOW_INDEX_ENABLED=false`
- `FORGE_QDRANT_URL`
- `FORGE_QDRANT_COLLECTION=forge_shadow_embeddings`
- `FORGE_QDRANT_VECTOR_SIZE`
- `FORGE_QDRANT_TIMEOUT_MS=3000`

The shadow index accepts only already-produced vectors plus safe relational refs. It never creates embeddings, executes retrieval, calls live search, changes retrieval ranking, admits evidence, writes canonical memory, or changes the live store backend. Qdrant payloads are ref-only and must not contain source text, chunk text, document content, prompts, completions, message bodies, raw queries, memory content, tool payloads, auth values, secrets, or large raw content.

Qdrant indexes remain rebuildable from relational embedding records. A future rebuild command may validate dimensions/model identity and explicitly recreate an index, but no automatic destructive rebuild is added in Phase 13F-G.

## Phase 13H Redis Ephemeral Boundary

Phase 13H adds `services/core/internal/ephemeral`:

- Redis role contracts for cache, queue, lock, pub/sub, progress stream, rate-limit window, and ephemeral coordination,
- forbidden role checks for canonical truth, durable memory, evidence admission, provenance authority, sole job record, canonical audit, canonical settings, and vector truth,
- safe key namespace policy with deterministic prefixing,
- TTL requirements for cache, lock, and progress entries,
- fake in-memory adapter tests,
- stdlib Redis client scaffold,
- optional Redis integration tests gated by `FORGE_REDIS_TEST_ADDR`.

Configuration:

- `FORGE_REDIS_ENABLED=false`
- `FORGE_REDIS_ADDR`
- `FORGE_REDIS_KEY_PREFIX=forge`
- `FORGE_REDIS_TIMEOUT_MS=1000`

Redis is ephemeral infrastructure only. It may mirror bounded status, coordinate short-lived locks, or hold recoverable queue/cache/progress metadata in future phases. It must not store canonical truth, durable memory, evidence admission state, provenance authority, audit authority, settings authority, raw prompts, raw content, secrets, or sole job records.

Phase 13H does not switch live job queues to Redis, make Redis required, add public routes, change gateway/modelruntime/retrieval behavior, write live memory, or alter SQLite/Postgres authority.

The Postgres runner executes migrations in deterministic version order, skips already-applied versions, records applied versions with checksums, and runs inside a transaction. A separate migration lock is not taken in Phase 13B-C; live Postgres migration execution is not part of the default daemon path yet. A future live migration phase must add explicit lock policy before concurrent operator execution is allowed.

## Target Roles

Postgres:
- Future durable relational store for canonical memory, jobs, approvals, settings, audit-adjacent records, retrieval metadata, and later FORGE-K persistence.
- May become canonical-capable only after parity and cutover gates pass.

Redis:
- Ephemeral coordination only: cache, queue, pub/sub, locks, streams, and progress metadata.
- Must be recoverable from durable records.
- Must not become canonical truth, memory, admissibility, or provenance authority.
- Phase 13H Redis support is disabled by default and non-canonical.

Qdrant:
- Vector retrieval acceleration only.
- Vectors, nearest-neighbor hits, and scores are retrieval signals, not evidence admission or truth.
- Qdrant indexes must be rebuildable from relational records.
- Phase 13F-G Qdrant shadow indexing is disabled by default and non-authoritative.

## Migration Phases

1. Phase 13A: backend config, capability contracts, Postgres connector scaffold, migration runner scaffold, docs, and parity plan.
2. Phase 13B: Postgres schema foundation for storage metadata and disabled shadow diagnostic schema.
3. Phase 13C: SQLite/Postgres foundation parity tests for migration shape, ordering, idempotence, JSONB fields, and timestamps.
4. Phase 13D: diagnostic store persistence after explicit approval.
5. Phase 13E: retrieval metadata relational adapter scaffold.
6. Phase 13F: Qdrant vector adapter design and scaffold.
7. Phase 13G: Qdrant shadow vector index, rebuildable, disabled by default, and non-authoritative.
8. Phase 13H: Redis queue/cache boundary with loss-safe behavior, disabled by default and non-canonical.
9. Phase 13I: store cutover readiness review.
10. Phase 14A: operational cutover design; no backend switch or live authority migration.

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
- repository parity tests pass for the selected table group.
- dual-write comparison is designed and disabled by default.
- Dual-write comparison is clean.
- Backup and rollback are documented.
- Operator-visible config clearly selects the backend.

## Rollback Strategy

- SQLite remains available until the cutover phase is complete and reversible.
- Postgres migrations must be versioned and idempotent.
- Redis state must be disposable.
- Qdrant indexes must be rebuildable.
- Backups must exist before any canonical table migration.
- Any authority migration must have a tested config disablement or rollback path before merge.

## Table Migration Priority

Group A:
- Shadow diagnostic report persistence, opt-in and disabled by default.
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

- Do not make Postgres the default backend in Phase 13A through Phase 13I or Phase 14A.
- Do not dual-write live data in Phase 13A through Phase 13I or Phase 14A.
- Do not wire Redis into live queues or caches in Phase 13A through Phase 13I or Phase 14A.
- Do not wire Qdrant into live retrieval in Phase 13A through Phase 13I or Phase 14A.
- Do not make Redis or Qdrant canonical truth.
- Do not make vector hits admissible evidence.
- Do not change public APIs, routes, gateway behavior, modelruntime behavior, retrieval behavior, or memory semantics.
