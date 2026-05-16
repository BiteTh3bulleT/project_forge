# FORGE Native Desktop Runtime

Date: 2026-05-15
Status: Initial VM implementation in progress

## Goal

Make FORGE-OS present as a real desktop operating system from boot through login
to the operator desktop:

```text
power on
-> FORGE-OS Runtime boot splash
-> graphical password login
-> FORGE native desktop session
```

The operator should no longer normally see a TTY and manually type
`forge-operator-session`. TTY access remains available as recovery, not the
default user experience.

## Product Intent

FORGE-OS is becoming a native desktop runtime, not a Linux distro that happens
to launch a FORGE window. NixOS/Linux remains the substrate for boot, hardware,
services, rollback, display plumbing, and recovery. FORGE is the visible desktop
environment and operator surface.

This pass is intentionally minimal-branding-first. It should make the system
feel structurally real before spending time on visual polish.

## Approaches Considered

### 1. Plymouth + greetd/regreet + FORGE session

Use Plymouth for the FORGE-OS boot image, greetd for native session handoff,
and the existing `forge-operator-session` as the default selected session.
The first VM pass uses the packaged FORGE desktop as the visible login screen;
PAM-backed greeter integration remains the hardening target.

Benefits:

- Wayland-native and lightweight.
- Good fit for VM and future appliance-style runtime images.
- Avoids pulling in a full desktop environment.
- Keeps FORGE as the primary desktop surface.

Costs:

- Login theme starts minimal.
- Slightly more NixOS plumbing than a manual TTY session.

### 2. Plymouth + SDDM + FORGE session

Use a more conventional graphical display manager with stronger theme support.

Benefits:

- Faster path to polished login visuals.
- Familiar display-manager behavior.

Costs:

- Heavier than needed.
- Can make the system feel like FORGE is hosted inside a conventional desktop
  stack rather than being the runtime surface.

### 3. Keep TTY login and improve launcher ergonomics

Keep the current manual boot path and add helper prompts.

Benefits:

- Lowest implementation risk.
- Strong recovery posture.

Costs:

- Does not meet the native desktop goal.
- Still feels like an app/session launcher, not FORGE-OS.

## Decision

Implement approach 1.

The native desktop path should be:

```text
firmware / bootloader
-> Plymouth FORGE-OS Runtime splash
-> greetd session handoff
-> forge-operator Wayland session
-> labwc compositor
-> forge-shell-session
-> packaged forge-desktop-shell
-> FORGE shell loading screen
-> FORGE login screen
-> empty FORGE desktop
-> local forge-core
```

The first normal interactive screen should be FORGE-branded. The operator must
be able to leave FORGE running but locked. The initial VM implementation's
FORGE login is a local UX gate; production hardening should replace or bind it
to a PAM-backed greeter.

## Phase 1 Scope

Add a NixOS native desktop profile/module layer that can be imported by the
canonical operator VM target.

Required behavior:

- show a minimal FORGE-OS Runtime boot splash
- show a FORGE shell loading screen before the login form
- start a FORGE login screen by default
- require password login
- make the FORGE operator session the default login session
- clear restored in-shell windows on login so the operator lands on an empty desktop
- preserve TTY fallback and Nix generation rollback
- keep SSH disabled by default unless the local VM config explicitly enables it
- keep `forge-core` bound to localhost by default
- keep FORGE shell safe-mode flags visible and unchanged
- keep the VM resource profile conservative

Out of scope for this pass:

- polished login theme
- autologin
- replacing NixOS rollback or TTY recovery
- service-control buttons in FORGE
- shell-side `systemctl` or `nixos-rebuild`
- shell-side model load/unload
- direct semantic memory writes
- FORGE-K live authority migration
- arbitrary host mutation from the desktop shell

## Desktop Experience Target

Phase 1 should feel like the first real FORGE-OS desktop boot, not final polish.

Expected normal operator flow:

1. Start VM or machine.
2. See FORGE-OS Runtime splash during boot.
3. See the FORGE shell loading screen.
4. See graphical password login.
5. Log in as `operator`.
6. Land directly on an empty FORGE desktop.
7. Launch terminal/files/toolbelt apps from FORGE.
8. Lock/logout from the desktop path without losing recovery access.

Longer-term Windows-like polish belongs to later phases:

- polished login theme
- taskbar/status area refinement
- desktop wallpaper and icons/widgets
- quick settings
- notifications
- stronger compositor/window lifecycle integration
- multi-monitor window movement and persistence
- session lock UX polish

## Authority Boundary

The native desktop path changes presentation and session ownership only. It does
not change FORGE authority.

NixOS remains responsible for:

- boot
- display manager/login
- service lifecycle
- package/runtime activation
- rollback
- emergency TTY access

FORGE remains responsible for:

- desktop/operator surface
- governed visibility
- governed chat/modelruntime use
- governed tool proposals and approvals

The shell must not gain direct authority to mutate host services, rebuild NixOS,
load/unload models, write semantic memory, or bypass gateway/permissions/lanes,
audit, controllane, memory, modelruntime, FORGE-H, or FORGE-K boundaries.

## Implementation Units

1. Native desktop NixOS module/profile
   - Add a focused profile or module for boot splash and graphical login.
   - Compose it with the existing operator desktop profile.
   - Keep the existing manual operator desktop profile usable as a fallback.

2. Display/login plumbing
   - Configure Plymouth with minimal FORGE branding.
   - Configure greetd/regreet or the selected lightweight greeter.
   - Select the `forge-operator` Wayland session by default.
   - Keep password login required.

3. Session safety checks
   - Extend static Nix checks to assert no autologin.
   - Assert TTY fallback remains available.
   - Assert FORGE session is present in the login session list.
   - Assert forbidden host mutation strings are not introduced.

4. Runbooks and status docs
   - Update operator desktop VM docs from manual TTY launch to native desktop
     boot/login flow.
   - Document recovery path: TTY, rollback, manual shell launch.
   - Document what is intentionally not in scope.

5. VM verification
   - Build the NixOS VM target.
   - Boot the VM.
   - Verify graphical login appears.
   - Log in as operator.
   - Verify FORGE desktop session starts.
   - Verify `forge-core`, modelruntime health, and local Ollama status.

## Test Expectations

Static checks:

- native desktop profile imports expected session/profile pieces
- autologin is disabled
- graphical login is enabled
- FORGE session descriptor exists
- TTY fallback is not disabled
- forbidden host mutation commands are absent from shell/session wrappers

Build checks:

- `nix build .#nixosConfigurations.forge-operator-vm.config.system.build.vm`
- relevant flake checks for shell/session/operator desktop

Manual VM evidence:

- boot reaches the FORGE login screen
- login reaches FORGE desktop
- terminal/toolbelt still launch
- modelruntime remains governed and local
- recovery notes are current

## Acceptance Criteria

- FORGE-OS boots into a FORGE-branded runtime splash.
- The first normal interactive screen is a FORGE login screen, not a TTY.
- Login requires a password in the VM UX gate; PAM-backed validation remains a
  hardening requirement before treating it as a security boundary.
- Successful login starts the FORGE native desktop session.
- TTY fallback remains available.
- Existing manual launch path remains documented as recovery.
- No display-manager autologin is introduced.
- No shell-side host mutation path is introduced.
- No service-control, Nix rebuild, model load/unload, semantic memory write, or
  FORGE-K live authority bypass is introduced.
- VM build and relevant checks pass.
