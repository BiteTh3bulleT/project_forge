# FORGE Phase G7 — Global Multi-Monitor Desktop Hosts

## Mission

Upgrade the FORGE operator desktop from a single-window shell into a true multi-monitor desktop environment.

FORGE must behave more like a window manager:

1. Every monitor gets a FORGE desktop host.
2. App windows are global desktop objects, not trapped inside one monitor.
3. Windows can migrate across monitor boundaries.
4. Taskbar/window focus works across all monitors.
5. Window state persists: host, monitor, position, size, route/tool, minimized/maximized.
6. Layout restore recreates the full desktop across monitors.
7. Secondary monitors are populated even when no explicit layout exists.
8. Closing/minimizing/focusing works from any monitor.
9. Future native app windows can be tracked alongside FORGE in-shell windows.

This phase is about desktop-shell correctness, not adding new AI/autonomy behavior.

FORGE is already a late-alpha local AI runtime with a Tauri desktop shell and NixOS operator-session direction; this phase continues that path by making the operator desktop behave like a real OS surface instead of a single monitor dashboard. :contentReference[oaicite:0]{index=0}

---

## Target Phase Name

`Phase G7 — Global Multi-Monitor Desktop Hosts`

Optional sub-label:

`OS Shell Behavior Pass`

---

## Read First

Inspect these files before editing:

- `apps/desktop/src/layout/AppShell.tsx`
- `apps/desktop/src/stores/desktopWindowStore.ts`
- `apps/desktop/src/stores/desktopShellStore.ts`
- `apps/desktop/src/stores/workspaceLayoutStore.ts`
- `apps/desktop/src/lib/desktop.ts`
- `apps/desktop/src/App.tsx`
- `apps/desktop/src/pages/LayoutsPage.tsx`
- `apps/desktop/src/pages/OperatorAppsPage.tsx`
- `apps/desktop/src-tauri/src/main.rs`
- `nix/packages/forge-shell-session.nix`
- `nix/packages/forge-operator-session.nix`
- `nix/nixos/modules/forge-shell-session.nix`
- existing desktop tests under `apps/desktop/src/**/*.test.tsx`

Also inspect any code related to:

- monitor detection
- Tauri window labels
- workspace layout persistence
- shell host windows
- detached tool windows
- taskbar/dock rendering
- window drag/resize behavior

---

# Current Problem

The current shell is moving toward an OS-like operator surface, but multi-monitor behavior is not yet complete.

Current likely limitations:

- The main shell host is tied to one monitor.
- Secondary monitors may remain blank or underused.
- In-shell windows are likely scoped to a single host/window.
- Taskbar/focus behavior may be local, not global.
- Dragging windows across monitor boundaries likely does not migrate the window record.
- Layout persistence may not fully restore host/monitor placement.

The target is full desktop behavior across monitors.

---

# Required Architecture

## Desktop Host

A `DesktopHost` is a FORGE shell instance bound to a detected monitor.

Each host must have:

- stable `hostLabel`
- monitor identity
- monitor bounds
- shell window label
- host role: `main` or `secondary`
- active/focused state
- visible desktop surface even when no windows exist

Example labels:

```text
main
forge-monitor-2
forge-monitor-3
````

Do not use unstable random labels.

## Global Window Object

A desktop window is global.

Each window must have at minimum:

```text
id
hostLabel
monitorId
route/tool/surface id
title
x
y
width
height
zIndex
focused
minimized
maximized
createdAt
updatedAt
```

Optional future fields:

```text
nativeWindowId
externalProcessId
appId
workspaceId
layoutId
restoreGeometry
```

## Global Desktop State

The desktop window store must treat all windows as global objects.

Hosts render a filtered view:

```text
host windows = all windows where window.hostLabel == this hostLabel
```

The taskbar may render all windows globally, or per-host with global awareness.

---

# Phase Plan

## Phase 1 — Global Multi-Monitor Desktop Hosts

### Goal

Auto-create one FORGE shell host per detected monitor.

### Requirements

1. Detect all monitors through the existing Tauri/desktop monitor mechanism.
2. Keep the primary/main shell host as:

```text
main
```

3. Create secondary shell hosts with stable labels:

```text
forge-monitor-2
forge-monitor-3
forge-monitor-N
```

4. Each host renders the same FORGE desktop shell.
5. Each host is scoped to its monitor.
6. Blank secondary monitors render a neutral FORGE desktop surface.
7. Secondary hosts must not show broken/empty app state.
8. Host creation must be idempotent.
9. Host labels must survive refresh/re-render where monitor identity is stable.
10. If monitor detection fails, FORGE must still render the main host normally.

### Expected User Result

When FORGE boots with multiple monitors:

* primary monitor shows main desktop
* secondary monitor(s) show FORGE desktop surface
* no monitor is blank unless the OS itself fails to expose it
* shell does not duplicate the same app window on every monitor

---

## Phase 2 — Cross-Host Window Migration

### Goal

Allow in-shell FORGE windows to move across monitor/host boundaries.

### Requirements

1. Update `desktopWindowStore` so every window has:

```text
hostLabel
monitorId
```

2. During window drag, track pointer/global position.
3. Determine whether pointer crosses into another monitor’s bounds.
4. If crossed, update the window record:

```text
window.hostLabel = targetHostLabel
window.monitorId = targetMonitorId
```

5. Preserve:

* route/tool state
* title
* window id
* width/height
* minimized/maximized state where applicable
* z/focus behavior

6. Convert geometry when migrating:

* from source host-local coordinates
* to target host-local coordinates

7. Clamp migrated window into the target monitor’s usable bounds.
8. Migration must be smooth and not duplicate the window.
9. The taskbar must still see the window globally.
10. Focus must transfer with the migrated window.

### Expected User Result

Operator can drag a FORGE shell window from monitor 1 to monitor 2 and it becomes owned/rendered by monitor 2 without losing the app state.

---

## Phase 3 — Global Taskbar and Focus

### Goal

Make taskbar/focus behavior work across all hosts.

### Requirements

1. The taskbar/window list must have global visibility.
2. Clicking a taskbar item from any monitor focuses the correct window on the correct host.
3. Minimize/restore works from any monitor.
4. Close works from any monitor.
5. Focus state is global and singular unless explicitly supporting per-monitor active focus.
6. Z-index ordering must be host-aware but globally consistent.
7. A focused window on monitor 2 should not incorrectly appear focused on monitor 1.
8. If a window is restored from minimized state, it restores to its last `hostLabel`/monitor unless that monitor is gone.

### Expected User Result

The operator can manage every open FORGE window from any monitor/taskbar without losing focus or duplicating state.

---

## Phase 4 — Layout Persistence and Restore

### Goal

Persist and restore the entire multi-monitor desktop.

### Requirements

Persist:

```text
hostLabel
monitorId
monitor signature
window route/tool
position
size
minimized
maximized
zIndex
workspace
layout id
```

On restore:

1. Recreate all hosts for detected monitors.
2. Restore windows to their saved host/monitor if available.
3. If a saved monitor is missing, migrate its windows to the primary host.
4. If a new monitor exists with no saved layout, show neutral desktop surface.
5. Clamp all windows to visible bounds.
6. Preserve route/tool state.
7. Do not drop windows silently.

### Expected User Result

An operator can save a multi-monitor layout, restart FORGE, and get the full desktop back across monitors.

---

## Phase 5 — Monitor Unplug/Replug Handling

### Goal

Handle monitor topology changes safely.

### Requirements

When a monitor disappears:

1. Detect missing host/monitor.
2. Move its windows to primary host or nearest available host.
3. Preserve original host/monitor as `lastKnownHostLabel` if useful.
4. Avoid losing/minimizing windows silently.
5. Emit a visible operator notification or status event.

When a monitor reappears:

1. Detect stable monitor identity if possible.
2. Offer restore/migration back if layout matches.
3. Do not automatically teleport windows unless policy is clear and tested.

---

## Phase 6 — Future Native Window Tracking

### Goal

Prepare the desktop model for native app windows.

This phase does not need to fully control native OS windows yet, but the model should not block it.

Future native app objects may include:

```text
nativeWindowId
processId
appId
title
hostLabel
monitorId
bounds
focused
minimized
```

Native windows should eventually appear beside FORGE in-shell windows in taskbar/window management.

Do not overbuild native tracking in this phase unless existing Tauri APIs already support it safely.

---

# Implementation Guidance

## Store Changes

Update `desktopWindowStore` or equivalent so window objects are not local to one shell.

Add fields:

```ts
hostLabel: string;
monitorId: string | null;
```

Add actions:

```ts
moveWindowToHost(windowId, hostLabel, monitorId, geometry)
focusWindowGlobal(windowId)
minimizeWindowGlobal(windowId)
restoreWindowGlobal(windowId)
closeWindowGlobal(windowId)
getWindowsForHost(hostLabel)
getAllWindows()
```

Do not create separate per-monitor stores unless there is a strong reason. One global store is simpler and more OS-like.

## Shell Host Store

Update `desktopShellStore` or equivalent with host records:

```ts
type DesktopHost = {
  hostLabel: string;
  monitorId: string;
  monitorIndex: number;
  bounds: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  role: "main" | "secondary";
  active: boolean;
};
```

Add actions:

```ts
syncHostsFromMonitors(monitors)
getHostForMonitor(monitorId)
getHostAtGlobalPoint(x, y)
getPrimaryHost()
```

## Geometry Conversion

Add helpers:

```ts
globalToHostPoint(globalPoint, hostBounds)
hostToGlobalPoint(hostPoint, hostBounds)
clampWindowToHost(windowGeometry, hostBounds)
findHostForGlobalPoint(globalPoint)
```

These should be tested.

## Drag Migration

During drag:

1. Track global pointer position.
2. Find target host by global point.
3. If target host differs from current host, migrate.
4. Convert geometry.
5. Continue drag on the new host if possible.

If seamless drag continuation is hard in the first pass, acceptable fallback:

* migrate on drag end if pointer ended on another monitor

Document the limitation if using fallback.

## Taskbar

Decide behavior:

### Recommended v1

Each monitor shows a taskbar, but each taskbar can see all windows globally.

Render:

* local windows first
* remote windows second or with monitor badge

Clicking remote window focuses/restores it on its host.

### Alternative

Primary monitor has global taskbar; secondary monitors have local taskbars.

Recommended v1 is global taskbar everywhere because it is simpler for operator awareness.

---

# Testing Requirements

Add or update tests for:

## Host Creation

* one monitor creates only `main`
* two monitors create `main` + `forge-monitor-2`
* three monitors create stable labels
* monitor reorder keeps stable identity where possible
* failed monitor detection falls back to `main`

## Window Store

* create window with hostLabel
* get windows by host
* focus window globally
* minimize/restore globally
* close globally
* move window to another host
* preserve route/tool state during migration

## Geometry

* global to host coordinate conversion
* host to global coordinate conversion
* clamp window to host bounds
* detect target host by global point
* migration from monitor 1 to monitor 2

## Layout Persistence

* save multi-monitor layout
* restore windows to correct host
* restore with missing monitor migrates windows to primary
* restore with new monitor creates blank host
* minimized/maximized state persists

## UI

* secondary monitor host renders neutral desktop
* taskbar renders global windows
* clicking taskbar item focuses remote host window
* closing window from non-owning host closes correct window
* no duplicate windows rendered across hosts

---

# Validation Commands

Run what is practical:

```bash
git status --short
rg -n "desktopWindowStore|desktopShellStore|workspaceLayoutStore|monitor|hostLabel|AppShell|listForgeWindows|monitorSignature" apps/desktop/src apps/desktop/src-tauri
```

Frontend:

```bash
cd apps/desktop
npm run typecheck
npm test
npm run build
```

Root:

```bash
npm run typecheck
npm test
npm run build
```

If Tauri/Nix validation is available:

```bash
nix flake check
```

If a real multi-monitor environment is unavailable, add deterministic unit tests for monitor geometry and host sync. Record manual validation as pending.

---

# Documentation Updates

Create or update:

```text
docs/architecture/operator_desktop_multi_monitor.md
docs/operations/operator_desktop.md
docs/status/operator_desktop_status.md
```

Document:

* desktop host model
* global window model
* monitor labels
* drag migration behavior
* taskbar behavior
* layout persistence behavior
* unplug/replug behavior
* what is implemented now vs future

Use status labels:

```text
LIVE
PARTIAL
FUTURE
```

Do not overclaim native app tracking unless actually implemented.

---

# Safety and Boundary Rules

This phase must not change backend authority behavior.

Do not touch:

* Control Lane semantic mutation
* Gateway execution policy
* Model runtime governance
* Dream Mode promotion
* Memory/truth authority
* Host kernel mutation flags
* Nix safe-mode defaults

The operator desktop may manage shell windows, not mutate system authority.

---

# What Not To Do

* Do not create a separate isolated window store per monitor.
* Do not duplicate app windows on every monitor.
* Do not lose route/tool state during migration.
* Do not assume monitor indexes are permanently stable if monitor IDs exist.
* Do not silently drop windows when monitors disappear.
* Do not add arbitrary native app control.
* Do not introduce arbitrary command execution.
* Do not break single-monitor behavior.
* Do not require multiple monitors for normal boot.
* Do not mark native app tracking LIVE unless real native window tracking exists.
* Do not modify backend kernel/gateway/autonomy behavior in this phase.
* Do not ship untested geometry conversion logic.
* Do not bury monitor logic in random React components; centralize it in store/helpers.

---

# Definition of Done

Phase G7 is complete when:

1. FORGE detects monitors and creates one desktop host per monitor.
2. Primary host remains `main`.
3. Secondary hosts have stable labels like `forge-monitor-2`.
4. Blank secondary monitors render a neutral FORGE desktop surface.
5. Desktop windows include `hostLabel` and `monitorId`.
6. Windows can migrate between hosts at least on drag end.
7. Taskbar can see and manage windows globally.
8. Focus/minimize/restore/close work across monitors.
9. Layout persistence records host/monitor/window state.
10. Layout restore recreates multi-monitor state.
11. Missing monitor restore does not lose windows.
12. Tests cover monitor host creation, geometry conversion, window migration, and layout restore.
13. Single-monitor behavior remains unchanged.
14. Docs describe the desktop-host model and status honestly.

---

# Final Response Format

Do not dump code in chat.

Respond with:

```text
Implemented Phase G7 — Global Multi-Monitor Desktop Hosts.

Files changed:
- ...

Behavior:
- ...

Validation:
- ...

Status:
- Phase 1:
- Phase 2:
- Phase 3:
- Phase 4:
- Future native app tracking:

Remaining gaps:
- ...
