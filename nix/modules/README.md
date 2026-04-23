# Nix Modules — Scaffold Only (Phase N1)

This directory is a **placeholder** for future NixOS modules that will
deploy FORGE services declaratively.

## Status in Phase N1

- **No real modules.** Do not import from a NixOS configuration.
- **Not exposed** from `flake.nix`.
- Phase N1 only provides dev shells, basic packages, and checks.

## Planned modules (Phase N4)

- `services.forge-core` — systemd unit, data dir, port, config.
- `services.forge-desktop` — optional kiosk/user-service profile.
- Shared hardening options (capability bounds, seccomp, cgroup limits).

## Prerequisites before implementation

1. FORGE service boundaries are unified (v1/v2 authoritative path
   clearly decided).
2. Config surface (`services/core/internal/config`) is stable enough
   to be lifted into NixOS module options.
3. Audit/permission defaults have been validated for server deployment.

Until then, deploy via the existing `npm run up` / systemd unit path
outside Nix.
