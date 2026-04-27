# FORGE Full Review + Punchlist

Status date: 2026-04-26

This is the consolidated one-file version of the FORGE punchlist review. It combines the executive summary, repo inventory, validation baseline, phase status, subsystem reviews, risk register, full punchlist, next passes, deferred ideas, open questions, and CSV-equivalent punchlist content.

## Review Status

GOOD: Review completed as a current-state, read-only pass. No runtime code was changed.

PARTIAL: `npm run smoke` could not run on this Windows environment because the script invokes `bash ./scripts/forge-smoke.sh` and `/bin/bash` is unavailable.

## Sources Used

- `AGENTS.md`
- `README.md`
- `docs/architecture/forge_ai_os.md`
- `docs/architecture/semantic_syscalls.md`
- `docs/data_model/cognitive_filesystem.md`
- `docs/roadmap/forge_ai_os_phases.md`
- requested `docs/status/*` files
- `docs/reviews/forge_full_system_review_20260425/*`
- `docs/journal/forge_master_technical_journal_20260425/*`
- static code inspection across `services/core`, `apps/desktop`, `packages`, and `scripts`
- parallel reviewer reports for authority, persistence, modelruntime, API/UI, architecture/autonomy, and security/testing/performance

MISSING: No binding source listed by the prompt was missing.

## Executive Summary

GOOD: FORGE has a real CPU/RAM core, semantic syscall control lane, durable cognitive filesystem tables, governed gateway, governed modelruntime management surface, deterministic context restore scoring, Dream Mode v0, Dream report persistence, backup coverage, and a desktop operator surface.

PARTIAL: Several features are implemented but not converged into a final operating model. The largest gaps are around security hardening, operator diagnostics, exact restore candidate behavior, frontend test discipline, modelruntime M4, and Rule Cell/Hyperlane substrate.

RISK: A few authority-adjacent APIs can still reshape runtime policy without strong approval/audit semantics: gateway capability status, lanes, and permission profile management.

BROKEN: Windows smoke validation is blocked by the Bash-only smoke script.

## Top 10 Findings

1. RISK: Core HTTP binds `":" + port`, exposing local APIs beyond loopback if the network/firewall allows it.
2. RISK: Gateway capability status changes can activate dangerous capabilities without a dedicated approval gate, and the API reason is not preserved by the gateway update call.
3. RISK: Telegram polling path appears to allow normal remote chat from any sender once remote polling is enabled; wake commands are allowlisted, normal messages appear weaker.
4. RISK: Context restore SQLite candidate listing exact-filters by query before scoring, undercutting documented lexical/near-match ranking.
5. RISK: Backup restore reports checksums/counts but does not clearly fail closed before mutation on bundle tampering or count mismatch.
6. RISK: SSRF/private-network denial and symlink traversal coverage is missing for network/file/archive/model/backup paths.
7. PARTIAL: Modelruntime M3 is real, but M4 remains substantial: streaming, process supervision, delete-file approval flow, durable scheduler state, and remote provider budget/cost policy.
8. PARTIAL: Dream reports persist as non-canonical evidence, but operator review/apply is intentionally absent and same-ID report persistence is upsert-style rather than append-only.
9. PARTIAL: UI inspector surfaces exist, but global degraded/runtime state is fragmented and the shell runtime pill is not modelruntime/safe-mode aware.
10. MISSING: Rule Cell/Hyperlane is still concept/scaffold, not an implemented deterministic low-latency rule substrate.

## Top 10 Punchlist Items

1. `AUTH-001`: Govern gateway capability status changes. Fixed in follow-up pass: high-risk elevation now requires matching approval, and actor/reason/provenance metadata is persisted.
2. `SEC-001`: Bind core to loopback by default and document/configure public binding explicitly.
3. `GATE-001`: Add SSRF/private-network denial for network tools and provider endpoints.
4. `CTX-001`: Fix restore candidate retrieval to rank near matches.
5. `DUR-001`: Make backup checksum/entity-count verification fail closed before restore mutation.
6. `SEC-003`: Add symlink/archive/path traversal fixture suite.
7. `UI-001`: Add global operator diagnostics and accurate safe-mode/modelruntime state.
8. `MR-001`: Implement modelruntime streaming and cancellation-safe accounting.
9. `TEST-001`: Add JS/TS test and lint lanes; replace or wrap Bash smoke for Windows.
10. `RULE-001`: Either implement Rule Cell/Hyperlane v0 or keep it explicitly concept-only in status docs.

## Next 3 Recommended Passes

1. Authority hardening: capability status governance plus lane/profile audit.
2. Security boundary hardening: loopback binding, SSRF denylist, symlink/path traversal tests.
3. Restore correctness: near-match candidate retrieval, matching indexes, and restore inspector UI scope cleanup.

## Repo Inventory

GOOD: The repo has a clear product split:

- `services/core`: Go core service, API server, SQLite persistence, AI-OS/control lane, gateway, approvals, modelruntime, backup, memory/retrieval, autonomy, Dream Mode.
- `apps/desktop`: Tauri + React desktop shell.
- `packages/shared`: shared TypeScript contracts.
- `packages/ui`: small shared UI primitives.
- `docs`: architecture, status, runbooks, reviews, and journal.
- `scripts`: Windows/Node/Bash launch, smoke, VSA preflight, desktop port/deps checks.
- `nix`: optional Nix substrate.

