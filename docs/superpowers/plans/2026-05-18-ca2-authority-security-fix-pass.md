# CA2 Authority And Security Fix Pass

## Goal

Resolve the first locally testable authority/security slice from the CA1 full codebase integrity audit without expanding FORGE live authority or bypassing documented approval, journal, gateway, or provenance boundaries.

## Scope

1. CA1-001: Disable direct desktop host shutdown/reboot mutation by default behind explicit operator opt-in.
2. CA1-002: Remove root workspace as the default and reject explicit root workspace unless explicitly enabled.
3. CA1-003: Make Docker Compose core bind defaults loopback and wildcard opt-in disabled.
4. CA1-006: Validate persisted Ollama base URL settings with local-only constraints.
5. CA1-012: Remove duplicate built-in lane metadata for `fs.mkdir` and add drift detection.

## Files

1. `apps/desktop/src-tauri/src/main.rs`
2. `services/core/internal/config/config.go`
3. `services/core/internal/config/config_test.go`
4. `services/core/main.go`
5. `services/core/main_test.go`
6. `docker-compose.yml`
7. `services/core/internal/api/server_settings.go`
8. `services/core/internal/api/settings_test.go`
9. `services/core/internal/lanes/service.go`
10. `services/core/internal/lanes/service_test.go`

## Implementation Steps

1. Add failing tests for each behavioral gate before implementation where a test harness exists.
2. Keep desktop power buttons available, but return a policy-disabled result unless `FORGE_SHELL_DIRECT_SYSTEM_CONTROL=true`.
3. Default `FORGE_WORKSPACE_DIR` to `${FORGE_DATA_DIR}/workspace`; add `FORGE_ALLOW_ROOT_WORKSPACE=false` as the default root override gate.
4. Validate startup configuration with both network bind and workspace authority checks.
5. Change Compose defaults to bind core on `127.0.0.1` and keep wildcard binding opt-in disabled.
6. Add persisted Ollama base URL validation that permits local Ollama endpoints and rejects non-local targets.
7. Extract lane defaults behind a testable helper and assert built-in lane IDs are unique.

## Validation

1. `cd services/core && go test ./internal/config ./internal/api ./internal/lanes .`
2. `cd apps/desktop/src-tauri && cargo test`
3. `npm test`
4. `npm run lint`
5. `npm run validate:js`
6. `npm run validate:desktop`
7. `npm run build`
8. `git status --short --branch`
9. Review `git diff --stat` and `git diff --check` before commit.

## Commit Rule

Commit only after the sanity checks pass or any remaining blocker is explicitly documented. Do not stage unrelated untracked provenance directories such as `FORGE-K_Online_Push/` unless the operator explicitly scopes them into the commit.
