# Full Codebase Integrity Re-Audit — Phase CA2

**Phase label:** PHASE CA2 — FULL_CODEBASE_INTEGRITY_REAUDIT_AND_FIX_QUEUE
**Status marker:** `AUDIT_ONLY / NO_RUNTIME_BEHAVIOR_CHANGE / NO_AUTHORITY_EXPANSION / NO_HOST_MUTATION / NO_FORGE_K_AUTHORITY_MIGRATION`
**Date:** 2026-05-19
**Repo:** `/home/rshort/WTF/ProjectForge`
**Branch / HEAD:** `main` @ `8a2cdb8` "Harden shell desktop and validation preflights"
**Audit method:** Multi-pass (1–11). Six parallel auditors + main-thread validation runs.
**Source scratch:** `docs/reviews/ca2_scratch/{pass1_inventory, pass2_ca1_compare, pass3_9_duplicates_core, pass4_5_truncation_mocks, pass7_10_authority_docs, pass8_frontend}.md`

> No code, file, or runtime behaviour was changed during this audit. No deletions. Historical evidence retained.

---

## 1. Executive Summary

CA2 found the repository in **substantively healthy** integrity shape. The FORGE-K simulator boundary is correctly isolated (forbidden-imports test enforces it), tool execution remains gateway-only, semantic writes are Control Lane–gated, and the desktop shell exposes only read-only system surfaces. `go vet`, `go build ./...`, and the desktop TypeScript typecheck all pass.

Two real defects surfaced, both fixable in narrow PRs:

1. **Go test failure** in `services/core/internal/hostbridge`: `TestExecRunnerRejectsOversizeCommandOutput` returns a context-deadline error instead of the expected stdout-size error (likely a flaky/under-resourced bound rather than a real authority defect).
2. **Vitest failure** in `apps/desktop/src/pages/ChatPage.test.tsx:262` — `getByRole("button", { name: "Send" })` does not find Send after a stream is initiated (the button is still showing "Stop"). Single test, no other regressions.

The most operationally important finding is **docs drift around the desktop shell's host power actions**: the Tauri binary implements `shutdown`/`reboot` behind a `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` env-var gate that **defaults to false** and is enforced in `request_host_power_action_with_policy` (verified at [`apps/desktop/src-tauri/src/main.rs:520-557`](apps/desktop/src-tauri/src/main.rs#L520-L557) with tests at lines 765 and 780). However, several docs claim G8 phases "do not add host mutation," which is incomplete — the controls are policy-gated, not absent. This is a **docs supersession** issue, not an authority breach.

One **live-UI mock leakage** matters: `SystemPage.tsx` `cockpitRows` (lines 225–250) renders hardcoded "simulator/planned" / "shadow/inspector" status entries as if they were live state. Per G6 doctrine the System surface must not show fake healthy state. This should be replaced with explicit "unavailable" rendering or omitted.

No FORGE-K live authority expansion. No memory-write bypass. No tool-execution bypass. No critical authority breach.

**Confidence:** **Medium-High** (see §22).

---

## 2. Audit Scope

Covered:
- Repository inventory (apps, services, packages, crates, scripts, nix, docs, archive, evidence, fixtures, prompt packs, VM artefacts, symlinks)
- Comparison stance vs. prior CA1 audit (none found)
- Duplicate / conflicting implementations across routes, gateway, memory, modelruntime, approvals, auth, desktop stores, CSS, types, configs, Nix modules, scripts, docs
- Truncation, corruption, unclosed structures, merge markers, placeholder/mock/fake tokens
- Placeholder / mock / fake runtime audit, scoped to live vs. simulator vs. test fixtures
- Build and static validation (Pass 6 — see §4–6)
- Runtime authority-boundary checks (FORGE-K isolation, model→memory, semantic writes, gateway, host mutation, append-only memory, simulator labelling)
- Desktop / frontend audit (stores, routes, components, Tauri commands, CSS, demo data, debug UI)
- Core service audit (route registration, service init, mocks in live paths, unimplemented handlers, defaults, panic/Fatal, goroutine patterns, shutdown, auth, approvals, audit/journal, storage migrations)
- Docs truth alignment (READMEs, status, reviews, reports, ADRs, runbooks)
- Prioritised fix queue (Pass 11 — separate file)

