# FORGE Host Kernel Bridge

The Host Kernel Bridge is a read-only diagnostic boundary between FORGE and the Linux/NixOS host substrate.

Phase N2 defines the bridge architecture only. It does not add host mutation, kernel mutation, automatic rebuilds, or live authority migration.

## Purpose

The bridge gives FORGE structured host evidence for operator awareness and future planning:

- kernel version
- boot parameters
- loaded modules
- CPU, RAM, disk, and swap pressure
- GPU and VRAM status when available
- thermal state when available
- running and failed services
- modelruntime health

## Read-Only Boundary

Allowed:

- read stable host metadata
- normalize diagnostics into refs, summaries, and bounded records
- report degraded host conditions
- recommend operator actions
- preserve provenance for observations

Forbidden in Phase N2:

- changing kernel parameters
- loading or unloading modules
- restarting services
- running package upgrades
- executing `nixos-rebuild`
- mutating `/forge` state outside documented service ownership
- calling model runtimes as authority
- writing semantic memory directly
- bypassing gateway, permissions, lanes, audit, or controllane paths

## Diagnostic Sources

Potential future sources include:

| Source | Diagnostic Use | Sensitivity Handling |
|---|---|---|
| `/proc/version` | Kernel version | Store normalized version |
| `/proc/cmdline` | Boot parameters | Redact secrets and tokens |
| `/proc/modules` | Loaded modules | Store module names and versions |
| `/proc/meminfo` | Memory pressure | Store aggregates |
| `/proc/stat` | CPU pressure | Store aggregates |
| `/sys/class/thermal` | Thermal state | Store sensor labels and temperatures |
| systemd DBus or `systemctl show` | Unit state | Store unit name and health summary |
| GPU tools | GPU/VRAM status | Store aggregate utilization and driver identity |
| FORGE health endpoints | modelruntime/core health | Store health state and correlation IDs |

The bridge should prefer metadata and hashes over raw sensitive payloads.

## Diagnostics Are Not Authority

Host observations can explain operational state but do not admit evidence, commit memory, execute tools, restart services, or change live responses. If a future phase adds control, every action must be explicit, bounded, approved where required, and journaled through existing FORGE authority paths.

## Failure Mode

If host observation fails, FORGE should report an unavailable diagnostic source and continue in degraded mode. Host observation failure must not prevent safe core startup.
