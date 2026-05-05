# Phase 12D Touchpoint Selection

Status: implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

## Summary

Phase 12D compares candidate shadow-mode expansion touchpoints and selects exactly one recommended candidate for a future Phase 12E implementation. It does not implement the selected touchpoint.

Selected future Phase 12E candidate: route envelope metadata.

The current implemented live observation remains `/health` request metadata only. Phase 12D adds no code, routes, public APIs, diagnostics endpoints, live observation, authority migration, modelruntime calls, retrieval/search/embedding calls, gateway execution, memory writes, or controllane mutation.

## Decision Matrix

| Candidate | Diagnostic Value | Risk Level | Data Sensitivity | Implementation Complexity | Testability | Response-Shape Drift Chance | Accidental Body Capture Chance | Modelruntime Risk | Retrieval/RAG Risk | Memory-Write Risk | Recommendation |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Route envelope metadata | High | Low | Low | Low-Medium | High | Low | Low | Low | Low | Low | Select for Phase 12E |
| Chat message submission metadata | High | High | High | Medium-High | Medium | Medium-High | High | Medium-High | Medium | Medium | Defer |
| Existing retrieval-result metadata | Medium-High | Medium-High | High | High | Medium | Medium | Medium-High | Low-Medium | High | Medium | Defer |
| Existing gateway trace metadata | Medium-High | High | Medium-High | High | Medium | Medium | Medium | Medium | Low-Medium | Medium | Defer |

## Selected Candidate

Route envelope metadata is the recommended first expansion because it can provide useful diagnostics without approaching prompt content, retrieval content, tool payloads, model outputs, or memory content.

Allowed future route-envelope fields:

- HTTP method
- matched route template or route class
- route owner component
- workspace id when already available
- correlation id or request id when already available and non-secret
- duration or bounded timing summary
- status class captured after response completion
- no-effect validation result

Forbidden future route-envelope fields:

- request body
- response body
- raw query string
- raw headers except explicitly allowlisted non-secret correlation ids
- authorization, cookie, session, credential, token, or secret values
- prompt text
- assistant or model output
- tool payloads
- retrieval result content
- embedding vectors or raw search chunks
- memory content

## Deferred Candidates

Chat message submission metadata is deferred because it is adjacent to prompt text, assistant streaming, response composition, and modelruntime selection. Even a metadata-only implementation needs stronger guards against body capture and response drift.

Existing retrieval-result metadata is deferred because it is adjacent to RAG, evidence selection, embedding records, search chunks, and memory provenance. It should follow only after route-envelope diagnostics prove stable and no-effect behavior across route classes.

Existing gateway trace metadata is deferred because it is adjacent to tool execution, approval records, artifacts, and gateway authority. It should wait until simpler observation has proven failure isolation, redaction, and bounded reporting.

## Required Future Phase 12E Conditions

Phase 12E may start only if separately approved as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Before implementation, Phase 12E must preserve:

- default disabled behavior
- no route inventory changes
- no public diagnostics API
- no response status, header, or body changes
- no request or response body capture
- no modelruntime calls
- no retrieval/search/embedding calls
- no gateway/tool execution
- no memory writes
- no controllane mutations
- no authority migration
- no persistent diagnostics unless separately approved

## No Live Behavior Change

Phase 12D is a selection and design pass only. It does not change live daemon behavior and does not expand the existing `/health`-only observation surface.
