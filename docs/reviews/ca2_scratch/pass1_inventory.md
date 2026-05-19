# PHASE CA2 PASS 1: REPOSITORY INVENTORY

**Status**: AUDIT_ONLY / NO_RUNTIME_BEHAVIOR_CHANGE / NO_AUTHORITY_EXPANSION / NO_HOST_MUTATION / NO_FORGE_K_AUTHORITY_MIGRATION

**Inventory Date**: 2026-05-19
**Repo Root**: `/home/rshort/WTF/ProjectForge`
**Total Size**: ~17 GB (dominated by node_modules 132M, forge-operator-vm.qcow2 9.9 GB, apps/desktop build artifacts)

---

## TOP-LEVEL PROJECT FILES

| File/Directory | Size | Classification | Evidence |
|---|---|---|---|
| `README.md` | 3.3K | ACTIVE | Entry point; references FORGE, phases, integration setup |
| `CLAUDE.md` | 917 B | ACTIVE | Claude Code integration config; `.claude/settings.json` mirrors |
| `CODEX.md` | 16.9 KB | ACTIVE | Comprehensive codebase reference and architecture overview |
| `AGENTS.md` | 19.5 KB | ACTIVE | Authority boundaries, phase doctrine, live/simulator status; last update 2026-05-18 |
| `FORGE_CONTEXT.md` | 3.3 KB | ACTIVE | Foundation project context; referenced by AGENTS.md |
| `LICENSE` | 1.1 KB | ACTIVE | MIT license |
| `package.json` | 2.9 KB | ACTIVE | Monorepo workspace config; scripts for desktop, core, docker, validation |
| `package-lock.json` | 154 KB | ACTIVE | npm dependency lock |
| `flake.nix` | 5.9 KB | ACTIVE | Nix flake for reproducible build; defines nixpkgs, overlays, packages |
| `flake.lock` | 1.5 KB | ACTIVE | Nix dependency lock |
| `docker-compose.yml` | 5.9 KB | ACTIVE | Desktop + core service orchestration; defines web/desktop/core services |
| `docker-compose.igpu.yml` | 464 B | ACTIVE | Integrated GPU variant config |
| `.env.docker.example` | 2.6 KB | ACTIVE | Environment template for Docker run |
| `.dockerignore` | 198 B | ACTIVE | Docker build exclusions |
| `.gitignore` | 349 B | ACTIVE | Git ignore patterns |
| `.gitattributes` | 54 B | ACTIVE | Git LFS/attribute directives |
| `HighPriorityFixes.txt` | 6.4 KB | ACTIVE | Operational status/issue list (2026-05-19) |
| `.vm-build-core.log` | 224 B | LEGACY-RETAINED | VM build log artifact |
| `.vm-nix-store/` | — | LEGACY-RETAINED | NixOS VM store snapshot |
| `.vm-nix-tmp/` | — | LEGACY-RETAINED | NixOS VM temp directory |
| `result` | symlink | GENERATED | → `/nix/store/a81vfpivcvpgixniw5nzjp15cjs84ama-nixos-vm` (May 18 12:08) |
| `result-1` | symlink | GENERATED | → `/nix/store/wlxcprkjqz20h6smxx7zw0ik964jx25y-nixos-vm` (May 18 09:57) |
| `Full-Code-Review.md` | 16.6 KB | DOCS-ONLY | Full codebase review; likely Phase CA1 output (2026-05-11) |

---

## APPS DIRECTORY

### `apps/desktop/` (7.4 GB total)

**Classification**: ACTIVE

**Structure**:
- **Frontend (TypeScript/React)**: `src/` directory with Vite + Vitest + TSLint tooling
  - `src/pages/`: Dashboard, Chat, Memory, Models, Inspectors, Settings pages (render tests + coverage)
  - `src/components/`: Shared UI components (modal, sidebar, workspace, memory, settings primitives)
  - `src/layout/`: Multi-monitor workspace layout system with workspace/workspace-layout stores
  - `src/lib/`: Format utilities, desktop host labels, API client, window manager, profile rendering, render tests
  - `src/stores/`: Svelte stores for desktop shell, workspace, workspace layout state
  - `src/styles/`: Tailwind CSS configuration
  - Config: `vite.config.ts`, `vitest.config.ts`, `tsconfig.json`, `package.json` (@forge/desktop)

