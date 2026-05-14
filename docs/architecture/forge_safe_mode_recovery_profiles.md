# FORGE Safe Mode and Recovery Profiles

Status: DESIGN_ONLY / PLANNED / NO_LIVE_AUTHORITY_CHANGE

## Purpose

FORGE Workstation needs explicit recovery postures before host substrate work can safely expand. These profiles describe intended NixOS/operator modes. They do not enable autologin, replace the default desktop, mutate host state, or change live modelruntime behavior in M5.

## Profile Matrix

| Profile | Enabled services | Disabled services | Modelruntime posture | GPU posture | Shell posture | Network posture | Rollback method | Operator command path |
|---|---|---|---|---|---|---|---|---|
| FORGE Normal | `forge-core`, Tauri shell, governed modelruntime, configured storage, optional telemetry. | Nothing disabled by profile alone. | Configured backend profiles. | Optional and policy-gated. | Normal operator desktop or selected FORGE shell. | Normal configured local/network policy. | Nix generation rollback or manual config revert. | Existing `npm run up`, `npm run desktop`, or enabled NixOS service/session. |
| FORGE Safe Mode | `forge-core`, TTY access, minimal shell/status, SQLite. | GPU background work, risky adapters, host mutation adapters, optional remote services. | CPU-only/degraded; no GPU-required runtime. | Off/unavailable. | Minimal UI or TTY-first. | Local-first; remote optional disabled unless operator chooses. | Boot fallback generation or disable FORGE shell/session. | `FORGE_SAFE_MODE_FORCE_CPU_ONLY=true` plus normal core startup or NixOS specialisation. |
| FORGE CPU Only | `forge-core`, desktop optional, CPU-capable modelruntime profile. | GPU runtime classes, GPU background jobs. | `cpu_safe` or CPU-capable local profile. | Off/deferred. | Normal shell or TTY. | Normal configured. | Env/profile switch; no data migration. | Set CPU-only env/profile and restart governed service manually. |
| FORGE GPU Runtime | `forge-core`, modelruntime, GPU telemetry, selected GPU backend endpoint. | Unsafe CUDA mutation, unmanaged kernel launches, direct modelruntime bypass. | Governed GPU profile such as `interactive_vllm`. | On when available and healthy. | Normal shell with read-only GPU posture. | Normal configured. | Disable GPU profile or rollback generation. | Operator enables profile in NixOS/env; no shell-side command. |
| FORGE Debug Shell | TTY, logs, manual core start, diagnostic commands chosen by operator. | Autostart shell, nonessential background jobs, optional modelruntime. | Manual only. | Off unless explicitly selected. | TTY/manual shell launch. | Operator-selected. | Manual Nix generation rollback. | TTY login, inspect logs, start `forge-core` or `forge-shell-session` manually. |
| FORGE Recovery | TTY, rollback tools, data backup/export access. | FORGE autostart, graphical shell autostart, modelruntime, GPU runtime. | Disabled. | Off. | TTY only. | Minimal; enough for operator recovery. | NixOS rollback, disable module imports, preserve `/forge`. | Boot previous generation or disable FORGE modules from host config. |

## Required Preservation

- TTY fallback.
- Non-FORGE desktop fallback during adoption.
- No autologin by default.
- No default desktop replacement.
- No destructive cleanup.
- No shell mutation controls.
- No direct `systemctl` or `nixos-rebuild` from FORGE UI/model paths.
- No FORGE-K live authority expansion.

## Promotion Requirements

Before any profile becomes a live NixOS specialization, it needs:

- explicit module/profile file;
- VM boot evidence;
- rollback instructions;
- operator runbook;
- proof that TTY fallback remains available;
- proof that non-FORGE desktop fallback is not removed by default;
- no host mutation path from Tauri shell.
