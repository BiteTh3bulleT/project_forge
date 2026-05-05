# Phase 12G Chat Metadata Expansion Design

Status: implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

## Executive Summary

Phase 12G designed the chat metadata shadow expansion. Phase 12H later implemented the bounded metadata-only observer behind disabled-by-default flags. Phase 12G itself did not implement chat metadata observation, add live touchpoints, observe chat routes, add routes, change public APIs, or change live daemon behavior.

The current implemented shadow scope remains:

- `/health` request metadata from Phase 12B
- route-envelope metadata from Phase 12E
- route-envelope hardening from Phase 12F

Chat metadata is higher risk than route-envelope metadata because it is adjacent to user message bodies, prompts, assistant responses, tool payloads, modelruntime calls, retrieval/RAG context, memory writes, and streaming behavior. Any future implementation must prove metadata-only capture and no-effect behavior before it can be enabled.

## Current Phase 12B Through 12F State

Implemented live-adjacent shadow behavior remains narrow:

- feature flag: `FORGE_K_SHADOW_MODE_ENABLED=false`
- implemented touchpoints: `/health` metadata and route-envelope metadata
- storage: bounded in-memory diagnostic reports only
- public diagnostics API: none
- persistence: none
- live authority: none

Phase 12G added no code and no observation path. Phase 12H is the implementation phase for the selected metadata-only boundary.

## Why Chat Metadata Is Higher Risk

Route-envelope metadata can be limited to method, matched route pattern, route class, timing, and safe request ids. Chat metadata sits closer to sensitive semantic material:

- request bodies may contain user messages and prompts
- response bodies may contain assistant output
- assistant streams may expose tokens or completion fragments
- chat routes may trigger modelruntime calls
- chat routes may trigger gateway/tool execution
- chat routes may include retrieval, memory, attachment, or artifact refs
- chat routes may write durable messages or metadata

Because of that adjacency, the chat metadata observer must fail closed and must never inspect, copy, summarize, hash, or persist content.

## Phase 12H Implemented Scope

Phase 12H implements this scope after separate approval. Phase 12I later hardens the same scope without adding touchpoints.

Candidate scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Candidate touchpoint:

- chat message submission metadata only

Phase 12H must stay metadata-only. It must not observe chat content, prompt text, completion text, response bodies, tool payloads, retrieval content, memory content, or raw request bodies.

## Allowed Metadata

Phase 12H may capture only bounded, stable metadata that is already available without reading bodies or outputs:

- route class: `chat`
- matched chat route pattern
- thread id if already safe and stable
- message id if already safe and stable
- workspace id if already safe and stable
- request id or correlation id
- message role class only if safe, such as `user`, `assistant`, `system`, or `tool`
- message count or bounded count summary
- status or timing metadata
- model/provider id only if already safely exposed and not secret
- diagnostic markers
- redacted warning summaries

All identifiers must pass the same secret-looking and dynamic-path safety expectations used by the route-envelope observer.

## Forbidden Metadata

Phase 12H and Phase 12I must not capture:

- message body
- prompt text
- completion text
- assistant response text
- system prompt
- request body
- response body
- assistant stream token content
- tool payloads
- tool outputs
- file contents
- source chunks
- retrieval result content
- search result content
- embedding vectors
- memory content
- auth headers
- cookies
- bearer tokens
- API keys
- raw authorization metadata
- unredacted model parameters if sensitive
- large raw content blobs
- secrets or secret-looking metadata

Metadata that cannot be proven safe must be dropped.

## No-Effect Guarantees

Phase 12H and Phase 12I must prove:

- feature flag defaults disabled
- disabled mode observes no chat metadata
- enabled mode does not change response status
- enabled mode does not change response headers
- enabled mode does not change response bodies
- assistant stream/SSE behavior is unchanged
- route inventory is unchanged
- no public diagnostics route is added
- shadow failure cannot fail the chat request
- modelruntime call count is unchanged
- retrieval/search/embedding call count is unchanged
- gateway/tool execution count is unchanged
- memory write count is unchanged
- controllane mutation count is unchanged

