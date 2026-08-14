# FORGE-K Live Cutover

Status: K20A-K20D active; full cutover in progress.

Date: 2026-08-14.

## Target

FORGE-K is complete when the live daemon has one production cognitive kernel
that owns semantic syscall admission, deterministic validation, canonical
commit, journal/replay, provenance, and subsystem coordination. Models,
retrievers, adapters, and tool runtimes remain external proposal/evidence
drivers. Gateway remains the only concrete tool execution boundary.

`services/core/internal/forgek` is the simulator and contract laboratory.
`services/core/internal/forgekernel` is the production authority boundary.
Simulator contracts are promoted into production-owned pure packages or durable
ports; simulator services are never imported wholesale into the daemon.

## Current live chain

```text
caller
  -> forgekernel.Kernel                    [live ingress authority]
     -> deterministic authority-claim gate
     -> DurablePort.Prepare                 [deterministic preflight]
     -> seal exact prepared request + plan
     -> DurablePort.Commit                  [one atomic canonical transaction]
        -> semantic mutation
        -> provenance-linked journal hash-chain entry + head
        -> immutable audit intent with typed commit receipt
        -> immutable idempotency proof, when a key is supplied
     -> validate typed commit receipt        [production Kernel]
     -> DurablePort.RecordResult            [best-effort external audit projection]
     -> DurablePort.ObserveResult           [best-effort, no decision authority]

The temporary DurablePort implementation lives in
`aios/controllane.Processor`; its combined `Process` method is used only by the
`legacy_v1` rollback mode and tests.
```

Boot selects one chain only. `FORGE_KERNEL_AUTHORITY_MODE=forge_k` is the
default. `legacy_v1` bypasses the FORGE-K ingress facade for rollback and must be
removed after final parity acceptance. No mode performs dual commits.

## Authority matrix

| Boundary | Current | Target | Exit gate |
| --- | --- | --- | --- |
| Kernel ingress | `forgekernel.Kernel` live by default | FORGE-K | Closed in K20A |
| Durable commit/journal orchestration | FORGE-K-owned `DurablePort`; Control Lane SQLite implementation | FORGE-K | Closed in K20B for stage ownership; K20D adds sealed plans, typed receipts, atomic integrity evidence, and verified replay |
| Tool need and selection | Deterministic FORGE chat routing | FORGE-K/Gateway policy contract | No model-selected schema or tool |
| Tool execution | Gateway | Gateway external driver | Keep single execution ingress |
| Courthouse | Production FORGE-K deterministic admission/rejection, immutable rulings, and appeals | FORGE-K | Closed in K20C; K20D makes its commit receipt, journal chain, provenance, audit intent, and optional idempotency proof atomic with the Court mutation |
| Semantic Algebra | Shape validation only | Governed live operations | Operation parity and journal tests |
| Memory Palace | Shadow/mirror plus current memory stores | Structured governed objects/routes | Current/history separation and migration proof |
| Context Compiler | Legacy `COMPILE_CONTEXT` plus attribution validation | FORGE-K bundles | Prompt parity, attribution, token-budget tests |
| Runtime Boundary | `modelruntime` proposal envelopes | FORGE-K driver contract over modelruntime | No model authority, cancellation/usage evidence |
| Consensus | Narrow final-response guard | Composition/admission guard | All response surfaces covered |
| Snapshots/replay | Existing backup/context snapshots | FORGE-K shape/replay authority | Restore seed, hash-chain, rollback tests |
| KV | Identity validation canary | Acceleration only | Exact identity plus invalidation proof |
| Lymphatic | Proposal metadata | Proposal-only maintenance lane | No silent mutation tests |

## Completion gates

- Every canonical mutation enters through the production FORGE-K Kernel.
- Every commit is atomic with its journal/provenance record.
- No production code constructs `controllane.Processor` outside the designated
  FORGE-K durable adapter assembly.
- No production code imports simulator services as live authority.
- Models cannot select tools, approve actions, execute tools, admit evidence, or
  mutate canonical state.
- Current truth and historical truth remain separately queryable.
- Replay detects journal divergence and cannot silently repair truth.
- Operator status reports the active authority owner and rollback posture.
- The complete test suite and native offline OptiPlex acceptance pass.
- `legacy_v1` mode and stale v1 authority documentation are removed only after
  all prior gates close.

## K20C Courthouse cutover

K20C promotes production-owned Courthouse contracts under
`services/core/internal/forgekernel/court`. `ADMIT_EVIDENCE` and
`APPEAL_RULING` pass ordinary capability, approval, scope, payload, and
idempotency preflight before the production Kernel computes the ruling. The
durable adapter may persist that typed decision but cannot invent one. Initial
exhibit/current-state rows, append-only ruling/appeal history, provenance, and
the semantic journal event share one SQLite transaction. Model actors,
proposal-only sources, and `legacy_v1` fail closed. The simulator Courthouse is
not imported.

K20D closes the canonical commit-integrity gap described by the K20C report.
The external audit sink and `audit_id` backfill still occur after the canonical
transaction and remain best-effort projections, but failures there cannot
invalidate or erase the immutable audit intent committed in the canonical
transaction.

## K20D commit integrity

K20D seals the exact post-Courthouse request and prepared plan before commit.
The temporary SQLite durable port returns a typed receipt binding that seal to
the transaction, committed object IDs, provenance IDs, journal event and hash,
audit-outbox record, and stable idempotency fingerprint. The production Kernel
validates the receipt before reporting success.

Semantic mutation, the sequenced journal hash-chain entry and head,
provenance, immutable audit-outbox intent, and immutable idempotency proof when
a caller supplies a key are committed in one SQLite transaction. Any failure
rolls back the whole unit. A matching retry reconstructs and validates the
original request, plan, seal, receipt, and result without re-committing. A
conflicting fingerprint or legacy unbound replay record fails closed.

K20D does not make FORGE-K the sole cognitive kernel. Control Lane still
implements validation/apply and the SQLite port, and Memory Palace, Semantic
Algebra, Context Compiler, Runtime, Snapshots, KV, Lymphatic, Consensus, direct
writers, and restore paths retain staged authority work.
