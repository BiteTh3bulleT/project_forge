# Semantic Syscalls (Phase 2)

Semantic syscalls are the deterministic mutation boundary for FORGE AI-OS Control Lane.

They turn candidate semantic actions into a kernel-validated flow:

`request -> normalize -> authorize -> validate -> dry-run or commit -> audit -> deterministic result`

## Why this exists

- keeps canonical memory/state mutation inside FORGE kernel boundaries
- enforces workspace-scoped, auditable writes
- prevents LLM/agent/future IRIS direct mutation of canonical truth
- provides a stable contract for Phase 3 persistence backends

## Lifecycle

1. Receive `SyscallRequest`.
2. Normalize IDs, timestamps, correlation/trace identifiers.
3. Resolve action in registry.
4. Run envelope validation.
5. Run capability check.
6. Run approval gate.
7. Run payload + transition validation.
8. If `dryRun=true`, return validated result without commit.
9. Commit inside transaction boundary.
10. Emit audit record.
11. Return deterministic `SyscallResult`.

## Phase 3 persistence binding

Phase 3 replaces in-memory-only commit behavior with a SQLite-backed cognitive filesystem store (`services/core/internal/aios/controllane/sqlite_store.go`) that runs behind the same syscall pipeline via `SQLiteTransactionRunner`.

Commit path properties now include:

- real durable writes to cognitive filesystem tables
- transaction rollback on partial failures
- append-only semantic journal events (`journal_events`)
- state current projection + timeline history (`state_items` + `state_versions`)
- persisted supersession and contradiction records
- scope-aware queryable objects for later phases
- post-commit audit-id linkage (`audit_id`) by `syscall_id` + correlation

## Request contract

`services/core/internal/aios/domain.SyscallRequest` and `packages/shared/src/aios.ts` define the contract.

Core fields:

- `id`
- `action`
- `actor` (`id`, `kind`)
- `source` (`user|system|internal_cell|adapter|future_iris|test`)
- `scope.workspaceId`
- `payload`
- `provenance`
- `correlationId`
- `traceId`
- `idempotencyKey`
- `dryRun`
- `requestedAt`
- `requiredCapability` (optional)
- `capabilityHints` (optional)
- `metadata` (optional)

## Result contract

`services/core/internal/aios/domain.SyscallResult` is always returned, including failures.

Core fields:

- `success`
- `action`
- `requestId`
- `correlationId` / `traceId`
- `idempotencyKey`
- `dryRun`
- `approvalStatus` (`allowed|denied|approval_required`)
- `committedObjectIds`
- `rejectedReasons[]` (structured errors)
- `warnings[]`
- `auditId`
- `validationDetails[]`
- `stateSummary`
- `deterministicErrorCode`

Deterministic error categories include:

- `INVALID_ACTION`
- `INVALID_PAYLOAD`
- `MISSING_REQUIRED_FIELD`
- `INVALID_SCOPE`
- `INVALID_PROVENANCE`
- `INVALID_STATE_TRANSITION`
- `UNSUPPORTED_ACTION`
- `UNAUTHORIZED`
- `CAPABILITY_DENIED`
- `APPROVAL_REQUIRED`
- `CONFLICT`
- `DUPLICATE`
- `NOT_FOUND`
- `PERSISTENCE_UNAVAILABLE`
- `INTERNAL_ERROR`

## Supported actions (Phase 2)

- `CREATE_NOTE`
- `CREATE_LINK`
- `UPDATE_STATE`
- `OPEN_LOOP`
- `CLOSE_LOOP`
- `MARK_SUPERSEDED`
- `REGISTER_CONTRADICTION`
- `DERIVE_MODEL`
- `ARCHIVE_NOTE`
- `COMPILE_CONTEXT` (read-only deterministic context stub in Phase 2)

Phase 3 durable mapping:

