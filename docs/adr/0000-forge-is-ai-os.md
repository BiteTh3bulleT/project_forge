# ADR 0000 - FORGE Is the AI-OS

- Date: 2026-04-18
- Status: accepted

## Context

FORGE already has kernel-like controls and execution doctrine:

- events are truth
- jobs are projections
- packets are contracts
- approvals are gates
- artifacts are evidence
- adapters are bounded workers

The current codebase also already contains core OS-like boundaries (`gateway`, `action_lanes`, `permission_profiles`, `audit_records`) and memory/retrieval services. What is missing is a single architecture decision that establishes FORGE as the operating substrate, not just a toolchain.

At the same time, IRIS work is planned. Without an explicit boundary, IRIS could be interpreted as owning state or canonical truth, which would violate current FORGE guarantees around deterministic validation, auditable writes, and workspace-scoped execution controls.

## Decision

FORGE is the AI-OS. IRIS is a future resident semantic service inside FORGE.

1. FORGE owns canonical memory/state authority and commit authority.
2. All durable state mutation must pass through FORGE validation paths (APIs/syscalls, approvals, permissions, audit).
3. LLM/semantic systems, including future IRIS, operate as proposers in user space.
4. FORGE control logic remains deterministic and auditable.
5. Derived policy/model outputs remain advisory unless committed through FORGE validation.

Operational rule:

> IRIS proposes. FORGE validates. FORGE commits.

## Why FORGE must own memory/state authority

1. Determinism: kernel validation must be replayable and inspectable independent of model variance.
2. Auditability: durable writes require traceable approval, permission, and correlation records.
3. Workspace safety: path/risk boundaries are enforced by FORGE lanes and permission profiles.
4. Contract integrity: jobs, packets, approvals, and artifacts are linked evidence; bypass writes would break these chains.
5. Multi-service future: plugin/semantic services can evolve independently only if commit authority stays centralized.

## Consequences

Positive:

- clear architectural ownership for runtime, memory, and commit paths
- reduced risk of semantic-service bypass of permissions/approval gates
- stable foundation for future IRIS integration and plugin support
- stronger separation of exact evidence from adaptive derived policy

Tradeoffs:

- more explicit validation interfaces are required as features expand
- semantic systems may need additional translation layers into FORGE syscall contracts
- some existing services will be refactored into clearer Control/I-O/Compute lane boundaries

## Non-goals

- building IRIS in this ADR
- replacing existing FORGE doctrine or data model
- granting direct state mutation rights to LLMs/adapters/IRIS
- introducing a new parallel architecture outside current FORGE modules

## How this affects IRIS

IRIS will be integrated as a FORGE-native semantic service seam, not as a kernel replacement.

IRIS responsibilities:

- propose semantic interpretations, rankings, and candidate actions
- provide derived meaning and recommendations

FORGE responsibilities:

- validate proposed actions against lanes/capabilities/approvals/workspace scope
- enforce deterministic state transitions
- write canonical state and audit records

Any IRIS output that should affect durable state must be converted into a FORGE contract (packet/job/syscall request) and committed through FORGE-owned paths.
