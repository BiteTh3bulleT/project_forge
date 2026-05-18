# AI Native Workstation Render-Safe Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the FORGE operator VM responsive enough for daily workstation use while preserving the Tauri/NixOS shell path.

**Architecture:** Add a runtime render profile, default the VirtualBox/operator shell to `vm-safe`, and make the frontend use that profile to disable expensive WebKitGTK compositing work. Keep the fix narrow: no authority changes, no modelruntime changes, no FORGE-K live authority.

**Tech Stack:** Tauri 2, React, Zustand, CSS, Nix shell/session wrappers.

---

### Task 1: Runtime Render Profile

**Files:**
- Create: `apps/desktop/src/lib/renderProfile.ts`
- Modify: `apps/desktop/src/App.tsx`
- Modify: `apps/desktop/src/stores/uiStore.ts`
- Test: `apps/desktop/src/lib/renderProfile.test.ts`

- [x] Add a typed render profile helper that accepts `default` and `vm-safe`.
- [x] Make VM-safe force initial effects to `off` unless the operator explicitly stored a valid preference.
- [x] Expose the render profile on `document.documentElement.dataset.renderProfile`.
- [x] Verify with Vitest.

### Task 2: VM-Safe CSS

**Files:**
- Modify: `apps/desktop/src/styles/forge-base.css`
- Modify: `apps/desktop/src/styles/forge-os-shell.css`
- Modify: `apps/desktop/src/styles/forge-os-start-menu.css`
- Modify: `apps/desktop/src/styles/forge-os-window-login.css`

- [x] Under `html[data-render-profile="vm-safe"]`, remove backdrop filters, image filters, masks, animated pseudo backgrounds, heavy shadows, wallpaper grid/glow layers, and expensive transitions.
- [x] Keep the desktop visually FORGE, just flatter and cheaper to paint.
- [x] Verify with desktop build.

### Task 3: Stop Duplicate Route Rendering

**Files:**
- Modify: `apps/desktop/src/layout/AppShell.tsx`
- Test: `apps/desktop/src/layout/AppShell.test.tsx`

- [x] Hide the router sink when the route is already represented as an in-shell window.
- [x] Add a test that an active shell tool does not leave duplicate routed content visible.

### Task 4: Reduce Pointer-Move Persistence

**Files:**
- Modify: `apps/desktop/src/stores/desktopWindowStore.ts`
- Test: `apps/desktop/src/stores/desktopWindowStore.test.ts`

- [x] Add non-persisting move/resize update helpers for live drag.
- [x] Persist geometry once on drag/resize commit.
- [x] Add tests proving pointer movement does not write localStorage on every move.

### Task 5: Nix Operator Defaults

**Files:**
- Modify: `nix/packages/forge-operator-session.nix`
- Modify: `nix/packages/forge-desktop-shell.nix`
- Modify: `nix/nixos/profiles/forge-operator-desktop.nix`
- Modify: `nix/nixos/configurations/forge-operator-vm.nix`

- [x] Export `FORGE_RENDER_PROFILE=vm-safe` for the VirtualBox/operator session by default.
- [x] Preserve override ability for future native/bare-metal sessions.
- [x] Verify static Nix text checks where available.

### Task 6: Evidence

**Files:**
- Create/update: `docs/evidence/vm_boot/2026-05-18-live-start/README.md`
- Modify: `docs/runbooks/forge_operator_desktop_vm.md`

- [x] Record the observed pre-fix WebKit CPU evidence.
- [x] Document the VM-safe render profile and how to disable it.
- [x] Verify desktop tests/build and core untouched.
