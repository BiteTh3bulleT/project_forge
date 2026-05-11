# FORGE Operator Desktop VM Runbook

Status: Phase G6 operator desktop bring-up

Last verified: 2026-05-11 on VirtualBox VM `FORGE-OS`.

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
pgrep -a core
curl -fsS http://127.0.0.1:18492/health
```

Do not add shell UI controls that run service-control commands. Service checks
from this runbook are operator verification only.

## Rebuild-Safe Profile Import

Keep the VM NixOS import pointed at the mounted main checkout:

```nix
/projectforge/nix/nixos/profiles/forge-operator-desktop.nix
```

Do not import operator desktop profiles from `.worktrees/*` paths. Worktree paths are temporary implementation sandboxes and can disappear or lag behind main, which makes VM rebuilds non-repeatable.

If `/etc/nixos/configuration.nix` references a `.worktrees/*` checkout, replace that import with the `/projectforge/...` path above before rebuilding.

The installed `FORGE-OS` VM mounts the host checkout through the VirtualBox
shared folder named `projectforge`:

```nix
fileSystems."/projectforge" = {
  device = "projectforge";
  fsType = "vboxsf";
  options = [ "rw" "uid=1000" "gid=100" "dmode=775" "fmode=664" "nofail" ];
};
```

## Current VM Install State

The `FORGE-OS` VM was reinstalled from the NixOS minimal ISO on 2026-05-11.
Only the guest disk exposed inside the VM as `/dev/sda` was partitioned and
formatted. The host backing file was verified as:

```text
/home/rshort/VirtualBox VMs/FORGE-OS/FORGE-OS.vdi
```

Installed layout:

- VM firmware: EFI.
- Graphics controller: VMSVGA with 128 MiB VRAM and 3D acceleration enabled.
- WebKit dmabuf rendering is disabled for the operator session
  (`WEBKIT_DISABLE_DMABUF_RENDERER=1`) because VirtualBox VMSVGA can expose a
  Wayland/dmabuf path that causes the Tauri shell to exit with a GTK protocol
  error.
- VM disk: `80G VBOX HARDDISK`.
- `/dev/sda1`: 512 MiB FAT32 ESP, label `NIXBOOT`, mounted at `/boot`.
- `/dev/sda2`: ext4, label `FORGEOS`, mounted at `/`.
- Host shared folder: `projectforge` mounted at `/projectforge`.
- FORGE storage root: `/forge`, owned by the `forge` service account.
- Operator user: `operator`, in `wheel`, `networkmanager`, `video`, `render`, and `vboxsf`.
- SSH is enabled for key-based access.
- Current host access uses VirtualBox NAT forwarding: `ssh -p 2222 operator@127.0.0.1`.
- Automatic login remains disabled.
- ISO is detached and disk is first in the VirtualBox boot order.
- The packaged Tauri shell defaults to `1180x680` so the maximized operator
  shell fits smaller VirtualBox viewer viewports without hiding the taskbar.

Verified installed paths:

```bash
command -v forge-shell-session
command -v forge-operator-session
command -v forge-wayland-session
command -v forge_desktop
command -v forge-desktop-shell
findmnt /projectforge
sudo ls -ld /forge /forge/data /forge/models /forge/workspaces/default /forge/runtime
curl -fsS http://127.0.0.1:18492/health
```

## Start Operator Desktop

From a TTY login:

```bash
mkdir -p "$HOME/forge-session-logs"
forge-operator-session >"$HOME/forge-session-logs/forge-operator-session.log" 2>&1
```

Expected result:

- `labwc` starts.
- FORGE desktop shell opens as the primary surface.
- FORGE core status shows online.
- Other operator apps can open as normal windows.

If the compositor opens to a black screen with only a cursor, inspect the log
for `Gdk-Message: Error 71 (Protocol error) dispatching to Wayland display`.
That indicates the WebKit dmabuf fallback is not active in the launched
environment.

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
mkdir -p "$HOME/forge-session-logs"
forge-wayland-session >"$HOME/forge-session-logs/forge-wayland-rollback.log" 2>&1
```

If the graphical session fails, keep working from TTY and inspect:

```bash
tail -200 "$HOME/forge-session-logs/forge-operator-session.log"
pgrep -a core
curl -fsS http://127.0.0.1:18492/health
```
