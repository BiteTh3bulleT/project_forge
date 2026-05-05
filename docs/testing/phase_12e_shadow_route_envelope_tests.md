# Phase 12E Shadow Route Envelope Test Plan

Status: future test plan only. Phase 12E is not started.

## Scope

This document defines the minimum tests required before implementing route envelope metadata shadowing in a future Phase 12E.

Expected future scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Phase 12E must not start until separately approved. This test plan does not add code, routes, public APIs, live observation, gateway calls, modelruntime calls, retrieval/search/embedding calls, memory writes, controllane mutations, or authority migration.

## Required Test Groups

### Disabled Default

- `FORGE_K_SHADOW_MODE_ENABLED` defaults to disabled.
- Disabled mode preserves the current `/health`-only observation behavior if Phase 12B/12C remains the baseline.
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

A future Phase 12E implementation must run at least:

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
