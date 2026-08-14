# K20D FORGE-K Commit Integrity Report

Date: 2026-08-14

Status: `FORGE_K_COMMIT_INTEGRITY_LIVE / FULL_KERNEL_AUTHORITY_FALSE`

## Outcome

The production boundary in `services/core/internal/forgekernel` now seals the
exact post-policy, post-Courthouse syscall request and normalized prepared plan.
It accepts a successful commit only when the durable port returns a typed
receipt that validates against that request, plan, seal, result, object IDs,
provenance IDs, journal event/hash, audit-outbox ID, and stable idempotency
fingerprint.

The receipt persists the complete typed journal-chain entry. The Kernel
recomputes its hash and binds its event, syscall, scope, selected paths,
correlation, trace, provenance, content hashes, proposer, committer, timestamp,
sequence, and prior-hash shape to the sealed plan. The deterministic
`transactionId` is a journal/outbox-bound transaction correlation identity,
not an engine-native SQLite transaction handle.

This is production authority. The simulator under
`services/core/internal/forgek` remains isolated and is not a live commit
source. Models, adapters, Future IRIS, and other proposal workers cannot create
or validate canonical commit proof.

## Atomic SQLite boundary

One SQLite unit of work contains:

- the semantic mutation and current/history updates;
- the provenance-linked, sequenced journal hash-chain entry and head;
- the immutable audit-outbox intent, including the typed receipt;
- and, when a caller supplies an idempotency key, the immutable original
  request, plan, seal, receipt, result, request fingerprint, and stable
  idempotency fingerprint.

A journal, receipt, audit-outbox, idempotency, or transaction failure rolls
back the complete unit. No successful syscall is reported unless the
production Kernel validates the returned receipt.

## Replay posture

A matching retry reconstructs and verifies the original request, prepared
plan, seal, receipt, result, and stable idempotency fingerprint. Verified replay
returns the prior result and does not append a journal event or repeat the
semantic mutation. A key reused for another action or fingerprint fails closed.
Legacy rows backfilled as unbound and legacy proof JSON placeholders fail
closed rather than being trusted as replay evidence.

## Audit projection boundary

The immutable audit-outbox record is canonical evidence and is atomic with the
mutation and journal transition. Delivery to the existing external audit sink
and `audit_id` backfill remain best-effort post-commit projections. Projection
failure cannot roll back an already committed transaction, but also cannot
erase or invalidate its immutable outbox evidence. Outbox delivery and backfill
hardening remain operational follow-up work.

## Authority limits

K20D does not establish sole FORGE-K cognitive-kernel authority. Control Lane
still implements deterministic policy/apply functions and the temporary SQLite
durable port. Memory Palace and direct memory writers, Semantic Algebra,
Context Compiler, Runtime, Snapshots/restore, KV, Lymphatic, Consensus, and
other subsystem migrations remain open. `legacy_v1` remains rollback-only
until final cutover acceptance.

## Validation evidence

- `cd services/core && go test ./...`
- `cd services/core && go vet ./...`
- `cd services/core && go test -race ./internal/aios/controllane ./internal/forgekernel/... ./internal/store`
- `npm run validate:desktop` (56 files, 186 tests, production build)
- `npm run validate:repo-hygiene`
- `npm run validate:os-integration` (64 checks)
- `nix flake check --no-build`
- `git diff --check`

The API coverage verifies the read-only kernel and system status surfaces
report `forge_k_commit_integrity_live`, expose the canonical integrity posture,
keep external audit projections false, and retain `live_kernel_authority=false`
while remaining authority gates are open. Focused persistence coverage also
proves restart verification, rollback, immutable proof rows, legacy backfill,
and stored-content tamper detection.
