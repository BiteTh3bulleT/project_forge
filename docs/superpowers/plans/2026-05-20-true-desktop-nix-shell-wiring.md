# True Desktop Nix Shell Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every Nix operator desktop entrypoint compose the same VM-safe FORGE desktop shell.

**Architecture:** Keep the normal `forge-desktop-shell` package as the general shell artifact, but introduce explicit operator shell package aliases that bake `renderProfile = "vm-safe"` into the Tauri frontend. Wire `forge-operator-session` through that operator shell in both flake outputs and the overlay so manual `nix run`, downstream `pkgs.*`, and NixOS profiles converge on the same true desktop path.

**Tech Stack:** Nix flakes, NixOS profiles/modules, Tauri desktop shell, static Nix checks, desktop Vitest/build validation.

---

### Task 1: Expose Operator-Specific Shell Packages

**Files:**
- Modify: `flake.nix`
- Modify: `nix/overlays/default.nix`
- Test: `nix/checks/forge-operator-desktop.nix`
- Test: `nix/checks/forge-operator-vm.nix`

- [x] Add `forge-operator-desktop-shell` using `renderProfile = "vm-safe"`.
- [x] Add `forge-operator-shell-session` wired to `forge-operator-desktop-shell`.
- [x] Wire `forge-operator-session` to `forge-operator-shell-session`.
- [x] Keep existing general `forge-desktop-shell` and `forge-shell-session` outputs intact for non-operator development.

### Task 2: Align Legacy VM Test Profile

**Files:**
- Modify: `nix/nixos/profiles/forge-vbox-graphics-test.nix`

- [x] Build the VirtualBox graphics test profile with `renderProfile = "vm-safe"` because it runs inside the same fragile WebKitGTK/VirtualBox graphics environment.

### Task 3: Verify From Available Host

**Commands:**
- `npm run validate:desktop`
- `git diff --check`
- `nix --version`

**Expected:**
- [x] Desktop validation passes.
- [x] Whitespace check passes.
- [x] Windows host reports `nix` unavailable; WSL is also blocked while Virtual Machine Platform is disabled for VirtualBox native VT-x.
