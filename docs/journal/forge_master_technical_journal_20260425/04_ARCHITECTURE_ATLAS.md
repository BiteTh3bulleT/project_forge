# Architecture Atlas

## Layer Map

| Layer | Purpose | Status | Evidence |
|---|---|---|---|
| Desktop/operator surface | Chat, jobs, approvals, audit, gateway, models, memory, layouts | PARTIAL | `apps/desktop/src/pages/*` |
| API service | HTTP routes, chat, gateway, modelruntime, autonomy, backup, trace | IMPLEMENTED | `services/core/internal/api` |
| Gateway | Tool execution authority | IMPLEMENTED | `services/core/internal/gateway` |
| Modelruntime | Inference/model management authority | PARTIAL / IMPLEMENTED M3 | `services/core/internal/modelruntime` |
| Control lane | Semantic syscall validation and commit | IMPLEMENTED | `services/core/internal/aios/controllane` |
| Cognitive filesystem | Durable semantic objects | PARTIAL | `store/migrate.go`, controllane repositories |
| Autonomy/Dream | Proposal-first maintenance | PARTIAL | `aios/autonomy`, `aios/dream` |
| Backup/store | SQLite persistence, backup/restore | PARTIAL | `store`, `backup` |

## Process / Service Map

- `forge-core`: Go HTTP service, SQLite owner, control/gateway/modelruntime orchestration.
- `@forge/desktop`: Tauri + React desktop shell.
- `packages/shared`: shared TypeScript contracts, including AI-OS syscall types.

## Repo Map

- `services/core`: authoritative backend.
- `apps/desktop`: operator UI.
- `packages/shared`: shared contracts.
- `packages/ui`: shared UI primitives.
- `docs`: architecture, runbooks, status, reviews, and this journal.
- `scripts`: build/smoke/dev orchestration.
- `nix`: optional Nix foundation, not verified in this pass.

## Implemented vs Conceptual Boundaries

IMPLEMENTED: Gateway, approvals, audit, semantic syscalls, cognitive tables, modelruntime, desktop shell pages.

PARTIAL: Cognitive lane isolation, public syscall facade, full operator trace UI, Dream persistence, restore scoring UX.

CONCEPT: IRIS, GHOST runtime intelligence, ARTEMIS cockpit, Hyperlane, Rule Cell engine.