- **Tauri Backend (Rust)**: `src-tauri/`
  - `Cargo.toml`: Rust dependencies; v1 structure
  - `src/`: Tauri command handlers and desktop platform integration
  - `target/`: Build artifacts (Rust compiled binaries, incremental)
  - `tauri.conf.json`: Desktop app manifest
  - `capabilities/default.json`: Tauri permission capabilities
  - `build.rs`: Build script

**File Counts**: 201 TS/TSX files in frontend; 722 Go files in core (see below)

**Build Artifacts**: node_modules (132M from monorepo), .next, dist, Tauri target builds

---

## SERVICES DIRECTORY

### `services/core/` (7.3 MB source, excludes node_modules/build artifacts)

**Classification**: ACTIVE

**Primary Language**: Go (722 `.go` files across ~50 internal packages)

**Structure**:

- **Root Level**:
  - `main.go`: Entry point; daemon/API server initialization
  - `main_test.go`: Main tests
  - `go.mod`, `go.sum`: Go dependency management

- **`internal/` Organization** (48 packages):

  **Data & State Management**:
  - `config/`: Configuration loading and validation
  - `store/`: Data persistence layer (SQLite default; backend abstraction for Redis, Qdrant future)
  - `events/`: Append-only event streams for execution truth
  - `artifacts/`: Artifact retrieval and lifecycle management
  - `backup/`: Backup and recovery operations

  **Core FORGE Systems**:
  - `projectcontext/`: Project metadata and normalization
  - `packets/`: Task packet contracts (scope, risk, context IDs)
  - `approvals/`: Approval gates with request/decision record separation
  - `jobs/`: Job orchestration pipeline; execution projection and lifecycle
  - `policy/`: Policy validation and routing
  - `automation/`: Automation rules and self-initiated action boundaries

  **Memory & Knowledge**:
  - `memory/`: Structured canonical memory (separate from raw chat)
  - `retrieval/`: Semantic retrieval system; multi-backend abstraction
  - `embeddings/`: Embedding generation; TEI support, model abstraction
  - `dossiers/`: Dossier creation and management
  - `lineage/`: Provenance tracking on semantic objects
  - `insights/`: Derived insights and recommendation generation

  **Model Runtime**:
  - `modelruntime/`: Governed model execution; gated calls, streaming, managed delete-file
  - `gpu/`: GPU acceleration policies
  - `hostbridge/`: Host-level integration (VM/kernel communication)

  **FORGE-K Cognitive Microkernel** (Simulator-Only):
  - `forgek/`: Main FORGE-K subsystem (does NOT run as live daemon authority yet)
    - `court/`: Evidence admission and validation
    - `neurons/`: Rule-based and neural-driven proposal neurons
    - `palace/`: Deterministic rule neurons and policy enforcement
    - `semantic/`: Semantic operation execution (proposal-only)
    - `snapshots/`: Snapshot preservation for restoration and inspection
    - `contextcompiler/`: Context compilation with scoring and deterministic ranking
    - `kv/`: KV identity validation system (manifests, gates, acceleration metadata)
    - `runtime/`: Runtime driver boundary with deterministic mock drivers
    - `lymphatic/`: Maintenance proposals and cleanup suggestions (output-only)
    - `consensus/`: Consensus protocol for distributed decision-making
    - `integrationready/`: Integration readiness and validation
    - `shadowharness/`: Shadow mode harness for disabled-by-default validation reports
  - `forgekshadow/`: FORGE-K shadow reporting (disabled-by-default, observational only)

  **Shared Validation Packages** (Live Validation Seams):
  - `kvidentity/`: KV identity gate validation (shared live Control Lane `VALIDATE_KV_IDENTITY`)
  - `refvalidation/`: Ref-shape validation (shared live Control Lane `VALIDATE_REF_SHAPE`)
  - `semanticvalidation/`: Semantic operation validation (live Control Lane `COMPARE_REF_SHAPE`, `VALIDATE_SEMANTIC_OPERATION`)
  - `contextattribution/`: Context attribution validation (live Control Lane `VALIDATE_CONTEXT_ATTRIBUTION`)
  - `admissionvalidation/`: Evidence admission validation

  **Control Plane & Governance**:
  - `lanes/`: Lane abstraction for permission/capability routing
  - `gateway/`: Central gateway for API request routing and validation
  - `permissions/`: Permission model and enforcement
  - `audit/`: Audit logging and provenance capture
  - `reviews/`: Review and signoff workflows
  - `reconciliation/`: State reconciliation and consistency checks

  **API & Platform**:
  - `api/`: HTTP API handler definitions and routing
  - `aios/`: AI-OS integration and control surfaces
  - `ingest/`: Project context ingestion from external sources
  - `imports/`: Import lifecycle management
  - `watch/`: File system watch and change detection

  **Specialized Subsystems**:
  - `chat/`: Chat interface and conversation management
  - `canvas/`: Canvas/workbench collaborative surface
  - `dashboard/`: Dashboard aggregation and metrics
  - `search/`: Search and indexing
  - `adapters/`: Bounded worker adapters for external integrations
  - `strategies/`: Strategy selection and execution
  - `failurepatterns/`: Failure mode recognition and recovery
  - `packetopt/`: Packet optimization
  - `forgeh/`: Historical tracing and forensics
  - `ephemeral/`: Ephemeral/temporary resource management
  - `storagebackend/`: Storage abstraction layer
  - `vectorstore/`: Vector database integration
  - `authoritymatrix/`: Authority mapping and control
  - `consensusgate/`: Consensus-based decision gate
  - `release/`: Release management workflows

