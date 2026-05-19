# FORGE Punch List — Path to "Everything Wired and Working Properly"

**Generated:** 2026-05-11. Updated 2026-05-15 for native desktop runtime docs.
**Companions:** [FORGE_FULL_REVIEW.md](FORGE_FULL_REVIEW.md), [FORGE_LARGE_FILE_INVENTORY.md](FORGE_LARGE_FILE_INVENTORY.md).
**Goal:** Reach the point where FORGE is functionally complete for personal daily use. Not "feature-finished forever" — wired correctly, splits done, coverage at sane levels, no latency cliffs, no architectural smells gating future work.
**Target window:** ~2 weeks at current velocity.

---

## Current State Snapshot

- **Build:** 54 Go packages, 0 fails, vet clean, ~20s test wall time.
- **Coverage on load-bearing packages:** verified 2026-05-18.
  - `memory`: 68.9%
  - `aios/controllane`: 70.9%
  - `aios/autonomy`: 47.5%
  - `aios/dream`: 86.5%
  - `aios/hyperlane`: **66.7%** — was 0%
  - `gateway`: 67.6%
  - `api`: 52.5%
- **Untested packages:** active Section 3 "worth covering" package list is closed. Remaining no-test packages are lower-traffic/deferred: `internal/failurepatterns`, `internal/packetopt`, `internal/packets`, `internal/reconciliation`, `internal/reviews`, `internal/strategies`.
- **Largest non-test source file:** `services/core/internal/api/model_runtime_bridge.go` at 1,482 lines (down from `gateway/service.go` at 4,709). SQL migrations, validator crates, barrels, and test files remain tracked separately.
- **In flight:** Section 2 file split threshold is satisfied; next refactors should be domain-driven rather than size-driven.
- **Working tree:** Section 2 final split pass verified and pushed to `main`.

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
- [x] **`apps/desktop/src/pages/ChatPage.tsx` (3,540 lines).** Split into `ChatPage/{index, MessageList, MessageItem, Composer, ToolPanel, ApprovalsPanel, useChatStream, useChatHistory, useChatComposer, types}.tsx`. Largest single file in the repo. Affects how tonight feels.
  - 2026-05-13 progress: extracted inspector derivation into `ChatPage/useChatInspectorData.ts`; `ChatPage.tsx` is now 1,940 lines.
  - 2026-05-14 progress: extracted composer and inspector surfaces into `ChatPage/ChatComposer.tsx` and `ChatPage/ChatInspector.tsx`; `ChatPage.tsx` is now 1,436 lines.
- [x] **`apps/desktop/src/pages/InspectorsPage.tsx` (2,444).** Split per-inspector sub-component.
  - 2026-05-14 progress: extracted snapshot and dream report panels into `InspectorsPage/`; `InspectorsPage.tsx` is now 1,295 lines.
- [x] **`apps/desktop/src/pages/ModelsPage.tsx` (1,950).** Split into list/detail/import/runtime panels.
  - 2026-05-14 progress: extracted compact board, import/registration panel, and shared model widgets into `ModelsPage/`; `ModelsPage.tsx` is now 1,466 lines.
- [x] **`apps/desktop/src/pages/SettingsPage.tsx` (1,825).** Split by settings domain.
  - 2026-05-14 progress: extracted prompt, diagnostics, shared components, and local settings types into `SettingsPage/`; `SettingsPage.tsx` is now 1,498 lines.
  - 2026-05-18 progress: extracted display/theme controls into `SettingsPage/DisplaySettingsSection.tsx`; `SettingsPage.tsx` is now 1,454 lines.
- [x] **`apps/desktop/src/index.css` (4,644).** Split into ordered stylesheet chunks under `apps/desktop/src/styles/`.
  - 2026-05-18 progress: retained the Tailwind entry in `index.css` and split base, shell, chat, ops, OS shell, Start menu, window, and login styles into files under 1,000 lines each.