Main entrypoints:

- Core service: `services/core/main.go`
- API router: `services/core/internal/api/server.go`
- Desktop app: `apps/desktop/src/main.tsx`, `apps/desktop/src/App.tsx`
- Tauri shell: `apps/desktop/src-tauri/src/main.rs`
- Root commands: `package.json`

Runtime processes:

- `forge-core`: Go HTTP API and kernel/runtime authority.
- Desktop shell: Tauri/Vite React client.
- Optional external/model backends: Ollama, llama.cpp endpoint, OpenAI-compatible endpoint, vLLM-compatible endpoint, TEI, DCGM, Level Zero.

API route groups:

- `/health`
- `/api/*`: settings, remote, sources, events, providers, autonomy, Dream, adapters, jobs, chat, canvas, artifacts, approvals, packets, project context, context inspectors, memory, dossiers, gateway, permissions, audit, backup, release, commands.
- `/forge/models*` and `/forge/model-runtime/*`: modelruntime management/inference.
- `/v1/*`: OpenAI-compatible minimum routes.

Authoritative seams:

- Semantic writes: `services/core/internal/aios/controllane`
- Tool execution: `services/core/internal/gateway`
- Approvals: `services/core/internal/approvals`
- Permissions/capabilities: `services/core/internal/permissions`, `services/core/internal/gateway/tool_capability_registry.go`
- Audit/trace: `services/core/internal/audit`, `services/core/internal/api/trace_report.go`
- Modelruntime: `services/core/internal/modelruntime`
- Backup/export/restore: `services/core/internal/backup`

Known compatibility/legacy seams:

- Legacy adapter direct route is retired; `legacy.adapter.invoke` remains gateway-wrapped.
- Legacy memory observation mutation endpoints are gated/retired.
- `events` operational stream coexists with canonical `journal_events`.
- `compute/librarian` runtime and `computelane` interfaces both exist.
- VSA/vector records are retrieval support, not truth authority.

Suspected duplicate systems:

- PARTIAL: `events` vs `journal_events` still needs consistent language and operator surfacing.
- PARTIAL: Some TypeScript contracts for modelruntime, Dream reports, restore inspector, and process health live in `apps/desktop/src/lib/api.ts` instead of `packages/shared`.

## Validation Results

`npm install` was not run because `node_modules` and `package-lock.json` are present.

| Command | Result | Notes | Confidence Impact |
|---|---:|---|---|
| `npm run build` | PASS | Desktop Vite build + VSA preflight + Go build passed. Vite emitted chunk-size warning. | High |
| `npm run build:core` | PASS | VSA tracked-file preflight + `go build ./...`. | High |
| `npm run build:desktop` | PASS | Vite build passed; chunk-size warning. | Medium |
| `npm run typecheck` | PASS | Desktop TypeScript typecheck passed. | Medium |
| `npm run validate:desktop` | PASS | Desktop typecheck + build passed. | Medium |
| `npm test` | PASS | VSA preflight + `cd services/core && go test ./...`. | High |
| `npm run lint` | PASS | Root lint delegates to `go vet ./...`; no JS lint configured. | Medium |
| `cd services/core && go test ./...` | PASS | Full Go test suite passed. | High |
| `cd services/core && go vet ./...` | PASS | Go vet passed. | High |
| `npm run smoke` | FAIL | Environment/tooling: invokes `bash ./scripts/forge-smoke.sh`; Windows shell reports `/bin/bash` missing. | Medium |

GOOD: Core build/test/vet and desktop build/typecheck are green.

RISK: Root `lint` is Go-only. There is no dedicated JS/TS lint lane.

RISK: No frontend test suite is configured.

BROKEN: Smoke is Bash-only and fails on this Windows machine. This is not evidence that core smoke behavior is broken, but it blocks Windows validation confidence.

## Phase Status Matrix