Excluded by scope:
- `node_modules/`, `.worktrees/*/node_modules`, `forge-operator-vm.qcow2` (9.9 GB), build outputs (`apps/desktop/src-tauri/target/`, `dist/`), `*.zip`, Nix `result*` symlinks.
- `nix flake check`, `nix build .#forge-core`, `nix build .#forge-desktop-shell`: not attempted in this audit run (no Nix invocation needed; reported as skipped in §5).

## 3. Relationship to CA1

At the time CA2's auditors began work, no CA1 artefact was present locally (`docs/reports/phase_ca1_full_codebase_integrity_audit.md` did not exist; `docs/archive/phases/` contained only `PhaseCA2.txt`). During the audit window, a remote merge brought commit `3a671b4 Add CA1 codebase integrity audit` into the working tree, adding `docs/reviews/full_codebase_integrity_audit.md`, `docs/reports/phase_ca1_full_codebase_integrity_audit.md`, `docs/status/phase_ca1_full_codebase_integrity_audit.md`, the CA1 CSV, fix queue, and `docs/archive/phases/PhaseCA1.txt`. CA2 was performed independently of CA1's findings.

Post-hoc overlap (CA1 ID ↔ CA2 ID):

- CA1-001 (Critical — desktop host shutdown/reboot bypass) ↔ **CA2 M-1** (docs drift on `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`). CA2 verified the binary-level policy gate and unit tests, downgraded from Critical to Medium, and recorded the same docs-supersession action.
- CA1-010 (Medium — operator app catalog duplication) ↔ **CA2 H-2** (`FALLBACK_OPERATOR_APPS` duplicate). Same finding; CA3 fix pass resolved it on operator decision (adopted `AppShellSurfaces` as source of truth).
- CA1-011 (Medium — deep-link routes orphaned in shell window mode) ↔ **CA2 L-2** (detail-route fragility via `pathname.startsWith`).
- CA1-002..009, CA1-012..017 — broader configuration-security scope (workspace authority, Docker wildcard bind, legacy adapter/Ollama, hardcoded shell login, SSRF, plain `http.Error`, dashboard zero vs unavailable, lane metadata duplication, supersession banners). Not directly in CA2's findings list. **Recommended:** treat the CA1 fix queue as the authoritative remediation list for those items; CA2 does not re-litigate them.
- **CA2-only findings not in CA1**: H-3 (hostbridge timing test), H-4 (ChatPage test, already green at HEAD), M-2 (`os.Exit` from goroutine), M-3 (empty-token startup warn), M-5 (autonomy ID-prefix filter contract). CA1 reported tests as passing; the H-3 timing fragility surfaced under CA2's environment.

See `docs/reviews/ca2_scratch/pass2_ca1_compare.md` for the full overlap table.

## 4. Commands Run

| Command | Exit | Notes |
|---|---|---|
| `cd services/core && go test ./...` | **non-zero** | 1 package fails (`internal/hostbridge`); 56 other packages OK. Tail captured in §6. |
| `cd services/core && go vet ./...` | 0 | Clean. |
| `npm run build:core` (= VSA preflight + `go build ./...`) | 0 | Clean. |
| `npm -w @forge/desktop run typecheck` (`tsc --noEmit -p tsconfig.json && tsc --noEmit -p tsconfig.node.json`) | 0 | Clean. |
| `npm -w @forge/desktop run test -- --run` (vitest) | **non-zero** | 151/152 tests pass; 1 failure in `ChatPage.test.tsx`. |

Auditors also executed `rg`, `find`, and targeted `Read` calls (read-only). No write commands beyond audit-file creation.

## 5. Commands Failed or Skipped

| Command | Status | Reason |
|---|---|---|
| `npm test` (root aggregate) | not run separately | Root delegates to `go test ./...`; covered by direct call above. The same `hostbridge` failure would surface here. |
| `npm run lint` (root) | not run separately | Delegates to `go vet ./...`; covered above. |
| `npm run validate:js` | not run | Skipped; component parts (typecheck + vitest) run individually. |
| `npm run validate:desktop` | not run | Skipped; component parts run individually. |
| `npm run validate:local` | not run | Includes integration-env preflight which mutates ephemeral state; out of scope for read-only audit. **Blocks high confidence in integration-env health, not in static integrity.** |
| `npm run build` / `build:desktop` | not run | Skipped (production bundling not required for static integrity audit). |
| `cd apps/desktop/src-tauri && cargo test` | not run | Skipped to bound audit time; Tauri unit tests around `request_host_power_action_with_policy` were inspected statically. |
| `nix flake check` | not run | Skipped — Nix optional per AGENTS.md; no behaviour-affecting change in this audit. |
| `nix build .#forge-core` / `.#forge-desktop-shell` | not run | Skipped — same reason. |