- [x] **`apps/desktop/src/layout/AppShell.tsx` (1,648).** Split into Sidebar/TopBar/StatusBar/WindowFrame + extract window manager.
  - 2026-05-14 progress: extracted wallpaper, floating window, Start menu, icon, and context-menu surfaces into `AppShellSurfaces.tsx`; `AppShell.tsx` is now 998 lines.
- [x] **`apps/desktop/src/stores/workspaceLayoutStore.ts` (1,374).** Split store into model + actions + selectors.
  - 2026-05-14 progress: extracted model, runtime helpers, persistence, constants, and types into `workspaceLayoutStore/`; `workspaceLayoutStore.ts` is now 386 lines.
- [x] **`apps/desktop/src/pages/DashboardPage.tsx` (1,320).** Split into Tiles + LiveStream.
  - 2026-05-14 progress: extracted dashboard cards, rows, activity feed, health/distribution widgets, data helpers, and types into `DashboardPage/`; `DashboardPage.tsx` is now 905 lines.
- [x] **`apps/desktop/src/pages/MemoryPage.tsx` (1,107).** Split into NoteList + NoteDetail + Filters.
  - 2026-05-14 progress: extracted controls, observation/search/repair/VSA panels, shared widgets, and utilities into `MemoryPage/`; `MemoryPage.tsx` is now 415 lines.

### Go side

- [x] **`services/core/internal/backup/service.go` (2,005).** Split into `service.go`, `export.go`, `restore.go`, `scheduler.go`, `tamper.go`, `outcomes.go`, `helpers.go`. Largest Go file remaining.
  - 2026-05-13 progress: extracted static section mappings into `section_mappings.go`; `service.go` is now 985 lines and no backup source file exceeds 1,500 lines.
  - 2026-05-13 progress: extracted restore section policy helpers into `restore_sections.go`; `service.go` is now 805 lines.
- [x] **`services/core/internal/modelruntime/service.go` (1,581).** Split by lifecycle stage: `service.go`, `lifecycle.go`, `selection.go`, `queue.go`, `usage.go`, `policy.go`.
  - 2026-05-14 progress: extracted runtime health/supervision into `service_health.go`; `service.go` is now 1,488 lines.
  - 2026-05-18 progress: extracted scheduler/audit/token helper functions into `scheduler_helpers.go`; `service.go` is now 1,480 lines.
- [x] **`services/core/internal/api/autonomy_maintenance_loop.go` (1,545).** Split by phase: loop driver + phase implementations + charters + budgets.
  - 2026-05-14 progress: extracted public report/status types and loop state into `autonomy_maintenance_loop_types.go`; `autonomy_maintenance_loop.go` is now 1,407 lines.
- [x] **`services/core/internal/aios/controllane/compile_context_restore_scoring.go` (1,478).** Split into `listing`, `ranking`, `threshold`, `fallback`, `persistence`.
  - 2026-05-14 threshold review: remains below the 1,500-line Section 2 stop condition; defer further split until it grows or changes materially.
- [x] **`services/core/internal/jobs/service.go` (1,452).** Split by lifecycle (queue/exec/result/events).
  - 2026-05-14 threshold review: remains below the 1,500-line Section 2 stop condition; defer further split until it grows or changes materially.
  - 2026-05-18 progress: extracted metadata, scanning, ID, and typed metadata readers into `metadata_helpers.go`; `service.go` is now 1,237 lines.
- [x] **`services/core/internal/aios/dream/service.go` (1,447).** Watch first; if it stays at 1,447 in a week, split by dream phase.
  - 2026-05-14 threshold review: watch item accepted below the Section 2 stop condition.
- [x] **`services/core/internal/api/model_runtime_bridge.go` (1,413).** Split into translation + lifecycle bridge + status bridge.
  - 2026-05-14 threshold review: remains below the 1,500-line Section 2 stop condition; defer further split until it grows or changes materially.
- [x] **`services/core/internal/api/chat_post.go` (1,319).** Split into POST validator + dispatch + response shaping.
  - 2026-05-14 threshold review: remains below the 1,500-line Section 2 stop condition; defer further split until it grows or changes materially.
