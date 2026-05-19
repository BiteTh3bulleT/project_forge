# PHASE CA1 Full Codebase Integrity Audit

Status: `AUDIT_ONLY / NO_RUNTIME_BEHAVIOR_CHANGE / NO_AUTHORITY_EXPANSION / NO_HOST_MUTATION / NO_FORGE_K_AUTHORITY_MIGRATION`

Date: 2026-05-18

## Executive Summary

CA1 completed a full integrity audit pass across the active FORGE repo, desktop shell, Go core, Nix/VM packaging, documentation/status sources, and untracked prompt-pack/provenance folders visible in the workspace.

No runtime code was changed. Validation is broadly green for npm, Go, desktop Vitest/typecheck/build, FORGE-K fixture parity, and Tauri Rust tests. Nix commands were attempted but blocked in this PowerShell environment because `nix` is not installed/on PATH.

The most serious integrity issue is authority drift: the desktop Start menu exposes shutdown/reboot controls that call Tauri host power commands directly, while current shell documentation and status surfaces still say direct host mutation and direct system control are disabled. Additional high-risk findings include broad default workspace authority when `FORGE_WORKSPACE_DIR` is unset, Docker Compose wildcard binding defaults, a legacy adapter gateway tool that can reach Ollama outside modelruntime governance metadata, and hardcoded client-side login credentials in the packaged shell.

## Audit Scope

Mandatory documents read or sampled:

- `README.md`
- `AGENTS.md`
- `CODEX.md`
- `docs/onboarding.md`
- `docs/reviews/current_phase_status.md`
- `docs/status/current_authority_sources.md`
- `docs/status/implementation_matrix.md`
- `docs/status/placeholders_and_stubs.md`
- `docs/status/duplicate_systems.md`
- `docs/status/test_gap_analysis.md`
- `docs/status/current_baseline_gate.md`
- `docs/DESKTOP_SHELL.md`
- `docs/operations/forge_graphical_shell_session.md`
- `package.json`
- `apps/desktop/package.json`
- `services/core/go.mod`
- `apps/desktop/src-tauri/Cargo.toml`
- `flake.nix`

Inventory snapshot from `rg --files`:

| Area | File count |
| --- | ---: |
| `apps` | 284 |
| `services` | 739 |
| `packages` | 5 |
| `scripts` | 36 |
| `docs` | 604 |
| `nix` | 41 |
| `FORGE-K_Online_Push` | 101 |
| root/other | 79 |
| total | 1889 |

Active areas: `apps/desktop`, `services/core`, `packages/shared`, `packages/ui`, `scripts`, `nix`, current `docs/status`, current `docs/reviews`, current runbooks, root package manifests, `flake.nix`.

Simulator/test-only areas: `services/core/internal/forgek`, `crates/forgek-validate`, `fixtures/forgek`, tests under `*_test.go` and `*.test.ts(x)`.

Historical/provenance areas: archived phase prompts, prompt packs, old reports/status docs, `FORGE-K_Online_Push`.

Generated/build areas: desktop `dist`, Rust `target`, TypeScript build info, Go/Rust caches. Build cache changes were not committed.

## Commands Run

| Command | Result | Notes |
| --- | --- | --- |
| `npm test` | pass | Go core test suite via root script. |
| `npm run lint` | pass | Go vet via root script. |
| `npm run validate:js` | pass | Desktop typecheck plus 52 Vitest files / 138 tests. |
| `npm run validate:desktop` | pass | Desktop typecheck, tests, Vite build. |
| `npm run validate:local` | pass | Optional integration env preflight reported missing DSNs as allowed; desktop, FORGE-K, core build passed. |
| `npm run build` | pass | Desktop build plus Go core build. |
| `npm run build:core` | pass | VSA preflight plus `go build ./...`. |
| `npm run build:desktop` | pass | Vite desktop build. |
| `cd services/core && go test ./...` | pass after rerun | One parallel run timed out while other validation jobs ran; clean rerun passed. |
| `cd services/core && go vet ./...` | pass | Focused Go vet. |
| `npm -w @forge/desktop run test` | pass | 52 test files / 138 tests. |
| `npm -w @forge/desktop run typecheck` | pass | Desktop TS projects. |
| `npm -w @forge/desktop run build` | pass | Vite build. |
| `cd apps/desktop/src-tauri && cargo test` | pass | 23 Tauri/Rust tests. |
| `nix --version` | blocked | `nix` command not recognized in this PowerShell environment. |
| `nix flake check` | blocked | Same environment blocker. |
| `nix build .#forge-core` | blocked | Same environment blocker. |
| `nix build .#forge-desktop-shell` | blocked | Same environment blocker. |

