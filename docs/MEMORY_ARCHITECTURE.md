# FORGE Memory Architecture

FORGE memory is governed evidence layered with historical observation data. It
does not use a single merged memory blob.

## Storage Backend Direction

Current live storage remains SQLite through `${FORGE_DATA_DIR}/forge.sqlite`.

The Docker stack now includes Postgres, Redis, and Qdrant as managed infrastructure for the next storage phase:

- Postgres is the intended durable relational backend for canonical memory, jobs, approvals, settings, audit-adjacent records, and retrieval metadata after explicit migration work.
- Redis is intended for ephemeral coordination: caches, queues, locks, pub/sub, and job progress streams. Redis must not become canonical truth.
- Qdrant is intended for vector retrieval acceleration. Qdrant vectors and scores must remain retrieval signals, not admissibility or truth.

Until storage adapters and migration tests are implemented, the live memory system still reads and writes SQLite.

Phase 13A introduces `FORGE_STORE_BACKEND=sqlite|postgres` and backend capability contracts, but it does not migrate live memory. Redis and Qdrant endpoint settings do not switch memory authority. Any future Postgres read switch requires parity tests, rollback, backup/restore coverage, and explicit approval.

## Memory Layers

1. `Cold memory`
- Raw observations, retrieved snippets, files, artifacts, logs, and job events.
- Persisted in SQLite with source references and lineage fields.

2. `Warm memory`
- Summaries, structural metadata, tags/entities, dossier scope, retrieval selection reasons, and usefulness stats.
- Includes retrieval result linkage and packet alignment notes.
- Includes governed evidence-FK VSA pointer/binding/association records used for inspectable additive reranking.

3. `Hot working memory`
- Active retrieval run results selected for packet assembly.
- Packet preview + alignment notes.

4. `Reflection layer`
- Usefulness/noise signals tied to observations and outcomes.
- Stale flags and re-verification metadata.

## Observation Store

Primary table: `memory_observations`.

Write status: K20H retires the legacy observation create, patch, usefulness,
link, and repair writers. Existing rows and repair history remain readable.
The three mounted legacy mutation routes are terminal `410 Gone` gates and no
production link route exists. New evidence is admitted and committed only
through production FORGE-K `MATERIALIZE_ADMITTED_EVIDENCE`; revisions preserve
the original through `REVISE_MEMORY_EVIDENCE`. The legacy exported Go methods
remain only as fail-closed source-compatibility seams.

Observation records include:
- id/timestamps
- type
- raw content
- summary
- source path
- dossier id
- tags/entities/related files
- task type
- confidence + verification state
- origin kind/id
- usefulness score + counts
- stale + last verified

Historical legacy relations:
- `memory_observation_links`
- `retrieval_result_observations`
- `memory_usefulness_events`
- `memory_vsa_pointers`
- `memory_vsa_role_bindings`
- `memory_vsa_associations`

Governed K20H evidence and acceleration use separate immutable/source-bound
tables: `forge_k_memory_evidence`,
`forge_k_memory_evidence_supersessions`, and `forge_k_memory_vsa_*`.

## Structural Metadata Routing

Retrieval and ranking use:
- dossier scope (`dossier_id`, linked sources)
- file path metadata (`abs_path`, `rel_path`)
- dossier profile biases (`high_value_files`, `noisy_files`)
- governed retrieval usefulness projections rebuilt from immutable K20G utility events; legacy result labels are ignored

## Retrieval Run Inspectability

Every run stores:
- query, mode, weighting
- ranked results
- selected-for-packet flag
- per-result selection reason JSON (`retrieval_result_selection`)
- linked observation id for each result (when generated)
- optional per-result VSA signal breakdown (`retrieval_result_vsa_signals`)

## VSA Inspectability

Operator UI/API now exposes:
- historical observation-level VSA detail without active-head authority
- governed memory-evidence VSA identity and active-manifest scoring
- retrieval-result VSA component scores (associative/role/relational/feedback/additive/applied)
- dossier-level VSA coverage/health summary
- persisted legacy VSA reindex run/item history

## Packet Alignment

Packet assembly stores alignment notes in `packet_alignment_notes`.

Each note explains why a retrieval result was included and can optionally link to:
- retrieval result id
- observation id

## Implemented vs Deferred

Implemented:
- read compatibility for historical observation metadata and links
- governed FORGE-K memory evidence materialization and immutable supersession
- selection reason persistence
- append-only FORGE-K retrieval-usefulness events and separately stored noncanonical score projections
- read compatibility for historical stale/verification and repair-run traces
- read compatibility for historical VSA reindex runs/items
- dossier memory view API
- packet alignment notes

Deferred:
- automated contradiction clustering
- automatic summary refresh scheduling
- embedding refresh orchestration based on stale observations

Repair is deterministic proposal inspection only. `PreviewRepairPass` selects
candidates and reports the rows a future governed action would affect; it does
not create repair runs, rewrite observations, or rebuild projections. The
background ticker also previews only.

## K20G Utility Boundary

Retrieval usefulness is not a Memory Palace observation mutation. Production
FORGE-K appends immutable `forge_k_retrieval_usefulness_events` and atomically
updates the disposable `retrieval_usefulness_projection`. The source retrieval
result, `memory_observations`, legacy `memory_usefulness_events`, and VSA
reliability counters remain unchanged. Ranking reads the governed projection
and ignores legacy mutable result labels.

Restore-outcome feedback follows the same split: the original
`restore_outcome_events` row remains evidence, immutable feedback events hold
history, and `restore_outcome_feedback_projection` is a separately labeled
noncanonical view. Original and projection must never be merged in reads.
