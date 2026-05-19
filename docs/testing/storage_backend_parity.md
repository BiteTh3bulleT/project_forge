# Storage Backend Parity Testing

This document defines the required tests before SQLite-backed live storage can be migrated to Postgres or supported by Redis/Qdrant infrastructure.

## Phase 13A Status

Phase 13A adds backend config, capability contracts, Postgres connector scaffolding, and migration runner scaffolding. Live storage remains SQLite by default.

## Phase 13B-C Status

Phase 13B-C adds Postgres foundation migrations and default-safe parity tests for a low-risk subset:

- migration version records,
- storage backend metadata,
- migration audit records,
- disabled shadow diagnostic report schema,
- disabled shadow diagnostic event schema,
- disabled shadow diagnostic redaction schema.

This is foundation parity, not data parity. The live SQLite schema does not write these new Postgres foundation tables, and live shadow diagnostics remain bounded in memory. Repository data parity begins only after a future repository adapter phase adds an explicit SQLite/Postgres implementation pair.

Default local tests do not require Docker or Postgres. The integration test runs locally only when `FORGE_POSTGRES_TEST_DSN` is set, while CI provisions Postgres and requires that environment variable so the test cannot silently skip there.

## Phase 13D-E Status

Phase 13D-E adds diagnostic persistence and retrieval metadata relational adapter tests without changing the live default backend:

- config tests prove `FORGE_SHADOW_DIAGNOSTIC_PERSISTENCE_ENABLED` defaults disabled and requires Postgres configuration when enabled,
- persistence tests prove disabled mode writes no repository rows while keeping in-memory diagnostics,
- persistence tests prove safe row construction, retention expiry, schema version, unsafe metadata rejection, payload-size rejection, and repository failure isolation,
- local-optional/CI-required Postgres tests migrate the schema, insert/get/list diagnostic reports, and query retention expiry when `FORGE_POSTGRES_TEST_DSN` is set,
- retrieval relational tests prove safe ref/count/class mapping, deterministic serialization, raw query/source/chunk/vector/embedding/memory-content rejection, and no retrieval execution.

This remains diagnostic storage parity only. It does not switch reads, dual-write canonical records, migrate memory/retrieval tables, wire Qdrant, wire Redis, or add public diagnostics APIs.

## Phase 13F-G Status

Phase 13F-G adds Qdrant shadow vector adapter and shadow index tests without changing live retrieval:

- config tests prove `FORGE_QDRANT_SHADOW_INDEX_ENABLED` defaults disabled and requires Qdrant URL configuration when enabled,
- payload safety tests prove safe ref/provenance payloads are accepted and source/chunk/prompt/completion/message/memory/raw-query/auth/secret metadata is rejected,
- adapter tests prove collection creation and upsert payload shape without requiring Qdrant,
- shadow index tests prove disabled mode skips writes, enabled mode upserts only precomputed vectors, vector dimensions are validated, Qdrant/search errors are isolated, and the service cannot execute retrieval or create embeddings,
- local-optional/CI-required Qdrant integration tests are gated by `FORGE_QDRANT_TEST_URL`.

This remains vector shadow infrastructure only. It does not switch retrieval reads, call Qdrant from live retrieval, generate embeddings, write canonical memory, admit evidence, persist content in Qdrant, wire Redis, or add public APIs.

## Phase 13H Status

Phase 13H adds Redis ephemeral coordination boundary tests without changing live jobs, queues, caches, retrieval, gateway, modelruntime, or memory behavior:

- config tests prove `FORGE_REDIS_ENABLED` defaults disabled and requires Redis addr configuration when enabled,
- capability tests prove Redis roles are ephemeral only and forbidden canonical/durable/admission/provenance roles are rejected,
- key policy tests prove safe namespaced keys are accepted while raw content, prompt, query, token, secret, auth, and unsafe path-like keys are rejected,
- TTL tests prove cache, lock, and progress keys require expiration,
- fake adapter tests prove cache set/get, queue push/pop, lock acquire/release, progress append/read, unsafe-key rejection, and health behavior,
- local-optional/CI-required Redis integration tests are gated by `FORGE_REDIS_TEST_ADDR`.

This remains ephemeral coordination infrastructure only. It does not switch live job queues to Redis, make Redis required, store canonical memory, store evidence admission, store provenance authority, add routes, or change public APIs.

## FORGE-K Online Phase 16 Status

Phase 16 adds a pure cutover readiness report under `services/core/internal/storagebackend` and surfaces it read-only through `GET /forge/system/status` as `storage.cutover_readiness`.

- default readiness remains `blocked` with `canonical_default=sqlite`,
- live owner is the existing SQLite store path,
- target owner is future Postgres repository adapters and later FORGE-K persistence only after a separate authority migration,
- Redis and Qdrant are explicitly non-authoritative,
- readiness can mark a selected Postgres domain as proposal-ready only when every required evidence flag is present,
- the report has no effect on backend defaults, dual-write, read switching, Redis authority, Qdrant authority, or FORGE-K authority migration.

