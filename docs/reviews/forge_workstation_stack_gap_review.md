# FORGE Workstation Stack Gap Review

Status: DESIGN_ONLY / READINESS_REVIEW / PARTIAL_PURE_LABELS / NO_LIVE_AUTHORITY_CHANGE

Date: 2026-05-12

## Scope

Phase M5 reviews the repository state for a future private FORGE Workstation target. It covers NixOS substrate composition, Tauri shell status, modelruntime backend profiles, FORGE-H resource governance, GPU/VRAM posture, safe-mode recovery, and read-only System cockpit planning.

This pass does not implement host mutation, service control, model load/unload behavior changes, public mutation APIs, semantic memory writes, storage cutover, or FORGE-K live authority.

## Current Implemented Substrate Pieces

| Area | Current state | Status label |
|---|---|---|
| Root workflows | `npm run build`, `npm run build:core`, `npm run build:desktop`, `npm test`, `npm run smoke`, and Nix commands are documented. | IMPLEMENTED |
| Nix flake | `flake.nix` exposes dev shells and package/app targets including core and graphical shell paths. | PARTIAL |
| NixOS modules | `nix/nixos/modules` contains opt-in module scaffolding for FORGE service/storage/host substrate paths. | DISABLED_BY_DEFAULT |
| `/forge` durable root | Architecture and module scaffolding reserve `/forge/data`, `/forge/state`, `/forge/logs`, `/forge/cache`, `/forge/backups`, `/forge/models`, and related paths. | PLANNED / OPT_IN |
| Wayland shell session | `forge-wayland-session` and VirtualBox test profile exist for manual/opt-in launch paths. | PARTIAL / TEST_ONLY |
| Tauri desktop shell | `apps/desktop` is the operator shell; G6 added a read-only System surface and `GET /forge/system/status`. | PARTIAL / READ_ONLY |
| HostBridge | `services/core/internal/hostbridge` provides read-only diagnostic snapshots. | IMPLEMENTED / READ_ONLY |
| FORGE-H | `services/core/internal/forgeh` provides advisory resource policy, proposals, and bounded internal execution records. | ADVISORY_ONLY / BOUNDED_INTERNAL |
| modelruntime | `services/core/internal/modelruntime` owns governed model registry, lifecycle, scheduler, backend health, and `/forge/models*` plus gated `/v1/*`. | PARTIAL_LIVE |
| Storage backends | SQLite remains live default; Postgres/Qdrant/Redis scaffolding remains disabled or non-authoritative. | DEFAULT_SQLITE / DISABLED_BY_DEFAULT |
| FORGE-K | Simulator packages and narrow live Control Lane validation seams exist, but simulator services are not live daemon authority. | SIMULATOR_ONLY / PARTIAL_LIVE_VALIDATION |

## Current Tauri/Nix Shell State

- `docs/architecture/forge_graphical_shell.md` defines the shell as the visible operator surface above NixOS, not a host control plane.
- `docs/architecture/shell_system_surfaces.md` documents the current G6 System surface.
- `nix/packages/forge-shell-session.nix`, `nix/packages/forge-desktop-shell.nix`, and Wayland session scaffolding keep launch paths explicit.
- The shell path remains opt-in, does not enable autologin, does not remove fallback desktops or TTY access, and does not run host commands.
- The next cockpit work should extend read-only display surfaces, not add buttons that apply host changes.

## Current Modelruntime Backend Support

| Backend/profile surface | Current implementation | Gap |
|---|---|---|
| llama.cpp | Endpoint adapter and optional explicitly allowed binary path are represented in config and backend code. | Process supervision remains limited. |
| OpenAI-compatible remote | OpenAI-compatible endpoint backend exists. | Provider policy and richer failure classification remain future work. |
| vLLM-compatible | M4 adds disabled-by-default endpoint profile via `FORGE_VLLM_BASE_URL` / `FORGE_VLLM_API_KEY` with legacy aliases. | No managed vLLM service, CUDA provisioning, batching policy, or process control. |
| Ollama dev compatibility | Existing Ollama adapter and local dev autodetect are present. | It remains a compatibility/development path, not modelruntime authority. |
| TEI embeddings | TEI is available as an embedding provider. | It is not a general modelruntime generation backend. |
| CPU safe | CPU-only safe-mode posture is documented/configured. | Needs clearer workstation profile mapping and operator recovery docs. |

## Current GPU and Safe-Mode Policy

- `docs/architecture/cpu_ram_kernel_gpu_accelerator_split.md` states that core truth authority is CPU/RAM-only.
- Optional NVIDIA DCGM and Intel Level Zero telemetry paths are diagnostic and degrade safely when unavailable.
- `FORGE_SAFE_MODE_FORCE_CPU_ONLY=true` is the current safe-mode switch for CPU-only posture.
- GPU-aware classes are modelruntime scheduling/admission metadata, not authority surfaces.
- There is no implemented VRAM lease registry, CUDA kernel launch approval flow, CUDA VMM/IPC mode, or runtime KV residency control.

## Current FORGE-H Advisory Capabilities

FORGE-H can:

- consume HostBridge snapshots;
- classify RAM, swap, disk, VRAM, and thermal pressure;
- recommend lane/model/background-work posture;
- create advisory resource action proposals;
- record bounded internal execution outcomes for allowed advisory actions.

FORGE-H cannot:

- run commands;
- call `systemctl` or `nixos-rebuild`;
- mutate host configuration;
- load/unload models;
- write semantic memory;
- bypass gateway, lanes, approvals, audit, Control Lane validation, modelruntime, or FORGE-K boundaries.

