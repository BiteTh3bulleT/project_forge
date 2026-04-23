# Nix Tool Capsules — Scaffold Only (Phase N1)

This directory is a **placeholder** for future FORGE tool capsules:
Nix-defined execution environments that will back AI-OS tool capabilities
behind the gateway.

## Status in Phase N1

- **Not active.** No capsules are wired into the gateway.
- **Not a promise of runtime behavior.** Do not reference these
  from autonomy charters, permissions, or approvals.
- **Not exposed** from `flake.nix`.

## What capsules will be (Phase N3+)

A capsule is a hermetic Nix environment describing:

- the exact binaries/runtimes a tool invocation may see,
- the network/filesystem posture (none / read-only / sandboxed),
- resource bounds,
- a deterministic identity the gateway can reference for audit.

Planned examples:

- **code-runner** — language toolchains with no network.
- **test-runner** — project test deps + ephemeral workspace.
- **no-network-runner** — strict offline env for deterministic work.
- **memory-runner** — read-only access to the cognitive filesystem snapshot.
- **restricted-shell** — coreutils-only shell for minimal ops.

## Prerequisites before wiring

Deep integration is blocked until:

1. v1/v2 authoritative runtime path is decided
   (see `docs/architecture/v1_v2_unification_plan.md` when it lands).
2. Gateway tool-capability registry has a stable contract for
   referencing a capsule by identity.
3. Autonomy/approval surfaces can enforce capsule policy.

Until then, keep this directory documentary only.
