# Persistence Inventory (Phase 3)

## Current persistence stack

- Primary storage: SQLite (`modernc.org/sqlite`) opened by `services/core/internal/store/store.go`.
- Database file: `${FORGE_DATA_DIR}/forge.sqlite`.
- Migration pattern: single schema string executed via `internal/store/migrate.go` (`CREATE TABLE IF NOT EXISTS ...` + indexes/triggers).
- Connection mode: WAL with foreign keys enabled, single open connection (`SetMaxOpenConns(1)`).

## Managed service direction

The Docker stack includes Postgres, Redis, and Qdrant as infrastructure for the next storage phase.

- Postgres is the intended durable relational backend after adapter and migration work.
- Redis is intended for ephemeral cache/queue/stream/lock workloads only.
- Qdrant is intended for vector retrieval acceleration only.

As of this inventory, the application still uses SQLite for live persistence. The managed services do not replace SQLite until explicit storage adapters, migrations, dual-write or migration tooling, backup/restore policy, and tests exist.

Phase 13A adds the storage backend foundation:

- `FORGE_STORE_BACKEND=sqlite|postgres` with `sqlite` as the default.
- `FORGE_POSTGRES_DSN` parsed for future Postgres connection scaffolding.
- `FORGE_REDIS_ADDR` and `FORGE_QDRANT_URL` parsed as infrastructure endpoints only.
- backend capability contracts that keep Redis and Qdrant out of canonical truth authority.
- a Postgres migration runner scaffold with a migration version table only.

Phase 13A does not migrate live tables, dual-write data, switch reads, wire Redis into queues/caches, or wire Qdrant into retrieval.

Phase 13B-C adds the first Postgres schema foundation and parity tests:

- idempotent Postgres migrations under `services/core/migrations/postgres`.
- migration version records with checksums.
- `storage_backend_metadata` and `storage_migration_audit`.
- disabled shadow diagnostic report/event/redaction schema.
- migration runner tests for deterministic ordering, applied-version skips, failure reporting, version recording, SQLite skip behavior, and optional Postgres integration through `FORGE_POSTGRES_TEST_DSN`.

Phase 13B-C does not persist live shadow diagnostics, migrate live memory or retrieval tables, dual-write data, switch reads, make Postgres the default, or wire Redis/Qdrant into live behavior.

Phase 13D-E adds the first safe diagnostic persistence primitives:

- `FORGE_SHADOW_DIAGNOSTIC_PERSISTENCE_ENABLED=false` by default.
- Postgres diagnostic repository for safe shadow diagnostic summaries only.
- retention expiry, no-effect verification, and schema-version metadata on `shadow_diagnostic_reports`.
- retrieval metadata relational adapter scaffold for refs/counts/classes/summaries only.
- optional Postgres integration tests gated by `FORGE_POSTGRES_TEST_DSN`.

Phase 13D-E does not switch the live store, dual-write canonical records, migrate memory or retrieval tables, persist raw diagnostic content, create a public diagnostics API, execute retrieval/search/embeddings, wire Qdrant, or wire Redis.

Phase 13F-G adds Qdrant shadow vector infrastructure:

- `FORGE_QDRANT_SHADOW_INDEX_ENABLED=false` by default.
- `services/core/internal/vectorstore` defines the vector-store interface, Qdrant adapter, safe payload schema, and disabled shadow index service.
- Qdrant payloads may contain safe object/source/workspace/embedding/provenance refs, embedding model/dims, source hash, retrieval strategy/index class, schema version, and timestamps only.
- Qdrant payloads must not contain source text, chunk text, document content, prompts, completions, message bodies, raw queries, memory content, tool payloads, auth values, secrets, or large raw blobs.
- Optional Qdrant integration tests are gated by `FORGE_QDRANT_TEST_URL`.

Phase 13F-G does not make Qdrant the live retrieval backend, execute retrieval through Qdrant, generate embeddings, switch reads, change result ordering, write canonical memory, or make vector hits admissible evidence.

Phase 13H adds Redis ephemeral coordination infrastructure:

- `FORGE_REDIS_ENABLED=false` by default.
- `FORGE_REDIS_KEY_PREFIX=forge` and `FORGE_REDIS_TIMEOUT_MS=1000` define safe namespace and connection defaults.
- `services/core/internal/ephemeral` defines Redis roles, safe key policy, TTL requirements, fake adapter tests, and a stdlib Redis client scaffold.
- Redis keys must be prefixed, bounded, workspace-safe, and free of raw prompt/content/query/auth/secret/token material.
- Cache, lock, and progress entries require TTLs.
- Optional Redis integration tests are gated by `FORGE_REDIS_TEST_ADDR`.

Phase 13H does not switch live job queues to Redis, make Redis required, store canonical truth, durable memory, evidence admission, provenance authority, settings authority, or sole job records, or change gateway/modelruntime/retrieval/memory behavior.

## Backend migration stages

Group A: non-authoritative diagnostics.

- shadow diagnostic reports
- route/chat/retrieval/advisory diagnostic summaries

Phase 13B-C defines the Postgres table shape for Group A diagnostics. Phase 13D-E adds the opt-in repository and persistence sink wrapper. The flag remains disabled by default, rows are non-authoritative diagnostics, and canonical memory remains SQLite-backed.

Group B: operational relational records after parity exists.

- jobs
- settings
- approvals
- audit-adjacent records
- gateway invocation metadata

Group C: memory and retrieval records after stronger parity exists.