| Phase / Pass | Status | Classification | Evidence | Remaining Gate |
|---|---|---|---|---|
| Phase 1 foundation | complete | accurate | `AGENTS.md`, architecture docs | Keep docs/current code aligned. |
| Phase 2 semantic syscall kernel | mostly complete | accurate | `aios/controllane/processor.go`, `validator.go`, tests | Public syscall API and more edge tests. |
| Phase 3 cognitive filesystem | partial / mostly complete | slightly understated in older docs | `store/migrate.go`, `sqlite_store.go` | Immutability decisions, backup hard gates. |
| Phase 4 ingest/librarian cells | mostly complete | accurate | `aios/compute/librarian/*` | Lane isolation and operator inspection. |
| Phase 5 truth engine | partial | accurate | `aios/truth/engine.go` | Repair/apply/operator surfacing. |
| Phase 5.5 rule agents | partial | accurate | `aios/autonomy/rule_agents.go` | Expand deterministic coverage or fold into Rule Cells. |
| Phase 5.75 autonomy | partial implemented | accurate | `aios/autonomy/*`, SQLite repos | Tool-call budget consumption and trace visibility. |
| Phase 5.9 gateway/tool policy | mostly complete | accurate | `gateway/*`, capability registry | Capability status governance and service-specific tests. |
| Phase 5.95 runtime authority cutover | mostly complete | accurate | retired legacy adapter route, gateway ingress | Operator trace cohesion. |
| Phase 5.997 convergence | mostly complete | stale in older review docs | current code includes recent passes | Update stale reviews/status docs. |
| Phase 6.25 context restore | mostly complete | overstated in candidate retrieval docs | scoring exists, exact-query SQL prefilter remains | Near-match candidate retrieval and indexes. |
| Phase 6.35 Dream Mode v0 | partial implemented | older docs stale | `dream_reports`, inspector API, backup coverage now exist | Operator apply/review not implemented by design. |
| Approval fingerprint hardening | implemented | implemented but older docs stale | gateway/approval/modelruntime tests | Shared permission semantics still need hardening. |
| Model management governance | implemented / partial | implemented but M4 remains | `api/model_runtime_governance.go`, tests | Cost/budget/provider policy, streaming, supervision. |
| Backup/restore parity | mostly implemented | accurate with caveats | `backup/service.go`, tests | Hard checksum/count fail-closed preflight. |
| Context restore scoring fix | partial | documented stronger than code | `compile_context_restore_scoring.go` | SQL candidate retrieval. |
| Dream report persistence | implemented | implemented but older docs stale | `dream_reports`, API tests | Same-ID upsert/append-only decision. |
| Operator restore/Dream inspector | implemented / partial | current | `operator_inspector.go`, `InspectorsPage.tsx` | Global diagnostics and UI state polish. |
| Rule Cell / Hyperlane v0 | concept/scaffold | planned only | journal docs | First deterministic router/rule registry pass. |
| Modelruntime M1 | mostly complete | accurate | manifest/store/registry/backends | Keep under governance. |
| Modelruntime M2 | mostly complete | accurate | scheduler, queue, usage | Durable scheduler/timeout behavior. |
| Modelruntime M3 | partial implemented | accurate | management APIs, governance tests | M4 items. |
| Modelruntime M3.5 | partial | accurate | DCGM, Level Zero, TEI diagnostics | Provider costs, LoRA, embedding refresh governance. |
| Modelruntime M4 | not implemented | accurate | no streaming/process supervision | Dedicated implementation passes. |
| Nix N1 | partial | not verified | `flake.nix`, docs | Run `nix flake check` where Nix daemon exists. |

## Authority Boundary Review

Scorecard:

- Semantic syscall kernel: GOOD
- Gateway-only tool execution: GOOD
- Approval fingerprinting: GOOD/PARTIAL
- Model management governance: GOOD/PARTIAL
- Capability/lane/profile governance: RISK
- Remote ingress authority: RISK

Findings:

- GOOD: Canonical semantic mutations use `aios/controllane` registry, validator, processor, transaction runner, journal/audit linkage, and SQLite store.
- GOOD: Legacy direct adapter API route is not registered; adapter execution is gateway-wrapped.
- GOOD: Gateway approval fingerprinting exists and tests reject mismatched replay.
- RISK: Gateway capability status updates can reshape dangerous tool posture without a dedicated high-risk approval gate. `handleGatewayCapabilityStatusUpdate` requires reason for some changes, but `Gateway.UpdateCapabilityStatus` persists via a path that does not preserve that reason in the override table.
- RISK: Lane and permission profile save/delete paths are authority-shaping but not consistently immutable-audit-backed.
- RISK: `permissions.Check` can lift soft gates for a granted job approval. Gateway later checks approval fingerprints, but direct non-gateway callers could rely on the broader permission decision.
- RISK: Telegram polling wake commands are allowlisted, but normal remote messages appear to lack sender/chat allowlist enforcement once remote polling is enabled.
- PARTIAL: Dream Mode is proposal/dry-run-only and Dream reports are non-canonical evidence. No commit/apply path exists, which is correct for current doctrine.

Punchlist:

- `AUTH-001`: Route capability status changes through governance and persist actor/reason.
- `AUTH-002`: Add sender/chat allowlist enforcement for Telegram remote message processing.
- `AUTH-003`: Add immutable audit records for lane/profile save/delete.
- `AUTH-004`: Make approval fingerprint validation shared or prevent broad direct permission lifting.
- `AUTH-005`: Add regression test that legacy adapter invoke remains unrouted.

## Durability / Backup / Restore Review

Scorecard:

- Cognitive filesystem tables: GOOD/PARTIAL
- SQLite migrations: GOOD/PARTIAL
- Backup section coverage: GOOD
- Restore integrity gates: PARTIAL/RISK
- VSA/vector posture: GOOD/PARTIAL

Findings:

- GOOD: Core cognitive tables exist for journal, memory notes, semantic links, state, versions, loops, artifacts, derived models, contradictions, supersessions, context snapshots, and Dream reports.
- GOOD: Full backup includes cognitive filesystem, gateway/audit traces, approvals/capabilities, modelruntime state, context snapshots, Dream reports, and export-only VSA sections.
- RISK: Restore computes/records row counts and checksums, but review did not confirm a fail-closed pre-mutation gate on mismatched bundle checksums/entity counts.
- RISK: `RestoreBundle` accepts a file path. This needs path-boundary tests and possibly stronger policy gating.
- PARTIAL: Only `journal_events` has hard append-only DB triggers. Other documented evidence/history rows need an explicit immutability decision: `state_versions`, `context_packet_snapshots`, `contradiction_records`, `supersession_records`, `artifact_refs`, and `dream_reports`.
- PARTIAL: Dream report persistence uses upsert by report ID. That is deterministic, but not append-only evidence.
- PARTIAL: VSA sections are export-only/rebuildable, while `embedding_records` are restored as retrieval index. The distinction should remain explicit.