This remains readiness metadata only. It does not switch reads, dual-write canonical records, migrate memory/retrieval tables, wire Qdrant into live retrieval, wire Redis into live queues/cache, or change public APIs beyond the existing read-only status payload.

## Required Parity Test Areas

Schema parity:
- Table and index existence.
- Column names, nullability, defaults, and uniqueness.
- Foreign-key behavior.
- JSON field representation.
- Timestamp precision.

Repository parity:
- Create, read, update, delete where supported.
- List ordering and pagination.
- Idempotency behavior.
- Workspace/scope filtering.
- Error behavior for missing records and conflicts.

Transaction parity:
- Commit and rollback behavior.
- Multi-step mutation atomicity.
- Foreign-key rollback behavior.
- Concurrent write behavior where relevant.

Migration parity:
- Migration version table creation.
- Idempotent re-run behavior.
- Transaction-wrapped migrations.
- Failure injection with rollback.
- Schema digest or checksum comparison.

Phase 13B-C migration parity:
- empty migration set succeeds,
- migrations are sorted by version,
- already-applied migrations are skipped,
- failed migrations report version/name context,
- applied versions are recorded,
- SQL registry entries match `services/core/migrations/postgres/*.sql`,
- local-optional/CI-required Postgres integration validates real table creation and idempotent rerun when `FORGE_POSTGRES_TEST_DSN` is provided.

Phase 13D-E diagnostic repository parity:
- diagnostic persistence config defaults disabled,
- enabled diagnostic persistence requires explicit Postgres configuration,
- disabled mode does not write repository rows,
- safe diagnostic rows contain only summaries, refs, counts, classes, warnings, retention metadata, and no-effect verification,
- unsafe or oversized payloads are rejected before persistence,
- repository failure is isolated from the in-memory diagnostic sink,
- local-optional/CI-required Postgres integration is gated by `FORGE_POSTGRES_TEST_DSN`.

Phase 13F-G Qdrant shadow parity:
- qdrant shadow index config defaults disabled,
- enabled qdrant shadow index requires explicit Qdrant URL,
- safe vector payloads preserve object/source/embedding/provenance refs,
- unsafe content-bearing payloads are rejected before upsert,
- disabled mode performs no Qdrant writes,
- local-optional/CI-required Qdrant integration is gated by `FORGE_QDRANT_TEST_URL`.

Phase 13H Redis ephemeral parity:
- redis config defaults disabled,
- enabled redis requires explicit addr,
- redis env does not switch storage backend or SQLite default,
- allowed redis roles are ephemeral only,
- forbidden redis roles cannot be canonical truth, durable memory, evidence admission, provenance authority, sole job record, canonical audit, canonical settings, or vector truth,
- safe key namespace and TTL policy are enforced,
- fake adapter behavior is deterministic,
- local-optional/CI-required Redis integration is gated by `FORGE_REDIS_TEST_ADDR`.

Phase 16 cutover readiness:
- default readiness report is blocked and preserves SQLite as canonical default,
- Postgres proposal readiness requires selected domain, baseline SQLite tests, Postgres migration tests, Postgres adapter tests, repository parity tests, dual-write comparison tests, read-compare mismatch tests, backup/rollback tests, and operator approval,
- Redis and Qdrant are not truth authority even when endpoint configuration is present,
- `/forge/system/status` exposes the readiness report without mutation controls.

Dual-write parity:
- SQLite/Postgres write comparison.
- Read-after-write comparison.
- Checksum/digest comparison per table group.
- Drift reporting.
- Operator-visible rollback plan.

Recovery parity:
- SQLite backup/restore.
- Postgres dump/restore.
- Qdrant rebuild from relational records.
- Redis loss recovery from durable records.

## Backend-Specific Boundaries

SQLite:
- Current live default.
- Source of truth until explicit migration.

Postgres:
- Future durable relational backend.
- Must not become live until parity, backup, rollback, and cutover readiness pass.

Redis:
- Ephemeral only.
- Loss must be recoverable.
- Must not be truth, memory, admissibility, or provenance.

Qdrant:
- Vector acceleration only.
- Must not be truth or admissibility.
- Indexes must be rebuildable from relational provenance records.
- Shadow vector indexing remains disabled by default and non-authoritative.

## Failure Injection Requirements

Future migration phases must test:

- Connection loss.
- Transaction rollback.
- Partial write failure.
- Duplicate migration execution.
- Out-of-order migration detection.
- Redis cache loss.
- Qdrant index loss and rebuild.
- Dual-write drift detection.

## Completion Gate

A backend read switch is not allowed until the migrated table group has:

- passing SQLite baseline tests,
- passing Postgres adapter tests,
- passing parity tests,
- passing migration idempotence tests,
- passing rollback/restore tests,
- operator runbook coverage,
- and documented approval to switch reads.
