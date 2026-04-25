# Current State

## Repo Topology

GOOD: Main repo areas are clear.

- `services/core`: Go core service, API router, AI-OS control lane, gateway, modelruntime, memory/retrieval, backup, audit, jobs, approvals, config, GPU/provider integrations.
- `apps/desktop`: Tauri + React desktop shell and operator UI.
- `packages/shared`: TypeScript contracts shared by desktop and API surfaces.
- `packages/ui`: shared UI package placeholder/minimal package.
- `docs`: architecture, runbooks, status matrices, operations notes, review artifacts.
- `scripts`: startup/smoke/preflight scripts, currently mixed Bash and Node.
- `nix`: flake, shells, checks, package specs, and future Nix module/capsule placeholders.

## Main Entrypoints

- Core service: `services/core/main.go`, `npm run core`.
- API router: `services/core/internal/api/server.go`.
- Desktop app: `apps/desktop/src/App.tsx`, `apps/desktop/src-tauri/src/main.rs`.
- Root build: `npm run build`.
- Root test: `npm test`.
- Root lint: `npm run lint`.
- Desktop validation: `npm run validate:desktop`.

## Major Modules

GOOD:
- Control lane: `services/core/internal/aios/controllane`.
- Domain contracts: `services/core/internal/aios/domain`.
- Truth engine: `services/core/internal/aios/truth`.
- Compute/librarian: `services/core/internal/aios/compute/librarian`.
- Dream Mode: `services/core/internal/aios/dream`.
- Autonomy/rule agents: `services/core/internal/aios/autonomy`.
- Gateway/tool policy: `services/core/internal/gateway`.
- Model runtime: `services/core/internal/modelruntime` plus `services/core/internal/api/model_runtime*.go`.
- Persistence/migrations: `services/core/internal/store`.
- Backup/restore: `services/core/internal/backup`.
- Retrieval/memory: `services/core/internal/retrieval`, `services/core/internal/memory`.

## Authoritative Seams

GOOD:
- Canonical semantic mutation: `controllane.Processor`.
- Tool execution: `gateway.Gateway.Execute`.
- Inference execution/management: `modelruntime.Service` behind API bridge.
- Approval records: `services/core/internal/approvals`.
- Audit/correlation: `services/core/internal/audit`, API trace report.
- Append-only semantic evidence: `journal_events`.

## Legacy or Duplicate Seams

PARTIAL:
- `events` remains an operational event stream beside canonical `journal_events`.
- Memory observation routes remain readable and mutation endpoints are retired with `410 Gone`.
- Retrieval/observation compatibility tables still hold important evidence but are not in full backup coverage.
- Tri-lane Neural/Arterial/Lymphatic language is mostly architectural doctrine, not hard package/runtime isolation.

