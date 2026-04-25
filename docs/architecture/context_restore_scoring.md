# Context Restore Scoring

Status date: 2026-04-24.

`COMPILE_CONTEXT` can rank persisted context snapshots before compiling a new packet. Snapshot rows are non-canonical evidence: canonical truth remains in notes, links, state, loops, artifacts, models, provenance, and journal tables.

## Authority

- Entry action: `COMPILE_CONTEXT`
- Authority boundary: semantic syscall validator, processor, transactional semantic store, audit linkage
- Persistence target: `context_packet_snapshots`
- Governed commit scoring persists `restore_scores_json`, `resume_hints_json`, and the header-first restore package metadata on snapshot rows.
- `dryRun=true` on the semantic syscall remains validation-only and does not write snapshot rows; use `persistSnapshot=false` for the historical non-restore compile path.

## Candidate Model

Candidate listing is header-first and scope-bound. A candidate exposes:

- `snapshot_id`, `context_packet_id`, `snapshot_kind`, `query`
- `workspace_id`, `lane_id`, `selected_paths`, `created_at`
- `snapshot_fingerprint`, `parent_snapshot_id`
- header availability, graph availability, delta availability
- `render_artifact_ref_id`
- lineage: `correlation_id`, `trace_id`, `syscall_id`, `audit_id`, `proposed_by`, `committed_by`

Wrong workspace candidates are excluded. Requested lane matches the same lane, and an empty requested lane may match any lane in the workspace.

## Scoring

All scoring is deterministic lexical/scoped scoring. There is no LLM, modelruntime, GPU, or vector dependency.

Weights:

- query score: `0.20`
- scope score: `0.20`
- snapshot kind score: `0.10`
- recency score: `0.15`
- lineage score: `0.10`
- state overlap score: `0.10`
- loop overlap score: `0.10`
- artifact overlap score: `0.05`

Bonuses:

- fingerprint match bonus: `0.05`
- preferred snapshot hint bonus: `0.15`

Penalties:

- contradiction penalty from conflict-marked graph nodes
- staleness penalty after one day
- freshness penalty after fourteen days
- header-only penalty when the graph body is unavailable

The score object includes `total_score`, `confidence`, `requires_fresh_compile`, and non-empty `explain[]`.

## Query Matching

Query matching lowercases, trims whitespace, removes trivial punctuation, tokenizes, and compares:

- exact normalized query
- token overlap
- token subset/prefix matches

Embeddings are not used as truth authority.

## Header-First Package

Restore scoring returns a package with:

- selected snapshot and context packet IDs
- restore confidence
- `requires_fresh_compile`
- selected score and score breakdown
- selected header
- resume hints
- compact selected evidence refs
- candidate summaries
- trace

Full graph/delta expansion is opt-in through `expandRestoreGraph`.

## Fresh Compile

`requires_fresh_compile=true` means the current restore candidates are not reliable enough for direct resume. This is set when no candidate exists, the top score is below threshold, restore hints force a fresh compile, or the candidate is hard-stale.

Fresh compile does not discard snapshots. It records why a new compile was chosen.

## Options

`COMPILE_CONTEXT` accepts restore options at payload root, `compileOptions`, or `restoreSnapshot`:

- `restoreMode`
- `restoreSnapshotKind`
- `restoreCandidateLimit`
- `restoreMinScore`
- `requireFreshCompileBelowThreshold`
- `expandRestoreGraph`
- `resumeHints`

`persistSnapshot=false` keeps the historical non-restore behavior. Restore scoring metadata is produced on the governed snapshot persistence path.
