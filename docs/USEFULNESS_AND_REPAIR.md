# Usefulness And Repair

FORGE tracks what memory helped and what memory hurt.

## Usefulness Signals

Signals are stored in `memory_usefulness_events`.

Signal sources:
- retrieval result usefulness marking
- memory observation marking from operator UI
- outcome tagging for selected retrieval context

Tracked signal values include:
- `useful`
- `not_useful`
- `noisy`
- `insufficient`
- outcome-like variants (`success`, `failed`) where provided

## Score Aggregation

Each observation keeps summary counters:
- `usefulness_score`
- `usefulness_count`
- `noise_count`

Scores are updated on each usefulness event and used as ranking hints.
When VSA data exists for an observation, usefulness/noise feedback also updates VSA binding/association reliability counters.

## Drift / Staleness Controls

Each observation supports:
- `stale` flag
- `last_verified_at`
- `verification_state`

Operators can mark stale or re-verified from the Memory page.

## Repair Workflow

1. Inspect observation detail.
2. Mark stale/noisy/useful explicitly.
3. Review retrieval runs that repeatedly surface noisy memory.
4. Adjust dossier high-value/noisy file lists.
5. Inspect VSA breakdown for noisy results (associative/role/relational/feedback).
6. Trigger VSA reindex if pointers/bindings are stale or missing.
7. Re-run retrieval and verify selection reasons.

## Repair Runs

FORGE now persists repair passes:

- `memory_repair_runs`: run-level metadata and totals
- `memory_repair_items`: per-observation before/after snapshots and status

Run modes:
- `manual`: operator-triggered from Memory UI
- `scheduled`: background ticker pass in core runtime

Operator controls:
- run repair now from `#/memory`
- inspect run history and per-item outcomes
- verify repaired/skipped/failed counts and notes

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
