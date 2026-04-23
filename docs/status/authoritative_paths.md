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
| Memory semantic mutation (AI-OS) | semantic syscall processor (`controllane.Processor`) | none observed for syscall objects | official | deterministic validation + approval + audit |
| Legacy v1 memory observation mutation | none (legacy only) | `/api/memory/observations` POST, `/api/memory/observations/{id}` PATCH, `/api/memory/observations/{id}/usefulness` POST | legacy-boundary | default-blocked unless `FORGE_ALLOW_LEGACY_MEMORY_MUTATIONS=true`; blocked/used attempts audited with correlation/trace/workspace payload context |
| Memory/read inspection | `/api/memory/*` GET read routes | n/a | official | read-only inspection remains enabled |
| Approval gate | `approvals.Service` via gateway/syscall flows | direct approval decision APIs | official | request/decision split preserved |
| Audit write authority | `audit.Service` | none observed | official | gateway/syscall/backup/memory guards emit audit records |
| Autonomy self-initiation | intent -> policy -> budget -> syscall/tool gateway | direct commit bypass not observed | partial | durability gate + approval escalation + propose-only fallback |
| Rule-agent actions | rule runtime -> intent proposals | direct destructive commit | official | propose-only runtime + destructive placeholder guards in runner |
| Backup export/restore | `/api/backup/*` -> `backup.Service` | manual DB/file copies | partial | restore is DB-transactional across supported sections (`atomic/applied/rolledBack`), explicit `atomicScope=db-supported-sections-only`, explicit `unsupported` + warnings, non-destructive upserts |
| Desktop mutation | desktop `fetch` -> backend `/api/*` | client-side DB bypass not observed | official | backend validation remains in path |

## Convergence notes

1. Gateway remains the single authoritative tool execution path in normal operation.
2. Legacy adapter invoke execution path is removed; `/api/adapters/{id}/invoke` is no longer routed, while legacy v1 memory mutation paths remain explicit opt-in boundaries with richer audit context.
3. Correlation-first trace inspection is now consolidated at `GET /api/audit/trace/{correlationId}` with a single report payload that includes gateway invocations, audit records, artifact records, provenance records, journal events, artifact refs, and explicit link edges (`auditToGateway`, `auditToArtifact`, `provenanceToAudit`, `journalToProvenance`, `artifactRefToProvenance`).
4. Backup restore now covers core operational sections including project context/evaluations/audit/gateway history; VSA-derived sections remain export-only and are reported explicitly as `exportOnly` in restore results.
5. API/chat ingress verification evidence is tracked in `docs/status/tool_execution_ingress_proof.md`.
