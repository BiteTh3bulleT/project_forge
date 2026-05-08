# FORGE Host Kernel Bridge

The Host Kernel Bridge is a read-only diagnostic boundary between FORGE and the Linux/NixOS host substrate.

Phase N2 defined the bridge architecture. Phase N3 implements a read-only diagnostic snapshot library at `services/core/internal/hostbridge`.

The bridge does not add host mutation, kernel mutation, automatic rebuilds, public routes, or live authority migration.

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

## Phase N3 Implementation

The Phase N3 implementation generates a bounded `Snapshot` with these top-level fields:

- `snapshot_id`
- `captured_at`
- `host`
- `kernel`
- `boot`
- `cpu`
- `memory`
- `disk`
- `gpu`
- `thermal`
- `services`
- `modelruntime`
- `degraded`
- `warnings`
- `redactions`
- `source_errors`

Each source fails independently. Missing `/proc` files, unavailable `nvidia-smi`, absent thermal sysfs, unavailable `systemctl`, and missing modelruntime health providers produce unavailable/degraded source records without failing the whole snapshot.

Snapshot persistence is an explicit helper that writes JSON under the configured report directory, normally `/forge/runtime/host-kernel`. It does not write semantic memory and is not called from core startup by default.

## Diagnostic Sources

Implemented sources include:

| Source | Diagnostic Use | Sensitivity Handling |
|---|---|---|
| `/proc/version` | Kernel version | Store normalized version |
| `/proc/cmdline` | Boot parameters | Redact secrets and tokens |
| `/proc/modules` | Loaded modules | Store module names and versions |
| `/proc/meminfo` | Memory pressure | Store aggregates |
| `/proc/loadavg`, `/proc/stat` | CPU pressure | Store aggregates |
| storage root statfs | Disk pressure | Store aggregate totals and pressure level |
| `/sys/class/thermal` | Thermal state | Store sensor labels and temperatures |
| `systemctl show forge-core.service` | Unit state | Store unit name and health summary |
| `nvidia-smi` when available | GPU/VRAM status | Store aggregate driver/device identity and VRAM totals |
| optional internal provider | modelruntime health | Store health state only |

The bridge should prefer metadata and hashes over raw sensitive payloads.

## Redaction

Boot parameters redact values whose keys or values look like passwords, tokens, secrets, credentials, keys, auth headers, bearer values, or URLs with embedded credentials. Redaction records cite the source and reason without retaining the sensitive value.

## Read-Only Command Boundary

Phase N3 may run only bounded read-only commands:

- `nvidia-smi --query-gpu=... --format=csv,noheader,nounits`
- `systemctl show forge-core.service --property=Id,LoadState,ActiveState,SubState --no-pager`

It must not run rebuild, package upgrade, service control, module load/unload, destructive filesystem, gateway, retrieval, embedding, or modelruntime execution commands.

## Resource Policy Consumer

Phase N4 adds FORGE-H resource policy under `services/core/internal/forgeh`. FORGE-H consumes Host Kernel Bridge snapshots as diagnostic input and produces advisory resource posture, lane decisions, model-load recommendations, background-work recommendations, warnings, and operator actions.

FORGE-H may classify and recommend. It does not make Host Kernel Bridge authoritative, execute host actions, call modelruntime, write semantic memory, or alter live responses.

## Diagnostics Are Not Authority

Host observations can explain operational state but do not admit evidence, commit memory, execute tools, restart services, or change live responses. If a future phase adds control, every action must be explicit, bounded, approved where required, and journaled through existing FORGE authority paths.

## Failure Mode

If host observation fails, FORGE should report an unavailable diagnostic source and continue in degraded mode. Host observation failure must not prevent safe core startup.
