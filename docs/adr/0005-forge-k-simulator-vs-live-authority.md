# ADR 0005 - FORGE-K Simulator vs Live Authority

Status: Accepted; amended by ADR 0017 / K20A

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
- `LIVE_INTEGRATION`: work touches live daemon paths. Read-only diagnostics must explicitly declare `READ_ONLY` and remain disabled by default. Live authority changes or live state mutation require an authority migration plan, integration design, and tests.
- `DOCS_ONLY`: work changes documentation/status/planning only.
- `RESEARCH_ONLY`: work is exploratory and cannot be treated as production authority.

## Consequences

- Phase 1-5 completion claims mean simulator implementation and tests, not live daemon enforcement.
- Phase 6 Snapshots can proceed only after its scope is recorded and its tests are defined. The current recorded scope is `SIMULATOR_ONLY`.
- Live daemon changes must continue to respect existing gateway, permissions, lane, audit, controllane, and model runtime boundaries unless a `LIVE_INTEGRATION` phase explicitly changes them.
- Status docs must distinguish FORGE-K simulator phase status from legacy/live AI-OS implementation status.

## Amendment Notes

These notes clarify later work without changing the accepted decision above.

- Later simulator phases added Snapshots, Context Compiler, KV System, Runtime Boundary, Lymphatic Lane, Consensus Mesh, integration readiness, and simulator shadow contracts. Those packages remain simulator or contract authorities unless a later live migration explicitly says otherwise.
- Phase 12 introduced disabled-by-default live shadow diagnostics in `services/core/internal/forgekshadow`. Phase 12 diagnostics are `LIVE_INTEGRATION / READ_ONLY`; they observe bounded metadata and produce diagnostic/advisory reports only. They do not admit evidence, compile live context, execute tools, call modelruntime, run retrieval/search/embeddings, write memory, change routes, or affect user-visible output.
- Phase 14 introduced partial live Control Lane validation seams through shared pure validator packages and existing live owners. Actions such as `VALIDATE_KV_IDENTITY`, `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, `VALIDATE_SOURCE_OBJECT_AUTHORITY`, and `VALIDATE_SEMANTIC_OPERATION` are validation/enforcement seams only. They do not import simulator services as live authority or execute simulator syscalls.
- Readiness/status surfaces may display blocked gates for Courthouse admission integration, live Context Compiler authority, governed semantic mutation routing, runtime driver authority, and broader Kernel authority. Displaying a gate does not grant mutation authority or live FORGE-K authority.
- Any future promotion beyond read-only diagnostics or validation-only seams still requires the integration design, authority migration plan, tests, rollback, and documentation updates required by this ADR.
- K20A (2026-08-14) satisfies that requirement for the first production boundary. The distinct `services/core/internal/forgekernel` package now owns semantic syscall ingress by default, with `aios/controllane` retained as a single temporary durable commit adapter and `legacy_v1` as rollback-only. This amendment does not promote the in-memory simulator under `services/core/internal/forgek`; all simulator-isolation rules in this ADR remain active. See ADR 0017.

## Alternatives considered

- Treat FORGE-K as live authority now: rejected because the live daemon still uses existing authority paths and non-test live code is not wired through FORGE-K.
- Let integration happen opportunistically during simulator phases: rejected because authority migration needs explicit design, tests, and rollback clarity.
- Freeze simulator work until live integration is complete: rejected because simulator-only phases can continue safely when they are labeled and kept out of live daemon mutation paths.
