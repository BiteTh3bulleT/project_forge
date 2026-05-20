# CA2 Fix Queue

Authoritative source: [`docs/reviews/full_codebase_integrity_reaudit_ca2.md`](full_codebase_integrity_reaudit_ca2.md). This file is the actionable queue extracted from CA2 findings. CA3 follow-up work is now in progress; rows marked closed below have landed after the original CA2 audit.

## Suggested PR batches

### Batch 1 — "Restore green CI" (H-3, H-4)
- H-3 — Closed by current validation evidence: `go test ./internal/hostbridge -run TestExecRunnerRejectsOversizeCommandOutput -count=1` and `go test ./...` now pass.
- H-4 — Closed by current validation evidence: `npm -w @forge/desktop run test -- ChatPage.test.tsx --run` and `npm -w @forge/desktop run test -- --run` now pass.

Acceptance: `go test ./...` and `npm -w @forge/desktop run test -- --run` both exit 0.

### Batch 2 — "G6 honesty + UI duplicates" (H-1, H-2, M-7)
- H-1 — Closed: `/forge/system/status` now returns `operator_cockpit.rows`, and `SystemPage` renders those API-owned rows with an explicit unavailable fallback when the API field is absent.
- H-2 — Closed: `FALLBACK_OPERATOR_APPS` is centralized in `apps/desktop/src/lib/operatorApps.ts`, consumed by Start/AppShell and Operator Apps, with Rust allowlist ID parity coverage.
- M-7 — Closed: `statusClass()` warns once per unknown status in dev builds.

Acceptance: SystemPage renders only data sourced from `api.system.status()`; no duplicate fallback list; typecheck and vitest still green.

### Batch 3 — "Power-action docs supersession" (M-1, OP-1)
- Closed: `docs/reviews/current_phase_status.md`, `docs/operations/forge_graphical_shell_session.md`, `docs/DESKTOP_SHELL.md`, `docs/status/dangerous_capabilities.md`, and `docs/reports/phase_g8_desktop_shell_verification.md` now describe host power actions as policy-gated and disabled by default through `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`.
- Closed: the full-review prompt template is labeled `[PROMPT-TEMPLATE-ONLY]` and archived as `docs/prompt-packs/PROMPT_FORGE_FULL_CODE_REVIEW.md`.

Acceptance: no doc claims "host mutation disabled" without referencing the policy gate.

### Batch 4 — "Core hardening" (M-2, M-3, M-5)
- M-2 — Closed: `ListenAndServe` reports through `serverErr` and the main flow returns the exit code after shutdown/defers can run.
- M-3 — Closed: startup warns when the API token is empty, while wildcard binds still require a token.
- M-5 — Closed: the prefix guard remains, with `TestLiveNoteIDGeneratorsAvoidPlaceholderPrefixes` covering live ID generator collision risk.

Acceptance: `go test ./...`, `go vet ./...`, `go build ./...` all clean; startup log emits the warn when token empty.

### Batch 5 — "Hygiene cleanup" (M-4, M-6, L-1 .. L-10)
- M-4 — Open: CSS dedup pass on `apps/desktop/src/styles/forge-os-shell.css`.
- M-6 — Closed: `DETACHED_TAURI_TOOL_WINDOWS=false` is hardcoded in `apps/desktop/src/lib/desktop.ts` and documented in `docs/architecture/desktop_window_manager.md`.
- L-1 — Closed: `docs/DESKTOP_SHELL.md` now documents the static CommandBar quick-action maintenance process and authority constraints.
- L-2 — Closed: `/jobs/:id` and `/memory/chunk/:id` are declared in `apps/desktop/src/App.tsx`.
- L-3 — Optional/open: zod/typia-style guards for `SystemPage` / `ModelsPage` payloads remain future hardening.
- L-4 — Closed: `ModelsPage` validates cached chat model selection against the live model list and reports when the cached model is absent.
- L-5 / L-9 — Closed: the prompt template moved from repo root to `docs/prompt-packs/PROMPT_FORGE_FULL_CODE_REVIEW.md`, preserving the `[PROMPT-TEMPLATE-ONLY]` header.
- L-6 — Closed: `AGENTS.md` and `docs/status/current_authority_sources.md` both describe legacy adapter direct invoke ingress as removed/non-authoritative, with gateway-only execution authority.
- L-7 / L-8 — Operator decision on retention (`result`, `result-1`, `.vm-build-core.log`, `.vm-nix-store/`, `.vm-nix-tmp/`).
- L-10 — Closed: `docs/status/implementation_matrix.md` now records remote gateways as partial, including Discord's intentionally narrow command coverage and `NOT_IMPLEMENTED` posture for unimplemented commands.

Acceptance: hygiene PR builds clean; no behaviour change.

## Items NOT in scope for the fix queue (audit notes only)

- Storage cutover authority (SQLite → Postgres dual-write) — documented as future work in `docs/status/current_authority_sources.md`; design-gated.
- FORGE-K live authority migration — explicitly out of scope by phase doctrine.
- Modelruntime hardening/supervision — already tracked under existing modelruntime phases (M3/M4).

## Per-item metadata

Severity, category, files, evidence and "safe for automated fix" flags live in [`full_codebase_integrity_ca2_findings.csv`](full_codebase_integrity_ca2_findings.csv).
