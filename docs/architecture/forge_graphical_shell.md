# FORGE Graphical Shell

Phase G1 defines FORGE as the graphical shell session for a NixOS-based FORGE-OS machine. Phase G2 adds a manually launchable Nix wrapper for that shell session. Phase G3 defines the Nix-packaged desktop shell target while preserving the same safe session boundaries. Phase G3.5 is the real Tauri Nix build phase. Phase G4 is the opt-in Wayland shell session integration lane. Phase G5 adds a test-only VirtualBox/minimal NixOS graphics profile for manual TTY launch. Phase G6 adds read-only system surfaces inside the graphical shell.

Current G3.5 status: `packages.forge-desktop-shell` and `apps.forge-desktop-shell` now point at a real Tauri build derivation and the package advertises `passthru.containsTauriBinary = true`. `nix build .#forge-desktop-shell` succeeds on Linux and produces both the stable `forge-desktop-shell` wrapper and the underlying `forge_desktop` binary.

Current G4 status: the intended integration is a selectable Wayland session, not a desktop replacement by default. The target flow is display-manager/session selection -> lightweight Wayland compositor substrate, preferably Cage when cleanly available through Nixpkgs -> `forge-shell-session` -> packaged `forge-desktop-shell` -> local `forge-core`. `forge-shell-session` remains in the path because it owns safe shell environment defaults and binary selection.

Current G5 status: `nix/nixos/profiles/forge-vbox-graphics-test.nix` is an opt-in test profile for minimal Oracle VirtualBox NixOS graphics bring-up. The target flow is TTY login -> `forge-wayland-session` -> Cage -> `forge-shell-session` -> packaged `forge-desktop-shell` -> local `forge-core`. The profile is not a general desktop profile and must not install a full desktop environment, enable automatic login, remove TTY fallback, or expose remote graphics by default.

Current G6 status: the desktop shell has a read-only System surface at `/system` backed by `GET /forge/system/status`. It displays core reachability, request-derived core URL, health and refresh time, shell/session safety flags, bounded HostBridge diagnostics, FORGE-H resource posture/proposals, bounded execution availability and safety fields, modelruntime availability, FORGE-K activation readiness, storage posture, approval queue wiring, and recent warnings. It does not expose mutation controls, execute tools, run host commands, load/unload models, write semantic memory, or make FORGE-K live authority.

This is not a web dashboard controlling a headless server. FORGE is intended to become the visible operating interface: the desktop shell, launcher, workspace surface, command center, approval surface, and system context surface. NixOS remains the boot, hardware, graphics, service, and host configuration substrate.

## Stack Position

| Layer | Responsibility |
|---|---|
| Hardware | CPU, RAM, storage, network, GPU, sensors, firmware |
| Linux kernel | Drivers, filesystems, networking, cgroups, process substrate |
| NixOS | Declarative host configuration, services, display/session plumbing |
| Display/session layer | Login/session activation, compositor/session environment |
| FORGE graphical shell | Visible operator shell and AI-OS work surface |
| `forge-core` and governed services | Gateway, permissions, lanes, approvals, audit, memory, HostBridge/FORGE-H, modelruntime status |
| Operator | Human authority for approvals, judgment, and rollback decisions |

FORGE is the graphical shell above NixOS. It is not the kernel, not the display server, not the package manager, and not a shortcut around existing FORGE authority boundaries.

## G1/G2/G3/G3.5/G4 Scope

G1 defines the shell session contract and an inert NixOS module foundation. G2 adds a `forge-shell-session` package and flake app that set safe shell-session environment variables and launch an existing local Tauri `forge_desktop` binary when one is available. G3 targets a Nix-buildable `forge-desktop-shell` package for the actual desktop/Tauri shell and wires the session wrapper to prefer that package when available. G3.5 implements the real Linux Tauri Nix build.

G4 adds the documentation/status lane for the real session integration contract: a NixOS-provided, disabled-by-default Wayland session that can be selected manually without removing the user's normal desktop. It may introduce a `forge-wayland-session` wrapper or a module-generated Wayland session descriptor, but it must launch the existing `forge-shell-session` wrapper inside the compositor instead of bypassing it.