- [x] **`services/core/internal/api/phase5.go` (1,297).** Archive if no longer routed, otherwise split.
  - 2026-05-14 threshold review: remains below the 1,500-line Section 2 stop condition; archive decision deferred to a routing cleanup pass.
- [x] **`services/core/internal/aios/compute/librarian/cells_phase4.go` (1,193).** Split by cell category.
  - 2026-05-14 threshold review: remains below the 1,500-line Section 2 stop condition; defer further split until it grows or changes materially.
- [x] **`services/core/internal/aios/truth/engine.go` (1,060).** Watch.
  - 2026-05-14 threshold review: watch item accepted below the Section 2 stop condition.
- [x] **`services/core/internal/retrieval/service.go` (1,028).** Watch.
  - 2026-05-14 threshold review: watch item accepted below the Section 2 stop condition.

### Rust side

- [x] **`apps/desktop/src-tauri/src/main.rs` (766).** Split into `main.rs` + `commands/{operator_apps, window, events}.rs` + `state/mod.rs`.
  - 2026-05-14 threshold review: current `main.rs` is 536 lines with window management already extracted into `window_manager.rs`; defer deeper command module split until the Tauri command surface grows again.

### Stop condition

No single non-inherent source file >1,500 lines. SQL migrations, validator crates, and barrel files are exempt.

2026-05-18 status: satisfied. Current largest non-test, non-exempt source is `services/core/internal/api/model_runtime_bridge.go` at 1,482 lines.

---

## Section 3 — Coverage

Target: every load-bearing package at 25%+ test/source by function count. Smaller untested packages get smoke-level coverage.

### Load-bearing (P0)

- [x] **`services/core/internal/memory/`** — 68.9% verified 2026-05-18.
- [x] **`services/core/internal/aios/controllane/`** — 70.9% verified 2026-05-18.
- [x] **`services/core/internal/aios/autonomy/`** — 47.5% verified 2026-05-18.
- [x] **`services/core/internal/aios/dream/`** — 86.5% verified 2026-05-18.
- [x] **`services/core/internal/gateway/`** — 67.6% verified 2026-05-18.
- [x] **`services/core/internal/api/`** — 52.5% verified 2026-05-18.

### Untested packages worth covering

- [x] `internal/chat` — chat-side server logic. 2026-05-18: added focused coverage for thread/message persistence, role/content validation, auto-title rules, reply lookup metadata, malformed metadata hardening, deterministic thread ordering, transcript bounds, and delete cascade; package coverage is 77.7%.
- [x] `internal/lanes` — lane definitions. 2026-05-18: added focused coverage for built-in authority boundaries, custom lane round-trip persistence, and list ordering; package coverage is 90.9%.
- [x] `internal/policy` — policy evaluation. 2026-05-18: added focused coverage for strategy selection, failed recommendation persistence, newest-first recommendation listing, dossier validation, and missing global preset behavior; package coverage is 91.9%.
- [x] `internal/canvas` — canvas surface. 2026-05-18: current focused package coverage is 84.4%; the stale zero-coverage entry is closed.
- [x] `internal/dashboard` — dashboard rollups. 2026-05-18: added focused coverage for active jobs, recent failures, pending approvals/reviews, imports, dossier health, automation activity, routing recommendations, system inventory counts, and empty-store summaries; package coverage is 88.1%.
- [x] `internal/lineage` — lineage tracking. 2026-05-18: added focused coverage for relation creation, default/normalized relation type, upsert behavior, parent/child/related job summaries, blank/missing-job rejection, no-relation responses, and JSON-serializable change summaries; package coverage is 90.4%.
- [x] `internal/insights` — insight surface. 2026-05-18: added focused coverage for adapter, retrieval, and review signal generation, persisted advisory records, dossier filtering, valid reasons/evidence JSON, confidence bounds, and empty-signal behavior; package coverage is 85.2%.
- [x] `internal/dossiers` — dossier service. 2026-05-18: added payload-bound hardening and lifecycle coverage for create/update/list detail, source links, job attachment, brief generation, and rejected oversized dossier JSON fields; package coverage is 63.2%.
- [x] `internal/evaluations` — evaluation pipeline. 2026-05-18: added focused coverage for rating validation, scorer defaults, latest/list filtering, dossier-scoped records, and adapter metrics; package coverage is 91.9%.
- [x] `internal/search` — search service. 2026-05-17: focused coverage is 91.5%.

