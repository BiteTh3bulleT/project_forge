# Phase 12J Retrieval Metadata Risk Review

Status: implemented as part of `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

## Summary

Phase 12J compares possible shadow metadata surfaces after Phase 12I chat metadata hardening. Retrieval metadata may be a valid next controlled expansion only if it captures refs, IDs, counts, score summaries, fingerprints, strategies, status, timing, and bounded warnings from live-produced metadata. It must never capture source text, chunks, snippets, embeddings, vectors, raw query text, prompts, model outputs, memory content, request/response bodies, or RAG output.

At Phase 12J exit, no implementation was authorized by this review. Phase 12K-L later implemented only the bounded metadata observer/config path under these constraints.

## Risk Review Matrix

| Metadata Surface | Semantic Sensitivity | Source Text Leakage Risk | Embedding/Vector Leakage Risk | Prompt/Query Leakage Risk | RAG Adjacency | Memory-Write Adjacency | Gateway/Tool Adjacency | Modelruntime Adjacency | Testability | Recommended Constraints |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Route envelope metadata | Low | Low | Low | Low | Low | Low | Low | Low | High | Keep as the safest live observation class. Capture matched route pattern, route class, method, bounded timing, and request/correlation refs. Never capture raw path params, query strings, request URI, headers, bodies, response bodies, auth data, or payload-derived values. |
| Chat metadata | High | Medium | Low | High | Medium-High | Medium-High | Medium-High | High | Medium | Keep Phase 12H/12I scope only: identifiers, classes, bounded counts, timing, stream class, role class, and safe model/provider refs. Never capture message text, prompts, completions, tool payloads, tool outputs, retrieval content, memory content, request/response bodies, auth/cookies/query strings, or SSE tokens. |
| Retrieval metadata | High | High | High | High | High | High | Low-Medium | Medium | Medium | Valid future expansion only as already-produced live metadata: run/result refs, workspace/request/correlation refs, source type/ref, source fingerprint if already available, counts, selected count, score summaries, rank positions, strategy/index names, safe embedding model id, freshness/status/timing, and bounded warnings. Never execute retrieval/search/embedding from FORGE-K. Never capture query text, snippets, chunks, document content, embeddings, vectors, RAG output, prompts, model outputs, memory raw content, or request/response bodies. |
| Gateway trace metadata | Medium-High | Medium | Low | Medium | Medium | Medium | High | Medium | Medium | Defer. If later approved, capture only gateway decision refs/classes, capability class, approval state class, tool category class, timing/status, and bounded errors. Never capture tool arguments, tool outputs, artifact content, command strings containing user data/secrets, approval rationale bodies, credential material, filesystem content, external response bodies, or mutation payloads. Must not affect gateway authorization, approval, execution, or audit paths. |
| Modelruntime metadata | High | Low-Medium | Low | High | Medium | Medium | Medium | High | Medium-Low | Defer. If later approved, capture only model/provider refs, backend class, lifecycle/status class, token/count summaries if already produced, latency/status, scheduler/limit class, and bounded diagnostics. Never capture prompts, messages, completions, streamed tokens, tool-call arguments/results, request/response bodies, provider errors containing content, model weights, KV cache contents, logits, embeddings, or secret provider configuration. Must not call modelruntime from FORGE-K. |

## Retrieval Metadata Constraints

Retrieval metadata is higher risk than chat metadata because the live retrieval path may handle query text, source chunks, snippets, file refs, embedding vectors, ranking scores, selected packet evidence, VSA scoring, result persistence, context evidence insertion, and memory observation records. A future observer must therefore be ref-only and must attach only after live retrieval behavior has already produced safe metadata.

Recommended allowed fields from the Phase 12J design:

- retrieval run id
- retrieval result id
- workspace id
- request id
- correlation id
- source type
- source ref id
- source hash or fingerprint, only if already available and non-secret
- result count
- selected count
- score summary, rounded and bounded
- ranking position
- retrieval strategy name from a bounded enum
- index name or index type from a bounded enum
- embedding provider/model id only when non-secret and already exposed as safe metadata
- freshness or staleness flags
- timing/status metadata
- diagnostic markers
- bounded warnings

Required constraints:

- disabled by default behind a separate retrieval metadata flag
- metadata-only, ref-only, bounded in memory
- no persistence without separate approval
- no public diagnostics route
- no request body read
- no response body read
- no retrieval/search/embedding execution from FORGE-K
- no modelruntime calls from FORGE-K
- no gateway/tool execution from FORGE-K
- no memory writes
- no controllane mutations
- no evidence admission
- no context compilation
- no RAG output creation
- no change to route inventory, response status, headers, body, retrieval ordering, result count, memory writes, modelruntime calls, or gateway/tool counts
- shadow observer failure must not fail or alter the live request

## Forbidden Retrieval Metadata

A future retrieval metadata observer must reject, redact, or refuse any field that is content-bearing, secret-bearing, prompt-bearing, vector-bearing, or authority-bearing.

Forbidden fields and equivalents:

- raw user query string
- query text after normalization
- source text
- chunk text
- document content
- file content
- retrieval result body
- snippets, including highlighted snippets
- memory raw content
- memory summaries if derived from source text
- prompt text
- system/developer/user messages
- model output
- completions
- streamed tokens
- RAG output
- context compile content
- request bodies
- response bodies
- embeddings
- vectors
- vector JSON
- logits or KV cache contents
- auth headers
- cookies
- bearer tokens
- API keys
- raw authorization metadata
- provider secrets or endpoints containing credentials
- unredacted search snippets
- raw tool arguments
- raw tool outputs
- large raw content blobs
- secret-looking refs or metadata values
- arbitrary JSON payloads not validated against an allowlist

## Expected Conclusion

Retrieval metadata may be a valid next controlled shadow expansion only if Phase 12K captures refs, IDs, counts, score summaries, fingerprints, strategies, status, and timing from live-produced metadata. It must never capture source text, chunks, snippets, embeddings, vectors, raw query text, prompts, model outputs, memory content, request/response bodies, or RAG output.

The observer must not execute retrieval, search, embeddings, modelruntime, tools, controllane mutations, memory writes, evidence admission, or context compilation. FORGE-K remains non-authoritative; the live daemon remains the only authority path.
