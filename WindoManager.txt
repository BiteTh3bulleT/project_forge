# FORGE Desktop Window Manager Implementation Prompt

You are working inside the `BiteTh3bulleT/project_forge` repository.

## Mission

Add a real FORGE desktop window-management layer to the existing Tauri desktop shell.

FORGE already has a Tauri desktop shell path and Nix shell-session wrappers. Do **not** rebuild the desktop from scratch. Do **not** bypass the existing `forge-shell-session`, `forge-desktop-shell`, or safe shell-session boundaries.

The goal is to add a deterministic, backend-owned window manager for the Tauri desktop app.

This is not just “make some windows open.” Build the proper authority layer.

---

## Current Assumption

The repo already contains:

- `apps/desktop`
- `apps/desktop/src-tauri`
- Tauri config under `apps/desktop/src-tauri/tauri.conf.json`
- Tauri Rust binary/package likely named `forge_desktop`
- frontend workspace under `apps/desktop/src`
- existing docs for graphical shell/session behavior
- safe shell-session rules around no direct host mutation

Treat these as existing architecture. Preserve them.

---

## Core Problem

Right now FORGE appears to have a Tauri shell, but not a dedicated FORGE Window Manager.

Add a window-management system that provides:

- stable window labels
- duplicate spawn prevention
- backend-owned window lifecycle
- open/focus/toggle/close commands
- basic layout/window registry state
- event broadcasting to frontend windows
- safe persistence of window layout/state
- documentation and tests where practical

FORGE’s window manager must behave like a shell authority layer, not random frontend window spawning.

---

## Architectural Rule

Rust/Tauri backend owns native window lifecycle.

Frontend owns internal layout/panels only.

Do **not** let React become the canonical source of native windows.

Correct model:

