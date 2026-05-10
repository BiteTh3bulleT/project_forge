# FORGE G6 Operator Desktop Design

Date: 2026-05-10
Status: Approved design direction

## Purpose

G6 turns the VM shell from a single fullscreen FORGE surface into a real operator desktop where FORGE remains the desktop authority surface and NixOS remains the operating substrate.

This is not a GNOME/KDE conversion. Full desktop environments solve app windows quickly, but they make FORGE an application inside another shell. G6 keeps the target architecture intact: NixOS provides boot, hardware, services, packages, rollback, graphics plumbing, and process isolation; FORGE provides the visible command surface, workspace model, approvals, jobs, memory, runtime inspection, and future governed autonomy.

## Design Principles

- FORGE is the desktop surface.
- NixOS is the substrate below FORGE.
- The compositor is infrastructure, not the product shell.
- External app launch is explicit, deterministic, and operator-initiated.
- Model output and FORGE-K proposal output must not directly launch programs, mutate host state, load models, or write canonical memory.
- The existing Cage fullscreen session remains available as rollback.
- G6 must improve operator usability without making FORGE-K live authority.

## Current State

The current VM graphics path is:

```text
NixOS TTY/manual launch
  -> forge-wayland-session
  -> Cage
  -> forge-shell-session
  -> forge-desktop-shell
  -> local forge-core
```

Cage is correct for Phase G4/G5 fullscreen shell testing, but it is intentionally a single-app compositor. It cannot become the long-term operator desktop because terminal windows, file managers, browser windows, model tools, and installer dialogs need normal window management.

## Target State

G6 introduces a second, opt-in session path:

```text
NixOS login / TTY
  -> forge-operator-session
  -> lightweight Wayland compositor
  -> FORGE desktop shell as the primary surface
  -> allowlisted external apps on demand
```

The recommended compositor substrate is `labwc` unless implementation evidence shows a better lightweight Wayland compositor in Nixpkgs for this VM. `labwc` provides real window management without bringing in a full desktop environment.

## Operator Apps

The first operator app set should be small and practical:

- terminal: `foot`
- file explorer: `pcmanfm` or `thunar`, selected by Nixpkgs fit and Wayland behavior
- desktop open helpers: `xdg-utils`
- portals: `xdg-desktop-portal` plus a GTK or wlroots-compatible portal backend
- clipboard helpers if needed after the base session works

Browser installation is allowed after the base terminal/file explorer path is verified. It should not block G6 because browser packaging and rendering may add VM performance cost.

## Launch Boundary

G6 adds a deterministic desktop app launcher boundary.

Allowed:

- FORGE shows operator buttons for known desktop tools.
- FORGE launches only allowlisted desktop app ids.
- Launch records include app id, executable path selected by the Nix package, timestamp, shell session id when available, and result status.
- The human operator can use the terminal to install or configure Ollama and models through normal NixOS workflows.

Not allowed:

- arbitrary shell command launch from the FORGE UI
- model-generated command execution from the UI
- FORGE-K simulator services becoming live authority
- bypassing gateway, permissions, approvals, audit, modelruntime governance, or Control Lane commit boundaries
- autologin, forced default session replacement, or removal of TTY fallback

## FORGE-K Boundary

G6 prepares the desktop for later FORGE-K operational integration, but it does not complete that integration.

FORGE-K may be surfaced only as:

- shadow diagnostics
- proposal summaries
- validation reports
- operator-visible status surfaces

FORGE-K must not own live daemon authority, execute tools, mutate canonical truth, compile live context, run retrieval, call modelruntime, load/unload models, or directly launch desktop programs in G6.

Future FORGE-K live work must be a separate phase with explicit integration gates, tests, documentation updates, and rollback.

## NixOS Profile Changes

G6 should add a new profile rather than mutate the G5 VirtualBox graphics test profile into a different role.

Expected new surfaces:

- `nix/packages/forge-operator-session.nix`
- `nix/checks/forge-operator-session.nix`
- `nix/nixos/profiles/forge-operator-desktop.nix`
- `docs/runbooks/forge_operator_desktop_vm.md`
- an ADR for the G6 compositor/session decision

The existing `forge-wayland-session` and `forge-vbox-graphics-test` profile remain rollback paths.

## Desktop UI Changes

FORGE desktop should gain explicit operator entries for:

- Terminal
- Files
- Optional browser when installed
- Shell/session status

These entries should use the existing shell/dock/start-menu patterns instead of adding a separate desktop metaphor. The app launcher UI is a command surface, not a general command prompt.

## Error Handling

Launcher failures must be visible and bounded:

- missing executable: show unavailable state
- launch failure: show result status and stderr summary when safe
- repeated failure: keep FORGE running and do not restart the compositor
- compositor/session failure: operator can return to TTY or use the old Cage session

## Testing Expectations

Minimum verification before G6 is considered working:

- Nix package/check for `forge-operator-session` passes.
- New NixOS profile evaluates.
- Existing Cage wrapper checks still pass.
- VM boots with TTY fallback.
- `forge-operator-session` starts the compositor and FORGE shell.
- `foot` opens as a separate window.
- file manager opens as a separate window.
- at least one external app can overlap or sit beside FORGE.
- FORGE core health remains reachable at `127.0.0.1:18492`.
- FORGE-K remains non-authoritative and disabled from live mutation.

## Rollback

Rollback remains simple:

- use TTY login
- launch `forge-wayland-session` for the prior fullscreen Cage shell
- disable or stop using the G6 profile
- keep `/forge`, `/mnt/vmdisk`, and project shared folder data intact

No G6 wrapper may remove TTY access, enable autologin, or force the operator desktop as the only session.
