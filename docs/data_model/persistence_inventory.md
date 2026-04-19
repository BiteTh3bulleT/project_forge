# Persistence Inventory (Phase 3)

## Current persistence stack

- Primary storage: SQLite (`modernc.org/sqlite`) opened by `services/core/internal/store/store.go`.
- Database file: `${FORGE_DATA_DIR}/forge.sqlite`.
- Migration pattern: single schema string executed via `internal/store/migrate.go` (`CREATE TABLE IF NOT EXISTS ...` + indexes/triggers).
- Connection mode: WAL with foreign keys enabled, single open connection (`SetMaxOpenConns(1)`).

## Current entity/table pattern

- Service-level SQL repositories in `services/core/internal/*` (`jobs`, `approvals`, `audit`, `gateway`, `artifacts`, `memory`, `retrieval`, `policy`, etc.).
- Strong use of JSON columns (`*_json`) for flexible metadata/payloads.
- Existing truth/projection evidence already persisted in:
  - `events`, `job_events`, `jobs`, `job_status_history`
  - `approval_requests`, `approval_decisions`
  - `gateway_invocations`, `audit_records`
  - `artifacts`
  - `task_packets`
  - memory/retrieval/policy tables.

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