Punchlist:

- `DUR-001`: Fail closed on backup checksum/entity-count mismatch before restore mutation.
- `DUR-002`: Add restore path-boundary and approval/policy tests.
- `DUR-003`: Decide and enforce immutability posture for evidence/history tables.
- `DUR-004`: Add FTS/index rebuild verification after restore.
- `DUR-005`: Document `embedding_records` restore policy vs VSA export-only policy.

## Context Restore Review

Scorecard:

- `COMPILE_CONTEXT` compatibility: GOOD
- Snapshot persistence: GOOD
- Deterministic scoring: GOOD/PARTIAL
- Candidate listing: RISK
- Header-first package: GOOD/PARTIAL
- Operator inspectability: GOOD/PARTIAL
- Performance/index posture: PARTIAL

Findings:

- GOOD: Restore scoring is deterministic, CPU-only, and non-LLM. It stores `restore_scores_json`, `resume_hints_json`, restore trace/package metadata, and non-canonical snapshot evidence.
- GOOD: Operator routes exist for recent restore snapshots, detail, candidates, score, and resume hints.
- RISK: SQLite candidate listing filters by exact `query = ?` before lexical scoring. This excludes near-match candidates before `scoreRestoreCandidate` can rank them.
- RISK: Existing indexes do not fully match the intended restore candidate access pattern if exact-query filtering is removed.
- PARTIAL: Generic snapshot detail route allows optional scope; strict restore routes require workspace. Desktop generic snapshot detail should pass workspace/lane or use strict restore routes for restore views.
- MISSING: Restore outcome feedback loop is not present.

Punchlist:

- `CTX-001`: Change candidate listing to fetch by workspace/lane/kind/recency and score query similarity in Go.
- `CTX-002`: Add composite index for restore candidate scans.
- `CTX-003`: Add tests for near-match candidates and wrong-workspace exclusion.
- `CTX-004`: Pass workspace/lane into desktop snapshot detail calls.
- `CTX-005`: Add restore outcome feedback table or report-only evidence path.
- `CTX-006`: Add benchmark/fixture for large snapshot sets.

## Dream Mode Review

Scorecard:

- Replay selector: GOOD/PARTIAL
- Salience scoring: GOOD
- Tier-routing proposals: GOOD
- Dry-run default: GOOD
- Report persistence: GOOD/PARTIAL
- Backup inclusion: GOOD
- Operator inspectability: GOOD/PARTIAL
- Commit boundary: GOOD

Findings:

- GOOD: Dream Mode v0 reads existing cognitive filesystem tables and produces deterministic replay/salience/routing proposals without modelruntime/GPU dependency.
- GOOD: `/api/dream/run` remains dry-run/proposal-first. `persistReport=true` stores a non-canonical `dream_reports` row.
- GOOD: Read-only report list/get/candidates/proposals/warnings routes exist and are workspace scoped.
- GOOD: Backup includes `dream_reports`.
- PARTIAL: Persistence is opt-in; if operators do not request `persistReport=true`, reports remain transient.
- PARTIAL: Same report ID persistence is upsert-style. Decide whether this is desired deterministic replacement or an evidence immutability issue.
- MISSING: No Dream proposal apply/commit path exists. This is correct now; future commit mode must be syscall-bound.
- MISSING: Dream report outputs do not yet feed restore scoring outcome feedback.

Punchlist:

- `DRM-001`: Decide and document Dream report upsert vs append-only behavior.
- `DRM-002`: Add operator workflow for report review without apply.
- `DRM-003`: Add future governed Dream apply syscall design only after evidence review UI stabilizes.
- `DRM-004`: Add restore feedback signals from Dream reports as non-canonical evidence.
- `DRM-005`: Add test proving Dream runs with modelruntime/GPU unavailable.

## Rule Cells / Hyperlane Review

Scorecard:

- Rule agents: PARTIAL
- Ingest cells: PARTIAL/GOOD
- Rule Cell substrate: MISSING
- Hyperlane router: MISSING
- No-mutation behavior: GOOD for existing rule agents

Findings:

- GOOD: Current autonomy rule agents are propose-only and have destructive guards.
- GOOD: Librarian ingest cells are real and generate proposals routed through control-lane authority.
- PARTIAL: Existing rule agents are not the planned Rule Cell/Hyperlane substrate.
- MISSING: There is no deterministic rule registry with lane/phase filtering, latency budgets, trace envelopes, disabled-rule handling, or starter rule packs.
- MISSING: Hyperlane remains a journal/design concept, not runtime code.

Punchlist:

- `RULE-001`: Either implement Rule Cell/Hyperlane v0 or keep all docs/status clearly concept-only.
- `RULE-002`: Add deterministic rule registry with priority, enabled/disabled state, lane/phase filters, and trace output.
- `RULE-003`: Add starter rule pack for restore hygiene, Dream review, gateway safety, and operator hints.
- `RULE-004`: Add latency budget tests and no-mutation tests.
- `RULE-005`: Consider folding current autonomy rule agents into the future Rule Cell substrate.

## Modelruntime / Provider Review

Scorecard:

- Registry/import/register/reconcile: GOOD
- Lifecycle management: GOOD/PARTIAL
- Governance/approval/capability: GOOD/PARTIAL
- Backend selection: PARTIAL
- Scheduler/queue: PARTIAL
- OpenAI-compatible path: PARTIAL
- vLLM path: PARTIAL
- DCGM/Level Zero/TEI: PARTIAL
- Safe degraded mode: GOOD

Findings:

- GOOD: Modelruntime M3 management exists with registry, store, import, scan, verify, enable/disable/archive/remove, load/unload, compatibility, usage, health, queue, and loaded-status surfaces.
- GOOD: High-risk model operations have governance/approval/capability handling and tests.
- RISK: Streaming is not implemented; streaming requests are rejected.
- RISK: vLLM path is OpenAI-compatible HTTP shape, not full vLLM lifecycle/process supervision.
- RISK: GPU-required admission checks config posture more than real telemetry availability.
- PARTIAL: Scheduler is in-memory/simple FIFO; restart durability and dispatch timeout enforcement need work.
- RISK: Archive path removes existing archive destination before rename; approval-gated at API level, but the primitive is destructive if called incorrectly.
- MISSING: Cost/budget/egress governance for remote/cloud provider inference.

M4 punchlist:

- `MR-001`: Implement streaming with cancellation-safe audit/usage.
- `MR-002`: Add backend process supervision for llama.cpp/vLLM.
- `MR-003`: Split remove registration, archive, and delete-file approval flow.
- `MR-004`: Make GPU-required admission depend on actual telemetry/backend availability.
- `MR-005`: Add durable scheduler state or explicitly document restart loss.
- `MR-006`: Add provider cost/budget/egress policy.
- `MR-007`: Add gateway `model.*` aliases only after governance semantics are stable.

## Gateway / Approval / Tool Review

Scorecard:

- Gateway-only execution: GOOD
- Tool taxonomy: GOOD
- Capability defaults: GOOD/PARTIAL
- Approval gates: GOOD/PARTIAL
- Approval fingerprinting: GOOD
- Dangerous defaults: GOOD/PARTIAL
- Service-specific tests: PARTIAL

Findings:

- GOOD: Gateway has a governed tool registry, policy checks, invocation records, audit/trace propagation, and approval fingerprint binding.
- GOOD: Dangerous tools are generally approval-only or guarded by capability status.
- RISK: Capability status management can change dangerous tool posture without high-risk approval semantics.
- RISK: `net.fetch` and network connectivity tools need SSRF/private-network denial.
- RISK: `proc.run` buffers stdout/stderr before final truncation, creating memory-pressure risk.
- PARTIAL: Service-specific harness tests exist, but coverage should expand for archive extraction, symlink paths, network denial, and output limits.

Punchlist:

- `GATE-001`: Add high-risk approval gate for dangerous capability activation.
- `GATE-002`: Add SSRF denial tests and implementation.
- `GATE-003`: Bound process output before buffering in memory.
- `GATE-004`: Add symlink traversal tests for file tools.
- `GATE-005`: Add route test proving direct adapter execution remains unavailable.

## API / UI / Operator Review

API scorecard:

- Health routes: GOOD/PARTIAL
- Restore inspector routes: GOOD
- Dream inspector routes: GOOD/PARTIAL
- Process routes: PARTIAL
- Websocket: MISSING / not found
- SSE chat stream: GOOD/PARTIAL

UI scorecard:

- Operator inspectability: PARTIAL
- Empty states: PARTIAL/GOOD
- Degraded modelruntime/no-GPU visibility: PARTIAL
- Dual-mode direction: PARTIAL
- Frontend contract centralization: PARTIAL

Findings:

- GOOD: Restore and Dream inspector APIs exist and expose non-canonical evidence.
- GOOD: Desktop has a compact Inspectors page with snapshots, Dream reports, packet inspection, trace, and process trace.
- RISK: App shell runtime pill is not modelruntime/safe-mode/GPU-aware and can imply online while subsystems are degraded.
- PARTIAL: `/api/process/health` is trace-scoped process health, not global process health.
- PARTIAL: Dream list/get and Dream subresource responses use different envelope styles.
- PARTIAL: Model lifecycle UI does not clearly show governance state: direct, approval required, approval pending, dry-run, blocked, unavailable.
- PARTIAL: Many desktop API contracts live locally in `apps/desktop/src/lib/api.ts` instead of `packages/shared`.
- MISSING: No WebSocket implementation found; chat streaming appears SSE-only.

Punchlist:

- `UI-001`: Make shell runtime state reflect `/health`, modelruntime, safe mode, GPU, and embedding provider state.
- `UI-002`: Add global diagnostics/operator state page or endpoint.
- `UI-003`: Rename UI language around process trace health unless a global process endpoint is added.
- `UI-004`: Pass workspace/lane into generic snapshot detail calls.
- `UI-005`: Standardize evidence badges across Dream/restore views.
- `UI-006`: Surface model action governance state in Models UI.
- `UI-007`: Move stable API contracts into `packages/shared`.

## Security / Safety Review