G5 adds a minimal VM graphics test profile. The profile is active only when an operator imports it into a NixOS configuration or uses its flake module output. It installs the existing FORGE shell wrappers and minimal graphics/session support for TTY launch; it does not make FORGE the compositor, implement a custom compositor, replace the display manager, or change the authority model.

G6 adds operator visibility surfaces to the existing desktop shell. The shell reads bounded summaries from `forge-core`; it does not query host state directly, run system commands, approve proposals, execute FORGE-H actions, load or unload models, or write memory. Missing data must be shown as unavailable or not wired rather than represented as healthy.

The only G1/G2/G3/G3.5 launch mode is `fullscreen-shell`. Future modes may include:

- `kiosk`
- `compositor-integrated`
- `remote-operator`
- `multi-monitor-shell`

Those modes are not implemented or promised by G1/G2/G3/G3.5. G4 narrows the next step to the `fullscreen-shell` Wayland path.

G1/G2/G3/G3.5 does not:

- edit Go services
- replace the user's desktop
- enable autologin
- add compositor dependencies
- install or require a compositor
- restart services
- rebuild NixOS
- mutate host state
- change modelruntime behavior
- write semantic memory
- route live authority through FORGE-K

G4 must preserve the same authority boundaries. It may add selectable Wayland session plumbing, but it must not:

- enable the session by default
- enable autologin
- remove KDE, GNOME, another desktop, or TTY fallback
- run `systemctl`, `nixos-rebuild`, package-manager mutation, `modprobe`, `rmmod`, reboot, or shutdown commands from shell/session code
- load, unload, or spawn model runtimes
- write semantic memory directly
- make FORGE-K live authority

G5 must preserve the same boundaries. It may provide a test profile for VirtualBox graphics bring-up, but it must not enable automatic login, remove TTY fallback, install a full desktop environment, expose remote graphics by default, mutate host state from wrappers, load or unload models, write semantic memory, or bypass gateway, permissions, lanes, audit, controllane, memory, modelruntime, FORGE-H, or FORGE-K authority.

G6 must preserve the same boundaries. It may add read-only API and UI surfaces, but it must not add public mutation routes, restart/shutdown/rebuild controls, model load/unload controls, cleanup/delete controls, raw log dumps, raw memory exports, shell-side host commands, direct retrieval execution, semantic memory writes, or FORGE-K live authority.

## Shell Responsibilities

The FORGE graphical shell owns the visible operating surface for a FORGE-OS session. Over time that surface should include:

- desktop/workspace surface
- launcher/start menu
- command palette
- workspace switcher
- window/panel surface
- system/resource status
- notification area
- approval queue
- memory/journal browser
- host diagnostics panel
- model/runtime status
- future Dream Mode review surface

G1/G2/G3/G3.5 does not need to implement all of these. It defines the contract that future work must respect, provides the first manual launcher, and adds the package boundary for the desktop shell binary.

## Existing Desktop App Mapping

The repository already has a desktop/Tauri development path documented through root npm scripts such as `npm run desktop`, `npm run dev`, and `npm -w @forge/desktop run build`.

G2/G3/G3.5 identifies the current shell implementation as:

- `apps/desktop`, workspace package `@forge/desktop`
- `apps/desktop/src-tauri`, Rust package `forge_desktop`
- `apps/desktop/src-tauri/tauri.conf.json`
- binary paths `apps/desktop/src-tauri/target/release/forge_desktop` and `apps/desktop/src-tauri/target/debug/forge_desktop`

The G3/G3.5 package target is `packages.forge-desktop-shell`, with a stable `/bin/forge-desktop-shell` executable and optionally `apps.forge-desktop-shell`. The package also exposes the underlying Tauri binary as `/bin/forge_desktop`; the stable wrapper command remains the operator-facing path. The package builds the real Tauri app and sets `passthru.containsTauriBinary = true`.

The `forge-shell-session` wrapper must select binaries in this order:

