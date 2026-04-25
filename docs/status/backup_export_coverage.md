# Backup / Export Coverage (Phase 5.996)

Date: 2026-04-22
Scope: `full_backup` export breadth vs restore parity in `services/core/internal/backup/service.go`

## Current state

`backup.Service` now restores almost all high-value exported AI-OS data with FK-safe ordering, explicit unsupported reporting, and transaction-backed apply semantics for supported sections.

Restore behavior:
- does **not** hard-delete existing rows
- uses upsert/insert semantics by section
- reports non-restorable sections under `RestoreResult.unsupported`
- reports policy-locked export-only sections under `RestoreResult.exportOnly`
- reports hard restore failures under `RestoreResult.errors`
- applies supported selected sections in one DB transaction (`RestoreResult.atomic=true`); failures roll back and mark `rolledBack=true`
- reports transactional guarantee scope via `RestoreResult.atomicScope` (`db-supported-sections-only`)
- explicitly reports non-global atomic posture via `RestoreResult.globalAtomic=false`
- reports per-section non-DB limitations via `RestoreResult.nonDbSideEffects` and mirrored `RestoreResult.warnings` (for example artifact file bytes are not imported/rollback-managed)

## Coverage matrix

| Data area | Exported in `full_backup` | Restorable | Parity status |
|---|---|---|---|
| Dossiers (`dossiers`) | yes | yes | parity |
| Packets (`task_packets`) | yes | yes | parity |
| Events (`events`) | yes | yes | parity |
| Jobs (`jobs`) | yes | yes | parity |
| Job timelines (`job_status_history`, `job_events`) | yes | yes | parity |
| Approvals (`approval_requests`, `approval_decisions`) | yes | yes | parity |
| Artifacts (`artifacts`) | yes | yes | parity |
| Policy/config (`permission_profiles`, `approval_presets`, `dossier_profiles`, `execution_strategies`, `automation_rules`, `action_lanes`) | yes | yes | parity |
| Provenance + journal (`provenance_records`, `journal_events`) | yes | yes (`journal_events` upsert is do-nothing on id conflict) | parity (append-safe) |
| Semantic idempotency (`semantic_idempotency_keys`) | yes | yes (insert-only on key conflict) | parity (replay-safe) |
| Cognitive FS core (`memory_notes`, `semantic_links`, `state_items`, `state_versions`, `open_loops`, `artifact_refs`, `derived_models`, `contradiction_records`, `supersession_records`, `context_packet_snapshots`) | yes | yes | parity |
| Autonomy settings (`autonomy_settings`) | yes | yes | parity |
| Project context (`project_context_records`) | yes | yes | parity |
| Evaluations (`evaluation_records`) | yes | yes | parity |
| Audit history (`audit_records`, `gateway_invocations`) | yes (limited extract window) | yes (`audit_records` are insert-only on id conflict to preserve immutability triggers) | parity within export window |
| VSA tables (`memory_vsa_*`, `retrieval_result_vsa_signals`) | yes | no | export-only by explicit restore policy |

## Claim boundary

FORGE can now claim **material high-value restore parity** for approvals/events/jobs/artifacts, semantic idempotency keys, project context/evaluations, audit/gateway history, and core cognitive/autonomy sections. Approval expiry fields are included in restore mappings, with a legacy 24-hour TTL backfill for older bundles that lack expiry columns.

FORGE still cannot claim full parity for VSA-derived export-only sections; restore now reports them as explicit policy-locked export-only sections.
Restore is atomic only for supported DB sections; it is not a global filesystem/external side-effect rollback engine.
`RestoreResult.globalAtomic` remains `false` even when `RestoreResult.atomic=true`.

## Next parity targets

1. If VSA restore is ever added, preserve derivation/provenance guarantees and avoid bypassing reindex invariants.
2. Keep export window limitations explicit for `audit_records`/`gateway_invocations` (`LIMIT 5000` extract policy).
