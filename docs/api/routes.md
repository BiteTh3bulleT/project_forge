# API Routes

Generated from `services/core/internal/api/routes.go` and `services/core/internal/api/metrics.go`, with route inventory behavior guarded by `services/core/internal/api/server_route_inventory_test.go`.

Regenerate with `node scripts/generate-api-routes.mjs`.
Check without writing with `node scripts/generate-api-routes.mjs --check`.

## Auth Posture

- Public: mounted without `requireAPIAuth`.
- Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty: mounted under `requireAPIAuth`. An empty token does not grant semantic authority: only verified loopback peers receive `local_loopback` origin proof, while arbitrary remote peers receive no authenticated user/proposer origin and fail closed at FORGE-K authorization.
- Handler-specific checks, approval gates, capability gates, and remote webhook signature/token validation are not expanded here unless visible at the route-mount layer.

## Retired memory mutation behavior

- `POST /api/memory/observations`, `PATCH /api/memory/observations/{id}`, and `POST /api/memory/observations/{id}/usefulness` are terminal audited retirement gates. They return `410 Gone` without decoding the request body or reaching a writer.
- No legacy observation-link mutation route is mounted.
- `POST /api/memory/repair/run` is deterministic proposal inspection only with explicit `dryRun=true`; non-dry requests fail closed.
- Historical observation, link, usefulness, and repair read surfaces remain available. Canonical creation and revision belong to authenticated production FORGE-K semantic syscalls.

## Inventory

| Method | Path | Auth posture | Mount condition |
| --- | --- | --- | --- |
| GET | `/health` | Public | Always mounted |
| GET | `/health/detailed` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/metrics` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | `EnableMetricsEndpoint` |
| GET | `/forge/models` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/import` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/scan` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/models/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/models/{id}/compatibility` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/{id}/verify` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/{id}/enable` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/{id}/disable` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/{id}/archive` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/{id}/remove` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/{id}/delete-file` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/{id}/load` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/{id}/unload` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/forge/models/{id}/chat` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/model-runtime/backends` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/model-runtime/usage` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/model-runtime/health` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/model-runtime/queue` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/model-runtime/loaded` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/kernel/status` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/system/status` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/forge/system/host` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/v1/models` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | `EnableOpenAICompatAPI` |
| POST | `/v1/chat/completions` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | `EnableOpenAICompatAPI` |
| GET | `/api/chat/threads/{id}/assistant-stream` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/meta` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/settings` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| PATCH | `/api/settings` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/settings/ollama-models` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/settings/ollama-models/` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/remote/telegram` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/remote/discord` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/telegram/status` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/discord/status` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/sources` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/sources` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| DELETE | `/api/sources/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/reindex` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/search` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/chunks/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/events` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/providers/capabilities` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/autonomy/status` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/autonomy/intents` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/autonomy/intents/{id}/explain` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/autonomy/decisions` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/autonomy/budgets` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/autonomy/charters` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/autonomy/events` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/autonomy/maintenance/sweep` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/dream/run` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/dream/reports` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/dream/reports/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/dream/reports/{id}/candidates` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/dream/reports/{id}/proposals` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/dream/reports/{id}/warnings` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/adapters` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/jobs/templates` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/jobs` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/jobs` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/jobs/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/jobs/{id}/cancel` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/jobs/{id}/retry` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/jobs/{id}/replay` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/chat/threads` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/chat/threads` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/chat/threads/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| PATCH | `/api/chat/threads/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| DELETE | `/api/chat/threads/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/chat/threads/{id}/messages` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/chat/threads/{id}/attachments` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/chat/threads/{id}/jobs` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/canvas/boards` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/canvas/boards` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/canvas/boards/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| DELETE | `/api/canvas/boards/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/canvas/boards/{id}/notes` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| PATCH | `/api/canvas/boards/{id}/notes/{noteId}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| DELETE | `/api/canvas/boards/{id}/notes/{noteId}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/artifacts` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/artifacts/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/artifacts/{id}/content` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/approvals` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/approvals/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/approvals/{id}/approve` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/approvals/{id}/deny` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/approvals/{id}/cancel` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/packets/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/context-inspector/snapshots` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/context-inspector/snapshots/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/context/restore/recent` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/context/restore/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/context/restore/{id}/candidates` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/context/restore/{id}/score` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/context/restore/{id}/resume-hints` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/context/restore/outcomes` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/context/restore/outcomes/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/context/restore/outcomes/{id}/feedback` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/process/health` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/project-context` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/project-context/import` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/project-context/regenerate` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/embeddings/status` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/embeddings/reembed` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/retrieval/runs` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/retrieval/runs` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/retrieval/runs/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/retrieval/runs/{id}/vsa-signals` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/retrieval/results/{id}/vsa-signal` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/retrieval/results/{id}/usefulness` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/observations` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/memory/observations` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/observations/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/observations/{id}/vsa` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| PATCH | `/api/memory/observations/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/memory/observations/{id}/usefulness` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/retrieval-runs/{id}/selection` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/packets/{id}/alignment` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/dossiers/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/dossiers/{id}/vsa-summary` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/vsa/reindex-runs` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/vsa/reindex-runs/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/vsa/reindex/runs` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/vsa/reindex/runs/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/memory/vsa/reindex/run` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/repair-runs` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/memory/repair-runs/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/memory/repair/run` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/dossiers` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/dossiers` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/dossiers/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| PATCH | `/api/dossiers/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/dossiers/{id}/briefs/generate` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/evaluations` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/evaluations` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/evaluations/metrics` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/lineage/jobs/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/imports/executions` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/imports/executions` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/insights` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/insights/generate` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/dashboard` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/strategies` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/strategies` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/policy/presets` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/policy/presets` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/policy/global-preset` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/policy/global-preset` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/policy/dossiers/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/policy/dossiers/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/policy/recommend` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/policy/recommendations` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/automation/rules` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/automation/rules` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/automation/history` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/automation/run` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/packet-guidance` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/packet-guidance/analyze` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/reconciliation/imports/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/reconciliation/imports/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/reconciliation` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/reviews` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/reviews` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| PATCH | `/api/reviews/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/failure-patterns` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/failure-patterns/analyze` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/gateway/tools` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/gateway/capabilities` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| PATCH | `/api/gateway/capabilities/{id}/status` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/gateway/invoke` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/gateway/invocations` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/action-lanes` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/action-lanes` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| DELETE | `/api/action-lanes/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/permissions/profiles` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/permissions/profiles` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/permissions/profiles/{id}/activate` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| DELETE | `/api/permissions/profiles/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/audit` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/audit/trace` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/audit/trace/{correlationId}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/backup/bundles` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/backup/bundles` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| DELETE | `/api/backup/bundles/{id}` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/backup/restore` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/release/readiness` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/release/artifacts` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/release/artifacts` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| GET | `/api/release/first-run` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
| POST | `/api/commands/execute` | Bearer token when `APIToken` is configured; transport-open when `APIToken` is empty | Always mounted |
