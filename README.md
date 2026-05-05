# FORGE

FORGE is a local-first AI workspace for inspectable, approval-gated engineering work.

## FORGE-K Architecture Baseline

FORGE-K is the deterministic cognitive microkernel architecture inside FORGE. It exists to keep canonical truth under Kernel authority while model runtimes, tools, adapters, and neurons operate as bounded proposal or driver surfaces.

Current checkpoint: FORGE-K Phase 1-11G are implemented and tested in the simulator under `services/core/internal/forgek` or adjacent research/tooling paths. Phase 6 adds Context-Shape Snapshots. Phase 7 adds the Context Compiler. Phase 8 adds the Deterministic KV metadata system. Phase 9 adds the simulator Runtime Driver Boundary with a deterministic mock driver only. Phase 10 adds the simulator Lymphatic Lane with Maintenance Reports and Cleanup Proposals only. Phase 11A is a research/planning checkpoint for a possible Rust deterministic kernel-core boundary. Phase 11B adds the standalone Rust validation crate `crates/forgek-validate` plus shared fixtures under `fixtures/forgek`. Phase 11C aligns Go and Rust against that shared fixture corpus with Go parity tests, Rust validation tests, golden canonical JSON, golden hashes, and the optional `npm run test:forgek:parity` helper. Phase 11D integrates the Rust validator, fixture validation, and Go/Rust parity checks into CI/tooling only. Phase 11E adds the simulator-only Consensus Mesh for governed claim acceptance before response/action composition. Phase 11F adds integration-readiness contracts, live path mappings, shadow-mode policy, and read-only adapter contracts, including RAG/retrieval mirror boundaries. Phase 11G defines the simulator-only shadow harness design, report contracts, and no-effect validator. Phase 12A adds live integration design documentation for a future disabled-by-default read-only shadow harness. Phase 12A is documentation/design only; no live FORGE-K shadow mode, live mutation, route change, public API change, modelruntime call, retrieval/embedding call, or memory write exists yet.

