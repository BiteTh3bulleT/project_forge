# Simulator To Live Migration

Status: Phase i1/PhaseI2 partial live KV validation/enforcement pattern, Phase 14A operational cutover design guidance, Phase 14B partial live ref-shape validation, Phase 14C partial live validation expansion, Phase 14D disabled-by-default validation shadow reporting, Phase 14E disabled-by-default validation shadow emission, Phase 14F explicit Control Lane partial live enforcement metadata, Phase 14G Control Lane validation contract-matrix hardening, Phase 14H semantic-operation lane closure, Phase 14I ref-shape lane closure, Phase 14J ref-shape comparison lane closure, Phase 14K read-only activation readiness surface, Phase 14L read-only authority gate matrix, and Phase 14M source-object authority validation.

## Purpose

FORGE-K simulator packages define target authority behavior. Live daemon integration must migrate narrow, tested contracts into live paths without importing simulator authority wholesale or creating a second live authority path.

Phase i1 demonstrates the preferred pattern with deterministic KV identity validation.

PhaseI2 extends that pattern with `[PARTIAL LIVE ENFORCEMENT]`: live Control Lane code wraps the pure validator in a live-side policy layer that fails closed, records audit fields, and increments internal counters. The simulator service remains simulator-only.

Phase 14A generalizes this into the operational cutover rule: make FORGE-K operational by migrating one narrow authority seam at a time through the existing live owner, not by importing simulator services wholesale.

Phase 14B applies the pattern to deterministic ref-shape validation. `services/core/internal/refvalidation` is shared pure validation logic, and the live Control Lane action `VALIDATE_REF_SHAPE` uses it without importing FORGE-K simulator services or mutating live memory.

Phase 14C extends the same pattern with diagnostic ref-shape comparison and semantic-operation shape validation. `COMPARE_REF_SHAPE` reports match/drift between candidate and observed refs. `VALIDATE_SEMANTIC_OPERATION` validates operation shape and rejects authority claims. Both remain Control Lane validation only.

Phase 14D adds a read-only internal diagnostic report shape for validation summaries under `services/core/internal/forgekshadow`. It is disabled by default, requires both global shadow mode and `FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED`, and does not change Control Lane decisions or route live mutation through FORGE-K.

Phase 14E wires live Control Lane validation results into that observer through an optional best-effort processor dependency. The emission happens after a syscall result exists, is bounded to scalar summaries, and cannot affect validation decisions, memory, routes, retrieval, modelruntime, gateway behavior, or live authority.

Phase 14F turns on the first explicit `[PARTIAL LIVE ENFORCEMENT]` FORGE-K activation mode through the live Control Lane. It adds activation/no-effect metadata to existing validation actions and keeps authority in `services/core/internal/aios/controllane`. It does not import FORGE-K simulator services into live authority, route semantic writes through `forgek.Kernel`, enable live KV reuse, call modelruntime, execute tools, run retrieval/search/embeddings, or write memory outside existing governed paths.

Phase 14G hardens Phase 14F with a Control Lane validation contract matrix. The tests prove every existing live validation action exposes the same activation/no-effect metadata in syscall state summaries and audit summaries. This remains hardening only: no new validation action, simulator service import, public API, route behavior change, live KV reuse, modelruntime call, retrieval/search/embedding execution, evidence admission, context compilation, or semantic memory write is added.

Phase 14H closes the `VALIDATE_SEMANTIC_OPERATION` validation lane in the narrow Control Lane sense: the lane is connected through the live owner, rejects a canonical normalized authority-claim set, preserves no-effect metadata on rejected claims, and remains validation-only. Closed does not mean semantic operation execution, memory write authority, evidence admission authority, Context Compiler authority, modelruntime authority, retrieval/gateway execution, or FORGE-K simulator live authority.

Phase 14I closes the `VALIDATE_REF_SHAPE` validation lane in the same narrow Control Lane sense: the lane is connected through the live owner, exposes a canonical allowed ref-type contract, rejects invalid ref shapes before commit, preserves no-effect metadata on rejected refs, and remains validation-only. Closed does not mean object truth lookup, evidence admission or rejection authority, Context Compiler authority, retrieval/search/embedding authority, semantic memory write authority, modelruntime authority, gateway/tool execution, or FORGE-K simulator live authority.

