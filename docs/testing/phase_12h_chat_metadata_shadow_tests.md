# Phase 12H Chat Metadata Shadow Test Plan

Status: future test plan only. Phase 12H is not started.

## Scope

Future Phase 12H scope, if approved: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

This document defines the tests required before any chat metadata shadow implementation can be accepted. It does not authorize implementation.

## Required Test Groups

### Disabled Default

- `FORGE_K_SHADOW_MODE_ENABLED` defaults to disabled.
- Disabled mode observes no chat metadata.
- Disabled mode writes no chat metadata diagnostic report.
- Disabled mode does not require a chat metadata observer or sink for request success.

### Enabled Metadata-Only Capture

- Enabled chat metadata mode captures metadata only.
- Allowed captured fields are limited to route class, matched chat route pattern, thread id if safe/stable, message id if safe/stable, workspace id if safe/stable, request id/correlation id, role class, bounded count summary, status/timing summary, safe model/provider id if already exposed, diagnostic markers, and warnings.
- Metadata count, string length, and report count are bounded.
- Reports stay in the bounded in-memory sink only unless persistence is separately approved.

### No Content Capture

- No request body is captured.
- No response body is captured.
- No message body is captured.
- No prompt text is captured.
- No completion or model output is captured.
- No assistant response text is captured.
- No system prompt is captured.
- No assistant stream token content is captured.
- No tool payload is captured.
- No tool output is captured.
- No retrieval content is captured.
- No search chunk content is captured.
- No embedding vector is captured.
- No memory content is captured.
- No auth, cookie, bearer token, API key, or secret-looking metadata is captured.

### No Response Or Route Drift

- Route inventory is unchanged.
- No public diagnostics route is added.
- Chat response status codes are unchanged.
- Chat response headers are unchanged.
- Chat response bodies are unchanged.
- Chat route matching and handler selection are unchanged.
- Assistant stream/SSE behavior is unchanged.
- Timeout behavior is unchanged.

### Call Count Stability

- Modelruntime call count is unchanged.
- Retrieval/search/embedding call count is unchanged.
- Gateway/tool execution count is unchanged.
- Memory write count is unchanged.
- Controllane mutation count is unchanged.
- Approval, permission, lane, and audit authority decisions are unchanged.

### Failure Isolation

- Shadow redaction failure drops the diagnostic report and does not fail the chat request.
- Shadow sink failure drops the diagnostic report and does not fail the chat request.
- No-effect validation failure drops the diagnostic report and does not fail the chat request.
- Observer panic recovery preserves the live response.
- Bounded sink overflow drops oldest or rejects new diagnostics according to policy without affecting requests.

### Workspace And Security

- Workspace ids are preserved only when already available and safe.
- Thread ids and message ids are preserved only when already available and safe.
- Secret-looking metadata is rejected.
- Cross-workspace report leakage is rejected.
- Reports remain diagnostic-only and non-authoritative.
- Reports are not treated as memory, evidence, ContextBlocks, KV metadata, runtime input, or Kernel truth.

## Required Commands

Future Phase 12H validation must run at least:

- `cd services/core && go test ./internal/forgek/...`
- `cd services/core && go test ./internal/forgekshadow/...`
- `cd services/core && go test ./internal/api -run "TestServerRouteInventory|TestServerRouteInventoryHealthAndMiddlewareSmoke" -count=1`
- `cd services/core && go test ./internal/api -count=1`
- `npm run build:core`
- `npm run lint`
- `npm test`
- `npm run test:forgek:parity`
- `git diff --check`

## Exit Criteria

Phase 12H is complete only if chat metadata diagnostics are disabled by default, read-only, bounded, metadata-only, non-authoritative, and proven to have no effect on live responses or live authority paths.

## What Not To Do

- Do not capture message content.
- Do not capture prompts.
- Do not capture completions.
- Do not capture request bodies.
- Do not capture response bodies.
- Do not observe tool payloads.
- Do not observe retrieval content.
- Do not add public diagnostics APIs.
- Do not change route behavior.
- Do not modify API response shape.
- Do not call modelruntime from FORGE-K.
- Do not execute tools from FORGE-K.
- Do not query retrieval/search/embeddings from FORGE-K.
- Do not write memory from FORGE-K.
- Do not call controllane mutations from FORGE-K.
- Do not make FORGE-K live authority.
