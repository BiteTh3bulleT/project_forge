# P0 Durable Audit Projection

**Date:** 2026-08-17  
**Authority effect:** none; the projector cannot authorize, decide, commit,
replay, or alter canonical semantic state.

## Outcome

Successful production FORGE-K commits now stop after atomically recording their
self-verifying `forge_k_audit_outbox` intent. `Processor.RecordResult` no longer
opens the old sink/backfill crash window for those commits. Rejections, dry
runs, and the isolated nonproduction processor facade still record synchronous
terminal audit evidence because they have no committed outbox.

The daemon starts one `AuditProjector` over the same durable store. Each sweep:

1. lists outbox rows without terminal delivery evidence;
2. revalidates the request fingerprint, authorization proof, result identity,
   commit receipt, committed object IDs, and embedded journal entry/hash;
3. delivers to the audit service with the outbox ID as an idempotency key; and
4. appends an immutable `delivered`, `retry`, or `quarantined` attempt.

Retries use bounded exponential backoff and survive daemon restart. Invalid
proof is quarantined before the sink is called. Audit-record insertion is
idempotent through `audit_records.forge_k_outbox_id`; a crash after insertion
but before attempt recording recovers by resolving the original audit row on
the next sweep.

Legacy postcommit updates of semantic rows with `audit_id` are not used for
production commits. Immutable outbox, delivery attempt, audit record, syscall,
correlation, and journal identities provide the trace join without rewriting
committed evidence.

## Schema

- `audit_records.forge_k_outbox_id`: unique nonempty projection identity.
- `forge_k_audit_delivery_attempts`: immutable append-only delivery evidence.
- A partial unique terminal index permits one delivered or quarantined terminal
  outcome per outbox while allowing any number of retry attempts.
- Update/delete triggers reject delivery-attempt mutation.

## Failure behavior

- Sink unavailable: append `retry`; canonical commit remains valid.
- Empty sink identity: append `retry`.
- Invalid/tampered outbox proof: append `quarantined`; do not call the sink.
- Duplicate/concurrent sweep: sink identity and terminal attempt constraints
  converge without duplicate audit records.
- Daemon restart: pending and retry rows are rediscovered from SQLite.

## Validation expectations

- Focused audit, Control Lane, store, Kernel, and API tests.
- Exact-once idempotent sink test.
- Retry/restart and proof-quarantine tests.
- Immutable attempt trigger tests.
- Full Go test and vet.
- Backup/export/offline-recovery preservation of delivery attempts and outbox
  identities.