Phase 14J closes the `COMPARE_REF_SHAPE` diagnostic lane in the same narrow Control Lane sense: the lane is connected through the live owner, reuses canonical ref-shape validation for candidate and observed refs, accepts match and drift as diagnostic outcomes, rejects invalid comparisons before commit, and preserves no-effect metadata on rejected comparisons. Closed does not mean object truth lookup, evidence admission or rejection authority, Context Compiler authority, retrieval/search/embedding authority, semantic memory write authority, modelruntime authority, gateway/tool execution, or FORGE-K simulator live authority.

Phase 14K exposes a live-owned, read-only activation readiness report for the already connected validation lanes. The report is produced by `services/core/internal/aios/controllane`, surfaced through `GET /forge/kernel/status`, included in `GET /forge/system/status` as `kernel_activation`, and rendered in the desktop System page. It reports whether the Control Lane validation seams are registered, non-mutating, audit/capability-backed, and still bounded by no-effect guarantees. It does not import FORGE-K simulator services, enable live Kernel authority, add mutation controls, execute semantic operations, write memory, load models, call retrieval/search/embeddings, or change gateway/tool authority.

Phase 14L extends that read-only readiness report with an authority gate matrix. The matrix marks the connected Control Lane validation/enforcement gate as ready and keeps source-object authority lookup, Courthouse admission integration, live Context Compiler authority, governed semantic mutation routing, and runtime driver authority blocked. It is operator visibility only and grants no mutation authority, approval controls, simulator authority, live Kernel authority, semantic writes, retrieval/search/embedding execution, modelruntime mutation, or gateway/tool execution.

Phase 14M closes the source-object authority lookup gate in the narrow Control Lane sense. `VALIDATE_SOURCE_OBJECT_AUTHORITY` validates ref shape first, then resolves only governed live read-store objects it can prove by id, workspace, lane-compatible scope, and supported kind. Missing, unsupported, or scope-conflicting refs fail closed. This remains validation-only: it does not admit evidence, execute semantic operations, compile context, write memory, call modelruntime, execute retrieval/search/embeddings, change gateway/tool authority, import simulator services, or grant live Kernel authority.

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

## Phase 14B Ref Shape Example

| Concern | Decision |
| --- | --- |
| Deterministic contract | typed ref-shape validation and normalization |
| Shared pure package | `services/core/internal/refvalidation` |
| Live caller | `services/core/internal/aios/controllane` |
| Live action | `VALIDATE_REF_SHAPE` |
| Capability | `ref.shape.validate` |
| Audit | existing Control Lane audit record with `refShapeValidation` fields |
| Mutation posture | validation-only; no object lookup, evidence admission, context compilation, retrieval, modelruntime call, or memory write |
| Still future | Courthouse admission integration, live context compilation, and broader FORGE-K Kernel authority |

## Phase 14C Validation Expansion Example

| Concern | Decision |
| --- | --- |
| Ref comparison contract | `refvalidation.CompareRefShapes` |
| Semantic operation contract | `semanticvalidation.ValidateOperation` |
| Live caller | `services/core/internal/aios/controllane` |
| Live actions | `COMPARE_REF_SHAPE`, `VALIDATE_SEMANTIC_OPERATION` |
| Capabilities | `ref.shape.compare`, `semantic.operation.validate` |
| Audit | existing Control Lane audit record with `refShapeComparison` and `semanticOperationValidation` fields |
| Mutation posture | validation/comparison only; no object lookup, evidence admission, context compilation, retrieval, modelruntime call, tool execution, or memory write |
| Still future | actual semantic operation execution, live Courthouse admission, live Context Compiler authority, and broader FORGE-K Kernel authority |

## Phase 14H Semantic Operation Lane Closure Example

| Concern | Decision |
| --- | --- |
| Closed lane | `VALIDATE_SEMANTIC_OPERATION` validation only |
| Shared pure package | `services/core/internal/semanticvalidation` |
| Live caller | `services/core/internal/aios/controllane` |
| Canonical guard | normalized forbidden authority claims |
| Rejected posture | fail closed with no-effect state and audit metadata |
| Mutation posture | validation-only; no semantic execution, memory write, evidence admission, context compilation, retrieval/search/embedding execution, gateway/tool execution, modelruntime call, route/API change, or FORGE-K simulator live authority |
| Still future | actual semantic operation execution and any broader FORGE-K authority migration |

## Phase 14I Ref Shape Lane Closure Example

