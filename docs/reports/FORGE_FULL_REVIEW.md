# FORGE Full Code Review

**Reviewer:** Independent senior-engineering pass.
**Repo HEAD at review:** `37d7231 fix: align operator desktop in vm`
**Date of review:** 2026-05-11.
**Scope:** Full repository — services/core (Go), apps/desktop (Tauri + React + TS), crates/forgek-validate (Rust), nix/ (NixOS modules, packages, checks), docs/, scripts/, CI workflow.

---

## 1. Executive Summary

FORGE is a substantially more disciplined project than its size suggests. The repo is ~102k lines of Go, ~36k of TypeScript/TSX, ~44k of Go tests, plus a Rust validator crate, Nix substrate, and a 5-language CI pipeline. The `go test ./...` suite finishes in ~20 seconds with 52 passing packages and zero failures. `go vet` is clean. `nix flake check` evaluates. The TypeScript desktop builds. As of yesterday, the project actually boots as a session-locked desktop shell inside a NixOS VM (operator validated this manually).

The architecture is real, not aspirational. There is a single semantic commit gate at [services/core/internal/aios/controllane/processor.go](services/core/internal/aios/controllane/processor.go) that wraps every canonical mutation in a transaction with explicit approval/validate/apply/journal/commit ordering. Simulator/live separation is enforced at build time by forbidden-imports tests in `forgek`, `forgekshadow`, `forgeh`, `refvalidation`, and `semanticvalidation`. The Nix derivations hardcode `allowMutation=false` as a literal and grep wrappers for forbidden mutation tokens. These are *enforced* properties, not aspirational principles.

The biggest weaknesses are also concrete: [services/core/internal/gateway/service.go](services/core/internal/gateway/service.go) is **4,709 lines in a single file**, [services/core/internal/api/chat_assistant_gateway.go](services/core/internal/api/chat_assistant_gateway.go) is **2,497 lines**, and the test/source ratios on the load-bearing kernel package (`aios/controllane` at ~12%) and the memory package (~6.5%) are thin relative to their importance. The documentation surface is large and uneven: ~40 architecture docs and ~33 status docs, with the high-quality status files (e.g. [docs/status/runtime_truth_vs_docs.md](docs/status/runtime_truth_vs_docs.md)) carrying more truth than some narrative architecture docs.

**Maturity:** Late alpha. Past prototype, not yet production. The shape of a 1.0 is visible.
**Coherence:** Yes — strong architectural backbone, consistent doctrine, repeatable migration pattern.
**Architecture visible in code:** Yes — semantic syscalls, kernel commit gate, forbidden-imports fences, capability registry, gateway tool registry, journal/audit chain are all present and tested.
**Biggest strength:** Build-time enforcement of doctrine (commit chokepoint, import fences, Nix safe-mode literals).
**Biggest problems:** Two monolithic Go files, thin tests on three load-bearing packages, ADR numbering collision, ~20 packages with no tests, doc surface area outruns reading capacity.
**Highest-leverage next move:** Split `gateway/service.go` along the existing capability-category lines (filesystem, command, http, git, identity, …) — every other refactor gets easier afterwards.

---

## 2. What FORGE Is

**FORGE is currently best understood as** a local-first AI workspace and runtime: a Go backend service (`forge-core`) that exposes a structured, capability-gated HTTP API; a Tauri/React desktop shell that consumes that API; and a NixOS substrate that can boot the shell as a session-locked operator surface inside a VM. The backend enforces a transactional, journaled commit pattern for semantic state changes and gates tool execution through a single registry with bounded inputs.

**FORGE is intended to become** a unified cognitive runtime — a local AI operating layer where models propose, deterministic validators check, and a kernel-style core commits state. The roadmap targets cleaner separation of "what a model says" from "what the system *did*," with audit trails sufficient to reconstruct every meaningful state transition.

**FORGE is not** a chatbot wrapper, a SaaS dashboard, or a Python sidecar. It is also not yet a self-improving autonomous agent — autonomy lanes exist as scaffolding with charter/budget gating, but the live daemon does not let models mutate canonical state without going through the deterministic gate.

---

## 3. What FORGE Can Do Today

### Confirmed Capabilities

Verified against code:

- **Boot and serve.** [services/core/main.go](services/core/main.go) brings up an HTTP server on a configurable bind host (default `127.0.0.1`) with graceful shutdown, logging, and CORS that locks origins to localhost/Tauri only. Default port surfaces as `18492` in the desktop wrapper.
- **Structured HTTP API.** 188 routes registered through `chi` in [services/core/internal/api/routes.go](services/core/internal/api/routes.go), organized into mount groups (`/forge`, `/health`, OpenAI-compat, etc.).
- **Persistent storage.** Two SQL backends with parity migrations: SQLite (default, [services/core/internal/store/store.go](services/core/internal/store/store.go)) and Postgres scaffolded ([services/core/internal/store/postgres.go](services/core/internal/store/postgres.go), [services/core/internal/store/postgres_migrations.go](services/core/internal/store/postgres_migrations.go)).
- **Transactional semantic commit.** Every Control Lane mutation goes through one apply/journal/commit transaction in [services/core/internal/aios/controllane/processor.go](services/core/internal/aios/controllane/processor.go); approval, capability, and idempotency checks happen before apply.
- **Tool execution gateway.** Bounded tool surface in [services/core/internal/gateway/service.go](services/core/internal/gateway/service.go) with input validation, SSRF defense, symlink scope containment, mode/PID/ref validation, bounded combined output, and a tool-capability registry that includes approval-only entries.
- **Model runtime management.** Lifecycle endpoints under `/forge/models/*` covering list, scan, import, verify, enable/disable, archive, remove, load/unload, chat. Implementation in [services/core/internal/modelruntime/](services/core/internal/modelruntime/) and API in [services/core/internal/api/model_runtime.go](services/core/internal/api/model_runtime.go).
- **Live Control Lane validation syscalls.** Four registered, all non-mutating: `VALIDATE_KV_IDENTITY`, `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, `VALIDATE_SEMANTIC_OPERATION`.
- **Approval system.** Request/decision separation, scope snapshots, fingerprint bounds enforcement. [services/core/internal/approvals/service.go](services/core/internal/approvals/service.go).
- **Audit & journal.** Append-only event capture across the kernel commit path; immutable-by-design tests in [services/core/internal/audit/immutable_test.go](services/core/internal/audit/immutable_test.go).
- **Backup/restore.** Service in [services/core/internal/backup/service.go](services/core/internal/backup/service.go) (~2,000 lines) with tests covering restore outcomes and tamper paths.
- **Tauri desktop shell.** 46 page components, 4 stores (workspace, layout, shell, UI), multi-monitor window manager, navigation/command palette pattern. [apps/desktop/src/App.tsx](apps/desktop/src/App.tsx) routes the lot.
- **VM-bootable operator desktop.** `nix build .#forge-operator-session` produces a launcher that chains TTY → labwc compositor → Tauri shell → `forge-core`. User confirmed end-to-end boot in VirtualBox; foot terminal and pcmanfm file manager launch through an allowlist surface.
- **Ollama adapter** with bounded response decoding ([services/core/internal/adapters/ollama.go](services/core/internal/adapters/ollama.go)).
- **Discord and Telegram integration scaffolding** with bounded payload validation ([services/core/internal/api/discord_gateway_router.go](services/core/internal/api/discord_gateway_router.go), [services/core/internal/api/telegram_gateway_service.go](services/core/internal/api/telegram_gateway_service.go)).
- **Rust validator parity.** `crates/forgek-validate` + shared fixtures under `fixtures/forgek` give Go/Rust agreement on validator behavior, exercised in CI.

### Partial Capabilities

- **FORGE-K cognitive microkernel.** Phases 1-11G implemented and tested in the simulator at [services/core/internal/forgek/](services/core/internal/forgek/), but the live daemon does *not* yet route mutation through FORGE-K. Only validation seams are live.
- **Postgres / Qdrant / Redis backends.** Schemas, capability contracts, and shadow-mode adapters exist; none are the live default.
- **Autonomy.** Charters, budgets, lanes, dream mode are scaffolded ([services/core/internal/aios/autonomy/](services/core/internal/aios/autonomy/), [services/core/internal/aios/dream/](services/core/internal/aios/dream/)) but propose-only by default.
- **Hyperlane intent classifier.** 114-line enum + route table in [services/core/internal/aios/hyperlane/intent.go](services/core/internal/aios/hyperlane/intent.go); gateway-side parser tests exist, but the package itself has no test file.
- **Lymphatic lane.** Maintenance reports and cleanup proposals exist in the simulator only.
- **Operator desktop apps.** Allowlist surface launches foot and pcmanfm; the Rust-side `launch_operator_app` Tauri command exists but firefox/full integration is partial.

### Planned But Not Yet Implemented

- **Live FORGE-K authority** for memory/state mutation paths.
- **Streaming model output, delete-file approval, fuller backend supervision** (Model Runtime M4 items).
- **Rule cells / Hyperlane v0 as a deterministic reflex substrate** (still concept).
- **Live Postgres/Qdrant/Redis** as canonical-truth backends.
- **Voice / mic / camera / continuous perception** surfaces.
- **Real effectors** beyond advisory FORGE-H resource policy proposals.

---

## 4. How FORGE Works

**Startup.** `services/core/main.go` loads config, opens the store, builds an `api.Server`, and starts an HTTP listener with a 10-second header timeout. A SIGINT/SIGTERM handler triggers `Shutdown` with an 8-second grace window. A warning is logged when bound to a wildcard host.

**Server construction.** `api.NewServer(st, cfg)` in [services/core/internal/api/server.go](services/core/internal/api/server.go) (1,589 lines) wires all subsystem services: gateway, modelruntime, controllane, approvals, audit, jobs, memory, retrieval, embeddings, backup, automation, autonomy, dream, hostbridge, forgeh, forgekshadow, and others. The server then mounts route groups via `routes.go`.

**Request flow.** A request enters chi middleware (`RequestID`, `RealIP`, `Logger`, `Recoverer`, CORS gate, optional route-envelope shadow observer), then routes to a handler on `*Server`. For tool execution requests the handler delegates to `gateway.Service.Invoke(...)`, which performs capability lookup → input bound checks → validator chain → tool function → bounded output capture → response. For semantic state mutation, the handler hands an envelope to `controllane.Processor.Process(...)`, which runs approval/capability/validator/idempotency in order before opening a transaction, applying the side effects, appending a journal record, and committing.

**Persistence.** SQLite is the live store. The schema is migrated through versioned SQL in [services/core/internal/store/migrate.go](services/core/internal/store/migrate.go) (1,609 lines) with idempotent steps. Postgres has parity migrations under [postgres_migrations.go](services/core/internal/store/postgres_migrations.go) but is not the live default.

**Config loading.** [services/core/internal/config/config.go](services/core/internal/config/config.go) reads from environment with explicit defaults; smoke-tested separately. The Nix wrappers set safe defaults (`FORGE_SHELL_SAFE_MODE=true`, `FORGE_SHELL_HOST_MUTATION=false`, `FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false`, etc.) before launching the binary.

**Event / logging.** Standard library `log` for daemon output; structured event capture for jobs and journal via the relevant services. Optional Rust validator parity tests run in CI.

**Workspace / project flow.** Workspace state in [apps/desktop/src/stores/workspaceStore.ts](apps/desktop/src/stores/workspaceStore.ts) (Zustand) backed by `/forge/*` API calls. Layouts and per-window surface assignments persist via `workspaceLayoutStore`.

**Model integration.** Modelruntime is a governed boundary in [services/core/internal/modelruntime/](services/core/internal/modelruntime/). Backends include managed llama.cpp ([backend_llama_cpp.go](services/core/internal/modelruntime/backend_llama_cpp.go)) and openai-compat ([backend_openai_compat.go](services/core/internal/modelruntime/backend_openai_compat.go)), plus an external Ollama adapter at [services/core/internal/adapters/ollama.go](services/core/internal/adapters/ollama.go). Lifecycle is approval/capability gated.

---

## 5. Public-Safe Architecture Explanation

