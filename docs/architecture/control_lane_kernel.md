# Control Lane Kernel (Phase 2)

Phase 2 makes FORGE Control Lane a deterministic semantic syscall processor.

Implementation location:

- `services/core/internal/aios/controllane`

## Responsibilities

Control Lane owns:

- semantic syscall registry and action metadata
- deterministic request normalization
- envelope and payload validation
- capability checks
- approval gate checks
- state transition guardrails
- transaction boundary for commits
- audit emission for commit/reject/dry-run
- deterministic typed results

Phase i1 adds `[LIVE] / VALIDATION_ONLY` `VALIDATE_KV_IDENTITY` to this boundary. It validates deterministic KV identity gates through the shared pure package `services/core/internal/kvidentity`, returns an acceleration-only result summary, and does not mutate memory, call modelruntime, reuse live KV cache, or route through FORGE-K `KVService`.

PhaseI2 marks this as `[PARTIAL LIVE ENFORCEMENT]`: `VALIDATE_KV_IDENTITY` is routed through Control Lane KV enforcement policy before acceptance. The policy rejects malformed claims, gate mismatches, unavailable manifests, and explicit or ambiguous live KV reuse requests; audit records include structured enforcement fields, and internal counters track accepted/rejected/malformed/unsupported decisions.

Phase 14B adds `[PARTIAL LIVE VALIDATION]` `VALIDATE_REF_SHAPE`. It validates deterministic ref shape through the shared pure package `services/core/internal/refvalidation`, returns normalized refs and no-mutation authority flags, and does not look up live objects, admit evidence, write memory, call modelruntime, execute retrieval, change routes, or route through FORGE-K simulator services.

Phase 14C adds `[PARTIAL LIVE VALIDATION]` `COMPARE_REF_SHAPE` and `VALIDATE_SEMANTIC_OPERATION`. Ref shape comparison reports diagnostic match/drift sets through `services/core/internal/refvalidation`; semantic operation validation checks operation envelope shape through `services/core/internal/semanticvalidation`. Both are non-mutating, capability-gated Control Lane validations and do not admit evidence, compile context, execute retrieval/search/embeddings, call modelruntime, write memory, execute tools, change routes, or make FORGE-K simulator services live authority.

Phase 14D adds disabled-by-default internal shadow reporting support for Control Lane validation summaries through `services/core/internal/forgekshadow`. It records bounded scalar diagnostics only when global shadow mode and `FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED` are both enabled. The reports do not alter Control Lane decisions, change routes, expose a public API, affect user-visible output, admit evidence, compile context, write memory, execute retrieval/search/embeddings, call modelruntime, execute tools, or make FORGE-K simulator services live authority.

Phase 14E wires those disabled-by-default diagnostic summaries into the live Control Lane processor through an optional best-effort observer. The observer receives bounded validation result metadata after a syscall result exists and cannot change the returned result. The live Control Lane remains the authority; FORGE-K simulator services remain outside the live authority path.

Phase 14F adds explicit FORGE-K partial live enforcement metadata to Control Lane validation actions. `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, `VALIDATE_SEMANTIC_OPERATION`, and `VALIDATE_KV_IDENTITY` expose activation mode and no-effect summaries while keeping the live owner as Control Lane. Phase 14M extends the same validation-only contract to `VALIDATE_SOURCE_OBJECT_AUTHORITY`, which resolves supported governed source objects by id/scope and fails closed. This is not full FORGE-K live authority and does not import simulator services, enable live KV reuse, execute tools, call modelruntime, execute retrieval/search/embeddings, admit evidence, compile context, or write memory outside existing governed paths.

Phase 14G adds a contract matrix test for the partial-enforcement validation actions. The matrix checks `VALIDATE_KV_IDENTITY`, `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, `VALIDATE_SOURCE_OBJECT_AUTHORITY`, and `VALIDATE_SEMANTIC_OPERATION` through the live processor and audit path, proving the activation/no-effect posture is consistent across state and audit summaries.

Phase 14H closes the `VALIDATE_SEMANTIC_OPERATION` lane as `LANE_CLOSED / VALIDATION_ONLY / NO_AUTHORITY_EXPANSION`. The shared validator owns a canonical normalized forbidden authority-claim list, the live Control Lane rejects truthy authority claims before commit, rejected claims preserve no-effect state/audit metadata, and false/non-truthy claims do not reject by themselves. This lane closure still does not execute semantic operations, write memory, admit evidence, compile context, call modelruntime, execute retrieval/search/embeddings, execute tools, change routes/APIs, or make FORGE-K simulator services live authority.

