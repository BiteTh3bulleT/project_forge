# FORGE Operator Desktop

Status date: 2026-05-15.

The FORGE operator desktop is the native desktop session for running the FORGE
graphical shell as an operator workstation. NixOS remains the substrate for
boot, graphical login, display plumbing, rollback, and emergency access. FORGE
remains the shell/operator surface.

The preferred native runtime is defined by
`nix/nixos/profiles/forge-native-desktop-runtime.nix`, which composes the
operator desktop profile at `nix/nixos/profiles/forge-operator-desktop.nix`.

## Launch Model

The preferred VM session chain is:

```text
Power on
-> FORGE-OS Runtime boot splash
-> graphical password login
-> FORGE native desktop session
-> forge-operator-session
-> labwc
-> forge-shell-session
-> forge-desktop-shell
-> empty FORGE desktop
-> local forge-core
```

Password login is required. Autologin is not allowed. TTY access and Nix
generation rollback remain preserved recovery paths.

The native/operator Nix shell build sets `bootLogin = false` and
`emptyDesktopOnBoot = true`. ReGreet/PAM is the login boundary; the older
client-side FORGE unlock screen is not part of the normal native desktop path.

`FORGE_SHELL_BINARY` remains a development fallback for non-operator shell
sessions. The `operator-desktop` path rejects that override and requires the
Nix-packaged `forge-desktop-shell`.

Manual TTY launch of `forge-operator-session` is recovery/fallback only. It is
not the normal operator path for the native desktop runtime.

## Nix-First VM

The canonical local VM output is:

```bash
nix build .#nixosConfigurations.forge-operator-vm.config.system.build.vm
./result/bin/run-forge-operator-vm-vm
```

That VM target is expected to import the FORGE-OS module and the native desktop
runtime profile. It is the preferred reproducible bring-up path because it
includes the boot splash, graphical password login, daemon, desktop shell,
empty desktop boot policy, session wrapper, operator toolbelt, and safe
defaults in one NixOS system closure.

The session remains safe by default:

- no automatic login
- password login required
- TTY fallback preserved
- Nix generation rollback preserved
- no autostart replacement of an existing desktop
- no direct system-control authority
- no model mutation authority
- no semantic memory write authority
- no FORGE-K live authority migration

## Operator Apps

The desktop Start menu is an allowlist. It exposes categorized launchers:

- Workspace: terminal, files, native Mousepad editor, archive manager
- Internet: browser and fixed core health/API probe
- AI Runtime: Ollama status and governed modelruntime status
- System: process monitor, logs, network diagnostics, hardware diagnostics
- Developer: SQLite browser and Git UI
- FORGE: local FORGE status

CLI tools are launched through fixed `forge-operator-*` wrappers in a terminal. Native GUI apps such as Mousepad, PCManFM, Xarchiver, and Firefox launch directly from the allowlist. The UI passes only an allowlisted app ID to Tauri; it does not accept arbitrary command text or user-provided launch arguments.

The Start menu also exposes Logout, Reboot, and Shutdown controls. Logout returns to the FORGE/OS session login boundary. Reboot and Shutdown are real host-power controls, but they are disabled by default and reflect the Tauri `read_host_power_policy` result in the UI. They only become clickable when `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` is explicitly enabled for the shell session.

## Runtime Services

Ollama is provided by Nix, not by a runtime installer. The operator profile provides the `ollama` CLI on `PATH`, and the canonical operator VM config enables a local `services.ollama` instance plus governed modelruntime with the local Ollama-compatible backend at `http://127.0.0.1:11434`.

The desktop shell does not start/stop Ollama, pull models, or expose model load/unload controls. The NixOS service owns the local Ollama process in the canonical VM. The operator pulls a model from the terminal, then uses the Chat surface's model refresh path. `forge-core` discovers local Ollama models through governed modelruntime list/scan calls and marks them as non-managed local runtime models.

To change the Ollama service posture later, make an explicit NixOS configuration change and review it as host configuration. Do not use `curl | sh` installers on FORGE-OS.

## Troubleshooting

If an app fails to launch:

- confirm the VM imports `nix/nixos/profiles/forge-native-desktop-runtime.nix`
- confirm the native runtime profile composes `nix/nixos/profiles/forge-operator-desktop.nix`
- confirm `forge-operator-toolbelt` is installed in `environment.systemPackages`
- launch a terminal and check that the wrapper exists with `command -v forge-operator-core-status`
- for Ollama, check `command -v ollama`, `ollama list`, and then use the fixed `Ollama Status` launcher
- for chat, confirm `OLLAMA_BASE_URL=http://127.0.0.1:11434`, pull a local model from the terminal if needed, refresh Chat models, and select the discovered `ollama_compat` model
- if graphical login fails, switch to TTY and run `forge-operator-session` as the recovery path

Core operator tools are required by the Nix package. Platform-specific tools,
such as GPU telemetry, are optional and may be absent.

## Safety Boundary

The operator desktop is visibility and workstation ergonomics. Rebuilds, service lifecycle changes, model lifecycle changes, cleanup, and canonical semantic writes must remain governed by their existing FORGE/NixOS authority paths. Host shutdown/reboot is the only current direct host-power path and remains an explicit disabled-by-default dangerous capability through `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`.
