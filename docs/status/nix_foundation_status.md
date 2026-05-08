# Nix Foundation Status (Phase N1 truth check)

Date: 2026-05-08
Scope: light Nix foundation plus G3.5 graphical shell package status

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
| NixOS modules | README-only | Deferred scaffold. |
| profiles | README-only | Deferred scaffold. |

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
- Compositor/session integration for using FORGE as a full desktop shell. The package exists, but it does not autostart or replace the user's desktop.
- Tool capsules/NixOS modules/profiles execution integration.

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

README and AGENTS now record the G3.5 package boundary.

## Must wait

Deep Nix/NixOS integration (tool capsules, NixOS modules, Nix-backed autonomy/agent execution, release snapshots) remains gated on authoritative runtime path convergence and stable clean builds.
