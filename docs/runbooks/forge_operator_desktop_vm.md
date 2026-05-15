# FORGE Operator Desktop VM Runbook

Status: Phase G6 operator desktop bring-up

Last verified: 2026-05-11 on VirtualBox VM `FORGE-OS`.

In-repo evidence record:
[docs/evidence/vm_boot/2026-05-11-forge-os-operator-desktop.md](../evidence/vm_boot/2026-05-11-forge-os-operator-desktop.md)

## Purpose

This runbook starts the opt-in FORGE operator desktop session in a NixOS VM. FORGE remains the primary desktop surface, while `labwc` provides the window-management substrate needed for terminal, file-manager, and other operator app windows.

## Canonical Nix VM Target

The Nix-first operator VM target is:

```bash
nix build .#nixosConfigurations.forge-operator-vm.config.system.build.vm
```

Run the VM script produced by that build:

```bash
./result/bin/run-forge-operator-vm-vm
```

This target imports:

- `nix/nixos/modules/forge-os.nix`
- `nix/nixos/profiles/forge-operator-desktop.nix`

It includes `forge-core`, the packaged desktop shell, `forge-operator-session`,
the operator toolbelt, `/forge` storage layout, local-only core binding, and
safe shell flags. It is the preferred reproducible bring-up path. Manual ISO
installation and VirtualBox shared-folder profiles are fallback/operator
debugging paths.

The canonical VM also starts `forge-core` with governed modelruntime enabled
and `FORGE_MODEL_DEFAULT_BACKEND=ollama_compat`. This does not start Ollama or
pull models by itself; it makes the local Ollama endpoint discoverable once the
operator starts Ollama from the toolbelt.

Default local VM login:

```text
user: operator
password: forge
```

The VM keeps SSH disabled by default and does not enable autologin. Change the
local password before exposing the VM beyond the host-only/local development
boundary.

## Boundaries

- Do not enable autologin.
- Do not remove TTY access.
- Do not remove the Cage fullscreen rollback session.
- Do not treat FORGE-K as live authority.
- Do not launch arbitrary commands from FORGE UI surfaces in this phase.
- Do not add `curl | sh` installers; Ollama and operator tools come from Nix.
- Do not add model load/unload, service restart, or rebuild controls to the UI.

## Start FORGE Core

Verify core is active:

```bash
pgrep -a core
curl -fsS http://127.0.0.1:18492/health
```

Do not add shell UI controls that run service-control commands. Service checks
from this runbook are operator verification only.

## Verify Local Model Loop

Inside the operator session:

1. Open Terminal from the Start menu.
2. Confirm toolbelt Ollama is present with `command -v ollama`.
3. Pull and run the intended local model from the terminal.
4. Confirm Ollama responds at `http://127.0.0.1:11434`.
5. Open FORGE Chat, use Refresh models, select the discovered local Ollama model, and send a short prompt.

Expected result: Chat records a normal assistant reply and trace metadata shows
`backend=ollama_compat`. If Chat says no runtime models are available, verify
the model appears in `ollama list`, then refresh the Chat model list again. The
refresh path re-discovers local Ollama models without restarting `forge-core`.

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

## Legacy Manual VirtualBox VM State

This section records the hand-built `FORGE-OS` VirtualBox VM used during
bring-up. It is useful evidence, but it is not the canonical Nix VM default.
The canonical target above keeps SSH disabled unless a reviewed configuration
changes that default.

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
- This manual VM has SSH enabled for key-based access.
- Manual VM host access uses VirtualBox NAT forwarding: `ssh -p 2222 operator@127.0.0.1`.
- Automatic login remains disabled.
- ISO is detached and disk is first in the VirtualBox boot order.
- The packaged Tauri shell defaults to `1180x680` as a fallback, and the locked
  operator session fits the shell window to the detected monitor bounds before
  maximizing so VirtualBox viewer size changes do not leave the desktop offset.

Verified installed paths:

```bash
command -v forge-shell-session
command -v forge-operator-session
command -v forge-wayland-session
command -v forge_desktop
command -v forge-desktop-shell
command -v forge-operator-ollama-status
command -v forge-operator-models
command -v forge-operator-btop
command -v forge-operator-lazygit
command -v ollama
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

After a successful VM run, update the evidence record with the exact repo
commit mounted at `/projectforge`, the session log, `/health`, `/api/meta`, and
a screenshot artifact. Do not mark a new VM boot as verified from static Nix or
Go tests alone.

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

The terminal is for local inspection and ordinary operator work. Ollama CLI
availability comes from Nix. Model lifecycle, service lifecycle, and NixOS
configuration changes must stay in reviewed configuration or existing governed
FORGE paths; the shell UI must not provide install, load/unload, restart, or
rebuild controls.

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
