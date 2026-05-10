# VirtualBox FORGE Shell Graphics Test

Status: Phase G5 test profile and runbook.

Scope: TEST PROFILE ONLY / OPT-IN ONLY / VIRTUALBOX/MINIMAL NIXOS GRAPHICS BRING-UP.

This runbook is for a minimal NixOS VM in Oracle VirtualBox where no full graphical desktop environment is installed. The goal is to launch FORGE Shell manually from a TTY:

```text
TTY login
-> minimal graphics/session environment
-> forge-wayland-session
-> Cage
-> forge-shell-session
-> packaged forge-desktop-shell
-> local forge-core
```

FORGE becomes the visible graphical shell. NixOS/Linux remains the graphics, package, service, hardware, and boot substrate.

## VM Checklist

VirtualBox system settings:

- EFI: enabled
- RAM: 8 GB minimum
- CPU: 4 minimum

VirtualBox display settings:

- Graphics Controller: VMSVGA
- Video Memory: 128 MB
- 3D Acceleration: try off first; enable only if needed

NixOS guest expectations:

- minimal NixOS install is bootable
- TTY login works
- repository checkout is available
- Nix flakes can be used by the operator
- `forge-core` can run locally at `http://127.0.0.1:18492`

## NixOS Profile Import Example

The G5 profile is opt-in. Import it explicitly from the repository checkout:

```nix
{ ... }:

{
  imports = [
    /path/to/project_forge/nix/nixos/profiles/forge-vbox-graphics-test.nix
  ];
}
```

If using the flake module output from a local flake-based host configuration, import:

```nix
inputs.project-forge.nixosModules.forge-vbox-graphics-test
```

The profile keeps automatic login disabled and preserves TTY fallback. It does not install a full desktop environment or make FORGE Shell the system default unless the operator explicitly imports the profile and launches the session.

## Manual TTY Launch

From a TTY login:

```bash
cd /path/to/project_forge
nix build .#forge-desktop-shell
nix build .#forge-shell-session
nix build .#forge-wayland-session
nix run .#forge-wayland-session
```

If Nix experimental features are not enabled globally:

```bash
nix --extra-experimental-features 'nix-command flakes' build .#forge-desktop-shell
nix --extra-experimental-features 'nix-command flakes' build .#forge-shell-session
nix --extra-experimental-features 'nix-command flakes' build .#forge-wayland-session
nix --extra-experimental-features 'nix-command flakes' run .#forge-wayland-session
```

If the profile has been imported and activated through the operator's normal NixOS workflow, the command should also be available directly:

```bash
forge-wayland-session
```

## `forge-core` Startup Requirement

The shell expects a local governed core endpoint. Start `forge-core` before launching the shell:

```bash
cd /path/to/project_forge
npm run core
```

Or use the existing service path if the local NixOS host configuration already manages `forge-core`.

Override the endpoint only when needed:

```bash
FORGE_CORE_URL=http://127.0.0.1:18492 nix run .#forge-wayland-session
```

## Missing Display Errors

If `DISPLAY` is missing, that is expected for the manual TTY path. `forge-wayland-session` starts Cage and creates the Wayland session.

If `XDG_RUNTIME_DIR` is missing or invalid, log in through a normal NixOS TTY session for the target user. Avoid launching as root. A user session should provide the runtime directory.

## Cage And Wayland Failures

If Cage exits immediately:

- confirm the profile or package build includes `cage`
- confirm VirtualBox graphics controller is VMSVGA
- try Video Memory at 128 MB
- try 3D Acceleration off first
- if rendering fails, retry with 3D Acceleration enabled

The wrapper path must remain:

```text
forge-wayland-session -> Cage -> forge-shell-session -> forge-desktop-shell
```

Do not bypass `forge-shell-session`; it owns safe shell environment defaults.

## Tauri And WebKit Runtime Failures

If the shell starts but the Tauri window fails:

- build the desktop package directly with `nix build .#forge-desktop-shell`
- verify `result/bin/forge-desktop-shell` exists
- verify the local VM has enough RAM
- run the desktop development path only as a fallback: `npm run desktop`

The packaged shell wraps the Linux WebKit/GTK runtime dependencies. Do not install a full graphical desktop environment just to satisfy this test profile.

## VirtualBox Graphics Issues

Recommended first settings:

- Graphics Controller: VMSVGA
- Video Memory: 128 MB
- 3D Acceleration: off

If the compositor cannot create a surface, enable 3D Acceleration and retry. If that makes rendering worse, turn it off again and use the TTY rollback path.

## Return To TTY

To leave the shell:

- use the shell's exit control if available
- otherwise switch to a TTY using the VirtualBox host key plus the guest TTY key sequence
- log in on the TTY and inspect the failed command output

TTY fallback must remain available. The G5 profile does not remove it.

## Disable The Test Profile

To disable the test profile, remove the profile import from the local NixOS configuration.

Then use the operator's normal NixOS rebuild workflow outside FORGE wrappers.

The wrappers do not run host configuration commands. They do not start or stop services, rebuild NixOS, load kernel modules, reboot, or shut down the host.

## Roll Back Using NixOS Generations

If the imported profile prevents a usable graphical launch, select an older NixOS generation from the boot menu or boot to a TTY and use the operator's normal rollback workflow. This is the NixOS generations rollback path.

Rollback should not delete `/forge` data, model files, workspace files, or semantic memory stores.

## What This Profile Does Not Do

- does not install a full desktop environment
- does not enable automatic login
- does not remove TTY fallback
- does not make FORGE the compositor
- does not implement a custom compositor
- does not expose remote graphics by default
- does not mutate host state from wrappers
- does not load or unload models
- does not write semantic memory directly
- does not change gateway, permissions, lanes, audit, controllane, memory, modelruntime, FORGE-H, or FORGE-K authority
