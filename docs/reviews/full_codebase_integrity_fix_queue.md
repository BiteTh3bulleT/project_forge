# Full Codebase Integrity Fix Queue

Source audit: `docs/reviews/full_codebase_integrity_audit.md`

Status: prioritized queue. CA1-001 is closed by disabled-by-default Tauri host-power policy plus Start menu policy reflection; remaining rows stay queued until explicitly closed.

| ID | Severity | Category | File path | Summary | Safe for automated fix | Operator decision needed |
| --- | --- | --- | --- | --- | --- | --- |
| CA1-001 | Critical | authority/host mutation | `apps/desktop/src-tauri/src/main.rs`; `apps/desktop/src/layout/AppShellSurfaces.tsx`; `apps/desktop/src/layout/AppShell.tsx` | Closed: shutdown/reboot are allowlisted, disabled by default through `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`, exposed to the UI through `read_host_power_policy`, and disabled in Start until explicitly enabled. | fixed | yes |
| CA1-002 | High | filesystem authority | `services/core/internal/config/config.go`; `services/core/internal/api/server.go`; `services/core/internal/permissions/service.go` | Default workspace `/` can become broad write authority. | yes | yes |
| CA1-003 | High | network exposure | `docker-compose.yml` | Compose defaults to wildcard bind and wildcard opt-in. | yes | yes |
| CA1-004 | High | gateway/modelruntime boundary | `services/core/internal/api/legacy_adapter_gateway_tool.go`; `services/core/internal/adapters/ollama.go` | Legacy adapter invoke underreports network/model authority and bypasses modelruntime metadata. | partial | yes |
| CA1-005 | High | login/auth semantics | `nix/packages/forge-desktop-shell.nix`; `apps/desktop/src/pages/ForgeLoginPage.tsx`; `apps/desktop/src/App.tsx` | Native/operator shell builds now use OS login (`bootLogin = false`) and explicit empty-desktop boot reset; optional client-side unlock remains only for builds that opt in. | partial | yes |
| CA1-006 | High | SSRF/config validation | `services/core/internal/api/server_settings.go`; `services/core/internal/adapters/ollama.go` | Persisted Ollama base URL is less constrained than query override validation. | yes | no |
| CA1-007 | High | docs truth alignment | `docs/operations/operator_desktop.md`; `docs/runbooks/forge_operator_desktop_vm.md`; `nix/nixos/configurations/forge-operator-vm.nix` | VM/Ollama docs conflict with Nix service enablement. | yes | no |
| CA1-008 | Medium | API UX/errors | `services/core/internal/api/*` | Plain `http.Error` responses remain across route families. | yes | no |
| CA1-009 | Medium | desktop telemetry truth | `apps/desktop/src/pages/DashboardPage.tsx` | Runtime queue unavailable can display as zero. | yes | no |
| CA1-010 | Medium | duplicate desktop catalog | `apps/desktop/src-tauri/src/main.rs`; `apps/desktop/src/layout/AppShellSurfaces.tsx`; `apps/desktop/src/pages/OperatorAppsPage.tsx` | Operator app catalogs are duplicated and can drift. | partial | no |
| CA1-011 | Medium | desktop route integration | `apps/desktop/src/App.tsx`; `apps/desktop/src/layout/AppShell.tsx` | Deep-link routes can be orphaned in shell window mode. | partial | no |
| CA1-012 | Medium | duplicate policy metadata | `services/core/internal/lanes/service.go` | Duplicate `fs.mkdir` built-in lane metadata. | yes | no |
| CA1-013 | Medium | docs supersession | `docs/status/desktop_shell_status.md`; `docs/status/desktop_nix_packaging_gap.md` | Stale status docs need supersession banners. | yes | no |
| CA1-014 | Medium | docs truth alignment | `docs/operations/forge_graphical_shell_session.md` | G3.5 future-language mixes with later G8 truth. | yes | no |
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
