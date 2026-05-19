# Phase Report — CA2 Full Codebase Integrity Re-Audit

**Phase:** PHASE CA2 — FULL_CODEBASE_INTEGRITY_REAUDIT_AND_FIX_QUEUE
**Scope marker:** `AUDIT_ONLY / NO_RUNTIME_BEHAVIOR_CHANGE / NO_AUTHORITY_EXPANSION / NO_HOST_MUTATION / NO_FORGE_K_AUTHORITY_MIGRATION`
**Date:** 2026-05-19
**HEAD:** `main` @ `8a2cdb8`
**Authoritative report:** [`docs/reviews/full_codebase_integrity_reaudit_ca2.md`](../reviews/full_codebase_integrity_reaudit_ca2.md)
**Status file:** [`docs/status/phase_ca2_full_codebase_integrity_reaudit.md`](../status/phase_ca2_full_codebase_integrity_reaudit.md)
**Fix queue:** [`docs/reviews/full_codebase_integrity_ca2_fix_queue.md`](../reviews/full_codebase_integrity_ca2_fix_queue.md)
**Findings CSV:** [`docs/reviews/full_codebase_integrity_ca2_findings.csv`](../reviews/full_codebase_integrity_ca2_findings.csv)

## Outcome

Audit complete. No runtime behaviour changed. No files deleted. No CA1 artefacts overwritten (CA1 was not present in the repo).

- **Critical findings:** 0
- **High findings:** 4 (1 live UI mock leakage, 1 frontend duplicate, 2 failing tests)
- **Medium findings:** 8
- **Low findings:** 10
- **Open operator decisions:** 6

## Method

Multi-pass audit (Passes 1–11) executed by six parallel auditors plus main-thread Pass-6 validation runs. Scratch outputs:

- `docs/reviews/ca2_scratch/pass1_inventory.md`
- `docs/reviews/ca2_scratch/pass2_ca1_compare.md`
- `docs/reviews/ca2_scratch/pass3_9_duplicates_core.md`
- `docs/reviews/ca2_scratch/pass4_5_truncation_mocks.md`
- `docs/reviews/ca2_scratch/pass7_10_authority_docs.md`
- `docs/reviews/ca2_scratch/pass8_frontend.md`

## Validation commands run

| Command | Result |
|---|---|
| `cd services/core && go test ./...` | 1 package fails (`internal/hostbridge` — `TestExecRunnerRejectsOversizeCommandOutput`); 56 packages OK |
| `cd services/core && go vet ./...` | exit 0 |
| `npm run build:core` (VSA preflight + `go build ./...`) | exit 0 |
| `npm -w @forge/desktop run typecheck` | exit 0 |
| `npm -w @forge/desktop run test -- --run` | 151/152 tests pass; 1 fail in `ChatPage.test.tsx:262` |

## Validation commands skipped

`npm run validate:local`, `npm run validate:js`, `npm run validate:desktop`, `npm run build:desktop`, `cd apps/desktop/src-tauri && cargo test`, `nix flake check`, `nix build .#forge-core`, `nix build .#forge-desktop-shell`.
None were skipped because they failed; all were excluded for scope/runtime reasons. None counted as passed.

## Authority posture (unchanged)

- FORGE-K simulator (`services/core/internal/forgek/*`) remains **simulator-only**. The forbidden-imports test enforces isolation; no live daemon call site found.
- Tool execution remains **gateway-only**; legacy adapter invoke is a read-only shim with `Executes()==false`.
- Semantic writes remain **Control Lane–gated**; legacy memory observation routes return `410 Gone` via `withLegacyMemoryMutationGate`.
- Memory mutation is append-only/journaled; state versioning preserved.
- Desktop shell host power actions exist behind `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` (default false; binary-enforced; unit-tested). **Docs need supersession** to reflect this rather than implying "no host mutation."

No FORGE-K live authority expansion was introduced. No semantic syscall added. No route/API change. No memory mutation. No tool execution change.

## Top recommendation

Schedule a single follow-up PR ("CA2 fix queue, batch 1") that addresses H-3, H-4, H-1, H-2, and M-1 (docs supersession). All other findings are hygiene or design-gated future work.

## Next phase

CA3 — Targeted Fix Pass — only if operator approves the fix queue. See `docs/reviews/full_codebase_integrity_ca2_fix_queue.md`.
