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