**Two-paragraph plain description.** FORGE is a local-first AI workspace built around the idea that AI behavior should be inspectable, gated, and recoverable. Instead of a chatbot that talks to a database, FORGE separates *proposing* (what a model says) from *committing* (what the system durably accepts as truth). Model output flows through deterministic validation, approval gates, and a single transactional commit boundary before anything becomes part of the system's state.

The result is an AI environment where every meaningful state change is auditable, every tool invocation is bounded, and the desktop shell that the operator interacts with is the same Tauri application that can be packaged and booted as a NixOS session. FORGE runs entirely on the operator's machine; external model providers (Ollama, OpenAI-compatible endpoints) plug in as governed adapters rather than embedded dependencies.

**One technical paragraph.** The backend is a Go service exposing a chi-routed HTTP API. State lives in SQLite by default with Postgres as a parity backend. A single semantic-commit chokepoint wraps every canonical mutation in approval → capability → validator → transactional apply → journal append. Tool execution goes through a separate gateway with input bound checking, SSRF defense, path containment, and a capability registry that enforces approval-only entries. The desktop is a Tauri + React + Zustand application with a workspace/window manager. The packaging substrate is Nix: dedicated packages, NixOS modules, profiles, and flake checks that statically enforce safe-mode environment defaults.

**What makes FORGE different.** Most local-AI projects either embed an LLM as a library and trust its output, or wrap a remote API and forward responses. FORGE treats every model as a *bounded driver* — its output is a proposal, not a commit, and the commit path is a deterministic kernel the model cannot bypass. The kernel boundary is enforced not by convention but by *tests* (forbidden-imports tests prevent the simulator from referencing live packages) and *build configuration* (Nix derivations hardcode safe-mode flags). The architecture survives an LLM being wrong.

**What this description does not reveal.** Internal naming for the cognitive lanes, the specific deterministic mechanisms used inside the simulator subsystems, the exact validator algorithms, and the forward-looking roadmap for moving authority from simulator to live. Those remain under the project's internal doctrine.

---

## 6. Repository Map

```
ProjectForge/
├── AGENTS.md                       # Operator/agent doctrine — 166 lines, authoritative
├── CLAUDE.md                       # Pointer file for Claude Code — 20 lines
├── CODEX.md                        # Forward-vision implementation prompt — 594 lines, marked [FUTURE]
├── README.md                       # Public-facing project description — 295 lines
├── FORGE_CONTEXT.md                # Source-of-truth context generator output
├── Full-Code-Review.md             # This review's brief
├── Operator-Toolbelt.txt           # EMPTY file — clutter, should delete or fill
├── package.json                    # npm workspace root, scripts
├── flake.nix                       # Nix flake — packages/apps/checks/devShells
├── flake.lock
├── docker-compose.yml              # Container orchestration (Postgres/Qdrant/Redis optional)
├── docker-compose.igpu.yml         # iGPU variant
├── .env.docker.example             # Reference env file
├── apps/
│   └── desktop/                    # Tauri + React + Vite + Zustand
│       ├── src/                    # 46 pages, 4 components, 4 stores, layout, lib
│       ├── src-tauri/              # Rust Tauri shell + capabilities/
│       └── package.json
├── services/
│   └── core/                       # Go daemon
│       ├── main.go                 # 73-line minimal entry
│       ├── go.mod                  # Go 1.24, minimal deps
│       └── internal/               # 74 packages — see Section 7
├── crates/
│   └── forgek-validate/            # Rust validator crate (parity testing)
├── fixtures/
│   └── forgek/                     # Shared validator fixtures (Go ↔ Rust)
├── packages/                       # npm workspace shared packages (@forge/shared, @forge/ui)
├── nix/
│   ├── packages/                   # 5 packages incl. forge-core, forge-operator-session
│   ├── nixos/
│   │   ├── modules/                # 5 modules incl. forge-host-kernel, forge-os
│   │   └── profiles/               # forge-operator-desktop, forge-vbox-graphics-test
│   ├── checks/                     # 12 flake checks (build, runtime, safe-defaults)
│   └── overlays/
├── scripts/                        # Dev tooling (smoke, parity, integration-env)
├── docs/
│   ├── adr/                        # 11 ADRs — TWO numbered 0001 (collision)
│   ├── architecture/               # ~40 architecture docs
│   ├── data_model/                 # Schema/model docs
│   ├── operations/                 # Operational guides
│   ├── runbooks/                   # 7 runbooks incl. current_forge_bringup
│   ├── status/                     # 33 status/reality docs — high quality
│   ├── reviews/                    # 28+ prior phase reviews
│   ├── reports/                    # (this report lives here)
│   ├── roadmap/
│   ├── testing/
│   ├── superpowers/                # Internal design notebook
│   ├── journal/                    # Master technical journal
│   ├── diagrams/                   # Mermaid diagrams
│   └── glossary.md
├── .forge/                         # Runtime data (logs, nix-results, run, vm) — gitignored
├── .vm-build-core.log              # VM build log — clutter, should gitignore
├── .vm-nix-store/                  # VM artifact — clutter
└── .vm-nix-tmp/                    # VM artifact — clutter
```

**Flagged:**
- **Empty:** `Operator-Toolbelt.txt` (0 bytes).
- **Clutter to gitignore:** `.vm-build-core.log`, `.vm-nix-store/`, `.vm-nix-tmp/`.
- **Numbering collision:** Two ADRs named `0001-*`. The `forge-is-ai-os` and `forge-k-is-a-cognitive-microkernel` ADRs cannot both be 0001.
- **Doc proliferation:** 28+ files in `docs/reviews/` is approaching read-fatigue territory. Older phase reviews (12A through 13H) could be moved to `docs/reviews/archive/`.
- **Forward-vision file with present-tense language:** `CODEX.md` has a `[FUTURE]` self-disclaimer at the top, good — but the body still reads as a current spec in some sections. Audit one more time for tense.

---

## 7. Major Systems and Modules

### Control Lane (semantic syscall kernel)

**Location:** [services/core/internal/aios/controllane/](services/core/internal/aios/controllane/)
**Purpose:** Single chokepoint for canonical semantic state mutation. Receives envelopes, validates, opens a transaction, applies, journals, commits.
**Current Status:** Working.
**Key Files:** `processor.go` (475 lines, the gate), `processor_apply.go`, `registry.go`, `validator.go`, `audit.go`, `capabilities.go`, `kv_enforcement.go`, `ref_validation.go`, `ref_shape_compare.go`, `semantic_operation_validation.go`, `sqlite_store.go` (2,244 lines), `kv_identity_test.go`, `kv_boundary_test.go`, `ref_validation_test.go`, others.
**Strengths:** Real commit gate. Transactional. Approval/capability/validator/idempotency chain visible and tested. Four live validation seams already migrated using a shared-pure-package pattern.
**Problems:** Test/source ratio ~12% on a load-bearing package. `sqlite_store.go` is 2,244 lines and likely benefits from splitting by repository concern.
**Recommended Action:** (1) Coverage pass on `processor.go` and `sqlite_store.go` to reach 25-30%. (2) Split `sqlite_store.go` along the same repository boundaries already evident in code.

### Gateway (bounded tool execution)

**Location:** [services/core/internal/gateway/](services/core/internal/gateway/)
**Purpose:** Sole governed tool execution surface. Capability registry + per-capability tool functions + input bound checks + SSRF/path/mode/PID/ref validators.
**Current Status:** Working — recently hardened.
**Key Files:** `service.go` (4,709 lines — **the largest single file in the repo**), `capability_backing_tool.go` (1,719 lines), `tool_capability_registry.go`, `command_output.go`, `http_response.go`, `hyperlane_intent_parser.go`, plus 46 test files including 24 `*_bounds_test.go` files and `network_fetch_ssrf_test.go`.
**Strengths:** Comprehensive input validation. SSRF coverage. Symlink containment. Bounded output buffers. Approval-only registry entries. 19% test-to-source ratio (up from 12% before the recent hardening pass).
**Problems:** `service.go` at 4,709 lines is the project's largest architectural smell. Navigation, review, and onboarding are all harder than they should be.
**Recommended Action:** Split `service.go` along capability-category lines (filesystem, command, http, git, identity, secret, observability, archive, system, …). Each category becomes its own file under `services/core/internal/gateway/` or its own subpackage. The capability registry stays in one place.

### API Server

**Location:** [services/core/internal/api/](services/core/internal/api/)
**Purpose:** HTTP transport for the daemon. Mounts route groups under `/forge`, `/v1`, `/health`, integrations.
**Current Status:** Working.
**Key Files:** `server.go` (1,589 lines, 23 handler methods), `routes.go` (188 routes), `chat_assistant_gateway.go` (2,497 lines — second-biggest file), `chat_post.go` (1,280 lines), `autonomy_maintenance_loop.go` (1,545 lines), `model_runtime.go` (1,160 lines), `model_runtime_bridge.go` (1,413 lines), `phase5.go` (1,297 lines), `system_status.go`, `remote.go`, `telegram_*.go`, `discord_*.go`.
**Strengths:** Routes are *in a separate file* from `server.go` — that's good structure. 22% test-to-source ratio. Comprehensive request body bound checking (`*_request_body_test.go` series).
**Problems:** `chat_assistant_gateway.go` and `autonomy_maintenance_loop.go` are both >1.5k lines. Several `phaseN.go` files exist alongside the live handlers — the line between "current handler" and "phase historical" is not obvious.
**Recommended Action:** Promote the API package into a `handlers/` subpackage by domain (chat, models, autonomy, system, integrations, phases). Move `phase*.go` history into `archive/` if no longer routed.

### Model Runtime

**Location:** [services/core/internal/modelruntime/](services/core/internal/modelruntime/)
**Purpose:** Governed inference substrate. Backend selection, lifecycle, manifest, store, queue, usage tracking.
**Current Status:** Working — M3 management features complete; M4 deferred (streaming, delete-file approval, stronger supervision).
**Key Files:** `service.go` (1,581 lines), `backend_llama_cpp.go`, `backend_openai_compat.go`, `store.go`, `store_management.go`, `manifest.go`, `state.go`, `json_file.go`, `policy.go`, `scheduler.go`, `limits.go`.
**Strengths:** 17.7% coverage. Bounded JSON file loading. Backend abstraction is clean.
**Problems:** `service.go` is large. Streaming is missing.
**Recommended Action:** Continue M4. Split `service.go` by lifecycle concern.

### Memory / State

**Location:** [services/core/internal/memory/](services/core/internal/memory/)
**Purpose:** Semantic memory authority — notes, links, supersession, contradiction, snapshots.
**Current Status:** Working live, but thinly tested.
**Key Files:** Multiple — including VSA-tracked files (`vsa_engine.go`, `vsa_indexer.go`, `vsa_signals.go`) protected by `scripts/check-vsa-files.sh`.
**Strengths:** Real semantic content. VSA tracking is enforced as a CI preflight.
**Problems:** **6.5% test/source ratio** — the lowest among load-bearing packages. This is the canonical-truth store; thin coverage here is the highest-risk gap in the project.
**Recommended Action:** Coverage triage. Pick the ten most-called functions and write substantive table-driven tests. Target 25% by next phase.

### Storage Backends

**Location:** [services/core/internal/store/](services/core/internal/store/), [services/core/internal/storagebackend/](services/core/internal/storagebackend/), [services/core/internal/vectorstore/](services/core/internal/vectorstore/), [services/core/internal/ephemeral/](services/core/internal/ephemeral/)
**Purpose:** SQL (SQLite/Postgres), vector (Qdrant shadow), ephemeral (in-memory/Redis shadow).
**Current Status:** SQLite live; others shadow-only / readiness reviewed.
**Strengths:** Capability contracts, parity migrations, shadow adapters. Ephemeral has explicit value bounds (`ephemeral/values.go`).
**Problems:** Postgres/Qdrant/Redis live cutover is not committed (Phase 13I marked `READINESS_REVIEW` only).
**Recommended Action:** When ready, pick one (Postgres for canonical truth is the natural first move) and dual-write behind a flag before cutting over.

### FORGE-K Simulator

