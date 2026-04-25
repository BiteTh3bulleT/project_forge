# FORGE

FORGE is a local-first AI workspace for inspectable, approval-gated engineering work.

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
- Dream Mode v0 dry-run replay/consolidation reports
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

2. Start the desktop shell:

```bash
npm run desktop
```

If the desktop window does not open, first check Tauri startup logs. The most common blockers are:

- **port conflict on `5173`** (existing Vite/dev server)
- **missing Linux webkit libs** (linker errors for `webkit2gtk-4.1` / `javascriptcoregtk-4.1`)

`npm run desktop` now performs a preflight check and clears stale FORGE-local Vite listeners on `5173` automatically.
If another non-FORGE process owns `5173`, startup will stop and print that process so you can resolve it.

Typical fixes:

```bash
# if something is already serving 5173, stop it first
sudo lsof -ti :5173 | xargs -r kill -9

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
- Desktop dev server: `http://localhost:5173`
- Default shell route: `#/chat`

Dry-run maintenance endpoints:

```bash
curl -sS http://127.0.0.1:18492/api/dream/run \
  -H 'Content-Type: application/json' \
  -d '{"workspaceId":"default","laneId":"control.semantic","mode":"microdream"}'
```

Dream Mode v0 is CPU-only and proposal-only. It does not require GPU/modelruntime and does not commit canonical memory.

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
- `docs/architecture/dream_mode.md`
- `docs/operations/restore_scoring.md`
- `docs/operations/dream_mode.md`
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
