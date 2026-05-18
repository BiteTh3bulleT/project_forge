# ADR 0015 - Ref Model Unification Between Simulator And Pure Validators

Status: Proposed

Date: 2026-05-18

## Context

FORGE-K Phase 14 introduced pure deterministic validator packages used by the live Control Lane:

- `services/core/internal/refvalidation`
- `services/core/internal/semanticvalidation`
- `services/core/internal/contextattribution`
- `services/core/internal/admissionvalidation`
- `services/core/internal/kvidentity`

All five operate on structured ref types — primarily `refvalidation.ObjectRef{RefType, RefID, WorkspaceID, SourceRef}`.

The FORGE-K simulator under `services/core/internal/forgek/` represents source refs as opaque `[]string`. Concrete evidence:

- `services/core/internal/forgek/semantic/models.go:10` — `SourceRefs []string`
- `services/core/internal/forgek/semantic/models.go:28` — same in operation context
- `services/core/internal/forgek/contextcompiler/bundle.go:21` — `SourceRefs []string` on ContextBundle
- `services/core/internal/forgek/contextcompiler/service.go:97,125,166,184` — compile/restore paths flow string refs
- `services/core/internal/forgek/contextcompiler/serialize.go:28,55` — `addRefs(&lines, "source_refs", ...)` consumes strings
- `services/core/internal/forgek/contextcompiler/hashing.go` — hashing depends on the string representation

Today, the only seam actually shared between simulator and live is `VALIDATE_KV_IDENTITY` via `kvidentity`. That seam works because `kvidentity.ManifestIdentity` and `kvidentity.RequestIdentity` are structured records that the simulator adapts to in `services/core/internal/forgek/kv/gates.go:54-87`.

The other four pure validator packages are imported by the live Control Lane (`services/core/internal/aios/controllane/`) and the disabled-by-default `services/core/internal/forgekshadow/` observer. They are not imported by the simulator because the simulator's string refs cannot pass through their `ObjectRef`-typed signatures without lossy conversion.

This is a structural blocker for future live integration of FORGE-K. Routing live mutation through the simulator cannot use the deterministic validator packages as a single boundary check because the simulator path would re-implement validation against a different ref representation, defeating the "shared deterministic validation" property.

ADR 0005 marker for this work: `LIVE_INTEGRATION` at the simulator/live seam boundary but `SIMULATOR_REFACTOR` in effect. Live daemon authority is unchanged by this ADR.

## Decision

Migrate FORGE-K simulator ref representations from opaque `[]string` to structured `refvalidation.ObjectRef`. The simulator becomes a consumer of the same pure validator packages the live Control Lane already uses.

Migration scope (simulator side only):

- `services/core/internal/forgek/semantic/models.go` — SemanticOperation source, derived, provenance refs.
- `services/core/internal/forgek/contextcompiler/bundle.go` — ContextBundle source and observation refs.
- `services/core/internal/forgek/contextcompiler/service.go` — Compile request and result refs.
- `services/core/internal/forgek/snapshots/` — Snapshot recommended and source refs.
- `services/core/internal/forgek/court/` — Exhibit and evidence ref handling.
- `services/core/internal/forgek/lymphatic/` — Orphan ref sets.
- `services/core/internal/forgek/objects.go`, `services/core/internal/forgek/types.go` — Shared simulator types.
- Persistence and serialization paths in `services/core/internal/forgek/contextcompiler/serialize.go`, `services/core/internal/forgek/contextcompiler/hashing.go`, and snapshot persistence.

Live Control Lane and live AI-OS daemon paths are not part of this migration. Pure validator packages keep their current API.

Migration constraints:

- Snapshot and journal serialization must remain replay-compatible OR include a versioned schema migration with explicit replay tests. Existing simulator snapshots/journals must either be regeneratable from sources or accompanied by upgrade tests.
- Hashing functions used for snapshot and bundle identity must produce stable hashes for equivalent ObjectRef shapes. Canonical ordering is the `refvalidation` sort order: workspace_id, then ref_type, then ref_id (see `services/core/internal/refvalidation/models.go:116-124`).
- Determinism tests in the simulator (e.g., `services/core/internal/forgek/lymphatic/lymphatic_test.go:106-120`) must continue to pass after migration.
- `services/core/internal/forgek/fixture_parity_test.go` baselines will require regeneration; the migration commit must include the new fixtures.
- The migration introduces no live daemon mutation through simulator paths. ADR 0005 boundary is preserved.

Post-migration outcomes:

- Simulator and live pure validator packages share the same ref model end to end.
- Future Phase 14 successors that scaffold a live production caller (e.g., a `VALIDATE_CONTEXT_ATTRIBUTION` shadow call from the existing `COMPILE_CONTEXT` path) can route through a single deterministic validation surface.
- The "shared pure package" wording in current Phase 14 status entries becomes literally shared between simulator and live, not only across live seams.