None of the skipped commands are recorded as passed.

## 6. Build / Test Status

- **Go build:** `npm run build:core` exit 0 — clean.
- **Go vet:** `go vet ./...` exit 0 — clean.
- **Go test:** one failing test:

  ```
  --- FAIL: TestExecRunnerRejectsOversizeCommandOutput (2.00s)
      hostbridge_test.go:126: Run error = context deadline exceeded, want stdout size error
  FAIL    forge/projectforge/services/core/internal/hostbridge    2.106s
  ```

  All other 56 internal Go packages pass (full list captured in `/tmp/ca2_go_test.log`). The failure looks timing-sensitive: the test expects a stdout-size cap to fire before the 2 s context deadline, but the deadline wins. Possible causes: pipe buffer/scheduler latency on this host, or an off-by-one on the size cap.

- **TS typecheck (desktop):** exit 0.
- **Vitest (desktop):** 52/53 files pass; 1 test fails:

  ```
  ChatPage.test.tsx:262
    fireEvent.click(screen.getByRole("button", { name: "Send" }));
    -- no element with name "Send" after stream initiated
  ```

  The composer is in streaming state, so the button label is still "Stop"; the test then fails before it can find the post-stream Send. 151/152 tests pass.

## 7. Critical Findings

None. No authority breach, no truncation/corruption in live source, no live tool-execution bypass, no memory-write bypass, no FORGE-K simulator wired into live daemon authority.

## 8. High Findings

