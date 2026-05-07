# Host Kernel Bridge Diagnostics

Phase N3 implements the Host Kernel Bridge as a read-only Go package at `services/core/internal/hostbridge`.

The bridge generates bounded host diagnostic snapshots. It does not add a public route, does not run at core startup by default, does not execute tools through the gateway, does not call model runtimes, and does not write semantic memory.

## What It Collects

- host architecture, hostname, and OS release when available
- kernel version from `/proc/version`
- boot parameters from `/proc/cmdline`
- loaded module names/refcounts from `/proc/modules`
- CPU count, load average, and aggregate utilization estimate
- memory and swap totals plus pressure level
- configured FORGE storage root disk totals plus pressure level
- NVIDIA GPU/VRAM metadata when `nvidia-smi` is available
- thermal sensor labels and temperatures from `/sys/class/thermal`
- `forge-core.service` state through `systemctl show` when systemd is available
- optional modelruntime health from an internal provider if one is supplied

Every source is best-effort. Missing sources become `source_errors` and unavailable/degraded source sections.

## Redaction

Boot parameters redact values whose keys or values look like:

- passwords
- tokens
- secrets
- credentials
- API keys
- private keys
- auth/bearer values
- URLs with embedded credentials

The snapshot records redaction source, field, and reason. It does not retain the sensitive value.

## Manual Developer Check

Run the focused package tests:

```bash
cd services/core && go test ./internal/hostbridge -count=1
```

Run the broader guard set before changing host diagnostics:

```bash
cd services/core && go test ./internal/hostbridge ./internal/forgek/... ./internal/forgekshadow/... -count=1
npm run test:core
npm run build:core
npm run lint
```

## Report Directory

The NixOS scaffold reserves:

```text
/forge/runtime/host-kernel
```

Snapshot JSON writing is explicit. Generating a snapshot does not write files by itself.

## Read-Only Boundary

Allowed command forms are limited to bounded diagnostic reads such as:

- `nvidia-smi --query-gpu=... --format=csv,noheader,nounits`
- `systemctl show forge-core.service --property=Id,LoadState,ActiveState,SubState --no-pager`

Forbidden:

- `nixos-rebuild`
- service restart/start/stop
- module load/unload
- package upgrades
- destructive filesystem operations
- gateway execution
- retrieval/search/embedding execution
- modelruntime calls
- direct semantic memory writes

Host diagnostics are operational evidence only. They are not canonical semantic truth.
