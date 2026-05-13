# Phase 12I Chat Metadata Shadow Hardening

Status: implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`.

## Implementation Summary

Phase 12I hardens the Phase 12H chat metadata observer without adding new live touchpoints or expanding the observed surface. Chat metadata remains disabled by default and requires both `FORGE_K_SHADOW_MODE_ENABLED=true` and `FORGE_K_SHADOW_CHAT_METADATA_ENABLED=true`.

The existing chat metadata touchpoint remains the chat message POST handler after the live handler owns the request and has already committed the user message through the live chat path. Reports remain bounded in-memory diagnostics only.

## Current Chat Metadata Scope

Allowed metadata remains limited to:

- observation type and route class
- operation kind from a bounded enum
- safe thread and message refs
- workspace and request refs when already available
- role class from a bounded enum
- stream class from a bounded enum
- safe model/provider refs when non-secret
- bounded count and timing metadata
- diagnostic markers and bounded warnings

## No-Effect Guarantees

Phase 12I preserves these guarantees:

- no live state mutation
- no tool execution
- no modelruntime call
- no retrieval/search/embedding call
- no memory write
- no controllane mutation
- no public API or route change
- no status, header, response body, route inventory, or SSE behavior change
- no user-visible output authority
- no FORGE-K live authority path

## Risk Review

Chat metadata remains higher risk than route-envelope metadata because it is adjacent to prompts, completions, tool payloads, retrieval content, and memory context. Phase 12I therefore keeps all content-bearing surfaces forbidden and verifies that invalid bodies, auth/cookie headers, query strings, assistant-stream routes, and sink failures do not leak into diagnostics or alter responses.

## Coverage Review

Hardening coverage now includes:

- dual-flag matrix for shadow/chat enablement
- bounded operation, role, and stream enum normalization
- ref length and secret-looking ref rejection
- deterministic serialization for stable chat metadata shape
- forbidden metadata key rejection for prompt/completion/body/message/tool/retrieval/memory-like keys
- disabled sink behavior and bounded retention
- no-effect policy rejection for all side-effect flags
- chat POST response shape equivalence
- invalid chat body no-capture regression
- auth/cookie/query no-capture regression
- assistant-stream no chat-metadata regression
- sink failure response isolation
- forbidden production import coverage

## Hardening Actions Completed

- Expanded raw-content metadata key rejection to include `message` and `message_body`.
- Added chat metadata dual-flag tests for all enabled/disabled combinations.
- Added enum normalization and reserved metadata override tests.
- Added deterministic serialization coverage for stable metadata shape.
- Added API tests for invalid body, auth/cookie/query leakage, assistant-stream safety, and sink failure isolation.
- Updated status, roadmap, architecture, runbook, and test documentation.

## Remaining Deferred Work

Deferred until separately approved phases:

- retrieval/search/embedding trace metadata
- gateway trace metadata
- modelruntime trace metadata
- persistent diagnostic storage
- public diagnostics APIs
- live authority migration

## Recommendation

Proceed next to a Phase 12J design-only pass if broader metadata is needed. Do not expand live observation beyond chat metadata without a separate design, threat review, no-effect tests, and rollback plan.
