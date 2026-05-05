# Phase 12B Shadow Harness Test Plan

Status: Phase 12A planning artifact. Phase 12B is not started.

These tests must exist before or during any future Phase 12B read-only shadow harness implementation.

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

## What Not To Do

- Do not treat this test plan as permission to implement Phase 12B.
- Do not add tests by changing public API behavior.
- Do not validate shadow mode by executing real tools, models, retrieval, embeddings, or memory writes.
