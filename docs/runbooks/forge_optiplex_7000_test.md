# FORGE OptiPlex 7000 Test Target

## Scope

`forge-optiplex-7000` is the physical x86_64 NixOS test target for the local
Dell OptiPlex 7000. It is not the canonical operator VM and does not make
FORGE-K simulator services live authority.

The target intentionally uses:

- UEFI and systemd-boot;
- labeled, unencrypted test filesystems;
- the low-memory `vm-safe` FORGE renderer through a target-built
  `forge-desktop-shell` package;
- password-gated `greetd`/`tuigreet` login into the Labwc-backed
  `forge-operator` session;
- FORGE as the fixed, undecorated, always-bottom desktop canvas, with Labwc
  compositing native application windows above that canvas and FORGE tracking
  those windows through its launcher and taskbar;
- the Nix-built Forge operator toolbelt, including Foot, PCManFM, Mousepad,
  Firefox, Micro, Helix, SQLiteBrowser, and bounded diagnostic wrappers;
- `WEBKIT_DISABLE_DMABUF_RENDERER=1` for the Forge shell session to avoid the
  unstable WebKit DMA-BUF renderer path on the physical Intel test target;
- the single-application Cage session installed only as a fullscreen rollback
  and test path;
- loopback-only `forge-core` in CPU-safe mode;
- governed modelruntime with a single loopback-only Ollama worker;
- a static, unprivileged Ollama service account in the `forge` group so the
  worker can access only its governed `/forge/models/ollama` subtree;
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
safe FORGE operator desktop. Forge fills the output as the desktop canvas; it
must not appear with ordinary application decorations. TTY fallback remains
available.

For an offline target, stage an Ollama cache containing only the declared model
manifests and their referenced blobs at `/forge/models/ollama/models`. Preserve
ownership as `ollama:forge` and mode `0750` for directories; do not copy an
unbounded workstation model cache. Automatic `services.ollama.loadModels`
downloads are disabled because that helper always checks the remote registry;
model updates must use the same bounded workstation-to-target staging flow.

## Verification

After boot:

```bash
systemctl is-active forge-core
systemctl is-active ollama
curl -fsS http://127.0.0.1:18492/health
ollama list
cat /etc/forge/optiplex-test.env
cat /etc/forge/shell-session.env
free -h
swapon --show
command -v foot firefox pcmanfm mousepad micro hx
```

After `operator` signs in, the Labwc session must show a borderless FORGE
surface covering the output. The live `forge_desktop` process must receive
`FORGE_SHELL_MODE=operator-desktop`, `FORGE_SHELL_FULLSCREEN=false`,
`FORGE_OPERATOR_DESKTOP_LOCKED=true`, `FORGE_RENDER_PROFILE=vm-safe`,
`WEBKIT_DISABLE_DMABUF_RENDERER=1`, and the configured
`FORGE_API_TOKEN_FILE`. The shell status must report runtime online when the
authenticated dashboard calls succeed. CPU-only safe-mode warnings are policy
advisories and do not by themselves mean modelruntime is degraded.

From the Forge launcher, open Terminal, Files, Editor, and Browser. Each must
appear as a native window on the Forge desktop, acquire a Forge taskbar entry,
and remain focusable/minimizable/closable through the bounded compositor
bridge. Forge remains the desktop canvas below those windows; it must not raise
over them as an ordinary peer application. Native applications are composed on
the Forge workspace rather than embedded inside the Tauri webview.

For freeze diagnosis after a forced restart, capture the previous boot before
switching generations:

```bash
journalctl -b -1 -k --no-pager | grep -Ei 'oom|out of memory|killed process|i915|gpu|hang|reset'
journalctl -b -1 --no-pager | grep -Ei 'forge_desktop|WebKit|labwc|cage|segfault|oom|i915|gpu'
coredumpctl list --no-pager
```

Then complete the native desktop smoke checklist in
`docs/runbooks/desktop_shell_operator_smoke_test.md`. The 8 GiB disk swap and
zram remain enabled for the 4 GiB test host, but swap is not evidence that a
graphics hang is fixed; inspect the journal and repeat the interactive smoke.

The expected boundary is live AI-OS/core authority with the existing governed
paths. FORGE-K remains simulator-only except for the already documented narrow
live validation seams.