**Location:** [services/core/internal/forgek/](services/core/internal/forgek/) (17 subpackages)
**Purpose:** Hermetic simulator for the cognitive microkernel architecture. Pure deterministic; no live daemon authority.
**Current Status:** Working as simulator only — explicitly fenced.
**Key Files:** `forbidden_live_imports_test.go` (the fence), `consensus/`, `court/`, `contextcompiler/`, `kv/`, `lymphatic/`, `palace/`, `runtime/`, `semantic/`, `shadowharness/`, `snapshots/`, `neurons/`.
**Strengths:** 18.4% coverage. Import fence prevents live coupling. Pure determinism.
**Problems:** None inside its scope. The *project-level* risk is "did we build a simulator nobody migrates from" — Section 25 addresses.
**Recommended Action:** Continue narrow migrations using the shared-pure-package pattern proven by `kvidentity`, `refvalidation`, `semanticvalidation`.

### FORGE-H Resource Policy

**Location:** [services/core/internal/forgeh/](services/core/internal/forgeh/)
**Purpose:** Advisory resource action proposals — bounded executor, never mutates.
**Current Status:** Working as advisory only — enforced at Nix level (`allowMutation=false` literal).
**Strengths:** 34% test/source ratio. Adapter interfaces, no concrete mutating impls. Forbidden-imports test.
**Problems:** Adapter side (`OperatorNotifier`, `LanePolicyWriter`, `ModelPolicyWriter`, `DegradedModeWriter`) has no concrete implementations yet, which is intentional but worth noting.

### Hostbridge

**Location:** [services/core/internal/hostbridge/](services/core/internal/hostbridge/)
**Purpose:** Read-only host kernel bridge diagnostic library — operational evidence, not semantic memory.
**Current Status:** Working as read-only.
**Strengths:** 23% coverage. Forbidden-imports test. Hard read-only.

### Forgekshadow

**Location:** [services/core/internal/forgekshadow/](services/core/internal/forgekshadow/)
**Purpose:** Disabled-by-default shadow diagnostic observers (route envelope, chat metadata, retrieval metadata, control-lane validation reports).
**Current Status:** Working. 63%+ coverage.
**Strengths:** All observers double-gated by env flags. Bounded scalar summaries only — no payload capture. Pure observation, no live effect.

### Desktop Shell

**Location:** [apps/desktop/](apps/desktop/)
**Purpose:** Tauri + React UI; operator surface; soon-to-be NixOS session.
**Current Status:** Working — boots inside a VM as a Wayland session.
**Key Files:** `src/App.tsx` (route registration), `src/layout/AppShell.tsx`, `src/pages/*` (46 pages), `src/stores/*` (4 stores), `src/lib/api.ts`, `src-tauri/src/main.rs`.
**Strengths:** Real multi-monitor support, workspace layouts, error boundary, Tauri-event-driven layout updates, command palette pattern.
**Problems:** 46 pages vs 4 shared components — likely cross-page duplication of network/state patterns. Pages largely import from `lib/api.ts` directly (23 of 46 reference `components/`).
**Recommended Action:** Section 28 covers this; tl;dr extract common patterns from pages into shared components.

### Nix Substrate

**Location:** [nix/](nix/)
**Purpose:** Reproducible packaging for backend, desktop shell, operator session, NixOS modules, and CI checks.
**Current Status:** Working — 12 flake checks, 5 packages, 5 NixOS modules, 2 profiles.
**Strengths:** Build-time enforcement of doctrine (safe-mode env literals, forbidden-token grep in wrapper scripts, `allowMutation=false` literal). Operator desktop profile asserts `autoStart=false` and `autoLogin.enable=false`.

### Autonomy / Dream / Maintenance

**Location:** [services/core/internal/aios/autonomy/](services/core/internal/aios/autonomy/), [services/core/internal/aios/dream/](services/core/internal/aios/dream/), [services/core/internal/api/autonomy_maintenance_loop.go](services/core/internal/api/autonomy_maintenance_loop.go)
**Purpose:** Self-initiated proposal lanes — chartered, budgeted, propose-only.
**Current Status:** Scaffolded.
**Problems:** Coverage thin (autonomy 9.2%, dream 17.2%); these lanes feed the kernel and should be deeply tested.

### Approvals / Audit / Permissions

**Location:** [services/core/internal/approvals/](services/core/internal/approvals/), [services/core/internal/audit/](services/core/internal/audit/), [services/core/internal/permissions/](services/core/internal/permissions/)
**Purpose:** Request/decision separation, append-only event capture, capability gating.
**Current Status:** Working.

### Other notable systems

- **Backup** — 2,005-line service with restore outcome tracking.
- **Jobs** — 1,452-line orchestration service.
- **Truth engine** — 1,060-line current-truth authority.
- **Compute librarian** — cell-based reasoning surface ([services/core/internal/aios/compute/librarian/](services/core/internal/aios/compute/librarian/)).

---

## 8. Backend Review

### Strengths

- **Clean entry point.** `main.go` is 73 lines, single-responsibility, with explicit shutdown.
- **Routes separated from handlers.** `routes.go` mounts; `server.go` and per-domain files handle.
- **Single commit chokepoint** enforced and tested.
- **Build-time doctrine.** Forbidden-imports tests act as architectural fences.
- **Bounded input validation** is consistent across the gateway, ephemeral, adapters, modelruntime, and API request paths.
- **Zero TODO/FIXME/HACK/`panic("` matches** across `services/core/internal/` — unusually clean.
- **Comprehensive CI** (Go test, vet, Rust validator, parity, typecheck, desktop build, smoke).

### Problems

- **Monolithic Go files.**
  - `gateway/service.go`: 4,709 lines.
  - `api/chat_assistant_gateway.go`: 2,497 lines.
  - `aios/controllane/sqlite_store.go`: 2,244 lines.
  - `backup/service.go`: 2,005 lines.
  - `gateway/capability_backing_tool.go`: 1,719 lines.
  - `store/migrate.go`: 1,609 lines.
  - `api/server.go`: 1,589 lines.
  - `modelruntime/service.go`: 1,581 lines.
  - `api/autonomy_maintenance_loop.go`: 1,545 lines.
  - These read fine and pass review individually but compound onboarding and review cost.
- **Coverage uneven on load-bearing packages.** Memory 6.5%, autonomy 9.2%, controllane 12%, gateway 18.8%.
- **20 packages with zero test files.** Including `events`, `search`, `policy`, `dossiers`, `lineage`, `insights`, `chat`, `canvas`, `aios/hyperlane`.
- **Several `phaseN.go` files in `api/`.** The split between "live handler" and "historical phase implementation" is unclear from filenames alone.
- **`store/migrate.go` is 1,609 lines** of versioned SQL migration code. Long migrations are unavoidable, but a split by phase/version would help.

### Suggested Server Split (Gateway First)

The big lift is `gateway/service.go`. Split by capability category, each file owning ~10-15 capabilities:

```
services/core/internal/gateway/
  service.go                       # Constructor, struct, top-level Invoke, registry wiring
  capability_registry.go           # already exists — keep
  tools_filesystem.go              # read/write/mkdir/list/chmod/path tools
  tools_command.go                 # process run/terminate, command output capture
  tools_archive.go                 # archive extract, scope checks
  tools_git.go                     # checkout/stash/message/patch
  tools_http.go                    # outbound HTTP, SSRF validation, body bounds
  tools_identity.go                # signing, tokens, messages
  tools_secret.go                  # secret payload bounds
  tools_system.go                  # systemd unit name validation, journal tail
  tools_observability.go           # alert bounds, time-schedule bounds
  tools_chat.go                    # chat-related tool surface
  tools_desktop_bridge.go          # desktop bridge payloads
  validation_url.go                # validateOutboundHTTPURL, blockedOutboundIP, resolver
  validation_paths.go              # workspace path/symlink validators
  validation_modes.go              # chmod/git-stash/git-ref modes
  output_buffer.go                 # boundedOutputBuffer (already partially extracted)
```

This is a one-day refactor with the existing test suite as a safety net.

---

## 9. Frontend / UI Review

### Current shape

- **Framework:** React 18 + Vite + TypeScript + Tauri 2.
- **State:** Zustand stores for `desktopShell`, `desktopWindow`, `workspaceLayout`, `workspace`, `ui`.
- **Routing:** `react-router-dom` with 40+ routes in `App.tsx`.
- **Layout:** `AppShell` wraps `RoutedViews` inside `ForgeErrorBoundary`.
- **Tauri integration:** Real — listens for `WORKSPACE_LAYOUT_EVENT` and `WORKSPACE_NAVIGATE_EVENT`, gets current window via `@tauri-apps/api/window`.
- **Multi-monitor support:** Yes (`desktopShellStore`, named workspace layouts).
- **Recent additions:** `SystemPage` (system status read-only surface), `OperatorAppsPage` (allowlist launch surface for foot/pcmanfm/Firefox).
- **Tests:** 8 page-level Vitest tests (`*.test.tsx`) plus `api.test.ts`, `desktopWindowStore.test.ts`, `workspaceLayoutStore.test.ts`. Most pages lack tests.

### Problems

- **46 pages, 4 shared components.** Suggests cross-page duplication of fetch/error/loading patterns. 23 of 46 pages currently reference `components/` — the rest probably duplicate.
- **Page-level test coverage is sparse** (only ~8 of 46 pages).
- **No visible design-system primitives** beyond `FoldSection`, `CommandBar`, `HumanDataView`, `ForgeErrorBoundary`. A Tauri-shell-grade UI usually needs more shared atoms (Panel, StatusRow, KeyValueList, Modal, Toast, …).
- **No theming system** that I could find — colors/styles likely hardcoded per page.
- **Accessibility** (focus management, keyboard nav, ARIA) needs an audit pass; not obviously broken but not obviously addressed.
- **Loading/error states** are page-by-page; no shared pattern (`<AsyncState>` wrapper) visible.

### Direction check

The target is an AI operating shell, not a SaaS dashboard. Current shape leans more toward dashboard. The Operator Desktop and System surfaces are moving toward shell. Section 28 covers the alignment plan.

---

## 10. Runtime / Service Layer Review

**Does FORGE behave like a runtime?** Mostly yes.

- **Service registry:** Implicit via `api.Server` struct fields — every subsystem is a field initialized in `NewServer`. Works but not introspectable at runtime.
- **Daemon/process concepts:** Yes — single Go process, graceful shutdown, signal handling.
- **Event bus:** Not a generic bus; per-domain event streams (jobs, audit). That's intentional — explicit channels per concern.
- **Scheduler / job runner:** Yes — `jobs` package, plus `autonomy_maintenance_loop`.
- **Watchers:** `fsnotify` in `internal/watch/`.
- **Internal services:** Many — gateway, modelruntime, controllane, approvals, audit, memory, retrieval, embeddings, backup, automation, autonomy, dream, hostbridge, forgeh, forgekshadow.
- **Health checks:** `/health` endpoint. Not deep (no per-service health surface visible to operator from the API directly, though `SystemPage` consumes a system_status surface).
- **Lifecycle hooks:** Server has `ShutdownWatch()`. Per-service shutdown is implicit.
- **Startup/shutdown discipline:** Adequate. Shutdown has an 8s grace window.
- **Runtime observability:** Logger middleware logs requests. No structured logging layer (no zap/slog) visible — uses stdlib `log`. Metrics endpoint not visible.

### Gaps

- No `slog`-style structured logging. For a runtime, this matters.
- No metrics endpoint (Prometheus or otherwise).
- No per-service health rollup.
- No "are we degraded?" surface beyond `SystemPage`'s rendering of system_status.

### Recommended

- Adopt `log/slog`. Tag every log line with request_id/correlation_id where available (you already have these in flight).
- Add a `/health/detailed` endpoint that aggregates per-service health (storage, modelruntime, gateway, hostbridge, …).
- Add a Prometheus-format `/metrics` endpoint behind a config flag.

---

## 11. Memory, State, and Context Review

The memory and state architecture is real but the test surface around it is the thinnest in the project.

**Persistence:** SQLite at `${FORGE_DATA_DIR}/forge.sqlite`. Migrations idempotent. Postgres parity exists but isn't live.

**Append-only event capture:** Yes — journal records flow through the Control Lane commit gate. Audit immutability tested in [audit/immutable_test.go](services/core/internal/audit/immutable_test.go).