## Commands Failed Or Skipped

- `npm install`: skipped because `node_modules` is present.
- First focused `go test ./...`: timed out during concurrent validation; rerun passed and is the authoritative result.
- Nix commands: blocked because `nix` is unavailable on PATH in this shell.

## Build/Test Status

Build/test status is strong for current non-Nix workflows. The Nix/VM packaging path cannot be certified from this Windows PowerShell shell until Nix is available or checks are rerun inside a Nix-enabled environment/VM.

## Critical Findings

### CA1-001 Direct Desktop Host Power Controls Bypass Documented Authority

Severity: Critical

Evidence:

- `apps/desktop/src/layout/AppShellSurfaces.tsx:677` exposes Reboot.
- `apps/desktop/src/layout/AppShellSurfaces.tsx:685` exposes Shutdown.
- `apps/desktop/src/layout/AppShell.tsx:794` confirms through `window.confirm`.
- `apps/desktop/src-tauri/src/main.rs:450` starts host power command execution.
- `apps/desktop/src-tauri/src/main.rs:467` and `apps/desktop/src-tauri/src/main.rs:487` use `systemctl`.
- `docs/operations/forge_graphical_shell_session.md:90` says `FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false`.
- `docs/operations/forge_graphical_shell_session.md:477` says no direct host mutation from shell controls.

Impact: The shell now performs direct host mutation despite current authority docs and PhaseCA1 boundaries saying the shell must not expose reboot/shutdown/systemctl controls. This is a live host mutation path outside gateway/approval/audit policy.

Recommended fix: Remove or disable direct power controls, or route them through an explicit governed capability with approval, audit, environment gating, and documentation updates.

## High Findings

### CA1-002 Default Workspace Can Become Broad Root Write Authority

Evidence: `services/core/internal/config/config.go:132`, `services/core/internal/config/config.go:134`, `services/core/internal/api/server.go:156`, `services/core/internal/api/server.go:160`, `services/core/internal/permissions/service.go:118`.

Impact: If `FORGE_WORKSPACE_DIR` is unset, workspace defaults to `/`. Startup then activates `workspace-write` when workspace is root, and policy seeding derives read/write roots from that workspace. Forbidden path lists reduce but do not eliminate broad host write risk.

Recommended fix: Fail startup when workspace resolves to `/` unless an explicit unsafe development opt-in is set; never auto-activate write policy for `/`.

### CA1-003 Docker Compose Still Defaults To Wildcard Bind

Evidence: `docker-compose.yml:11` defaults `FORGE_CORE_BIND_HOST` to `0.0.0.0`; `docker-compose.yml:12` defaults `FORGE_ALLOW_WILDCARD_BIND` to `true`.

Impact: Container bring-up defaults to network exposure. Core code now requires wildcard opt-in and auth, but Compose still supplies the opt-in by default, weakening the safe default posture.

Recommended fix: Default Compose to loopback or require operators to explicitly set wildcard bind and token values.

### CA1-004 Legacy Adapter Gateway Tool Underreports Network/Model Authority

Evidence: `services/core/internal/api/legacy_adapter_gateway_tool.go:21`, `services/core/internal/api/legacy_adapter_gateway_tool.go:53`, `services/core/internal/adapters/ollama.go:81`, `services/core/internal/adapters/ollama.go:216`.

Impact: `legacy.adapter.invoke` is classified as low-risk/non-network gateway metadata but can call Ollama over HTTP and produce model output outside modelruntime scheduling/accounting semantics.

Recommended fix: Retire this compatibility tool or split it into accurately classified capabilities; route model generation through governed modelruntime.

### CA1-005 Packaged Shell Login Uses Hardcoded Client-Side Credentials

Evidence: `nix/packages/forge-desktop-shell.nix:46`, `nix/packages/forge-desktop-shell.nix:48`, `apps/desktop/src/pages/ForgeLoginPage.tsx:7`, `apps/desktop/src/App.tsx:276`.

Impact: The shell presents a login-like gate with static Nix-provided credentials and client-side `sessionStorage` trust. This can mislead operators into treating the shell unlock screen as an OS/auth boundary.

