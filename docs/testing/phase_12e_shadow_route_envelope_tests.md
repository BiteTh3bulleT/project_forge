# Phase 12E Shadow Route Envelope Test Plan

Status: implemented test plan and validation record. Phase 12E is implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`. Phase 12F extends this record as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY` and hardens the route-envelope observer without adding touchpoints.

## Scope

This document defines the tests for route envelope metadata shadowing in Phase 12E.

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Phase 12E adds disabled-by-default route-envelope observation only. It does not add routes, public APIs, gateway calls, modelruntime calls, retrieval/search/embedding calls, memory writes, controllane mutations, persistent diagnostics, response changes, or authority migration.

Phase 12F hardens the same route-envelope observation surface only. It does not add touchpoints, public diagnostics routes, persistence, body/query/header/secret capture, route behavior changes, gateway calls, modelruntime calls, retrieval/search/embedding calls, memory writes, controllane mutations, response changes, or authority migration.

## Required Test Groups

### Disabled Default

- `FORGE_K_SHADOW_MODE_ENABLED` defaults to disabled.
- Disabled mode stores no route-envelope diagnostics.
- Disabled mode does not generate route-envelope diagnostics.
- Disabled mode does not allocate or require a route-envelope sink for request success.

### Enabled Route Envelope Capture

- Enabled route-envelope mode captures method, matched route template or route class, route owner classification, timing summary, workspace/correlation ids when already available, and no-effect validation only.
- Enabled route-envelope mode does not capture raw paths with query strings when a route template is available.
- Enabled route-envelope mode bounds metadata count, string length, and report count.
- Enabled route-envelope mode stores diagnostics in an in-memory sink only unless persistence is separately approved.

### No Body Capture

- No request body is captured.
- No response body is captured.
- No prompt text is captured.
- No assistant/model output is captured.
- No tool payload is captured.
- No retrieval result content, search chunk content, embedding vector, or memory content is captured.
- Unsafe metadata keys such as body, prompt, completion, model output, content, authorization, cookie, session, token, password, credential, bearer, secret, api key, and private key are rejected.

### No Response Or Route Drift

- Route inventory is unchanged.
- No public diagnostics routes are added.
- Public response headers are unchanged.
- Public response status codes are unchanged.
- Public response bodies are unchanged.
- Route matching and handler selection are unchanged.
- Non-`/api` behavior remains unchanged unless explicitly included by the approved Phase 12E design.
- `/api/chat/threads/{id}/assistant-stream` SSE behavior is unchanged.
- Timeout behavior is unchanged.

### Failure Isolation

- Shadow redaction failure drops the diagnostic report and does not fail the request.
- Shadow sink failure drops the diagnostic report and does not fail the request.
- No-effect validation failure drops the diagnostic report and does not fail the request.
- Panic recovery in the observer preserves the live response.
- Bounded sink overflow drops oldest or rejects new diagnostics according to policy without affecting requests.

### Forbidden Execution

- No modelruntime calls originate from route-envelope shadowing.
- No retrieval/search/embedding calls originate from route-envelope shadowing.
- No gateway/tool execution originates from route-envelope shadowing.
- No memory writes originate from route-envelope shadowing.
- No controllane mutations originate from route-envelope shadowing.
- No approval, permission, lane, or audit authority decision is changed by route-envelope shadowing.

### Workspace And Security

- Workspace ids are preserved only when already available.
- Secret-looking metadata is rejected.
- Cross-workspace report leakage is rejected.
- Reports remain diagnostic-only and non-authoritative.
- Reports are not treated as memory, evidence, ContextBlocks, KV metadata, runtime input, or Kernel truth.

## Required Commands

Phase 12E validation must run at least:

- `cd services/core && go test ./internal/forgek/...`
- `cd services/core && go test ./internal/forgekshadow/...`
- `cd services/core && go test ./internal/api -run "TestServerRouteInventory|TestServerRouteInventoryHealthAndMiddlewareSmoke" -count=1`
- `npm run build:core`
- `npm run lint`
- `npm test`
- `npm run test:forgek:parity`
- `git diff --check`

## Exit Criteria

Phase 12E is complete only if route-envelope diagnostics remain disabled by default, read-only, bounded, metadata-only, non-authoritative, and proven to have no effect on live responses or live authority paths.

## Implemented Coverage

- `services/core/internal/forgekshadow/observer_test.go` covers disabled route-envelope no-op behavior, enabled diagnostic report creation, typed `RouteEnvelopeObservation` fields, route class normalization, unsafe metadata rejection, best-effort unsafe report dropping, bounded sink reuse, sink failure isolation, and no-effect policy enforcement.
- `services/core/internal/api/server_route_inventory_test.go` covers route inventory stability, no public diagnostics route, `/health` response equivalence, `/api/meta` disabled/enabled response equivalence, route-envelope report creation for `/api/meta`, invalid POST body non-capture on `/api/commands/execute`, `/forge` route inventory, conditional `/v1` route inventory, and SSE mount/order guard.
- Status code capture is intentionally omitted in Phase 12E to avoid response writer wrapping. Response status, headers, and bodies are tested for equivalence instead.

## Phase 12F Hardening Coverage

- `services/core/internal/forgekshadow/observer_test.go` covers matched route-pattern preference over raw paths, unsafe raw dynamic route-pattern rejection when no template is available, provided route class normalization to known classes, rejection of raw query/request URI metadata, rejection of metadata that tries to reintroduce path or route pattern values, expanded secret/header/content key rejection, deterministic scalar metadata enforcement, bounded retention drop-oldest behavior, and best-effort sink-failure isolation.
- `services/core/internal/forgekshadow/forbidden_imports_test.go` guards the route-envelope observer package against imports from gateway, modelruntime, retrieval, search, embeddings, memory, and AI-OS controllane packages.
- `services/core/internal/forgek/shadowharness/policy_test.go` covers search execution and controllane mutation side-effect policy rejection.
- `services/core/internal/api/server_route_inventory_test.go` covers matched-pattern preference over raw paths/query strings for `/api/meta`, unmatched-route response equivalence with no report, `/forge/model-runtime/health` response equivalence, conditional `/v1/models` response equivalence, sink-failure response equivalence, route inventory stability, no public diagnostics route, SSE mount/order preservation, and timeout middleware preservation.