| Concern | Decision |
| --- | --- |
| Closed lane | `VALIDATE_REF_SHAPE` validation only |
| Shared pure package | `services/core/internal/refvalidation` |
| Live caller | `services/core/internal/aios/controllane` |
| Canonical guard | copied allowed ref-type list plus fail-closed shape/safety/scope gates |
| Rejected posture | fail closed with no-effect state and audit metadata |
| Mutation posture | validation-only; no object truth lookup, evidence admission or rejection, context compilation, retrieval/search/embedding execution, semantic memory write, gateway/tool execution, modelruntime call, route/API change, or FORGE-K simulator live authority |
| Still future | source-object authority lookup, evidence admission, live context compilation, semantic writes based on refs, and any broader FORGE-K authority migration |

## Phase 14J Ref Shape Comparison Lane Closure Example

| Concern | Decision |
| --- | --- |
| Closed lane | `COMPARE_REF_SHAPE` diagnostic comparison only |
| Shared pure package | `services/core/internal/refvalidation` |
| Live caller | `services/core/internal/aios/controllane` |
| Canonical guard | candidate and observed refs validated through the canonical ref-shape validator |
| Accepted posture | match and drift are accepted diagnostic outcomes only |
| Rejected posture | invalid candidate or observed refs fail closed with no-effect state and audit metadata |
| Mutation posture | diagnostic/validation-only; no object truth lookup, evidence admission or rejection, context compilation, retrieval/search/embedding execution, semantic memory write, gateway/tool execution, modelruntime call, route/API change, or FORGE-K simulator live authority |
| Still future | source-object authority lookup, evidence admission, live context compilation, semantic writes based on comparison output, and any broader FORGE-K authority migration |

## Phase 14K/14L/14M Activation Readiness Surface Example

| Concern | Decision |
| --- | --- |
| Readiness owner | live Control Lane package `services/core/internal/aios/controllane` |
| Read-only API | `GET /forge/kernel/status` |
| Shell composition | `kernel_activation` in `GET /forge/system/status` |
| Required actions | `VALIDATE_KV_IDENTITY`, `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, `VALIDATE_SOURCE_OBJECT_AUTHORITY`, `VALIDATE_SEMANTIC_OPERATION` |
| Operator surface | desktop System page `FORGE-K Activation Readiness` panel with readiness and authority-gate sections |
| Authority gates | ready: `control_lane_validation_enforcement`, `source_object_authority_lookup`; blocked: `courthouse_admission_integration`, `live_context_compiler_authority`, `governed_semantic_mutation_routing`, `runtime_driver_authority_boundary` |
| Mutation posture | read-only validation/status only; authority gates grant no mutation authority; no approval buttons, execution buttons, semantic operation execution, semantic memory write, gateway/tool execution, modelruntime mutation, retrieval/search/embedding execution, host mutation, or FORGE-K simulator live authority |
| Still future | explicit live Kernel authority migration, Courthouse admission integration, context compilation authority, governed semantic mutation routing, runtime driver authority, and rollback/observability gates |

## Phase 14D Validation Shadow Reporting Example

| Concern | Decision |
| --- | --- |
| Diagnostic surface | internal `forgekshadow` validation summary report |
| Flagging | `FORGE_K_SHADOW_MODE_ENABLED=true` plus `FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED=true` |
| Captured data | bounded scalar metadata: action, validation kind, decision, pass/fail, match/drift counts, operation type, warning/failure counts, duration |
| Mutation posture | read-only diagnostics; no Control Lane mutation, public API, route behavior change, user-visible output, memory write, modelruntime call, retrieval/search/embedding execution, evidence admission, or context compilation |
| Still future | live hook policy for when to emit these reports from specific validation call sites and any operator-visible diagnostics surface |

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
- `[PARTIAL]` Ref shape validation: live validation exists; object lookup, evidence admission, context compilation, and memory writes do not.
- `[PARTIAL]` Ref shape shadow comparison: diagnostic comparison exists; it does not affect live output or state.
- `[PARTIAL]` Semantic operation shape validation: operation envelope validation exists; operation execution and authority migration do not.
- `[PARTIAL]` Control Lane validation shadow reporting: internal diagnostic report support exists; it remains disabled by default and does not affect live output or state.
- `[PARTIAL]` FORGE-K authority gate matrix: operator-visible blockers exist; blocked gates do not authorize live authority.
- `[FUTURE]` Context Compiler mirror: requires no-effect live comparison before any prompt authority migration.
- `[FUTURE]` Runtime driver identity capture: requires modelruntime trace-only adapters before live reuse.
- `[FUTURE]` Retrieval evidence admission: must go through Courthouse/control-lane boundaries, not vector-store scores.
- `[FUTURE]` Storage/backend cutover: requires explicit live authority owner, repository parity tests, migration tests, backup/rollback proof, observability, and operator approval before any read switch or dual-write.
