# FORGE-OS Operator Desktop VM Boot Evidence

Date recorded: 2026-05-11

Repository commit at evidence recording: `41e5f55`

VM: Oracle VirtualBox `FORGE-OS`

## Evidence Summary

This is a manual VM boot evidence record for the opt-in FORGE operator desktop session.

The VM was not running from this shell when this artifact was written, so no new screenshot file or live command transcript is being invented here. This record preserves the verified operator checkpoint already captured in the runbooks and ties it to an in-repo evidence artifact for follow-up validation.

## Verified Checkpoint

- VM reaches the FORGE operator desktop session.
- Launch path is:
  `TTY -> forge-operator-session -> labwc -> forge-shell-session -> forge-desktop-shell -> forge-core`
- `forge-core` health was verified at `http://127.0.0.1:18492/health`.
- `/api/meta` was verified to report:
  - data dir: `/forge/data`
  - database: `/forge/data/forge.sqlite`
  - workspace: `/forge/workspaces/default`
- VirtualBox display path used VMSVGA with 128 MiB VRAM and 3D acceleration enabled.
- The operator session sets `WEBKIT_DISABLE_DMABUF_RENDERER=1` to avoid the observed VirtualBox Wayland/WebKit dmabuf failure mode.
- The shell window fit/maximize behavior was adjusted after operator-observed right-edge clipping in the VirtualBox viewer.

## Installed NixOS Generation

```text
/nix/store/iy4v4h28zl65x0a5nw64332cvllfxx5v-nixos-system-forge-os-vm-25.11.10470.0c88e1f2bdb9
```

## Next Evidence Capture

On the next live VM run, capture and commit:

- `~/forge-session-logs/forge-operator-session.log`
- `curl -fsS http://127.0.0.1:18492/health`
- `curl -fsS http://127.0.0.1:18492/api/meta`
- a screenshot artifact showing the FORGE shell visible in the VirtualBox viewer
- the exact repo commit mounted in the guest at `/projectforge`

Do not treat this evidence artifact as shell authority. It is operator validation evidence only; VM rebuilds, service checks, and screenshots remain terminal/operator actions.
