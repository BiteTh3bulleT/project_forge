# AUDIT_AND_TRACE

## What is logged

Gateway actions persist:

- full invocation request envelope
- policy outcome + status
- approval linkage (when present)
- completion metadata and result payload summary

Audit records persist:

- category (`gateway`, `permissions`, `backup`, etc.)
- action (`tool.executed`, `tool.denied`, `tool.needs_approval`, ...)
- correlation id
- job and invocation linkage
- risk class
- outcome
- summary + payload

## Correlation

`correlationId` ties together:

- chat/job origin
- gateway invocation rows
- approval steps
- audit records

## Inspection

- `GET /api/gateway/invocations`
- `GET /api/audit`
- `GET /api/audit/trace/{correlationId}` (correlation-first report: gateway invocations, audit records, artifact records, provenance records, journal events, artifact refs, and explicit link edges)

Desktop pages:

- `#/gateway`
- `#/audit`

## Failure Visibility

Failed or denied invocations remain visible with reason text.

No hidden retries or silent mutation paths are used by the gateway pipeline.

## Retention and Archive Posture

Current behavior:

- `audit_records` and `journal_events` are append-only SQLite records. Store migrations install triggers that block `UPDATE` and `DELETE` on both tables.
- Audit rows are written through `internal/audit.Service`; semantic journal rows are appended through the Control Lane SQLite journal store.
- Query surfaces are bounded for normal inspection (`GET /api/audit`, trace reports, and recent journal lookups), but the underlying tables are not automatically pruned.
- `full_backup` includes `audit_records`, `gateway_invocations`, `provenance_records`, and `journal_events`. Audit and journal restore use insert-only semantics (`ON CONFLICT ... DO NOTHING`) so existing immutable rows are not rewritten.
- Operational backup/export files are written under `${FORGE_DATA_DIR}/backups/` and `${FORGE_DATA_DIR}/exports/`; `${FORGE_DATA_DIR}` defaults to `~/.config/forge`.

Retention gap:

- There is no live retention window, compaction job, automated rotation, cold archive writer, retention manifest, or operator alert for audit/journal table growth yet.
- Because audit and journal tables are intentionally append-only, any future retention mechanism must not delete or rewrite rows in-place as its first step.

Recommended rotation/archive approach:

1. Create a governed archive bundle for closed time ranges, for example monthly or when the SQLite database crosses an operator-defined size threshold.
2. Include `audit_records`, `gateway_invocations`, `provenance_records`, `journal_events`, bundle metadata, row counts, min/max timestamps, and content checksums.
3. Write the bundle to the governed backup/export directories first, then copy it to operator-managed cold storage.
4. Verify archive readability and checksums before considering any pruning phase.
5. Add pruning only as a later explicit implementation phase with approval, audit evidence for the archive/prune decision, and tests proving trace reconstruction still works from live plus archived segments.

Not implemented yet:

- automatic audit/journal rotation
- archive manifests beyond the existing backup bundle metadata
- retention policy configuration
- disk-usage warning thresholds for audit/journal growth
- pruning of archived audit or journal rows
