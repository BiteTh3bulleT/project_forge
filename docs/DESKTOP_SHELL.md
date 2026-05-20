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
- operator notification count and center for job, approval, and shell notification events
- active layout switcher
- direct access to layout management
- clock and shell mode

### Dock

Primary tool launcher for day-to-day surfaces.

### App Launcher

The operator app launcher is backend-owned. It always lists the curated FORGE
toolbelt first, then appends visible native `.desktop` applications discovered
from the system XDG application directories. Curated entries keep stable IDs and
ordering, so Terminal, Files, Editor, browser, diagnostics, and FORGE wrappers
remain predictable.

Scanned native entries use `xdg:<desktop-file>` IDs and launch through parsed
`Exec=` tokens, not shell strings. The backend drops desktop field-code
placeholders such as `%U`, rejects hidden/no-display/terminal entries, and
refuses shell or host-mutation launch paths including shell interpreters,
`sudo`, `pkexec`, `systemctl`, and `nixos-rebuild`. Scanned app discovery does
not grant new FORGE authority; it only exposes normal native application
launches through bounded Tauri command handling.

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

## Notifications

The shell exposes an operator notification center from the top bar. The current
frontend bridge polls core events with `api.events(20)` and promotes operator-
relevant job, approval, and notification events into the center. Raw event
history remains available through the Activity log.

On Linux, the Tauri backend also starts an opportunistic
`org.freedesktop.Notifications` service at `/org/freedesktop/Notifications`
using `zbus`. It implements `Notify`, `CloseNotification`, `GetCapabilities`,
`GetServerInformation`, and the standard notification signals. Native app
notifications are emitted into the shell as `forge://notification` events and
shown in the same notification center. If another notification daemon already
owns the D-Bus name, FORGE logs the unavailable service and continues running;
core event notifications still work.

## Host Power Controls

The desktop shell's Tauri binary exposes a `request_host_power_action` command that can request host `shutdown` or `reboot`. This is policy-gated, not absent:

- Default disabled. The binary reads the environment variable `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`; unless it is set to `1`, `true`, `TRUE`, `yes`, or `YES`, the command returns a `requested:false` result and does not spawn any host process.
- The Start menu reads the same policy through `read_host_power_policy`. Lock and Logout remain available, but Restart and Shutdown are disabled with the policy message until direct system control is explicitly enabled. Restart maps to the host `reboot` action.
- Enabling the gate grants host mutation authority to the desktop shell. The operator must opt in explicitly through the environment variable.
- Allowlist: only `shutdown` and `reboot` actions are accepted.
- Binary-level enforcement is unit-tested. Frontend coverage verifies that disabled policy state prevents Start menu host mutation requests.
- This supersedes earlier docs language describing the shell as "no host mutation". The accurate posture is "policy-gated host power actions, disabled by default". See also `docs/status/dangerous_capabilities.md` `shell.power_action` entry and `docs/operations/forge_graphical_shell_session.md`.

## Shell Session Controls

Lock is an interim shell-owned overlay. It covers the primary FORGE shell
window and re-authenticates with the local FORGE operator login form. It is a
usability lock for the current shell surface, not a compositor/session security
boundary against other native windows. Real session locking waits for
`ext-session-lock-v1` in Track C.

Logout remains an in-shell FORGE operator-login transition: it clears the
cached API token promise, resets in-shell desktop session state, removes the
operator login marker, and returns to the login surface.

The lower-level `request_shell_session_action` command remains bounded to
`restart_shell` and gated by `FORGE_SHELL_SESSION_ENABLED`, but it is not part
of the Start power row. The normal Start power semantics are OS-style:
Lock, Logout, Restart, and Shutdown.

## Host Settings

The Settings surface includes a read-only Host and Hardware section backed by
`GET /forge/system/host`. The endpoint uses HostBridge diagnostics for host,
CPU, memory, storage, GPU, thermal, and session/config visibility while keeping
display apply, audio control, network mutation, and power mutation outside the
route. Those subsystem blocks report status and preserve
`mutation_disabled:true`.

This is the B2 host-settings surface, not the B5 display-control surface.
Display topology can be inspected, but applying monitor layout or resolution
changes waits for compositor-owned output-management support.

The Workspace Layouts surface now records a B5 interim display layout intent:
preserve, extend, or mirror, plus the current primary monitor and detected
display order. This is saved operator preference only. The shell does not apply
output topology, resolution, or mirroring changes until the compositor
output-management gate exists.

## Command Bar Maintenance

The desktop CommandBar quick actions are a static frontend affordance in `apps/desktop/src/components/CommandBar.tsx`. They are not a backend authority registry, do not grant capabilities, and must only point at existing routes or already-governed API commands.

When adding or changing a quick action:

- update the `commandActions` list in `CommandBar.tsx`
- keep labels short and category names stable for scanability
- verify the command still resolves through existing UI/API authority paths
- update `apps/desktop/src/components/CommandBar.test.tsx` when visible actions or command behavior changes
- do not add host mutation, model lifecycle mutation, approval decisions, or gateway execution shortcuts through CommandBar without a separate authority design and tests

## Phase G8 Native App Lifecycle

Phase G8 records the desktop shell as `DESKTOP_SHELL_POLISH / OPERATOR_UI_AUTHORITY / LABWC_SUBSTRATE_PRESERVED / POLICY_GATED_HOST_POWER_DISABLED_BY_DEFAULT / NO_FORGE_K_AUTHORITY_EXPANSION`.

The shell now handles native launch lifecycle edges explicitly:

- refused native launches (`launched:false`) report the backend message and do not create taskbar placeholders
- duplicate launch requests for the same native app are blocked while a launch is pending
- pending native placeholders expire if no matching compositor window appears within 30 seconds
- compositor-reported native windows remain keyed by compositor window id, so multiple native windows can be represented separately
- bounded native action failures are visible to the operator instead of silently returning

Native taskbar left-click requests focus/restore for inactive or minimized native windows and minimize for focused non-minimized native windows. Middle-click requests bounded close. Right-click exposes bounded focus/minimize/maximize/fullscreen/close requests.

The compositor bridge does not yet report per-window action capability metadata. Until it does, unsupported native actions fail visibly and safely instead of being hidden with precise capability-aware UI.

Current G8 evidence lives in:

- `docs/reports/phase_g8_desktop_shell_verification.md`
- `docs/status/phase_g8_desktop_shell_verification.md`
- `docs/runbooks/desktop_shell_operator_smoke_test.md`

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
