# Phase 16 Storage Cutover Readiness Report

Status: `STORAGE_CUTOVER_READINESS_METADATA / SYSTEM_STATUS_READ_ONLY / SQLITE_DEFAULT_PRESERVED / NO_DUAL_WRITE / NO_READ_SWITCH / NO_FORGE_K_PERSISTENCE_AUTHORITY`

Date: 2026-05-18.

## Scope

Advance Postgres/Qdrant/Redis cutover readiness without changing canonical defaults.

Chosen substage: readiness/blocker cleanup.

## Current Authority

- Live owner: SQLite through `services/core/internal/store.Open` and `${FORGE_DATA_DIR}/forge.sqlite`.
- Target storage owner: future Postgres repository adapters for selected domains after parity, dual-write comparison, read-compare, rollback, and operator approval evidence.
- Target FORGE-K owner: future FORGE-K persistence only after a separate authority migration phase. This phase does not make FORGE-K storage authority.

## Changes

- Added `services/core/internal/storagebackend/cutover_readiness.go`.
  - Produces a pure readiness report.
  - Keeps `canonical_default=sqlite`.
  - Records live owner, target owner, blockers, required tests, passing tests, rollback path, and no-effect flags.
  - Treats Redis and Qdrant as non-authoritative even when configured.
- Added `services/core/internal/storagebackend/cutover_readiness_test.go`.
  - Proves default readiness is blocked.
  - Proves SQLite remains canonical default.
  - Proves Redis/Qdrant are not truth authority.
  - Proves full evidence can mark a selected Postgres domain proposal-ready without switching reads or defaults.
- Updated `services/core/internal/api/system_status.go`.
  - Adds `storage.cutover_readiness` to the existing read-only system status payload.
  - Falls back to SQLite readiness metadata if storage backend config is invalid.
- Updated `services/core/internal/api/system_status_test.go`.
  - Proves the system status route exposes blocked cutover readiness read-only and preserves SQLite defaults.
- Updated storage architecture, parity testing, current authority, current phase status, and this status/report pair.

## Validation

Passed:

- `cd services/core && go test ./internal/storagebackend -run CutoverReadiness -count=1`
- `cd services/core && go test ./internal/api -run TestForgeSystemStatusReadOnlySurface -count=1`
- `cd services/core && go test ./internal/storagebackend ./internal/store ./internal/config ./internal/forgekshadow ./internal/vectorstore ./internal/ephemeral -count=1`
- `cd services/core && go test ./internal/api -run "ForgeSystemStatus|RouteInventory" -count=1`
- `rg -n "forgek/(kernel|court|palace|snapshots|contextcompiler|kv|runtime|lymphatic|consensus)|services/core/internal/forgek" services/core/internal/storagebackend services/core/internal/api/system_status.go -g "*.go"` returned no matches.
- `npm run docs:routes:check`
- `git diff --check` passed with line-ending warnings only.
- `npm run lint`
- `npm test`
- `npm run validate:forgek`
- `npm run build:core`

## Not Done

- No Postgres canonical repository adapter is activated.
- No dual-write or read-compare is enabled.
- No read switch is enabled.
- No Redis live queue/cache authority is added.
- No Qdrant live retrieval authority is added.
- No FORGE-K simulator storage or persistence service is imported into the live daemon.
