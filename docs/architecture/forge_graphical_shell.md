# FORGE Graphical Shell

Phase G1 defines FORGE as the graphical shell session for a NixOS-based FORGE-OS machine. Phase G2 adds a manually launchable Nix wrapper for that shell session. Phase G3 defines the Nix-packaged desktop shell target while preserving the same safe session boundaries.

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

## G1/G2/G3 Scope

G1 defines the shell session contract and an inert NixOS module foundation. G2 adds a `forge-shell-session` package and flake app that set safe shell-session environment variables and launch an existing local Tauri `forge_desktop` binary when one is available. G3 targets a Nix-buildable `forge-desktop-shell` package for the actual desktop/Tauri shell and wires the session wrapper to prefer that package when available.

The only G1/G2/G3 launch mode is `fullscreen-shell`. Future modes may include:

- `kiosk`
- `compositor-integrated`
- `remote-operator`
- `multi-monitor-shell`

Those modes are not implemented or promised by G1/G2/G3.

G1/G2/G3 does not:

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

G1/G2/G3 does not need to implement all of these. It defines the contract that future work must respect, provides the first manual launcher, and adds the package boundary for the desktop shell binary.

## Existing Desktop App Mapping

The repository already has a desktop/Tauri development path documented through root npm scripts such as `npm run desktop`, `npm run dev`, and `npm -w @forge/desktop run build`.

G2/G3 identifies the current shell implementation as:

- `apps/desktop`, workspace package `@forge/desktop`
- `apps/desktop/src-tauri`, Rust package `forge_desktop`
- `apps/desktop/src-tauri/tauri.conf.json`
- binary paths `apps/desktop/src-tauri/target/release/forge_desktop` and `apps/desktop/src-tauri/target/debug/forge_desktop`

The G3 package target is `packages.forge-desktop-shell`, with a stable `/bin/forge-desktop-shell` executable and optionally `apps.forge-desktop-shell`. The package may also expose the underlying Tauri binary as `/bin/forge_desktop`; the stable wrapper command remains the operator-facing path. Until the package contains a Nix-built Tauri binary, it may remain a loud-failing launcher placeholder with `passthru.containsTauriBinary = false`.

The `forge-shell-session` wrapper must select binaries in this order:

1. `FORGE_SHELL_BINARY` when set and executable.
2. A Nix-provided `forge-desktop-shell` binary when wired into the wrapper and marked with `passthru.containsTauriBinary = true`.
3. `apps/desktop/src-tauri/target/release/forge_desktop`.
4. `apps/desktop/src-tauri/target/debug/forge_desktop`.
5. Loud failure with exact build and override instructions.

The G2 local-binary fallback remains part of the G3 contract. Placeholder desktop-shell packages must be skipped by `forge-shell-session` until they advertise a real Tauri binary. `FORGE_CORE_URL` remains an environment override, defaulting to `http://127.0.0.1:18492`, and `VITE_FORGE_API_URL` should follow it for current desktop code paths.

Nix/Tauri package limitations must remain explicit. A G3 derivation may fail honestly when npm dependency hashing, Cargo vendor hashing, Linux WebKit/GTK runtime dependencies, or platform-specific Tauri bundle support are incomplete. It must not claim that the package contains the desktop shell without producing or wrapping the real Tauri binary.

## Shell-To-Core Boundary

The shell talks to `forge-core` through governed local APIs/interfaces. It may render structured state and submit user requests, but it does not own truth authority, command authority, approval authority, memory authority, modelruntime authority, or FORGE-K authority.

Allowed shell behavior:

- read host diagnostics through existing safe/internal paths when available
- read FORGE-H policy, proposal, and execution records through governed paths when available
- show service and resource status
- show approval queues when existing APIs support them
- show shell-safe placeholders when APIs are not available yet
- submit operator requests through gateway, permission, lane, approval, audit, and controllane paths

Forbidden shell behavior:

- direct `systemctl`
- direct `nixos-rebuild`
- direct package-manager mutation at runtime
- enabling autologin
- replacing the user's desktop/session choices
- installing, starting, or requiring a compositor in G3
- direct kernel or module calls
- direct modelruntime load/unload
- direct filesystem cleanup
- direct semantic memory writes
- direct gateway execution bypass
- direct mutation of host configuration
- treating model output as canonical state
- treating FORGE-K simulator code as live daemon authority

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

The G1 shell session must remain opt-in. G3 does not change the disabled-by-default posture. Future NixOS scaffolding should default to:

- `enable = false`
- `mode = "fullscreen-shell"`
- `displayBackend = "wayland"`
- `autoStart = false`
- `coreURL = "http://127.0.0.1:18492"`
- `safeMode = true`

Autologin must not be enabled by default. Existing desktop environments must not be disabled by default. A normal desktop or TTY fallback must remain available. G3 does not introduce a compositor dependency, display-manager replacement, host mutation path, modelruntime mutation path, semantic memory write path, or FORGE-K live authority path.

Rollback expectations:

- disable the FORGE shell session option
- keep existing desktop/session choices available
- keep `/forge` data intact
- keep `forge-core` usable through existing workflows
- use `FORGE_SHELL_BINARY=/path/to/forge_desktop forge-shell-session` when the packaged shell is unavailable
- fall back to `npm run desktop` or the local Tauri release/debug binary paths during package work
- restart the display manager or reboot only when the operator chooses

## Session Scaffolding And Launcher Shape

G1 adds an inert, opt-in NixOS module at `nix/nixos/modules/forge-shell-session.nix`. G2 adds `nix/packages/forge-shell-session.nix` and exposes `packages.forge-shell-session` plus `apps.forge-shell-session`. G3 adds the target package `nix/packages/forge-desktop-shell.nix` and flake output `packages.forge-desktop-shell`; if a stable app is exposed, it should be `apps.forge-desktop-shell`.

The module prepares:

- a session descriptor
- environment variables such as `FORGE_CORE_URL`
- a desktop/session entry placeholder
- runtime directory for shell session state
- safe local core URL wiring

The wrapper prepares:

- `FORGE_SHELL_SESSION_ENABLED=true`
- `FORGE_SHELL_MODE=fullscreen-shell`
- `FORGE_CORE_URL=http://127.0.0.1:18492` unless overridden
- `VITE_FORGE_API_URL=$FORGE_CORE_URL` unless overridden
- `FORGE_SHELL_SAFE_MODE=true`
- `FORGE_SHELL_FULLSCREEN=true`
- explicit false flags for host mutation, direct system control, model mutation, semantic memory writes, and FORGE-K live authority

The placeholder `forge-desktop-shell` command may also accept `FORGE_DESKTOP_SHELL_BINARY=/path/to/forge_desktop` for direct operator testing, but session-level binary override remains `FORGE_SHELL_BINARY`.

It does not add a systemd user service, autologin, compositor dependency, display-manager replacement, or automatic launch path. Compositor choices remain future implementation decisions. Candidate paths include Wayland with cage, Wayland with sway, Wayland with Hyprland, or an X11 fallback if required. G3 does not claim any compositor path is implemented.

## Authority Non-Changes

G1/G2/G3 does not change:

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
