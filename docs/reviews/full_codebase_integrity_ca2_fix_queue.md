# CA2 Fix Queue

Authoritative source: [`docs/reviews/full_codebase_integrity_reaudit_ca2.md`](full_codebase_integrity_reaudit_ca2.md). This file is the actionable queue extracted from CA2 findings. **No fix work has been performed under CA2 itself.** Operator must approve a CA3 fix pass to enact any of the items below.

## Suggested PR batches

### Batch 1 — "Restore green CI" (H-3, H-4)
- H-3 — `services/core/internal/hostbridge` `TestExecRunnerRejectsOversizeCommandOutput` returns context-deadline instead of size error. Investigate root cause (pipe buffer race / cap mis-calc) before relaxing the deadline.
- H-4 — `apps/desktop/src/pages/ChatPage.test.tsx:262` looks for `Send` before stream completes. Fix: assert `Stop` first, click `Stop`, then `findByRole("button", { name: "Send" })` with wait.

Acceptance: `go test ./...` and `npm -w @forge/desktop run test -- --run` both exit 0.

### Batch 2 — "G6 honesty + UI duplicates" (H-1, H-2, M-7)
- H-1 — Remove hardcoded `cockpitRows` from `apps/desktop/src/pages/SystemPage.tsx:225-250`. Replace with API-sourced rows or explicit "unavailable" state.
- H-2 — Export `FALLBACK_OPERATOR_APPS` from `apps/desktop/src/layout/AppShellSurfaces.tsx`; delete the duplicate in `apps/desktop/src/pages/OperatorAppsPage.tsx:13`; import the shared symbol.
- M-7 — Add `console.warn` for unknown status codes in `statusClass()` (`apps/desktop/src/pages/SystemPage.tsx:17-35`) in non-prod builds.

Acceptance: SystemPage renders only data sourced from `api.system.status()`; no duplicate fallback list; typecheck and vitest still green.

### Batch 3 — "Power-action docs supersession" (M-1, OP-1)
- Update `docs/reviews/current_phase_status.md` (~L43): replace "do not add host mutation" with the policy-gated wording.
- Update `docs/operations/forge_graphical_shell_session.md` (~L43): same clarification.
- Update `docs/DESKTOP_SHELL.md`: add a "Host Power Controls" section pointing to `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`.
- Update `docs/status/dangerous_capabilities.md`: add `shell.power_action` entry.
- Update `docs/reports/phase_g8_desktop_shell_verification.md`: add verification line for the default `false`.
- Relabel `Full-Code-Review.md` at repo root with `[PROMPT-TEMPLATE-ONLY]` header (or move to `docs/prompt-packs/` per OP-4).

Acceptance: no doc claims "host mutation disabled" without referencing the policy gate.

### Batch 4 — "Core hardening" (M-2, M-3, M-5)
- M-2 — Pull `ListenAndServe` failure out of `os.Exit(1)` in a goroutine (`services/core/main.go:46-52`). Either signal main via a channel, or call `st.Close()` before `os.Exit`.
- M-3 — Add a `slog.Warn("API token not set", ...)` at startup regardless of bind host (`services/core/internal/api/auth.go` / `services/core/main.go`).
- M-5 — Replace prefix-based ID filter in `services/core/internal/aios/autonomy/runner.go:669` with a typed tag, or add a unit test asserting live ID generators never produce those prefixes.

Acceptance: `go test ./...`, `go vet ./...`, `go build ./...` all clean; startup log emits the warn when token empty.

### Batch 5 — "Hygiene cleanup" (M-4, M-6, L-1 .. L-10)
- M-4 — CSS dedup pass on `apps/desktop/src/styles/forge-os-shell.css`.
- M-6 — Confirm `DETACHED_TAURI_TOOL_WINDOWS=false` in prod build; document the flag.
- L-1 — Either document CommandBar manual-update process or sketch a registry API.
- L-2 — Move `/jobs/:id`, `/memory/chunk/:id` into the declared route table.
- L-3 — Optional zod/typia guards for `SystemPage` / `ModelsPage` payloads.
- L-4 — Validate cached model selection against live list in `ModelsPage`.
- L-5 / L-9 — Header-label or relocate `Full-Code-Review.md`; consider `.gitignore`.
- L-6 — Unify "legacy adapter" wording across `AGENTS.md` and `docs/status/current_authority_sources.md`.
- L-7 / L-8 — Operator decision on retention (`result`, `result-1`, `.vm-build-core.log`, `.vm-nix-store/`, `.vm-nix-tmp/`).
- L-10 — Add Discord deferred state to `docs/status/implementation_matrix.md`.

Acceptance: hygiene PR builds clean; no behaviour change.

## Items NOT in scope for the fix queue (audit notes only)

- Storage cutover authority (SQLite → Postgres dual-write) — documented as future work in `docs/status/current_authority_sources.md`; design-gated.
- FORGE-K live authority migration — explicitly out of scope by phase doctrine.
- Modelruntime hardening/supervision — already tracked under existing modelruntime phases (M3/M4).

## Per-item metadata

Severity, category, files, evidence and "safe for automated fix" flags live in [`full_codebase_integrity_ca2_findings.csv`](full_codebase_integrity_ca2_findings.csv).
