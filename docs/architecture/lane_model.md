# FORGE-K Lane Model

Status: Phase 10 simulator baseline. The Lymphatic Lane is implemented/tested in `services/core/internal/forgek` as `SIMULATOR_ONLY`.

FORGE-K uses a tri-lane system with a Hyperlane overlay. Lanes describe scheduling and authority boundaries. They do not grant bypass rights.

## Neural Lane

Responsibilities:

- model-driver proposals
- semantic interpretation
- intent proposals
- summarization drafts
- classification proposals
- action proposals that still require validation

Synchronous work:

- small intent proposals
- compact summarization or classification when already authorized by runtime policy

Deferred work:

- broad semantic exploration
- high-cost model calls
- speculative drafts
- non-urgent ranking

Authority limit: Neural Lane output is proposal material only.

## Arterial Lane

Responsibilities:

- case opening
- semantic syscall validation
- rule validation
- Courthouse admission
- Kernel commit
- journal append
- response assembly from admitted context

Synchronous work:

- scope validation
- capability checks
- approval gate checks
- compact context admission needed for the current response
- commit or rejection of requested semantic syscalls

Deferred work:

- non-critical enrichment
- large replay analysis
- deep contradiction review

Authority limit: Arterial Lane is the governed commit path, but only Kernel syscall transactions mutate canonical state.

## Lymphatic Lane

Responsibilities:

- deterministic maintenance sweeps
- contradiction sweeps
- stale-loop detection
- KV hygiene proposals
- snapshot hygiene proposals
- runtime result hygiene proposals
- orphan reference findings
- Maintenance Reports
- Cleanup Proposals
- stale route analysis

Synchronous work:

- only checks required to prevent immediate correctness failure

Deferred work:

- almost all cleanup review, compaction review, cache hygiene, contradiction sweep, and stale-loop work

Authority limit: Lymphatic work may produce dry-run reports and cleanup proposals, but it must not silently mutate source objects, delete provenance, resolve contradictions, expire snapshots, evict KV manifests, submit evidence, admit evidence, or call model runtimes. Any canonical mutation still requires explicit semantic syscalls, capability checks, Kernel commit boundaries, and journal evidence.

## Hyperlane

Hyperlane is a deterministic reflex-routing overlay. It runs CPU-local rules for obvious decisions before expensive context assembly, runtime-driver calls, gateway execution, or Lymphatic processing.

Hyperlane may:

- classify obvious requests
- produce route hints
- avoid unnecessary model calls
- flag degraded runtime states
- suggest deferral

Hyperlane must not:

- commit truth
- admit evidence by itself
- execute tools
- call modelruntime
- weaken capability, approval, scope, or policy rules

## Hot, Warm, and Cold Path

| Path | Expected work | Examples |
|---|---|---|
| Hot path | Minimal deterministic work needed now | scope check, case open, small retrieval, rule validation, admission, syscall result |
| Warm path | Useful but not always required immediately | context compiler expansion/contraction, snapshot creation, cache eligibility |
| Cold path | Maintenance and compaction | contradiction sweeps, stale-loop detection, cache eviction, snapshot compaction |

Rule: Do not run the full architecture on every turn. The hot path must stay small.
