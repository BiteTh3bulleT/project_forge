# Kernel And Semantic Syscalls

## Syscall Model

IMPLEMENTED: Semantic syscalls are the deterministic mutation boundary:

`request -> normalize -> authorize -> validate -> dry-run or commit -> audit -> result`

Evidence: `services/core/internal/aios/controllane/registry.go`, `validator.go`, `processor.go`, `processor_apply.go`, `sqlite_store.go`, and tests.

## Supported Actions

IMPLEMENTED actions include `CREATE_NOTE`, `CREATE_LINK`, `UPDATE_STATE`, `OPEN_LOOP`, `CLOSE_LOOP`, `MARK_SUPERSEDED`, `REGISTER_CONTRADICTION`, `DERIVE_MODEL`, `ARCHIVE_NOTE`, and `COMPILE_CONTEXT`.

## Dry-Run Behavior

IMPLEMENTED: Dry-runs validate without committing. Audit records still describe the attempt.

## Commit Boundary

IMPLEMENTED: SQLite transaction runner commits cognitive filesystem mutations and appends `journal_events`. `journal_events` update/delete is blocked by SQLite triggers.

## Result States

Results include success/failure, dry-run flag, approval status, committed IDs, rejected reasons, warnings, audit ID, validation details, state summary, and deterministic error code.

## Edge Cases Remaining

- PARTIAL: Public operator syscall facade is missing.
- PARTIAL: More negative-path tests are needed around rollback, idempotency conflict, and approval-required transitions.
- PARTIAL: Operator-facing syscall inspection is limited.

