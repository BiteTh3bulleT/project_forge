# FORGE-K Live Cutover

Status: K20A-K20G active; full cutover in progress.

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
| Utility evidence | FORGE-K append-only retrieval usefulness and restore-outcome feedback events; separate rebuildable projections | FORGE-K | K20G closes the mutable feedback writers; retrieval job-outcome recording remains fail-closed pending an exact batch contract |
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
Algebra, Context Compiler, Runtime, Snapshots, KV, Lymphatic, Consensus, and
other direct writers retain staged authority work.

## K20E authenticated authorization proof

The production Kernel now resolves authorization before `Prepare` and seals a
typed proof of the authenticated principal, action-registry definition,
scope-exact capability grant, and approval policy/decision. System and
internal requests bind the one `forge.core` service principal constructed at
daemon assembly. User, adapter, and future-IRIS sources require an
authenticated origin from request context; caller-supplied actor/source maps
are attribution, not authority. Adapter and future-IRIS mutations remain
proposal-only and require an approved durable decision from a separate actor.

Bearer authentication contributes a non-secret credential fingerprint and
session identity. If API token authentication is explicitly absent, only a
verified loopback peer receives `local_loopback` origin evidence; arbitrary
remote callers receive no authenticated origin and therefore cannot authorize
a user/proposer semantic syscall.

The Kernel verifies that the prepared plan matches the authorization record,
then places the full proof in the atomic idempotency and audit-outbox records.
The audit intent also stores the exact authorized request and revalidates its
proof, row identity, and commit-receipt request fingerprint. Replay restores
the original request, plan, seal, receipt, and authorization proof and requires
the retry to resolve to the same immutable authority records. Missing,
legacy-unbound, swapped, or tampered evidence fails closed without commit.

## K20E restore apply retirement

K20E removes backup restore from the live mutation boundary while a safe
Kernel-owned recovery contract is absent. `POST /api/backup/restore` is now a
dry-run inspection surface only. Every non-dry request returns
`FORGE_K_RESTORE_APPLY_DISABLED` before path access, approval creation, or
state mutation. The backup service applies the same fail-closed rule to direct
callers, and the live I/O lane exposes inspection rather than restore apply.

Inspection binds the raw bundle SHA-256 to normalized effective sections,
computed counts and checksums, declared manifest policy and authority, and a
deterministic plan digest. Each section reports a disposition and blockers.
Journal events and idempotency proof are never live-mergeable; Courthouse
state, journal head, and immutable audit-outbox intent are offline-recovery
only. Historical/provenance evidence requires quarantine or a dedicated
semantic migration rather than raw upsert.

`full_backup` exports K20C/K20D Courthouse, journal-head, audit-outbox, and
idempotency state for inspection and future whole-store recovery. Export does
not grant restore authority. Live apply remains disabled until a daemon-stopped,
whole-store recovery path verifies the bundle/manifest, SQLite integrity,
journal chain and head, immutable proof state, workspace identity, and an exact
rollback procedure before atomically replacing the store.

## K20G append-only utility evidence

K20G replaces mutable retrieval-usefulness and restore-outcome feedback with
`RECORD_RETRIEVAL_USEFULNESS` and
`RECORD_RESTORE_OUTCOME_FEEDBACK`. Both actions require production FORGE-K
ingress, exact workspace/lane/selected-path identity, source evidence bound to
a prior syscall and provenance, and an idempotency key. Adapter, future-IRIS,
model-proposer, legacy-unbound, and `legacy_v1` attempts fail closed.

Each accepted request appends an immutable utility event in the same SQLite
transaction as provenance, the journal hash-chain entry/head, canonical audit
intent, and idempotency proof. The original `retrieval_results` and
`restore_outcome_events` rows are never rewritten. Separately labeled
`retrieval_usefulness_projection` and
`restore_outcome_feedback_projection` rows are explicitly noncanonical and
rebuildable. Reads expose the original restore evidence and its
`feedbackProjection` separately; governed retrieval projections ignore legacy
mutable usefulness columns.

Usefulness events do not mutate Memory Palace observations or VSA reliability
counters. Retrieval job-outcome recording is fail-closed because its legacy
signature cannot carry exact scope, actor, provenance, and idempotency; a
future batch syscall must establish those bindings before that writer returns.
