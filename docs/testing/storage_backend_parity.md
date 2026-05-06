# Storage Backend Parity Testing

This document defines the required tests before SQLite-backed live storage can be migrated to Postgres or supported by Redis/Qdrant infrastructure.

## Phase 13A Status

Phase 13A adds backend config, capability contracts, Postgres connector scaffolding, and migration runner scaffolding. Live storage remains SQLite by default.

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
