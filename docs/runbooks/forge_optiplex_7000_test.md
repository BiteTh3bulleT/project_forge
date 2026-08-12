# FORGE OptiPlex 7000 Test Target

## Scope

`forge-optiplex-7000` is the physical x86_64 NixOS test target for the local
Dell OptiPlex 7000. It is not the canonical operator VM and does not make
FORGE-K simulator services live authority.

The target intentionally uses:

- UEFI and systemd-boot;
- labeled, unencrypted test filesystems;
- the physical/default FORGE renderer through `pkgs.forge-desktop-shell`;
- password-gated `greetd`/`tuigreet` login into the fullscreen Cage session;
- loopback-only `forge-core` in CPU-safe mode;
- governed modelruntime with a single loopback-only Ollama worker;
- `gemma3:1b-it-q4_K_M` (approximately 815 MB) as the default governed worker;
- `smuxo/smuxoAI:0.8b` as an optional downloaded secondary worker
  (approximately 1 GB); only one model may be loaded and one request may run at
  a time;
- key-only SSH for the explicitly provisioned operator key;
- no VirtualBox guest integration, autologin, or automatic host mutation.

## Disk contract

The configuration consumes an already-created disk layout by label and never
partitions or formats a disk itself:

| Label | Filesystem | Purpose |
| --- | --- | --- |
| `FORGE_EFI` | FAT32 | 1 GiB EFI system partition mounted at `/boot` |
| `FORGE_SWAP` | swap | 8 GiB disk swap for the 4 GB test machine |
| `FORGE_ROOT` | ext4 | remaining NVMe capacity mounted at `/` |

Disk erasure and partition creation are separate operator-approved installation
steps. Never infer the install disk from `/dev/sd*` or `/dev/nvme*`; validate its
model, serial, size, removability, and mount state before destructive work.

## Build and install

Evaluate the target before installation:

```bash
nix eval .#nixosConfigurations.forge-optiplex-7000.config.system.build.toplevel.drvPath
nix build .#checks.x86_64-linux.forge-optiplex-7000
```

With the approved labeled filesystems mounted below `/mnt`:

```bash
sudo nixos-install --flake .#forge-optiplex-7000 --no-root-passwd
```

Set the mutable local operator password explicitly from the installer after the
build completes. On first boot, sign in as `operator`; `tuigreet` launches the
safe fullscreen FORGE shell. TTY fallback remains available.

## Verification

After boot:

```bash
systemctl is-active forge-core
systemctl is-active ollama
curl -fsS http://127.0.0.1:18492/health
ollama list
cat /etc/forge/optiplex-test.env
free -h
swapon --show
```

The expected boundary is live AI-OS/core authority with the existing governed
paths. FORGE-K remains simulator-only except for the already documented narrow
live validation seams.
