# FORGE Operator VM Live-Start Evidence

Date recorded: 2026-05-18

Scope: VirtualBox/operator desktop render-safety evidence.

## Summary

This evidence directory captures the live-start render investigation that
preceded the `vm-safe` default for the operator shell.

Observed pre-fix behavior:

- The FORGE operator desktop reached the live shell surface in VirtualBox.
- WebKitGTK remained on the hot path for the Tauri shell even after the dmabuf
  fallback was set with `WEBKIT_DISABLE_DMABUF_RENDERER=1`.
- The shell was visually usable, but the VirtualBox/WebKit path produced high
  CPU pressure and sluggish repaint behavior before effects were forced down.
- `render-diagnosis-before-effects-off.png` records the pre-fix visual state
  before the low-cost effects profile was applied.

No FORGE-K live-authority change is implied by this evidence. The render
profile is a shell/frontend performance knob only; it does not mutate memory,
gateway authority, modelruntime authority, approvals, or journal state.

## Fix Captured By This Slice

The Nix/operator path now defaults the render profile to:

```text
FORGE_RENDER_PROFILE=vm-safe
VITE_FORGE_RENDER_PROFILE=vm-safe
```

The default is wired through the operator desktop profile, the operator session
launcher, and the canonical operator VM environment record. The desktop shell
package still accepts a `renderProfile` argument so non-VM/native profiles can
build with the normal `default` profile.

## Disabling VM-Safe For Diagnosis

For a one-off manual launch from a TTY:

```bash
FORGE_RENDER_PROFILE=default VITE_FORGE_RENDER_PROFILE=default forge-operator-session
```

For a persistent operator preference inside the web shell, set
`forge.render.profile` to `default` in local storage, then restart the shell.

For a Nix profile intended for non-VirtualBox or bare-metal use, instantiate
`nix/packages/forge-desktop-shell.nix` with:

```nix
renderProfile = "default";
```

Keep this override diagnostic or hardware-specific. The VirtualBox/operator VM
default remains `vm-safe` until live evidence shows the heavier WebKit effects
path is stable.
