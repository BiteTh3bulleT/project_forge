# ADR 0013 - FORGE G6 Operator Desktop

Status: Accepted

Date: 2026-05-10

## Context

Phase G4/G5 proved that FORGE can run as a graphical shell surface in a minimal NixOS VM through the `forge-wayland-session -> Cage -> forge-shell-session -> forge-desktop-shell` path.

That path is intentionally fullscreen and single-application. It is useful as a stable rollback shell, but it cannot serve the operator desktop target because operators need terminal windows, file managers, browser windows, model tools, and installer dialogs to open beside or over FORGE.

Installing GNOME or KDE would solve app-window usability quickly, but it would make FORGE an application inside another desktop shell. The target architecture is different: NixOS is the substrate, FORGE is the operator desktop, and the compositor is infrastructure below FORGE.

## Decision

Phase G6 adds a separate opt-in operator desktop session:

```text
NixOS login / TTY
  -> forge-operator-session
  -> labwc
  -> forge-shell-session
  -> forge-desktop-shell
  -> local forge-core
```

`labwc` is the first compositor substrate because it provides normal Wayland window management without bringing in a full desktop environment. FORGE remains the primary desktop surface. External programs are operator tools that run in normal windows, not authority surfaces.

The existing Cage session remains installed and supported as rollback.

## Consequences

- NixOS remains responsible for boot, drivers, packages, services, display plumbing, and rollback.
- FORGE remains the visible operator desktop surface.
- The compositor becomes a narrow window-management substrate, not the product shell.
- Terminal and file-manager access can exist without installing a full desktop environment.
- G6 does not enable autologin, force a default graphical session, remove TTY fallback, or remove the Cage rollback path.
- G6 does not authorize direct `systemctl`, `nixos-rebuild`, package-manager mutation, kernel/module changes, reboot/shutdown, modelruntime load/unload/spawn, semantic memory writes, route/API changes, gateway bypass, or FORGE-K live authority.

## Alternatives Considered

- Install GNOME/KDE: rejected as the target path because FORGE would become an app inside a full external desktop shell.
- Extend Cage with app launch hacks: rejected because Cage is intentionally single-app and does not fit normal operator window management.
- Replace NixOS desktop/session plumbing entirely: rejected because NixOS remains the substrate and rollback authority.
