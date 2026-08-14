# Durability Gaps (Phase 5.996)

Date: 2026-04-21
Scope: durable storage + backup/restore parity + restart safety.

## Durability matrix

| Subsystem | Durable storage | In `full_backup` export | Restore parity | Audit-linked | Lost on restart |
|---|---|---|---|---|---|
| Autonomy settings-backed repos (charters/intents/budgets/decisions/reservations/curiosity) | yes | yes | yes | partial | no |
| Approvals (`approval_requests`, `approval_decisions`) | yes | yes | yes | yes | no |
| Events (`events`) | yes | yes | yes | partial | no |
| Jobs + job timeline (`jobs`, `job_status_history`, `job_events`) | yes | yes | yes | partial | no |
| Artifacts (`artifacts`) | yes | yes | yes (DB row + path metadata) | partial (improved) | no |
| Semantic syscall journal + head (`journal_events`, `forge_k_journal_head`) | yes | yes | no live merge; offline whole-store recovery required | yes | no |
| FORGE-K commit proof (`semantic_idempotency_keys`, `forge_k_audit_outbox`) | yes | yes | no live merge; offline whole-store recovery required | yes | no |
| Courthouse state/history (`court_exhibits`, `court_rulings`, `court_appeals`) | yes | yes | no live merge; offline whole-store recovery required | yes | no |
| Cognitive filesystem core tables | yes | yes | yes | yes | no |
| Audit/event execution history (`audit_records`, `gateway_invocations`) | yes | yes (uncapped deterministic export) | yes | yes | no |
| Modelruntime registry/lifecycle (`model_manifests`, `model_registry_status`, `model_runtime_loads`) | yes | yes | yes | yes | no |
| Chat/canvas operator state (`chat_threads`, `chat_messages`, `canvas_boards`, `canvas_notes`) | yes | yes | yes | partial | no |
| Gateway capability overrides (`tool_capability_overrides`) | yes | yes | yes | partial | no |
| Dream reports (`dream_reports`) | yes | yes | yes | correlation/trace + event | no |
| Retrieval/index lineage (`sources`, `files`, `chunks`, retrieval runs/results, memory observations/repair/usefulness) | yes | yes | yes for DB rows; index files/FTS/VSA rebuild remains explicit | partial | no |
| Project-context/evaluation sections | yes | yes | no live apply; deterministic inspection only | partial | no |
| VSA tables | yes | yes | no | partial | no |

## Remaining durability blockers

1. Live restore apply is retired. Full backup exports are inspectable, but no section is applied to a running store. A daemon-stopped whole-store recovery tool must verify the bundle, database, journal chain/head, immutable proof state, workspace identity, and rollback before restore parity can be claimed.
2. End-to-end traceability for artifact creation is improved (chat attachment uploads now audited with correlation/trace/workspace), but still partial across all producers.
3. Fresh-clone reproducibility is still blocked by VSA file commit state (`services/core/internal/memory/vsa_*.go`), though preflight checks now fail fast with actionable guidance.
4. The retired raw apply engine had only DB-scoped atomicity and could not roll back artifact bytes or external effects. It remains reachable solely through an unexported legacy/offline test seam; production has no callsite.
5. Full backup JSON carries a section manifest and per-section checksums. Inspection validates schema, manifest, computed counts/checksums, duplicates, authority, raw SHA-256, and deterministic plan digest without mutation.
6. Dream Mode dry-run reports are now durable as `dream_reports` non-canonical evidence when `persistReport=true`; they remain proposal/report records and do not mutate canonical memory/state.
