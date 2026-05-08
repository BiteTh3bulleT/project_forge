# Nix Modules — Legacy Placeholder

This directory was the Phase N1 placeholder for future NixOS modules.

## Current Status

Phase N2 added private NixOS host substrate scaffolding under:

- `nix/nixos/modules/forge-os.nix`
- `nix/nixos/modules/forge-services.nix`
- `nix/nixos/modules/forge-storage.nix`
- `nix/nixos/modules/forge-host-kernel.nix`
- `nix/nixos/modules/forge-shell-session.nix`

Those modules are exported from `flake.nix` as `nixosModules.*`.

This directory remains for older notes and future non-NixOS module
planning. Do not import files from `nix/modules/` into a host
configuration.

## Boundaries

Phase N2/G1 modules are scaffolding only. They do not migrate live
authority, add public routes, alter modelruntime behavior, wire FORGE-K
into live authority, replace the desktop, autostart the graphical shell,
or add autonomous host mutation.
