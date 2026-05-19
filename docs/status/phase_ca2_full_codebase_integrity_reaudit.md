# Status — Phase CA2 Full Codebase Integrity Re-Audit

**Phase:** PHASE CA2 — FULL_CODEBASE_INTEGRITY_REAUDIT_AND_FIX_QUEUE
**Scope marker:** `AUDIT_ONLY / NO_RUNTIME_BEHAVIOR_CHANGE / NO_AUTHORITY_EXPANSION / NO_HOST_MUTATION / NO_FORGE_K_AUTHORITY_MIGRATION`
**Date:** 2026-05-19
**Branch / HEAD:** `main` @ `8a2cdb8`

## State

`COMPLETE / AUDIT_ONLY / NO_RUNTIME_BEHAVIOR_CHANGE`

## Deliverables

- [`docs/reviews/full_codebase_integrity_reaudit_ca2.md`](../reviews/full_codebase_integrity_reaudit_ca2.md) — full report (22 sections)
- [`docs/reports/phase_ca2_full_codebase_integrity_reaudit.md`](../reports/phase_ca2_full_codebase_integrity_reaudit.md) — phase report
- [`docs/reviews/full_codebase_integrity_ca2_findings.csv`](../reviews/full_codebase_integrity_ca2_findings.csv) — flat findings
- [`docs/reviews/full_codebase_integrity_ca2_fix_queue.md`](../reviews/full_codebase_integrity_ca2_fix_queue.md) — prioritised fix queue
- `docs/reviews/ca2_scratch/pass{1,2,3_9,4_5,7_10,8}*.md` — per-pass scratch reports

## Counts

| Severity | Count |
|---|---|
| Critical | 0 |
| High | 4 |
| Medium | 8 |
| Low | 10 |

## Validation matrix

| Command | Status | Notes |
|---|---|---|
| `cd services/core && go test ./...` | FAIL | 1 package: `internal/hostbridge` (`TestExecRunnerRejectsOversizeCommandOutput`). 56 other packages OK. |
| `cd services/core && go vet ./...` | PASS | exit 0 |
| `npm run build:core` | PASS | exit 0 |
| `npm -w @forge/desktop run typecheck` | PASS | exit 0 |
| `npm -w @forge/desktop run test -- --run` | FAIL | 151/152 pass; failing test: `apps/desktop/src/pages/ChatPage.test.tsx:262`. |
| `npm run validate:local` | SKIPPED | Out of scope for read-only audit. |
| `npm run validate:js` / `:desktop` | SKIPPED | Component parts run individually. |
| `cargo test` (Tauri) | SKIPPED | Audit time-bound. Static inspection of `request_host_power_action_with_policy` confirms binary-level gate with tests at `main.rs:765` / `:780`. |
| `nix flake check` / `nix build .#forge-core` / `nix build .#forge-desktop-shell` | SKIPPED | Nix optional per AGENTS.md. |

No skipped command is recorded as passed.

## Authority posture (unchanged by this phase)

- FORGE-K simulator (`services/core/internal/forgek/*`) remains `[SIMULATOR-ONLY]`. Forbidden-imports test confirmed.
- Tool execution: gateway-only ingress.
- Semantic writes: Control Lane–gated.
- Memory: append-only / journaled.
- Live validation seams (`VALIDATE_REF_SHAPE`, `VALIDATE_KV_IDENTITY`, `COMPARE_REF_SHAPE`, `VALIDATE_SEMANTIC_OPERATION`, `VALIDATE_CONTEXT_ATTRIBUTION`) remain `[PARTIAL LIVE VALIDATION]` only.
- Desktop shell `shutdown`/`reboot`: binary-gated by `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` (default false), unit-tested. Docs require supersession (M-1).

## Open operator decisions

OP-1 (power-action documentation), OP-2 (empty token policy), OP-3 (stale repo-root artefacts), OP-4 (`Full-Code-Review.md` label/move), OP-5 (vitest gate during ChatPage stream rework), OP-6 (extend Pass-6 to Nix/cargo in next audit). Detail in main report §21.

## Next

Operator may approve a single CA3 fix-pass PR addressing the H-1..H-4 + M-1 items in the fix queue. No fix work has been performed in CA2 itself.
