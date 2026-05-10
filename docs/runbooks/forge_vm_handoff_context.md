# FORGE VM Handoff Context

Updated: 2026-05-10

This file is a handoff note for continuing FORGE VM bring-up on another machine.
It captures the stopping point from the VirtualBox test lane and the next safe
actions.

## Current Repository State

- Remote: `https://github.com/BiteTh3bulleT/project_forge.git`
- Branch: `main`
- Latest pushed checkpoint at handoff:
  - `6a32516 chore: checkpoint forge vm handoff`
- The checkpoint includes:
  - G6 read-only shell system surfaces and docs.
  - VM/NixOS bring-up docs and Nix shell/session packaging updates.
  - Core hardening/test additions from the improvement loop.
  - Fixed `nix/packages/forge-core.nix` `vendorHash` for the current Go module graph.

## Verification Already Run Before Handoff

These passed before pushing the checkpoint:

```sh
npm run typecheck
npm run build:core
```

The VM build of the full Wayland shell package was started but not completed
before shutdown. It reached the desktop/Tauri package chain.

## Previous VM State

The previous VM was named `FORGE-OS` in VirtualBox.

Important storage fact:

- The only disk formatted during that attempt was the VM disk exposed inside the
  guest as `/dev/sda`.
- Host-side backing file was verified as a VirtualBox VDI:
  `/home/rshort/VirtualBox VMs/FORGE-OS/FORGE-OS.vdi`
- Do not assume that path exists on the next desktop machine.
- Do not format any host disk.

The VM was shut down cleanly before this handoff.

## Previous Bring-Up Method

The NixOS live ISO had too little writable root store for a full build. The
working path was:

1. Attach the repository as a VirtualBox shared folder named `ProjectForge`.
2. Mount it in the NixOS live guest at `/mnt/projectforge`.
3. Use a VM-local ext4 disk mounted at `/mnt/vmdisk` for build/store space.
4. Build with a VM-local Nix store, not the shared folder:

```sh
cd /mnt/projectforge
TMPDIR=/mnt/vmdisk/nix-tmp \
  nix --store /mnt/vmdisk/nix-store \
  --extra-experimental-features 'nix-command flakes' \
  --option download-buffer-size 536870912 \
  build .#forge-core --out-link /mnt/vmdisk/result-core
```

The shared folder cannot safely host the Nix store because VirtualBox shared
folders do not support the symlink behavior Nix expects.

## Recommended Next Desktop Path

On the desktop machine:

1. Pull latest:

```sh
git pull origin main
```

2. Prefer building on the host or in a real VM disk-backed Nix store:

```sh
nix build .#forge-core
nix build .#forge-wayland-session
```

3. If building inside a fresh NixOS live VM, create or attach a dedicated VM disk
   and mount it as `/mnt/vmdisk`. Use that disk for `TMPDIR` and the Nix store.

4. Once packages build, start core first:

```sh
mkdir -p /mnt/vmdisk/forge-data/{data,models,workspaces/default}
FORGE_DATA_DIR=/mnt/vmdisk/forge-data/data \
FORGE_MODEL_HOME=/mnt/vmdisk/forge-data/models \
FORGE_WORKSPACE_DIR=/mnt/vmdisk/forge-data/workspaces/default \
FORGE_CORE_BIND_HOST=127.0.0.1 \
FORGE_CORE_PORT=18492 \
  /mnt/vmdisk/result-core/bin/core >/mnt/vmdisk/forge-core.log 2>&1 &
```

5. Verify core health from inside the VM:

```sh
curl -f http://127.0.0.1:18492/health || tail -n 80 /mnt/vmdisk/forge-core.log
```

6. Launch the shell session only after core is reachable:

```sh
FORGE_CORE_URL=http://127.0.0.1:18492 \
  /mnt/vmdisk/result-wayland/bin/forge-wayland-session
```

## G6 Authority Boundaries To Preserve

The shell system surfaces are read-only operator visibility.

Do not add or use shell controls that:

- Run `systemctl`.
- Run `nixos-rebuild`.
- Restart, shut down, rebuild, clean, delete, install, remove, or upgrade host state.
- Load or unload models.
- Write semantic memory directly.
- Bypass gateway, permissions, lanes, audit, controllane, memory, modelruntime,
  FORGE-H, or FORGE-K authority.

FORGE-K remains simulator-only unless a future explicit integration phase says
otherwise.

## Useful Docs

- `docs/runbooks/current_forge_bringup.md`
- `docs/operations/forge_graphical_shell_session.md`
- `docs/architecture/forge_graphical_shell.md`
- `docs/architecture/shell_system_surfaces.md`
- `docs/adr/0012-forge-wayland-shell-session.md`

