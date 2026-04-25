# Persistence, Migration, and Backup Review

## Schema Health

GOOD: SQLite is the durable substrate. Migrations are centralized in `services/core/internal/store/migrate.go`.

GOOD: Cognitive filesystem tables, journal tables, context snapshots, model runtime tables, VSA tables, autonomy tables, and audit/gateway tables are present.

GOOD: Append-only `journal_events` behavior is enforced with DB triggers.

PARTIAL: Migration style is monolithic. It is workable now, but schema growth argues for versioned migration steps with explicit compatibility tests.

## Backup/Restore

GOOD: Backup/restore coverage is broader than early phases and includes many cognitive and operational tables.

RISK: `full_backup` misses retrieval/observation evidence:

- `retrieval_runs`
- `retrieval_results`
- `retrieval_result_selection`
- `packet_retrieval_runs`
- `memory_observations`
- observation links/usefulness compatibility state

Impact: restore can lose retrieval provenance, packet selection rationale, and usefulness feedback even though those surfaces affect operator trust and future ranking.

RISK: Bundle restore does not verify bundle hash/entity counts before applying. Export stores SHA-256 metadata, but restore accepts a path and structural JSON validity.

RISK: Restore atomicity is DB-scoped only. This is documented, but operator UI should make `globalAtomic=false` impossible to miss.

## Retrieval Persistence

RISK: Retrieval run persistence is not transaction-scoped. A partial failure can leave a run without complete result/observation side effects.

RECOMMENDATION: Wrap retrieval run/result persistence in a transaction. Decide whether observation/VSA side effects are part of that transaction or explicitly best-effort with report warnings.

## Index Recommendations

RECOMMENDATION: Confirm indexes for:

- context snapshots by workspace/lane/kind/created_at
- restore score/report inspection by correlation and snapshot id
- retrieval runs by workspace/dossier/created_at
- audit/gateway/journal by correlation id and trace id
- Dream reports if persisted

## Next Migration Recommendations

1. Add durable `dream_runs`/`dream_proposals` or append-only equivalent.
2. Add backup sections for retrieval/observation compatibility tables.
3. Add restore bundle verification fields and tests.
4. Add approval/request fingerprint table or columns for gateway approval binding.
5. Add model management approval/audit state if not routed through gateway.

