# FORGE Memory Architecture

FORGE memory is observation-based. It does not use a single merged memory blob.

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
- Includes VSA pointer/binding/association records used for inspectable additive reranking.

3. `Hot working memory`
- Active retrieval run results selected for packet assembly.
- Packet preview + alignment notes.

4. `Reflection layer`
- Usefulness/noise signals tied to observations and outcomes.
- Stale flags and re-verification metadata.

## Observation Store

Primary table: `memory_observations`.

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

Relations:
- `memory_observation_links`
- `retrieval_result_observations`
- `memory_usefulness_events`
- `memory_vsa_pointers`
- `memory_vsa_role_bindings`
- `memory_vsa_associations`

## Structural Metadata Routing

Retrieval and ranking use:
- dossier scope (`dossier_id`, linked sources)
- file path metadata (`abs_path`, `rel_path`)
- dossier profile biases (`high_value_files`, `noisy_files`)
- historical usefulness labels from prior results

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
- observation-level VSA pointer/binding/association detail
- retrieval-result VSA component scores (associative/role/relational/feedback/additive/applied)
- dossier-level VSA coverage/health summary
- persisted VSA reindex run/item history

## Packet Alignment

Packet assembly stores alignment notes in `packet_alignment_notes`.

Each note explains why a retrieval result was included and can optionally link to:
- retrieval result id
- observation id

## Implemented vs Deferred

Implemented:
- observation persistence with metadata
- retrieval-result-to-observation linking
- selection reason persistence
- usefulness event recording and score aggregation
- stale flags + verification timestamp updates
- persisted repair runs with before/after item traces
- persisted VSA reindex runs/items with fingerprint transitions
- dossier memory view API
- packet alignment notes

Deferred:
- automated contradiction clustering
- automatic summary refresh scheduling
- embedding refresh orchestration based on stale observations
