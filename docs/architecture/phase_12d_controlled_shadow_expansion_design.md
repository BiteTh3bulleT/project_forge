# Phase 12D Controlled Shadow Expansion Design

Status: implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`; Phase 12E later implemented the selected route-envelope touchpoint as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

## Executive Summary

Phase 12D designed the next controlled shadow-mode expansion. It did not implement the expansion.

Phase 12B added the first disabled-by-default read-only observer for `/health` metadata only. Phase 12C hardened that observer. Phase 12D selected route envelope metadata for Phase 12E. Phase 12E now implements that selected touchpoint behind `FORGE_K_SHADOW_MODE_ENABLED`.

Route envelope metadata is selected because it provides useful route-level diagnostics with the lowest semantic and authority risk. It can be designed around method, matched route, route class, timing, workspace/correlation ids, and diagnostic status only. It must not capture request bodies, response bodies, prompts, tool payloads, retrieval content, memory content, or model output.

## Current Phase 12B - 12E State

Current implemented shadow scope:

- feature flag: `FORGE_K_SHADOW_MODE_ENABLED=false`
- implemented touchpoints: `/health` request metadata and route-envelope metadata
- sink: bounded in-memory diagnostic reports only
- public diagnostics API: none
- persistence: none
- authority: none

Phase 12C hardened:

- disabled sink behavior
- metadata secret rejection
- raw content key rejection
- metadata size limits
- route inventory and response equivalence tests
- forbidden live import tests

After Phase 12E, route-envelope metadata is implemented and `/health` remains supported.

## Candidate Touchpoints

| Candidate | Description | Initial Assessment |
|---|---|---|
| Route envelope metadata | Observe method, matched route, route class, timing, workspace/correlation ids. | Recommended for Phase 12E. Lowest semantic risk and easiest no-effect proof. |
| Chat message submission metadata | Observe chat submission refs and timing only. | Deferred. High sensitivity and higher accidental body/prompt capture risk. |
| Existing retrieval-result metadata | Observe already-produced retrieval run/result refs and scores. | Deferred. Useful later, but near RAG/evidence boundaries. |
| Existing gateway trace metadata | Observe already-produced gateway invocation/approval refs. | Deferred. Valuable but close to tool execution authority. |

## Comparison Matrix

| Candidate | Diagnostic Value | Risk | Sensitivity | Complexity | Testability | Body Capture Risk | Authority Risk | Recommendation |
|---|---|---|---|---|---|---|---|---|
| Route envelope metadata | High | Low | Low | Low-Medium | High | Low | Low | Select for Phase 12E |
| Chat message submission metadata | High | High | High | Medium-High | Medium | High | Medium-High | Defer |
| Existing retrieval-result metadata | Medium-High | Medium-High | High | High | Medium | Medium-High | Medium | Defer |
| Existing gateway trace metadata | Medium-High | High | Medium-High | High | Medium | Medium | High | Defer |

## Selected Touchpoint For Phase 12E

Phase 12E implements route envelope metadata shadowing.

Allowed Phase 12E data:

- HTTP method
- matched route pattern or route class
- path template, not raw query content
- request id when safely available
- duration
- no-effect validation result

Forbidden Phase 12E data:

- request body
- response body
- raw headers except explicit non-secret correlation ids
- cookies
- authorization headers
- prompts
- chat message text
- model output
- retrieval result content
- tool payloads
- memory content

## Excluded Touchpoints

Chat message submission metadata is excluded from Phase 12E because it is adjacent to user prompt content, assistant streaming behavior, modelruntime selection, and response composition. Even metadata-only capture has a higher chance of accidental body capture or response-shape drift.

Existing retrieval-result metadata is excluded because it is adjacent to RAG, embeddings, evidence selection, and memory provenance. It should wait until route-envelope observation proves stable and ref-only RAG diagnostics have stronger tests.

Existing gateway trace metadata is excluded because it is adjacent to tool execution, approvals, artifacts, and gateway authority. It should wait until the route-envelope path proves failure isolation and no-effect behavior.

## No-Effect Guarantees

Phase 12E must prove:

- feature flag defaults disabled
- route inventory unchanged
- public API shape unchanged
- request status codes unchanged
- response headers unchanged
- response bodies unchanged
- no request body capture
- no response body capture
- no public diagnostics route
- shadow failure does not fail live request
- no modelruntime calls
- no retrieval/search/embedding calls
- no gateway/tool execution
- no memory writes
- no controllane mutations
- `/api/chat/threads/{id}/assistant-stream` SSE behavior unchanged
- timeout behavior unchanged

## Data Capture Policy

Route-envelope diagnostics must be metadata-only. They should prefer route templates/classes over raw paths when available. Query strings should be excluded by default. Correlation ids may be recorded only if they are not secret-bearing.

Reports must remain diagnostic-only and non-authoritative. They must not become canonical memory, Courthouse evidence, ContextBlocks, audit authority, or route behavior inputs.

## Redaction Policy

Phase 12E must reuse or strengthen the Phase 12C metadata safety rules:

- reject secret-looking keys or values
- reject authorization, cookie, session, token, password, credential, bearer, secret, api key, private key, and plaintext indicators
- reject raw content fields such as body, prompt, completion, model output, and content
- enforce bounded string length
- fail closed by dropping reports rather than storing unsafe metadata

## Performance Budget

Future route-envelope shadowing must be cheap:

- disabled path near zero overhead
- no blocking model, retrieval, gateway, memory, or controllane calls
- bounded report construction time
- bounded report size
- bounded in-memory retention
- no additional route matching beyond what live routing already performs unless proven safe

## Failure Isolation

Shadow failures must be isolated:

- redaction failure drops the report
- sink failure drops the report
- no-effect validation failure drops the report
- observer panic is recovered
- live request continues through the existing live path

No failure may change response status, headers, body, route selection, timeout handling, stream behavior, modelruntime behavior, gateway behavior, retrieval behavior, memory behavior, or controllane behavior.

## Rollback / Kill Switch

Phase 12E must keep:

- `FORGE_K_SHADOW_MODE_ENABLED=false` default
- a single environment/config kill switch
- no dependency from live request success to shadow reports
- in-memory diagnostics until persistence is separately approved
- tests proving disabled mode behaves like the baseline

Rollback must be disabling the flag or reverting the Phase 12E implementation. No live data migration should be required.

## Phase 12E Tests

The Phase 12E route-envelope test plan and implemented coverage are recorded in `docs/testing/phase_12e_shadow_route_envelope_tests.md`.

Minimum test groups:

- disabled default behavior
- enabled no-effect behavior
- route inventory stability
- response equivalence
- no body/header/status mutation
- no public diagnostics routes
- sink failure isolation
- forbidden execution/import checks
- SSE and timeout stability

## What Not To Do

- Do not broaden beyond route-envelope metadata.
- Do not add route-envelope body, query, header, prompt, model output, retrieval content, or memory content capture.
- Do not capture request bodies.
- Do not capture response bodies.
- Do not add public diagnostics APIs.
- Do not change route behavior.
- Do not modify API response shape.
- Do not call modelruntime.
- Do not execute tools.
- Do not query retrieval/search/embeddings.
- Do not write memory.
- Do not call controllane mutations.
- Do not make FORGE-K live authority.
