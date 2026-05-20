# FORGE Operator Desktop VM Runbook

Status: Phase G8 native desktop runtime and operator shell UX bring-up

Last verified: 2026-05-16 on VirtualBox VM `FORGE-OS` for the appliance-style
FORGE shell loading/login path. Native graphical password login evidence is
pending for the canonical Nix VM target.

In-repo evidence record:
[docs/evidence/vm_boot/2026-05-11-forge-os-operator-desktop.md](../evidence/vm_boot/2026-05-11-forge-os-operator-desktop.md)

Latest render evidence:
[docs/evidence/vm_boot/2026-05-18-live-start/README.md](../evidence/vm_boot/2026-05-18-live-start/README.md)

Phase G8 desktop shell verification:
[docs/status/phase_g8_desktop_shell_verification.md](../status/phase_g8_desktop_shell_verification.md)

Operator smoke checklist:
[docs/runbooks/desktop_shell_operator_smoke_test.md](desktop_shell_operator_smoke_test.md)

## Purpose

This runbook covers the preferred FORGE native desktop runtime path in a NixOS
VM. FORGE remains the primary desktop surface, while NixOS/Linux remains the
substrate for boot, graphical login, display plumbing, rollback, and recovery.
`labwc` provides the window-management substrate needed for terminal,
file-manager, and other operator app windows.

Preferred operator flow:

```text
Power on
-> FORGE-OS Runtime boot splash
-> graphical password login
-> FORGE native desktop session
```

The operator should not normally land at a TTY and manually type
`forge-operator-session`. Manual TTY launch remains documented below as a
recovery/fallback path.

## Canonical Nix VM Target

The Nix-first operator VM target is:

```bash
nix build .#nixosConfigurations.forge-operator-vm.config.system.build.vm
```

Run the VM script produced by that build:

```bash
./result/bin/run-forge-operator-vm-vm
```

The native desktop runtime target imports:

- `nix/nixos/modules/forge-os.nix`
- `nix/nixos/profiles/forge-native-desktop-runtime.nix`

The native runtime profile composes the operator desktop profile and adds the
FORGE-OS Runtime boot splash plus graphical password login. It includes
`forge-core`, the packaged desktop shell, `forge-operator-session`, the operator
toolbelt, `/forge` storage layout, local-only core binding, and safe shell
flags. It is the preferred reproducible bring-up path. Manual ISO installation,
VirtualBox shared-folder profiles, and direct TTY session launch are
fallback/operator debugging paths.

The VirtualBox/operator shell defaults to the low-cost render profile:

```text
FORGE_RENDER_PROFILE=vm-safe
VITE_FORGE_RENDER_PROFILE=vm-safe
```

`vm-safe` is a frontend/rendering performance profile for the Tauri/WebKitGTK
surface. It does not change FORGE authority, modelruntime behavior, gateway
execution, approvals, memory writes, or FORGE-K live authority. It exists
because pre-fix live-start evidence showed the VirtualBox/WebKit path could
produce high CPU pressure and sluggish repaint behavior before effects were
forced down.

The canonical VM also starts `forge-core` with governed modelruntime enabled
and `FORGE_MODEL_DEFAULT_BACKEND=ollama_compat`. This does not start Ollama or
pull models by itself; it makes the local Ollama endpoint discoverable once the
operator starts Ollama from the toolbelt.

Default local VM login:

```text
user: operator
password: forge
```

The VM keeps SSH disabled by default and does not enable autologin. Password
login is required. TTY access and Nix generation rollback must remain available.
Change the local password before exposing the VM beyond the host-only/local
development boundary.
The packaged desktop shows a FORGE loading screen before the FORGE login screen
at boot. The current VM greeter is a local operator UX gate, not a PAM-backed
security boundary yet.

## Boundaries

- Do not enable autologin.
- Do not replace password login with autologin.
- Do not remove TTY access.
- Do not remove Nix generation rollback.
- Do not remove the Cage fullscreen rollback session.
- Do not treat FORGE-K as live authority.
- Do not launch arbitrary commands from FORGE UI surfaces in this phase.
- Do not add `curl | sh` installers; Ollama and operator tools come from Nix.
- Do not add model load/unload, service restart, or rebuild controls to the UI.

## Normal Native Desktop Login

Expected normal path:

```text
firmware / bootloader
-> Plymouth FORGE-OS splash
-> greetd/ReGreet graphical password login
-> labwc compositor
-> forge-shell-session
-> packaged forge-desktop-shell
-> FORGE shell loading screen
-> FORGE login screen
-> empty FORGE operator desktop
```

