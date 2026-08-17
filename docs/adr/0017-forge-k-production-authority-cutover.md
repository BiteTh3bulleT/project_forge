# ADR 0017 - FORGE-K Production Authority Cutover

Status: Accepted; K20J sole-authority cutover active; P0 completion in progress

Date: 2026-08-14

## Context

The FORGE-K simulator proves the target cognitive microkernel contracts, while
the live daemon historically assigned semantic mutation authority to
`internal/aios/controllane`. Promoting the simulator directly would replace a
durable SQLite transaction and journal path with in-memory registries, creating
data-loss and dual-authority risks.

The chat gateway also previously exposed the full tool proposal catalog on
some ambiguous operational turns. Gateway policy still governed execution, but
allowing a model to choose a tool conflicted with the stronger FORGE doctrine:
models format proposals; FORGE decides tool use.

## Decision

FORGE-K moves to production one authority boundary at a time:

1. `internal/forgekernel` is the production FORGE-K authority package.
2. The simulator under `internal/forgek` remains isolated until its pure
   deterministic contracts are promoted into production-owned packages.
3. Stage K20A makes FORGE-K the default live semantic syscall ingress owner.
4. Stage K20B makes FORGE-K own the durable stage order through its
   `DurablePort`: deterministic prepare, exactly one atomic apply+journal
   commit, audit recording/linkage, then best-effort observation. The existing
   Control Lane code implements that temporary port and the rollback facade;
   it is not a second authority.
5. Stage K20C makes production FORGE-K the only deterministic Courthouse
   decision owner. `ADMIT_EVIDENCE` and `APPEAL_RULING` atomically persist
   current exhibit state, immutable ruling/appeal history, provenance, and the
   semantic journal event. Proposal-only sources, model actors, and
   `legacy_v1` cannot rule.
6. Stage K20D makes canonical commit evidence fail closed. The production
   Kernel seals the exact request and prepared plan, and validates a typed
   receipt returned by the durable port. Semantic mutation, provenance-linked
   journal hash-chain entry/head, immutable audit intent, and optional
   immutable idempotency proof commit in one SQLite transaction. Matching
   replay is verified without another commit; conflicting or legacy unbound
   proof fails closed.
7. Stage K20E makes authorization evidence a Kernel prerequisite. The Kernel
   resolves and independently verifies the constructed `forge.core` service
   principal or trusted request origin, effective action definition,
   scope-exact capability grant, and approval policy/decision before prepare.
   The exact request and full proof persist with the atomic audit intent;
   idempotent replay requires the same immutable authorization binding.
8. K20E also retires raw live backup restore. The live endpoint performs
   deterministic inspection only; whole-store recovery is daemon-stopped,
   staged, chain-verified future work. Foreign journal and idempotency proof
   can never be merged into the active local authority chain.
9. Stage K20F contains behavior-affecting memory writers: automatic and manual
   repair/reindex are proposal-only and retrieval no longer rewrites legacy
   observations. VSA state is a deterministic rebuildable projection rather
   than a usefulness-feedback mutation target.
10. Stage K20G admits utility feedback only as append-only FORGE-K evidence.
   Retrieval usefulness and restore-outcome feedback preserve their original
   evidence, atomically append immutable events, and update only separately
   labeled rebuildable projections. Legacy direct writers are retired.
11. Boot selects exactly one mode through `FORGE_KERNEL_AUTHORITY_MODE`:
   `forge_k` (default) or `legacy_v1` (rollback). There is no dual commit mode.
12. Full FORGE-K authority is not claimed until the adapter implementation and remaining
   subsystem gates are migrated and v1 is retired.
13. FORGE deterministically decides whether a chat turn needs a tool and selects
   exactly one eligible tool before any model proposal call. If it cannot do so,
   no tool schema is exposed. A model may only format bounded arguments for the
   schema FORGE selected; the gateway remains execution authority.
14. K20J removes the live authority selector. Production assembly constructs
   exactly one `forgekernel.Kernel`; rollback is daemon-stopped verified
   store/generation recovery rather than a second live orchestrator.
15. P0 replaces synchronous best-effort audit delivery for successful commits
   with a restart-safe projector. The projector revalidates immutable outbox
   proof, uses the outbox ID as the sink idempotency key, and appends immutable
   delivery/retry/quarantine attempts. Legacy object-row `audit_id` backfill is
   retired.

## Cutover order

1. Production Kernel ingress and single-authority boot selection. Closed K20A.
2. Durable journal/commit orchestration owned by FORGE-K through narrow ports. Closed K20B; Control Lane implementation extraction remains.
3. Courthouse admission and ruling authority. Closed K20C.
4. Commit integrity: sealed prepare plans, typed receipts, atomic audit intent
   and idempotency proof, persisted journal hash chain, and replay divergence.
   Closed K20D for the canonical SQLite transaction and production Kernel
   receipt validation. P0 makes its external audit projection durable,
   idempotent, restart-safe, and independently observable.
5. Authenticated principal, registry, capability, approval, and replay proof.
   Closed K20E for production semantic syscalls.
6. Retire unsafe live restore and mutable restore-outcome feedback. Closed
   K20E; daemon-stopped whole-store recovery and bounded semantic import remain.
7. Contain legacy Memory Palace observation/repair/VSA writers. Closed K20F for
   proposal-only maintenance and retrieval observation duplication; K20G adds
   deterministic, scoped VSA rebuild authority with an atomic manifest swap.
8. Append-only utility evidence and noncanonical projection rebuild. Closed
   K20G for retrieval usefulness and restore-outcome feedback; scoped retrieval
   job-outcome evidence remains future work.
9. Semantic Algebra operations and structured Memory Palace objects.
10. Context Compiler live bundle authority.
11. Runtime driver proposal boundary and response composition gate.
12. Snapshots, KV acceleration, and Lymphatic proposal lanes.
13. Remove `legacy_v1`, compatibility facades, and stale authority claims.
    Closed for daemon assembly in K20J; the combined facade remains test-only
    while the temporary durable-port implementation is extracted.

Each step requires deterministic parity tests, malformed-input failure tests,
capability/approval tests, journal and audit evidence, a tested rollback path,
operator-visible status, and documentation updates.

## Consequences

- FORGE-K authenticated ingress, durable stage orchestration, Courthouse
  decisions, and commit-integrity verification are live now, but the
  full-kernel flag remains false.
- Existing durable data remains in the current SQLite schema during migration.
- The canonical immutable audit-outbox intent is atomic with the mutation and
  journal proof. External delivery is idempotent and restart-safe with
  append-only attempt evidence. Mutable semantic-row `audit_id` backfill is
  retired; projection failure cannot invalidate canonical evidence.
- The Control Lane can be reduced behind ports rather than duplicated or
  bypassed.
- Simulator service imports remain forbidden in live paths.
- Models have no tool-selection, execution, admission, or mutation authority.
- The migration is observable and reversible until final v1 retirement.

## Rejected alternatives

- Directly run `forgek.Kernel` in production: rejected because its registry and
  journal are simulator-memory structures.
- Big-bang replacement: rejected because it removes useful parity and rollback
  evidence across multiple authority domains at once.
- Dual writes: rejected because two commit authorities make truth ambiguous.
- Model-selected tools with gateway validation: rejected because validation of
  execution does not make model-owned selection a FORGE decision.