1. `FORGE_SHELL_BINARY` when set and executable.
2. A Nix-provided `forge-desktop-shell` binary when wired into the wrapper and marked with `passthru.containsTauriBinary = true`.
3. `apps/desktop/src-tauri/target/release/forge_desktop`.
4. `apps/desktop/src-tauri/target/debug/forge_desktop`.
5. Loud failure with exact build and override instructions.

The G2 local-binary fallback remains part of the G3.5 contract. Desktop-shell packages must be skipped by `forge-shell-session` until they advertise a real Tauri binary. Because the current package advertises `passthru.containsTauriBinary = true`, the wrapper is expected to prefer it after `FORGE_SHELL_BINARY`. `FORGE_CORE_URL` remains an environment override, defaulting to `http://127.0.0.1:18492`, and `VITE_FORGE_API_URL` should follow it for current desktop code paths.

Nix/Tauri package limitations must remain explicit. The current G3.5 package is validated as a Linux Nix package. It does not add compositor integration, display-manager integration, autostart, desktop replacement, or host mutation. G4 is the separate opt-in compositor/session lane and does not change the G3.5 package truth.

Exact operator commands:

```bash
nix build .#forge-desktop-shell
nix run .#forge-desktop-shell
nix build .#forge-shell-session
nix run .#forge-shell-session
```

G5 manual VirtualBox TTY commands:

```bash
nix build .#forge-desktop-shell
nix build .#forge-shell-session
nix build .#forge-wayland-session
nix run .#forge-wayland-session
```

Local development fallback commands:

```bash
nix develop .#desktop
npm install
npm run build:desktop
npm -w @forge/desktop run tauri -- build
FORGE_SHELL_BINARY="$PWD/apps/desktop/src-tauri/target/release/forge_desktop" nix run .#forge-shell-session
```

## Shell-To-Core Boundary

The shell talks to `forge-core` through governed local APIs/interfaces. It may render structured state and submit user requests, but it does not own truth authority, command authority, approval authority, memory authority, modelruntime authority, or FORGE-K authority.

G6 introduces `GET /forge/system/status` as a bounded read-only status route for the shell. The route composes core/session metadata, SQLite storage posture, HostBridge summary data, FORGE-H advisory policy/proposals, modelruntime health, and approval queue wiring. The reported core URL is derived from the current request host and scheme rather than discovered through host probing. Command-backed HostBridge probes are disabled for this route, so the shell does not become a service-control or host-command path.

Allowed shell behavior:

- read host diagnostics through existing safe/internal paths when available
- read FORGE-H policy, proposal, and execution records through governed paths when available
- show service and resource status
- show approval queues when existing APIs support them
- show shell-safe placeholders when APIs are not available yet
- refresh status on a conservative manual or 30 second interval
- submit operator requests through gateway, permission, lane, approval, audit, and controllane paths

Forbidden shell behavior:

- direct `systemctl`
- direct `nixos-rebuild`
- direct package-manager mutation at runtime
- enabling autologin
- replacing the user's desktop/session choices
- installing, starting, or requiring a compositor in G3.5
- bypassing `forge-shell-session` in G4 Wayland launchers
- direct kernel or module calls
- direct modelruntime load/unload
- direct filesystem cleanup
- direct semantic memory writes
- direct gateway execution bypass
- direct mutation of host configuration
- treating model output as canonical state
- treating FORGE-K simulator code as live daemon authority

G6 specifically forbids approval, rejection, execution, model load/unload, cleanup, restart, shutdown, rebuild, and package mutation controls on the System surface.

## System Context Principle

FORGE as graphical shell provides full operating awareness, not full LLM context.

The shell may observe structured system/session context:

- active workspace
- open panels/windows
- current project
- resource posture
- service health
- model status
- approvals
- recent errors
- user-triggered actions

That context is for operator awareness and governed request construction. The context compiler decides what subset reaches an LLM.

Do not dump raw full system state into prompts. Do not dump raw logs, raw desktop state, raw process lists, raw memory records, raw host diagnostics, or full session state into model prompts. Prompt context must stay bounded, relevant, provenance-linked where applicable, and routed through the existing context compilation path.

## Safe Fallback And Rollback

The shell session must remain opt-in. G3.5 does not change the disabled-by-default posture, and G4 must keep the same conservative defaults:

