# FORGE Graphical Shell

Phase G1 defines FORGE as the graphical shell session for a NixOS-based FORGE-OS machine. Phase G2 adds a manually launchable Nix wrapper for that shell session.

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

## G1/G2 Scope

G1 defines the shell session contract and an inert NixOS module foundation. G2 adds a `forge-shell-session` package and flake app that set safe shell-session environment variables and launch an existing local Tauri `forge_desktop` binary when one is available.

The only G1/G2 launch mode is `fullscreen-shell`. Future modes may include:

- `kiosk`
- `compositor-integrated`
- `remote-operator`
- `multi-monitor-shell`

Those modes are not implemented or promised by G1/G2.

G1/G2 does not:

- edit Go services
- replace the user's desktop
- enable autologin
- add compositor dependencies
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

G1/G2 does not need to implement all of these. It defines the contract that future work must respect and provides the first manual launcher.

## Existing Desktop App Mapping

The repository already has a desktop/Tauri development path documented through root npm scripts such as `npm run desktop`, `npm run dev`, and `npm -w @forge/desktop run build`.

G2 identifies the current shell implementation as:

- `apps/desktop`, workspace package `@forge/desktop`
- `apps/desktop/src-tauri`, Rust package `forge_desktop`
- `apps/desktop/src-tauri/tauri.conf.json`
- binary paths `apps/desktop/src-tauri/target/release/forge_desktop` and `apps/desktop/src-tauri/target/debug/forge_desktop`

The Nix package in G2 is a wrapper around those existing binary paths. It does not yet build the Tauri application inside Nix. If the binary is unavailable, it fails loudly with exact build/run instructions.

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

The G1 shell session must remain opt-in. Future NixOS scaffolding should default to:

- `enable = false`
- `mode = "fullscreen-shell"`
- `displayBackend = "wayland"`
- `autoStart = false`
- `coreURL = "http://127.0.0.1:18492"`
- `safeMode = true`

Autologin must not be enabled by default. Existing desktop environments must not be disabled by default. A normal desktop or TTY fallback must remain available.

Rollback expectations:

- disable the FORGE shell session option
- keep existing desktop/session choices available
- keep `/forge` data intact
- keep `forge-core` usable through existing workflows
- restart the display manager or reboot only when the operator chooses

## Session Scaffolding And Launcher Shape

G1 adds an inert, opt-in NixOS module at `nix/nixos/modules/forge-shell-session.nix`. G2 adds `nix/packages/forge-shell-session.nix` and exposes `packages.forge-shell-session` plus `apps.forge-shell-session`.

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
- `FORGE_SHELL_SAFE_MODE=true`
- `FORGE_SHELL_FULLSCREEN=true`
- explicit false flags for host mutation, direct system control, model mutation, semantic memory writes, and FORGE-K live authority

It does not add a systemd user service, autologin, compositor dependency, display-manager replacement, or automatic launch path. Compositor choices remain future implementation decisions. Candidate paths include Wayland with cage, Wayland with sway, Wayland with Hyprland, or an X11 fallback if required. G2 does not claim any compositor path is implemented.

## Authority Non-Changes

G1 does not change:

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
