# Operator UI And ARTEMIS Surface

## Purpose

The operator surface should let a human see what FORGE is doing, what it knows, what it proposes, what it executed, and why.

## Current Desktop State

PARTIAL / IMPLEMENTED UI: The desktop shell includes pages for chat, workbench, canvas, dossiers, jobs, reviews, approvals, audit, settings, layouts, memory, retrieval, autonomy, gateway, models, backup, and inspectors.

Evidence: `apps/desktop/src/pages/*`, `App.tsx`, `AppShell.tsx`, `workspaceLayoutStore.ts`.

## API / Streaming Reality

IMPLEMENTED: Desktop uses REST APIs through `apps/desktop/src/lib/api.ts`. Chat assistant streaming uses SSE/EventSource via `/api/chat/threads/{id}/assistant-stream`.

NOT VERIFIED / NOT FOUND: A general websocket server route was not found in the current API scan. Modelruntime streaming is explicitly unsupported today.

## ARTEMIS Concept

CONCEPT: ARTEMIS is the future operator/perception surface: a cockpit that combines cognitive state, metrics, traces, attention routing, voice/vision if enabled, and review queues.

## Recommended UI Model

| Mode | Purpose | Status |
|---|---|---|
| Cognitive State Viewer | Trace, memory, context restore, Dream proposals, current truth | PARTIAL |
| Metrics Board | Health, queues, modelruntime, GPU, jobs, events | PARTIAL |
| Approval Console | Dangerous actions, model management, syscalls | PARTIAL |
| Work Surface | Chat/workbench/canvas/files | IMPLEMENTED / PARTIAL |

## Gaps

- MISSING: Dedicated frontend test suite.
- PARTIAL: Unified trace page is incomplete.
- PARTIAL: Restore scoring and Dream reports are not first-class operator workflows.
- PARTIAL: Shared frontend approval status types should be checked against backend expiry states.
- PLANNED: Voice/vision/perception should remain optional input surfaces, not authority.
