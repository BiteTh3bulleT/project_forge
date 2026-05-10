# FORGE Operator Desktop VM Runbook

Status: Phase G6 operator desktop bring-up

## Purpose

This runbook starts the opt-in FORGE operator desktop session in a NixOS VM. FORGE remains the primary desktop surface, while `labwc` provides the window-management substrate needed for terminal, file-manager, and other operator app windows.

## Boundaries

- Do not enable autologin.
- Do not remove TTY access.
- Do not remove the Cage fullscreen rollback session.
- Do not treat FORGE-K as live authority.
- Do not launch arbitrary commands from FORGE UI surfaces in this phase.

## Start FORGE Core

Verify core is active:

```bash
systemctl is-active forge-core
curl -fsS http://127.0.0.1:18492/health
```

## Start Operator Desktop

From a TTY login:

```bash
forge-operator-session >/mnt/vmdisk/forge-operator-session.log 2>&1
```

Expected result:

- `labwc` starts.
- FORGE desktop shell opens as the primary surface.
- FORGE core status shows online.
- Other operator apps can open as normal windows.

## Open Operator Tools

From a terminal inside the session:

```bash
foot &
pcmanfm &
```

Use the terminal for operator-owned setup work such as Ollama installation, model downloads, and NixOS configuration changes.

## Host Health Check

From the Windows host:

```powershell
Invoke-RestMethod -Uri 'http://127.0.0.1:18492/health'
```

## Rollback

Exit the operator session or return to TTY, then start the prior fullscreen shell path:

```bash
forge-wayland-session >/mnt/vmdisk/forge-wayland-rollback.log 2>&1
```

If the graphical session fails, keep working from TTY and inspect:

```bash
tail -200 /mnt/vmdisk/forge-operator-session.log
systemctl status forge-core --no-pager
```
