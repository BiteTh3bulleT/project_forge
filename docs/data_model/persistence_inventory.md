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

## Backend migration stages

Group A: non-authoritative diagnostics, only if persistence is approved later.

- shadow diagnostic reports
- route/chat/retrieval/advisory diagnostic summaries

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

## Qdrant candidates

Qdrant candidates are vector acceleration only:

- embedding vector indexes
- nearest-neighbor retrieval acceleration
- future shadow vector indexes

Qdrant must not store canonical truth, admissibility, provenance authority, or memory source of record. Qdrant indexes must be rebuildable from relational records.

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