---

## PACKAGES DIRECTORY

### `packages/` (Shared Monorepo Libraries)

**Classification**: ACTIVE

- **`packages/shared/`**: TypeScript shared utilities and types (referenced by desktop)
- **`packages/ui/`**: UI component library and design system

---

## CRATES DIRECTORY

### `crates/forgek-validate/` (Rust)

**Classification**: ACTIVE

- **Purpose**: FORGE-K snapshot manifest validation
- **Build System**: Cargo
- **Test Fixtures**: `fixtures/forgek/` (valid, invalid, golden manifests; see Fixtures below)
- **Validation Commands**: Exposed through npm scripts `test:rust:forgek`, `validate:forgek-fixtures`, `test:forgek:parity`

---

## NIX DIRECTORY

**Classification**: ACTIVE

**Structure**:

- **`flake.nix`**: Root flake definition (overlays, packages, checks, shells, NixOS modules)
- **`checks/`**: NixOS/Nix build checks and validations
- **`lib/`**: Utility functions for Nix expressions
- **`modules/`**: NixOS module definitions
- **`nixos/`**: NixOS system configurations for test/operator VMs
- **`overlays/`**: Package overrides and extensions
- **`packages/`** (6 files):
  - `forge-core.nix`: Go core service build
  - `forge-desktop-shell.nix`: Tauri-built desktop shell (real package, May 19 08:34)
  - `forge-shell-session.nix`: Safe-mode shell session wrapper (May 18 08:11)
  - `forge-operator-session.nix`: Operator session and tooling (May 19 08:34)
  - `forge-operator-toolbelt.nix`: Tooling utilities (May 18 11:59)
  - `forge-wayland-session.nix`: Wayland session integration (May 11 16:09)
- **`profiles/`**: NixOS profile selections
- **`shells/`**: Dev shell environments (mkShell definitions)
- **`tool-capsules/`**: Containerized tool environments

**Status**: Phase G5 graphics test profile scaffolding in place; no live authority mutations

---

## DOCS DIRECTORY (Comprehensive Documentation Tree)

**Total**: 40 root-level `.md` files + 67 subdirectories; well-organized reference architecture

**Key Structure**:

- **Architecture & Design**:
  - `ARCHITECTURE.md`: System architecture overview
  - `UI_ARCHITECTURE.md`: Frontend/Tauri architecture
  - `MEMORY_ARCHITECTURE.md`: Memory subsystem design
  - `DATA_INTEGRITY_AND_WIRING.md`: Data flow validation
  - ADR folder: Architecture Decision Records

- **Phase Documentation**:
  - `PHASE1_SCOPE.md` through `PHASE5_SCOPE.md`: Phase scope definitions
  - `PHASE5_OPERATIONS.md`: Operations phase runbook

- **Feature Documentation**:
  - `DESKTOP_SHELL.md`: Desktop shell interface
  - `WORKSPACE_GUIDE.md`, `WORKSPACE_LAYOUTS.md`, `MULTI_MONITOR_LAYOUTS.md`: Workspace systems
  - `NAVIGATION_AND_WORKSPACES.md`: Navigation and workspace interaction
  - `CHAT.md`: Chat interface design
  - `MEMORY_ARCHITECTURE.md`: Memory system
  - `RETRIEVAL_AND_EMBEDDINGS.md`, `RETRIEVAL_PIPELINE.md`: Retrieval systems
  - `JOBS_AND_APPROVALS.md`, `POLICY_AND_APPROVALS.md`: Job and approval workflows
  - `TASK_PACKETS.md`: Task packet format and contracts
  - `CAPABILITY_BROKERS.md`: Capability model
  - `AUDIT_AND_TRACE.md`: Audit and tracing
  - `REVIEWS_AND_RECONCILIATION.md`: Review workflows