- `enable = false`
- `mode = "fullscreen-shell"`
- `displayBackend = "wayland"`
- `compositor = "cage"`
- `autoStart = false`
- `coreURL = "http://127.0.0.1:18492"`
- `safeMode = true`
- `fullscreen = true`

Autologin must not be enabled by default. Existing desktop environments must not be disabled by default. A normal desktop or TTY fallback must remain available. G4 may make a FORGE Shell session selectable when explicitly enabled, but it must not make that session the default automatically. G4 does not introduce a display-manager replacement, host mutation path, modelruntime mutation path, semantic memory write path, or FORGE-K live authority path.

Rollback expectations:

- disable the FORGE shell session option
- select a normal desktop session from the display manager
- use TTY login if the display session is broken
- keep existing desktop/session choices available
- keep `/forge` data intact
- keep `forge-core` usable through existing workflows
- keep manual `nix run .#forge-shell-session` and `nix run .#forge-desktop-shell` paths available
- use `FORGE_SHELL_BINARY=/path/to/forge_desktop forge-shell-session` when the packaged shell is unavailable
- fall back to `npm run desktop` or the local Tauri release/debug binary paths during package work
- restart the display manager or reboot only when the operator chooses

## Session Scaffolding And Launcher Shape

G1 adds an inert, opt-in NixOS module at `nix/nixos/modules/forge-shell-session.nix`. G2 adds `nix/packages/forge-shell-session.nix` and exposes `packages.forge-shell-session` plus `apps.forge-shell-session`. G3 adds the target package `nix/packages/forge-desktop-shell.nix` and flake output `packages.forge-desktop-shell`; if a stable app is exposed, it should be `apps.forge-desktop-shell`. G3.5 turns that package into the real Linux Tauri derivation.

The module prepares:

- a session descriptor
- environment variables such as `FORGE_CORE_URL`
- a desktop/session entry placeholder
- runtime directory for shell session state
- safe local core URL wiring

G4 extends this shape with explicit Wayland session semantics:

- `forge.shellSession.wayland.enable` remains disabled unless `forge.shellSession.enable` is explicitly enabled
- `forge.shellSession.wayland.sessionName` names the selectable session
- `forge.shellSession.wayland.package` or equivalent wrapper provides the session command when implemented
- Cage is the preferred lightweight compositor substrate when available through Nixpkgs
- the compositor launches `forge-shell-session`, not `forge-desktop-shell` directly
- generated environment/session files must keep host mutation, direct system control, model mutation, semantic memory write, and FORGE-K live authority flags set to `false`

The wrapper prepares:

- `FORGE_SHELL_SESSION_ENABLED=true`
- `FORGE_SHELL_MODE=fullscreen-shell`
- `FORGE_CORE_URL=http://127.0.0.1:18492` unless overridden
- `VITE_FORGE_API_URL=$FORGE_CORE_URL` unless overridden
- `FORGE_SHELL_SAFE_MODE=true`
- `FORGE_SHELL_FULLSCREEN=true`
- explicit false flags for host mutation, direct system control, model mutation, semantic memory writes, and FORGE-K live authority

The former launcher placeholder accepted `FORGE_DESKTOP_SHELL_BINARY=/path/to/forge_desktop` for direct operator testing. The current real-build derivation still supports that override for wrapper tests; normal local fallback should use `FORGE_SHELL_BINARY` at the session wrapper layer.

G3.5 does not add a systemd user service, autologin, compositor dependency, display-manager replacement, or automatic launch path. G4 is the opt-in Wayland session integration lane. The preferred substrate is Cage because it is small and suitable for a single full-screen shell; sway, Hyprland, or X11 fallback paths require separate justification and tests. G4 still must not claim host authority: NixOS owns boot, hardware, display-manager plumbing, service management, packages, and rollback.

## Authority Non-Changes

G1/G2/G3/G3.5 does not change:

- host mutation authority
- gateway authority
- permission authority
- approval authority
- audit authority
- controllane authority
- semantic memory authority
- modelruntime authority
- FORGE-H resource execution authority
- FORGE-K simulator-only/live authority separation

The graphical shell is an operator surface. It is not a new authority plane.