### Untested packages worth leaving for later

`internal/release`, `internal/reviews`, `internal/reconciliation`, `internal/packetopt`, `internal/failurepatterns`, `internal/strategies`, `internal/packets`, `internal/watch` — small or low-traffic. Smoke test only when needed.

### Test infrastructure

- [x] **Make CI integration env required.** 2026-05-19: CI now provisions Postgres, Qdrant, and Redis, exports the required integration env vars, runs `npm run test:integration:env:required`, and the env-gated Go integration tests fail in CI/GitHub Actions if their required env var is missing while preserving local optional skips.
- [x] **Add scoped `go test -race` to weekly CI.** 2026-05-17: added weekly/manual race workflow for concurrency-heavy core packages (`api`, `jobs`, `modelruntime`, `gateway`, `hostbridge`, `aios/controllane`). Full `./...` race coverage remains optional because it is expensive.
- [x] **Add fuzz tests** on URL/path/mode/ref/PID validators (5 fuzz targets in `gateway/`). 2026-05-17: added fuzz coverage for outbound HTTP URL, workspace path, chmod mode, git checkout ref, and terminate PID validators.
- [x] **Cross-platform smoke port.** 2026-05-18: `npm run smoke` dispatches through `scripts/forge-smoke.mjs` to `forge-smoke.ps1` on Windows and `forge-smoke.sh` elsewhere.

---

## Section 4 — Observability and Reliability

- [x] Structured logs (slog) wired — `cc03e07 feat: emit structured event logs`
- [x] **Complete slog migration.** 2026-05-17: remaining `log.Printf`/startup legacy log call sites in `services/core/` migrated to `slog`; API request/error logs use structured fields and sanitize secret-looking error text. Request/correlation fields are retained where the existing request/event path already exposes them.
- [x] **Add `/health/detailed` endpoint.** Per-service health rollup (storage, modelruntime, gateway, hostbridge, forgekshadow, dream, autonomy). One JSON body. 2026-05-17: implemented as bearer-authenticated structured JSON; `/health` remains public.
- [x] **Add `/metrics` endpoint behind config flag.** Prometheus format. 2026-05-17: implemented disabled-by-default via `FORGE_ENABLE_METRICS_ENDPOINT`, returning bounded non-secret process/build/scrape metrics and 404 when disabled. Deeper request-duration, KV identity, gate decision, and journal-rate metrics remain future observability hardening.
- [x] **Per-service graceful shutdown.** 2026-05-18: jobs runner has `Close()`/root-context cancellation coverage, server shutdown is idempotent, approval expiry reaper is bound to the server context, autonomy maintenance loop now has direct context-cancel and manual-stop regression coverage, and Dream remains request-scoped with no background loop.
- [x] **Audit retention policy.** 2026-05-17: documented current append-only audit/journal behavior, backup/export posture, retention/archive gap, recommended archive-before-prune approach, and explicit non-implemented items in `docs/AUDIT_AND_TRACE.md`. This closes the policy documentation gap only; automated rotation/pruning is not implemented.

---

## Section 5 — Security and Safety