- memory observations, notes, links, usefulness, and provenance
- retrieval runs and results
- embedding metadata
- VSA inspectability tables

Group D: FORGE-K canonical persistence, later only.

- Kernel journal/state persistence
- Courthouse evidence/admission records
- Semantic Algebra objects
- snapshots, context compiler metadata, KV manifests, runtime proposal records, consensus records

Tables remain SQLite until their group has schema parity, write/read parity, transaction parity, rollback/restore coverage, and explicit cutover approval.

## Redis candidates

Redis candidates are ephemeral only:

- queue coordination
- job progress stream mirrors
- locks
- bounded caches
- pub/sub notifications

Redis must not store canonical truth, durable memory, evidence admission, provenance, or source-of-record state. Redis loss must be recoverable from SQLite/Postgres records.

Phase 13H implements the boundary only. Redis remains disabled by default and may not become a live queue/cache authority without a later explicit wiring and parity phase.

## Qdrant candidates

Qdrant candidates are vector acceleration only:

- embedding vector indexes
- nearest-neighbor retrieval acceleration
- future shadow vector indexes

Qdrant must not store canonical truth, admissibility, provenance authority, or memory source of record. Qdrant indexes must be rebuildable from relational records.

Phase 13F-G shadow vector indexes are rebuildable from relational embedding records and remain disabled by default.

## Current entity/table pattern

- Service-level SQL repositories in `services/core/internal/*` (`jobs`, `approvals`, `audit`, `gateway`, `artifacts`, `memory`, `retrieval`, `policy`, etc.).
- Strong use of JSON columns (`*_json`) for flexible metadata/payloads.
- Existing truth/projection evidence already persisted in:
  - `events`, `job_events`, `jobs`, `job_status_history`
  - `approval_requests`, `approval_decisions`
  - `gateway_invocations`, `audit_records`
  - `artifacts`
  - `task_packets`
  - memory/retrieval/policy tables, including VSA inspectability tables.

## Current transaction strategy

- Most services use direct `db.ExecContext`/`QueryContext`.
- Multi-step sensitive flows use explicit `sql.Tx` (`approvals.Decide`, ingest update chains, and now AI-OS semantic transaction runner).
- Phase 2 kernel transaction abstraction now supports:
  - in-memory runner
  - SQLite runner (`internal/aios/controllane/sqlite_store.go`)

## Current test strategy

- Tests create isolated SQLite stores with `store.Open(t.TempDir())`.
- Patterns already present in `permissions`, `approvals`, `gateway` tests.
- Phase 3 adds integration tests using real migrated DB through syscall processor.

## Audit persistence strategy

- `audit_records` is append-style, queried by correlation id and trace.
- `controllane.CoreAuditSink` bridges syscall audit to `internal/audit.Service`.
- Phase 3 adds audit linkage columns to cognitive filesystem entities (`audit_id`, `correlation_id`, `trace_id`, `syscall_id`).

## Backup/export/import strategy

- `internal/backup` exports selected table sections by bundle type.
- Phase 3 extends `full_backup` extraction inventory with cognitive filesystem tables.
- Restore path remains conservative (only selected policy-shaped tables are currently import-mapped), documented as a follow-up.

## Existing workspace/scope concepts

- Existing repo has path/scope semantics in lanes/permissions/gateway.
- Cognitive filesystem persistence uses explicit `workspace_id`, `lane_id`, `selected_paths_json` for semantic entities.

## Existing ID/correlation/trace concepts

- Existing systems already use integer ids and string ids (jobs, packets, approvals, audit).
- Correlation id is already first-class in audit/gateway.
- Phase 3 semantic entities store:
  - object id
  - `syscall_id`
  - `correlation_id`
  - `trace_id`
  - `audit_id` (linked after audit write)
  - provenance references.

## Cognitive filesystem placement

Added in SQLite schema (`internal/store/migrate.go`):

- `provenance_records`
- `journal_events`
- `memory_notes`
- `semantic_links`
- `state_items`
- `state_versions`
- `open_loops`
- `artifact_refs`
- `derived_models`
- `contradiction_records`
- `supersession_records`
- `context_packet_snapshots`
- `semantic_idempotency_keys`

## Retrieval + Memory VSA persistence

Added for inspectable VSA behavior in retrieval/memory lanes:

- `memory_vsa_pointers`
- `memory_vsa_role_bindings`
- `memory_vsa_associations`
- `retrieval_result_vsa_signals`
- `memory_vsa_reindex_runs`
- `memory_vsa_reindex_items`

These remain in the existing memory/retrieval subsystem (not canonical cognitive syscall state), and are linked to observations/results for operator inspection.

## Reused concepts

- Reused existing SQLite + service SQL pattern (no new framework).
- Reused existing audit service and correlation id discipline.
- Reused existing backup extraction mechanism.
- Extended Phase 2 transaction runner rather than replacing kernel flow.

## New schema required and why

- Existing `events` table is minimal and lacks workspace/provenance/correlation/audit linkage needed for cognitive filesystem invariants.
- Existing memory tables are specialized for retrieval observations, not canonical AI-OS semantic syscall state.
- New cognitive filesystem tables provide deterministic, inspectable, workspace-scoped semantic state/history.

## Risks and assumptions

- Migration remains monolithic; future refactor to versioned migrations may be needed as schema grows.
- Backup extraction includes cognitive tables, but restore import mappings are intentionally limited today.
- `context_packet_snapshots` is implemented as evidence; full context compiler behavior remains a later phase.
- Current read paths still rely on SQLite JSON functions; portability to other stores would need adapter work.