**Semantic notes / links / state / loops:** All represented in the Control Lane semantic syscall surface — `CapMemoryNoteCreate`, `CapMemoryNoteArchive`, `CapMemoryLinkCreate`, `CapStateUpdate`, `CapLoopOpen`, `CapLoopClose`, `CapMemoryContradictionReg`, `CapMemorySupersessionMark`.

**Derived models / context compilation:** `CapModelDerive` and `CapContextCompile` syscalls exist. Context restore scoring is implemented at [controllane/compile_context_restore_scoring.go](services/core/internal/aios/controllane/compile_context_restore_scoring.go) (1,478 lines).

**Snapshots vs truth:** Explicitly separated. Doctrine: snapshots preserve shape for restoration; truth lives in canonical state. ADR 0003 and tests enforce.

**Deduplication / idempotency:** Idempotency table per envelope. Processor checks before apply.

**Conflict / contradiction handling:** `CapMemoryContradictionReg` and `CapMemorySupersessionMark` exist as syscalls; the live behavior beyond registration is partially scaffolded.

**Concerns:**

- Memory package coverage 6.5% — the lowest of any load-bearing package.
- The 1,478-line `compile_context_restore_scoring.go` is a single file holding a substantial scoring system. Hard to review.
- No public-facing description of how a piece of memory is "retired" vs "superseded" without leaking internal doctrine. Should be safe to document with appropriate abstraction.

**Recommended:**

- Coverage triage on `memory/` immediately.
- Split `compile_context_restore_scoring.go` if a natural seam exists.
- Document the lifecycle of a memory note (create → link → supersede/archive → audit reconstruction) at a public-safe level.

---

## 12. API and Integration Review

### Structure

- 188 routes mounted via chi.
- Route groups: `/health`, `/forge/*`, `/v1/*` (OpenAI-compat behind config flag), plus operator/system/integration surfaces.
- Per-request middleware: RequestID, RealIP, Logger, Recoverer, CORS gate, optional route-envelope shadow.
- Request bodies guarded by `request_body_decoder_guard_test.go` and per-handler `*_request_body_bounds_test.go`.

### Endpoint naming

`/forge/models/{id}/load`, `/forge/model-runtime/health`, `/forge/system/status` — naming is consistent and resource-oriented within the `/forge` group.

### Request/response models

Models live in [services/core/internal/api/models/](services/core/internal/api/models/). Bounds enforced.

### Error handling

Standard HTTP status codes. Recoverer middleware catches panics. Need to confirm uniform error envelope; spot-check suggests responses are not always wrapped in a consistent error type — worth an audit.

### LLM abstraction

- `modelruntime` is the governed inference boundary.
- Backends: managed llama.cpp, openai-compat.
- External: Ollama adapter (separate package, bounded).
- Optional OpenAI-compat HTTP surface gated by `EnableOpenAICompatAPI` config.

### External dependencies

- **Inbound integrations** scaffolded for Discord and Telegram. Both have bounded payload tests.
- **Outbound HTTP** for gateway tools has SSRF defense + URL byte limit + DNS resolver-aware allowlist (`validateOutboundHTTPURL`, `blockedOutboundIP`).

### Brittleness

- The chat assistant path (`chat_assistant_gateway.go`, 2,497 lines) is the busiest seam between the model and the rest of the system. Worth a structural audit.
- Discord/Telegram integration is wired but I haven't seen evidence of end-to-end live use.

---

## 13. Testing Review

### Existing tests

- **Go core:** 52 packages pass, 0 fail, ~20s wall time. ~1,500+ test functions across the tree.
- **Go-Rust parity:** Rust validator tests + shared fixture validation + `forgek-parity.mjs` script.
- **Desktop:** ~10 Vitest test files (pages and stores).
- **Smoke:** `npm run smoke` boots core against an ephemeral data dir and probes endpoints.
- **Integration env preflight:** `scripts/check-integration-env.mjs`.

### Coverage

| Package | Test funcs | Source funcs | Ratio |
|---|---|---|---|
| `gateway` | 158 | 840 | 18.8% |
| `api` | 241 | 1091 | 22.1% |
| `aios/controllane` | 76 | 628 | 12.1% |
| `forgekshadow` | 64 | 175 | 36.6% |
| `forgek` | 66 | 359 | 18.4% |
| `forgeh` | 34 | 100 | 34.0% |
| `modelruntime` | 50 | 282 | 17.7% |
| `aios/autonomy` | 18 | 196 | 9.2% |
| `aios/dream` | 11 | 64 | 17.2% |
| `memory` | 7 | 107 | 6.5% |

### Untested packages (20)

`adapters`, `aios/hyperlane`, `canvas`, `chat`, `dashboard`, `dossiers`, `evaluations`, `events`, `failurepatterns`, `imports` (partial), `ingest` (partial), `insights`, `lanes`, `lineage`, `packets`, `packetopt`, `policy`, `reconciliation`, `release`, `reviews`, `search`, `strategies`, `watch`.

### Test quality (spot-checked)

- `forgek/kv/kv_manifest_test.go`: real assertions, not smoke-stub.
- `forgeh/resource_policy_test.go`: 288 lines, 9 funcs, table-driven.
- `aios/dream/service_test.go`: 497 lines, scenario-style.
- `gateway/network_fetch_ssrf_test.go`: 10 SSRF cases including AWS metadata IP.
- `embeddings/tei_test.go`: mocks HTTP, reasonable.

### Gaps

- **Memory at 6.5%** is the critical gap.
- **Hyperlane** has zero test files for its core package, while its parser test lives in `gateway/`.
- **Chat path** is one of the busiest and least directly tested.
- **No load tests** that I could find.
- **No fuzz tests** — surprising given the comprehensive bound-checking.

### Proposed test matrix (next sprint)

| Surface | Type | Target |
|---|---|---|
| `memory/` core | Unit | 25% test/source |
| `aios/hyperlane/` | Unit | Intent classification: 100% of declared intents |
| `aios/autonomy/` | Unit | Charter/budget/proposal cycle: 20% test/source |
| `aios/controllane/processor.go` | Unit | 30% test/source |
| `aios/controllane/sqlite_store.go` | Unit + integration | 25% test/source |
| `chat_assistant_gateway.go` | Integration | At least 5 end-to-end happy/edge paths |
| `gateway/service.go` (post-split) | Unit per-tool | 40% test/source |
| Outbound HTTP / SSRF | Fuzz | URL/host fuzzer over allowlist |
| Request body bounds | Fuzz | Existing handlers |
| Smoke | E2E | Existing — add: model load/unload, approval grant/deny, backup/restore |

### CI

