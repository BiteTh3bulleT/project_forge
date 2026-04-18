# Retrieval and Embeddings

## Purpose

Provide inspectable retrieval behavior for packet and job preparation using:

- keyword retrieval (FTS)
- semantic retrieval (embeddings)
- hybrid fusion (weighted)

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