Phase 14I closes the `VALIDATE_REF_SHAPE` lane as `LANE_CLOSED / VALIDATION_ONLY / NO_AUTHORITY_EXPANSION`. The shared validator owns a canonical allowed ref-type list, exposes a copied list for tests/operator status, fails closed for empty refs, missing ids, unsafe ids, unknown types, and workspace mismatches, and the live Control Lane rejects invalid refs before commit with no-effect state/audit metadata. This closure validates ref shape only; it does not look up object truth, admit or reject evidence, compile context, execute retrieval/search/embeddings, write semantic memory, call modelruntime, execute gateway/tools, change routes/APIs, import FORGE-K simulator services, or make FORGE-K live authority.

Phase 14J closes the `COMPARE_REF_SHAPE` lane as `LANE_CLOSED / DIAGNOSTIC_ONLY / VALIDATION_ONLY / NO_AUTHORITY_EXPANSION`. The shared comparison code reuses canonical ref-shape validation, accepted match/drift outcomes remain diagnostic only, and invalid candidate or observed refs reject before commit with no-effect state/audit metadata. This closure does not prove object truth, admit or reject evidence, compile context, execute retrieval/search/embeddings, write semantic memory, call modelruntime, execute gateway/tools, change routes/APIs, import FORGE-K simulator services, or make FORGE-K live authority.

## FORGE-K Partial Enforcement

Control Lane validation actions now expose FORGE-K partial live enforcement metadata. This means deterministic doctrine is enforced by the current live Control Lane owner, not by importing the simulator Kernel. Validation summaries include the activation mode and no-effect posture so callers and operator surfaces can distinguish partial enforcement from full live authority.

## Kernel / user-space boundary

- user space: users, adapters, internal cells, future IRIS propose semantic actions
- kernel space: validates proposals, authorizes, commits, audits

No proposer source can directly mutate canonical memory/state outside syscall processing.

## Deterministic validation

Validation is layered and non-LLM:

1. envelope
2. capability
3. approval
4. idempotency
5. payload
6. transition/invariant checks
7. commit conflict checks

All errors are mapped to deterministic `SyscallErrorCode`.

## Relationship to existing FORGE controls

This lane scaffolding extends existing doctrine rather than replacing it:

- gateway/action-lane concepts remain kernel-like boundaries
- permissions/capabilities are explicit in syscall metadata
- approval gates are represented through `ApprovalGate` interface
- audit integrates through `AuditSink` and can bridge to `internal/audit.Service` (`CoreAuditSink`)

Phase 2 does not remove existing production gateway/approval/audit paths; it establishes a deterministic semantic syscall core that can be integrated deeper in later phases.

## Transaction boundary

Phase 2 introduced an in-memory transaction abstraction; Phase 3 adds SQLite-backed durable commits through the same interfaces.

- `TransactionRunner`
- `UnitOfWork`
- `SemanticStore` / `SemanticReadStore`

Concrete runners:

- `InMemoryTransactionRunner` (tests/scaffold)
- `SQLiteTransactionRunner` (durable cognitive filesystem persistence)

Properties:

- dry-run never commits
- validation/capability/approval failures never commit
- commit errors rollback partial writes
- idempotency replay is deterministic
- committed semantic objects can be linked to audit record ids after commit

Phase 3 persistence is attached to this boundary; later phases build richer ingest/compute behavior on top.

## Failure modes

Primary deterministic failures:

- unsupported action
- malformed envelope
- capability denied
- approval required/denied
- invalid payload
- invalid state transition
- missing referenced object
- duplicate/idempotency conflict
- commit conflict

Failure results still emit audit records when safe.

## Autonomy integration (Phase 5.75)

Self-initiated actions from FORGE autonomy still enter through the same syscall processor.

- intent/charter/budget decisions do not commit directly
- autonomous runners submit `SyscallRequest` like any other source
- kernel validation, capability checks, approval checks, transition rules, and audit all remain mandatory
- no autonomy mode can bypass Control Lane commit boundaries

## Tool capability integration (Phase 5.9)

Tool execution is governed in the gateway, but kernel sovereignty remains intact:

- tool calls are policy/audit/approval controlled through the tool gateway and capability registry
- tool output is treated as evidence artifacts and invocation records
- no tool output may directly mutate canonical semantic truth
- any truth-changing effect from tool output must be represented as semantic syscall proposals and processed by Control Lane

This preserves the separation:

- gateway controls externalized side-effect execution
- Control Lane controls semantic truth commits

## Discord gateway integration

Discord is an external control and conversational surface, not a kernel authority.

- inbound Discord events are normalized and routed in `internal/api/discord_gateway_*`
- Discord handlers may enqueue chat/intent work and produce tool/evidence outputs
- any semantic truth mutation still requires syscall validation/authorization/audit in Control Lane
- Discord identity does not bypass permission/approval gates

## Key modules

- `registry.go`
- `processor.go`
- `validator.go`
- `capabilities.go`
- `approval.go`
- `transitions.go`
- `store.go`
- `audit.go`