Already strong. Add: race detector run (`go test -race ./...` weekly), coverage report artifact, integration env required (don't allow silent skip).

---

## 14. Build, Packaging, and Deployment Review

### Build commands

Comprehensive, in `package.json`:

- `npm run build:core` — VSA preflight + `go build ./...`
- `npm run build:desktop` — Tauri desktop bundle
- `npm run test:core` — VSA preflight + full Go suite
- `npm run vet:core` — VSA preflight + `go vet`
- `npm run test:rust:forgek`, `npm run validate:forgek-fixtures`, `npm run test:forgek:parity`
- `npm run smoke` — boot core against ephemeral dir, probe endpoints
- `npm run docker:up`/`docker:down`/`docker:config` — Compose orchestration
- `npm run desktop:check`/`desktop:clean-tauri`/`desktop:clean-port` — Tauri lifecycle helpers

### Docker

[docker-compose.yml](docker-compose.yml) provisions Postgres, Qdrant, Redis (likely — Section 6 references). [docker-compose.igpu.yml](docker-compose.igpu.yml) is the iGPU variant. `services/core/Dockerfile` exists.

### Nix

Mature relative to the project's age:
- **5 packages:** `forge-core`, `forge-desktop-shell`, `forge-shell-session`, `forge-wayland-session`, `forge-operator-session`.
- **5 NixOS modules:** `forge-host-kernel`, `forge-os`, `forge-services`, `forge-shell-session`, `forge-storage`.
- **2 NixOS profiles:** `forge-operator-desktop`, `forge-vbox-graphics-test`.
- **12 flake checks:** `forge-core-bind-host`, `forge-desktop-shell`, `forge-operator-desktop`, `forge-operator-session`, `forge-shadow-env`, `forge-shell-session`, `forge-vbox-graphics-test`, `forge-wayland-session`, `forge-workspace-default`, `go-tests`, `go-vet`, `js-build`.

### CI

[.github/workflows/ci.yml](.github/workflows/ci.yml) runs the full ladder: Rust FORGE-K validator → fixture validation → Go/Rust parity → integration env preflight → Go tests → Go vet → typecheck → desktop build → smoke. Good.

### Cross-platform

Backend is Go — portable.
Desktop is Tauri — portable but currently exercised on Linux.
Nix substrate is Linux-only by definition.
The README mentions Windows launch parity work; smoke script is bash. Section 31 notes this.

### Deployment assumptions

- Local-first; no cloud deployment story currently in scope.
- VM deployment validated for the operator desktop session.

---

## 15. Documentation Review

### Strengths

- **README is comprehensive** (295 lines) and tracks phase status carefully.
- **AGENTS.md** is 166 lines of operator/agent doctrine with explicit `[LIVE]`/`[SIMULATOR-ONLY]`/`[PARTIAL]` tagging. High quality.
- **11 ADRs** covering the major decisions.
- **40+ architecture docs** — deep coverage.
- **33 status docs** — the *most honest* part of the documentation surface. Files like `docs/status/runtime_truth_vs_docs.md`, `docs/status/placeholders_and_stubs.md`, `docs/status/duplicate_systems.md` are unusually self-critical.
- **7 runbooks** — operator-facing, with [docs/runbooks/current_forge_bringup.md](docs/runbooks/current_forge_bringup.md) and [docs/runbooks/forge_operator_desktop_vm.md](docs/runbooks/forge_operator_desktop_vm.md) being the authoritative operator paths.
- **Diagrams** in Mermaid (`docs/diagrams/`).
- **Glossary** at `docs/glossary.md`.

### Problems

- **ADR numbering collision.** Two ADRs are 0001 (`forge-is-ai-os.md` and `forge-k-is-a-cognitive-microkernel.md`). Renumber one.
- **CODEX.md** is 594 lines and reads as a forward-vision document. It has a self-disclaimer at the top now (good), but the body still uses present-tense language in places that could confuse a reader who skims.
- **Doc surface outruns review capacity.** 40+ arch docs + 33 status docs + 28+ review docs + 11 ADRs + 7 runbooks = a lot. Some review docs (12A through 13H phase reviews) could be archived.
- **Multiple "full review" docs.** `docs/reviews/full_project_forge_review.md`, `full_project_review.md`, `full_project_review_checklist.md`, `forge_full_system_review_20260425/` — overlap.
- **Diagram count is low** for the architectural complexity.

### Recommended structure

```
docs/
  README.md
  architecture/        # keep ~40 files but consolidate near-duplicates
  adr/                 # renumber collision; tag superseded ADRs explicitly
  build/               # NEW — consolidate build/packaging docs
  dev/                 # NEW — onboarding, dev workflow
  api/                 # NEW — route reference, request/response shapes
  ui/                  # NEW — UI patterns, layouts, shell direction
  runtime/             # rename from operations/ if appropriate
  memory/              # docs for memory model — keep simulator/live tagging
  security/            # safety review, dangerous capabilities, env hygiene
  runbooks/            # operator-facing — keep
  status/              # truth-vs-docs — keep, this is gold
  reports/             # one-off reports including this one
  reviews/             # active reviews
    archive/           # NEW — move 12A-13H phase reviews here
  roadmap/             # keep
  testing/             # keep
  diagrams/            # add more
  glossary.md
```

---

## 16. Security and Safety Review

### Secrets

- No obvious hardcoded secrets in source (grep clean for `api[_-]?key|password|secret|token = "..."` with 16+ char values).
- `.env.docker.example` provides reference; real `.env` files are gitignored.

### Env handling

- Wrappers (Nix) set explicit safe-mode defaults: `FORGE_SHELL_SAFE_MODE=true`, `FORGE_SHELL_HOST_MUTATION=false`, `FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false`, several more.
- Forbidden-tokens grep in shell wrappers blocks `systemctl`, `nixos-rebuild`, `LoadModel`, `UnloadModel`, etc.

### Command execution

- Gateway is the *only* path. Tool inputs validated for mode, PID, ref, path containment, symlink escape, bound size.
- `boundedCombinedOutput` + `boundedOutputBuffer` cap captured process output and flag truncation explicitly.
- `normalizeSystemdUnitName`, `normalizeGitCheckoutRef`, `normalizeChmodMode` enforce typed bounds.

### File access

- Workspace path containment via `validateWorkspacePath`.
- Symlink scope tests for both filesystem and command capabilities.
- Archive extraction is scope-checked.

### Outbound network

- `validateOutboundHTTPURL` blocks loopback, RFC1918, link-local, ULA, IPv4-mapped IPv6 internals, AWS metadata IP. URL byte cap.
- Resolver-aware: works against `outboundHTTPResolver` interface, supports test injection.

### Prompt injection surface

- Chat path lives in `chat_assistant_gateway.go` (2,497 lines). Worth a dedicated audit: does any model output flow into a system command, file path, or URL without bounds revalidation?
- The doctrine says model output is a proposal, but in a chatbot path the boundary between proposal and execution is subtle. Recommend Section 31 P1.

### Untrusted input handling

- Request body decoder guard (`request_body_decoder_guard_test.go`) + per-handler body bound tests.
- Source admission test (`source_admission_test.go`) — verify what this enforces.

### Auth

- CORS gate locks origins to localhost/Tauri.
- `X-Forge-Remote-Token` header allowed in CORS — implies a remote-access token mechanism (`api/remote.go`). Audit the token rotation/storage path.

### Authz

- Capability registry; approval-only entries.
- Permissions package + execution permissions UI surface.

### Localhost assumptions

- Default bind is `127.0.0.1`. Wildcard bind logs a `WARNING`. Good — should arguably be a hard refuse without `FORGE_ALLOW_WILDCARD_BIND=true`.

### Dangerous defaults

- Operator desktop session: NOT auto-started by default, NOT auto-login. Cage fallback preserved. Good.
- Postgres/Qdrant/Redis backends: NOT default. SQLite default.

### Findings (action items)

- **Audit `chat_assistant_gateway.go` for prompt-injection → effector paths.** Specifically: does any string flowing from a model response ever land as a workspace path, command argument, URL, or capability-input without going through the existing validators?
- **Wildcard bind: make it fail-closed.** Currently logs a warning; should require `FORGE_ALLOW_WILDCARD_BIND=true` to actually bind.
- **Remote token (`X-Forge-Remote-Token`):** ensure rotation, storage, and revocation paths are tested.

---

## 17. Performance Review

### Blocking / synchronous work

- Gateway tool invocations are synchronous per-request — appropriate.
- Modelruntime calls block per request — appropriate, but consider streaming for chat.
- `compile_context_restore_scoring.go` (1,478 lines) does scoring inline. Worth profiling for hot-path latency.

### Polling

- `fsnotify` is used (`internal/watch/`) — good, event-based rather than polling.

### File reads

- Modelruntime uses bounded JSON file loads (`json_file.go`).
- Migration runs at startup; SQLite migrations are fast enough at current schema size.

### Caching

- KV identity validation is a stated acceleration mechanism; reuse remains simulator-only.
- LRU dependency present (`hashicorp/golang-lru/v2`).

### Frontend

- Vite + React; no obvious render hotspots without profiling.
- 46 pages — code-splitting via `import()` is not visible in `App.tsx` (all pages eagerly imported). Consider `React.lazy` for tier-2 pages.

### Backend

- 188 routes share a single chi mux — fine.
- One observed slow test: `TestGatewayAllRegisteredToolsSmoke` runs the gateway test package for ~5-25 seconds. Tag it with `-short` or split.

### Startup overhead

- Backend boot is sub-second on dev hardware.
- Desktop boot is dominated by Tauri/WebView initialization.

### Memory growth

- Bounded value sizes (1MB) in ephemeral and modelruntime.
- Journal/audit append-only — must be reviewed for retention/rotation as the project ages.

---

## 18. Reliability Review

### Error handling

- Standard Go `if err != nil` patterns throughout.
- Wrapped errors with `%w` in critical paths.
- No `panic` calls in production code (grep clean).

### Recovery

- `middleware.Recoverer` catches HTTP-layer panics.
- Modelruntime backend supervision is partial (M4 item).

### Restart behavior

- SIGINT/SIGTERM with 8s grace.
- Restart is `npm run up` / `forge-core` binary. No supervisor in-tree.

### Data durability

- SQLite WAL mode (likely — need to confirm) + journal pattern.
- Backup service exists.

### Corruption

- Backup tamper tests exist.
- Journal append-only.

### Idempotency

- Per-envelope idempotency in Control Lane.

### Duplicates

- Idempotency table prevents double-apply.

### Logging quality

- Stdlib `log` — basic. Section 10's slog recommendation applies.

### Observability

- Request ID middleware.
- Audit trail.
- No metrics endpoint.

### Graceful shutdown

- Server shutdown with timeout.
- Open question: per-service Close() ordering (`store.Close()` is called via defer; other services rely on the server's `ShutdownWatch`).

### Degraded mode

- CPU-only safe mode documented and partially wired (`docs/architecture/cpu_ram_kernel_gpu_accelerator_split.md`, `docs/runbooks/no_gpu_boot_and_recovery.md`).
- No-GPU boot runbook exists.

---

## 19. Code Quality Review

### Naming

Mostly excellent. Domain vocabulary is consistent (envelope, syscall, validator, capability, gate, lane). A few cases where the naming is *too* internal-doctrine-y for someone new (e.g., "lymphatic", "courthouse", "memory palace") — fine for internal use, but Section 5's public-safe abstraction matters when external developers join.

### File organization

Good per-domain organization at the package level. The monolithic files inside packages (`gateway/service.go`, `chat_assistant_gateway.go`, `sqlite_store.go`) are the exception.

### Type discipline

Strong. Custom types per envelope/result; explicit interfaces (`outboundHTTPResolver`, capability interfaces).

### Modularity

Good at package boundaries. Within `gateway/service.go`, the modularity is conceptual (the capabilities are clearly separated *in idea*) but not physical (they're all in one file).

### Coupling

Low. The forbidden-imports tests enforce this directly.

### Duplication

- 4 components vs 46 pages on the desktop suggests page-level duplication of fetch/error/loading.
- `docs/reviews/full_project_forge_review.md` vs `full_project_review.md` vs `full_project_review_checklist.md` — duplicate review documents.

### Dead code

- `Operator-Toolbelt.txt` (empty).
- Old `phaseN.go` files in `api/` — verify all are still routed before retiring.

### Comments

Sparse, which is consistent with the project's CLAUDE.md guidance ("Default to writing no comments"). Where comments exist they're load-bearing (e.g., the cause of a subtle invariant). Good.

### Complexity

- `compile_context_restore_scoring.go` (1,478 lines) — likely the highest cyclomatic complexity hotspot.
- `gateway/service.go` complexity is breadth, not depth — many small handlers in one file.

### Consistency

Strong. The repo reads like it had a single author plus consistent LLM collaboration, which it does.

### Maintainability

Average-to-good. The big files are the main maintainability risk. Coverage gaps are the second.

---

## 20. What Is Working

Evidence by file:

- **HTTP server lifecycle** — [services/core/main.go](services/core/main.go).
- **Route mounting** — [services/core/internal/api/routes.go](services/core/internal/api/routes.go) (188 routes registered, exercised by smoke).
- **Transactional semantic commit** — [services/core/internal/aios/controllane/processor.go](services/core/internal/aios/controllane/processor.go), tested in `processor_test.go`.
- **Bounded tool execution** — [services/core/internal/gateway/service.go](services/core/internal/gateway/service.go) + 46 test files.
- **SSRF defense** — [services/core/internal/gateway/network_fetch_ssrf_test.go](services/core/internal/gateway/network_fetch_ssrf_test.go) (10 cases).
- **Capability registry + approval-only entries** — [services/core/internal/gateway/tool_capability_registry.go](services/core/internal/gateway/tool_capability_registry.go).
- **Live KV identity validation** — [services/core/internal/kvidentity/](services/core/internal/kvidentity/) + Control Lane `VALIDATE_KV_IDENTITY`.
- **Live ref shape / semantic op validation** — `refvalidation`, `semanticvalidation` packages.
- **Audit immutability** — [services/core/internal/audit/immutable_test.go](services/core/internal/audit/immutable_test.go).
- **Approvals service** — [services/core/internal/approvals/service.go](services/core/internal/approvals/service.go).
- **Modelruntime lifecycle (M3)** — [services/core/internal/modelruntime/](services/core/internal/modelruntime/) + `/forge/models/*` routes.
- **Backup/restore** — [services/core/internal/backup/service.go](services/core/internal/backup/service.go).
- **SQLite store + migrations** — [services/core/internal/store/](services/core/internal/store/).
- **Tauri desktop app** — [apps/desktop/](apps/desktop/), boots and renders.
- **Multi-monitor desktop shell** — [apps/desktop/src/stores/desktopShellStore.ts](apps/desktop/src/stores/desktopShellStore.ts).
- **Operator Desktop NixOS session** — [nix/packages/forge-operator-session.nix](nix/packages/forge-operator-session.nix), VM-validated.
- **Forbidden-imports fences** — `forbidden_live_imports_test.go` files across `forgek`, `forgekshadow`, `forgeh`, `refvalidation`, `semanticvalidation`, `hostbridge`.
- **CI pipeline** — [.github/workflows/ci.yml](.github/workflows/ci.yml), runs Go test/vet, Rust validator, parity, typecheck, desktop build, smoke.
- **VSA dependency preflight** — `scripts/check-vsa-files.sh` + npm script wiring.

---

## 21. What Is Partially Working

- **FORGE-K live authority migration.** Four validation syscalls live; full mutation still through legacy AI-OS paths. Pattern is proven; cadence is slow.
- **Postgres / Qdrant / Redis backends.** Schemas and shadow adapters exist; not the live default. Phase 13I marked `READINESS_REVIEW` only.
- **Autonomy lanes.** Charters/budgets scaffolded, propose-only, not exercised at scale.
- **Hyperlane intent classification.** 114-line enum + route table; gateway parser tests exist; core package has no tests; classifier not yet routing live traffic.
- **Streaming model output.** Modelruntime M4 deferred.
- **Operator apps allowlist.** Foot + pcmanfm working; Firefox path partial; Tauri-side `launch_operator_app` is invoked but Rust implementation needs end-to-end review.
- **Discord / Telegram integration.** Wired with bounded payload tests; live operator use not in evidence.
- **System status surface.** `SystemPage.tsx` exists and renders; underlying `system_status.go` is 271 lines, reasonable; not all subsystems report through it yet.
- **VM session.** Manual boot succeeded; no CI proof; no recorded session log/screenshot in repo evidence beyond the operator's word.
- **Lymphatic / cleanup lane.** Simulator-only — produces proposals; no live execution.

---

## 22. What Is Stubbed or Placeholder

This is unusually short — the repo is clean.

- **`Operator-Toolbelt.txt`** — empty file at repo root.
- **FORGE-H adapter interfaces** (`OperatorNotifier`, `LanePolicyWriter`, `ModelPolicyWriter`, `DegradedModeWriter` in `services/core/internal/forgeh/executor.go`) — defined as interfaces with no concrete implementations in-tree. Intentional but worth noting as planned work.
- **Hyperlane core package** — has `intent.go` with enums and a `ParserVersion`, no tests, classifier not routed live.
- **Aios/rulecells** — scaffolded; deterministic reflex substrate documented as concept, code is placeholder.
- **Cleanup proposal agent** (per `docs/status/placeholders_and_stubs.md`) — guarded off by default.
- **Lane mutation endpoints** — explicitly retired with `410 Gone` responses (intentional placeholder for "this used to be a thing").

No TODO/FIXME/HACK markers in services/core/internal. The placeholders are tracked in `docs/status/placeholders_and_stubs.md`, which is honest about what's deferred.

---

## 23. What Appears Broken

Nothing currently failing. Tests pass, vet clean, smoke green. The items below are *risks of breaking* or *latent issues*, not active breakage.

### Issue: VM boot validation has no in-repo evidence

**Location:** [docs/runbooks/forge_vm_handoff_context.md](docs/runbooks/forge_vm_handoff_context.md)
**Problem:** The handoff doc states the VM session was started but not completed before shutdown, and never reached the desktop/labwc integration layer. The operator confirmed verbally and via screenshot that it now boots, but no log/screenshot/test artifact lives in the repo.
**Impact:** A future regression in the labwc/session chain could pass all flake checks (which only verify static safe-defaults) and still fail at runtime.
**Suggested Fix:** Add a manual-validation artifact (screenshot + commit hash + date) to `docs/runbooks/forge_operator_desktop_vm.md`. Consider a `nix run` smoke for the operator session that boots a headless Wayland and probes that the FORGE window opens.
**Difficulty:** Easy.

### Issue: Wildcard bind only warns

**Location:** [services/core/main.go:29-32](services/core/main.go)
**Problem:** Binding to `0.0.0.0`/`::` logs a `WARNING` but proceeds. For a local-first service, this is a footgun.
**Impact:** Operator misconfiguration could expose the daemon to the network.
**Suggested Fix:** Refuse to bind unless `FORGE_ALLOW_WILDCARD_BIND=true` is also set.
**Difficulty:** Easy.

### Issue: ADR numbering collision

**Location:** [docs/adr/0001-forge-is-ai-os.md](docs/adr/0001-forge-is-ai-os.md) + [docs/adr/0001-forge-k-is-a-cognitive-microkernel.md](docs/adr/0001-forge-k-is-a-cognitive-microkernel.md)
**Problem:** Two ADRs share number 0001.
**Impact:** Cross-references in README, AGENTS.md, and elsewhere become ambiguous.
**Suggested Fix:** Renumber the older one (`forge-is-ai-os.md`) to 0000 or move FORGE-K to a fresh number.
**Difficulty:** Easy.

### Issue: `chat_assistant_gateway.go` may have prompt-injection → effector paths

**Location:** [services/core/internal/api/chat_assistant_gateway.go](services/core/internal/api/chat_assistant_gateway.go) (2,497 lines)
**Problem:** This is the busiest seam between LLM output and the rest of the system. The doctrine says model output is a proposal, but in a chat loop the boundary between "model said this" and "this happens" needs explicit auditing.
**Impact:** A model-controlled string could land in a workspace path, capability input, or URL if any path doesn't go through existing validators.
**Suggested Fix:** Dedicated audit pass: trace every assignment from a model response field to (a) command args, (b) file paths, (c) URLs, (d) capability inputs. Confirm validator coverage.
**Difficulty:** Medium.

### Issue: VM artifacts and empty file in repo root

**Location:** `/.vm-build-core.log`, `/.vm-nix-store/`, `/.vm-nix-tmp/`, `/Operator-Toolbelt.txt`
**Problem:** Untracked but not gitignored. `Operator-Toolbelt.txt` is 0 bytes.
**Impact:** Pollutes `git status`. Possible accidental commit later.
**Suggested Fix:** Add to `.gitignore`. Delete or fill `Operator-Toolbelt.txt`.
**Difficulty:** Trivial.

### Issue: `npm run smoke` is bash; CI uses Linux. Windows parity claim in README.

**Location:** `scripts/forge-smoke.mjs` (probably calls bash) + README mention of Windows launch parity.
**Problem:** Smoke isn't cross-platform yet.
**Impact:** Windows operators can't run the validation entry point.
**Suggested Fix:** Port the smoke script to pure Node.
**Difficulty:** Easy-Medium.

### Issue: 1,478-line `compile_context_restore_scoring.go`

**Location:** [services/core/internal/aios/controllane/compile_context_restore_scoring.go](services/core/internal/aios/controllane/compile_context_restore_scoring.go)
**Problem:** Single file with a substantial scoring system; hard to review under load.
**Impact:** Latent correctness bugs harder to catch; onboarding cost.
**Suggested Fix:** Split by scoring concern (candidate listing, ranking, threshold, fallback, persistence).
**Difficulty:** Medium.

---

## 24. What Is Duplicated or Confusing

- **Duplicate review docs:** `docs/reviews/full_project_forge_review.md`, `full_project_review.md`, `full_project_review_checklist.md`, plus `forge_full_system_review_20260425/` and a `.zip` of same. Keep one as the canonical historical review and move others to `docs/reviews/archive/`.
- **ADR 0001 collision** (see §23).
- **Multiple architecture docs near the same concept:** `forge_ai_os.md`, `forge_k_overview.md`, `core_doctrine.md`, `control_lane_kernel.md` all touch the same kernel idea from different angles. Add a top-of-file `Read this if you want X` note to each, or consolidate into a single "kernel architecture" doc with sections.
- **CODEX.md vs AGENTS.md vs README.md vs CLAUDE.md** — four doctrine surfaces. AGENTS.md is the operator/agent doctrine; CLAUDE.md is a 20-line pointer; CODEX.md is forward vision (tagged `[FUTURE]`); README.md is public-facing. The tagging is now correct, but a reader's first question "which one do I read first?" deserves a `docs/onboarding.md` answer.
- **Phase planning files vs phase review files:** Plans live in `docs/superpowers/specs/` and `docs/archive/phases/`; reviews live in `docs/reviews/`. The mapping is mostly clean but could be cross-linked.
- **Empty placeholder files:** `Operator-Toolbelt.txt`.
- **Old phase reviews accumulating in `docs/reviews/`:** Move 12A-13H phase reviews to `docs/reviews/archive/` to keep the active surface readable.

---

## 25. Architectural Risks

Ranked by severity.

1. **Gateway monolith.** A 4,709-line `service.go` is the single largest architectural smell. Today it's reviewable; in three months with another 30% surface area added, it won't be.
2. **Memory package coverage.** 6.5% test/source on the canonical-truth store. The kernel commit gate is well-tested; the thing the kernel commits *to* is not.
3. **Simulator/live drift risk.** Four migrations done, dozens of simulator subsystems remain. If the pace of simulator additions exceeds the pace of live migrations indefinitely, the simulator becomes a museum and the live daemon stays on legacy AI-OS paths.
4. **Doctrine surface area.** AGENTS.md + CODEX.md + README + 40 arch docs + 33 status docs is a lot of doctrine for a project that one person + LLMs is building. The most honest docs (status files) are the least linked from the top-level surfaces.
5. **Chat path as injection surface.** `chat_assistant_gateway.go` is the busiest LLM↔system seam and the largest API file. Worth a dedicated audit.
6. **No structured logging / no metrics.** When (not if) FORGE has to be debugged in flight, `log.Printf` won't be enough.
7. **No load tests, no fuzz tests.** The bound-checking work is extensive, but bound-checking is the kind of thing fuzz tests find new edge cases for.
8. **Operator Toolbelt is empty.** Placeholder for a real operator interaction surface — needs scoping or removal.
9. **Hyperlane intent classifier is in production-doc language but lab-grade code.** 114 lines of enums + no tests + zero classifier routing.
10. **CI doesn't run integration env required.** The shadow paths (Postgres, Qdrant, Redis) skip silently without env vars. Make required for at least the parity test surface.

---

## 26. Refactor Recommendations

Ordered low-risk, high-impact first.

### Refactor Item: Split `gateway/service.go`

**Why:** 4,709 lines is the project's largest single-file complexity hotspot.
**Files Affected:** [services/core/internal/gateway/service.go](services/core/internal/gateway/service.go) split into ~12 capability-category files.
**Expected Benefit:** Faster review, faster onboarding, smaller diffs, easier targeted testing.
**Risk:** Low — pure refactor with existing tests as safety net.
**Difficulty:** Medium (mechanical work; testing during the split is easy).

### Refactor Item: Split `chat_assistant_gateway.go`

**Why:** 2,497 lines in the busiest LLM↔system seam.
**Files Affected:** [services/core/internal/api/chat_assistant_gateway.go](services/core/internal/api/chat_assistant_gateway.go) into request handling, response handling, tool routing, model selection, validator chain.
**Expected Benefit:** Same as above + reduces prompt-injection audit cost.
**Risk:** Low-Medium.
**Difficulty:** Medium.

### Refactor Item: Promote `api/` to `api/handlers/` by domain

**Why:** 188 routes flat in one package is hard to navigate.
**Files Affected:** `services/core/internal/api/*.go` into subpackages: `handlers/models/`, `handlers/autonomy/`, `handlers/chat/`, `handlers/system/`, `handlers/integrations/`, `handlers/phases/archive/`.
**Expected Benefit:** Clearer ownership; isolated test suites per domain; `routes.go` stays slim.
**Risk:** Low.
**Difficulty:** Medium.

### Refactor Item: Coverage triage on memory, autonomy, hyperlane

**Why:** Three load-bearing packages with <20% coverage.
**Files Affected:** Tests under `services/core/internal/memory/`, `services/core/internal/aios/autonomy/`, `services/core/internal/aios/hyperlane/`.
**Expected Benefit:** Bring coverage to 25%+ on each; raise confidence in the parts the kernel commits.
**Risk:** None.
**Difficulty:** Medium.

### Refactor Item: Adopt `log/slog` + correlation IDs in log lines

**Why:** Runtime observability is currently `log.Printf`-grade.
**Files Affected:** Logger middleware + every service that logs.
**Expected Benefit:** Real structured logs; trivial integration with future metrics/tracing.
**Risk:** Low — drop-in for `log` calls.
**Difficulty:** Medium (touch surface is wide).

### Refactor Item: Lazy-load tier-2 desktop pages

**Why:** 46 eager imports in `App.tsx` is a bundle-size and startup cost.
**Files Affected:** [apps/desktop/src/App.tsx](apps/desktop/src/App.tsx).
**Expected Benefit:** Faster cold start in Tauri.
**Risk:** Low — `React.lazy` + Suspense.
**Difficulty:** Easy.

### Refactor Item: Extract shared desktop components

**Why:** 46 pages, 4 shared components. Cross-page duplication of fetch/error/loading patterns is inevitable.
**Files Affected:** Audit `apps/desktop/src/pages/*.tsx` for repeated patterns; lift into `apps/desktop/src/components/`.
**Expected Benefit:** Smaller pages, consistent UX, fewer bugs.
**Risk:** Low.
**Difficulty:** Medium.

### Refactor Item: Renumber the duplicate 0001 ADR

**Why:** Cross-references are ambiguous.
**Files Affected:** [docs/adr/0001-forge-is-ai-os.md](docs/adr/0001-forge-is-ai-os.md) → `0000-forge-is-ai-os.md` (or new number). Update all cross-references.
**Risk:** Low — text refactor.
**Difficulty:** Easy.

### Refactor Item: Archive old phase review docs

**Why:** `docs/reviews/` has 28+ files, most historical.
**Files Affected:** Move `phase_12*`, `phase_13*` reviews to `docs/reviews/archive/phase_12/`, `phase_13/`.
**Risk:** None.
**Difficulty:** Trivial.

---

## 27. Server / Backend Split Plan

The proposed structure adapted to Go and the current package layout. None of this requires moving packages out of `services/core/internal/`; this is *file-level* reorganization within existing packages.

```
services/core/
  main.go                                 # keep
  go.mod
  internal/
    config/                               # keep
    store/
      store.go
      backend.go
      contracts.go
      sqlite/                             # NEW subpackage
        sqlite.go
        migrations/                       # NEW — split migrate.go by version
          v001_initial.go
          v002_journal.go
          ...
      postgres/                           # NEW subpackage
        postgres.go
        migrations/
    storagebackend/                       # keep
    ephemeral/                            # keep
    vectorstore/                          # keep
    api/
      server.go                           # constructor + shared helpers
      routes.go                           # route mounting (keep)
      handlers/                           # NEW
        chat/
          chat_post.go
          chat_assistant_gateway/         # split this 2,497-line file
            handler.go
            request.go
            response.go
            tool_routing.go
            validator_chain.go
        models/
          model_runtime.go
          model_runtime_bridge.go
        autonomy/
          autonomy_api.go
          maintenance_loop/               # split 1,545-line file
        system/
          system_status.go
          health.go
        integrations/
          discord_*.go
          telegram_*.go
        backup/
        approvals/
        archive/                          # NEW — old phaseN.go files
          phase2.go ... phase5.go
      middleware/                         # NEW
        request_body_guard.go
        route_envelope_shadow.go
    gateway/                              # split service.go here
      service.go                          # constructor + Invoke
      registry.go
      tools_filesystem.go
      tools_command.go
      tools_archive.go
      tools_git.go
      tools_http.go
      tools_identity.go
      tools_secret.go
      tools_system.go
      tools_observability.go
      tools_chat.go
      tools_desktop_bridge.go
      validation_url.go
      validation_paths.go
      validation_modes.go
      output_buffer.go
      hyperlane_intent_parser.go          # keep
    aios/
      controllane/
        processor.go                      # keep — this is the kernel gate
        processor_apply.go
        registry.go
        validator.go
        capabilities.go
        audit.go
        sqlite_store/                     # NEW — split 2,244-line file
          journal.go
          memory.go
          links.go
          state.go
          loops.go
          contradictions.go
        kv_enforcement.go
        ref_validation.go
        ref_shape_compare.go
        semantic_operation_validation.go
        compile_context_restore_scoring/  # NEW — split 1,478-line file
          listing.go
          ranking.go
          threshold.go
          fallback.go
          persistence.go
      autonomy/
      dream/
      hyperlane/
        intent.go                         # add classifier
        intent_test.go                    # add tests
        parser.go
        parser_test.go
      truth/
      compute/
      rulecells/
    modelruntime/
      service.go
      backends/                           # NEW
        llama_cpp.go
        openai_compat.go
      store/                              # split store_management.go
      manifest.go
      state.go
      scheduler.go
      limits.go
      policy.go
    memory/                               # focus coverage here
    retrieval/
    embeddings/
    backup/
    approvals/
    audit/
    permissions/
    jobs/
    automation/
    forgek/                               # keep — simulator
    forgekshadow/                         # keep — observers
    forgeh/                               # keep — advisory
    hostbridge/                           # keep — read-only
    kvidentity/                           # keep — shared pure
    refvalidation/                        # keep — shared pure
    semanticvalidation/                   # keep — shared pure
    adapters/
      ollama/
        ollama.go
        ollama_chat.go
    chat/
    canvas/
    dashboard/
    dossiers/
    evaluations/
    events/
    insights/
    lineage/
    packets/
    policy/
    projectcontext/
    release/
    reviews/
    search/
    strategies/
    watch/
```

This isn't a wholesale rearrangement — it's targeted splits of the largest files and one new `handlers/` subpackage under `api/`. Most package boundaries stay.

---

## 28. UI Improvement Plan

The current UI is closer to a structured dashboard than an AI operating shell. The Operator Apps page and SystemPage are the steps in the right direction. To finish the alignment:

**Shell layout.** Adopt a persistent left rail (current `AppShell` likely does this) + a bottom command bar + a contextual right-side inspector. The right-side inspector is the place an LLM's proposals/context can live while the operator works in the main pane.

**Window/workspace model.** Already strong. Multi-monitor support exists. Named layouts persist.

**Navigation model.** Three layers:
1. Cmd/Ctrl+K command palette (looks like `CommandBar.tsx` exists — invest in it).
2. Left rail for stable surfaces.
3. URL/route per surface (already done).

**Status bar.** Bottom strip showing: model runtime status, autonomy lane state, last journal entry, network status, current workspace. One line. SystemPage already has the data — surface a one-line summary always.

**Service monitor.** Promote `SystemPage` content into an "always-on" peek surface (collapsible) rather than a page you navigate to.

**Memory/context inspector.** Right-side panel that shows the current context being compiled, the recent journal entries, and the active loops/approvals. The data exists; the surface doesn't yet.

**Activity log.** Bottom-right popover or status-bar accordion showing the last 20 audit events. Tied to `events/` and `audit/`.

**Settings/config surface.** `SettingsPage` exists. Audit for completeness vs `config.go` field set.

**Theme system.** Adopt a minimal CSS-var-driven theme (light/dark + accent). Each Page renders against the same vars.

**Responsive behavior.** The shell is Tauri-native; responsive isn't a primary concern, but window-resize behavior should be tested.

**Error/loading states.** Lift to a shared `<AsyncState>` wrapper. 46 pages writing their own is ~46 inconsistencies.

**Accessibility.** Audit focus management, keyboard nav, ARIA labels. Especially on the command bar and approval flows.

---

## 29. Documentation Cleanup Plan

```
docs/
  onboarding.md                            # NEW — "where do I start as a new dev?"
  README.md                                # exists
  architecture/                            # keep, consolidate near-duplicates
    forge_ai_os.md                         # keep — top-level
    forge_k_overview.md                    # keep — kernel
    control_lane_kernel.md                 # keep — kernel commit gate
    core_doctrine.md                       # merge subset into forge_ai_os.md or onboarding.md
    lane_model.md
    hyperlane.md
    memory_palace_and_courthouse.md
    semantic_algebra.md
    snapshots.md
    consensus_mesh.md
    forge_h_resource_policy.md
    forge_host_kernel_bridge.md
    forge_os_host_substrate.md
    forge_graphical_shell.md
    cpu_ram_kernel_gpu_accelerator_split.md
    storage_backend_migration.md
    ...
  adr/
    0000-forge-is-ai-os.md                 # renumbered
    0001-forge-k-is-a-cognitive-microkernel.md
    0002 ... 0013
  build/                                   # NEW
    nix.md                                 # current nix_foundation_status.md
    docker.md                              # current docker_containerization.md
    desktop.md                             # current desktop_bringup.md
  dev/                                     # NEW
    workflow.md
    coding_standards.md
    testing.md
  api/                                     # NEW
    routes.md                              # generated from chi route inventory
    requests.md
    errors.md
  ui/                                      # NEW
    shell_layout.md
    components.md
    theme.md
  runtime/                                 # rename from operations/
    bringup.md                             # current current_forge_bringup.md
    runbooks/
      operator_desktop_vm.md
      no_gpu_boot_and_recovery.md
      performance_and_latency.md
      config_reference.md
  memory/                                  # NEW
    model.md
    persistence.md
    snapshots.md
  security/                                # NEW
    dangerous_capabilities.md
    safe_defaults.md
    audit.md
  status/                                  # keep — this is gold
  reviews/
    archive/
      phase_12/
      phase_13/
    active/                                # only current-cycle reviews
  reports/
    FORGE_FULL_REVIEW.md                   # this report
  roadmap/
  testing/
  diagrams/                                # add more
  glossary.md
```

Specific moves:
- Move `docs/reviews/phase_12*.md`, `phase_13*.md` to `docs/reviews/archive/`.
- Delete or archive `full_project_forge_review.md`, `full_project_review.md`, `full_project_review_checklist.md`; keep the dated review folder.
- Renumber `docs/adr/0001-forge-is-ai-os.md`.
- Add `docs/onboarding.md` as the single answer to "where do I start?"
- Add `docs/api/routes.md` generated from chi route inventory.

---

## 30. Testing and CI Plan

### Unit tests

Add to packages currently at <20%:

- `memory/` — top 10 functions, target 25%.
- `aios/controllane/processor.go` — target 30%.
- `aios/controllane/sqlite_store.go` — target 25%.
- `aios/autonomy/` — target 20%.
- `aios/hyperlane/` — bring from 0 to 30%.

### Integration tests

- `chat_assistant_gateway` end-to-end happy path + edge cases.
- Model runtime load/unload cycle with fixture model.
- Approval grant/deny lifecycle.
- Backup → restore round-trip with state delta verification.
- Postgres parity (when env available).

### API tests

- Generate route inventory from `chi` and assert all 188 routes have at least a smoke test.
- Request body bound tests already strong; extend to remaining handlers.

### UI tests

- `apps/desktop/src/pages/` — at least a render test for each page (Vitest).
- Stores already partially tested; extend.
- Snapshot tests for `AppShell` layout variants.

### Smoke tests

Keep current `npm run smoke`. Add:
- Model load/unload smoke.
- Approval grant/deny smoke.
- Memory note create + retrieve smoke.

### Linting / formatting / typechecks

- Go: `gofmt` (assumed enforced by editor) + `go vet`.
- TypeScript: `tsc --noEmit` (already in `npm run typecheck`).
- ESLint/Prettier — not visible; recommend adding if not present.
- Rust: `cargo fmt --check`, `cargo clippy` — recommend adding if not present.
- Nix: `nix fmt` formatter (existing flake formatter check).

### CI additions

- Race detector: `go test -race ./...` on weekly cron.
- Coverage report artifact upload.
- Integration env required for the integration test job (don't allow silent skip).
- Fuzz test job (Go 1.18+ native fuzz) on the validators.
- Manual operator-desktop boot artifact gate (screenshot + log) when `nix/packages/forge-operator-session.nix` or `nix/nixos/profiles/forge-operator-desktop.nix` changes.

### Local developer test command

Already strong: `npm test`. Recommend `npm run validate:all` that runs the full ladder locally.

---

## 31. Priority Punch List

| Priority | Task | Area | Why It Matters | Difficulty |
|---|---|---|---|---|
| P0 | Audit `chat_assistant_gateway.go` for prompt-injection → effector paths | Security | Largest LLM↔system seam, model-controlled strings could land in capability inputs | Medium |
| P0 | Bring `memory/` test/source ratio from 6.5% to 25% | Reliability | Canonical-truth store with thinnest coverage | Medium |
| P0 | Split `gateway/service.go` along capability-category lines | Maintainability | 4,709 lines in one file gates every future gateway change | Medium |
| P1 | Resolve ADR 0001 numbering collision | Docs | Cross-references are ambiguous | Easy |
| P1 | Make wildcard bind fail-closed | Security | Operator misconfiguration footgun | Easy |
| P1 | Add VM boot evidence to operator-desktop runbook | Reliability | Static checks don't catch runtime regressions | Easy |
| P1 | Coverage triage on `aios/controllane/processor.go` to 30% | Reliability | The kernel commit gate deserves better tests | Medium |
| P1 | Hyperlane core package: add tests + start routing live traffic | Runtime | Intent classifier sitting unused | Medium |
| P1 | Split `chat_assistant_gateway.go` | Maintainability | 2,497-line LLM seam | Medium |
| P1 | Adopt `log/slog` + correlation IDs | Observability | Runtime debug story is currently `log.Printf` | Medium |
| P2 | Archive old phase review docs | Docs | `docs/reviews/` is approaching read-fatigue | Trivial |
| P2 | Split `aios/controllane/sqlite_store.go` (2,244 lines) | Maintainability | Repository concerns already separated in code, not in file | Medium |
| P2 | Split `compile_context_restore_scoring.go` (1,478 lines) | Maintainability | Hottest scoring path in one file | Medium |
| P2 | Extract shared desktop components | UI | 46 pages, 4 components — duplication tax | Medium |
| P2 | Lazy-load tier-2 desktop pages | Performance | Cold-start cost | Easy |
| P2 | Add `/metrics` endpoint behind config flag | Observability | Standard runtime observability | Easy |
| P2 | Add `/health/detailed` with per-service rollup | Reliability | Operator visibility under degraded mode | Easy |
| P2 | Make CI integration env required (no silent skip) | Reliability | Shadow-path regressions can land invisibly | Easy |
| P2 | Cross-platform smoke (port from bash to Node) | DX | Windows operators currently locked out | Easy-Medium |
| P3 | Add fuzz tests for validators (URL, mode, path, ref) | Security | Bound checks deserve fuzz coverage | Medium |
| P3 | Add a Tauri-side end-to-end test of operator-app launch | Reliability | Allowlist enforcement test path | Medium |
| P3 | Theme system + design system primitives | UI | Long-term consistency | Medium |
| P3 | Onboarding doc | DX | "Where do I start as a new dev?" answer | Easy |
| P3 | Renumber and tag superseded ADRs explicitly | Docs | Clarity | Easy |
| P3 | Delete or fill `Operator-Toolbelt.txt`; gitignore VM artifacts | Hygiene | Repo cleanliness | Trivial |

---

## 32. Difficulty-Ranked Fix List

### Easy Fixes
- Resolve ADR 0001 numbering collision.
- Make wildcard bind fail-closed (`FORGE_ALLOW_WILDCARD_BIND` required).
- `.gitignore` the VM artifacts (`.vm-*`, `.vm-nix-store/`, `.vm-nix-tmp/`).
- Delete or fill `Operator-Toolbelt.txt`.
- Lazy-load tier-2 desktop pages.
- Add `/metrics` endpoint behind config flag.
- Add `/health/detailed` with per-service rollup.
- Make CI integration env required.
- Archive old phase review docs to `docs/reviews/archive/`.
- Add VM boot evidence to operator-desktop runbook.
- Add `docs/onboarding.md`.

### Medium Fixes
- Split `gateway/service.go` along capability-category lines.
- Split `chat_assistant_gateway.go`.
- Split `aios/controllane/sqlite_store.go`.
- Split `compile_context_restore_scoring.go`.
- Coverage triage on `memory/`, `aios/controllane/`, `aios/autonomy/`, `aios/hyperlane/`.
- Adopt `log/slog` + correlation IDs throughout.
- Extract shared desktop components.
- Cross-platform smoke port.
- Audit `chat_assistant_gateway.go` for prompt-injection → effector paths.
- Hyperlane: tests + live classifier routing.

### Hard Fixes
- Bring autonomy lanes to live (charter/budget/approval round-trip exercised at scale).
- Postgres canonical-truth cutover.
- Streaming model output (Model Runtime M4).
- FORGE-K full mutation authority migration for one subsystem end-to-end (next narrow seam after the four already done).
- Theme/design system overhaul.

### Deep Architecture Work
- Full simulator-to-live migration of one cognitive lane (lymphatic cleanup, consensus mesh, or rule cells) — this is the project's North Star.
- Real continuous perception surface (mic/screen/camera) feeding hyperlane intent.
- Effectors layer beyond advisory FORGE-H — adapter implementations that actually mutate (with approval gates).
- Multi-host federation (FORGE on multiple boxes, governed identity, cross-host syscalls).

---

## 33. Suggested Next Build Phases

### Phase 0 — Stabilize and Map
**Goal:** Resolve quick wins; make CI strict; document onboarding.
**Deliverables:**
- ADR renumber. Wildcard-bind fail-closed. Gitignore VM artifacts. Empty file deleted.
- CI integration env required.
- `docs/onboarding.md`.
- VM-boot evidence committed to operator-desktop runbook.
- Phase reviews archived under `docs/reviews/archive/`.
**Definition of done:** All P1 hygiene items closed. CI fails when integration env is missing.
**Risks:** Trivial; nothing material to break.

### Phase 1 — Split and Structure
**Goal:** Cut the three biggest files.
**Deliverables:**
- `gateway/service.go` split into ~12 category files.
- `chat_assistant_gateway.go` split.
- `aios/controllane/sqlite_store.go` split.
**Definition of done:** No single Go file >1,500 lines.
**Risks:** Refactor regressions. Mitigated by the existing test suite + commit-per-split discipline.

### Phase 2 — Runtime Hardening
**Goal:** Observability + memory coverage + chat audit.
**Deliverables:**
- `log/slog` migration across the daemon.
- `/metrics`, `/health/detailed` endpoints.
- `memory/` to 25% coverage.
- `chat_assistant_gateway.go` prompt-injection audit + any required validator fixes.
**Definition of done:** Operator can see daemon health from one endpoint and trace any request via correlation ID across logs.
**Risks:** Audit may surface findings requiring code changes; budget for that.

### Phase 3 — Hyperlane to Live
**Goal:** Move the intent classifier from "lab grade" to "routing real traffic."
**Deliverables:**
- Tests on `aios/hyperlane/intent.go`.
- Classifier consumed by chat path and gateway for routing decisions.
- Live metrics on intent distribution.
**Definition of done:** ≥80% of incoming chat messages classified deterministically; remainder fall through to model with telemetry.
**Risks:** Wrong classification → wrong route. Mitigated by shadow-mode first.

### Phase 4 — Memory/State Contract
**Goal:** Make memory the second well-tested package (after gateway).
**Deliverables:**
- Memory at 35% coverage.
- Public-safe lifecycle doc.
- Snapshot-vs-truth contract explicitly tested.
**Definition of done:** Memory has a published contract and tests proving each path.
**Risks:** Discovering existing bugs (good outcome, but adds work).

### Phase 5 — UI Shell Alignment
**Goal:** Move from dashboard-shape to shell-shape.
**Deliverables:**
- Status bar with persistent runtime/autonomy/journal indicators.
- Right-side context inspector.
- Activity log accordion.
- Theme variables; shared async state wrapper.
- Lazy-loaded tier-2 pages.
**Definition of done:** Operator can run a full workflow inside the shell without leaving for terminal/browser most of the time.
**Risks:** UI scope creep. Mitigate by scoping to current pages, not new ones.

### Phase 6 — Testing and CI Depth
**Goal:** Race detector, fuzz, integration depth.
**Deliverables:**
- `go test -race` weekly.
- Fuzz tests on URL, path, mode, ref validators.
- Postgres parity test job in CI.
- Operator-desktop boot smoke (headless Wayland probe).
**Definition of done:** CI catches regressions in race, fuzz, parity, and operator-desktop boot.
**Risks:** Flakiness from headless Wayland — keep that test isolated.

### Phase 7 — Packaging and Deployment
**Goal:** Cross-platform parity + reproducible artifacts.
**Deliverables:**
- Windows-compatible smoke + bringup.
- Nix-built operator-desktop release artifact.
- Docker image with the same wrapper safety defaults.
**Definition of done:** Operator can boot FORGE on Linux + Windows from a single instruction sheet.
**Risks:** Tauri/Windows quirks; budget time.

---

## 34. Public-Facing FORGE Description

### Short Version
FORGE is a local-first AI workspace where every meaningful state change is inspectable, gated, and reversible. Instead of trusting a model's output, FORGE separates what a model *says* from what the system *does* — model output becomes a proposal, deterministic validators check it, and a single transactional commit boundary records the result.

### Technical Version
FORGE is a Go-based local AI runtime with a Tauri desktop shell, packaged for NixOS as a session-locked operator surface. The runtime exposes a structured HTTP API for tool execution, model lifecycle, semantic state mutation, approval workflows, audit, backup/restore, and integrations. State persists to SQLite by default with a Postgres parity path. The gateway is the only governed tool execution surface and enforces input bounds, SSRF defense, path containment, and a capability registry. A single semantic-commit chokepoint wraps every canonical mutation in approval → validate → apply → journal → commit, all transactional. The simulator/live boundary is enforced at build time by forbidden-imports tests and Nix-level safe-mode literals.

### Builder Version
FORGE is what happens when a single operator (with consistent LLM collaboration) decides to treat AI safety as an engineering problem, not a vibe. The kernel commit gate is real, the import fences are tested, the Nix derivations hardcode the safety properties, and the runtime can boot as a desktop shell inside a NixOS VM. Coverage is uneven, the largest file is 4,709 lines, and there are honest gaps in memory testing and autonomy exercise — but the bones are correct and the project knows where it stands. Status documents in `docs/status/` track simulator-vs-live truth explicitly. Builders who want to add a tool, a syscall, or a UI surface have a clear pattern to follow.

### What Makes FORGE Different
- **Build-time-enforced doctrine.** Forbidden-imports tests, Nix `allowMutation=false` literals, and grep'd wrapper scripts make safety properties enforceable, not aspirational.
- **Single semantic commit chokepoint.** Every canonical state mutation goes through one transactional gate with explicit approval/capability/validator/idempotency ordering.
- **Simulator-first migration pattern.** New cognitive subsystems live in a hermetically sealed simulator with shared pure validator packages, then migrate narrowly into the live daemon under capability gates. Four such migrations done; the pattern is proven.
- **Operator-owned substrate.** Boots as a NixOS Wayland session, not a SaaS dashboard. Local-first by default; no telemetry, no remote authority.
- **Honest self-audit.** Status documents track simulator vs live, phase deliverables vs aspirations, dangerous capabilities, and durability gaps explicitly. The project knows what it is.

---

## 35. One-Page Investor / Partner Safe Summary

**Problem.** Current AI tooling is either a chat box, a feature embedded in an editor, or a wrapper around a remote API. None of these treats the AI itself as a *system* — something with state, lifecycle, audit, governance, and durable consequences. As AI moves from "ask a question" to "take actions on my behalf," the absence of that systems layer becomes the bottleneck.

**Solution.** FORGE is a local-first AI workspace built as a real operating layer. Models propose; deterministic validators check; a kernel-style commit boundary records what actually happened. The operator gets a desktop shell that can be inspected, approved, and rolled back at every meaningful step.

**Why now.** Local model inference is becoming credible. Approval and audit patterns from traditional software engineering have not yet been brought to AI runtimes. Operators with serious work to do are not well served by chatbots or generic agents.

**What FORGE does.** Provides a governed tool execution layer, a semantic state store with append-only journaling, a model runtime that treats inference as a bounded driver, an approval/audit/lineage system, and a desktop shell that can run as a NixOS session. All on the operator's hardware.

**Technical advantage (safe abstraction).**
- A transactional semantic commit gate that makes AI state changes auditable.
- Build-time enforcement of architectural safety properties.
- A simulator/live separation that lets new cognitive subsystems be developed and proven before they touch real state.
- A packaging substrate (Nix) that produces reproducible, session-locked operator environments.

**Current status.** Late alpha. ~100k lines of Go core, ~36k lines of TypeScript desktop, ~20-second test suite passing green, ~190 HTTP routes, four live deterministic validation surfaces wired into the kernel commit boundary, and a working VM session that boots the shell as the operator's desktop. Built by one operator with consistent LLM-assisted development over ~4-6 weeks.

**Next milestones.** Split the largest source files, raise coverage on the canonical-truth store, route the intent classifier live, migrate one more cognitive subsystem from simulator to live authority, and ship a cross-platform operator bring-up.

**Why it matters.** The next phase of AI tools is not "smarter models." It is "trustworthy substrates" — systems where AI behavior can be exercised, observed, and corrected without losing work or trust. FORGE is being built as one of those substrates, locally, with the safety properties baked into the build rather than the marketing copy.

---

## 36. Final Verdict

**FORGE is currently** a substantially complete late-alpha local AI runtime with a working desktop shell, a real kernel commit gate, build-time-enforced safety properties, and a proven simulator-to-live migration pattern. It boots as a NixOS session and survives review under load.

**The strongest part of the project is** the build-time-enforced doctrine. Forbidden-imports tests, Nix safe-mode literals, the single semantic commit chokepoint, and the shared-pure-package migration pattern make the architecture real, not aspirational. The status documents in `docs/status/` are unusually honest. The project knows what it is.

**The weakest part of the project is** the combination of monolithic source files (`gateway/service.go` at 4,709 lines, `chat_assistant_gateway.go` at 2,497 lines, `sqlite_store.go` at 2,244 lines) and thin test coverage on the canonical-truth memory package (6.5%). The gateway is the busiest surface; the memory package holds what the kernel commits. Both deserve better.

**The highest-leverage next move is** splitting `gateway/service.go` along capability-category lines. It makes every subsequent refactor cheaper, every subsequent test easier to target, every subsequent review more productive. A two-day refactor with the existing test suite as a safety net. Pair it with a coverage triage on `memory/`.

**The project should not proceed to** broader autonomy mode (charter "improve yourself" / unattended runs / multi-hour autonomous sessions) until: (a) the chat assistant gateway has been audited for prompt-injection → effector paths, (b) memory coverage is at 25%, (c) structured logging with correlation IDs is in place so unattended sessions are reconstructable, and (d) the operator has a `/health/detailed` surface that can show degradation in flight.

**Overall verdict.** This is a serious project. The architecture is real, the discipline is sustainable, the simulator-to-live pattern works, and the desktop shell boots. The next phase is *depth*, not breadth — split the big files, cover the memory package, audit the chat seam, route the intent classifier live, and migrate one more cognitive subsystem out of the simulator. Do those things and FORGE crosses from "ambitious alpha" to "early-production local AI substrate."

The thing is too good to risk on shortcuts. Don't take any.