```text
Frontend request
  -> Tauri command
  -> Rust WindowManager
  -> Tauri window/webview operation
  -> registry update
  -> event broadcast
  -> frontend sync

Incorrect model:

React randomly opens native windows and hopes Tauri/Linux behaves.

That way lies raccoon math.

Required Deliverables
1. Rust Window Manager Module

Create a dedicated Rust module under the Tauri app, likely something like:

apps/desktop/src-tauri/src/window_manager.rs

or equivalent based on existing structure.

Implement a WindowManager or similarly named service.

It should manage:

WindowKind
WindowDescriptor
WindowRegistryEntry
WindowManagerState

Recommended window kinds:

MainShell
Workspace
Terminal
MemoryPanel
TaskPanel
SystemPanel
Settings
Inspector
ArtifactViewer
DebugConsole

Support static labels:

main
forge-shell
workspace-main
terminal-panel
memory-panel
task-panel
system-panel
settings
inspector
debug-console

Support dynamic labels:

workspace-{id}
artifact-{id}
terminal-{session_id}

Add helper functions to normalize/sanitize labels. Dynamic IDs must not allow path traversal, weird shell characters, or arbitrary labels.

2. Backend Commands

Expose Tauri commands for the frontend.

Implement commands similar to:

forge_window_open(kind, payload)
forge_window_close(label)
forge_window_focus(label)
forge_window_show(label)
forge_window_hide(label)
forge_window_toggle(label)
forge_window_list()
forge_window_snapshot()
forge_window_restore_layout(layout_id?)
forge_window_sync_state()

Use the existing app handle / manager APIs properly.

Behavior expectations:

Opening an already-existing singleton window should focus/show it instead of creating duplicates.
Closing should update registry state.
Hiding should not delete the registry entry unless the native window is actually gone.
Focus should safely no-op or return a useful error if the window is missing.
List/snapshot should return frontend-friendly JSON.
All command errors should be structured and readable.
3. Native Window Creation Policy

Create all native windows through the Window Manager.

Use stable defaults per window kind:

Settings:
  singleton
  centered
  reasonable size
  not fullscreen

DebugConsole:
  singleton
  dev/debug only unless already allowed by repo conventions

ArtifactViewer:
  multi-instance
  label: artifact-{id}

Terminal:
  either singleton panel or terminal-{session_id}, depending on current app design

MemoryPanel / TaskPanel / SystemPanel:
  strongly consider singleton native windows only if existing frontend UX requires detaching.
  Otherwise document that these are internal panels, not native windows.

Important: Do not force every panel into a native OS window. FORGE should use native Tauri windows only when useful.

Recommended split:

Native Tauri windows:
- main shell
- settings
- detached terminal
- detached artifact viewer
- debug console

Internal frontend panels:
- memory
- tasks
- system
- context
- inspector unless detached

If existing UI already treats panels differently, adapt cleanly without breaking it.

4. Window State Persistence

Add safe persistence for window registry/layout state.

Use the existing Tauri plugin setup if available. If tauri-plugin-window-state is already present, wire into it properly. If not present, add it only if dependency policy allows.

Requirements:

Persist size/position where supported.
Restore layout safely.
Do not fake restore success.
Avoid ugly startup flashing where practical by creating windows hidden first, restoring, then showing.
Do not persist sensitive runtime contents.
Do not persist model outputs, memory contents, terminal scrollback, or secrets.
Persist only geometry/layout/window labels/kinds/routes.

If adding a local JSON store, keep it under the app config/data directory, not random repo paths.

5. Event Broadcasting

When window state changes, emit events to all relevant frontend windows.

Events should be named consistently, for example:

forge://window/opened
forge://window/closed
forge://window/focused
forge://window/hidden
forge://window/shown
forge://window/registry-updated
forge://layout/restored

Payloads should include:

label
kind
route
visible
focused
timestamp
workspace_id?
artifact_id?

Do not spam events in loops.

6. Frontend Bridge

Add a clean frontend API wrapper, likely under:

apps/desktop/src/lib/windowManager.ts

or existing API location.

Expose functions like:

openForgeWindow()
closeForgeWindow()
focusForgeWindow()
toggleForgeWindow()
listForgeWindows()
subscribeToForgeWindowEvents()

The frontend should call this wrapper instead of raw Tauri APIs scattered everywhere.

Search the frontend for any direct window open/focus/close usage and replace with the wrapper where safe.

Do not create random one-off window logic inside React components.

7. Capabilities / Permissions

Review Tauri v2 capability files.

Ensure the frontend windows that need window commands can invoke them.

Add the smallest necessary permissions/capabilities.

Do not grant broad shell/system permissions.

Do not enable host mutation.

Do not bypass FORGE gateway, approval, memory, modelruntime, or shell-session rules.

8. Documentation

Add or update docs:

docs/architecture/desktop_window_manager.md

Include:

what the FORGE Window Manager owns
what the frontend owns
window kinds
label policy
native-window vs internal-panel policy
persistence rules
event names
safety boundaries
known Linux/Wayland limitations

Also update any existing graphical shell/session docs with a short note that the Tauri shell now has a backend-owned window manager.

9. Tests / Validation

Add tests where practical.

At minimum:

label normalization tests
singleton duplicate prevention tests
registry snapshot serialization tests
invalid label rejection tests
open/focus/toggle behavior unit tests if the Tauri app structure allows it

If full Tauri runtime tests are hard, add pure Rust tests around the registry logic and keep native window calls thin.

Also run existing relevant checks:

npm run build:desktop
npm -w @forge/desktop run tauri -- build
nix build .#forge-desktop-shell
nix flake check

If any command is unavailable locally, document exactly what failed and why.

Safety Boundaries

This implementation must not:

run systemctl
run nixos-rebuild
mutate NixOS config
replace the desktop session
enable autologin
install or require a compositor
bypass forge-shell-session
bypass forge-core
bypass approvals
directly write semantic memory
directly load/unload models
treat LLM output as canonical state
add uncontrolled shell command execution
create arbitrary native windows from untrusted labels/routes
expose terminal/session contents through persisted window state
make frontend React state the native-window authority
Implementation Phases
Phase 0 — Discovery

Inspect:

apps/desktop/src-tauri
apps/desktop/src
apps/desktop/package.json
apps/desktop/src-tauri/tauri.conf.json
docs/operations/forge_graphical_shell_session.md
docs/status/desktop_bringup.md

Find current Tauri command setup, frontend API style, and existing window usage.

Do not edit yet.

Produce a short implementation note in your working summary.

Phase 1 — Registry Core

Implement pure registry logic first.

Create:

WindowKind
WindowDescriptor
WindowRegistryEntry
WindowRegistry

Include:

singleton/multi-instance policy
label generation
label validation
JSON serialization
snapshot output

Add tests.

Phase 2 — Tauri Integration

Wire registry into Tauri managed state.

Register commands.

Implement actual native operations:

open
close
focus
show
hide
toggle
list
snapshot
restore

Use existing Tauri conventions in repo.

Do not break existing app startup.

Phase 3 — Frontend Bridge

Add the TypeScript wrapper.

Replace scattered direct usage with wrapper calls.

Add event subscriptions.

Make sure UI can request window actions without owning canonical state.

Phase 4 — Persistence

Add safe layout persistence.

Use plugin if already present or add carefully.

Persist only safe geometry/layout metadata.

Document persistence path and limitations.

Phase 5 — Docs and Validation

Add architecture doc.

Update shell-session doc if needed.

Run builds/tests.

Leave a final report with:

Files changed
Commands run
Tests passed
Known limitations
Follow-up recommendations
Expected Final Shape

After implementation, FORGE should have this architecture:

FORGE Desktop Shell
  frontend workspace
    internal panels
    command palette
    UI requests
  frontend windowManager.ts
    typed invoke/event wrapper
  Tauri commands
    safe bridge
  Rust WindowManager
    registry
    lifecycle
    persistence
    event broadcast
  Tauri native windows
    shell/settings/artifacts/terminal/debug

The result should feel like FORGE has a real shell-level window authority instead of a frontend duct-tape convention.

Build it clean, boring, deterministic, and auditable.

No magic. No vibes. No haunted raccoon windows.
