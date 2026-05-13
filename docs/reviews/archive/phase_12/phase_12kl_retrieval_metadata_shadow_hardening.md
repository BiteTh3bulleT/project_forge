# Phase 12K-L Retrieval Metadata Shadow Hardening

Status: implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / HARDENED_IN_PASS`.

## Summary

Phase 12K-L implements and hardens bounded retrieval metadata diagnostics in `services/core/internal/forgekshadow`.

The observer is disabled by default and requires both:

- `FORGE_K_SHADOW_MODE_ENABLED=true`
- `FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED=true`

The live daemon remains authority. The retrieval metadata hook runs only after the live `/api/retrieval/runs` handler has already created the retrieval run. FORGE-K does not execute retrieval, search, embeddings, live RAG, tools, modelruntime, memory writes, evidence admission, context compilation, or controllane mutations.

## Captured Metadata

Phase 12K-L captures bounded metadata only:

- retrieval run id
- first retrieval result id, when available
- workspace/request refs, when available
- source type/ref id, when available
- result count
- selected count
- bounded score summary
- ranking position
- retrieval strategy
- index type
- safe embedding model id
- duration
- diagnostic markers
- bounded warnings

Reports remain bounded in-memory diagnostics only. No public diagnostics route or persistent report store was added.

## Explicitly Not Captured

Phase 12K-L does not capture:

- source text
- chunk text
- document or file content
- raw user query text
- search snippets
- retrieval result bodies
- embeddings or vectors
- RAG output
- prompts
- model outputs
- request bodies
- response bodies
- memory content
- auth headers
- cookies
- bearer tokens
- API keys
- secret-looking metadata

## Hardening Coverage

Tests cover:

- retrieval metadata flag defaults disabled
- global flag and retrieval-specific flag matrix
- metadata-only diagnostic report creation
- forbidden metadata rejection
- unsafe typed refs/warnings rejection
- deterministic serialization
- bounded sink retention
- disabled sink behavior
- sink failure isolation
- no-effect policy rejection
- route inventory stability
- retrieval route response shape stability
- retrieval result count and selected count stability
- invalid request body, auth header, cookie, and query no-capture
- no public diagnostic routes
- forbidden `forgekshadow` imports remain blocked

## No-Effect Guarantees

Phase 12K-L does not change public APIs, routes, response status codes, response headers, response bodies, retrieval execution, search execution, embedding execution, modelruntime behavior, gateway behavior, permissions, lanes, audit, memory writes, controllane behavior, or user-visible output.

## Remaining Work

Future phases may consider broader metadata surfaces only with separate approval, tests, and rollback. Retrieval metadata diagnostics must not become authority, evidence admission, memory, ContextBlocks, runtime input, or live RAG.
