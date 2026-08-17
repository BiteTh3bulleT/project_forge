# FORGE OptiPlex 7000 Test Target

## Scope

`forge-optiplex-7000` is the physical x86_64 NixOS test target for the local
Dell OptiPlex 7000. It is not the canonical operator VM. Production
`forgekernel` owns bootstrap ingress and canonical commit order through the
production Kernel owner; Control Lane is currently partial-live validation metadata
while the separate `forgek` simulator packages remain non-authoritative.

The target intentionally uses:

- UEFI and systemd-boot;
- labeled, unencrypted test filesystems;
- the full `default` FORGE renderer through a target-built
  `forge-desktop-shell` package;
- password-gated `greetd`/`tuigreet` login into the Labwc-backed
  `forge-operator` session;
- FORGE as the fixed, undecorated, always-bottom desktop canvas, with Labwc
  compositing native application windows above that canvas and FORGE tracking
  those windows through its launcher and taskbar;
- the Nix-built Forge operator toolbelt, including Foot, PCManFM, Mousepad,
  Firefox, Micro, Helix, SQLiteBrowser, and bounded diagnostic wrappers;
- a complete offline development workstation closure: LibreOffice, Thunderbird,
  PDF/annotation, graphics, media, password-vault, printing/scanning and audio
  applications; VSCodium with pinned Go/Rust/Python/Nix/YAML/TOML/Prettier
  extensions; and the Go, Node/TypeScript, Rust, Python, Nix, shell, C/C++ and
  Tauri toolchains used by this repository;
- a setgid group-writable `/forge/workspaces/default` development tree shared
  by the local `operator` and the bounded `forge-core` service, with
  `/forge/workspaces/default/Projects` created at boot;
- rootless Podman/Buildah/Skopeo, Git LFS, database clients, archive tools,
  hardware diagnostics, and Restic/Borg/rsync for operator-directed local
  development and backup work;
- `WEBKIT_DISABLE_DMABUF_RENDERER=1` for the Forge shell session to avoid the
  unstable WebKit DMA-BUF renderer path on the physical Intel test target;
- the single-application Cage session installed only as a fullscreen rollback
  and test path;
- loopback-only `forge-core` with safe-mode CPU forcing disabled so available
  local accelerators and the full runtime policy can be exercised;
- governed modelruntime with a single loopback-only Ollama worker;
- a static, unprivileged Ollama service account in the `forge` group so the
  worker can access only its governed `/forge/models/ollama` subtree;
- `qwen2.5:1.5b` (approximately 986 MB) as the default governed tool router;
  its Ollama template supports native tool calls and live routing probes must
  preserve exact arguments before it is promoted;
- `gemma3:1b-it-q4_K_M` (approximately 815 MB) as the completion fallback;
  only one model may be loaded and one request may run at a time;
- bounded native chat settings (`num_ctx=2048`, `num_predict=192`, six CPU
  threads, and `think=false`) so the small tool worker emits structured calls
  instead of consuming its response budget with a thinking trace;
- key-only SSH for the explicitly provisioned operator key;
- a static offline direct Ethernet link at `192.168.50.2/24` with no gateway,
  no DNS, no IPv6, and `never-default=true`, plus an nftables output policy
  that permits only loopback and `192.168.50.0/24`;
- no VirtualBox guest integration or autologin;
- explicit full-test shell policy enabling host power, model-lifecycle, and
  semantic-write controls. These remain authenticated and FORGE-K/gateway
  governed where a production contract exists; the shell still cannot claim
  Kernel authority;
- a native settings group inside FORGE Settings and the Start menu with
  NetworkManager adapter/profile selection, Wayland display arrangement,
  PipeWire audio, CUPS printers, Bluetooth devices, and GTK appearance tools.

## Offline direct-link contract

The OptiPlex is intentionally offline. Its `enp0s31f6` interface uses the
declarative NetworkManager profile `forge-direct-link` at
`192.168.50.2/24`. The directly attached workstation uses
`192.168.50.1/24`. Neither side has a gateway or DNS server on this link.

The workstation profile must use `ipv4.method manual`, not
`ipv4.method shared`. NetworkManager shared mode starts DHCP/DNS forwarding and
NAT, which would give the OptiPlex internet access through the workstation.
Repository updates, Nix closures, and model artifacts must be staged on the
workstation and transferred across the direct link.

The nftables `forge_offline` table is the hard boundary. Its output and forward
chains default to drop. Output is allowed only over loopback or through
`enp0s31f6` to `192.168.50.0/24`; inbound traffic is limited to established
connections, ICMP echo, and SSH from that direct subnet. A newly created
NetworkManager profile therefore cannot silently add internet egress.

## Full development workstation contract

The OptiPlex profile is self-contained after its Nix closure and project source
have been staged. It does not depend on a package marketplace, extension
download, browser install script, Flatpak, or language-version manager.

The installed workstation covers:

| Area | Included surface |
| --- | --- |
| Office and documents | LibreOffice, Hunspell US English, Thunderbird, Evince, Xournal++ and Pandoc |
| IDE and editors | VSCodium with pinned offline extensions, Mousepad, Helix, Micro and Vim |
| FORGE toolchains | Go/gopls/delve; Node 20/TypeScript/ESLint/Prettier; Rust/Cargo/Clippy/rust-analyzer/Tauri; Python/uv/ruff/Black; Nix/nil/nixfmt; GCC/Clang/GDB/LLDB/CMake/Ninja/Make/pkg-config |
| Desktop utilities | PCManFM/GVFS/UDisks, KeePassXC, GIMP, Inkscape, MPV, image/PDF viewers, screenshot/clipboard tools, calculator and archive tools |
| Devices | PipeWire audio, PulseAudio compatibility, CUPS/Gutenprint printing, SANE/AirScan scanning, firmware and thermal/power services |
| Data and operations | SQLite/PostgreSQL/Redis clients, Git/Git LFS/GitHub CLI, rootless Podman/Buildah/Skopeo, Restic, Borg and rsync |

All GUI applications that publish a safe `.desktop` entry are discovered by
the FORGE native-app catalog and open as compositor-managed windows above the
FORGE desktop canvas. The shell remains the desktop and taskbar; applications
are native Linux processes, not webview iframes.

The machine has only 4 GiB of RAM. Zram and the 8 GiB disk swap make the full
toolset available, but do not make simultaneous heavyweight workloads cheap.
Keep one local model loaded, and normally run only one of VSCodium,
LibreOffice, Thunderbird, GIMP, or Inkscape at a time. Rootless container tools
are installed, but no container daemon or image is started automatically.

Internet-dependent collaboration, cloud sync, extension marketplaces, package
registries and email delivery remain unavailable while the offline firewall is
active. Their clients are present for local files and for a later explicitly
approved network posture; they do not weaken the current egress policy.

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

The realized full-workstation NixOS closure is approximately 16 GiB. Reserve at
least 40 GiB of free root-filesystem capacity for multiple Nix generations,
source trees, build outputs, local databases and the staged model cache. A
64 GiB root device is the practical minimum for this test profile; 128 GiB or
more leaves substantially better development headroom.

## Build and install

Evaluate the target before installation:

```bash
nix eval .#nixosConfigurations.forge-optiplex-7000.config.system.build.toplevel.drvPath
nix build .#checks.x86_64-linux.forge-optiplex-7000
nix build .#nixosConfigurations.forge-optiplex-7000.config.system.build.toplevel --no-link
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

The `forge-core` systemd service carries Git in its own declared runtime path.
Installing Git only in `environment.systemPackages` is insufficient for the
governed `git.*` gateway lanes because system services use an isolated PATH.

For an offline target, stage an Ollama cache containing only the declared model
manifests and their referenced blobs at `/forge/models/ollama/models`. Preserve
ownership as `ollama:forge` and mode `0750` for directories; do not copy an
unbounded workstation model cache. Automatic `services.ollama.loadModels`
downloads are disabled because that helper always checks the remote registry;
model updates must use the same bounded workstation-to-target staging flow.
Remove a superseded model with `ollama rm` on both staging and target hosts only
after the replacement passes direct tool-selection, exact-argument, no-tool,
and governed FORGE runtime-proposal probes. `ollama rm` is destructive locally;
recovery requires restaging or downloading the model again.

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
command -v libreoffice codium go gopls node npm rustc cargo python uv gcc clang cmake nil nixfmt
command -v podman buildah skopeo git-lfs psql redis-cli restic borg rsync
command -v nm-connection-editor wdisplays pavucontrol system-config-printer blueman-manager lxappearance
test -w /forge/workspaces/default
test -d /forge/workspaces/default/Projects
systemctl is-active cups
systemctl is-active udisks2
systemctl --user is-active pipewire pipewire-pulse
lpstat -r
scanimage -L || true
podman info --format '{{.Host.Security.Rootless}}'
ip -4 address show dev enp0s31f6
ip route
nmcli connection show forge-direct-link
sudo nft list table inet forge_offline
! ip route get 1.1.1.1
! curl --connect-timeout 3 http://1.1.1.1
curl -fsS http://127.0.0.1:18492/forge/kernel/status | jq -e '
  .status == "partial_live_validation_ready" and
  .kernel_authority_exclusive == true and
  .capability_implemented == true and
  .projection_healthy == true and
  .host_ready == true and
  .recovery_verified == true and
  .unsafe_test_mode == false and
  .live_owner == "forge_k.kernel" and
  .live_kernel_authority == false and
  .authority_blocked_gates == 0'
```

`ollama list` must contain `qwen2.5:1.5b` and
`gemma3:1b-it-q4_K_M`, and must not contain the retired Smuxo model. Test the
router through FORGE as well as directly through Ollama: a direct tool-call
success alone does not prove Context Compiler, runtime-proposal, consensus, or
model-visibility enforcement.

