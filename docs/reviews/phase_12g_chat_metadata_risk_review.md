# Phase 12G Chat Metadata Risk Review

Status: implemented as part of `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

## Summary

Phase 12G compares possible shadow metadata surfaces after Phase 12F route-envelope hardening. The conclusion is that chat metadata may be a valid next expansion only if it captures identifiers, classes, bounded counts, timing, and diagnostic markers. It must never capture message content, prompts, completions, model outputs, tool payloads, retrieval content, memory content, request bodies, or response bodies.

No implementation is authorized by this review.

## Candidate Comparison

| Candidate | Semantic Sensitivity | Prompt Leakage Risk | Response Leakage Risk | Body Capture Risk | Modelruntime Adjacency | Retrieval/RAG Adjacency | Memory-Write Adjacency | Tool Execution Adjacency | Testability | Recommendation |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Route envelope metadata | Low | Low | Low | Low | Low | Low | Low | Low | High | Already implemented and hardened. Keep as current widest live observation surface. |
| Chat metadata | High | High | High | High | High | Medium-High | Medium-High | Medium-High | Medium | Possible next expansion only as identifiers/classes/counts/timing. No content capture. |
| Retrieval metadata | High | Medium-High | Medium | Medium-High | Medium | High | High | Low-Medium | Medium | Defer until chat metadata proves no-content capture and RAG mirror rules are stronger. |
| Gateway trace metadata | Medium-High | Medium | Medium | Medium | Medium | Medium | Medium | High | Medium | Defer because it is close to tool execution, approvals, artifacts, and gateway authority. |

## Route Envelope Metadata

Route envelope metadata is the lowest-risk implemented surface because it can be captured after routing without inspecting request or response bodies. Phase 12F hardened it by preferring matched route patterns, rejecting raw dynamic paths, rejecting query/request URI metadata, expanding redaction, preserving bounded retention, and proving response equivalence.

Residual risk remains limited to metadata drift and accidental unsafe metadata insertion. Current tests and redaction rules control that risk.

## Chat Metadata

Chat metadata has higher diagnostic value, but it is adjacent to sensitive and authoritative surfaces:

- user message bodies
- system prompts
- assistant responses
- stream tokens
- modelruntime calls
- retrieval or memory context
- tool payloads and tool outputs
- durable chat message writes

Recommended constraints for future Phase 12H:

- capture route class and matched route pattern only
- capture thread id and message id only when already safe/stable
- capture workspace id, request id, and correlation id only when already safe/stable
- capture role class only, never role content
- capture bounded counts and timing summaries only
- reject all content-like fields
- prove no request body read and no response body read
- prove no SSE buffering or stream token capture
- prove modelruntime/retrieval/gateway/memory/controllane call counts do not change

Expected conclusion: chat metadata may be a valid next expansion only if it captures identifiers/classes/counts/timing, never content.

## Retrieval Metadata

Retrieval metadata is useful for RAG diagnostics but sits near evidence selection, source chunks, embedding records, memory provenance, and search result content. A metadata-only retrieval observer must wait until the project has stronger ref-only RAG diagnostic tests and clear rules for score/ref capture without source content capture.

Recommended status: defer.

## Gateway Trace Metadata

Gateway trace metadata is useful for tool execution diagnostics, but it is close to approvals, tool payloads, artifacts, capabilities, and execution authority. Even ref-only trace capture has authority risk if diagnostics are later mistaken for permission decisions.

Recommended status: defer until route and chat metadata paths have proven no-effect behavior.

## Required Constraints For Any Future Expansion

Any future metadata surface must keep these constraints:

- disabled by default
- read-only
- metadata-only
- bounded in memory
- no public diagnostics route
- no persistence without separate approval
- no request body capture
- no response body capture
- no prompt or completion capture
- no modelruntime call from FORGE-K
- no retrieval/search/embedding call from FORGE-K
- no gateway/tool execution from FORGE-K
- no memory write
- no controllane mutation
- no user-visible output change
- no live authority migration

## Recommendation

Use Phase 12H only for chat metadata if separately approved. Keep the first chat metadata implementation narrower than route-envelope observation in practice: identifiers, classes, counts, timing, and warnings only. Defer retrieval and gateway trace metadata until the chat metadata boundary has passing no-effect and no-content tests.
