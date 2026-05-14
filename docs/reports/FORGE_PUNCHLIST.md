# FORGE Punch List — Path to "Everything Wired and Working Properly"

**Generated:** 2026-05-11.
**Companions:** [FORGE_FULL_REVIEW.md](FORGE_FULL_REVIEW.md), [FORGE_LARGE_FILE_INVENTORY.md](FORGE_LARGE_FILE_INVENTORY.md).
**Goal:** Reach the point where FORGE is functionally complete for personal daily use. Not "feature-finished forever" — wired correctly, splits done, coverage at sane levels, no latency cliffs, no architectural smells gating future work.
**Target window:** ~2 weeks at current velocity.

---

## Current State Snapshot

- **Build:** 54 Go packages, 0 fails, vet clean, ~20s test wall time.
- **Coverage on load-bearing packages:**
  - `memory`: 11.0% — was 6.5%
  - `aios/controllane`: 13.9% — was 12.1%
  - `aios/autonomy`: 9.2% — unchanged
  - `aios/dream`: 20.5% — was 17.2%
  - `aios/hyperlane`: **66.7%** — was 0%
  - `gateway`: 18.8%
  - `api`: 22.6%
- **Untested packages:** 18 (down from 27).
- **Largest source file:** `backup/service.go` at 2,005 lines (down from `gateway/service.go` at 4,709).
- **In flight:** `lib/api.ts` split underway (3 commits in, chat/runtime/memory surfaces extracted).
- **Working tree:** clean. Pushed.

---

## Section 1 — Hygiene (Easy, P1)

Trivial wins. Close the small stuff so it stops appearing in every review. ~1-2 hours total.

- [x] **ADR 0001 numbering collision.** Rename `docs/adr/0001-forge-is-ai-os.md` → `0000-forge-is-ai-os.md`. Grep+update cross-references in README, AGENTS.md, CODEX.md, and the architecture docs.
- [x] **Wildcard bind: fail-closed.** In `services/core/main.go`, refuse to bind to `0.0.0.0`/`::` unless `FORGE_ALLOW_WILDCARD_BIND=true`. Currently only logs a warning.
- [x] **Gitignore VM artifacts.** Add `.vm-*`, `.vm-nix-store/`, `.vm-nix-tmp/`, `operator_toolbelt.txt` to `.gitignore`.
- [x] **Archive phase root files.** Move `PhaseM4.txt` from repo root to `docs/superpowers/specs/2026-05-11-forge-phase-m4-vllm.md` to match existing pattern.
- [x] **Archive loose follow-up phase prompts.** Move the root G7/M5 phase prompts into `docs/superpowers/specs/`; move the legacy Phase 9 prompt into `docs/archive/phases/`.
- [x] **Archive old phase reviews.** Move `docs/reviews/phase_12*.md`, `docs/reviews/phase_13*.md` into `docs/reviews/archive/phase_12/` and `archive/phase_13/`.
- [x] **Delete or fill empty `Operator-Toolbelt.txt`.** It's now superseded by the Nix package + `docs/operations/operator_toolbelt.md`.
- [x] **Resolve duplicate review docs.** Pick one canonical historical review in `docs/reviews/` (`full_project_forge_review.md` vs `full_project_review.md` vs `full_project_review_checklist.md`). Archive the others.

---

## Section 2 — File Splits (Refactor)

Detail in [FORGE_LARGE_FILE_INVENTORY.md](FORGE_LARGE_FILE_INVENTORY.md). Summary order of operations:

### Desktop side (highest leverage — affects daily use)

- [x] `lib/api.ts` — chat surface extracted (d5d5984)
- [x] `lib/api.ts` — runtime surface extracted (d5d5984)
- [x] `lib/api.ts` — memory surface extracted (e684a28)
- [x] `lib/api.ts` — extract remaining domains (`memory`, `approvals`, `audit`, `jobs`, `retrieval`, `gateway`, `system`, `autonomy`, `dream`, `backup`, `integrations`). Each into `apps/desktop/src/lib/api/<domain>.ts`.
  - 2026-05-13 progress: extracted `canvas` into `apps/desktop/src/lib/api/canvas.ts`; `api.ts` is now 1,001 lines.
  - 2026-05-13 progress: extracted `approvals` and `jobs` into `apps/desktop/src/lib/api/`; `api.ts` is now 933 lines.
  - 2026-05-14 progress: extracted `system`, `settings`, `remote`, `sources`, `autonomy`, `gateway`, `backup`, `artifacts`, and `release` domains; `api.ts` is now 488 lines.
  - 2026-05-14 progress: extracted remaining inline API domains (`packets`, `projectContext`, `embeddings`, `retrieval`, `dossiers`, `evaluations`, `lineage`, `imports`, `insights`, `dashboard`, `strategies`, `policy`, `automation`, `packetGuidance`, `reconciliation`, `reviews`, `failurePatterns`); `api.ts` is now a 149-line aggregator.
