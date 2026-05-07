# FORGE-OS Boot Flow

Phase N2 defines the desired private host boot flow for a future NixOS-based FORGE-OS profile.

## Normal Boot

1. Power on.
2. Firmware initializes hardware.
3. Linux kernel boots.
4. NixOS activates the declared host configuration.
5. `/forge` storage directories are available.
6. FORGE service dependencies start.
7. `forge-core` starts with `FORGE_DATA_DIR=/forge/data`.
8. Model runtime services start or report degraded status.
9. FORGE memory, journal, audit, and governance surfaces become available.
10. FORGE shell becomes the operator surface.

## Degraded Boot

| Condition | Expected Behavior |
|---|---|
| No GPU | Boot CPU-safe mode; modelruntime reports degraded accelerator status |
| Model runtime unavailable | Core remains available; runtime-dependent actions fail closed or report unavailable |
| Database unavailable | Core reports storage degradation; no silent memory writes |
| Vector store unavailable | Retrieval reports degraded; canonical memory authority remains separate |
| `/forge` unavailable | Service should fail clearly or use an explicitly configured fallback |
| Host diagnostics unavailable | Host Kernel Bridge reports unavailable source; no boot block |
| Policy/config invalid | Fail closed and require operator correction |

## Safe Mode

FORGE safe mode means:

- core starts with minimal dependencies where possible
- no autonomous host mutation
- no modelruntime authority escalation
- no semantic writes outside governed paths
- diagnostics and operator recovery surfaces remain visible where possible

## Ordering Principles

- Boot must not depend on FORGE-K simulator becoming live authority.
- Model runtimes are drivers and may be unavailable.
- Host observation is diagnostic and should not block core startup.
- Durable state must remain under explicit `/forge` ownership or configured fallback paths.

## Rollback

Disable the NixOS FORGE module or set `services.forge.core.enable = false`. Existing repository bring-up scripts remain the non-Nix fallback.
