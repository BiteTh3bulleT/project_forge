# FORGE-K Control Lane Enforcement Design

Date: 2026-05-11

Status: design approved for planning

## Purpose

Start turning FORGE-K on by promoting the existing live Control Lane validation seams into a stronger operational boundary. This phase does not make the FORGE-K simulator the live kernel. It makes selected FORGE-K doctrine enforceable through the current live owner: `services/core/internal/aios/controllane`.

The first activation target is deterministic validation enforcement for semantic and ref-shaped operations:

- `VALIDATE_REF_SHAPE`
- `COMPARE_REF_SHAPE`
- `VALIDATE_SEMANTIC_OPERATION`
- existing `VALIDATE_KV_IDENTITY` posture remains in scope for consistency checks, not live KV reuse

## Context

The repository already has the approved migration pattern:

1. Keep simulator services under `services/core/internal/forgek` simulator-owned.
2. Extract deterministic contracts into live-safe shared packages such as `refvalidation`, `semanticvalidation`, and `kvidentity`.
3. Invoke those contracts from the live Control Lane, where semantic validation already belongs.
4. Record audit and bounded diagnostics without changing route, gateway, memory, modelruntime, retrieval, or user-visible behavior.

This design continues that path. It does not import `forgek.Kernel`, simulator syscalls, Context Compiler, Courthouse, Memory Palace, Consensus Mesh, Runtime Boundary, KV service, or Lymphatic services into live daemon authority.

## Alternatives Considered

### Recommended: Control Lane Validation Enforcement

Promote the existing validation actions into an explicit FORGE-K activation surface with stronger policy metadata, counters, tests, docs, and operator status. This is the safest first step because the Control Lane already owns semantic validation and mutation boundaries.

Tradeoff: it is less visually dramatic than enabling a new shell or kernel process, but it gives real live enforcement without authority confusion.

### Context Compiler Mirror

Add a read-only live comparison between existing context compilation and FORGE-K context shape rules. This would be useful soon, but prompt/context authority is closer to user-visible response behavior and should wait until the validation boundary is hardened.

### Courthouse / Evidence Admission Prep

Begin live evidence admission validation. This is the real path toward FORGE-K truth authority, but it is higher risk because evidence admission and memory commit behavior are core authority surfaces. It should follow the Control Lane enforcement phase.

## Design

### Boundary

The phase is `PARTIAL LIVE ENFORCEMENT / CONTROL_LANE / NO_SIMULATOR_AUTHORITY`.

Live authority remains with:

- Control Lane for semantic validation and governed semantic mutation.
- Gateway for tool execution.
- Permissions and lanes for policy gates.
- Audit for trace records.
- Existing memory/retrieval/modelruntime paths for their current responsibilities.

FORGE-K contributes deterministic doctrine through shared pure validation packages and bounded diagnostics only.

### Behavior

The Control Lane validation actions should become an operator-visible activation surface with these properties:

- reject malformed validation payloads before any semantic mutation path can use them
- reject semantic operation envelopes that claim live authority, modelruntime mutation, memory mutation, evidence admission, or context compilation outside their current allowed phase
- keep ref shape comparison diagnostic unless a specific caller requires match-only enforcement
- emit bounded validation diagnostics when shadow validation flags are enabled
- include policy version, validator version, source, decision, reason, failure count, warning count, and no-effect fields in summaries
- expose status through existing internal or shell-readable read-only status surfaces if already present, or document it if no endpoint exists yet

No new mutation actions are added.

### Data Flow

1. A live caller submits a Control Lane syscall.
2. The existing action registry resolves the action and required capability.
3. The relevant shared pure validator runs.
4. The Control Lane returns a validation result or rejection.
5. Audit/state summary records bounded validation metadata.
6. Optional `forgekshadow` validation observer receives scalar summary data after the Control Lane result exists.
7. No shadow report or validation advisory becomes canonical truth.

### Feature Flags And Rollback

The phase should keep defaults conservative:

- existing validation actions remain available through the Control Lane
- shadow validation emission remains disabled unless `FORGE_K_SHADOW_MODE_ENABLED=true` and `FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED=true`
- any new stricter enforcement rule that could reject previously accepted traffic should be gated by an explicit config flag or introduced only for validation-only actions
- disabling the new flag restores the previous validation behavior while preserving existing audit records

Rollback is config disablement for new strictness, or commit revert if validation action behavior regresses.

### Operator Surface

The shell/system status surface should eventually show:

- FORGE-K activation mode: `simulator-only`, `partial-live-validation`, or `partial-live-enforcement`
- enabled validation actions
- last validation decision summary
- shadow validation emission enabled/disabled
- no-effect guarantees: no modelruntime mutation, no memory write, no gateway execution, no live Kernel authority

This phase may add read-only status if a safe internal status route already exists. It must not add public mutation routes.

## Non-Goals

- Do not import FORGE-K simulator services as live authority.
- Do not replace the live Control Lane.
- Do not route live semantic writes through `forgek.Kernel`.
- Do not enable live KV reuse.
- Do not compile live prompts through the FORGE-K Context Compiler.
- Do not admit evidence through the FORGE-K Courthouse.
- Do not write semantic memory directly.
- Do not call modelruntime from FORGE-K.
- Do not execute tools from FORGE-K.
- Do not change public route behavior unless a later implementation plan explicitly scopes a read-only status endpoint.

## Testing Requirements

Implementation planning should include tests for:

- malformed ref validation fails closed
- malformed semantic operation validation fails closed
- semantic operation authority claims are rejected
- capability denial still rejects before validation success is treated as authority
- audit/state summaries include bounded FORGE-K activation metadata
- shadow validation observer remains best-effort and cannot change Control Lane decisions
- disabled flags preserve previous behavior
- no `services/core/internal/forgek` simulator service imports from live validation packages
- no gateway/tool execution
- no modelruntime call
- no retrieval/search/embedding execution
- no memory write
- no public unauthenticated mutation route

## Documentation Updates

The implementation phase should update:

- `docs/architecture/simulator_to_live_migration.md`
- `docs/architecture/forge_k_operational_cutover_design.md`
- `docs/architecture/control_lane_kernel.md`
- `docs/reviews/current_phase_status.md`

The docs should mark this phase as partial live enforcement through the Control Lane, not full FORGE-K live authority.

## Acceptance Criteria

- FORGE-K has a clearly named partial live enforcement mode.
- The mode is implemented through existing Control Lane authority.
- Existing deterministic validation seams are harder to bypass and better reported.
- No simulator service becomes live authority.
- No memory, modelruntime, gateway, retrieval, or route mutation is introduced.
- Rollback is documented and tested.
- Existing touched Go tests pass.
