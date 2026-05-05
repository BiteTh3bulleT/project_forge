# Phase 12K Retrieval Metadata Shadow Test Plan

Status: future test plan only. Phase 12K is not started.

## Scope

Future Phase 12K scope, if approved: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

This document defines the tests required before any retrieval metadata shadow implementation can be accepted. It does not authorize implementation.

## Required Test Groups

### Disabled Default

- `FORGE_K_SHADOW_MODE_ENABLED` defaults to disabled.
- A separate retrieval metadata flag defaults to disabled.
- Disabled mode observes no retrieval metadata.
- Disabled mode writes no retrieval metadata diagnostic report.
- Disabled mode does not require a retrieval metadata observer or sink for request success.

### Enabled Metadata-Only Capture

- Enabled retrieval metadata mode captures metadata only.
- Allowed captured fields are limited to retrieval run/result refs, workspace/request/correlation refs, source type/ref, existing source fingerprint, result counts, selected count, score summaries, ranking position, retrieval strategy, index name/type, safe embedding model id, freshness/staleness flags, timing/status metadata, diagnostic markers, and bounded warnings.
- Metadata count, string length, and report count are bounded.
- Reports stay in the bounded in-memory sink only unless persistence is separately approved.

### No Retrieval Execution From FORGE-K

- FORGE-K shadow does not execute retrieval queries.
- FORGE-K shadow does not execute search queries.
- FORGE-K shadow does not call embedding providers.
- FORGE-K shadow does not compute vectors or embeddings.
- FORGE-K shadow does not run VSA scoring.
- FORGE-K shadow does not compile context.
- FORGE-K shadow does not create RAG output.

### No Content Capture

- No source text is captured.
- No chunk text is captured.
- No document content is captured.
- No file content is captured.
- No retrieval result body is captured.
- No raw user query text is captured.
- No search snippet is captured.
- No embedding or vector is captured.
- No memory content is captured.
- No prompt text is captured.
- No completion or model output is captured.
- No request body is captured.
- No response body is captured.
- No auth, cookie, bearer token, API key, or secret-looking metadata is captured.

### No Response Or Route Drift

- Route inventory is unchanged.
- No public diagnostics route is added.
- Representative retrieval/search route response status codes are unchanged.
- Representative retrieval/search route response headers are unchanged.
- Representative retrieval/search route response bodies are unchanged.
- Retrieval route matching and handler selection are unchanged.
- Timeout behavior is unchanged.

### Behavior And Count Stability

- Retrieval behavior is unchanged.
- Retrieval result count is unchanged.
- Retrieval result ordering is unchanged unless live retrieval itself changes independently.
- Context compile behavior is unchanged.
- Modelruntime call count is unchanged.
- Gateway/tool execution count is unchanged.
- Memory write count is unchanged.
- Controllane mutation count is unchanged.
- Approval, permission, lane, and audit authority decisions are unchanged.

### Failure Isolation

- Shadow redaction failure drops the diagnostic report and does not fail the live request.
- Shadow sink failure drops the diagnostic report and does not fail the live request.
- No-effect validation failure drops the diagnostic report and does not fail the live request.
- Observer panic recovery preserves the live response.
- Bounded sink overflow drops oldest or rejects new diagnostics according to policy without affecting requests.

### Workspace And Security

- Workspace ids are preserved only when already available and safe.
- Retrieval run/result refs are preserved only when already available and safe.
- Source refs/fingerprints are preserved only when already available and safe.
- Secret-looking metadata is rejected.
- Cross-workspace report leakage is rejected.
- Reports remain diagnostic-only and non-authoritative.
- Reports are not treated as memory, evidence, ContextBlocks, KV metadata, runtime input, or Kernel truth.

## Required Commands

Future Phase 12K validation must run at least:

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

Phase 12K is complete only if retrieval metadata diagnostics are disabled by default, read-only, bounded, metadata-only, non-authoritative, and proven to have no effect on live responses, live retrieval behavior, live memory writes, live context compilation, or live authority paths.

## What Not To Do

- Do not capture source text.
- Do not capture chunks.
- Do not capture document content.
- Do not capture embeddings or vectors.
- Do not capture raw queries.
- Do not capture memory content.
- Do not capture prompts.
- Do not capture completions.
- Do not capture request bodies.
- Do not capture response bodies.
- Do not observe tool payloads.
- Do not add public diagnostics APIs.
- Do not change route behavior.
- Do not modify API response shape.
- Do not call modelruntime from FORGE-K.
- Do not execute tools from FORGE-K.
- Do not query retrieval/search/embeddings from FORGE-K.
- Do not implement live RAG.
- Do not write memory from FORGE-K.
- Do not call controllane mutations from FORGE-K.
- Do not make FORGE-K live authority.