## Current FORGE-K Boundary

ADR 0005 remains authoritative. FORGE-K is the target cognitive microkernel architecture. The live daemon still uses existing AI-OS, gateway, permissions, lane, audit, modelruntime, retrieval, embeddings, memory, and API authority paths unless an explicit integration phase changes one narrow seam with tests.

M5 does not import FORGE-K simulator services into live authority and does not route live state mutation through FORGE-K.

## ADR Numbering Note

`PhaseM5.txt` requested `docs/adr/0013-forge-workstation-substrate-and-nix-mutation-proposals.md`, but ADR 0013 already exists as `docs/adr/0013-forge-g6-operator-desktop.md`. M5 therefore records the workstation/Nix mutation decision as `docs/adr/0014-forge-workstation-substrate-and-nix-mutation-proposals.md` to preserve ADR history and avoid duplicate numbering.

## Missing Pieces For Desired Workstation Substrate

| Missing piece | Why it matters | Likely future files |
|---|---|---|
| Durable `NixMutationProposal` records | Needed before FORGE can safely propose, review, build, test, and apply Nix changes. | `services/core/internal/nixproposals/*`, migrations, API read-only routes, approval integration |
| Sandbox Nix build runner | Needed to prove a proposal builds before VM smoke testing or apply approval. | future governed host adapter package; not Tauri |
| VM smoke-test adapter | Needed to test workstation changes without mutating the live host. | `nix/nixos/profiles/*`, test harness scripts, proposal evidence store |
| Rollback evidence model | Needed before host apply can be approved. | proposal store, audit/journal refs, runbooks |
| Workstation profile module | Needed for an opt-in full FORGE Workstation composition. | `nix/nixos/profiles/forge-workstation.nix`, module docs |
| Read-only cockpit panels | Needed for operator visibility into Nix generation, proposals, backend profile, GPU posture, storage readiness, and test/build state. | `apps/desktop/src/pages/System*`, shared contracts, `services/core/internal/api/system_status.go` |
| Backend profile contracts | Needed to show runtime posture without forcing vLLM or GPU as the only path. | `services/core/internal/modelruntime`, config docs, shell status DTOs |
| VRAM/CUDA governance objects | Needed before any CUDA acceleration can be governed safely. | future `services/core/internal/forgehgpu` or focused FORGE-H subpackage |
| Safe-mode NixOS specializations | Needed for reliable fallback and recovery. | `nix/nixos/profiles/*`, runbooks |

## Optional Scaffolding Decision

Phase M5 adds only two pure, non-wired label contracts with tests:

- `services/core/internal/modelruntime/profiles.go` for backend profile labels;
- `services/core/internal/forgeh/cuda_lane.go` for future GPU work-class labels.

These labels do not alter backend selection, lifecycle, scheduler behavior, route behavior, shell behavior, host execution, or modelruntime execution. They are contract vocabulary for future read-only/operator surfaces and policy design.

M5 intentionally skips Nix mutation proposal status constants because there is no existing owning package or durable proposal store. The next code phase should choose one narrow target with tests:

- durable Nix proposal data models and store tests;
- read-only System cockpit status DTO expansion; or
- backend profile labels consumed by existing health/status code.

## Non-Negotiable Boundaries

- No host mutation is introduced.
- No direct `systemctl`, `nixos-rebuild`, package-manager, reboot, shutdown, kernel/module, or destructive cleanup path is introduced.
- No shell mutation buttons are introduced.
- No model load/unload behavior changes are introduced.
- No semantic memory writes are introduced.
- No storage backend default changes are introduced.
- No Qdrant/Redis canonical authority is introduced.
- No FORGE-K live authority migration is introduced.

## Recommendation

Close M5 as a design/readiness hardening phase. The recommended next implementation phase is a durable, non-executing Nix mutation proposal record with store tests and read-only listing surfaces. Host apply should remain out of scope until proposal approval, build proof, VM smoke proof, rollback proof, audit refs, and a governed adapter design exist.

## Validation Evidence

Commands run on 2026-05-12:

| Command | Result | Notes |
|---|---|---|
| `npm install` | PASS | Installed worktree dependencies; npm reported 6 moderate audit findings. |
| `cd services/core && go test ./internal/modelruntime -run TestBackendProfile -count=1` | RED then PASS | First run failed on missing M5 profile labels; passed after pure label implementation. |
| `cd services/core && go test ./internal/forgeh -run TestGpuWork -count=1` | RED then PASS | First run failed on missing M5 GPU work-class labels; passed after pure label implementation. |
| `cd services/core && go test ./internal/modelruntime ./internal/forgeh -count=1` | PASS | Focused packages touched by M5. |
| `cd services/core && go test ./internal/hostbridge -count=1` | PASS | Read-only host diagnostic dependency check. |
| `npm test` | FAIL | Full core sweep failed outside M5 in `internal/backup` Windows symlink privilege, `internal/forgek` golden canonical JSON drift, and `internal/ingest` filesystem-root scope behavior. |
| `npm run build:core` | PASS | Go core build passes. |
| `npm run build:desktop` | PASS | Vite desktop build passes with existing large chunk warning. |
| `npm run build` | PASS | Desktop and core aggregate build passes with existing large chunk warning. |
| `git diff --check` | PASS | No whitespace errors; Git reported CRLF conversion warnings on Windows. |

Known blockers:

- Full `npm test` is not green in this Windows worktree because of existing environment/golden issues unrelated to M5.
- Nix mutation proposal execution remains intentionally unimplemented.
- System cockpit expansion remains design-only until a read-only DTO/API/UI phase is approved.