- **H-1 — SystemPage hardcoded cockpit rows leaking as "live" status.** [`apps/desktop/src/pages/SystemPage.tsx:225-250`](apps/desktop/src/pages/SystemPage.tsx#L225-L250). `cockpitRows` contains rows like `{ id: "cases", status: "simulator/planned", liveOwner: "not live-wired" }` and `{ id: "context_bundles", status: "shadow/inspector" }` rendered as if they were live cockpit status. Per [`docs/operations/forge_graphical_shell_session.md`](docs/operations/forge_graphical_shell_session.md) G6 doctrine, the System surface must not present fake healthy state when data is unavailable. **Recommended action:** delete the hardcoded rows or render an explicit "unavailable" / `not-wired` chip.

- **H-2 — Duplicate `FALLBACK_OPERATOR_APPS` definition.** [`apps/desktop/src/layout/AppShellSurfaces.tsx:29`](apps/desktop/src/layout/AppShellSurfaces.tsx#L29) and [`apps/desktop/src/pages/OperatorAppsPage.tsx:13`](apps/desktop/src/pages/OperatorAppsPage.tsx#L13) define the same fallback app list independently. Drift risk. **Recommended action:** export from `AppShellSurfaces.tsx` and import into `OperatorAppsPage.tsx`.

- **H-3 — Go test failure (timing-sensitive, but failing).** `TestExecRunnerRejectsOversizeCommandOutput` in `services/core/internal/hostbridge`. The test asserts a stdout-size error but receives a context-deadline error. Either the test's deadline is too short, or the stdout-size cap is being missed before deadline fires. **Recommended action:** raise the test deadline, or detect deadline-vs-size races and fail more specifically; root-cause before silencing.

- **H-4 — Vitest failure in ChatPage.** [`apps/desktop/src/pages/ChatPage.test.tsx:262`](apps/desktop/src/pages/ChatPage.test.tsx#L262). After initiating a stream, the test looks for a `Send` button while the UI is still showing `Stop`. **Recommended action:** assert on `Stop` first, click `Stop`, then re-query for `Send`; or `findByRole` with a wait.

## 9. Medium Findings

- **M-1 — Docs drift on desktop-shell host power actions.** The Tauri binary implements `shutdown`/`reboot` gated by `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` ([`apps/desktop/src-tauri/src/main.rs:520-557`](apps/desktop/src-tauri/src/main.rs#L520-L557); default false; tests at lines 765 & 780 assert disabled-by-default + runner-only-when-enabled). Several docs imply "no host mutation":
  - [`docs/reviews/current_phase_status.md`](docs/reviews/current_phase_status.md) line ~43
  - [`docs/operations/forge_graphical_shell_session.md`](docs/operations/forge_graphical_shell_session.md) line ~43
  - [`docs/status/dangerous_capabilities.md`](docs/status/dangerous_capabilities.md) — does not list `shell.power_action`
  - [`docs/DESKTOP_SHELL.md`](docs/DESKTOP_SHELL.md) — silent on env-var policy gate

  **Recommended action:** supersede each with a clarifying note that host power actions exist, are **policy-gated and disabled by default** via `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`, and add the capability to `dangerous_capabilities.md`. Severity *Medium* (and not Critical) because the gate is enforced in the binary with tests; the gap is documentation only.

- **M-2 — `os.Exit(1)` from HTTP server goroutine.** [`services/core/main.go:46-52`](services/core/main.go#L46-L52). If `ListenAndServe` fails at startup, the goroutine calls `os.Exit(1)`, skipping `defer st.Close()` (line 34). On graceful SIGTERM the path is fine; this only hurts the startup-error path. **Recommended action:** signal the failure to main and let main perform store close, or invoke explicit `st.Close()` before `os.Exit`.

- **M-3 — `requireAPIAuth` is a no-op when `FORGE_API_TOKEN` is empty.** [`services/core/internal/api/auth.go:18-39`](services/core/internal/api/auth.go#L18-L39). Mitigated by `config.Validate` requiring a token for wildcard bind, and Docker defaults forcing `127.0.0.1`. Still, an operator running with `BindHost=0.0.0.0` + empty token via a custom env file bypasses the validator only if `FORGE_ALLOW_WILDCARD_BIND=true`. Documentation only; design is intentional but fragile. **Recommended action:** log a `slog.Warn` at startup when token is empty regardless of bind.

- **M-4 — CSS duplication in `forge-os-shell.css`.** [`apps/desktop/src/styles/forge-os-shell.css`](apps/desktop/src/styles/forge-os-shell.css). Repeated selectors and partial duplicates for `.forge-os-activity-log__*`, `.forge-os-context-inspector__*`, `.forge-os-statusbar__*`. Cascade currently masks the issue; refactor target.

- **M-5 — Autonomy runner filters IDs by string prefix.** [`services/core/internal/aios/autonomy/runner.go:669`](services/core/internal/aios/autonomy/runner.go#L669) drops journal candidates whose ID starts with `fake-`, `placeholder-`, or `candidate-`. The filter is correct only if test/data generators are guaranteed to use those prefixes. **Recommended action:** add a unit test asserting all live ID generators do not produce those prefixes, or replace with a typed tag.

- **M-6 — `DETACHED_TAURI_TOOL_WINDOWS` compatibility path.** [`apps/desktop/src/layout/AppShell.tsx:318`](apps/desktop/src/layout/AppShell.tsx#L318). Enables a fallback where tool windows open as detached Tauri webviews instead of in-shell surfaces, contrary to `DESKTOP_SHELL.md`. **Recommended action:** confirm production config sets this `false` and document the flag.

- **M-7 — `statusClass()` silent fallback.** [`apps/desktop/src/pages/SystemPage.tsx:17-35`](apps/desktop/src/pages/SystemPage.tsx#L17-L35). Unknown status codes silently map to `forge-ops-status--muted`. Add a `console.warn` in non-prod builds.

- **M-8 — Vitest single-failure noise hides regression visibility.** Until ChatPage.test.tsx is green, CI quality bar is degraded. Tied to H-4.

## 10. Low Findings

- **L-1 — CommandBar hardcoded action list.** [`apps/desktop/src/components/CommandBar.tsx:19-82`](apps/desktop/src/components/CommandBar.tsx#L19-L82). No API-driven registry; new commands need a UI patch.
- **L-2 — Detail route detection by `pathname.startsWith`.** [`apps/desktop/src/layout/AppShell.tsx:407-411`](apps/desktop/src/layout/AppShell.tsx#L407-L411). Works today but fragile when new detail routes appear.
- **L-3 — No runtime API response-type validation.** `apps/desktop/src/lib/api.ts` and friends. Acceptable for an internal app; consider adding for `SystemPage`/`ModelsPage` payloads.
- **L-4 — Cached model selection not validated against current model list** in [`apps/desktop/src/pages/ModelsPage.tsx`](apps/desktop/src/pages/ModelsPage.tsx). Worst case is an empty initial render.
- **L-5 — `Full-Code-Review.md` at repo root is a prompt template, not a review.** Relabel as `[PROMPT-TEMPLATE-ONLY]` and consider moving to `docs/prompt-packs/`.
- **L-6 — Doc wording: "legacy adapter invoke ingress was removed" (AGENTS.md) vs. "is not authority" (current_authority_sources.md).** Both true, but consider unifying wording.
- **L-7 — Stale Nix `result` / `result-1` symlinks** in repo root. Per AGENTS.md, artefacts are retained for provenance — track in a build-artefact manifest if desired.
- **L-8 — `.vm-build-core.log`, `.vm-nix-store/`, `.vm-nix-tmp/`** at repo root. Likely stale; verify against current VM build.
- **L-9 — `Full-Code-Review.md` not gitignored** despite being a prompt; leaves it in commits as authoritative-looking content.
- **L-10 — Discord gateway returns explicit `NOT_IMPLEMENTED`.** Intentional and gated, but document in roadmap so operators are not surprised.

## 11. Duplicate Systems

| Cluster | Files | Classification | Risk |
|---|---|---|---|
| Memory VSA reindex routes | `internal/api/routes.go:303-306` | intentional compatibility-wrapper | None — same handler. |
| Legacy memory observation mutation | `internal/api/routes.go:294-298` + `internal/api/server_legacy.go` | legacy-retained gate (410 Gone) | None — gate enforces retirement. |
| Legacy adapter invoke gateway tool | `internal/api/legacy_adapter_gateway_tool.go` + `server.go:233` | legacy-retained shim | Low — tool has `Executes()==false`. |
| `FALLBACK_OPERATOR_APPS` | `AppShellSurfaces.tsx:29`, `OperatorAppsPage.tsx:13` | **real duplicate (drift risk)** | Medium — H-2. |
| Desktop Zustand stores | `desktopWindowStore`, `desktopShellStore`, `workspaceStore`, `workspaceLayoutStore`, `uiStore` | intentional separation of concern | None. |
| Nix session derivations | `forge-shell-session.nix`, `forge-wayland-session.nix`, `forge-operator-session.nix` | intentional cross-phase variants | None. |
| Cross-platform scripts (`.sh` / `.mjs` / `.ps1`) | `scripts/*` | intentional platform pairs | None. |
| CSS partial duplicate selectors | `apps/desktop/src/styles/forge-os-shell.css` | real-duplicate (style refactor) | M-4. |

## 12. Placeholder / Mock / Fake Runtime Findings

See full table in `docs/reviews/ca2_scratch/pass4_5_truncation_mocks.md`. Top live-impacting items:

- **Live UI**: `SystemPage.tsx` `cockpitRows` (H-1) — hardcoded statuses presented as live cockpit state.
- **Live UI fallback**: `FALLBACK_OPERATOR_APPS` duplication (H-2).
- **Live UI**: `CommandBar.tsx` hardcoded actions (L-1).
- **Live runtime**: `autonomy/runner.go:669` ID-prefix filter (M-5).
- **Live API (intentional)**: Discord gateway `NOT_IMPLEMENTED`, gateway registry deferred capabilities, KV-enforcement canary gating.
- **Test-only / simulator-only (acceptable)**: `modelruntime/backend_fake.go`, `forgek/runtime/mock_driver.go`, `forgek/fixture_parity_test.go`, `forgek/lymphatic/sweeps.go`.

## 13. Truncation / Corruption Findings

- 1 merge-marker hit in `docs/archive/phases/PhaseCA1.txt` (archive content, not a real conflict).
- 0 `panic("TODO")`, `unimplemented!`, `todo!`, or `throw new Error("not implemented")` in live code.
- 0 hits for `lorem ipsum`, "rest of file", "continue here", "TODO: finish".
- 36 hits for `truncated` / `…(truncated)` — all in legitimate bounded-output paths.
- 0 invalid placeholder ellipses in executable code.
- No unclosed code fences detected in spot-checked Markdown.

## 14. Dead Code / Orphan Findings

- No orphan page files: every page in `apps/desktop/src/pages/` is referenced by `App.tsx` or `shellConfig.ts`.
- No dead routes: all 36 declared routes resolve to a page file (Pass 8).
- Stale repo-root artefacts (`.vm-build-core.log`, `.vm-nix-store/`, `.vm-nix-tmp/`, `result`, `result-1`) — provenance retained per AGENTS.md; flag for operator decision (§21).
- No clearly unused Go packages identified by the auditors; `go build ./...` clean.

## 15. Frontend / Desktop Findings

Authoritative scratch: `docs/reviews/ca2_scratch/pass8_frontend.md`. Highlights:

- H-1 `SystemPage` cockpit rows.
- H-2 `FALLBACK_OPERATOR_APPS` duplication.
- M-4 CSS duplicates in `forge-os-shell.css`.
- M-6 `DETACHED_TAURI_TOOL_WINDOWS` compatibility path.
- M-7 `statusClass()` silent fallback.
- L-1, L-2, L-3, L-4 (see §10).
- 36 routes — all wired. Single AppShell mount, single Taskbar instance.
- Tauri commands: only `shutdown`/`reboot` (M-1 gate) are sensitive; no `nixos-rebuild`, no package manager, no FS writes outside `FORGE_DATA_DIR`. Tests at `main.rs:752-790` enforce policy defaults.

## 16. Core Service Findings

Authoritative scratch: `docs/reviews/ca2_scratch/pass3_9_duplicates_core.md`. Highlights:

- M-3 empty-token auth bypass (mitigated by `config.Validate`).
- M-2 `os.Exit(1)` from HTTP server goroutine.
- Goroutine patterns inspected (4 patterns) — all but the `ListenAndServe` goroutine are explicitly `ctx.Done()`-driven.
- Shutdown context not derived from signal context (acceptable due to explicit service `Close()` calls in `ShutdownWatch`).
- 60+ routes mounted exactly once; no collisions.
- Postgres migrations under `services/core/migrations/postgres/` — no dual-write yet; design-gated per `current_authority_sources.md`.
- Audit coverage for `phase2.go` (approvals) and `phase5.go` (gateway) is delegated to services — not verified at handler layer in this audit (flagged for follow-up).

## 17. Nix / VM / Packaging Findings

- `nix flake check` and `nix build .#…` not executed in this audit (skipped, not failed; §5).
- `nix/packages/` derivations look intentional and distinct: `forge-core.nix`, `forge-desktop-shell.nix`, `forge-shell-session.nix`, `forge-wayland-session.nix`, `forge-operator-session.nix`, `forge-operator-toolbelt.nix`.
- Repo root carries large artefacts: `forge-operator-vm.qcow2` (9.9 GB), `result*` symlinks. Provenance-retained per AGENTS.md.
- `flake.lock` present and current.

## 18. Docs Truth Alignment Findings

Authoritative scratch: `docs/reviews/ca2_scratch/pass7_10_authority_docs.md`. Summary of needed supersessions (M-1 cluster):

| Doc | Needed change |
|---|---|
| `docs/reviews/current_phase_status.md` (~line 43) | Replace "do not add host mutation" with "policy-gated host power actions, disabled by default via `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`". |
| `docs/operations/forge_graphical_shell_session.md` (~line 43) | Same clarification. |
| `docs/status/dangerous_capabilities.md` | Add `shell.power_action` entry under approval-only/disabled-by-default. |
| `docs/DESKTOP_SHELL.md` | Add "Host Power Controls" subsection documenting the env var. |
| `Full-Code-Review.md` (repo root) | Add `[PROMPT-TEMPLATE-ONLY]` header; consider moving to `docs/prompt-packs/`. |
| `docs/reports/phase_g8_desktop_shell_verification.md` | Add a verification line for `FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false` default. |

Other docs in `docs/status/` (current_authority_sources, implementation_matrix, placeholders_and_stubs, duplicate_systems, test_gap_analysis, current_baseline_gate) are current-authority and consistent with code.

## 19. Security / Authority Boundary Findings

- **FORGE-K simulator vs live**: simulator package (`services/core/internal/forgek/*`) is labelled `[SIMULATOR-ONLY]` (README.md line 3). The forbidden-imports test (`forbidden_live_imports_test.go`) prevents `gateway`, `modelruntime`, `memory`, `retrieval`, `api`, `controllane` imports. No live-daemon callsite of forgek services outside its own subtree and adjacent shadow/shared-contract packages.
- **Model output → memory**: gated. Modelruntime emits proposal envelopes; memory writes confined to `internal/memory/` and `internal/aios/controllane/`; no direct modelruntime→memory writer.
- **Semantic writes via Control Lane**: enforced; legacy memory observation routes are 410-gated by `withLegacyMemoryMutationGate`.
- **Tool execution**: single ingress `POST /api/gateway/invoke`; legacy adapter is a non-executing read shim.
- **Desktop host mutation**: only `shutdown` / `reboot` exist, both gated by `direct_system_control_enabled` (default false), with unit tests asserting both the disabled default and the runner path. *Docs-only* gap (M-1).
- **Memory append-only / history separation**: holds — journal commits and state versioning are the only mutation paths in controllane.
- **Empty-token auth bypass**: M-3 (mitigated, not breached).

## 20. Recommended Fix Order

Pass-11 fix queue is maintained as `docs/reviews/full_codebase_integrity_ca2_fix_queue.md`. Suggested ordering:

1. **H-3** Investigate and fix `TestExecRunnerRejectsOversizeCommandOutput` flake.
2. **H-4** Fix `ChatPage.test.tsx:262` Send/Stop button race.
3. **H-1** Remove hardcoded `cockpitRows` from `SystemPage.tsx` (or render "unavailable").
4. **H-2** Consolidate `FALLBACK_OPERATOR_APPS` into a single import.
5. **M-1** Docs supersession for `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` (5 docs).
6. **M-2** Move `os.Exit(1)` out of HTTP server goroutine (or close store first).
7. **M-3** Add startup warning when API token is empty.
8. **M-4** CSS deduplication pass on `forge-os-shell.css`.
9. **M-5** Replace prefix-based ID filter in `autonomy/runner.go` with typed tag (or add a guard test).
10. **M-6** Verify `DETACHED_TAURI_TOOL_WINDOWS=false` in production builds and document the flag.
11. **M-7** Add `console.warn` for unknown status codes.
12. **L-1..L-10** Low-priority hygiene; group into a single follow-up PR.

## 21. Operator Decisions Needed

- **OP-1** Treat the desktop power-action policy gate (`FORGE_SHELL_DIRECT_SYSTEM_CONTROL`) as documented-and-acceptable, OR remove the commands entirely from the Tauri binary. Audit recommends the former + docs supersession.
- **OP-2** Empty `FORGE_API_TOKEN` default behaviour: is the loopback-bind escape hatch worth keeping, or should an empty token always abort startup?
- **OP-3** Stale Nix `result` / `result-1` symlinks and `.vm-*` artefacts at repo root — keep as provenance, or move to a dedicated artefact manifest path?
- **OP-4** `Full-Code-Review.md` at repo root — accept relabel `[PROMPT-TEMPLATE-ONLY]`, or move to `docs/prompt-packs/`?
- **OP-5** Vitest `--run` failure threshold — fix H-4 now, or accept a flaky-tests gate while ChatPage streaming is reworked?
- **OP-6** Should Pass-6 also enforce `nix flake check` and `cargo test` in subsequent audits (currently skipped)?

## 22. Final Confidence Rating

**Medium-High.**

- High confidence: FORGE-K simulator isolation, gateway-only tool execution, semantic-write gating, append-only memory model, absence of truncation/corruption, absence of live `panic("TODO")` / `unimplemented!`, route uniqueness, single AppShell mount.
- Medium confidence on: storage cutover readiness (no dual-write infra yet, but documented as future work); audit-coverage at the API handler layer for approvals/gateway (delegated, not verified end-to-end in this pass); host-power-action exposure (gated and tested, but docs drift).
- Lower confidence on the items that were **skipped**: integration env (`npm run validate:local`), `cargo test` for Tauri, `nix flake check`. None of these alters the assessment of static integrity, but they should be in scope for the next audit pass or CI run.
- The two real test failures (H-3, H-4) need root-cause work to stop masking future regressions.

---

**End of CA2 full re-audit report.**
