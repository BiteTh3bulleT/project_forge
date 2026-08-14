# ADR Index

Status date: 2026-08-14.

This index records current ADR status so historical architecture decisions are
not mistaken for stale or conflicting authority.

## Active Decisions

| ADR | Status | Scope |
| --- | --- | --- |
| [0000 - FORGE Is the AI-OS](0000-forge-is-ai-os.md) | Accepted | FORGE owns AI-OS authority; IRIS/model systems propose only. |
| [0001 - FORGE-K Is a Cognitive Microkernel](0001-forge-k-is-a-cognitive-microkernel.md) | Accepted | Target cognitive microkernel doctrine. |
| [0002 - Models Are Drivers, Not Authority](0002-models-are-drivers-not-authority.md) | Accepted | Model output is proposal/evidence, not canonical authority. |
| [0003 - Snapshots Are Shape, Not Truth](0003-snapshots-are-shape-not-truth.md) | Accepted | Snapshots preserve restorable shape only. |
| [0004 - KV Cache Is Acceleration, Not Memory](0004-kv-cache-is-acceleration-not-memory.md) | Accepted | KV reuse is deterministic acceleration metadata only. |
| [0005 - FORGE-K Simulator vs Live Authority](0005-forge-k-simulator-vs-live-authority.md) | Accepted | FORGE-K simulator is not live daemon authority. |
| [0006 - Rust Kernel Core Boundary](0006-rust-kernel-core-boundary.md) | Accepted | Rust boundary is deterministic primitives only. |
| [0007 - FORGE-OS Host Substrate](0007-forge-os-host-substrate.md) | Accepted | Linux/NixOS substrate and `/forge` durable root direction. |
| [0011 - FORGE Graphical Shell Session](0011-forge-graphical-shell-session.md) | Accepted | Graphical shell session direction above NixOS. |
| [0012 - FORGE Wayland Shell Session](0012-forge-wayland-shell-session.md) | Accepted | Opt-in Wayland/Cage session path. |
| [0013 - FORGE G6 Operator Desktop](0013-forge-g6-operator-desktop.md) | Accepted | Opt-in labwc operator desktop path; Cage remains rollback. |
| [0014 - FORGE Workstation Substrate and Nix Mutation Proposals](0014-forge-workstation-substrate-and-nix-mutation-proposals.md) | Accepted | Workstation substrate and advisory Nix mutation proposal boundary. |
| [0015 - Ref Model Unification](0015-ref-model-unification.md) | Accepted | Shared deterministic ref contracts across simulator and production validation seams. |
| [0016 - FORGE AXIOM Cognition Engine](0016-forge-axiom-cognition-engine.md) | Accepted | AXIOM proposal and deterministic cognition boundary. |
| [0017 - FORGE-K Production Authority Cutover](0017-forge-k-production-authority-cutover.md) | Accepted; K20A-K20B active | Staged single-authority production cutover from Control Lane v1 to FORGE-K. |

## Superseded Decisions

None currently.

Later ADRs refine and extend earlier decisions, but no ADR in this directory is
currently superseded. If a future decision replaces one, update this index and
add a clear supersession note near the top of the superseded ADR.
