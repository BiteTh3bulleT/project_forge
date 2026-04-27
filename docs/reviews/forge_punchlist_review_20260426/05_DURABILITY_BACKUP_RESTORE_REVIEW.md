# Durability / Backup / Restore Review

## Scorecard

- Cognitive filesystem tables: GOOD/PARTIAL
- SQLite migrations: GOOD/PARTIAL
- Backup section coverage: GOOD
- Restore integrity gates: PARTIAL/RISK
- VSA/vector posture: GOOD/PARTIAL

## Findings

GOOD: Core cognitive tables exist for journal, memory notes, semantic links, state, versions, loops, artifacts, derived models, contradictions, supersessions, context snapshots, and Dream reports.

GOOD: Full backup includes cognitive filesystem, gateway/audit traces, approvals/capabilities, modelruntime state, context snapshots, Dream reports, and export-only VSA sections.

RISK: Restore computes/records row counts and checksums, but review did not confirm a fail-closed pre-mutation gate on mismatched bundle checksums/entity counts.

RISK: `RestoreBundle` accepts a file path. This needs path-boundary tests and possibly stronger policy gating.

PARTIAL: Only `journal_events` has hard append-only DB triggers. Other documented evidence/history rows need an explicit immutability decision: `state_versions`, `context_packet_snapshots`, `contradiction_records`, `supersession_records`, `artifact_refs`, and `dream_reports`.

PARTIAL: Dream report persistence uses upsert by report ID. That is deterministic, but not append-only evidence.

PARTIAL: VSA sections are export-only/rebuildable, while `embedding_records` are restored as retrieval index. The distinction should remain explicit.

## Migration / Index Recommendations

- Add restore candidate index matching workspace/lane/snapshot_kind/created_at after candidate query is fixed.
- Decide whether immutable evidence tables need triggers or audited update-only paths.
- Add migration tests for Dream report table idempotence, restore snapshot metadata columns, and backup manifest coverage.

## Punchlist

- `DUR-001`: Fail closed on backup checksum/entity-count mismatch before restore mutation.
- `DUR-002`: Add restore path-boundary and approval/policy tests.
- `DUR-003`: Decide and enforce immutability posture for evidence/history tables.
- `DUR-004`: Add FTS/index rebuild verification after restore.
- `DUR-005`: Document `embedding_records` restore policy vs VSA export-only policy.

