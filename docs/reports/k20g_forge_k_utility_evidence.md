# K20G FORGE-K Utility Evidence Report

Date: 2026-08-14

Status: implemented for retrieval usefulness and restore-outcome feedback;
full FORGE-K authority remains incomplete.

## Outcome

K20G replaces two mutable feedback paths with production FORGE-K semantic
syscalls:

- `RECORD_RETRIEVAL_USEFULNESS`
- `RECORD_RESTORE_OUTCOME_FEEDBACK`

Both require exact workspace, lane, and selected-path identity; a source row
bound to a prior syscall and provenance; authenticated production authority;
and an idempotency key. The action payload cannot carry authority, decision,
projection, journal, receipt, seal, audit, or provenance claims. Adapter,
Future IRIS, model/proposal, `legacy_v1`, and legacy-unbound requests fail
closed.

## Persistence contract

Accepted utility feedback appends an immutable event. It does not update the
source `retrieval_results` or `restore_outcome_events` row. The event and its
explicitly noncanonical projection persist in the same FORGE-K unit of work as
provenance, the sequenced journal hash-chain transition, immutable audit-outbox
intent, typed commit receipt, and idempotency proof. Journal failure rolls the
whole unit back, and a verified retry returns the original receipt without a
second event.

`retrieval_usefulness_projection` and
`restore_outcome_feedback_projection` are disposable/rebuildable views of the
immutable event stream. Restore reads return original evidence and a separate
`feedbackProjection`; retrieval ranking and result reads use only the governed
projection and ignore legacy mutable usefulness columns.

## Retired bypasses

- `UpdateRestoreOutcomeFeedback` is removed from public, in-memory,
  transactional, and SQLite store surfaces.
- `MarkUsefulness` no longer mutates retrieval, Memory Palace, or VSA state.
- Usefulness events do not increment VSA reliability counters.
- `RecordOutcome` no longer performs a query-then-insert into
  `context_evidence`; it fails closed until an exact scoped batch utility
  contract carries job/run/result identity, actor, provenance, and
  idempotency.

## Validation evidence

Focused coverage proves production ingress enforcement, source-class denial,
recursive metadata-claim rejection, exact scope matching, immutable event
triggers, source/projection separation, replay without re-commit, rollback on
journal failure, zero legacy memory/VSA mutation, projection-only ranking, API
body-size ordering, and production direct-writer static guards.

This slice does not make FORGE-K the sole cognitive kernel. Control Lane still
implements the temporary policy/apply/SQLite port, live raw restore remains
disabled, and other Memory Palace, Context Compiler, Runtime, snapshot, KV,
Lymphatic, Consensus, and direct-writer migrations remain.
