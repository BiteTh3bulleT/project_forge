# Nix Foundation Status (Phase N1 truth check)

Date: 2026-04-21
Scope: light Nix foundation only

## Presence and structure

| Item | Status | Notes |
|---|---|---|
| `flake.nix` | present | Defines packages/apps/devShells/checks/formatter. |
| `flake.lock` | present | Lock file committed. |
| Dev shells | present | `default`, `core`, `desktop`, `aios`. |
| `forge-core` package | present | `nix/packages/forge-core.nix` with real `vendorHash`. |
| `forge-desktop` package | deferred | Not yet exposed as a flake package. |
| checks | present | `go-tests`, `go-vet`, `js-build`. |
| tool capsules | README-only | Deferred scaffold. |
| NixOS modules | README-only | Deferred scaffold. |
| profiles | README-only | Deferred scaffold. |

## Fake-hash status

- `forge-core`: real `vendorHash` (not fake).
- Desktop package capture: deferred.
- Any references implying core fake-hash completion are documentation drift and should be treated as stale.

## Commands validated

- `nix flake check` -> fails unless experimental features are enabled.
- `nix --extra-experimental-features 'nix-command flakes' flake check` -> fail (daemon socket unavailable in this environment).
- `nix --extra-experimental-features 'nix-command flakes' develop --command true` -> fail (daemon socket unavailable).
- `nix --extra-experimental-features 'nix-command flakes' build .#forge-core` -> fail.

## Current safe usage

Safe now:
- Optional dev shells.
- Flake checks with explicit experimental feature flags.

Not complete yet:
- Clean `forge-core` Nix build path.
- Desktop Nix package.
- Tool capsules/NixOS modules/profiles execution integration.

## Must wait

Deep Nix/NixOS integration (tool capsules, NixOS modules, Nix-backed autonomy/agent execution, release snapshots) remains gated on authoritative runtime path convergence and stable clean builds.
