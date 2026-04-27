# Repo Inventory

## Topology

GOOD: The repo has a clear product split:

- `services/core`: Go core service, API server, SQLite persistence, AI-OS/control lane, gateway, approvals, modelruntime, backup, memory/retrieval, autonomy, Dream Mode.
- `apps/desktop`: Tauri + React desktop shell.
- `packages/shared`: shared TypeScript contracts.
- `packages/ui`: small shared UI primitives.
- `docs`: architecture, status, runbooks, reviews, and journal.
- `scripts`: Windows/Node/Bash launch, smoke, VSA preflight, desktop port/deps checks.
- `nix`: optional Nix substrate.

## Main Entrypoints

- Core service: `services/core/main.go`
- API router: `services/core/internal/api/server.go`
- Desktop app: `apps/desktop/src/main.tsx`, `apps/desktop/src/App.tsx`
- Tauri shell: `apps/desktop/src-tauri/src/main.rs`
- Root commands: `package.json`

## Runtime Processes

- `forge-core`: Go HTTP API and kernel/runtime authority.
- Desktop shell: Tauri/Vite React client.
- Optional external/model backends: Ollama, llama.cpp endpoint, OpenAI-compatible endpoint, vLLM-compatible endpoint, TEI, DCGM, Level Zero.

## API Route Groups

- `/health`
- `/api/*`: settings, remote, sources, events, providers, autonomy, Dream, adapters, jobs, chat, canvas, artifacts, approvals, packets, project context, context inspectors, memory, dossiers, gateway, permissions, audit, backup, release, commands.
- `/forge/models*` and `/forge/model-runtime/*`: modelruntime management/inference.
- `/v1/*`: OpenAI-compatible minimum routes.

## Authoritative Seams

- Semantic writes: `services/core/internal/aios/controllane`
- Tool execution: `services/core/internal/gateway`
- Approvals: `services/core/internal/approvals`
- Permissions/capabilities: `services/core/internal/permissions`, `services/core/internal/gateway/tool_capability_registry.go`
- Audit/trace: `services/core/internal/audit`, `services/core/internal/api/trace_report.go`
- Modelruntime: `services/core/internal/modelruntime`
- Backup/export/restore: `services/core/internal/backup`

## Major AI-OS Modules

- Control lane: `aios/controllane`
- Compute/librarian cells: `aios/compute/librarian`
- Truth engine: `aios/truth`
- Autonomy: `aios/autonomy`
- Dream Mode: `aios/dream`
- I/O lane abstractions: `aios/iolane`

## Dream / Restore Modules

- Restore scoring: `aios/controllane/compile_context_restore_scoring.go`
- Snapshot persistence: `aios/controllane/sqlite_store.go`
- Snapshot inspector API: `api/operator_inspector.go`
- Dream service: `aios/dream/service.go`
- Dream API: `api/dream.go`
- Desktop inspector: `apps/desktop/src/pages/InspectorsPage.tsx`

## Known Compatibility / Legacy Seams

- Legacy adapter direct route is retired; `legacy.adapter.invoke` remains gateway-wrapped.
- Legacy memory observation mutation endpoints are gated/retired.
- `events` operational stream coexists with canonical `journal_events`.
- `compute/librarian` runtime and `computelane` interfaces both exist.
- VSA/vector records are retrieval support, not truth authority.

## Suspected Duplicate Systems

PARTIAL: `events` vs `journal_events` still needs consistent language and operator surfacing.

PARTIAL: Some TypeScript contracts for modelruntime, Dream reports, restore inspector, and process health live in `apps/desktop/src/lib/api.ts` instead of `packages/shared`.

