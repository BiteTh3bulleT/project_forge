# FORGE-K Live Cutover

Status: K20A-K20B active; full cutover in progress.

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
     -> DurablePort.Commit                  [one atomic apply+journal transaction]
     -> DurablePort.RecordResult            [audit + lineage linkage]
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
| Durable commit/journal orchestration | FORGE-K-owned `DurablePort`; Control Lane implementation | FORGE-K | Closed in K20B: stage order, parity, replay, denial, approval, rollback |
| Tool need and selection | Deterministic FORGE chat routing | FORGE-K/Gateway policy contract | No model-selected schema or tool |
| Tool execution | Gateway | Gateway external driver | Keep single execution ingress |
| Courthouse | Candidate validation only | Live admission/ruling authority | Evidence provenance and appeal tests |
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

## Next bounded slice: K20C

Promote Courthouse admission/ruling contracts behind the production Kernel.
K20C must preserve evidence provenance, deterministic admission policy,
appeal/rejection history, current-versus-historical truth separation, and no
model admission authority. The simulator Courthouse remains isolated; only
pure contracts or production-owned implementations may cross the boundary.
