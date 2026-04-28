# Context Restore Scoring

Status date: 2026-04-24.

`COMPILE_CONTEXT` can rank persisted context snapshots before compiling a new packet. Snapshot rows are non-canonical evidence: canonical truth remains in notes, links, state, loops, artifacts, models, provenance, and journal tables.

Memory taxonomy reference: [memory_taxonomy.md](memory_taxonomy.md).

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

## Memory Taxonomy Role

Restore scoring is the arterial recall/score function for working, structural, semantic, prospective, episodic, and utility memory.

It consumes:

- working memory from active context packets and snapshot headers
- structural memory from snapshot fingerprints, links, artifacts, selected paths, and lineage
- semantic memory from notes/state/links included in prior snapshots
- prospective memory from open loop overlap
- episodic memory from journal/event evidence and snapshot creation history
- utility memory from `restore_outcome_events`

It does not make any of those sources canonical. Restore scoring chooses evidence for context assembly and may require fresh compile. Canonical mutation still requires semantic syscalls.

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

## Rule Cell Adjustments

Phase 7 v0 adds optional Arterial Rule Cell adjustments after deterministic base scoring and before the threshold/fresh-compile decision.

Rule Cells may:

- boost exact or overlapping query matches
- penalize stale or contradiction-marked snapshots
- emit `FreshCompileRequired` for low base scores
- emit warnings/reject outputs for wrong-workspace candidates

Rule Cells are not the workspace boundary. Wrong workspace candidates are excluded by the store/query/scoring flow before Rule Cells run.

Score safety:

- individual restore `ScoreAdjustment` is capped at `0.06`
- total rule-based restore adjustment is capped at `0.12`
- final restore score is clamped to `0.0..1.0`

If the rule engine fails, `COMPILE_CONTEXT` appends an explicit warning and continues deterministic base scoring. Rule traces are persisted only inside existing non-canonical restore metadata and include rule pack id/version.

## Outcome Feedback

Phase 8 adds `restore_outcome_events` as non-canonical evidence for the restore loop. A persistent `COMPILE_CONTEXT` run emits one outcome event after snapshot selection with:

- selected context packet and source snapshot IDs
- selected evidence, state keys, loop IDs, and artifact IDs
- assigned restore score and `requires_fresh_compile`
- restore decision, correlation, trace, syscall, and audit linkage
- initial outcome `unknown`, `no_candidate`, or `fresh_compile_required`

Read-only compile paths and semantic dry-runs do not write outcome rows. They may return a draft outcome in the response summary so operators can inspect what would have been recorded.

Operator feedback can update the non-canonical event outcome, confidence, feedback text, and correction summary through the restore outcome API. This update does not mutate canonical notes, state, loops, links, or memory truth.

Future scoring consumes prior outcome evidence as a bounded utility signal:

- prior helpful evidence/snapshot: small boost
- stale, harmful, contradictory, failed, or operator-corrected outcomes: penalty/review signal
- repeated `fresh_compile_required` or `no_candidate` for the same query: lower confidence and memory-gap signal

Safety caps:

- individual outcome utility adjustment is capped at `0.04`
- total outcome utility adjustment is capped at `0.08`
- final restore score remains clamped to `0.0..1.0`

Outcome lookup failures append an explicit warning and `COMPILE_CONTEXT` continues deterministic base scoring.

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

## Operator Inspector API

Status: IMPLEMENTED.

Restore inspector routes are read-only and require workspace scope for the `/api/context/restore/*` surface:

- `GET /api/context/restore/recent?workspaceId=<id>&laneId=<lane>&limit=20`
- `GET /api/context/restore/<snapshot-id>?workspaceId=<id>&laneId=<lane>`
- `GET /api/context/restore/<snapshot-id>/candidates?workspaceId=<id>&laneId=<lane>`
- `GET /api/context/restore/<snapshot-id>/score?workspaceId=<id>&laneId=<lane>`
- `GET /api/context/restore/<snapshot-id>/resume-hints?workspaceId=<id>&laneId=<lane>`

Responses label restore rows as `non_canonical_evidence` and set `canonicalWriteCommitted=false`.
Wrong-workspace IDs return not found instead of leaking snapshot evidence. These routes do not apply
Dream proposals, execute tools, call modelruntime, or mutate canonical memory/state.

## Restore Scoring Cache

Phase 8.1 adds a bounded in-memory scoring cache for deterministic restore selection. The cache stores the ranked selection summary, not canonical truth.

The key includes workspace id, lane id, normalized query, snapshot kind, restore hints, candidate set fingerprint, and restore outcome fingerprint. New snapshots, outcome feedback changes, wrong workspace/lane, hint changes, TTL expiry, or an explicit `restoreCacheDisabled=true` compile option produce a miss.

The cache is used after scoped candidate listing, so wrong-workspace exclusion remains enforced by store/query logic. Trace metadata includes `cache.hit` and a compact key fingerprint.

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