- **Operational Documentation**:
  - `USER_MANUAL.md`: End-user guide
  - `PACKAGING.md`: Packaging and deployment
  - `REMOTE_ACCESS.md`: Remote access setup
  - `USEFULNESS_AND_REPAIR.md`: Troubleshooting and repair

- **Subdirectories**:
  - `adr/`: Architecture Decision Records (numbered)
  - `academy/`: Educational/tutorial content
  - `api/`: API endpoint documentation
  - `architecture/`: Architecture detail documents
  - `data_model/`: Data model specifications
  - `diagrams/`: Architecture diagrams (visuals)
  - `evidence/`: 12 MB of evidence artifacts (memory, vm_boot validation)
  - `journal/`: Status journal entries
  - `memory/`: Memory system specifications
  - `operations/`: Operational runbooks
  - `prompt-packs/`: Prompt pack definitions
  - `reports/`: Review and audit reports
  - `reviews/`: Code review and audit outputs; **ca2_scratch/** created for this audit
  - `roadmap/`: Project roadmap and timeline
  - `runbooks/`: Operational runbooks (CPU/RAM/GPU split, no GPU boot recovery, etc.)
  - `status/`: Current authority sources and implementation matrix
  - `superpowers/`: Operator capability documentation
  - `testing/`: Test documentation and strategies

- **Archive**:
  - `docs/archive/`: 520 KB; 42 files (legacy phase documentation, retained for reference)

---

## SCRIPTS DIRECTORY

**Classification**: ACTIVE

**Purpose**: Build, orchestration, and validation scripts

**File Count**: 34 scripts (sh, mjs, ps1 variants)

**Key Scripts**:

- **Orchestration**:
  - `forge-up.{mjs,sh,ps1}`: Start FORGE desktop + core
  - `forge-down.{mjs,sh,ps1}`: Stop FORGE services
  - `forge-core.{mjs,sh,ps1}`: Run core service

- **Docker Integration**:
  - `forge-docker-up.{mjs,sh,ps1}`: Docker Compose up
  - `forge-docker-down.{mjs,sh,ps1}`: Docker Compose down
  - `forge-docker-desktop.{mjs,sh,ps1}`: Docker desktop variant

- **Validation & Checks**:
  - `check-desktop-deps.{mjs,sh,ps1}`: Desktop dependency verification
  - `check-vsa-files.{mjs,sh,ps1}`: VSA (version source authority) file checks
  - `check-integration-env.mjs`: Integration environment readiness
  - `generate-api-routes.{mjs,test.mjs}`: API route generation and testing
  - `forgek-parity.mjs`: FORGE-K validator parity checks

- **Cleanup**:
  - `desktop-clean-tauri.{mjs,ps1}`: Tauri build artifact cleanup
  - `desktop-clean-port.{mjs,sh,ps1}`: Port cleanup for desktop dev

- **Testing**:
  - `forge-smoke.{mjs,sh,ps1}`: Smoke test runner

---

## FIXTURES DIRECTORY

**Classification**: TEST-ONLY / REFERENCED

**Structure**:

- **`fixtures/forgek/`**: Test data for FORGE-K snapshot validation
  - `valid/`: Valid snapshot manifests (context_block, context_bundle, kv_cache_manifest, runtime_driver_manifest, snapshot_manifest)
  - `invalid/`: Invalid test cases (missing fields, bad cache modes, secret presence, etc.)
  - `golden/`: Canonical reference manifests and hashes
  - `README.md`: Fixture documentation

**Referenced By**:
- `npm run validate:forgek-fixtures` (crates/forgek-validate)
- Integration and validation test suites

---

## .CLAUDE DIRECTORY

**Classification**: ACTIVE

**Contents**:
- `README.md`: Claude Code integration guide
- `settings.json`: Claude Code harness configuration (1.7 KB)
- `settings.local.json`: User-local overrides (501 B)
- `hooks/`: Custom hook definitions (lifecycle, on-start, on-change)
- `rules/`: Project rules and constraints
- `skills/`: Custom Claude Code skill definitions

---

## .FORGE DIRECTORY

**Classification**: ACTIVE / EVIDENCE

**Contents**:

- **Operational Logs**:
  - `logs/core.log`: Core service execution log
  - `logs/desktop.log`: Desktop app execution log
  - `nix-results/operator-stack-build.log`: NixOS build log
  - `nix-results/operator-checks.log`: Nix check output

- **VM Artifacts**:
  - `docker-api-token`: Docker authentication token artifact
  - `vm/`: 20 screenshot and metadata files (boot sequence, SSH setup, operator session capture)
  - `vm/forge-vm-ed25519*`: SSH keypair for VM access

**Size**: Logs + screenshots ~100 KB

---

## FORGE-HMK_Ultimate_Prompt_Pack

**Classification**: DOCS-ONLY / PROMPT-PACK

**Purpose**: Centralized prompt, context, and authority documentation for operators

**Structure** (40 files total):

- **Main Artifact**:
  - `FORGE-HMK_ULTIMATE_PROMPT_PACK_COMBINED.md`: 34 KB combined manifest (May 18 14:15)
  - `manifest.json`: Pack metadata (May 18 14:15)
  - `README.md`: Pack overview (May 18 13:45)

- **Subdirectories**:
  - `adr_prompts/`: Architecture Decision Record prompt templates
  - `context/`: Context injection prompts
  - `handoff/`: Handoff protocols and templates
  - `prompts/`: Prompt collection
  - `rules/`: Project rules (decision gates, authority boundaries)
  - `skills/`: Skill definitions
  - `validation/`: Validation checklists

---

## FORGE DIRECTORY

**Classification**: DOCS-ONLY / OBSIDIAN-VAULT

**Structure**:
- `.obsidian/`: Obsidian vault configuration (graph.json, workspace, appearance settings)
- `.trash/`: Trash folder (Welcome.md)

**Purpose**: Obsidian-backed knowledge vault (not integrated into live FORGE)

---

## .CURSOR DIRECTORY

**Classification**: LEGACY-RETAINED

- Cursor IDE configuration artifacts (ignored, not active in current workflows)

---

## .GITHUB DIRECTORY

**Classification**: ACTIVE

- GitHub Actions and workflow configurations (CI/CD pipelines)

---

## .GIT DIRECTORY

**Classification**: ACTIVE / METADATA

- Git repository metadata, refs, objects
- Latest commits (2026-05-19): "Harden desktop UI routes", "Harden CA2 authority defaults", "Add CA1 codebase integrity audit"

---

## .OBSIDIAN DIRECTORY

**Classification**: LEGACY-RETAINED

- Obsidian vault configuration (mirrors FORGE/.obsidian setup)

---

## NODE_MODULES

**Classification**: GENERATED

- **Size**: 132 MB
- **Content**: npm workspace dependencies (desktop frontend, scripts, build tooling)
- **Origin**: npm install from package-lock.json

---

## LARGE FILES & STORAGE ARTIFACTS

| File | Size | Classification | Evidence |
|---|---|---|---|
| `forge-operator-vm.qcow2` | 9.9 GB | GENERATED | NixOS VM image; used for operator environment testing |
| `package-lock.json` | 154 KB | ACTIVE | npm dependency lock |
| `docs/evidence/` | 12 MB | EVIDENCE-ONLY | VM boot validation, memory system evidence |
| `node_modules/` | 132 MB | GENERATED | npm dependencies (frontend, monorepo tooling) |
| `apps/desktop/src-tauri/target/` | ~500 MB (approx) | GENERATED | Rust build artifacts |
| Total Repository | ~17 GB | — | Dominated by VM image, build artifacts, and dependencies |

---

## SYMLINKS & ARTIFACTS

| Link | Target | Classification |
|---|---|---|
| `result` | `/nix/store/a81vfpivcvpgixniw5nzjp15cjs84ama-nixos-vm` | GENERATED (May 18 12:08) |
| `result-1` | `/nix/store/wlxcprkjqz20h6smxx7zw0ik964jx25y-nixos-vm` | GENERATED (May 18 09:57) |

**Note**: Per AGENTS.md, `.worktrees/` is NOT present; no task branches retained as worktrees currently.

---

## PROJECT GOVERNANCE & PHASE STATUS

**Authority Documents** (last updated 2026-05-18/19):

1. **AGENTS.md** (186 lines): Authority boundaries, live/simulator status, phase doctrine
   - FORGE-K is simulator-only; NOT live daemon authority
   - Narrow live validation seams: KV identity gates, ref-shape validation, semantic operation validation, context attribution validation (2026-05-08 through 2026-05-18)
   - Phase 13I (store cutover): docs-only readiness review; SQLite remains default
   - Phase 14D-E: Disabled-by-default shadow reporting (optional best-effort observer)
   - Phase 19: Context attribution validation (shared `contextattribution` package, live Control Lane `VALIDATE_CONTEXT_ATTRIBUTION`)

2. **CODEX.md** (596 lines): Comprehensive codebase reference

3. **CLAUDE.md** (22 lines): Claude Code integration config

4. **FORGE_CONTEXT.md** (3.3 KB): Foundation project context

5. **Full-Code-Review.md** (16.6 KB): Phase CA1 full codebase audit (2026-05-11)

**Current Implementation Status** (per AGENTS.md, 2026-05-14):

- **ACTIVE**: Model Runtime M3/M4 (gated, approval-required delete-file, disabled-by-default vLLM)
- **SIMULATOR-ONLY**: FORGE-K court, neurons, palace, KV system, runtime driver, lymphatic lane, consensus, integration validation
- **PARTIAL LIVE VALIDATION**: KV identity gates (Control Lane), ref-shape validation, semantic operation validation, context attribution validation

**No Authority Expansion in CA2**: FORGE-K simulator services remain unbundled from live daemon; shadow reporting disabled by default; no modelruntime changes; no route/API changes; no memory mutation; no evidence admission execution in simulator.

---

## NOTABLE PATTERNS & OBSERVATIONS

1. **Monorepo Structure**: npm workspaces (desktop, shared, ui) + Go services/core + Rust crates (forgek-validate) + Nix overlays
2. **Multi-Platform**: sh, mjs, ps1 script variants (Linux, macOS, Windows support)
3. **Deterministic Build**: Nix flakes + flake.lock for reproducibility
4. **Modular Services**: 48 internal Go packages; clear lane/gateway/approval/audit subsystems
5. **Simulator-First Architecture**: FORGE-K cognitive microkernel exists as isolated simulator; narrow validation seams only
6. **Evidence-Driven**: Audit logging, provenance tracking, snapshot preservation, review/reconciliation workflows
7. **Operational Maturity**: Runbooks, troubleshooting guides, multi-phase documentation; VM test harness
8. **GPU/CPU Abstraction**: Explicit kernel + accelerator split; safe-mode CPU-only recovery documented

---

## OPEN QUESTIONS FOR SYNTHESIS

1. **Nix/NixOS Build Status**: `result` and `result-1` symlinks point to successful VM builds (May 18); should these be cleaned up as build artifacts or retained for operator provenance?

2. **VM Image Lifecycle**: `forge-operator-vm.qcow2` (9.9 GB) is current; `.vm-nix-store` and `.vm-nix-tmp` may be stale. Are these retained for reproducibility or can they be cleaned in later phases?

3. **Obsidian Vault Integration**: Both `FORGE/` and `.obsidian/` directories suggest parallel vault setups. Is one deprecated or do they serve different purposes?

4. **Archive Depth**: `docs/archive/` contains 42 legacy phase files (520 KB). Are these retained for legal/operational provenance, or are they candidates for external archival in Phase CA3+?

5. **Evidence Retention**: `docs/evidence/vm_boot/` contains 12 MB of boot sequence screenshots. Is this retained for operator training/validation evidence, or should it migrate to a separate artifact store?

6. **Shadow Reporting**: FORGE-K shadow reports are disabled by default (requires `FORGE_K_SHADOW_MODE_ENABLED=true`). Are CI pipelines or operator runbooks verifying shadow report health/consistency?

7. **Live Authority Boundary Validation**: How is the simulator/live boundary audited in automated tests? Are there integration tests that explicitly verify no FORGE-K simulator services are wired into live daemon paths?

8. **ModelRuntime M3/M4 Supervision**: AGENTS.md cites "hardening/supervision: stronger backend/process control, deeper scheduling/backpressure, cancellation/usage accounting hardening" as remaining modelruntime work. Are there tracking docs (ADRs, issues, runbooks) for these hardening phases?

9. **Phase CA2 vs. CA3 Scope**: This inventory covers repos structure and file classification. Should Phase CA2 Pass 2 drill into authority boundaries within services/core and apps/desktop, or is that deferred to Phase CA3?

10. **Symlink Cleanup Policy**: Per AGENTS.md worktree policy, artifacts are retained for provenance. Do the `result` and `result-1` Nix symlinks follow the same retention policy, or should they be tracked separately in a build artifact manifest?

---

**End of Pass 1 Inventory**