- [ ] **`apps/desktop/src/pages/ChatPage.tsx` (3,540 lines).** Split into `ChatPage/{index, MessageList, MessageItem, Composer, ToolPanel, ApprovalsPanel, useChatStream, useChatHistory, useChatComposer, types}.tsx`. Largest single file in the repo. Affects how tonight feels.
  - 2026-05-13 progress: extracted inspector derivation into `ChatPage/useChatInspectorData.ts`; `ChatPage.tsx` is now 1,940 lines.
- [ ] **`apps/desktop/src/pages/InspectorsPage.tsx` (2,444).** Split per-inspector sub-component.
- [ ] **`apps/desktop/src/pages/ModelsPage.tsx` (1,950).** Split into list/detail/import/runtime panels.
- [ ] **`apps/desktop/src/pages/SettingsPage.tsx` (1,825).** Split by settings domain.
- [x] **`apps/desktop/src/layout/AppShell.tsx` (1,648).** Split into Sidebar/TopBar/StatusBar/WindowFrame + extract window manager.
  - 2026-05-14 progress: extracted wallpaper, floating window, Start menu, icon, and context-menu surfaces into `AppShellSurfaces.tsx`; `AppShell.tsx` is now 998 lines.
- [ ] **`apps/desktop/src/stores/workspaceLayoutStore.ts` (1,374).** Split store into model + actions + selectors.
- [ ] **`apps/desktop/src/pages/DashboardPage.tsx` (1,320).** Split into Tiles + LiveStream.
- [ ] **`apps/desktop/src/pages/MemoryPage.tsx` (1,107).** Split into NoteList + NoteDetail + Filters.

### Go side

- [ ] **`services/core/internal/backup/service.go` (2,005).** Split into `service.go`, `export.go`, `restore.go`, `scheduler.go`, `tamper.go`, `outcomes.go`, `helpers.go`. Largest Go file remaining.
  - 2026-05-13 progress: extracted static section mappings into `section_mappings.go`; `service.go` is now 985 lines and no backup source file exceeds 1,500 lines.
  - 2026-05-13 progress: extracted restore section policy helpers into `restore_sections.go`; `service.go` is now 805 lines.
- [ ] **`services/core/internal/modelruntime/service.go` (1,581).** Split by lifecycle stage: `service.go`, `lifecycle.go`, `selection.go`, `queue.go`, `usage.go`, `policy.go`.
  - 2026-05-14 progress: extracted runtime health/supervision into `service_health.go`; `service.go` is now 1,488 lines.
- [x] **`services/core/internal/api/autonomy_maintenance_loop.go` (1,545).** Split by phase: loop driver + phase implementations + charters + budgets.
  - 2026-05-14 progress: extracted public report/status types and loop state into `autonomy_maintenance_loop_types.go`; `autonomy_maintenance_loop.go` is now 1,407 lines.
- [ ] **`services/core/internal/aios/controllane/compile_context_restore_scoring.go` (1,478).** Split into `listing`, `ranking`, `threshold`, `fallback`, `persistence`.
- [ ] **`services/core/internal/jobs/service.go` (1,452).** Split by lifecycle (queue/exec/result/events).
- [ ] **`services/core/internal/aios/dream/service.go` (1,447).** Watch first; if it stays at 1,447 in a week, split by dream phase.
- [ ] **`services/core/internal/api/model_runtime_bridge.go` (1,413).** Split into translation + lifecycle bridge + status bridge.
- [ ] **`services/core/internal/api/chat_post.go` (1,319).** Split into POST validator + dispatch + response shaping.
- [ ] **`services/core/internal/api/phase5.go` (1,297).** Archive if no longer routed, otherwise split.
- [ ] **`services/core/internal/aios/compute/librarian/cells_phase4.go` (1,193).** Split by cell category.
- [ ] **`services/core/internal/aios/truth/engine.go` (1,060).** Watch.
- [ ] **`services/core/internal/retrieval/service.go` (1,028).** Watch.