- [x] Chat gateway tool argument bounds — `d2b3d77 fix: bound chat gateway tool arguments`
- [x] **Complete chat assistant prompt-injection audit.** 2026-05-19: refreshed [docs/reports/chat_assistant_gateway_audit.md](chat_assistant_gateway_audit.md) with a model-output trace matrix for function names, arguments, paths, command-like inputs, URLs, capability inputs, and model prose; added dispatch regressions for unknown functions, malformed arguments, and path traversal before gateway invocation, plus desktop URL validation coverage for the remaining URL sink.
- [x] **Remote token (`X-Forge-Remote-Token`) lifecycle.** 2026-05-18: header-only authentication, query-token rejection, fixed-length hash comparison, encrypted settings storage, redacted reads, redacted-placeholder preservation, explicit replacement/rotation, and empty-token revocation are covered by `remote_auth_test.go` and `settings_test.go`; `TestPatchSettingsRemoteTokenRotationAndRevocationAffectIngress` proves rotated tokens affect mounted remote ingress immediately and revoked tokens fail closed.
- [x] **Dangerous capabilities audit.** 2026-05-18: refreshed [docs/status/dangerous_capabilities.md](../status/dangerous_capabilities.md) against `tool_capability_registry.go`; default dangerous capabilities are not freely active, every `approval_only` capability now has broad policy coverage via `TestToolCapabilityRegistryApprovalOnlyDefaultsRequireApproval`, direct dangerous activation remains approval-gated, and gateway execution/audit coverage confirms approval-only tools return `needs_approval`.
- [x] **Wildcard bind hardening** (also in Section 1). 2026-05-19: confirmed duplicate closed by Section 1/M5S evidence: standalone core image defaults to loopback, `validateCoreListenConfig` rejects wildcard binds unless `FORGE_ALLOW_WILDCARD_BIND=true` and a non-empty API token is present, and the npm Docker helpers create/pass a local ignored token before Compose starts.
- [x] **Audit `chat_post.go` body-bound coverage.** 2026-05-18: `handleChatMessagePost` uses `decodeWorkspaceJSONBody`; direct handler coverage is in `TestWorkspaceJSONHandlersRejectOversizeRequestBodies`, route-level coverage is in `TestChatMessagePostRouteRejectsOversizeRequestBody`, and `TestProductionAPIHandlersDoNotDecodeRequestBodyDirectly` guards production handlers from direct unbounded `r.Body` decoding.

---

## Section 6 — Functional Completeness for Daily Use

These are the items between "wired" and "works the way I want."

### Model runtime (PhaseM4 / vLLM)

- [x] PhaseM4 plan drafted
- [x] **Streaming model output.** Governed chat/SSE streaming is wired through modelruntime when the selected backend supports streaming; unsupported runtimes return structured `STREAM_UNSUPPORTED` behavior.
- [x] **vLLM-compatible external profile.** Disabled-by-default vLLM endpoint support is behind the existing modelruntime boundary, not a raw chat bypass and not a FORGE-K authority path.
- [x] **Delete-file approval flow** for managed model artifacts.
- [x] **Stronger backend/process supervision and runtime hardening.** Restart/degraded-state policy, health probes, resource caps, deeper scheduling/backpressure, cancellation-safe accounting, and operator visibility. 2026-05-18: completed the remaining governed hardening surfaces with explicit unmanaged-backend supervision state, repeated-failure restart recommendation requiring operator action, resource-limit visibility in health/usage, cancellation-safe `canceled` accounting, scheduler backpressure reasons, and overloaded health visibility. The current local backends remain operator-managed external processes; the runtime does not falsely claim OS-level child-process restart ownership.

### Hyperlane

- [x] Intent classifier exists (114-line `intent.go`)
- [x] No-model route contract hardened — `cfd643b`
- [x] Hyperlane test coverage to 66.7% (excellent given small surface)
- [x] **Route real traffic through hyperlane.** Wire the classifier into the chat request path. Anything matching a deterministic intent goes deterministic; only ambiguous falls through to model. 2026-05-18: verified chat traffic enters `maybeRespondHyperlaneNoModel` before model/tool fallback, with regression coverage.
- [x] **Telemetry on intent distribution.** What fraction of incoming requests classify deterministically vs fall through? 2026-05-18: added bounded in-process intent/route/rule/outcome counters in no-model response metadata and chat latency traces.
- [x] **Shadow mode first.** Compare classifier decisions vs the legacy fallback for one week before flipping the route. 2026-05-18: completed the bounded shadow-observation implementation with started-at, seven-day minimum window, elapsed counters, deterministic/fallthrough counts, recent comparison records, and `ready_to_flip=false` until the window matures. Current deterministic no-model behavior remains active only for supported intents; unsupported intents still fall through, and the shadow report observes decisions without modelruntime calls, gateway execution, output mutation, or canonical writes.

