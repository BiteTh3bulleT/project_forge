# ADR 0007 - FORGE-OS Host Substrate

Status: Accepted

Date: 2026-05-07

## Context

FORGE is the AI-OS operating environment. Linux remains the hardware and boot substrate, and NixOS is the preferred future host configuration substrate for reproducible private deployments.

FORGE already owns the higher-level AI-OS responsibilities: runtime orchestration, governed memory/state, context and journal boundaries, approvals, audit, model runtime governance, and the desktop shell direction. FORGE-K simulator and shadow phases are not live daemon authority unless a later explicit live integration phase changes that boundary.

Phase N2 prepares the host substrate foundation. It must not become a Linux fork, a live authority migration, or an autonomous host mutation path.

## Decision

Define the FORGE-OS Host Substrate as a layered private host profile:

- hardware
- Linux kernel
- NixOS host configuration
- FORGE services
- FORGE-K, FORGE-H, model runtime, memory, journal, governance layers
- FORGE shell

NixOS modules may now exist as scaffolding under `nix/nixos/modules`. They can declare options, users, groups, directory layout, and a default-safe `forge-core` systemd service shape.

The Host Kernel Bridge is read-only diagnostics in this phase. It may observe host and kernel metadata, but it must not mutate host state, execute system rebuilds, change kernel settings, or bypass existing FORGE gateway, permissions, lane, audit, controllane, or modelruntime authority paths.

## Boundaries

- Linux/NixOS owns boot, hardware initialization, kernel module loading, and normal host service management.
- FORGE owns AI-OS runtime behavior above the host substrate.
- `/forge` is the intended durable storage root for private FORGE host deployments.
- Host observation is diagnostic evidence only until a later approved control phase.
- Existing live daemon authority paths remain unchanged.

## Non-Goals

- No Linux fork.
- No public distro.
- No live FORGE-K authority migration.
- No autonomous host mutation.
- No automatic NixOS rebuilds.
- No route, gateway, controllane, memory, modelruntime, or public API behavior changes.
- No second memory or state authority path.

## Risks

- Host controls could become an unsafe side channel if diagnostics are later connected to mutation without approvals.
- NixOS module defaults could be mistaken for a production migration.
- `/forge` storage could drift from existing `FORGE_DATA_DIR` usage if not mapped carefully.
- Model runtime and GPU failures could block boot if service dependencies are too strict.

## Consequences

Positive:

- FORGE has a clear private host-substrate path without forking Linux.
- `/forge` storage ownership becomes explicit for future host deployments.
- NixOS module work can proceed incrementally while non-Nix workflows remain authoritative.
- Host diagnostics have a documented read-only boundary before any future control path exists.

Negative:

- NixOS module scaffolding adds another deployment surface to maintain.
- Operators may mistake module availability for a completed host migration.
- Future control phases will need stricter tests to prevent host mutation from bypassing existing approval and audit boundaries.

## Alternatives Considered

### Keep FORGE As An App Only

Rejected as the long-term direction. It does not match the FORGE AI-OS goal, though the app-only workflow remains valid during N2.

### Wait Until Full NixOS Modules Are Production-Ready

Rejected. Scaffolding now gives future phases a concrete contract while keeping behavior disabled and safe.

### Fork Linux Or Build A Bare-Metal Kernel

Rejected. Linux/NixOS remains the boot and hardware substrate.

### Container-Only Substrate

Rejected as the sole direction. Containers are useful for services and databases, but they do not define the full host boot, shell, storage, and diagnostics substrate.

## Rollback Strategy

The NixOS modules are opt-in scaffolding. Rollback is to remove the module imports from the host configuration, stop any manually enabled `forge-core` systemd unit, and continue using existing repository startup commands such as `npm run up` and `npm run down`.

No live daemon behavior changes are introduced by this ADR.