ADR 0005 defines the live-authority boundary: FORGE-K is the target architecture, but the live daemon still uses the existing AI-OS, gateway, permissions, lane, audit, model runtime, retrieval, embeddings, memory, and API authority paths. Phases 6, 7, 8, 10, 11B, 11C, 11D, 11E, 11F, and 11G are implemented as `SIMULATOR_ONLY`, `RESEARCH_ONLY`, `TOOLING_ONLY`, `INTEGRATION_PREP_ONLY`, or `SHADOW_DESIGN_ONLY` where applicable; Phase 9 is implemented as `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`; Phase 12A is `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. These phases are not wired into the live daemon and do not modify live AI-OS snapshot/restore, `COMPILE_CONTEXT`, dream/autonomy cleanup behavior, model runtime, retrieval, embeddings, memory, gateway, route, or public API behavior. Consensus accepted does not mean canonical truth; Kernel commit remains the only canonical truth path. Phase 11F defines read-only adapter contracts, including RAG/retrieval mirror contracts, but does not implement live RAG. Phase 11G designs the future shadow harness but does not implement live observation. Phase 12A designs the first possible live shadow implementation and explicitly requires separate approval before Phase 12B code. Root `npm test` remains the Go core test path and does not depend on Rust validation.

Development principles:

- models are drivers, not authority
- neural neurons propose; rule neurons validate
- Memory Palace retrieves candidate references; Courthouse controls admissibility
- Semantic Algebra transforms admitted meaning without bypassing Kernel authority
- Courthouse admits evidence; Kernel commits through semantic syscalls
- snapshots preserve context shape, not truth
- deterministic KV cache is acceleration, not memory
- provenance, journal evidence, and approval boundaries are required for meaningful state transitions

FORGE-K architecture links:

- `docs/architecture/forge_k_overview.md`
- `docs/architecture/core_doctrine.md`
- `docs/architecture/neuron_fabric.md`
- `docs/architecture/lane_model.md`
- `docs/architecture/memory_palace_and_courthouse.md`
- `docs/architecture/semantic_algebra.md`
- `docs/architecture/snapshots.md`
- `docs/architecture/context_compiler_and_kv_cache.md`
- `docs/architecture/runtime_driver_boundary.md`
- `docs/architecture/lymphatic_lane.md`
- `docs/architecture/consensus_mesh.md`
- `docs/architecture/forge_k_integration_readiness.md`
- `docs/architecture/forge_k_adapter_contracts.md`
- `docs/architecture/shadow_mode.md`
- `docs/architecture/shadow_mode_harness.md`
- `docs/architecture/forge_k_live_integration_design.md`
- `docs/architecture/phase_12b_shadow_harness_spec.md`
- `docs/architecture/phase_12b_adapter_interfaces.md`
- `docs/architecture/rust_kernel_core_plan.md`
- `docs/architecture/kernel_simulator.md`
- `docs/architecture/forge_1_cpu_concept.md`
- `docs/roadmap/forge_k_build_phases.md`
- `docs/glossary.md`
- `docs/testing/definition_of_done.md`
- `docs/testing/rust_validation.md`

FORGE-K ADRs and diagrams:

- `docs/adr/0001-forge-k-is-a-cognitive-microkernel.md`
- `docs/adr/0002-models-are-drivers-not-authority.md`
- `docs/adr/0003-snapshots-are-shape-not-truth.md`
- `docs/adr/0004-kv-cache-is-acceleration-not-memory.md`
- `docs/adr/0005-forge-k-simulator-vs-live-authority.md`
- `docs/adr/0006-rust-kernel-core-boundary.md`
- `docs/diagrams/forge_k_master_flow.mmd`
- `docs/diagrams/forge_k_layer_model.mmd`

## CPU/RAM Kernel + GPU Accelerator Split

FORGE core authority is CPU/RAM-only by design.

- kernel/control/journal/state truth authority remains in `forge-core`
- GPU-aware inference is isolated to governed modelruntime paths
- safe mode supports CPU-only degraded operation without breaking core authority

References:

- `docs/architecture/cpu_ram_kernel_gpu_accelerator_split.md`
- `docs/runbooks/no_gpu_boot_and_recovery.md`

The desktop client now supports a **monitor-aware desktop shell**:

- multiple real Tauri shell windows
- real display detection through the desktop runtime
- named workspace layout presets
- per-window surface assignments
- layout activation, restore, and fallback when monitor availability changes
- chat attachments with thread-linked artifacts and a right-side code/files inspector
- observation-based memory architecture with retrieval run inspectability, packet alignment notes, and usefulness/repair controls
- scheduled + manual memory repair runs with persisted before/after traces
- governed full tool layer with typed actions, lane/profile policy checks, approvals, and audit traceability
- deterministic context restore scoring with header-first restore packages
- CPU-only dry-run maintenance replay and consolidation reports
- optional NVIDIA DCGM / Intel Level Zero GPU telemetry and Hugging Face TEI embedding provider diagnostics

This is a real desktop feature. FORGE does not simulate monitors or invent off-screen window state.

## Desktop Model

FORGE now works in four layers:

- `Desktop shell`: the overall workstation environment
- `Workspace windows`: real Tauri windows in the same shared session
- `Surfaces`: Chat, Workbench, Canvas, Dossiers, Jobs, Reviews, Approvals, Logs, Settings, Layouts
- `Layouts`: named monitor-aware arrangements of those windows and surfaces

## Multi-Monitor Features

Implemented:

- real multi-window shell support through Tauri webview windows
- monitor detection from Tauri monitor APIs
- named layout presets: `Build`, `Research`, `Ops`, `Deep Work`
- layout editor for monitor assignment, role assignment, and per-window surfaces
- active layout switcher in the shell top bar
- layout restore from the main shell window on relaunch
- monitor fallback when a saved display is missing

Implemented with explicit limits:

- no floating tiling/window-manager simulation
- no fake monitor IDs; monitor identity is derived from real monitor properties exposed by Tauri
- browser-only Vite runtime can inspect saved layouts but does not simulate monitor or extra-window support

## Requirements

- Go `1.22+`
- Rust toolchain for Tauri desktop windows
- Node.js + npm

## Run (development)

1. Start the core service:

```bash
npm run core
```

The development core launcher enables the governed modelruntime surface by default so the desktop Models and Chat model selectors can connect. It does not configure a remote/cloud provider or default model automatically; provider endpoints still require explicit `FORGE_MODEL_*` configuration.

2. Start the desktop shell:

```bash
npm run desktop
```

If the desktop window does not open, first check Tauri startup logs. The most common blockers are:

- **port conflict on `1420`** (existing Vite/Tauri dev server)
- **missing Linux webkit libs** (linker errors for `webkit2gtk-4.1` / `javascriptcoregtk-4.1`)

`npm run desktop` now performs a preflight check and clears stale FORGE-local Vite listeners on `1420` automatically.
If another non-FORGE process owns `1420`, startup will stop and print that process so you can resolve it.

Typical fixes:

```bash
# if something is already serving 1420, stop it first
sudo lsof -ti :1420 | xargs -r kill -9