1. Start the VM.
2. Confirm the FORGE-OS Runtime boot splash appears during boot.
3. Log in through the graphical greeter as `operator`.
4. Confirm the login starts the FORGE native desktop session.
5. At the FORGE login screen, sign in with the local VM operator credentials above when the shell presents it.
6. Open terminal/files/toolbelt apps from FORGE.
7. Lock or log out through the desktop path when done.

The shell clears restored in-shell FORGE windows during login so the first
desktop view is wallpaper plus the pinned dock only. Do not mark a new native
desktop boot as verified without screenshot/log evidence showing the
splash/login/session path.

## Verify FORGE Core

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

Keep the VM NixOS import pointed at the mounted main checkout. The preferred
native runtime profile path is:

```nix
/projectforge/nix/nixos/profiles/forge-native-desktop-runtime.nix
```

The manual operator desktop profile remains the fallback building block:

```nix
/projectforge/nix/nixos/profiles/forge-operator-desktop.nix
```

Do not import operator desktop profiles from `.worktrees/*` paths. Worktree paths are temporary implementation sandboxes and can disappear or lag behind main, which makes VM rebuilds non-repeatable.

If `/etc/nixos/configuration.nix` references a `.worktrees/*` checkout, replace
that import with the `/projectforge/...` native runtime path before rebuilding.
Use the operator desktop profile path only when intentionally recovering or
debugging the older manual session flow.

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
- The operator shell defaults to `FORGE_RENDER_PROFILE=vm-safe` to reduce the
  WebKitGTK paint/compositing cost observed during the 2026-05-18 live-start
  render investigation.
- VM disk: `80G VBOX HARDDISK`.
- `/dev/sda1`: 512 MiB FAT32 ESP, label `NIXBOOT`, mounted at `/boot`.
- `/dev/sda2`: ext4, label `FORGEOS`, mounted at `/`.
- Host shared folder: `projectforge` mounted at `/projectforge`.
- FORGE storage root: `/forge`, owned by the `forge` service account.
- Operator user: `operator`, in `wheel`, `networkmanager`, `video`, `render`, and `vboxsf`.
- This manual VM has SSH enabled for key-based access.
- Manual VM host access uses VirtualBox NAT forwarding: `ssh -p 2222 operator@127.0.0.1`.
- Display-manager automatic login remains disabled.
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

## Recovery: Manual TTY Operator Desktop

If the graphical login path fails, switch to a TTY and launch the operator
desktop manually:

```bash
mkdir -p "$HOME/forge-session-logs"
forge-operator-session >"$HOME/forge-session-logs/forge-operator-session.log" 2>&1
```

Expected result:

- `labwc` starts.
- FORGE desktop shell opens as the primary surface.
- FORGE core status shows online.
- Other operator apps can open as normal windows.

After a successful recovery run, update the evidence record with the exact repo
commit mounted at `/projectforge`, the session log, `/health`, `/api/meta`, and
a screenshot artifact. Do not mark a new VM boot as verified from static Nix or
Go tests alone.

If the compositor opens to a black screen with only a cursor, inspect the log
for `Gdk-Message: Error 71 (Protocol error) dispatching to Wayland display`.
That indicates the WebKit dmabuf fallback is not active in the launched
environment.

## Render Profile Override

Keep `vm-safe` enabled for normal VirtualBox operator work. To disable it for a
diagnostic launch from a TTY:

```bash
FORGE_RENDER_PROFILE=default VITE_FORGE_RENDER_PROFILE=default forge-operator-session
```

The operator desktop profile builds the packaged shell with
`renderProfile = "vm-safe"`. For a persistent non-VirtualBox or bare-metal
profile, instantiate `nix/packages/forge-desktop-shell.nix` with:

```nix
renderProfile = "default";
```

Inside the shell, local storage can only override when the runtime has not
already supplied an explicit render profile. The packaged VirtualBox/operator
build supplies `vm-safe`, so stale browser storage must not silently re-enable
the heavier default path. Keep any non-VM override tied to hardware-specific
evidence; the canonical VirtualBox/operator VM default remains `vm-safe`.

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

The native runtime must preserve TTY access and Nix generation rollback. If the
graphical login/session path fails, use the bootloader generation menu to roll
back, or switch to TTY and start the prior fullscreen shell path:

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
