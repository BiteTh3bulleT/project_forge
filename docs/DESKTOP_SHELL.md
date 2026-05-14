# Desktop Shell

## Intent

FORGE uses a desktop-style shell for serious AI workstation use.

The shell now supports:

- multiple real windows
- monitor-aware placement
- shared workspace state across windows
- named layout activation and restore
- graceful reflow when displays change
- compositor-reported native app taskbar entries
- bounded compositor controls for native app focus/minimize/maximize/fullscreen/close

The shell is evolving into the FORGE desktop environment on top of the labwc
Wayland compositor substrate. labwc still owns real native app placement,
interactive drag, resize, output routing, and compositor policy. FORGE owns the
operator desktop surface, taskbar, launcher, in-shell FORGE windows, and bounded
native-window controls exposed through the compositor.

The generated operator session asks labwc to load its default keyboard and mouse
bindings. That keeps normal window-manager behavior such as window switching,
server-side decoration interactions, and modifier-drag movement available while
FORGE develops its own shell surfaces.

## Shell Regions

### Top Bar

Backed by real state:

- workspace identity and core status
- active model and adapter readiness
- queue counts
- active layout switcher
- direct access to layout management
- clock and shell mode

### Dock

Primary tool launcher for day-to-day surfaces.

### Task Strip

Per-window task strip for the surfaces open inside that window only.

This strip is no longer shared across all windows. Each shell window has its own surface session state.

### Workspace Surface

Each window hosts one active surface at a time inside a framed shell surface.

### Context Panel

Shows real context for:

- current object
- active layout
- open workspace windows
- recent surfaces
- queue state

## Window Model

FORGE now distinguishes between:

- `runtime window label`: the actual Tauri window label
- `layout window record`: the saved window slot in a named preset
- `surface session`: the routes open inside one window
- `native window snapshot`: a compositor-reported Linux toplevel

The `main` Tauri window is always kept as the recovery anchor for restore and fallback.

FORGE native webview windows now go through a backend-owned desktop window
manager in `apps/desktop/src-tauri/src/window_manager.rs`. The frontend calls
`apps/desktop/src/lib/windowManager.ts`, and the Rust manager validates labels,
prevents singleton duplicate creation, updates lifecycle state, emits registry
events, and persists only safe geometry/layout metadata.

Secondary shell-host windows such as `forge-monitor-2` are now backend window
manager descriptors (`shell_host`) rather than frontend-created webview windows.
The backend derives those labels from sanitized host IDs and routes them through
`/?host=<label>`.

In-shell FORGE tool windows are global desktop objects owned by the FORGE
frontend store. Native Linux app windows are compositor objects. FORGE may list
and request bounded actions for native windows, but it does not yet have a
compositor-native move/resize/output migration protocol.

Native app visibility now flows through a backend-owned Tauri registry. The
registry refreshes from the compositor, preserves stable first-seen/last-seen
lifecycle metadata for each toplevel, marks missing windows closed, and only
executes focus/minimize/maximize/fullscreen/close actions against active
registered windows. The frontend taskbar is a consumer of that registry; it no
longer treats raw polling output as the lifecycle owner.

## Monitor Model

FORGE uses the real Tauri monitor list.

Tauri does not expose a permanent opaque monitor UUID in the frontend API. FORGE therefore derives a stable usable monitor key from the real monitor fields Tauri does provide:

- monitor name
- monitor position
- monitor size
- scale factor

That key is not random and is not simulated. It is derived from real monitor data.

## Restore Model

On relaunch:

1. the `main` shell window starts
2. FORGE loads the active layout
3. FORGE recreates any missing secondary windows
4. FORGE places windows on the best available displays
5. FORGE routes each window to its saved active surface

Window-manager layout writes use a temp file plus rename. Restore validates each
saved native window record independently; valid windows can be restored while
bad records are reported as restore failures, and an all-failed restore remains
an error.

## Fallback Model

If a saved display is unavailable:

- the affected window is moved onto an available display
- the fallback reason is recorded
- the shell shows a fallback notice
- the operator can reopen `#/layouts` and change assignments

Invisible orphan windows are not accepted behavior.