| ID | Severity | Risk | Affected Modules | Recommendation |
|---|---:|---|---|---|
| SEC-001 | high | Core binds all interfaces by default. | `services/core/main.go` | Default to loopback; require explicit public bind config. |
| SEC-002 | high | SSRF/private-network egress controls incomplete. | gateway network tools, provider endpoints | Deny metadata, loopback, link-local, private ranges unless explicitly allowed. |
| SEC-003 | high | Symlink/path traversal test coverage incomplete. | gateway file tools, artifacts, backup, model import, archive extraction | Add fixture suite and enforce resolved-path containment. |
| SEC-004 | high | Backup restore accepts arbitrary file path. | `backup/service.go`, API restore | Add boundary tests and policy gate. |
| SEC-005 | high | Telegram polling normal chat allowlist gap. | remote Telegram service | Enforce sender/chat allowlist. |
| SEC-006 | medium | CORS/local API posture depends on bind assumptions. | API server | Add CORS matrix tests. |
| SEC-007 | medium | Secret redaction coverage thin outside `secret.get`. | config/provider/remote diagnostics | Add redaction tests. |
| SEC-008 | medium | Process execution output can grow before truncation. | gateway process tool | Stream/cap buffers. |
| SEC-009 | medium | Workspace root defaults broad. | config/workspace path policy | Better first-run warning and tests. |

GOOD: Dangerous tools are generally gateway/approval/capability governed.

GOOD: Dream Mode and context snapshots are non-canonical evidence.

RISK: Network, filesystem, and remote ingress boundaries need adversarial tests before wider exposure.

## Performance / Latency Review

Risks:

- RISK: Restore scoring may degrade with many snapshots if candidate listing broadens without matching indexes.
- RISK: `proc.run` captures output into memory before final truncation.
- RISK: Modelruntime scheduler is simple in-memory FIFO with limited durable accounting.
- RISK: Provider health/telemetry calls can become visible latency if pulled into hot UI polling loops.
- PARTIAL: Dream Mode is CPU-only and deterministic, but scheduling policy and cadence are not yet a mature background workload model.
- PARTIAL: UI bundle exceeds Vite's 500 kB warning threshold after minification.

Recommendations:

- Add restore candidate query benchmarks and indexes.
- Cap process output while streaming from child process, not after buffering.
- Add modelruntime queue latency metrics and timeout tests.
- Keep Dream jobs CPU-side and background-only unless explicitly operator-triggered.
- Cache provider capability/health summaries with short TTLs for UI.
- Consider route-level code splitting for desktop pages.

## Test Coverage Review

| Subsystem | Current Coverage | Gaps |
|---|---|---|
| Config | GOOD | Bind-host/public exposure tests missing. |
| Migrations | GOOD/PARTIAL | More idempotence and evidence immutability tests needed. |
| Backup/restore | GOOD/PARTIAL | Tamper/count fail-closed tests missing. |
| Semantic syscalls | GOOD | Public syscall API tests missing because API is absent. |
| Gateway/approval | GOOD/PARTIAL | SSRF, symlink, capability activation governance gaps. |
| Modelruntime | GOOD/PARTIAL | Streaming, process supervision, GPU telemetry admission, cost governance missing. |
| Context restore | PARTIAL | Near-match candidate retrieval and large-set benchmarks missing. |
| Dream Mode | GOOD/PARTIAL | Upsert/append decision and no-GPU test coverage should expand. |
| Rule Cells/Hyperlane | MISSING | No substrate tests because substrate is not implemented. |
| CPU-only safe mode | PARTIAL | Runbook exists; automated Windows/Linux smoke coverage missing. |
| API/UI | PARTIAL | Go API tests good; frontend tests absent. |
| Security | PARTIAL | SSRF, symlink, CORS, bind, secrets, process stress missing. |
| Startup/shutdown | PARTIAL | Some server shutdown tests; end-to-end smoke blocked on Windows. |
| Smoke path | BROKEN on Windows | Bash-only script fails without `/bin/bash`. |

Highest-risk missing tests:

1. SSRF/private-network denial.
2. Symlink/path traversal across file, artifact, backup, model import, archive extraction.
3. Capability status governance approval/audit tests.
4. Backup restore tamper fail-closed tests.
5. Context restore near-match candidate tests.
6. Bind/CORS exposure tests.
7. Telegram remote sender/chat allowlist tests.
8. Frontend degraded-state tests.
9. Process output/time/resource limit tests.
10. Windows-compatible smoke test.

## Full Punchlist

| ID | Title | Category | Severity | Complexity | Status | Short Pass |
|---|---|---|---|---|---|---|
| TEST-001 | Windows-compatible smoke | testing | high | medium | test | yes |
| AUTH-001 | Govern capability status changes | authority | high | medium | gap | yes |
| AUTH-002 | Remote Telegram sender allowlist | authority | high | medium | bug | yes |
| AUTH-003 | Audit authority-shaping APIs | authority | medium | small | gap | yes |
| AUTH-004 | Shared approval fingerprint semantics | authority | medium | medium | refactor | yes |
| DUR-001 | Restore tamper fail-closed | durability | high | medium | bug | yes |
| DUR-002 | Evidence immutability decision | durability | medium | medium | gap | no |
| CTX-001 | Near-match restore candidate retrieval | context_restore | high | medium | bug | yes |
| CTX-002 | Restore indexes and benchmarks | context_restore | medium | medium | upgrade | yes |
| DRM-001 | Dream report append/upsert decision | dream | medium | small | docs/test | yes |
| DRM-002 | Restore feedback feed loop | dream | medium | medium | upgrade | no |
| RULE-001 | Rule Cell / Hyperlane v0 | rule_cells | medium | large | gap | no |
| MR-001 | Streaming | modelruntime | high | large | upgrade | no |
| MR-002 | Backend process supervision | modelruntime | high | large | upgrade | no |
| MR-003 | Provider cost/egress governance | modelruntime | medium | medium | gap | yes |
| GATE-001 | SSRF denial | gateway | high | medium | bug | yes |
| GATE-002 | Process output pre-buffer cap | gateway | medium | medium | bug | yes |
| UI-001 | Accurate global runtime state | api_ui | high | medium | gap | yes |
| UI-002 | Shared frontend contracts | api_ui | medium | medium | refactor | yes |
| SEC-001 | Loopback default bind | security | high | small | bug | yes |
| SEC-003 | Symlink/path traversal suite | security | high | medium | test | yes |
| PERF-001 | Restore large-set benchmark | performance | medium | medium | test | no |
| DOC-001 | Update stale review/status docs | docs | medium | small | docs | yes |
| TEST-002 | Frontend tests and lint | testing | medium | medium | test | no |

