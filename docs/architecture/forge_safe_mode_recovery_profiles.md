# FORGE Safe Mode and Recovery Profiles

Status: DESIGN_ONLY / PLANNED

| Profile | Enabled | Disabled | Modelruntime | GPU | Shell | Network | Rollback |
|---|---|---|---|---|---|---|---|
| FORGE Normal | core, shell, governed runtime | none by default | configured profiles | optional | operator desktop | normal configured | Nix generation/manual |
| FORGE Safe Mode | core, TTY, minimal shell | GPU background work, risky adapters | CPU-only/degraded | off | minimal | local-first | boot fallback generation |
| FORGE CPU Only | core, desktop optional | GPU runtime classes | CPU-safe only | off | normal or TTY | normal configured | env/profile switch |
| FORGE GPU Runtime | core, modelruntime, telemetry | unsafe CUDA mutation | governed GPU profiles | on if available | normal | normal configured | disable profile or rollback |
| FORGE Debug Shell | TTY, logs, core manual start | autostart shell | manual only | off unless selected | TTY/manual | operator-selected | manual generation rollback |
| FORGE Recovery | TTY, rollback tools | FORGE autostart, modelruntime | disabled | off | TTY | minimal | NixOS rollback |

## Required Preservation

- TTY fallback.
- Non-FORGE desktop fallback during adoption.
- No autologin by default.
- No default desktop replacement.
- No destructive cleanup.
- No shell mutation controls.
