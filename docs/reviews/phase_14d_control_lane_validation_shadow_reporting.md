# Phase 14D Control Lane Validation Shadow Reporting

Status: implemented and tested.

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / CONTROL_LANE_VALIDATION_DIAGNOSTICS`.

Date: 2026-05-09.

## Summary

Phase 14D adds internal shadow reporting for the Phase 14B/14C Control Lane validators.

This phase does not wire FORGE-K simulator services into live authority. It adds a bounded diagnostic observation type under `services/core/internal/forgekshadow` for validation summaries only.

## What Changed

- Added `ControlLaneValidationInput`.
- Added `ControlLaneValidationObservation`.
- Added `Observer.ObserveControlLaneValidation`.
- Added `Observer.ObserveControlLaneValidationBestEffort`.
- Added disabled-by-default config field `FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED`.
- Added diagnostic persistence mapping for `control_lane_validation` reports.
- Added tests for flag gating, metadata-only storage, forbidden effects, unsafe metadata rejection, side-effect policy rejection, config defaults/env parsing, and no forbidden imports through existing guards.

## Boundaries

Control Lane validation shadow reporting stores only bounded scalar metadata:

- action,
- validation kind,
- decision,
- pass/fail,
- match/drift counts,
- operation type,
- normalized ref count,
- failure/warning counts,
- duration.

It does not store prompts, completions, request bodies, response bodies, source text, chunk text, memory content, tokens, secrets, raw refs with sensitive values, or model outputs.

## No-Effect Policy

Reports are diagnostic only.

Phase 14D does not:

- change routes,
- add public APIs,
- affect user-visible output,
- execute Control Lane requests,
- mutate Control Lane state,
- write semantic memory,
- admit or reject evidence,
- compile context,
- execute retrieval/search/embeddings,
- call modelruntime,
- execute tools,
- make comparison drift a live decision,
- make FORGE-K simulator services live authority.

## Validation

Validation command set for this pass:

- `cd services/core && go test ./internal/forgekshadow ./internal/config -count=1`
- `cd services/core && go test ./internal/semanticvalidation ./internal/refvalidation ./internal/aios/controllane -count=1`
- `cd services/core && go test ./internal/forgek/... -count=1`
- `cd services/core && go test ./internal/api -run 'TestServerRouteInventory|TestServerRouteInventoryHealthAndMiddlewareSmoke' -count=1`
- `npm run build:core`
- `npm run lint`
- `npm test`
- `npm run test:forgek:parity`
- `npm run test:integration:env`
- `git diff --check`
