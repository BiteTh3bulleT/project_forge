# Nix Foundation Status

Date: 2026-05-18
Scope: light Nix foundation, opt-in NixOS host envelope, graphical shell/session lanes, and canonical operator VM status

## Presence and structure

| Item | Status | Notes |
|---|---|---|
| `flake.nix` | present | Defines packages/apps/devShells/checks/formatter. |
| `flake.lock` | present | Lock file committed. |
| Dev shells | present | `default`, `core`, `desktop`, `aios`. |
| `forge-core` package | present | `nix/packages/forge-core.nix` with real `vendorHash`. |
| `forge-shell-session` package/app | present | Manual safe-mode graphical shell session wrapper from G2. |
| `forge-desktop-shell` package/app | present, validated | Phase G3.5 stable command for the desktop shell package surface; the package advertises `passthru.containsTauriBinary = true`, builds the real Linux Tauri app, and exposes both `forge-desktop-shell` and `forge_desktop`. |
| checks | present | `go-tests`, `go-vet`, `js-build`, plus shell-session wrapper checks when exposed by the current flake. |
| tool capsules | README-only | Deferred scaffold. |
| NixOS modules | present, opt-in only | `forge-os`, `forge-services`, `forge-storage`, `forge-host-kernel`, and `forge-shell-session` remain disabled unless imported/enabled by an operator NixOS configuration. |
| profiles | present, opt-in only | VirtualBox graphics test, operator desktop, and canonical operator VM composition are available for explicit NixOS use. |

## Fake-hash status

- `forge-core`: real `vendorHash` (not fake).
- Desktop package capture: Phase G3.5 real Tauri packaging is implemented with real npm/Cargo hashes, Linux WebKit/GTK runtime wrapping, and package checks.
- Any references implying core fake-hash completion are documentation drift and should be treated as stale.

## Commands validated

- `nix --extra-experimental-features 'nix-command flakes' run nixpkgs#prefetch-npm-deps -- package-lock.json` produced the npm dependency hash used by the desktop package.
- `nix --extra-experimental-features 'nix-command flakes' build path:$PWD#forge-desktop-shell --no-link --print-out-paths` succeeds and produces the real Tauri package.
- `nix --extra-experimental-features 'nix-command flakes' build path:$PWD#checks.x86_64-linux.forge-desktop-shell --no-link --print-out-paths` validates the desktop wrapper and binary surface.
- Full flake/core validation is tracked in the phase report for the session that updated this file.

## Current safe usage

Safe now:
- Optional dev shells.
- Flake checks with explicit experimental feature flags.

Not complete yet:
- Clean `forge-core` Nix build path.
- Deeper compositor/window-manager behavior beyond the first G4 fullscreen Wayland session. The implemented G4 path is selectable session -> Cage substrate -> `forge-shell-session` -> packaged `forge-desktop-shell`; it must not autostart or replace the user's desktop.
- Tool capsules/profiles execution integration.

## Phase 01 NixOS host envelope

FORGE-K Online Phase 01 records the current NixOS host envelope as
`PARTIAL / OPT_IN_ONLY / NO_HOST_MUTATION`. The `services.forge-core`
module now keeps localhost binding as the default, requires
`services.forge-core.allowWildcardBind = true` before `0.0.0.0` or `::`
can be used, exports `FORGE_ALLOW_WILDCARD_BIND=false` by default, and
adds stricter systemd sandboxing for the service unit.

This does not make NixOS mandatory, enable autologin, remove TTY/fallback
access, run `nixos-rebuild`, run `systemctl`, mutate host state, load or
unload models, write semantic memory, or make FORGE-K live authority.

## G3.5 graphical shell package commands

```sh
nix build .#forge-desktop-shell
nix run .#forge-desktop-shell
nix build .#forge-shell-session
nix run .#forge-shell-session
```

Local development fallback:

```sh
nix develop .#desktop
npm install
npm run build:desktop
npm -w @forge/desktop run tauri -- build
FORGE_SHELL_BINARY="$PWD/apps/desktop/src-tauri/target/release/forge_desktop" nix run .#forge-shell-session
```

`FORGE_CORE_URL` overrides the governed local core endpoint. `FORGE_SHELL_BINARY`
overrides the session wrapper binary when that wrapper is available. If package
realization is unavailable on a local machine, use `npm run desktop` or the
local Tauri binary directly. None of these paths add compositor integration,
autologin, desktop replacement, service control, host mutation, modelruntime
mutation, semantic memory writes, or FORGE-K live authority.

README and AGENTS now record the G3.5 package boundary and G4 Wayland session boundary.

## G4 Wayland shell session lane

G4 is the opt-in session integration lane, not a new authority plane. NixOS/Linux remains the boot, display-manager, package, service, and rollback substrate. The preferred compositor substrate is Cage when cleanly available through Nixpkgs. The implemented `forge-wayland-session` wrapper and generated Wayland session descriptor launch `forge-shell-session`, preserve safe environment defaults, honor `FORGE_CORE_URL`, and fail loudly if the compositor or shell wrapper is unavailable.

Required safe posture:

- `forge.shellSession.enable = false` by default.
- `forge.shellSession.autoStart = false` by default.
- `forge.shellSession.safeMode = true` by default.
- normal desktop sessions remain selectable.
- TTY fallback remains available.
- no autologin by default.
- no host mutation, direct service control, NixOS rebuild, package-manager mutation, kernel-module command, modelruntime mutation, semantic memory write, or FORGE-K live authority from shell/session code.

## Must wait

Deep Nix/NixOS integration (tool capsules, NixOS modules, Nix-backed autonomy/agent execution, release snapshots) remains gated on authoritative runtime path convergence and stable clean builds.
