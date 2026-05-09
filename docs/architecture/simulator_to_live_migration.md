# Simulator To Live Migration

Status: Phase i1/PhaseI2 partial live KV validation and enforcement pattern plus Phase 14A operational cutover design guidance.

## Purpose

FORGE-K simulator packages define target authority behavior. Live daemon integration must migrate narrow, tested contracts into live paths without importing simulator authority wholesale or creating a second live authority path.

Phase i1 demonstrates the preferred pattern with deterministic KV identity validation.

PhaseI2 extends that pattern with `[PARTIAL LIVE ENFORCEMENT]`: live Control Lane code wraps the pure validator in a live-side policy layer that fails closed, records audit fields, and increments internal counters. The simulator service remains simulator-only.

Phase 14A generalizes this into the operational cutover rule: make FORGE-K operational by migrating one narrow authority seam at a time through the existing live owner, not by importing simulator services wholesale.

## Migration Pattern

1. Identify the smallest pure deterministic contract.
2. Extract that contract into a live-safe shared package with no simulator or live-daemon side effects.
3. Keep simulator packages as simulator-owned services.
4. Add a live Control Lane or gateway boundary only where the existing live authority path already owns that class of decision.
5. Add live acceptance tests that prove no unintended mutation, no route/API change, and fail-closed behavior.
6. Update reality/status docs with `[LIVE]`, `[SIMULATOR-ONLY]`, `[PARTIAL]`, and `[FUTURE]` markers.

## Phase i1 Example

| Concern | Decision |
| --- | --- |
| Deterministic contract | KV identity gate validation |
| Shared pure package | `services/core/internal/kvidentity` |
| Simulator caller | `services/core/internal/forgek/kv` |
| Live caller | `services/core/internal/aios/controllane` |
| Live action | `VALIDATE_KV_IDENTITY` |
| Capability | `kv.identity.validate` |
| Mutation posture | validation-only; no memory/runtime mutation; no live KV reuse |

## PhaseI2 Enforcement Example

| Concern | Decision |
| --- | --- |
| Enforcement policy | `services/core/internal/aios/controllane/kv_enforcement.go` |
| Metrics | `KVIdentityEnforcementCounters`, internal process counters only |
| Audit | Existing Control Lane audit record with `kvIdentityEnforcement` fields |
| Rejected inputs | gate mismatch, malformed payload, unavailable manifest, explicit or ambiguous live KV reuse request |
| Still future | live KV reuse, runtime cache lookup, tokenizer-specific token IDs, exported metrics |

## Live-Safe Shared Package Rules

A shared package may be used by simulator and live code only when it:

- has deterministic inputs and outputs
- has no runtime/model/gateway/retrieval side effects
- does not import `services/core/internal/forgek` simulator services
- does not import live daemon stateful services unless it is intentionally live-owned
- does not create canonical memory, evidence, ContextBundles, snapshots, or KV tensors
- is covered by focused unit tests

## Live Control Lane Integration Rules

Live Control Lane integration is appropriate when:

- the operation is semantic validation or canonical semantic mutation
- capability checks are explicit
- payload validation is deterministic
- failures reject before canonical mutation
- commit behavior is journaled or explicitly documented as validation-only
- audit/result state records no-effect guarantees

## Forbidden Shortcut

Do not import FORGE-K Kernel, Context Compiler, KVService, Runtime Driver Boundary, Lymphatic Lane, Consensus Mesh, or simulator syscalls into live daemon paths and call that "integration." Those packages remain target architecture until an explicit live authority migration phase owns the risks and tests.

## Readiness Checklist

- Pure contract extracted or adapter boundary designed
- Simulator behavior still passes
- Live authority owner identified
- Capability added only where needed
- Live acceptance tests prove no unauthorized mutation
- Route/API behavior unchanged unless explicitly approved
- Gateway/modelruntime/retrieval behavior unchanged unless explicitly approved
- Docs updated with status markers
- Validation commands recorded

## Future Candidates

- `[PARTIAL]` KV identity validation: live validation exists; live reuse does not.
- `[FUTURE]` Context Compiler mirror: requires no-effect live comparison before any prompt authority migration.
- `[FUTURE]` Runtime driver identity capture: requires modelruntime trace-only adapters before live reuse.
- `[FUTURE]` Retrieval evidence admission: must go through Courthouse/control-lane boundaries, not vector-store scores.
- `[FUTURE]` Storage/backend cutover: requires explicit live authority owner, repository parity tests, migration tests, backup/rollback proof, observability, and operator approval before any read switch or dual-write.
- `[FUTURE]` Phase 14B operational validation seam: should extract one pure deterministic contract into a shared package and call it from the live Control Lane without replacing live authority.
