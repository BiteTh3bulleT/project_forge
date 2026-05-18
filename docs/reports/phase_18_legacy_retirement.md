# Phase 18 Legacy Retirement Report

Status: `LEGACY_RETIREMENT_PROOF_READ_ONLY / DIRECT_MUTATION_RETIRED / REPLACEMENT_AND_ROLLBACK_RECORDED / NO_AUTHORITY_EXPANSION`

Date: 2026-05-18.

## Scope

Retire legacy direct mutation paths only after default-live replacement and rollback proof.

## Current Authority

- Live owner: existing API route inventory, Gateway, and Control Lane retirement gates.
- Target FORGE-K owner: future FORGE-K capability, Courthouse admission, and Kernel syscall boundaries after separate migration phases.

## Changes

- Updated `services/core/internal/api/system_status.go`.
  - Adds read-only `legacy_retirement` metadata to `GET /forge/system/status`.
  - Records retired adapter direct invoke and retired memory observation mutation entries.
  - Includes route state, mutation-disabled flag, default-live replacement, rollback proof, audit expectation, live owner, and target FORGE-K owner.
- Updated `services/core/internal/api/system_status_test.go`.
  - Proves the status payload includes both retired entries and required replacement/rollback fields.
  - Proves direct mutation remains disabled in the report.
- Updated current status, authority, legacy adapter, and shell surface documentation.

## Validation

Passed:

- Red test before implementation: `cd services/core; go test ./internal/api -run TestForgeSystemStatusReadOnlySurface -count=1` failed because `legacy_retirement` was missing.
- `cd services/core; go test ./internal/api -run TestForgeSystemStatusReadOnlySurface -count=1`
- `cd services/core; go test ./internal/api -run "ForgeSystemStatus|LegacyAdapterInvokeRouteRemoved|LegacyMemoryMutationEndpointsAreRetired|RouteInventoryCompatibilityAndRetiredRoutes" -count=1`
- Forbidden simulator import scan over touched live API files returned no matches.
- `npm run docs:routes:check`
- `git diff --check` passed with line-ending warnings only.
- `npm run lint`
- `npm test`
- `npm run validate:forgek`
- `npm run build:core`

## Not Done

- No legacy route was restored.
- No memory mutation endpoint was reopened.
- No new syscall-native memory write facade was added.
- No FORGE-K simulator service was imported into live authority.
