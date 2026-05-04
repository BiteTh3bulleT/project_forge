# ADR 0005 - FORGE-K Simulator vs Live Authority

Status: Accepted

Date: 2026-05-04

## Context

FORGE-K Phase 1-5 is implemented and tested under `services/core/internal/forgek`. That package is a deterministic cognitive microkernel simulator for kernel syscalls, journal behavior, neuron envelopes, Courthouse admission, Memory Palace retrieval topology, and Semantic Algebra.

The live FORGE daemon still uses the existing AI-OS, gateway, permissions, lane, audit, model runtime, and API authority paths. Non-test live daemon code does not route canonical state mutation through FORGE-K yet.

Without an explicit boundary, project status can imply that FORGE-K doctrine is already enforced in live paths. That would blur simulator readiness with daemon authority and could lead future work to bypass the existing live governance paths without integration design.

## Decision

FORGE-K is the target architecture, but it is not live daemon authority yet.

The live daemon continues to use the current AI-OS/gateway/permissions/lane/audit authority paths until an explicit live integration phase changes that boundary.

No live state mutation may route through FORGE-K without an integration design, authority migration plan, tests, and documentation updates that identify the affected live paths.

Every future FORGE-K phase must declare one scope marker before work starts:

- `SIMULATOR_ONLY`: work is confined to the FORGE-K simulator, docs, and tests; live daemon authority is unchanged.
- `LIVE_INTEGRATION`: work intentionally changes live daemon authority or routes live state through FORGE-K boundaries, with integration design and tests required.
- `DOCS_ONLY`: work changes documentation/status/planning only.
- `RESEARCH_ONLY`: work is exploratory and cannot be treated as production authority.

## Consequences

- Phase 1-5 completion claims mean simulator implementation and tests, not live daemon enforcement.
- Phase 6 Snapshots can proceed only after its scope is recorded and its tests are defined. The current recorded scope is `SIMULATOR_ONLY`.
- Live daemon changes must continue to respect existing gateway, permissions, lane, audit, controllane, and model runtime boundaries unless a `LIVE_INTEGRATION` phase explicitly changes them.
- Status docs must distinguish FORGE-K simulator phase status from legacy/live AI-OS implementation status.

## Alternatives considered

- Treat FORGE-K as live authority now: rejected because the live daemon still uses existing authority paths and non-test live code is not wired through FORGE-K.
- Let integration happen opportunistically during simulator phases: rejected because authority migration needs explicit design, tests, and rollback clarity.
- Freeze simulator work until live integration is complete: rejected because simulator-only phases can continue safely when they are labeled and kept out of live daemon mutation paths.