Shadow diagnostics remain non-authoritative and cannot affect response composition, model selection, retrieval selection, gateway execution, approvals, lanes, audit authority, memory writes, or Kernel truth.

## Data Capture Policy

Chat metadata capture must be metadata-only and ref-only.

Allowed records should prefer:

- route templates over raw paths
- bounded counts over content
- stable ids over bodies
- role classes over message text
- timing summaries over execution traces with content
- diagnostic warnings over raw error payloads

No chat metadata observer may parse request bodies to find metadata unless a separate phase explicitly approves a parser, redaction model, and tests proving no content retention. Phase 12G did not approve that work, and Phase 12H/12I do not add it.

## Redaction Policy

Phase 12H/12I must reuse or strengthen Phase 12F redaction:

- reject secret-looking keys or values
- reject raw content keys such as body, prompt, completion, model output, content, message text, system prompt, and stream token
- reject auth, cookie, session, token, password, credential, bearer, secret, API key, private key, JWT, access token, refresh token, and set-cookie indicators
- reject raw query strings and request URI values
- enforce bounded string length
- reject non-deterministic metadata values
- drop the entire report on unsafe metadata

## Performance Budget

Phase 12H/12I must remain cheap:

- disabled path near zero overhead
- no blocking modelruntime, retrieval, gateway, memory, or controllane calls
- no request or response body reads
- bounded report construction time
- bounded report size
- bounded in-memory retention
- no stream buffering

## Failure Isolation

Chat metadata shadowing must fail closed:

- redaction failure drops the report
- sink failure drops the report
- no-effect validation failure drops the report
- observer panic is recovered
- live request continues through the existing live path

No shadow failure may change status, headers, body, route selection, timeout behavior, SSE behavior, modelruntime behavior, gateway behavior, retrieval behavior, memory behavior, or controllane behavior.

## Rollback And Kill Switch

Phase 12H/12I must keep:

- `FORGE_K_SHADOW_MODE_ENABLED=false` default
- one config/environment kill switch
- no dependency from live request success to shadow diagnostics
- in-memory diagnostics only unless persistence is separately approved
- tests proving disabled mode matches baseline behavior

Rollback must be disabling the flag or reverting the implementation. No live data migration should be required.

## Required Tests

Phase 12H/12I require tests for:

- disabled-by-default behavior
- enabled metadata-only behavior
- no body, prompt, completion, output, tool payload, retrieval content, or memory content capture
- route inventory stability
- chat response status/body/header equivalence
- assistant stream/SSE equivalence
- modelruntime, retrieval/search/embedding, gateway/tool, memory, and controllane call-count stability
- sink failure isolation
- bounded sink retention
- no public diagnostics route

The detailed test plan is `docs/testing/phase_12h_chat_metadata_shadow_tests.md`; the Phase 12I hardening review is `docs/reviews/phase_12i_chat_metadata_shadow_hardening.md`.

## Phase 12J Handoff

Phase 12H implemented bounded chat metadata. Phase 12I hardened that chat metadata path. Phase 12J designs retrieval metadata only.

Phase 12K may implement retrieval metadata only if separately approved. No retrieval content, source text, chunk text, raw query, RAG output, embedding, vector, memory content, prompt, model output, request body, or response body capture is approved by Phase 12J.

## What Not To Do

- Do not expand beyond the approved Phase 12H chat metadata observer.
- Do not add additional chat metadata observer code outside the approved touchpoint.
- Do not observe additional chat routes.
- Do not capture message content.
- Do not capture prompts.
- Do not capture completions.
- Do not capture response bodies.
- Do not capture request bodies.
- Do not observe tool payloads.
- Do not observe retrieval content.
- Do not add public diagnostics APIs.
- Do not change route behavior.
- Do not modify API response shape.
- Do not call modelruntime.
- Do not execute tools.
- Do not query retrieval/search/embeddings.
- Do not write memory.
- Do not call controllane mutations.
- Do not make FORGE-K live authority.