### Rust side

- [ ] **`apps/desktop/src-tauri/src/main.rs` (766).** Split into `main.rs` + `commands/{operator_apps, window, events}.rs` + `state/mod.rs`.

### Stop condition

No single non-inherent source file >1,500 lines. SQL migrations, validator crates, and barrel files are exempt.

---

## Section 3 — Coverage

Target: every load-bearing package at 25%+ test/source by function count. Smaller untested packages get smoke-level coverage.

### Load-bearing (P0)

- [ ] **`services/core/internal/memory/`** — 11% → 25%. The canonical-truth store. Pick the top 10 most-called functions and write substantive table-driven tests.
- [ ] **`services/core/internal/aios/controllane/`** — 14% → 25%. Focus on `processor.go`, `validator.go`, `apply_*.go`.
- [ ] **`services/core/internal/aios/autonomy/`** — 9% → 20%. Charter/budget/proposal cycle.
- [ ] **`services/core/internal/aios/dream/`** — 21% → 25%. Almost there.
- [ ] **`services/core/internal/gateway/`** — 19% → 25%. Coverage on the recent split files.

### Untested packages worth covering

- [ ] `internal/chat` — chat-side server logic
- [ ] `internal/lanes` — lane definitions
- [ ] `internal/policy` — policy evaluation
- [ ] `internal/canvas` — canvas surface
- [ ] `internal/dashboard` — dashboard rollups
- [ ] `internal/lineage` — lineage tracking
- [ ] `internal/insights` — insight surface
- [ ] `internal/dossiers` — dossier service
- [ ] `internal/evaluations` — evaluation pipeline
- [ ] `internal/search` — search service

### Untested packages worth leaving for later

`internal/release`, `internal/reviews`, `internal/reconciliation`, `internal/packetopt`, `internal/failurepatterns`, `internal/strategies`, `internal/packets`, `internal/watch` — small or low-traffic. Smoke test only when needed.

### Test infrastructure

- [ ] **Make CI integration env required.** No more silent skips on Postgres/Qdrant/Redis env vars.
- [ ] **Add `go test -race ./...` to weekly CI.**
- [ ] **Add fuzz tests** on URL/path/mode/ref/PID validators (5 fuzz targets in `gateway/`).
- [ ] **Cross-platform smoke port.** Move `scripts/forge-smoke.mjs` off bash so it runs on Windows.

---

## Section 4 — Observability and Reliability

- [x] Structured logs (slog) wired — `cc03e07 feat: emit structured event logs`
- [ ] **Complete slog migration.** Audit remaining `log.Printf` call sites in `services/core/` and migrate to slog. Tag every log line with `request_id` + `correlation_id` where available.
- [ ] **Add `/health/detailed` endpoint.** Per-service health rollup (storage, modelruntime, gateway, hostbridge, forgekshadow, dream, autonomy). One JSON body.
- [ ] **Add `/metrics` endpoint behind config flag.** Prometheus format. Hit counters, request durations, KV identity decisions, gate decisions, journal append rate.
- [ ] **Per-service graceful shutdown.** Ensure every long-lived service (jobs runner, dream loop, autonomy maintenance) responds to context cancellation cleanly.
- [ ] **Audit retention policy.** Journal and audit are append-only — confirm a documented rotation/archive plan before this becomes a disk-space problem.

---

## Section 5 — Security and Safety

- [x] Chat gateway tool argument bounds — `d2b3d77 fix: bound chat gateway tool arguments`
- [ ] **Complete chat assistant prompt-injection audit.** Trace every assignment from a model response field to `chat_assistant_*.go` files into (a) command args, (b) file paths, (c) URLs, (d) capability inputs. Verify validator coverage on each.
- [ ] **Remote token (`X-Forge-Remote-Token`) lifecycle.** Verify rotation, storage, revocation paths are tested.
- [ ] **Dangerous capabilities audit.** Run [docs/status/dangerous_capabilities.md](../status/dangerous_capabilities.md) against current `tool_capability_registry.go` — every approval-only capability still gated?
- [ ] **Wildcard bind hardening** (also in Section 1).
- [ ] **Audit `chat_post.go` (1,319 lines) for body-bound coverage.** Already has `request_body_bounds_test.go` series — verify the new chat surface didn't introduce unbounded paths.

---

## Section 6 — Functional Completeness for Daily Use

These are the items between "wired" and "works the way I want."