## CSV Punchlist

```csv
id,title,category,severity,complexity,status,affected_modules,recommended_fix,acceptance_criteria,next_pass_candidate
AUTH-001,Govern capability status changes,authority,high,medium,gap,"api/phase5.go; gateway/service.go; gateway/tool_capability_registry.go","Require approval/audit for activating high-risk capabilities and persist actor/reason","Dangerous capability activation without approval is denied; reason and actor are persisted",yes
AUTH-002,Add Telegram sender/chat allowlist,authority,high,medium,bug,"api/telegram_gateway_service.go; api/remote.go","Enforce sender/chat allowlist for normal remote polling messages","Unauthorized sender cannot create/process remote chat",yes
AUTH-003,Audit authority-shaping APIs,authority,medium,small,gap,"lanes; permissions profiles","Emit immutable audit records for save/delete/activate","Audit rows exist for all lane/profile mutations",yes
AUTH-004,Shared approval fingerprint semantics,authority,medium,medium,refactor,"permissions; gateway approvals","Prevent broad permission lifting outside fingerprint validation","Unrelated job approval cannot authorize changed request shape",yes
DUR-001,Fail closed on restore tampering,durability,high,medium,bug,"backup/service.go","Validate checksums and entity counts before mutation","Tampered bundle leaves DB unchanged",yes
DUR-002,Decide evidence immutability,durability,medium,medium,gap,"store/migrate.go; evidence tables","Add triggers or audited update-only paths","Evidence/history mutation semantics are enforced and tested",no
CTX-001,Fix near-match restore candidate retrieval,context_restore,high,medium,bug,"aios/controllane/sqlite_store.go","Fetch by scope/kind/recency then rank lexical similarity","Similar queries are ranked; wrong workspace excluded",yes
CTX-002,Add restore indexes and benchmark,context_restore,medium,medium,upgrade,"store/migrate.go; restore tests","Add composite index and large fixture benchmark","Restore candidate query remains bounded",yes
DRM-001,Decide Dream report upsert vs append-only,dream,medium,small,docs,"aios/dream/service.go; docs","Document or change same-ID persistence behavior","Tests match documented report persistence semantics",yes
DRM-002,Add restore feedback evidence loop,dream,medium,medium,upgrade,"dream; restore scoring","Record restore outcomes as non-canonical evidence","Restore scorer can inspect prior outcome evidence",no
RULE-001,Implement or clearly defer Rule Cell Hyperlane v0,rule_cells,medium,large,gap,"aios future rule substrate; docs","Build deterministic registry or mark concept-only","Rules produce traceable proposals only under latency budget",no
MR-001,Implement modelruntime streaming,modelruntime,high,large,upgrade,"modelruntime; api/model_runtime.go","Add streaming with cancellation-safe audit and usage","Streaming clients work and partial streams are accounted",no
MR-002,Add backend process supervision,modelruntime,high,large,upgrade,"modelruntime backends","Own backend process lifecycle and health","Backend crash/degraded states are bounded and visible",no
MR-003,Add provider cost and egress governance,modelruntime,medium,medium,gap,"modelruntime providers; config","Add remote provider budgets and policy","Cloud usage cannot exceed policy budget",yes
GATE-001,Add SSRF denial,gateway,high,medium,bug,"gateway network tools; provider endpoints","Deny private/link-local/metadata/loopback unless allowed","SSRF deny table tests pass",yes
GATE-002,Bound process output before buffering,gateway,medium,medium,bug,"gateway process tool","Use bounded writers/streaming capture","Huge output does not grow memory unbounded",yes
UI-001,Accurate global runtime state,api_ui,high,medium,gap,"AppShell.tsx; health diagnostics","Show safe-mode modelruntime GPU provider degradation","Shell accurately shows CPU-only/modelruntime unavailable",yes
UI-002,Move stable contracts to shared,api_ui,medium,medium,refactor,"apps/desktop/src/lib/api.ts; packages/shared","Promote modelruntime Dream restore contracts to shared","Desktop imports shared contracts without drift",yes
SEC-001,Default core bind to loopback,security,high,small,bug,"services/core/main.go; config","Bind 127.0.0.1 by default and require explicit public bind","Default server does not listen on all interfaces",yes
SEC-003,Add symlink and traversal suite,security,high,medium,test,"gateway; artifacts; model import; backup","Add fixtures and resolved-path containment","Symlink/path escapes are denied",yes
PERF-001,Restore large-set benchmark,performance,medium,medium,test,"context restore scorer","Benchmark large snapshot sets and query plans","Threshold and index use are documented",no
DOC-001,Update stale review/status docs,docs,medium,small,docs,"docs/reviews/forge_full_system_review_20260425; docs/status","Mark completed passes and remaining gaps accurately","Docs no longer list completed passes as missing",yes
TEST-001,Add Windows-compatible smoke,testing,high,medium,test,"scripts; package.json","Add Node/PowerShell smoke or wrapper","npm run smoke passes on Windows and Linux",yes
TEST-002,Add frontend tests and lint,testing,medium,medium,test,"apps/desktop; packages","Add Vitest/RTL and JS lint or equivalent","Root scripts include JS/TS test and lint lane",no
```

