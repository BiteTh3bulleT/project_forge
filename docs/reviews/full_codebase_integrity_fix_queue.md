# Full Codebase Integrity Fix Queue

Source audit: `docs/reviews/full_codebase_integrity_audit.md`

Status: prioritized queue. CA1-001 through CA1-007, CA1-009, and CA1-012 through CA1-014 are closed by disabled-by-default host-power policy, root-workspace startup validation, fail-closed raw Compose bind defaults, corrected legacy adapter gateway/lane metadata, OS-login native shell posture, persisted Ollama URL validation, VM/Ollama docs alignment, desktop queue telemetry truth, lane default duplicate-ID coverage, and desktop/Nix doc supersession; remaining rows stay queued until explicitly closed.

| ID | Severity | Category | File path | Summary | Safe for automated fix | Operator decision needed |
| --- | --- | --- | --- | --- | --- | --- |
| CA1-001 | Critical | authority/host mutation | `apps/desktop/src-tauri/src/main.rs`; `apps/desktop/src/layout/AppShellSurfaces.tsx`; `apps/desktop/src/layout/AppShell.tsx` | Closed: shutdown/reboot are allowlisted, disabled by default through `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`, exposed to the UI through `read_host_power_policy`, and disabled in Start until explicitly enabled. | fixed | yes |
| CA1-002 | High | filesystem authority | `services/core/internal/config/config.go`; `services/core/main.go`; `services/core/main_test.go`; `docs/runbooks/config_reference.md` | Closed: default workspace resolves to `${FORGE_DATA_DIR}/workspace`, core startup rejects filesystem root unless `FORGE_ALLOW_ROOT_WORKSPACE=true`, and config docs record the unsafe opt-in. | fixed | yes |
| CA1-003 | High | network exposure | `docker-compose.yml`; `scripts/forge-docker-up.sh`; `scripts/forge-docker-up.ps1`; `services/core/main_test.go`; `docs/runbooks/docker_containerization.md` | Closed: raw Compose defaults to loopback + wildcard opt-in disabled; Docker helper scripts explicitly opt into container-internal wildcard only after generating/passing a local API token. | fixed | yes |
| CA1-004 | High | gateway/modelruntime boundary | `services/core/internal/api/legacy_adapter_gateway_tool.go`; `services/core/internal/api/server_adapters_test.go`; `services/core/internal/lanes/service.go`; `services/core/internal/permissions/service.go` | Closed: `legacy.adapter.invoke` now advertises `scoped_execute` / `L2` / network use at both tool and lane metadata, remains gateway-only, and requires network/profile approval before adapter invocation. | fixed | yes |
| CA1-005 | High | login/auth semantics | `nix/packages/forge-desktop-shell.nix`; `apps/desktop/src/pages/ForgeLoginPage.tsx`; `apps/desktop/src/App.tsx` | Closed: native/operator shell builds use OS login (`bootLogin = false`) and explicit empty-desktop boot reset; optional client-side unlock remains only for builds that opt in. | fixed | yes |
| CA1-006 | High | SSRF/config validation | `services/core/internal/api/server_settings.go`; `services/core/internal/api/settings_test.go` | Closed: persisted `ollamaBaseUrl` is validated before storage; unsafe link-local/remote targets are rejected while loopback/local Docker host targets remain allowed. | fixed | no |
| CA1-007 | High | docs truth alignment | `docs/operations/operator_desktop.md`; `docs/runbooks/forge_operator_desktop_vm.md`; `nix/nixos/configurations/forge-operator-vm.nix` | Closed: docs now match canonical VM config, which enables local NixOS `services.ollama` while leaving model pull/load controls out of the desktop shell. | fixed | no |
| CA1-008 | Medium | API UX/errors | `services/core/internal/api/*` | Plain `http.Error` responses remain across route families. | yes | no |
| CA1-009 | Medium | desktop telemetry truth | `apps/desktop/src/pages/DashboardPage.tsx`; `apps/desktop/src/pages/DashboardPage.test.tsx` | Closed: unavailable model-runtime queue telemetry stays `null` and renders as `unavailable` / `queue unavailable` instead of a synthetic zero. | fixed | no |
| CA1-010 | Medium | duplicate desktop catalog | `apps/desktop/src-tauri/src/main.rs`; `apps/desktop/src/layout/AppShellSurfaces.tsx`; `apps/desktop/src/pages/OperatorAppsPage.tsx` | Operator app catalogs are duplicated and can drift. | partial | no |
| CA1-011 | Medium | desktop route integration | `apps/desktop/src/App.tsx`; `apps/desktop/src/layout/AppShell.tsx` | Deep-link routes can be orphaned in shell window mode. | partial | no |
| CA1-012 | Medium | duplicate policy metadata | `services/core/internal/lanes/service.go`; `services/core/internal/lanes/defaults_test.go` | Closed: default lane IDs are covered by `TestDefaultBuiltinsHaveUniqueIDs`, preventing duplicate built-in metadata such as repeated `fs.mkdir`. | fixed | no |
| CA1-013 | Medium | docs supersession | `docs/status/desktop_shell_status.md`; `docs/status/desktop_nix_packaging_gap.md` | Closed: historical desktop/API and G3.5 package status docs now point to current G8/operator VM sources and identify labwc operator desktop vs Cage rollback/test scope. | fixed | no |
| CA1-014 | Medium | docs truth alignment | `docs/operations/forge_graphical_shell_session.md` | Closed: runbook now states the current `forge-operator`/labwc path, keeps Cage as fullscreen rollback/test coverage, and separates G3.5 package truth from later G8 operator desktop truth. | fixed | no |
| CA1-015 | Medium | docs verification | `docs/runbooks/forge_operator_desktop_vm.md`; `docs/evidence/vm_boot/2026-05-18-section6-final/README.md` | VM verification status appears stale. | yes | no |
| CA1-016 | Low | stale/debug surface | `apps/desktop/src-tauri/src/window_manager.rs` | Debug console window kind exists without obvious owned UI surface. | yes | no |
| CA1-017 | Low | registry drift detection | `apps/desktop/src/layout/toolRegistry.tsx`; `apps/desktop/src/layout/AppShellSurfaces.tsx` | Placeholder surface fallback can hide registry drift. | yes | no |

## Suggested CA2 Slice

Start with items that reduce authority/security risk and are locally testable:

1. CA1-001
2. CA1-002
3. CA1-003
4. CA1-006
5. CA1-012

Then run:

- `npm test`
- `npm run lint`
- `npm run validate:js`
- `npm run validate:desktop`
- `npm run build`
- `cd apps/desktop/src-tauri && cargo test`