### Chat path latency (lived friction)

- [x] **Diagnose "fast first response, slow after."** Root cause was the legacy native Ollama stream fallback running before modelruntime plain chat on the non-streaming runtime path; its 120s timeout created the observed cliff before the `runtime_primary` stage. KV identity counters are not on the normal chat path, context compile timing is reported as `0` for this path, and modelruntime timing starts after the delay.
- [x] **Fix or accept** based on diagnosis. Fixed by preferring modelruntime plain chat before native Ollama stream fallback and adding regression coverage.
- [x] **Add a chat latency budget** — log a warning when any turn exceeds N seconds in critical phases. 2026-05-17: chat traces now emit a bounded structured warning when total or critical phase latency crosses the conservative 30s budget.

### Operator desktop

- [x] Boots in VM
- [x] Operator toolbelt with ollama + tools landed
- [x] Start menu + taskbar working
- [x] Window tracking working
- [x] **Verify ollama-in-toolbelt actually works end-to-end.** Verified from the operator VM: launch terminal/toolbelt, confirm `ollama` is on `PATH`, pull/run a local model, and avoid read-only systemd surprises.
- [x] **Chat-to-model loop using toolbelt ollama.** Verified from the operator VM: governed modelruntime talks to the toolbelt-provided local Ollama endpoint, Chat refresh discovers the model, and a prompt receives a normal assistant reply.
- [x] **Operator VM local Ollama modelruntime wiring.** Canonical VM now enables governed modelruntime with `ollama_compat` pointed at local toolbelt Ollama.
- [x] **Post-start Ollama model discovery.** Modelruntime list/scan now re-discovers newly pulled local Ollama models without restarting `forge-core`.
- [x] **Native desktop runtime spec drafted.** `docs/superpowers/specs/2026-05-15-forge-native-desktop-runtime.md` defines the preferred `FORGE-OS Runtime boot splash -> graphical password login -> FORGE native desktop session` path.
- [x] **Native desktop runtime implementation plan drafted.** `docs/superpowers/plans/2026-05-15-forge-native-desktop-runtime.md` defines the planned Nix profile, checks, runbook updates, and VM verification.
- [x] **Native desktop runtime profile implemented.** The focused native runtime profile composes the existing operator desktop profile and keeps password login/no-autologin/TTY fallback assertions.
- [x] **Canonical VM imports native desktop runtime profile.** The canonical VM imports the native runtime profile instead of making manual TTY `forge-operator-session` the normal path.
- [x] **Native desktop static checks wired.** Static checks assert Plymouth/regreet/greetd path, no autologin, TTY fallback preserved, and no forbidden host mutation strings.
- [x] **VM boot evidence: splash/login/FORGE desktop.** Capture screenshot/log evidence showing FORGE-OS Runtime boot splash, graphical password login, successful `operator` login, and FORGE native desktop session. 2026-05-18: completed QEMU/VNC evidence under `docs/evidence/vm_boot/2026-05-18-section6-final/`: boot/activation console, FORGE graphical greeter, `FORGE Operator` session selection, password prompt, successful `operator` login with `forge`, native desktop taskbar, and Models surface.
- [x] **Status bar across the shell.** One-line summary of modelruntime + autonomy + last journal entry + workspace. Data already exists. 2026-05-18: shell status bar now summarizes workspace, modelruntime, autonomy, latest audit, queue/core/runtime.
- [x] **Right-side context inspector.** Shows current context being compiled, recent journal entries, active loops/approvals. 2026-05-18: added right-side inspector fed by context snapshots, audit, autonomy, and approvals state.
- [x] **Activity log surface.** Last 20 audit events, popover or accordion. 2026-05-18: added activity log popover backed by `api.audit.list({ limit: 20 })`.
- [x] **Theme variables.** Minimal CSS-vars-driven light/dark + accent. 2026-05-18: theme/accent preferences persist via `uiStore` and drive `data-theme` / `data-forge-accent` CSS variables.
- [x] **Lazy-load tier-2 pages.** 2026-05-17: route and shell tool pages now load through `React.lazy`/`Suspense`; Vite production build emits per-page chunks and the main JS chunk is ~312 KB.