- `CREATE_NOTE` -> `memory_notes`
- `CREATE_LINK` -> `semantic_links`
- `UPDATE_STATE` -> `state_items` upsert + `state_versions` append
- `OPEN_LOOP` / `CLOSE_LOOP` -> `open_loops`
- `MARK_SUPERSEDED` -> `supersession_records` + supersedes link + note status update
- `REGISTER_CONTRADICTION` -> `contradiction_records` + contradicts link
- `DERIVE_MODEL` -> `derived_models`
- `ARCHIVE_NOTE` -> note status transition in `memory_notes`
- `COMPILE_CONTEXT` -> deterministic read path (snapshot repository available; not auto-committed by default)

Committed semantic actions also append a `journal_events` row as semantic truth trace.

## Validation layers

- envelope validation (request shape, source, scope, provenance, timestamp)
- capability validation
- approval gate evaluation
- idempotency conflict/replay check
- payload validation
- transition/object invariant validation
- commit-time conflict validation

No validator layer depends on live LLM behavior.

## Permission model

Capabilities mapped in `services/core/internal/aios/controllane/registry.go`:

- `memory.note.create`
- `memory.note.archive`
- `memory.link.create`
- `state.update`
- `loop.open`
- `loop.close`
- `memory.contradiction.register`
- `memory.supersession.mark`
- `model.derive`
- `context.compile`

Source classes are mapped in `services/core/internal/aios/controllane/capabilities.go`.

`future_iris` is a proposer source class, not a commit authority.

## Dry-run behavior

- full validation executes
- no transaction commit executes
- `success=true` only when validations pass
- `committedObjectIds=[]`
- audit record still emitted with `dryRun=true`

## Audit behavior

All syscall outcomes are auditable:

- committed
- rejected (validation/capability/approval/commit conflict)
- dry-run
- idempotent replay

See `services/core/internal/aios/controllane/audit.go`.

## Future IRIS path

IRIS submits candidates as `source=future_iris` through the same syscall processor.

IRIS cannot directly mutate memory/state. Approval/capability/validation rules still apply.

Rule remains: **IRIS proposes. FORGE validates. FORGE commits.**

## Examples

### CREATE_NOTE

```json
{
  "action": "CREATE_NOTE",
  "payload": {
    "type": "fact",
    "title": "Kernel invariant",
    "content": "Durable writes pass through semantic syscalls."
  }
}
```

### CREATE_LINK

```json
{
  "action": "CREATE_LINK",
  "payload": {
    "type": "supports",
    "sourceId": "note-a",
    "targetId": "note-b",
    "confidence": 0.8
  }
}
```

### UPDATE_STATE

```json
{
  "action": "UPDATE_STATE",
  "payload": {
    "key": "runtime.mode",
    "value": { "value": "deterministic" },
    "derivedFrom": ["note-a"]
  }
}
```

### OPEN_LOOP / CLOSE_LOOP

```json
{
  "action": "OPEN_LOOP",
  "payload": { "title": "Implement Phase 3 store", "state": "open", "priority": "high" }
}
```

```json
{
  "action": "CLOSE_LOOP",
  "payload": { "loopId": "loop-1", "reason": "implemented" }
}
```

### MARK_SUPERSEDED / REGISTER_CONTRADICTION

```json
{
  "action": "MARK_SUPERSEDED",
  "payload": { "oldObjectId": "note-old", "newObjectId": "note-new", "reason": "newer evidence" }
}
```

```json
{
  "action": "REGISTER_CONTRADICTION",
  "payload": {
    "leftObjectId": "note-1",
    "rightObjectId": "note-2",
    "reason": "claims conflict",
    "severity": "medium",
    "confidence": 0.7
  }
}
```

### DERIVE_MODEL / ARCHIVE_NOTE / COMPILE_CONTEXT

```json
{
  "action": "DERIVE_MODEL",
  "payload": {
    "type": "routing",
    "expression": "score = evidence * confidence",
    "derivedFrom": ["note-1", "note-2"],
    "supportCount": 2
  }
}
```

```json
{
  "action": "ARCHIVE_NOTE",
  "payload": { "noteId": "note-1", "reason": "superseded by note-2" }
}
```

```json
{
  "action": "COMPILE_CONTEXT",
  "payload": {
    "query": "summarize active blockers",
    "budget": { "maxTokens": 4000, "maxEvents": 50, "maxNotes": 50 }
  }
}
```
