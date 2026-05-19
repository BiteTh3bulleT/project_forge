# PHASE CA2 — Pass 4 & 5: Truncation/Corruption Scan + Placeholder/Mock/Fake Runtime Audit

**Auditor:** Truncation-Mocks
**Date:** 2026-05-19
**Repo:** /home/rshort/WTF/ProjectForge
**Status:** AUDIT_ONLY / NO_RUNTIME_BEHAVIOR_CHANGE

Skipped paths in scans: `node_modules/`, `.git/`, `.worktrees/*/node_modules`, `*.qcow2`, `*.zip`, `result*`, `FORGE-HMK_Ultimate_Prompt_Pack/`, `FORGE/.obsidian/`, `apps/desktop/src-tauri/target/`.

---

## Pass 4 — Truncation / Corruption Scan

### Merge-conflict markers (`<<<<<<<`, `=======`, `>>>>>>>`)
- 1 hit total in scanned set: `docs/archive/phases/PhaseCA1.txt` — content of an archived prompt artefact discussing the markers, not an unresolved conflict. **No live conflict markers in source.**

### `panic("TODO")`, `unimplemented!`, `todo!`, `throw new Error("not implemented")`
- **0 hits** in live Go/Rust/TS code.

### `TODO` / `FIXME` / `HACK` / `STUB` / `MOCK` / `PLACEHOLDER` / `DUMMY` / `FAKE` / `NOT_IMPLEMENTED` (representative live-code hits, excluding tests / simulator / docs)
- `services/core/internal/forgek/runtime/statuses.go` — `DriverKindMock` enum constant (simulator deterministic driver kind; intentional).
- `services/core/internal/api/discord_gateway_service.go` — returns error code `"NOT_IMPLEMENTED"` for deferred Discord features (explicit gating, not silent stub).
- `services/core/internal/gateway/tool_capability_registry.go` — `model.embed` / `model.benchmark` registered as deferred capabilities (explicit gating).
- `services/core/internal/aios/controllane/kv_enforcement.go` — message `"not implemented outside canary path"` (correctly scopes KV reuse to canary).
- `services/core/internal/forgek/lymphatic/sweeps.go` — Phase 10+ "not implemented" warnings (simulator audit-only).
- `services/core/internal/forgek/fixture_parity_test.go` — `"MOCK"` token in test fixture (test-only).
- `crates/forgek-validate/src/types.rs` — internal test utility (test-only).

### "truncated", "rest of file", "continue here", "TODO: finish", "lorem ipsum", "sample data", "hardcoded demo"
- 36 matches for `truncated`/`…(truncated)` — all in bounded-output paths (log truncation, token-counting suffixes, response-size limiters). Legitimate.
- 0 hits for `lorem ipsum` / `rest of file` / `continue here` / `TODO: finish` in live code.

### Placeholder ellipses in executable code
- No invalid `...` placeholders in Go or Rust. All occurrences are legitimate variadic (`args ...`, `dest ...`) or TS spread.

### Dangling imports
- Spot-checked Go and TS imports — no broken paths surfaced; `go vet ./...` (Pass 6) reports clean.

### Mid-sentence / unclosed-fence docs
- Spot-checked 20+ Markdown files >100 lines under `docs/` — all properly closed code fences; no truncation tails observed.

**Pass 4 verdict:** No truncation or corruption defects in the live working set. Single merge-marker hit is inside an archived prompt artefact (`docs/archive/phases/PhaseCA1.txt`), not a real conflict.

---

## Pass 5 — Placeholder / Mock / Fake Runtime Audit

### Live-runtime risks (in the working tree, not test-only/simulator-only)

| # | File:line | Category | Severity if Live | Notes |
|---|---|---|---|---|
| 1 | `apps/desktop/src/pages/SystemPage.tsx:225-250` (`cockpitRows`) | live UI | **High** | Hardcoded "simulator/planned", "shadow/inspector" status rows rendered as if live state. Violates G6 doctrine ("no fake healthy state"). Cross-checked against `shellConfig.ts` — page is a current route. |
| 2 | `apps/desktop/src/layout/AppShellSurfaces.tsx:29` + `apps/desktop/src/pages/OperatorAppsPage.tsx:13` (`FALLBACK_OPERATOR_APPS`) | live UI fallback | Medium | Duplicate fallback list. Not fake data per se but a drift risk: two places to update. |
| 3 | `apps/desktop/src/components/CommandBar.tsx:19-82` (`commandActions`) | live UI | Low-Medium | Hardcoded action array; no API-driven command registry. Acceptable as static surface for now, but document. |
| 4 | `services/core/internal/aios/autonomy/runner.go:669` | live runtime | Medium | Filters out IDs starting with `"fake-"`, `"placeholder-"`, `"candidate-"` in journal cleanup. Implies test-data leakage risk if generators don't enforce that prefix discipline. |
| 5 | `services/core/internal/api/discord_gateway_service.go` (`NOT_IMPLEMENTED`) | live API | Low | Deferred feature explicitly returns gated error code. Intentional. |
| 6 | `services/core/internal/gateway/tool_capability_registry.go` (`model.embed`, `model.benchmark`) | live runtime | Low | Deferred capabilities, properly gated. Intentional. |
| 7 | `services/core/internal/aios/controllane/kv_enforcement.go` ("not implemented outside canary path") | live runtime | Low | KV reuse path gated by canary; informational. |

### Test-only / simulator-only / docs-only (acceptable per CA2 doctrine)

- `services/core/internal/modelruntime/backend_fake.go` — `FakeBackend`. Test fixture; not imported by live `service.go`. **Acceptable.**
- `services/core/internal/forgek/runtime/mock_driver.go` — `MockRuntimeDriver`. Phase 9 simulator deterministic driver. **Acceptable.**
- `services/core/internal/forgek/fixture_parity_test.go` — fixture token. **Acceptable.**
- `services/core/internal/forgek/lymphatic/sweeps.go` — simulator-only audit/reporting. **Acceptable.**

### Top-10 most concerning placeholder/mock findings (live-impacting)

1. SystemPage `cockpitRows` hardcoded — **High** (G6 violation)
2. Duplicate `FALLBACK_OPERATOR_APPS` — Medium (drift risk)
3. Autonomy runner `fake-`/`placeholder-`/`candidate-` prefix filter relying on convention — Medium
4. CommandBar hardcoded actions — Low-Medium
5. Discord `NOT_IMPLEMENTED` (explicit) — Low (acceptable)
6. Tool registry deferred entries — Low (acceptable)
7. KV-enforcement "not implemented outside canary" — Low (acceptable)
8. ModelRuntime FakeBackend — Test-only, **no live risk**
9. FORGE-K MockRuntimeDriver — Simulator-only, **no live risk**
10. Archived prompt with merge markers — Archival only, **no live risk**

---

## Synthesis Notes

- No truncation, corruption, or unclosed code fences in live source.
- Live placeholder/mock risk concentrates in **one** location: `apps/desktop/src/pages/SystemPage.tsx` cockpit rows that present non-live subsystems with statuses formatted like live state. Recommend explicit "unavailable" rendering or omission.
- `FALLBACK_OPERATOR_APPS` duplication is the only true code-level duplicate placeholder pattern.
- Autonomy runner's prefix filter is the only live-runtime use of `fake/placeholder/candidate` tokens; assess whether test-data generators are guaranteed to use those prefixes.

**End of Pass 4 & 5 scratch.**