### Model runtime (PhaseM4 / vLLM)

- [x] PhaseM4 plan drafted
- [x] **Streaming model output.** Governed chat/SSE streaming is wired through modelruntime when the selected backend supports streaming; unsupported runtimes return structured `STREAM_UNSUPPORTED` behavior.
- [x] **vLLM-compatible external profile.** Disabled-by-default vLLM endpoint support is behind the existing modelruntime boundary, not a raw chat bypass and not a FORGE-K authority path.
- [x] **Delete-file approval flow** for managed model artifacts.
- [ ] **Stronger backend/process supervision and runtime hardening.** Restart/degraded-state policy, health probes, resource caps, deeper scheduling/backpressure, cancellation-safe accounting, and operator visibility.

### Hyperlane

- [x] Intent classifier exists (114-line `intent.go`)
- [x] No-model route contract hardened — `cfd643b`
- [x] Hyperlane test coverage to 66.7% (excellent given small surface)
- [ ] **Route real traffic through hyperlane.** Wire the classifier into the chat request path. Anything matching a deterministic intent goes deterministic; only ambiguous falls through to model.
- [ ] **Telemetry on intent distribution.** What fraction of incoming requests classify deterministically vs fall through?
- [ ] **Shadow mode first.** Compare classifier decisions vs the legacy fallback for one week before flipping the route.

### Chat path latency (lived friction)

- [ ] **Diagnose "fast first response, slow after."** Most likely culprit: KV identity cache rejecting reuse on turn 2. Check `kv_enforcement.go` counters and the journal logs across turn 1 vs turn 2.
- [ ] **Fix or accept** based on diagnosis. If KV: figure out why reuse is rejected. If context-compile: profile and cache appropriately. If model-side: defer to streaming.
- [ ] **Add a chat latency budget** — log a warning when any turn exceeds N seconds in critical phases.

### Operator desktop

- [x] Boots in VM
- [x] Operator toolbelt with ollama + tools landed
- [x] Start menu + taskbar working
- [x] Window tracking working
- [ ] **Verify ollama-in-toolbelt actually works end-to-end.** Boot, launch foot, run `ollama pull phi4-mini`, run `ollama run phi4-mini`. No PATH issues, no read-only systemd surprises.
- [ ] **Chat-to-model loop using toolbelt ollama.** Configure FORGE's modelruntime to talk to the toolbelt-provided ollama. Verify chat works inside the operator session.
- [ ] **Status bar across the shell.** One-line summary of modelruntime + autonomy + last journal entry + workspace. Data already exists.
- [ ] **Right-side context inspector.** Shows current context being compiled, recent journal entries, active loops/approvals.
- [ ] **Activity log surface.** Last 20 audit events, popover or accordion.
- [ ] **Theme variables.** Minimal CSS-vars-driven light/dark + accent.
- [ ] **Lazy-load tier-2 pages.** `React.lazy` for everything past the operator surface in `App.tsx`.

### Memory and state

- [ ] **Verify cross-session memory recall.** Open FORGE, write a note, close, reopen, ask for the note. Should work; confirm it does. This is the load-bearing daily use case.
- [ ] **Memory observation listing hardening.** Already touched in `1d119ff` — verify the fix covers your use case.
- [ ] **Document the memory note lifecycle** in `docs/memory/model.md`: create → link → supersede/archive → audit reconstruction. Public-safe abstraction level.

### Approvals and audit

- [ ] **End-to-end approval flow exercise.** Request a tool that's approval-only, see the approval prompt, grant it, watch the tool run, see the audit row.
- [ ] **Approval inspector polish.** Make sure ApprovalsPage shows pending, recent, denied with one-click grant/deny.

---

## Section 7 — Simulator-to-Live Migration

Continue the proven pattern (kvidentity, refvalidation, semanticvalidation). Pick one more narrow seam.

- [ ] **Pick the next migration target.** Candidates: lymphatic cleanup proposal validation, context compile attribution check, neural neuron proposal validation, consensus mesh claim check. Smallest seam wins.
- [ ] **Create the shared pure package** (`services/core/internal/<name>validation/`) with forbidden-imports test.
- [ ] **Add the live Control Lane syscall** (validation-only, `[PARTIAL LIVE VALIDATION]` tagged).
- [ ] **Update `AGENTS.md` and `docs/reviews/live_integration_reality_check.md`.**
- [ ] **One phase review doc** under `docs/reviews/`.

---

## Section 8 — Documentation