## Next Passes

### Next 3 Immediate Short Passes

1. Capability governance hardening
   - Goal: approval/audit-gate dangerous capability status activation.
   - Why now: closes an authority-shaping side door.
   - Scope: gateway capability API/service/tests.
   - Tests: dangerous capability activation denied without approval; actor/reason persisted.
   - Do not do: redesign gateway taxonomy.

2. Security boundary pass
   - Goal: loopback bind default, SSRF deny table, symlink/path traversal tests.
   - Why now: prevents local-first API becoming accidental network-exposed unsafe surface.
   - Scope: config/main/gateway/file/network tests.
   - Tests: bind/CORS/SSRF/symlink fixtures.
   - Do not do: add remote auth product features.

3. Restore candidate correctness
   - Goal: remove exact-query prefilter and add indexes.
   - Why now: restore scoring claims depend on ranking near matches.
   - Scope: `ListContextSnapshots`, restore scorer tests, migrations.
   - Tests: similar query selected, wrong workspace excluded, query remains bounded.
   - Do not do: introduce LLM/vector truth scoring.

### Next 10 Implementation Passes

1. Capability status governance.
2. Loopback/SSRF/symlink security hardening.
3. Context restore candidate retrieval/index fix.
4. Backup restore tamper fail-closed gate.
5. Telegram remote sender/chat allowlist.
6. Global operator diagnostics and shell degraded state.
7. Modelruntime provider budget/egress governance.
8. Dream report append/upsert decision plus operator review workflow docs.
9. Public semantic syscall dry-run/submit/inspect API.
10. Rule Cell/Hyperlane v0 design spike and starter registry.

### Next 5 Test-Only Passes

1. Security boundary fixture suite.
2. Backup integrity tamper suite.
3. Context restore large-candidate suite.
4. Frontend degraded-state tests.
5. Windows-compatible smoke test.

### Next 5 Documentation Passes

1. Update stale 20260425 review docs with completed passes.
2. Update roadmap phase matrix for Dream persistence/operator inspector.
3. Document public binding/security defaults.
4. Document restore candidate scoring reality after fix.
5. Document Rule Cell/Hyperlane as concept until implemented.

### Do Not Do Yet

- Do not implement Dream apply/commit mode before syscall design and operator review workflow.
- Do not add GPU Dream jobs.
- Do not make cloud providers default fallback.
- Do not launch a full UI cockpit redesign before diagnostics/security hardening.
- Do not add Rule Cells that can mutate canonical truth directly.

## Deferred Ideas

- Full dual-mode cockpit redesign.
- Rule Cell marketplace or scripting/eval engine.
- Dream Mode commit/apply.
- GPU Dream jobs.
- LoRA/PEFT adapter registry.
- TensorRT-LLM/NIM/Riva/cuVS integration.
- Deep Nix/NixOS module packaging.
- Full artifact byte restore across backup bundles.
- Cross-provider cost dashboards beyond minimal governance.

Reason: these are useful later, but current highest-risk gaps are authority, security, restore correctness, backup integrity, operator diagnostics, and test coverage.

## Open Questions

| Question | Why It Matters | Current Best Answer | Needed Evidence |
|---|---|---|---|
| Should core bind loopback-only by default? | Local-first safety. | Yes. | Config/main tests and operator docs. |
| Should Dream reports be append-only? | Evidence integrity. | Not decided; current behavior upserts by ID. | Product/security decision + tests. |
| Which evidence tables need DB immutability triggers? | Historical truth integrity. | `journal_events` only today. | Migration design. |
| Should embedding records restore exactly or rebuild? | Vector/index truth boundary. | VSA is export-only; embeddings restore today. | Explicit policy decision. |
| Should modelruntime remote calls have budgets? | Cloud cost/egress control. | Yes for M4. | Governance design/tests. |
| Is `/api/process/health` a trace endpoint or global health? | Operator clarity. | Trace endpoint today. | Endpoint naming/UI decision. |
| Should `/forge/*` routes move under `/api`? | Client consistency. | Not urgent. | API versioning plan. |
| What is the first real Rule Cell substrate? | Avoid agent sprawl. | Deterministic registry with trace/no-mutation. | Design pass. |
| Should public syscall API be exposed? | External proposer integration. | Yes, bounded dry-run/submit/inspect. | Security/authority design. |
| How should Windows smoke be supported? | Current review/validation is Windows-hosted. | Add Node/PowerShell smoke. | Script pass. |
