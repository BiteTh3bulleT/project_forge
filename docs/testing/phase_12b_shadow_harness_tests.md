# Phase 12B Shadow Harness Test Plan

Status: Phase 12B implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

These tests define the required no-effect threshold for the read-only shadow harness.

## Required Tests

- Feature flag defaults to disabled.
- Disabled flag produces no shadow reports.
- Enabled flag does not change live response body, status, headers, or public response shape.
- Route inventory is unchanged.
- No extra routes are added unless separately approved.
- Shadow report generation occurs only behind the flag.
- Shadow report failure does not fail the live request.
- Kill switch stops report generation.
- No live state mutation occurs from shadow mode.
- No tool execution occurs from shadow mode.
- No gateway invocation or approval decision is created by shadow mode.
- No modelruntime call, model load, model unload, or scheduler mutation occurs from shadow mode.
- No retrieval, search, embedding, or VSA query is executed from FORGE-K.
- No memory note, memory observation, usefulness, repair, or controllane write occurs from shadow mode.
- No live context compile is executed from FORGE-K.
- Diagnostic reports reject or redact secret-looking metadata.
- Diagnostic reports use refs and bounded summaries instead of large raw blobs.
- Diagnostic records are clearly marked non-authoritative if persistence is approved.
- Shadow mode preserves workspace and correlation scope.
- Shadow mode works with report sink failure.
- Shadow mode has bounded timeout and size behavior.

## Acceptance Threshold

Phase 12B cannot be considered complete unless:

- all no-effect tests pass
- route inventory tests pass
- public response equivalence tests pass
- mutation denial tests pass
- forbidden execution tests pass
- rollback and kill-switch tests pass

## Implemented Test Coverage

Current Phase 12B tests cover:

- `FORGE_K_SHADOW_MODE_ENABLED` defaults to disabled and parses explicit enable/invalid values.
- Disabled shadow observer stores no reports.
- Enabled shadow observer stores diagnostic reports only.
- `/health` response status, body, and content type are unchanged with shadow mode enabled.
- Route inventory key set is unchanged with shadow mode enabled.
- Shadow report sink failure does not fail the live `/health` request.
- Diagnostic sink retention is bounded and drops oldest reports.
- Secret-looking metadata is rejected.
- No-effect policy failures are rejected.
- `forgekshadow` forbidden import tests prevent gateway, modelruntime, retrieval/search/embedding, memory, AI-OS, and controllane imports.

Current validation commands:

- `cd services/core && go test ./internal/forgekshadow/...`
- `cd services/core && go test ./internal/api -run "TestServerRouteInventory|TestServerRouteInventoryHealthAndMiddlewareSmoke" -count=1`
- `cd services/core && go test ./internal/forgek/...`
- `npm run build:core`
- `npm run lint`
- `npm test`
- `npm run test:forgek:parity`
- `git diff --check`

## What Not To Do

- Do not add tests by changing public API behavior.
- Do not validate shadow mode by executing real tools, models, retrieval, embeddings, or memory writes.
