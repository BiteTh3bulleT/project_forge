# FORGE Traceability Model (Phase 5.996)

Observed baseline date: 2026-04-21.

Purpose: define practical end-to-end trace requirements across gateway, syscall, autonomy, backup/restore, and legacy boundaries.

## Required identifiers (minimum)

Audit-producing operations should include:
- `workspace_id`
- `correlation_id`
- `trace_id` (when available)
- actor identity (`actor_id`/source)

Path-specific IDs (when applicable):
- `job_id`, `packet_id`
- `intent_id`, `charter_id`, `budget_id`
- `syscall_id`
- `approval_request_id`/`approval_decision_id`
- `gateway_invocation_id`
- `artifact_id`
- `audit_id`

## Canonical chains

1. Gateway tool execution
- request -> policy decision -> invocation record -> audit allow/deny/fail -> optional artifacts

2. Semantic syscall commit
- request -> deterministic validation -> commit -> journal append -> audit record -> audit linkage into persisted objects

3. Autonomy
- intent -> policy/budget/approval decision -> syscall/tool execution -> decision trace + audit

4. Legacy boundaries (explicit)
- adapter invoke legacy route is removed; no legacy adapter-route audit action remains
- v1 memory observation mutation legacy path emits `legacy.memory.observation.*.blocked/used`
- legacy boundary audit payloads now carry `correlationId` and include `traceId` / `workspaceId` when supplied by request context

5. Backup restore
- restore result includes `imported`, `skipped`, `unsupported`, `errors`
- restore result now includes `atomic`, `atomicScope`, `applied`, `rolledBack`, `warnings`
- unsupported sections are explicit and must be treated as non-restored evidence

6. Artifact creation (API-managed uploads)
- chat attachment uploads emit `artifact.uploaded` audit records
- payload carries `artifactId`, `threadId`, `correlationId`, and request-supplied `traceId`; when request `workspaceId` is absent it now falls back to the server workspace-derived id (`workspace:<basename>`)
- gateway `tool.executed` audit payloads now include `artifactCount` and `artifacts` summaries (type/path/summary) for artifact-producing tools, in addition to `correlationId` / `traceId` / `workspaceId`

## Known gaps

1. Dedicated operator UI for full trace graph remains partial, even though API correlation inspection is now consolidated (`GET /api/audit/trace/{correlationId}` returns gateway/audit/artifact/provenance/journal links in one report).
2. Artifact creation trace consistency is improved for chat upload path but still uneven across all producers.
3. VSA export-only sections prevent full trace-graph reconstruction from restore in VSA-heavy histories.
