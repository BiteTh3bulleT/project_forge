# FORGE Operator Desktop

Status date: 2026-05-11.

The FORGE operator desktop is an opt-in NixOS session profile for running the FORGE graphical shell as an operator workstation. NixOS remains the substrate. FORGE remains the shell/operator surface. The profile is defined at `nix/nixos/profiles/forge-operator-desktop.nix`.

## Launch Model

The intended session chain is:

```text
TTY or display-manager session
-> forge-operator-session
-> labwc
-> forge-shell-session
-> forge-desktop-shell
-> local forge-core
```

`FORGE_SHELL_BINARY` remains a development fallback for non-operator shell
sessions. The `operator-desktop` path rejects that override and requires the
Nix-packaged `forge-desktop-shell`.

## Nix-First VM

The canonical local VM output is:

```bash
nix build .#nixosConfigurations.forge-operator-vm.config.system.build.vm
./result/bin/run-forge-operator-vm-vm
```

That VM target imports the FORGE-OS module and the operator desktop profile. It
is the preferred reproducible bring-up path because it includes the daemon,
desktop shell, session wrapper, operator toolbelt, and safe defaults in one
NixOS system closure.

The session remains safe by default:

- no automatic login
- no autostart replacement of an existing desktop
- no direct system-control authority
- no model mutation authority
- no semantic memory write authority
- no FORGE-K live authority migration

## Operator Apps

The desktop Start menu is an allowlist. It exposes categorized launchers:

- Workspace: terminal, files, editor, archive manager
- Internet: browser and fixed core health/API probe
- AI Runtime: Ollama status and governed modelruntime status
- System: process monitor, logs, network diagnostics, hardware diagnostics
- Developer: SQLite browser and Git UI
- FORGE: local FORGE status

CLI tools are launched through fixed `forge-operator-*` wrappers in a terminal. The UI passes only an allowlisted app ID to Tauri; it does not accept arbitrary command text or user-provided launch arguments.

## Runtime Services

Ollama is installed through Nix as part of the operator toolbelt, not by a runtime installer. The profile provides the `ollama` CLI on `PATH`, but this pass does not enable an Ollama service by default and does not add model load/unload controls.

To enable an Ollama service later, add an explicit NixOS configuration change and review it as host configuration. Do not use `curl | sh` installers on FORGE-OS.

## Troubleshooting

If an app fails to launch:

- confirm the profile imports `nix/nixos/profiles/forge-operator-desktop.nix`
- confirm `forge-operator-toolbelt` is installed in `environment.systemPackages`
- launch a terminal and check that the wrapper exists with `command -v forge-operator-core-status`
- for Ollama, check `command -v ollama` and then use the fixed `Ollama Status` launcher

Core operator tools are required by the Nix package. Platform-specific tools,
such as GPU telemetry, are optional and may be absent.

## Safety Boundary

The operator desktop is visibility and workstation ergonomics. It is not host mutation authority. Rebuilds, service lifecycle changes, model lifecycle changes, cleanup, and canonical semantic writes must remain governed by their existing FORGE/NixOS authority paths.