Recommended fix: Treat display-manager/OS auth as the real login, derive per-install credentials through a server-backed challenge, or relabel this as a local splash/unlock screen.

### CA1-006 Persisted Ollama Base URL Is Less Constrained Than Query Override

Evidence: `services/core/internal/api/server_settings.go:162`, `services/core/internal/api/server_settings.go:546`, `services/core/internal/adapters/ollama.go:39`, `services/core/internal/adapters/ollama.go:121`.

Impact: Settings can persist an arbitrary `ollamaBaseUrl` used later by adapter/chat paths. Query-time model-list overrides are validated more tightly than persisted settings, creating authenticated SSRF/local probing risk.

Recommended fix: Apply the same URL validation to persisted `ollamaBaseUrl`; prefer loopback-only unless explicitly configured otherwise.

### CA1-007 Current VM/Ollama Docs Conflict With Nix VM Configuration

Evidence: `docs/operations/operator_desktop.md:84`, `docs/operations/operator_desktop.md:86`, `docs/runbooks/forge_operator_desktop_vm.md:84`, `nix/nixos/configurations/forge-operator-vm.nix:122`, `nix/nixos/configurations/forge-operator-vm.nix:123`.

Impact: Operator docs say the canonical VM does not start Ollama and service support is future, while the Nix VM config enables `services.ollama` by default.

Recommended fix: Update docs to say the canonical VM enables local Ollama service by default while the shell still does not pull/load models or expose model lifecycle controls.

## Medium Findings

### CA1-008 Plain `http.Error` Responses Remain Across API Routes

Evidence: `rg -n "http\\.Error\\(" services/core/internal/api --glob '!**/*_test.go'` returned remaining occurrences across `autonomy_api.go`, `chat_post.go`, `dream.go`, `operator_inspector.go`, `phase3.go`, `phase4.go`, `phase5.go`, `phase_memory.go`, `remote.go`, `restore_outcomes.go`, `server_sources.go`, and `workspace.go`.

Impact: The API still mixes structured errors with plain Go/string responses, hurting UI consistency and making error handling/audit correlation harder.

Recommended fix: Migrate route families to `writeAPIError` or equivalent structured errors in small slices.

### CA1-009 Runtime Queue Can Display Zero When Runtime Is Unavailable

Evidence: `apps/desktop/src/pages/DashboardPage.tsx:101`, `apps/desktop/src/pages/DashboardPage.tsx:254`, `apps/desktop/src/pages/DashboardPage.tsx:435`, `apps/desktop/src/pages/DashboardPage.tsx:783`.

Impact: Missing modelruntime telemetry can look like a healthy empty queue.

Recommended fix: Render unavailable/unknown state distinctly from depth `0`.

### CA1-010 Operator App Catalogs Are Duplicated

Evidence: `apps/desktop/src-tauri/src/main.rs:94`, `apps/desktop/src/layout/AppShellSurfaces.tsx:29`, `apps/desktop/src/pages/OperatorAppsPage.tsx:13`.

Impact: Backend catalog, shell fallback catalog, and page fallback catalog can drift in labels/native flags/executables.

Recommended fix: Use the backend catalog as sole authority or generate TypeScript fallback data from the Rust catalog.

### CA1-011 Deep-Link Detail Routes Can Be Orphaned In Shell Window Mode

Evidence: `apps/desktop/src/App.tsx:212`, `apps/desktop/src/App.tsx:225`, `apps/desktop/src/layout/AppShell.tsx:428`, `apps/desktop/src/layout/AppShell.tsx:1108`.

Impact: Detail routes such as jobs and memory chunks can be routed but not hosted when shell windows exist.

Recommended fix: Register detail surfaces as shell-capable windows or always provide a foreground routed detail host.

### CA1-012 Duplicate Built-In Lane ID

Evidence: `services/core/internal/lanes/service.go:75`, `services/core/internal/lanes/service.go:168`, `services/core/internal/lanes/service.go:583`.

Impact: `fs.mkdir` is defined twice with different metadata and the later upsert wins, reducing policy review clarity.

Recommended fix: Keep one default lane definition and add a uniqueness test for built-in lane IDs.

### CA1-013 Stale Desktop/Nix Status Docs Need Supersession Notes

Evidence: `docs/status/desktop_shell_status.md:1`, `docs/status/desktop_nix_packaging_gap.md:50`, `docs/status/desktop_nix_packaging_gap.md:63`, `docs/status/current_authority_sources.md:26`.

