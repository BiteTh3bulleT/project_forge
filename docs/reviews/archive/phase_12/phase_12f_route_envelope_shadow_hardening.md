# Phase 12F Route Envelope Shadow Hardening

Status: implemented and tested as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`.

## Implementation Summary

Phase 12F hardens the existing Phase 12E route-envelope shadow diagnostics. It does not add live touchpoints, public diagnostics routes, persistence, public APIs, route behavior changes, response changes, or authority migration.

The implemented route-envelope observer remains disabled by default behind `FORGE_K_SHADOW_MODE_ENABLED=false`. When enabled, it records bounded in-memory diagnostic reports only after live handlers complete. Reports remain non-authoritative and cannot influence routing, response composition, gateway behavior, modelruntime behavior, retrieval/search/embedding behavior, memory writes, controllane state, approval decisions, lanes, audit authority, or Kernel truth.

## Current Route-Envelope Scope

Allowed metadata remains narrow:

- HTTP method
- matched route pattern when available
- normalized route class
- duration
- safe request id when already available
- diagnostic markers and no-effect validation result

Forbidden metadata includes:

- request bodies
- response bodies
- raw query strings
- raw request URI values
- raw headers except separately allowlisted non-secret correlation ids
- cookies
- authorization values
- secrets, credentials, API keys, bearer tokens, session values, JWTs, access tokens, refresh tokens, and set-cookie values
- prompts
- model output
- tool payloads
- retrieval content
- search chunks
- embedding vectors
- memory content
- large raw content blobs

## No-Effect Guarantees

Phase 12F preserves the Phase 12B through Phase 12E no-effect guarantees:

- disabled mode stores no route-envelope reports
- enabled mode does not change live response status, headers, or body
- shadow report failure does not fail live requests
- route inventory is unchanged
- no public diagnostics route exists
- SSE mount/order behavior is unchanged
- timeout middleware behavior is unchanged
- unmatched routes do not generate reports
- `/health` keeps the Phase 12B per-handler observer and is skipped by route-envelope middleware
- diagnostics remain bounded in memory only

## Risk Review

| Risk | Phase 12F Control |
| --- | --- |
| Raw dynamic path leakage | Prefer matched route patterns; reject unsafe raw dynamic route patterns when a template is unavailable. |
| Query leakage | Strip query data from route patterns and reject query/request URI metadata keys. |
| Header or secret leakage | Expand unsafe metadata terms and reject secret-looking keys or values. |
| Caller metadata overriding route identity | Reject reserved metadata keys that could reintroduce `path`, `route_pattern`, or route class values. |
| Non-deterministic metadata capture | Accept deterministic scalar values only. |
| Report retention growth | Keep bounded in-memory retention and test oldest-report drop behavior. |
| Sink failure affecting live routes | Treat sink writes as best-effort and preserve live responses on failure. |
| Route behavior drift | Route inventory and response equivalence tests cover `/api`, `/forge`, and conditional `/v1` surfaces. |
| Stream or timeout drift | SSE mount/order and timeout middleware tests remain stable. |
| Live authority coupling | Forbidden import tests reject gateway, modelruntime, retrieval, search, embeddings, memory, and controllane imports in `forgekshadow`. |

## Coverage Review

Phase 12F adds or strengthens tests for:

- matched route-pattern preference over raw paths and query strings
- unsafe raw dynamic route-pattern rejection
- route class normalization to known classes
- query and request URI metadata rejection
- reserved route envelope metadata key rejection
- non-deterministic metadata value rejection
- expanded secret/header/content metadata rejection
- bounded retention behavior
- sink failure isolation
- no public diagnostics route
- unmatched-route no-observation behavior
- `/api/meta` response equivalence
- `/forge/model-runtime/health` response equivalence
- conditional `/v1/models` response equivalence
- SSE mount/order preservation
- timeout middleware preservation
- search execution and controllane mutation policy rejection
- forbidden live authority imports

## Hardening Actions Completed

- Route envelope creation now prefers framework-matched route patterns over raw paths.
- Unsafe raw dynamic route patterns are dropped when a route template is unavailable.
- Provided route classes are normalized back to known route classes.
- Reserved metadata keys cannot override or reintroduce route identity fields.
- Query and request URI metadata keys are rejected.
- Secret/header/content redaction covers more common key forms.
- Metadata values are limited to deterministic scalar values.
- Bounded report retention and sink failure isolation are covered by tests.
- API route tests cover route stability and response equivalence across representative `/api`, `/forge`, and conditional `/v1` paths.

## Deferred Work

Deferred work remains unchanged:

- no broader route observation beyond the existing route-envelope scope
- no chat content observation
- no retrieval/search/embedding content observation
- no gateway payload observation
- no public diagnostics API
- no persistent diagnostic report store
- no live RAG
- no FORGE-K-driven tool execution
- no modelruntime calls
- no live memory writes
- no controllane mutations
- no authority migration

## Recommendation

Route-envelope diagnostics are hardened enough to remain in place as disabled-by-default observability. The next phase should not broaden capture by default. Recommended next work is a docs-first design for any future metadata surface, with the same constraints: explicit scope, no content capture, no live mutation, no public diagnostics route, no persistence without separate approval, and route/API response equivalence tests before implementation.