For maximum currently supported autonomous operation, use the authenticated
settings API to select `maintain`, activate governed VSA influence, and select
deep Dream analysis. `maintain` may commit only actions allowed by an active
durable charter, freedom budget, authorization proof, and FORGE-K validation;
it does not bypass approval-required actions. Memory maintenance and Dream v0
remain proposal-only where no production Kernel commit contract exists.

```bash
token="$(cat /forge/data/auth/api_token)"
curl -fsS -X PATCH \
  -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json' \
  http://127.0.0.1:18492/api/settings \
  --data '{"autonomyMode":"maintain","retrievalVSAMode":"active","dreamMode":{"enabled":true,"defaultDryRun":true,"mode":"deep_dream","allowLongTermPromotion":true,"requireOperatorReviewForLongTerm":true,"allowCommits":false}}'
curl -fsS -H "Authorization: Bearer $token" \
  http://127.0.0.1:18492/api/autonomy/status | jq -e \
  '.available == true and .mode == "maintain" and .counts.activeCharters > 0 and .counts.budgets > 0'
```

Do not set `mission` merely as a stronger synonym for `maintain`. Mission mode
requires an explicit mission-class charter and otherwise fails closed. The
current default charters intentionally authorize bounded maintenance and
context preparation, not an open-ended mission.

The route table must contain only `192.168.50.0/24` through `enp0s31f6`.
There must be no default route. `ip route get 1.1.1.1` must fail with
`Network is unreachable`; failure to resolve an internet hostname alone is not
sufficient evidence because a public route could still exist without DNS.

After `operator` signs in, the Labwc session must show a borderless FORGE
surface covering the output. The live `forge_desktop` process must receive
`FORGE_SHELL_MODE=operator-desktop`, `FORGE_SHELL_FULLSCREEN=false`,
`FORGE_OPERATOR_DESKTOP_LOCKED=true`, `FORGE_RENDER_PROFILE=default`,
`WEBKIT_DISABLE_DMABUF_RENDERER=1`, and the configured
`FORGE_API_TOKEN_FILE`. The shell status must report runtime online when the
authenticated dashboard calls succeed. This profile deliberately sets
`FORGE_SHELL_SAFE_MODE=false`, enables the implemented unsafe test controls,
and retains the offline firewall and authenticated API as its outer boundary.
`FORGE_UNSAFE_TEST_MODE=true` also permits a model proposal to become visible
only after its live FORGE-K Context Compiler receipt and runtime-proposal
decision verify. Missing, caller-forged, or tampered bindings remain withheld.

Open **Settings → Host + hardware → Native System Controls** and verify that
Network Connections, Displays, Audio, Printers, Bluetooth, and Appearance each
launch as native windows. Network Connections must enumerate NetworkManager
adapters and profiles, including `forge-direct-link`. Editing or adding a
profile does not override the nftables offline-egress boundary.

The running graphical session must come from the current generation. NixOS can
activate new packages without replacing an already-running Labwc/Tauri session;
that stale process retains its old environment and policy flags. After a shell
or session-policy deployment, log out and back in (or terminate the old local
session from a TTY) before testing power or settings controls. Confirm with:

```bash
pid="$(pgrep -n forge_desktop)"
tr '\0' '\n' < "/proc/$pid/environ" | grep -E \
  '^(FORGE_SHELL_DIRECT_SYSTEM_CONTROL|FORGE_SHELL_SAFE_MODE|PATH)='

# direct shell proof: correct compositor/path, no auto-started browser daemon
./scripts/verify-forge-shell-session.sh
```

In full test mode, `FORGE_SHELL_DIRECT_SYSTEM_CONTROL=true`. Restart and Shut
Down wait for the local `systemctl --no-block` request to be accepted and show
an error if logind rejects it. The NixOS test profile grants the active local
`operator` session only the bounded logind reboot/poweroff and NetworkManager
control actions; it does not grant blanket passwordless sudo.

From the Forge launcher, open Terminal, Files, Editor, Browser, VSCodium,
LibreOffice Writer, Evince, and KeePassXC. Each must
appear as a native window on the Forge desktop, acquire a Forge taskbar entry,
and remain focusable/minimizable/closable through the bounded compositor
bridge. Forge remains the desktop canvas below those windows; it must not raise
over them as an ordinary peer application. Native applications are composed on
the Forge workspace rather than embedded inside the Tauri webview.

Confirm the compositor-visible Forge identity and active desktop rule:

```bash
XDG_RUNTIME_DIR=/run/user/1000 WAYLAND_DISPLAY=wayland-0 lswt -j
grep -F 'identifier="forge_desktop"' /run/user/1000/forge-operator-labwc/rc.xml
```

The installed Tauri binary currently reports `app-id: forge_desktop`. A rule
that matches only the bundle identifier `dev.forge.workshop` does not prove the
desktop surface is pinned below native windows.

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

The expected boundary is production FORGE-K as the observed partial-live
semantic, context, and model-visibility authority. The `services/core/internal/forgek`
simulator remains isolated, Control Lane remains a bounded durable port, and
disabled capabilities retain no alternate authority.