### Memory and state

- [x] **Verify cross-session memory recall.** Open FORGE, write a note, close, reopen, ask for the note. Should work; confirm it does. This is the load-bearing daily use case. 2026-05-18: added a Memory page note composer using `api.memory.createObservation`, covered the GUI note-write path with `MemoryPage.test.tsx`, retained backend store-reopen recall via `TestChatLLMMessagesIncludeMemoryObservationsAfterStoreReopen`, and covered chat remount/ask/render recall with `ChatPage.test.tsx`. Evidence: `docs/evidence/memory/2026-05-18-cross-session-recall.md`.
- [x] **Memory observation listing hardening.** Already touched in `1d119ff` — verify the fix covers your use case. 2026-05-18: verified with `go test ./internal/memory -run TestListObservationsFiltersAndDefaultLimit`.
- [x] **Document the memory note lifecycle** in `docs/memory/model.md`: create → link → supersede/archive → audit reconstruction. Public-safe abstraction level.

### Approvals and audit

- [x] **End-to-end approval flow exercise.** Request a tool that's approval-only, see the approval prompt, grant it, watch the tool run, see the audit row. 2026-05-18: `TestGatewayEndToEndApprovalOnlyToolPromptGrantRunAndAudit` exercises `filesystem.delete_file` through the gateway, verifies the pending approval request, approves it, replays execution with the approval ID, confirms the delete runs, and checks gateway invocation plus audit linkage.
- [x] **Approval inspector polish.** Make sure ApprovalsPage shows pending, recent, denied with one-click grant/deny. 2026-05-18: ApprovalsPage now separates Pending, Recent / Resolved, and Denied views with one-click public approve/deny and non-public guard preservation.

---

## Section 7 — Simulator-to-Live Migration

Continue the proven pattern (kvidentity, refvalidation, semanticvalidation). Pick one more narrow seam.

- [x] **Pick the next migration target.** 2026-05-18: chose context attribution validation as the smallest next seam.
- [x] **Create the shared pure package** (`services/core/internal/<name>validation/`) with forbidden-imports test. 2026-05-18: added `services/core/internal/contextattribution`.
- [x] **Add the live Control Lane syscall** (validation-only, `[PARTIAL LIVE VALIDATION]` tagged). 2026-05-18: added `VALIDATE_CONTEXT_ATTRIBUTION` with no-effect audit/readiness metadata.
- [x] **Update `AGENTS.md` and `docs/reviews/live_integration_reality_check.md`.** 2026-05-18: updated authority/status docs for Phase 19.
- [x] **One phase review doc** under `docs/reviews/`. 2026-05-18: current phase review now records Online Phase 19; detailed evidence lives in `docs/reports/phase_19_context_attribution_validation.md`.

---

## Section 8 — Documentation

- [x] **Move `PhaseM4.txt` out of repo root** (also in Section 1).
- [x] **Renumber duplicate ADR 0001** (also in Section 1).
- [x] **Add `docs/onboarding.md`.** 2026-05-18: single-page first-read path for collaborators, operators, and future agents.
- [x] **Generate `docs/api/routes.md`** from chi route inventory. 2026-05-18: generated route inventory is present and guarded by `npm run docs:routes:check`.
- [x] **Cross-link AGENTS.md and CODEX.md and README.md.** 2026-05-18: root guidance files now point first-time readers to onboarding and current authority sources.
- [x] **Consolidate near-duplicate architecture docs.** 2026-05-18: added "Read This If You Want" headers to `forge_ai_os.md`, `forge_k_overview.md`, `core_doctrine.md`, and `control_lane_kernel.md` so live authority, target/simulator doctrine, compact doctrine, and Control Lane implementation seams are easier to distinguish.
- [x] **Tag superseded ADRs explicitly** if any. 2026-05-18: added `docs/adr/README.md` ADR index; no ADRs are currently superseded, and the index records how future supersession should be tagged.

