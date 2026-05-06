# Phase 12J Retrieval Metadata Expansion Design

Status: implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

Current note: Phase 12K-L later implemented and hardened the approved bounded retrieval metadata observer boundary. This document remains the Phase 12J design record.

## Executive Summary

Phase 12J designed a possible future retrieval metadata shadow expansion. It did not implement retrieval metadata observation, add live touchpoints, add routes, add feature flags, or change live daemon behavior.

At Phase 12J exit, the implemented shadow surfaces were `/health` metadata, route-envelope metadata, and bounded chat metadata. Phase 12K-L later implemented bounded retrieval metadata diagnostics under the constraints defined here.

Core rule: retrieval metadata is routing and evidence metadata, not truth. It must not admit evidence, compile context, run retrieval, call embeddings, write memory, create RAG output, or affect responses.

## Current Phase 12B Through 12I State

- Phase 12B implements disabled-by-default `/health` request metadata diagnostics.
- Phase 12C hardens `/health` diagnostics.
- Phase 12D designs controlled shadow expansion.
- Phase 12E implements disabled-by-default route-envelope metadata diagnostics.
- Phase 12F hardens route-envelope diagnostics.
- Phase 12G designs chat metadata expansion.
- Phase 12H implements disabled-by-default chat metadata diagnostics.
- Phase 12I hardens chat metadata diagnostics.

At Phase 12J exit, no implemented shadow path observed retrieval routes, search results, embeddings, chunks, source text, memory content, prompts, model output, request bodies, or response bodies. Phase 12K-L later implemented bounded post-run retrieval metadata diagnostics only; it still does not capture retrieval/search/embedding content, source/chunk text, raw queries, prompts, model output, request bodies, or response bodies.

## Why Retrieval Metadata Is Higher Risk Than Chat Metadata

Retrieval metadata is closer to evidence selection, source material, embedding records, memory observations, VSA scores, context assembly, and RAG behavior than chat metadata. Even metadata can become risky if it includes raw user queries, source snippets, chunk text, embeddings, vectors, or unredacted search payloads.

The risk is not only leakage. Retrieval diagnostics can be mistaken for authority if they are later used to decide what is true, which evidence is admitted, or what context should be compiled. Phase 12J therefore keeps retrieval metadata as design-only and requires future tests proving that shadow output cannot affect retrieval, context compilation, memory, modelruntime, gateway execution, or user-visible responses.

## Proposed Future Phase 12K Scope

Future Phase 12K candidate scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Candidate touchpoint:

- retrieval/search/embedding metadata already produced by live paths

The observer, if approved later, must observe only existing records or already-computed metadata. FORGE-K must not initiate retrieval, search, embedding, VSA, context compilation, modelruntime, gateway, memory, or controllane work.

## Allowed Metadata

Future Phase 12K may capture only bounded, stable metadata that is already available without reading source text, chunk text, raw queries, vectors, prompts, model outputs, request bodies, or response bodies:

- retrieval run id
- retrieval result id
- workspace id
- request id or correlation id
- source type
- source ref id
- source hash or fingerprint if already available
- result count
- selected count
- score summary
- ranking position
- retrieval strategy name
- index name or index type
- embedding model id if already safe and non-secret
- freshness or staleness flags
- timing and status metadata
- diagnostic markers
- bounded warnings

All identifiers and model/index names must pass secret-looking and size-bound checks. Scores must be summaries or scalar values, not source content or query text.

## Forbidden Metadata

Future Phase 12K must not capture:

- source text
- chunk text
- document content
- file content
- prompt text
- model output
- retrieval result body
- embeddings
- vectors
- raw user query text
- raw query strings when user-authored or sensitive
- auth headers
- cookies
- bearer tokens
- API keys
- raw authorization metadata
- memory content
- unredacted search snippets
- source chunks
- large raw content blobs
- secrets or secret-looking metadata

Metadata that cannot be proven safe must be dropped or must reject the diagnostic report.

## No-Effect Guarantees

Future Phase 12K must prove:

- feature flag defaults disabled
- disabled mode observes no retrieval metadata
- enabled mode does not change response status, headers, or body
- route inventory is unchanged
- no public diagnostics route is added
- live retrieval behavior is unchanged
- retrieval result count and ordering are unchanged
- context compile behavior is unchanged
- modelruntime call count is unchanged
- gateway/tool execution count is unchanged
- memory write count is unchanged
- controllane mutation count is unchanged
- shadow failure cannot fail the live request

