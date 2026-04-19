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

## Key modules

- `registry.go`
- `processor.go`
- `validator.go`
- `capabilities.go`
- `approval.go`
- `transitions.go`
- `store.go`
- `audit.go`
