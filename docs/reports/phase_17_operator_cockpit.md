# Phase 17 Operator Cockpit Report

Status: `OPERATOR_COCKPIT_INDEX_READ_ONLY / EXISTING_STATUS_AND_INSPECTOR_POINTERS / NO_MUTATION_CONTROLS / NO_AUTHORITY_EXPANSION`

Date: 2026-05-18.

## Scope

Add UI/status surfaces for gates, cases, context bundles, proposals, journal/replay, and lymphatic reports without changing live authority.

## Current Authority

- Live owner: existing desktop System surface backed by `GET /forge/system/status` and existing inspector APIs.
- Target FORGE-K owner: future FORGE-K Kernel, Courthouse, Context Compiler, Journal/Replay, and Lymphatic surfaces after separate authority migration phases.

## Changes

- Updated `apps/desktop/src/pages/SystemPage.tsx`.
  - Adds read-only Operator Cockpit Index rows for authority gates, cases, context bundles, proposals, journal/replay, and lymphatic reports.
  - Renders FORGE-K subsystem authority matrix rows when reported by `kernel_activation.authority_matrix`.
  - Renders storage cutover readiness metadata when reported by `storage.cutover_readiness`.
- Updated `apps/desktop/src/lib/api/types.ts`.
  - Adds TypeScript shape for `kernel_activation.authority_matrix`.
  - Adds TypeScript shape for `storage.cutover_readiness`.
- Updated `apps/desktop/src/pages/SystemPage.test.tsx`.
  - Proves the cockpit index and read-only matrix/readiness panels render.
  - Continues to prove no approve, reject, execute, delete, cleanup, restart, shutdown, rebuild, or model load/unload buttons appear.
- Updated system cockpit and shell surface documentation plus current status/authority docs.

## Validation

Passed:

- Red test before implementation: `npm -w @forge/desktop run test -- src/pages/SystemPage.test.tsx` failed because `FORGE-K Subsystem Cockpit` was not rendered.
- `npm -w @forge/desktop run test -- src/pages/SystemPage.test.tsx`
- `npm run lint:js`
- `npm run validate:js`
- `npm run typecheck`
- `npm run docs:routes:check`
- `git diff --check` passed with line-ending warnings only.
- `npm run lint`
- `npm test`
- `npm run validate:forgek`
- `npm run build:core`
- `npm run validate:desktop`

## Not Done

- No dedicated case UI.
- No dedicated ContextBundle detail panel.
- No journal replay execution UI.
- No lymphatic cleanup execution UI.
- No approval/action controls.
- No new API routes.
- No FORGE-K simulator services imported into live authority.
