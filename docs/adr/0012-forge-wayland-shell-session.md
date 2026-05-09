# ADR 0012 - FORGE Wayland Shell Session

Status: Accepted

Date: 2026-05-08

## Context

Phase G3.5 made `packages.forge-desktop-shell` a real Linux Tauri Nix package that contains the `forge_desktop` binary and exposes the stable `forge-desktop-shell` command. The next integration step is not more packaging surface; it is a real, opt-in session path that lets an operator select FORGE Shell as a graphical session while preserving NixOS/Linux as the substrate.

The shell session still cannot become host authority. Existing FORGE authority boundaries remain in force: gateway, permissions, lanes, approvals, audit, controllane validation, semantic memory authority, modelruntime governance, FORGE-H boundaries, and FORGE-K simulator/live separation.

## Decision

Phase G4 uses an opt-in Wayland session model.

The target launch flow is:

```text
NixOS login/session selection
  -> FORGE Shell session
  -> lightweight Wayland compositor substrate
  -> forge-shell-session
  -> forge-desktop-shell
  -> local forge-core
```

Cage is the preferred compositor substrate when it is cleanly available through Nixpkgs because the first supported mode is a single full-screen shell. The session must launch `forge-shell-session` inside the compositor rather than launching `forge-desktop-shell` directly, because `forge-shell-session` owns safe shell environment defaults, `FORGE_CORE_URL` wiring, fallback binary selection, and false host-authority flags.

The session remains disabled by default. It must not enable autologin, make FORGE Shell the default session automatically, remove normal desktop sessions, or remove TTY fallback.

## Consequences

- NixOS/Linux remains responsible for boot, hardware, package management, services, display-manager plumbing, and rollback.
- FORGE becomes a selectable graphical shell interface, not a replacement kernel, display manager, package manager, service manager, or modelruntime authority.
- Any `forge-wayland-session` wrapper or generated session descriptor must preserve safe defaults and fail loudly if the compositor or `forge-shell-session` is unavailable.
- Rollback is session/configuration rollback: select a normal desktop or TTY, disable the opt-in shell session option, keep `/forge` data intact, and keep manual `forge-shell-session`/`forge-desktop-shell` launch paths available.
- G4 does not authorize direct `systemctl`, `nixos-rebuild`, package-manager mutation, kernel-module commands, reboot/shutdown, modelruntime load/unload/spawn, semantic memory writes, route/API changes, gateway bypass, or FORGE-K live authority.

## Alternatives Considered

- Launch `forge-desktop-shell` directly from the session descriptor: rejected because it bypasses the existing wrapper boundary for safe defaults and fallback behavior.
- Replace the user's desktop by default: rejected because the shell must remain opt-in and rollback-safe.
- Use a full desktop compositor as the default substrate: deferred until there is a requirement for multi-window compositor behavior. Cage fits the first full-screen shell mode better.
- Let shell code rebuild NixOS or restart services to enable itself: rejected because host mutation must remain outside the shell and under explicit operator-controlled NixOS workflows.
