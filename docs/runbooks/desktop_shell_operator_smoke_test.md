# Desktop Shell Operator Smoke Test

## Purpose

Use this checklist to manually verify the Phase G8 FORGE desktop shell in the operator VM or a native FORGE shell session.

## Prerequisites

- Record the repo commit, date, VM/session name, and whether the shell is running packaged or dev build.
- Confirm `forge-core` health from a terminal or host:

```bash
curl -fsS http://127.0.0.1:18492/health
```

- Confirm the shell is running through the expected labwc path:

```text
graphical login -> labwc -> forge-shell-session -> forge-desktop-shell
```

For this profile, `forge-desktop-shell` is a Tauri/WebKit process, so a
`WebKitWebProcess` child is normal. The real validation is that the shell is
started through the expected session path, and no separate browser daemon is
running before you intentionally launch one.

You can run this directly with:

```bash
cd /path/to/ProjectForge
./scripts/verify-forge-shell-session.sh
```

```bash
# Direct-shell proof (run immediately after login, before opening Firefox/Browser):
pgrep -af "(^|/)labwc([[:space:]]|$)" || true
pgrep -af "(^|/)forge-shell-session([[:space:]]|$)" || true
pgrep -af "(^|/)forge-desktop-shell([[:space:]]|$)" || true
pgrep -af "(^|/)forge_desktop([[:space:]]|$)" || true

for browser in firefox firefox-esr chromium chrome google-chrome microsoft-edge webex; do
  pgrep -af "(^|/)${browser}([[:space:]]|$)" && exit 1 || true
done

# Expected webkit child; use presence only as evidence this is a native shell.
pgrep -af "(^|/)WebKitWebProcess([[:space:]]|$)" || true
```

Important: if you require **zero WebKit code at runtime**, the current
`forge-desktop-shell` cannot satisfy it. This shell is Tauri-based by design.
Removing WebKit requires a native non-WebKit operator shell implementation
(for example GTK/Rust/Wayland) and a new session package path. That is a
separate architecture slice, not just a session-path switch.

## Boundaries

This smoke test must not run `systemctl`, `nixos-rebuild`, package installs, reboot/shutdown operations, kernel module changes, model load/unload commands, direct semantic memory writes, or host mutation controls from the shell UI.

## Checklist

1. Boot or start the operator VM/session.
2. Confirm the FORGE-branded OS graphical login appears before the desktop.
3. Log in and confirm the desktop is empty except for normal shell chrome/taskbar.
4. Open Terminal from Start.
5. Open a second Terminal and confirm both native windows have separate taskbar entries.
6. Close one Terminal and confirm only that taskbar entry disappears.
7. Close the second Terminal and confirm no stale Terminal entry remains.
8. Open Files from Start.
9. Open a second Files window if available and confirm separate taskbar entries.
10. Open Browser from Start.
11. Left-click inactive native taskbar entries and confirm they focus/restore.
12. Left-click a focused native taskbar entry and confirm it minimizes.
13. Middle-click a native taskbar entry and confirm it closes or fails visibly without a stale entry.
14. Right-click a native taskbar entry and test bounded focus/minimize/maximize/fullscreen/close actions.
15. Confirm unsupported native actions fail visibly and safely.
16. Restart the shell session through the normal session path and confirm no stale in-shell windows return after login.
17. Confirm fallback desktop/session or TTY rollback remains available.

## Expected Results

- No failed or refused native launch leaves a phantom taskbar entry.
- Pending native launch placeholders disappear when the compositor window appears.
- Stale pending placeholders expire with a visible diagnostic.
- Multiple native windows remain separate by compositor window id.
- Unknown apps use fallback labels/icons.
- FORGE remains the operator-facing shell while labwc owns compositor mechanics.

## Evidence To Capture

- Current commit.
- Screenshot of empty desktop after login.
- Screenshot of multiple native windows and taskbar entries.
- Screenshot or log of a bounded failure diagnostic if encountered.
- `curl -fsS http://127.0.0.1:18492/health` output.
- Any skipped or blocked checks.

## Rollback

Use TTY, fallback session selection, or Nix generation rollback. Keep `/forge` data intact. Do not use cleanup, package-manager mutation, service-control, or modelruntime commands as part of shell rollback unless a separate reviewed recovery procedure explicitly calls for them.
