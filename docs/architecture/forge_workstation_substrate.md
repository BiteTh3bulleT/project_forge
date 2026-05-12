# FORGE Workstation Substrate

Status: DESIGN_ONLY / PLANNED / NO_LIVE_AUTHORITY_CHANGE

## Intent

FORGE Workstation is the operator target where NixOS/Linux remains the substrate and FORGE is the graphical operator shell. The substrate owns boot, drivers, hardware, filesystems, login, rollback, and package realization. FORGE owns operator visibility, governed proposals, and AI-OS workflows.

## Layer Model

1. Hardware and firmware.
2. NixOS/Linux kernel, drivers, filesystems, users, networking, rollback.
3. Display manager or TTY launch path.
4. Wayland compositor substrate.
5. FORGE graphical shell.
6. `forge-core` and governed AI-OS services.
7. Modelruntime drivers and external inference backends.

## Required Fallbacks

- TTY login remains available.
- A non-FORGE desktop or recovery path remains available during adoption.
- Safe mode can force CPU-only modelruntime posture.
- Failed graphical shell startup must not block operator recovery.
- Nix generation rollback remains an operator/substrate action until a governed adapter exists.

## Forbidden Workstation Behavior

- No autologin by default.
- No default desktop replacement without explicit operator selection.
- No direct shell-to-host mutation.
- No restart, shutdown, rebuild, package-manager, kernel/module, cleanup, or model load/unload buttons outside governed proposal paths.
- No semantic memory writes from workstation diagnostics.
- No FORGE-K live authority expansion.

## Status Labels

- Current NixOS shell/session support: PARTIAL / OPT_IN.
- Full FORGE Workstation target: PLANNED.
- Governed Nix mutation proposal system: FUTURE.
- System cockpit mutation controls: FUTURE and approval-gated only.

## Next Implementation Gate

Before workstation mutation is implemented, FORGE needs a durable proposal object, review state, sandbox build evidence, rollback evidence, explicit operator approval, and an adapter that is separate from the Tauri shell.