## Consequences

- Snapshot/journal format must be versioned. Existing simulator snapshot/journal records in development data dirs may require regeneration or schema migration. Production live state is unaffected because the simulator is not live authority.
- Hashing changes will invalidate captured hashes in simulator fixtures. The migration commit must rebaseline `services/core/internal/forgek/fixture_parity_test.go` and any snapshot-hash assertions.
- Shared types in `services/core/internal/forgek/objects.go` and `services/core/internal/forgek/types.go` ripple through every forgek subpackage. The migration must land as one phase, not piecemeal; a partially-typed simulator (some packages on strings, others on ObjectRef) is a worse state than either endpoint.
- Forbidden-imports tests in the pure validator packages already forbid `services/core/internal/forgek` imports. The migration goes the other direction (simulator imports pure validators), which is permitted.
- After migration, the simulator may add its own forbidden-imports tests asserting that every subpackage that consumes refs imports `refvalidation` rather than redefining ObjectRef-shaped types.

## Alternatives Considered

- **Keep two ref models and bridge at the seam.** Rejected. A bridge layer would be a second source of truth for validation semantics and would contradict the deterministic-validator boundary that justifies the pure packages in the first place.
- **Migrate live pure packages to use `[]string` instead.** Rejected. String refs lose the workspace_id, ref_type, and ref_id discriminators that the deterministic gates depend on (`GateWorkspace`, `GateRefType`, `GateScope` in `services/core/internal/refvalidation/models.go:9-15`). Live validators would lose enforcement strength.
- **Do not migrate; mark simulator-shared seams as live-only-shared in status docs.** Acceptable as short-term doctrine reconciliation but rejected as the long-term answer because it blocks future live integration of any seam beyond `VALIDATE_KV_IDENTITY`.

## Sequence Of Migration

Recommended order. Each step is internally consistent and testable before moving to the next.

1. Introduce a typed alias in `services/core/internal/forgek/types.go` re-exporting `refvalidation.ObjectRef` as the canonical simulator ref type. Do not change persisted shape yet. Update one consumer to exercise the alias.
2. Migrate `services/core/internal/forgek/contextcompiler/` first (largest LOC, deepest serialization and hashing surface). Includes `bundle.go`, `service.go`, `serialize.go`, `hashing.go`. Regenerate fixture hashes.
3. Migrate `services/core/internal/forgek/semantic/`, `services/core/internal/forgek/court/`, `services/core/internal/forgek/lymphatic/`.
4. Migrate `services/core/internal/forgek/snapshots/` last; snapshots depend on every other ref-producing subsystem.
5. Update simulator tests; rebase `services/core/internal/forgek/fixture_parity_test.go`.
6. Add a forbidden-imports test in `services/core/internal/forgek` asserting that simulator subpackages either import `refvalidation` or define ref-handling internally without redefining ObjectRef-shaped types.

## Open Questions

- Should `refvalidation.ObjectRef.SourceRef` (the text-based annotation field) propagate through simulator types? Current simulator code does not use it; the migration could ignore it or carry it through.
- Should `services/core/internal/forgek/kv/` move its identity-field record types (`KVCacheManifest`, `KVLookupRequest`) into `kvidentity` so the simulator and live share manifest types as well as identity validation? Current state: simulator adapts via `manifestIdentity()` / `requestIdentity()` helpers at `services/core/internal/forgek/kv/gates.go:54-87`. The adapter pattern is acceptable, but unifying record types would remove a bridge.
- Does `services/core/internal/forgek/court/` need its own pure validator package for evidence-ref shape, mirroring `admissionvalidation`? Currently it has internal validation; aligning with the pure-validator pattern would let future live court integration share validation, but adds a package.

## Out Of Scope

- Scaffolding a live production caller for any seam. That is a separate `LIVE_INTEGRATION` phase with its own design, feature flag, rollback plan, and operator review.
- Changes to live Control Lane handlers, live API routes, or live modelruntime paths.
- Migration of `services/core/internal/aios/controllane/source_object_authority.go` to a pure package. That handler is intrinsically store-dependent (it verifies referenced objects exist in canonical state via `SemanticReadStore`) and the pure portion is already shared via `refvalidation`. See note in `docs/status/current_authority_sources.md`.

## References

- ADR 0001 - FORGE-K is a Cognitive Microkernel
- ADR 0005 - FORGE-K Simulator vs Live Authority
- `services/core/internal/refvalidation/models.go`
- `services/core/internal/forgek/kv/gates.go` (existing precedent for simulator/live sharing)
- `docs/status/current_authority_sources.md`
