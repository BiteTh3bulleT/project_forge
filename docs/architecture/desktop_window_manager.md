# Desktop Window Manager

## Status

Phase G7 now has a backend-owned FORGE Tauri window manager for native FORGE
webview windows. This is separate from the Linux compositor window registry:

- `window_manager.rs` owns FORGE Tauri window lifecycle.
- `linux_windows.rs` observes external compositor toplevels and exposes bounded
  controls for active native app windows.
- React owns in-shell panels and layout composition, not native Tauri window
  authority.

## Ownership Boundary

Frontend requests native window actions through `apps/desktop/src/lib/windowManager.ts`.
That wrapper invokes Tauri commands such as `forge_window_open`,
`forge_window_focus`, and `forge_window_close` for supported FORGE native
windows.

The Rust window manager validates the request, derives stable labels/routes,
creates or focuses the Tauri webview window, updates the registry, emits events,
and persists safe layout metadata.

The frontend still owns internal MDI-style tool panels while
`DETACHED_TAURI_TOOL_WINDOWS` is disabled. Those in-shell tool panels are not
canonical native windows.

## Window Kinds

Supported backend kinds:

- `main_shell`
- `workspace`
- `terminal`
- `memory_panel`
- `task_panel`
- `system_panel`
- `settings`
- `inspector`
- `artifact_viewer`
- `debug_console`
- `shell_host`

Singleton labels include `main`, `settings`, `memory-panel`, `task-panel`,
`system-panel`, `inspector`, and `debug-console`.

Dynamic labels are backend-derived from sanitized IDs, such as
`workspace-{id}`, `terminal-{session_id}`, `artifact-{id}`, and
`forge-monitor-{host_id}`.

Shell-host windows represent secondary native FORGE host windows for monitor
surfaces. The backend accepts either a compact sanitized host ID such as `2` or
an existing sanitized label such as `forge-monitor-2`; both produce the stable
label `forge-monitor-2` and fallback route `/?host=forge-monitor-2`. The `main`
window remains the recovery anchor and is not represented as a shell-host
window.

## Persistence

The manager persists only safe layout metadata under the Tauri app config
directory as `forge-window-layout.json`.

Persisted data is limited to labels, kinds, routes, titles, visibility/focus
flags, timestamps, and geometry. It must not include terminal scrollback, model
outputs, memory contents, secrets, raw logs, or semantic memory records.

Layout persistence writes a JSON temp file in the same directory, syncs it, and
renames it over the prior layout file. Restore validates each saved entry
against backend label and route policy before creating a window. When a restore
has both valid and invalid entries, valid windows are restored and invalid
entries are reported in `restoreFailures`; if no entry can be restored, restore
returns an error instead of reporting success.

## Events

The manager emits bounded state events:

- `forge://window/opened`
- `forge://window/closed`
- `forge://window/focused`
- `forge://window/hidden`
- `forge://window/shown`
- `forge://window/registry-updated`
- `forge://layout/restored`

Per-window events carry registry entry metadata. Registry/layout events carry a
snapshot.

## Safety

The desktop window manager does not run `systemctl`, run `nixos-rebuild`, mutate
NixOS config, write semantic memory, load or unload models, execute arbitrary
shell commands, or bypass forge-core authority.

Window labels and routes are validated before use. Unsupported current shell-host
labels are rejected rather than used as arbitrary native window labels.

## Known Limits

labwc remains the real Wayland compositor. FORGE can create/focus/hide/close its
own Tauri webview windows and request bounded actions for observed native app
windows, but it does not yet own a compositor-native move/resize/output protocol
for arbitrary external Linux windows.
