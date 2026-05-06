# Retrieval and Embeddings

## Purpose

Provide inspectable retrieval behavior for packet and job preparation using:

- keyword retrieval (FTS)
- semantic retrieval (embeddings)
- hybrid fusion (weighted)
- optional VSA additive scoring (shadow/active)

## Vector Store Direction

The current live retrieval path stores embedding metadata and vectors in SQLite-backed `embedding_records`, with deterministic local hash embeddings available as the default provider.

The Docker stack now includes Qdrant as a managed vector-store service for the next storage phase. Qdrant is not yet wired into the live retrieval service. When it is wired, it must remain retrieval acceleration only:

- vectors are not truth
- vector hits are not admitted evidence
- Qdrant scores cannot bypass Courthouse, memory usefulness evidence, policy, or Kernel authority
- SQLite/Postgres records remain the provenance and inspectability source of record
- Qdrant indexes must be rebuildable from relational records

Phase 13A does not wire Qdrant into live retrieval. `FORGE_QDRANT_URL` is infrastructure config only, and vector hits remain non-authoritative retrieval acceleration.

## Embedding Pipeline

Module: `services/core/internal/embeddings`

- configurable provider via settings
  - `embedding_provider` (`local_hash` or `ollama`)
  - `embedding_model`
  - `embedding_dims`
- endpoint support:
  - `GET /api/embeddings/status`
  - `POST /api/embeddings/reembed`
- persistence:
  - `embedding_records` with provider/model/content hash/status metadata

### Providers

- `local_hash`
  - deterministic local embedding fallback
  - no external dependency
- `ollama`
  - uses `/api/embeddings` against configured local Ollama endpoint

## Retrieval Runs

Module: `services/core/internal/retrieval`

- endpoint support:
  - `POST /api/retrieval/runs`
  - `GET /api/retrieval/runs`
  - `GET /api/retrieval/runs/{id}`
  - `POST /api/retrieval/results/{id}/usefulness`

Persisted in:

- `retrieval_runs`
- `retrieval_results`
- `packet_retrieval_runs`

Each run stores:

- query
- mode
- weighting snapshot
- dossier/job/packet links
- ranked results with score components
- selected-for-packet flags

## Hybrid Ranking

Default weighted fusion uses settings keys:

- `retrieval_weight_keyword`
- `retrieval_weight_semantic`

Hybrid score is an explicit weighted combination of normalized keyword and semantic components.

No black-box ranking: score components are visible in API and UI.

## VSA Settings + Signals

VSA behavior is controlled by settings:

- `retrieval_vsa_mode` (`off`, `shadow`, `active`)
- `retrieval_vsa_dims`
- `retrieval_vsa_seed`
- `retrieval_vsa_weight_associative`
- `retrieval_vsa_weight_role_match`
- `retrieval_vsa_weight_relational`
- `retrieval_vsa_weight_feedback`
- `retrieval_vsa_max_additive`

Persisted inspectability:

- `retrieval_result_vsa_signals` stores per-result VSA score components and explain payload.
- Retrieval UI renders component-level breakdown and matched observation context when present.

## Usefulness Evidence

Marking a retrieval result as `useful`, `not_useful`, `noisy`, or `insufficient` updates:

- `retrieval_results.usefulness_*`
- `context_evidence` rows for downstream insights

Job terminal outcomes additionally emit evidence for selected packet results.

## Packet Integration

Job packet preparation can consume selected retrieval-run items directly.

- retrieval run created during packet prep
- selected results attached to packet context
- run-to-packet linkage persisted for traceability

VSA data remains inspectable even when mode is `shadow` (no ranking mutation).