# Linux dependencies (Debian/Ubuntu)
sudo apt-get update && sudo apt-get install -y libwebkit2gtk-4.1-dev libjavascriptcoregtk-4.1-dev libgtk-3-dev

# Linux dependencies (openSUSE)
sudo zypper install -y webkitgtk3-devel gtk3-devel

# if package names differ on your snapshot, locate providers
zypper search --provides 'pkgconfig(webkit2gtk-4.1)'
zypper search --provides 'pkgconfig(javascriptcoregtk-4.1)'
```

Build commands:

```bash
npm run build
npm run build:desktop
npm run build:core
```

Default endpoints:

- Core API: `http://127.0.0.1:18492`
- Desktop dev server: `http://localhost:1420`
- Default shell route: `#/chat`

## Daily Flow

1. Start the main FORGE shell window.
2. Choose or activate a layout from the top-bar switcher or `#/layouts`.
3. Let FORGE place or restore windows on the available monitors.
4. Work in Chat, Workbench, Canvas, Dossiers, Jobs, Reviews, Approvals, or Logs across screens.
5. If a display changes, review the fallback notice and adjust the layout if needed.

## Documentation

- `docs/USER_MANUAL.md`
- `docs/DESKTOP_SHELL.md`
- `docs/MULTI_MONITOR_LAYOUTS.md`
- `docs/WORKSPACE_LAYOUTS.md`
- `docs/REMOTE_ACCESS.md`
- `docs/UI_ARCHITECTURE.md`
- `docs/MEMORY_ARCHITECTURE.md`
- `docs/RETRIEVAL_PIPELINE.md`
- `docs/TASK_PACKETS.md`
- `docs/USEFULNESS_AND_REPAIR.md`
- `docs/TOOL_GATEWAY.md`
- `docs/CAPABILITY_BROKERS.md`
- `docs/POLICY_AND_APPROVALS.md`
- `docs/AUDIT_AND_TRACE.md`
- `docs/architecture/context_restore_scoring.md`
- `docs/operations/restore_scoring.md`
- `docs/operations/nvidia_dcgm.md`
- `docs/operations/intel_level_zero.md`
- `docs/operations/huggingface_tei.md`

Surface/system references:

- `docs/CHAT.md`
- `docs/WORKBENCH.md`
- `docs/CANVAS.md`
- `docs/JOBS_AND_APPROVALS.md`
- `docs/DOSSIERS.md`
- `docs/DATA_INTEGRITY_AND_WIRING.md`

## Repository Layout

- `apps/desktop` - Tauri + React desktop shell
- `services/core` - Go core service
- `packages/shared` - shared contracts
- `packages/ui` - shared UI primitives
