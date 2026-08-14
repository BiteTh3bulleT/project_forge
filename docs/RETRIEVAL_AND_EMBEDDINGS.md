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

Phase 13F-G adds a Qdrant shadow vector adapter and disabled-by-default shadow index scaffold under `services/core/internal/vectorstore`. It accepts already-produced embedding vectors plus safe relational refs only. It does not create embeddings, execute retrieval, change live result ordering, admit evidence, write memory, or make Qdrant the source of record.

Qdrant shadow indexing is controlled by:

- `FORGE_QDRANT_SHADOW_INDEX_ENABLED=false`
- `FORGE_QDRANT_URL`
- `FORGE_QDRANT_COLLECTION=forge_shadow_embeddings`
- `FORGE_QDRANT_VECTOR_SIZE`
- `FORGE_QDRANT_TIMEOUT_MS=3000`

Allowed Qdrant payload metadata is limited to object/source/workspace refs, embedding record/model/dims, source hash or fingerprint, retrieval strategy or index class, creation time, schema version, and provenance refs. Forbidden payload metadata includes source text, chunk text, document content, prompts, completions, message bodies, tool payloads or outputs, memory content, raw queries, auth data, cookies, tokens, secrets, vectors as payload fields, and large raw blobs.

The Qdrant index must be rebuildable from relational embedding records. Any future destructive rebuild command must be explicit, validate collection dimensions and model identity, and remain separate from live retrieval execution.

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

Marking a retrieval result as `useful`, `not_useful`, `noisy`, or `insufficient` through K20G:

- appends an immutable `forge_k_retrieval_usefulness_events` row through production FORGE-K
- updates only `retrieval_usefulness_projection`, which is explicitly noncanonical and rebuildable
- does not rewrite `retrieval_results`, memory observations, legacy usefulness events, context evidence, or VSA counters

Job terminal outcome evidence is temporarily fail-closed until an exact scoped
batch syscall carries job/run/result identity, actor, provenance, and
idempotency.

## Packet Integration

Job packet preparation can consume selected retrieval-run items directly.

- retrieval run created during packet prep
- selected results attached to packet context
- run-to-packet linkage persisted for traceability

VSA data remains inspectable even when mode is `shadow` (no ranking mutation).