Impact: Older status docs live in `docs/status` and can be mistaken for current truth.

Recommended fix: Add top-of-file supersession banners pointing to current G8 and Nix authority sources.

### CA1-014 Graphical Shell Operation Doc Mixes G3.5 Future Limits With Later G8 Truth

Evidence: `docs/operations/forge_graphical_shell_session.md:130`, `docs/operations/forge_graphical_shell_session.md:15`, `docs/runbooks/forge_operator_desktop_vm.md:42`.

Impact: The doc can make implemented G4/G6/G8 session work look future-only when discussing a G3.5 package boundary.

Recommended fix: Narrow the future-language to G3.5 package-only context and link later G8/native desktop docs.

### CA1-015 Canonical VM Verification Status Is Stale

Evidence: `docs/runbooks/forge_operator_desktop_vm.md:5`, `docs/runbooks/forge_operator_desktop_vm.md:7`, `docs/evidence/vm_boot/2026-05-18-section6-final/README.md:5`, `docs/evidence/vm_boot/2026-05-18-section6-final/README.md:41`.

Impact: Runbook underclaims canonical VM verification and can cause duplicate or misdirected verification.

Recommended fix: Update header to cite the 2026-05-18 QEMU/NixOS evidence if that evidence remains current authority.

## Low Findings

### CA1-016 Debug Console Window Kind Is Present Without An Obvious Product Surface

Evidence: `apps/desktop/src-tauri/src/window_manager.rs:32`, `apps/desktop/src-tauri/src/window_manager.rs:357`.

Impact: Stale/debug-only route can linger as an unreviewed surface.

Recommended fix: Remove until needed or register/test it explicitly.

### CA1-017 Placeholder Surface Fallback Can Hide Registry Drift

Evidence: `apps/desktop/src/layout/toolRegistry.tsx:19`, `apps/desktop/src/layout/AppShellSurfaces.tsx:399`.

Impact: The fallback is safe but can hide launchable tools without mounted components.

Recommended fix: Add a shell config test asserting every launchable tool has a component or intentional detail-route handling.

## Duplicate Systems

| Area | Classification | Notes |
| --- | --- | --- |
| Gateway/tool execution | mostly converged, one compatibility risk | Gateway remains sole routed execution ingress; `legacy.adapter.invoke` remains a compatibility tool with inaccurate model/network metadata. |
| Memory mutation | converged for direct route retirement | Legacy memory mutation routes are gated `410 Gone`; current desktop memory note tests still exercise the observation API path and should be reviewed during the next fix pass for naming/semantics. |
| Operator app catalog | real duplicate requiring cleanup | Rust catalog plus two TS fallback catalogs. |
| Lane definitions | real duplicate requiring cleanup | Duplicate `fs.mkdir` built-in metadata. |
| Desktop route/window registry | partial duplicate requiring cleanup | App routes and shell window registry do not fully cover detail routes. |
| Docs/status truth | stale/conflicting duplicates | Current authority docs exist, but old status/runbook claims need supersession notes. |
| Nix module/profile docs | stale placeholder overlap | `nix/modules/README.md` still says no compositor while current profiles intentionally include compositor substrate. |

## Placeholder / Mock / Fake Runtime Findings

Acceptable test/simulator-only mocks:

- `services/core/internal/forgek/runtime/mock_driver.go` and `services/core/internal/forgek/runtime/statuses.go` are simulator-only.
- `services/core/internal/modelruntime/backend_fake_test.go` and fake model runtime helpers are test-only.
- Nix check `fake-*` binaries are package checks/fixtures.

Runtime or operator-facing placeholders needing follow-up:

- `services/core/internal/modelruntime/backend_fake.go` provides a fake backend implementation outside tests. It appears guarded by production refusal in `services/core/internal/api/model_runtime_bridge.go`, but should remain explicitly disabled outside tests.
- `services/core/internal/aios/compute/librarian/cells.go:111` declares a deterministic stub intake cell. Current posture is scaffold/proposal-only, but it should not be promoted as production semantic inference.
- `services/core/internal/backup/restore_sections.go:158` returns `restore mapping not implemented`; current docs acknowledge export/restore asymmetry. Keep visible until parity is complete.
- `apps/desktop/src/pages/SystemPage.tsx` includes honest `not wired`/unavailable labels for several surfaces. These are acceptable when visibly non-authoritative.

