# Usefulness And Repair

FORGE tracks what memory helped and what memory hurt.

## Usefulness Signals

Legacy observation signals remain in `memory_usefulness_events`. K20G retrieval
result feedback is stored instead as immutable FORGE-K utility evidence with a
separate rebuildable projection.

Governed signal sources:
- retrieval-result usefulness through K20G immutable utility events
- restore-outcome feedback through K20G immutable utility events

Legacy operator observation usefulness is read-only history. Its API route is
a terminal `410 Gone` gate and its Go writer fails closed.

Tracked signal values include:
- `useful`
- `not_useful`
- `noisy`
- `insufficient`
- outcome-like variants (`success`, `failed`) where provided

## Score Aggregation

Historical observations may contain legacy summary counters:
- `usefulness_score`
- `usefulness_count`
- `noise_count`

K20G retrieval usefulness events update only
`retrieval_usefulness_projection`, which ranking may use as a bounded hint.
They do not mutate observations or VSA binding/association reliability
counters; VSA projections are rebuilt deterministically from their governed
source set.

## Drift / Staleness Controls

Each observation supports:
- `stale` flag
- `last_verified_at`
- `verification_state`

Legacy stale/re-verification mutation is retired. The fields remain readable;
new evidence revision requires the governed FORGE-K revision syscall.

## Repair Workflow

1. Inspect observation detail and immutable evidence history.
2. Request a deterministic repair preview with `dryRun=true`.
3. Review retrieval runs that repeatedly surface noisy memory.
4. Submit usefulness only through the governed retrieval utility surface.
5. Inspect VSA breakdown and governed projection provenance.
6. Submit any evidence revision through `REVISE_MEMORY_EVIDENCE`.

## Repair Runs

FORGE preserves historical repair passes for read compatibility:

- `memory_repair_runs`: run-level metadata and totals
- `memory_repair_items`: per-observation before/after snapshots and status

Live repair execution is retired. `POST /api/memory/repair/run` accepts only
explicit `dryRun=true` and returns a proposal without writing either table.
Non-dry requests and direct `RunRepairPass` calls fail closed. The scheduled
ticker runs the same preview-only selection.

## VSA Reindex Runs

FORGE also persists VSA-specific indexing maintenance:

- `memory_vsa_reindex_runs`: run-level mode/status/totals/settings snapshot
- `memory_vsa_reindex_items`: per-observation before/after fingerprints and status

Operator controls:
- run VSA reindex from `#/memory`
- inspect VSA run history and per-item transitions
- verify indexed/skipped/failed totals and notes

## Why This Exists

Without usefulness and repair:
- stale summaries silently dominate retrieval
- duplicate noise remains highly ranked
- packet quality drifts over time

FORGE keeps these corrections inspectable and persisted.

## Deferred

- automatic contradiction resolution
- scheduled stale sweep and auto-refresh jobs
- confidence decay over long inactivity windows
