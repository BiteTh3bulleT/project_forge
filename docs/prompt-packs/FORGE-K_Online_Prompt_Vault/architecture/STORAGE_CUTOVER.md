# Storage Cutover

## Rule

Do not combine FORGE-K authority migration with database cutover unless a phase explicitly requires it.

## Safe order

1. SQLite remains canonical.
2. Postgres repository adapters.
3. Dual-write shadow mode.
4. Read-compare mode.
5. Checksums and drift reports.
6. Rollback to SQLite.
7. Operator approval.
8. Default switch.

## Boundaries

- Qdrant is shadow/vector infrastructure, not truth.
- Redis is ephemeral coordination, not canonical state.
