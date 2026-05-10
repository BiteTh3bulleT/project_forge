# FORGE — Phase 5 Operations (Gateway, Security, Audit, Portability)

Phase 5 hardens the **execution plane**, **auditability**, and **operator portability** of FORGE without turning the product into a feature sprawl.

## Tool execution gateway

- **Single entry**: `POST /api/gateway/invoke` → `internal/gateway` (`Gateway.Execute`).
- **Registration**: tools are small, composable builtins (`fs.read`, `fs.list`, `git.status`, `fs.write`, …) registered in code — not arbitrary shell strings from the UI.
- **Pipeline**: resolve lane → validate tool vs lane write-intent → path scope checks → **permission profile** check → optional **approval / dry-run** short-circuit → persist `gateway_invocations` → execute → audit.

There must be **no alternate execution path** that bypasses the gateway for the same class of action.

## Action lanes

- Table: `action_lanes`
- Each lane declares: `actionType`, allowed/forbidden paths, `writeIntent`, `riskClass`, `requiresApproval`, `expectedArtifacts`, `builtin`, `enabled`.
- API: `GET/POST /api/action-lanes`, `DELETE /api/action-lanes/{id}` (builtins refuse delete at service layer).

## Permission model (execution)

- Table: `permission_profiles` — orthogonal to **routing Policy** (`/api/policy/*`), which remains about adapters/strategies/dossiers.
- Dimensions: read/write/execute path sets, forbidden paths, **allowed tool ids**, approval-required risk list, max write bytes, network flag.
- API: `GET/POST /api/permissions/profiles`, `POST .../activate`, `DELETE .../{id}`.
- UI: **Exec Permissions** page (`/execution-permissions`).

## Audit and trace

- Table: `audit_records` (append-only usage pattern).
- Correlation ids tie gateway invocations, permission changes, backups, and (over time) jobs/approvals.
- API: `GET /api/audit` (filters), `GET /api/audit/trace/{correlationId}` (ordered trace).

## Backup / export / import

- Service: `internal/backup` — writes versioned JSON **bundles** under `${FORGE_DATA_DIR}/backups` and `${FORGE_DATA_DIR}/exports`.
- Kinds: see `backup.KnownKinds` (includes `portable_snapshot`, `full_backup`, domain slices like `dossiers`, `audit_history`, …).
- API: `GET/POST /api/backup/bundles`, `DELETE .../{id}`, `POST /api/backup/restore`.
- Restores are **conservative merges** with explicit `sections` or inferred defaults; always support `dryRun`.
- Restore inputs must be staged under `${FORGE_DATA_DIR}/backups` or `${FORGE_DATA_DIR}/exports`; arbitrary host paths and symlinked bundle paths are rejected before bundle parsing.
- Non-dry-run restore is a critical write operation and now requires a governed approval request. The approval fingerprint binds the exact staged path and section set; approved ids cannot be replayed for a different bundle or restore shape.

## Release readiness

- Service: `internal/release` — `BuildVersion` constant, readiness checklist, `release_artifacts` bookkeeping, `firstRun` summary.
- API: `GET /api/release/readiness`, `GET/POST /api/release/artifacts`, `GET /api/release/first-run`.
- UI: `/release`.

## Deferred / known gaps

- **Approval consumption for gateway**: invocations can return `needs_approval`; wiring those requests into the existing `approval_requests` flow as a first-class consumer is the next integration step (avoid silent auto-approve).
- **Existing databases**: seed profiles only run on empty tables; updating `allowedTools` for already-seeded profiles may require a manual SQL patch or a future migration version table.
- **Gateway ↔ jobs**: jobs remain on their adapter/artifact pipeline; gateway is parallel until explicitly bridged for “tool steps inside jobs.”
