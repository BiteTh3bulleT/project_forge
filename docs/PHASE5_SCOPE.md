# FORGE — Phase 5 Scope

## In scope (delivered)

- **Tool execution gateway**: `internal/gateway` + `POST /api/gateway/invoke` + tool catalog + invocation history.
- **Action lanes**: persisted lanes with explicit scope, risk, write intent, approval requirements, expected artifacts.
- **Execution permission model**: profiles with read/write/execute path sets, forbidden paths, tool allowlist, risk→approval mapping, byte limits, network flag.
- **Audit / trace**: `audit_records` with filters + correlation trace endpoint.
- **Import/export/backup**: JSON bundles with kinds, SHA-256 inventory, restore with `dryRun` and section selection defaults.
- **Packaging / release readiness**: checklist API, first-run summary, release artifact log API.
- **UI**: Tool Gateway, Action Lanes, Exec Permissions, Audit, Backup/Export, Release pages wired to APIs.
- **Documentation**: `PHASE5_OPERATIONS.md`, `PACKAGING.md`, `IRIS_TRANSFER.md`, architecture/README updates, `forge_context.txt` pointer.

## Explicitly deferred

- End-to-end **approval ticket consumption** for every `needs_approval` gateway outcome (record exists; operator workflow completion is the next wiring step).
- Automatic **purge** of orphaned `files` rows when sources disappear from disk (separate hygiene phase).
- **Remote** audit sinks / signed logs (local-first remains baseline).

## Acceptance checklist

- [x] All tool execution for gateway builtins routes through `Gateway.Execute`
- [x] Permission checks are centralized in `permissions.Service.Check`
- [x] Risky operations can be blocked or forced into `needs_approval` / `dry_run` paths without silent execution
- [x] Operator can export bundles, restore with dry-run, and inspect readiness
- [x] UI exposes gateway, lanes, permissions summary, audit filters, backup actions, release checklist
