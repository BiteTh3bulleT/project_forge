# Nix Profiles — Scaffold Only (Phase N1)

This directory is a **placeholder** for future FORGE composition
profiles (developer / secure / demo / ci).

## Status in Phase N1

- **No real profiles.** Empty scaffold.
- **Not exposed** from `flake.nix`.

## Planned profiles (Phase N4+)

- **developer** — permissive shell with all tooling.
- **secure** — minimal closure, no network tools, stricter defaults.
- **demo** — bundled sample workspace, deterministic data dir.
- **ci** — headless build/test environment for automation.

## Prerequisites

Profiles depend on NixOS modules (`nix/modules/`) and tool capsules
(`nix/tool-capsules/`). Both are deferred — so are profiles.
