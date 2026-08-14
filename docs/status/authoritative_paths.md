# Authoritative Paths (Phase 5.996)

Date: 2026-04-22
Scope: canonical runtime paths, bounded legacy side doors, and default posture.

Legend:
- `official`: canonical path
- `legacy-boundary`: still present, explicitly guarded
- `partial`: operational but convergence incomplete

| Operation | Official path | Legacy/alternate path | Status | Current guard |
|---|---|---|---|---|
| Tool execution | `/api/gateway/invoke` -> `gateway.Execute` | none | official | legacy adapter invoke route removed; `/api/adapters/{id}/invoke` is not registered |
| Memory semantic mutation (AI-OS) | `forgekernel.Kernel` -> K-owned `DurablePort` -> Control Lane SQLite implementation | `FORGE_KERNEL_AUTHORITY_MODE=legacy_v1` rollback | partial cutover (K20B) | one boot authority; K owns prepare/commit/audit/observe order; atomic object+journal commit; no dual commit |
| Retired memory observation mutation | production FORGE-K semantic syscall path | `/api/memory/observations` POST, `/api/memory/observations/{id}` PATCH, `/api/memory/observations/{id}/usefulness` POST | retired | mutation endpoints return `410 Gone`; retired attempts are audited with correlation/trace/workspace payload context |
| Memory/read inspection | `/api/memory/*` GET read routes | n/a | official | read-only inspection remains enabled |
| Approval gate | `approvals.Service` via gateway/syscall flows | direct approval decision APIs | official | request/decision split preserved |
| Audit write authority | `audit.Service` | none observed | official | gateway/syscall/backup/memory guards emit audit records |
| Autonomy self-initiation | intent -> policy -> budget -> syscall/tool gateway | direct commit bypass not observed | partial | durability gate + approval escalation + propose-only fallback |
| Rule-agent actions | rule runtime -> intent proposals | direct destructive commit | official | propose-only runtime + destructive placeholder guards in runner |
| Backup export/restore inspection | `/api/backup/*` -> `backup.Service`; `/api/backup/restore` is dry inspection only | unexported legacy raw-apply seam used only by offline compatibility tests | partial | all non-dry requests fail with `FORGE_K_RESTORE_APPLY_DISABLED` before approval or mutation; inspection validates path, raw SHA-256, manifest, counts, checksums, duplicates, authority, normalized sections, and deterministic plan digest |
| Desktop mutation | desktop `fetch` -> backend `/api/*` | client-side DB bypass not observed | official | backend validation remains in path |

## Convergence notes

1. Gateway remains the single authoritative tool execution path in normal operation.
2. Legacy adapter invoke execution path is removed; `/api/adapters/{id}/invoke` is no longer routed. Legacy memory observation mutation endpoints are no longer opt-in boundaries; they are retired and return `410 Gone`.
3. Correlation-first trace inspection is now consolidated at `GET /api/audit/trace/{correlationId}` with a single report payload that includes gateway invocations, audit records, artifact records, provenance records, journal events, artifact refs, and explicit link edges (`auditToGateway`, `auditToArtifact`, `provenanceToAudit`, `journalToProvenance`, `artifactRefToProvenance`).
4. Full backup export covers core operational sections plus K20C/K20D Court, journal-head, audit-outbox, and idempotency proof state. None is live-mergeable: the restore endpoint only produces a deterministic inspection plan, and raw apply is retired pending daemon-stopped whole-store recovery.
5. API/chat ingress verification evidence is tracked in `docs/status/tool_execution_ingress_proof.md`.