- [x] **Move `PhaseM4.txt` out of repo root** (also in Section 1).
- [x] **Renumber duplicate ADR 0001** (also in Section 1).
- [ ] **Add `docs/onboarding.md`.** Single-page answer to "where do I start as a new dev / collaborator / future me?"
- [ ] **Generate `docs/api/routes.md`** from chi route inventory. Run once, commit, regenerate when routes change.
- [ ] **Cross-link AGENTS.md and CODEX.md and README.md.** First-time reader should know which to start with.
- [ ] **Consolidate near-duplicate architecture docs.** `forge_ai_os.md`, `forge_k_overview.md`, `core_doctrine.md`, `control_lane_kernel.md` all touch the same kernel concept — add a "read this if you want X" header to each.
- [ ] **Tag superseded ADRs explicitly** if any.

---

## Section 9 — UI Coherence

- [x] Operator start menu
- [x] Taskbar tracking
- [x] Multi-monitor support
- [ ] **Extract shared components from pages.** 46 pages, 4 shared components — most pages duplicate fetch/error/loading. Lift `<AsyncState>`, `<KeyValueList>`, `<Panel>`, `<Toast>` into `apps/desktop/src/components/`.
- [ ] **Page-level test coverage**. ~8 of 46 pages have `.test.tsx`. Target: render test for every page.
- [ ] **Command palette polish.** `CommandBar.tsx` exists; flesh out the actions surface.
- [ ] **Accessibility audit pass.** Focus management, keyboard nav, ARIA labels on the operator-critical surfaces (Chat, Approvals, Operator Apps).

---

## Section 10 — Definition of Done

The project is "wired and working properly" when:

- [ ] No source file >1,500 lines (except SQL migrations, validator crates, barrel files).
- [ ] Memory, controllane, autonomy, gateway, api all at 25%+ test/source.
- [ ] All Section 1 hygiene items closed.
- [ ] `/health/detailed` and `/metrics` endpoints live.
- [ ] slog migration complete with correlation IDs.
- [x] Streaming model output working through modelruntime where supported.
- [x] vLLM-compatible external endpoint profile integrated per PhaseM4, disabled by default and governed by modelruntime.
- [ ] Hyperlane routing real traffic deterministically.
- [ ] Chat latency cliff diagnosed and resolved or accepted.
- [ ] Cross-session memory recall verified working.
- [ ] Operator desktop session running with toolbelt-provided ollama as the model backend.
- [ ] One more simulator-to-live migration landed.
- [ ] CI is strict (integration env required, race detector weekly, fuzz on validators).
- [ ] No remaining items in Sections 1, 4, 5, 6.

---

## Estimated Effort

| Section | Items | Estimate | Risk |
|---|---|---|---|
| 1 — Hygiene | 7 | 1-2 hours | Trivial |
| 2 — Splits | ~20 files | 20-30 hours | Low (mechanical) |
| 3 — Coverage | ~15 targets | 15-20 hours | Low |
| 4 — Observability | 5 | 6-8 hours | Low |
| 5 — Security | 6 | 4-8 hours | Medium (audit work) |
| 6 — Functional | ~15 | 25-40 hours | Medium-High (real product work) |
| 7 — Migration | 5 | 6-10 hours | Low |
| 8 — Docs | 7 | 4-6 hours | Trivial |
| 9 — UI | 4 | 10-15 hours | Low-Medium |

**Total:** ~90-140 hours of focused work. At your demonstrated pace, that's well inside 2 weeks. The functional/lived-friction items (Section 6) are the ones that will take longest because they're use-shaped, not code-shaped.

---

## Recommended Daily Cadence

If you want to actually finish in 2 weeks:

- **Mornings:** one Go file split + one coverage target (mechanical, predictable).
- **Afternoons:** one functional/Section-6 item OR one observability/security item (engaged, requires thinking).
- **Evenings:** use FORGE for real work. File the friction reports the next morning.

Don't try to clear sections in order. Sections 1, 2, 3 are background grind. Section 6 is foreground product work. Run them in parallel.

---

## Stop Conditions

You're done when:
1. The "definition of done" checklist is fully checked.
2. You can open FORGE on Monday morning, do a real task with him, close the laptop, come back Tuesday, and pick up where you left off without weird friction.
3. The shell, the kernel, the model, the memory, the approvals, the gateway, and the operator surface all do what you tell them, in the order you tell them, with the audit trail to prove it.

That's the bar. Get there.