Shadow diagnostics remain non-authoritative and cannot affect retrieval selection, evidence admission, response composition, model selection, gateway execution, approvals, lanes, audit authority, memory writes, Context Compiler behavior, or Kernel truth.

## RAG Boundary

Phase 12J does not authorize live RAG. Future retrieval metadata diagnostics may describe refs and already-computed scores, but they must not:

- run retrieval from FORGE-K
- call search from FORGE-K
- call embedding providers from FORGE-K
- create or alter RAG output
- compile context
- admit evidence
- promote retrieved content to truth
- write memory
- alter response composition

Read-only retrieval metadata is a diagnostic report only. It is not a ContextBlock, not admitted evidence, not canonical memory, and not a model prompt.

## Data Capture Policy

Retrieval metadata capture must be metadata-only and ref-only.

Allowed records should prefer:

- source refs over paths or text
- fingerprints over content
- counts over item bodies
- score summaries over full payloads
- ranking positions over snippets
- strategy names over raw queries
- diagnostic warnings over raw errors

No future retrieval metadata observer may parse request bodies, response bodies, source files, chunk content, embedding arrays, or raw query text to find metadata unless a separate phase explicitly approves a parser, redaction model, and tests proving no content retention. Phase 12J does not approve that work.

## Redaction Policy

Future Phase 12K must reuse or strengthen Phase 12I redaction:

- reject secret-looking keys or values
- reject raw content keys such as body, prompt, completion, content, message, source text, chunk text, snippet, document text, file contents, embedding, vector, and raw query
- reject auth, cookie, session, token, password, credential, bearer, secret, API key, private key, JWT, access token, refresh token, and set-cookie indicators
- reject raw query strings and request URI values
- enforce bounded string length
- reject non-deterministic metadata values
- drop the entire report on unsafe metadata

## Performance Budget

Future Phase 12K must remain cheap:

- disabled path near zero overhead
- no blocking retrieval, search, embedding, modelruntime, gateway, memory, or controllane calls
- no request or response body reads
- no source/chunk/vector reads
- bounded report construction time
- bounded report size
- bounded in-memory retention
- no stream buffering

## Failure Isolation

Future retrieval metadata shadowing must fail closed:

- redaction failure drops the report
- sink failure drops the report
- no-effect validation failure drops the report
- observer panic is recovered
- live request continues through the existing live path

No shadow failure may change status, headers, body, route selection, timeout behavior, SSE behavior, retrieval behavior, embedding behavior, modelruntime behavior, gateway behavior, memory behavior, or controllane behavior.

## Rollback And Kill Switch

Future Phase 12K must include:

- `FORGE_K_SHADOW_MODE_ENABLED=false` global default
- a separate retrieval metadata flag defaulting to false
- no dependency from live request success to shadow diagnostics
- in-memory diagnostics only unless persistence is separately approved
- tests proving disabled mode matches baseline behavior

Rollback must be disabling flags or reverting the implementation. No live data migration should be required.

## Required Tests Before Implementation

Future Phase 12K cannot start without tests for:

- disabled-by-default behavior
- enabled metadata-only behavior
- no retrieval/search/embedding execution from FORGE-K
- no source text, chunk text, document content, embedding, vector, raw query, memory content, prompt, model output, request body, or response body capture
- route inventory stability
- API response status/body/header equivalence
- retrieval behavior and result count equivalence
- context compile behavior unchanged
- modelruntime, gateway/tool, memory, and controllane call-count stability
- sink failure isolation
- bounded sink retention
- no public diagnostics route

The detailed future test plan is `docs/testing/phase_12k_retrieval_metadata_shadow_tests.md`.

## What Not To Do

- Do not implement Phase 12K.
- Do not add retrieval metadata observer code yet.
- Do not observe retrieval routes.
- Do not execute retrieval/search/embedding calls.
- Do not implement live RAG.
- Do not capture source text.
- Do not capture chunks.
- Do not capture embeddings or vectors.
- Do not capture raw queries.
- Do not capture memory content.
- Do not capture prompts or model outputs.
- Do not capture request or response bodies.
- Do not add public diagnostics APIs.
- Do not add routes.
- Do not change route behavior.
- Do not modify API response shape.
- Do not call modelruntime.
- Do not execute tools.
- Do not write memory.
- Do not call controllane mutations.
- Do not make FORGE-K live authority.
