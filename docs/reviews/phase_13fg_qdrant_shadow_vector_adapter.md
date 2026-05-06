# Phase 13F-G Qdrant Shadow Vector Adapter

Status: implemented and tested.

Scope: `LIVE_INFRA / VECTOR_SHADOW / DISABLED_BY_DEFAULT / NON_AUTHORITATIVE`.

## Summary

Phase 13F-G adds a Qdrant adapter scaffold and disabled-by-default shadow vector index under `services/core/internal/vectorstore`.

Qdrant remains retrieval acceleration only. It is not the live retrieval backend, not canonical truth, not memory, not evidence admission, and not a provenance authority.

## Config And Flags

- `FORGE_QDRANT_SHADOW_INDEX_ENABLED=false`
- `FORGE_QDRANT_URL`
- `FORGE_QDRANT_COLLECTION=forge_shadow_embeddings`
- `FORGE_QDRANT_VECTOR_SIZE`
- `FORGE_QDRANT_TIMEOUT_MS=3000`

Enabling shadow indexing requires a configured Qdrant URL and does not switch `FORGE_STORE_BACKEND`.

## Adapter Behavior

The `VectorStore` interface supports:

- collection creation,
- vector upsert,
- vector search for shadow/test use only,
- vector delete,
- health checks.

The Qdrant HTTP adapter is generic and is not imported by live retrieval. Normal tests use fake stores or HTTP test servers. Optional live Qdrant checks are gated by `FORGE_QDRANT_TEST_URL`.

## Shadow Index Behavior

`ShadowIndexService` accepts already-produced vectors and safe refs. It:

- skips all work when disabled,
- validates payload safety before upsert,
- validates vector dimensions,
- can ensure a collection when explicitly configured,
- generates deterministic point IDs when a point ID is not supplied,
- returns Qdrant errors to callers without mutating relational records.

It cannot create embeddings, execute retrieval, call search providers, change ranking, admit evidence, write memory, or change live response behavior.

## Allowed Payload

Qdrant payload metadata may contain:

- object id/ref,
- source ref id,
- workspace id,
- embedding record id,
- embedding model id,
- embedding dims,
- source hash/fingerprint,
- retrieval strategy or index class,
- created timestamp,
- schema version,
- provenance ref id,
- bounded safe ref metadata.

## Forbidden Payload

Qdrant payload metadata must not contain:

- source text,
- chunk text,
- document content,
- prompt text,
- completion text,
- message bodies,
- tool payloads or outputs,
- memory content,
- raw queries,
- auth/cookie/token/secret values,
- API keys or passwords,
- large raw content blobs.

## Rebuild Requirement

Qdrant indexes must be rebuildable from relational embedding records. Relational storage remains the source of record for provenance and inspectability. Any future destructive rebuild command must be explicit, validate collection dimensions and model identity, and remain separate from live retrieval execution.

## Tests

Implemented tests cover:

- config defaults and fail-closed enabled behavior,
- live backend non-switching,
- valid safe payloads,
- forbidden payload rejection,
- oversized payload rejection,
- deterministic point IDs,
- disabled shadow index no-op behavior,
- enabled fake-store upsert behavior,
- no retrieval or embedding execution contract,
- dimension validation,
- Qdrant HTTP payload shape,
- optional Qdrant integration gating.

## Live Behavior

Live retrieval is unchanged. Qdrant is not authoritative and is not used for live retrieval reads. No public API, route behavior, gateway behavior, modelruntime behavior, memory semantics, or canonical store behavior changed.