## Truncation / Corruption Findings

No merge conflict markers or obvious truncation/corruption markers were found by the scan for:

- `<<<<<<<`, `=======`, `>>>>>>>`
- `truncated`, `omitted`, `rest of file`, `continue here`
- `TODO: finish`
- placeholder ellipses in executable code patterns

No unclosed JSON/TOML/Nix syntax failure was detected by the validation commands that ran. Nix syntax/build confidence remains blocked by the missing `nix` executable.

## Dead Code / Orphan Findings

- Potential orphan/debug surface: debug console window kind and route.
- Potential orphan route handling: job detail and memory chunk deep-link routes in shell window mode.
- Historical prompt packs and old status docs are retained as provenance; do not delete without a dedicated archival/supersession pass.

## Frontend/Desktop Findings

Primary risks:

1. Direct host power controls.
2. Hardcoded shell login credentials/client-side unlock.
3. Runtime queue unavailable displayed as zero.
4. Duplicated operator app catalogs.
5. Deep-link routes not fully integrated into shell window model.

Desktop tests/build pass, but these are design/authority/integrity findings rather than syntax failures.

## Core Service Findings

Primary risks:

1. Root workspace default and policy activation.
2. Legacy adapter gateway tool metadata/modelruntime boundary.
3. Persisted Ollama URL validation mismatch.
4. Remaining plain `http.Error` route responses.
5. Duplicate `fs.mkdir` built-in lane metadata.

Go tests/vet/build pass.

## Nix/VM/Packaging Findings

Primary risks:

1. Compose wildcard bind defaults remain permissive.
2. Nix VM enables Ollama service while docs say it does not.
3. Shell package carries static login defaults.
4. Several Nix/session docs need supersession alignment.
5. Nix validation could not be run in this shell.

## Docs Truth Alignment Findings

Current authority docs (`README.md`, `AGENTS.md`, `docs/onboarding.md`, `docs/status/current_authority_sources.md`, `docs/reviews/current_phase_status.md`) consistently preserve the FORGE-K simulator/live boundary.

Stale/conflicting areas:

- operator desktop/Ollama service behavior
- canonical VM verification header
- G3.5 package-boundary future language vs later G8 session truth
- old desktop shell and desktop Nix packaging status files lacking explicit supersession banners
- `nix/modules/README.md` placeholder language vs current opt-in compositor profiles

## Security/Authority Boundary Findings

Critical/high boundary issues:

- Direct shell host power controls.
- Broad root workspace default with write policy activation.
- Docker Compose wildcard bind opt-in supplied by default.
- Legacy adapter tool can invoke model/network behavior outside accurate capability metadata.
- Persisted Ollama URL validation gap.
- Hardcoded shell login gate can misrepresent auth strength.

FORGE-K simulator/live boundary appears preserved in live code: observed live integrations are through Control Lane validation/shadow metadata and not direct simulator authority.

## Recommended Fix Order

1. Disable or govern desktop shutdown/reboot controls.
2. Fail closed on root/default workspace authority.
3. Remove Compose wildcard bind defaults.
4. Retire or accurately gate `legacy.adapter.invoke`.
5. Fix persisted `ollamaBaseUrl` validation.
6. Replace static shell login credentials/client-only unlock semantics.
7. Convert remaining plain API errors to structured errors by route family.
8. Fix desktop runtime-unavailable queue display.
9. Consolidate operator app catalog ownership.
10. Add supersession notes for stale desktop/Nix docs and align Ollama VM docs.
11. Resolve duplicate `fs.mkdir` lane metadata.
12. Integrate shell detail routes or remove dead route assumptions.

## Operator Decisions Needed

- Decide whether Start menu shutdown/reboot should exist at all. If yes, decide the required gateway capability, approval policy, audit shape, and disabled-by-default environment gate.
- Decide whether the Forge login screen is a real authentication boundary or a visual splash/unlock after OS login.
- Decide whether Docker Compose should ever default to external exposure in local development.
- Decide whether `legacy.adapter.invoke` remains as compatibility or is retired in favor of modelruntime-only generation.
- Decide whether `FORGE-K_Online_Push` should remain untracked provenance, be archived under `docs/prompt-packs`, or be ignored.

## Final Confidence Rating

Confidence: Medium-high for non-Nix code integrity and authority findings; medium for Nix/VM packaging because Nix validation was environment-blocked.

No runtime behavior changes were made in CA1.
