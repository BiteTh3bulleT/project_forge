# Desktop Shell

## Intent

FORGE uses a desktop-style shell for serious AI workstation use.

The shell now supports:

- multiple real windows
- monitor-aware placement
- shared workspace state across windows
- named layout activation and restore
- graceful reflow when displays change

The shell does not attempt to be a fake operating system or a toy window manager.

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

The `main` Tauri window is always kept as the recovery anchor for restore and fallback.

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

## Fallback Model

If a saved display is unavailable:

- the affected window is moved onto an available display
- the fallback reason is recorded
- the shell shows a fallback notice
- the operator can reopen `#/layouts` and change assignments

Invisible orphan windows are not accepted behavior.
