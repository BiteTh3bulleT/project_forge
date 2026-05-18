# NixOS Workstation Substrate

## Purpose

FORGE-K online must run inside a NixOS-defined host envelope. The daemon should not improvise host policy at runtime.

## Required substrate pieces

- `services.forge-core` NixOS module
- `/forge` filesystem layout
- systemd hardening for `forge-core`
- tool execution sandbox profiles
- safe-mode/recovery profile
- VM smoke tests
- rollback documentation

## Files to inspect or create

- `nix/nixos/modules/forge-core.nix`
- `nix/nixos/modules/forge-storage.nix`
- `nix/nixos/modules/forge-shell-session.nix`
- `nix/nixos/profiles/forge-operator-desktop.nix`
- `nix/checks/*`
- `docs/architecture/nix_substrate.md`
- `docs/runbooks/no_gpu_boot_and_recovery.md`

## What not to do

- Do not make FORGE-K run `nixos-rebuild` directly.
- Do not remove fallback desktop/TTY access.
- Do not require GPU for boot.
- Do not make host mutation automatic.
