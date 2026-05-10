# FORGE NixOS Profiles

These profiles are opt-in NixOS composition fragments for targeted operator
bring-up. Importing a profile is an explicit local host-configuration choice;
none of these profiles are enabled by default.

## Available Profiles

### `forge-vbox-graphics-test.nix`

Scope: TEST PROFILE ONLY / OPT-IN ONLY / VIRTUALBOX/MINIMAL NIXOS GRAPHICS BRING-UP.

This profile adds the minimal graphics and session substrate needed to launch
FORGE Shell from a TTY in a minimal VirtualBox NixOS VM:

```text
TTY login
-> forge-wayland-session
-> Cage
-> forge-shell-session
-> packaged forge-desktop-shell
-> local forge-core
```

It does not install a full desktop environment, enable automatic login, remove
TTY fallback, expose remote graphics, run host-control commands from wrappers,
load or unload models, write semantic memory, or make FORGE-K live authority.

See `docs/operations/virtualbox_forge_shell_test.md` for the operator runbook.

### `forge-operator-desktop.nix`

Scope: OPERATOR DESKTOP PROFILE ONLY / OPT-IN ONLY / FORGE G6 WAYLAND WINDOW SUBSTRATE.

This profile adds the lightweight compositor and operator tools needed to run
FORGE as the primary desktop surface while allowing normal app windows:

```text
TTY login
-> forge-operator-session
-> labwc
-> forge-shell-session
-> packaged forge-desktop-shell
-> local forge-core
```

It installs terminal and file-manager support for operator-owned setup work,
keeps the previous Cage fullscreen session as rollback, keeps TTY fallback, and
does not install a full desktop environment, enable automatic login, run
host-control commands from wrappers, load or unload models, write semantic
memory, or make FORGE-K live authority.

See `docs/runbooks/forge_operator_desktop_vm.md` for the G6 operator runbook.
