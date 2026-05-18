# Windows WSL Nix Install Status

Status date: 2026-05-18.

## Scope

Host setup record for the Windows workstation development path. This records the local Nix installation used to validate FORGE flake outputs from WSL; it is not a FORGE-K live authority change and does not alter daemon runtime behavior.

## Installed Host Surface

- WSL version: 2.6.3.0.
- Development distro: Ubuntu-24.04, WSL2.
- Default Ubuntu user: `rasho`.
- Init: systemd enabled in `/etc/wsl.conf`.
- Nix install: upstream multi-user daemon install from `https://nixos.org/nix/install`.
- Nix version: 2.34.7.
- Nix daemon socket: active.
- `/etc/nix/nix.conf` enables `nix-command` and `flakes`.

## Verification Evidence

Commands run from PowerShell through `wsl -d Ubuntu-24.04`:

- `nix --version` -> `nix (Nix) 2.34.7`.
- `systemctl is-active nix-daemon.socket` -> `active`.
- `cd /mnt/e/dev/imrobman-dev/project_forge && nix flake metadata --no-write-lock-file` -> flake evaluated.
- `cd /mnt/e/dev/imrobman-dev/project_forge && nix develop --refresh .#core -c go version` -> `go version go1.26.1 linux/amd64`.
- `cd /mnt/e/dev/imrobman-dev/project_forge && nix build .#forge-core --no-link --print-build-logs` -> built `forge-core-0.1.0`.

## Notes

The flake reports a dirty Git tree because the original `FORGE-K_Online_Push/` source directory remains untracked by policy as source/provenance. It is not part of the Nix install.