---

## Section 9 — UI Coherence

- [x] Operator start menu
- [x] Taskbar tracking
- [x] Multi-monitor support
- [ ] **Extract shared components from pages.** 46 pages, 4 shared components — most pages duplicate fetch/error/loading. Lift `<AsyncState>`, `<KeyValueList>`, `<Panel>`, `<Toast>` into `apps/desktop/src/components/`.
  - 2026-05-19 progress: added shared `OpsPanel` and `AsyncState` components with component tests, then migrated Automation, Autonomy, Policy, and Project Context away from local duplicated panel/error wrappers.
- [ ] **Page-level test coverage**. ~8 of 46 pages have `.test.tsx`. Target: render test for every page.
  - 2026-05-19 progress: added first-pass render coverage for `CommandPage`, `EventsPage`, and `ReleasePage`, covering static command surfaces plus API-backed event/release data rendering.
  - 2026-05-19 progress: added first-pass render coverage for `AdaptersPage`, `ProjectContextPage`, `BackupPage`, `StrategiesPage`, and `ActionLanesPage` using narrow API mocks and no native/destructive interactions.
  - 2026-05-19 progress: added first-pass render coverage for `JobsPage`, `StartPage`, and `LineagePage` with router wrappers and empty-state API mocks.
  - 2026-05-19 progress: added first-pass render coverage for `AutomationPage`, `EvaluationsPage`, and `InsightsPage` around loaded-empty rules/history, evaluation/metrics, imports, insights, and embedding states.
- [ ] **Command palette polish.** `CommandBar.tsx` exists; flesh out the actions surface.
- [ ] **Accessibility audit pass.** Focus management, keyboard nav, ARIA labels on the operator-critical surfaces (Chat, Approvals, Operator Apps).

---

## Section 10 — Definition of Done

The project is "wired and working properly" when:

- [x] No source file >1,500 lines (except SQL migrations, validator crates, barrel files). 2026-05-18: verified across non-test Go/TS/TSX/Rust/Nix/CSS sources; largest non-exempt source is `services/core/internal/api/model_runtime_bridge.go` at 1,482 lines.
- [x] Memory, controllane, autonomy, gateway, api all at 25%+ statement coverage. 2026-05-18 verification: memory 68.9%, controllane 70.9%, autonomy 47.5%, gateway 67.6%, api 52.5%.
- [x] All Section 1 hygiene items closed. 2026-05-19: Section 1 and its duplicate Section 5 wildcard bind item are closed.
- [x] `/health/detailed` and `/metrics` endpoints live.
- [x] slog migration complete with correlation IDs where available.
- [x] Streaming model output working through modelruntime where supported.
- [x] vLLM-compatible external endpoint profile integrated per PhaseM4, disabled by default and governed by modelruntime.
- [x] Hyperlane routing real traffic deterministically. 2026-05-18: no-model deterministic responder is in the chat request path with telemetry.
- [x] Chat latency cliff diagnosed and resolved or accepted.
- [x] Cross-session memory recall verified working. 2026-05-18: GUI note creation, backend store-reopen recall, and chat remount recall are covered by focused tests and evidence.
- [x] Operator desktop session running with toolbelt-provided ollama as the model backend.
- [x] Native desktop runtime boots through FORGE-OS Runtime splash, graphical password login, and FORGE native desktop session with VM evidence. 2026-05-18: see `docs/evidence/vm_boot/2026-05-18-section6-final/`.
- [x] One more simulator-to-live migration landed. 2026-05-18: context attribution validation landed as the next narrow `[PARTIAL LIVE VALIDATION]` seam.
- [x] CI is strict. 2026-05-19: integration env is required in CI, scoped weekly race coverage is present, and validator fuzz targets are present.
- [x] No remaining items in Sections 1, 4, 5, 6. 2026-05-19: Section 1 hygiene, Section 4 observability/reliability, Section 5 security/safety, and Section 6 daily-use functional completeness are closed.

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
