# Phase 18 Legacy Retirement Status

Status: `LEGACY_RETIREMENT_PROOF_READ_ONLY / DIRECT_MUTATION_RETIRED / REPLACEMENT_AND_ROLLBACK_RECORDED / NO_AUTHORITY_EXPANSION`

Date: 2026-05-18.

## Summary

Phase 18 records the retired direct mutation surfaces in the read-only system status payload:

- `POST /api/adapters/{id}/invoke` remains unrouted.
- `POST/PATCH /api/memory/observations*` remains retired as `410 Gone` with denied audit records.

Each retired surface now has operator-visible owner, replacement, and rollback-proof metadata under `legacy_retirement` in `GET /forge/system/status`.

## Authority

- Live owner: existing API route inventory, Gateway, and Control Lane retirement gates.
- Target FORGE-K owner: future FORGE-K capability/admission/Kernel boundaries after separate migration phases.

## Boundaries

- No route is added.
- No adapter execution path is reopened.
- No memory mutation path is reopened.
- No gateway, Control Lane, modelruntime, retrieval, or storage authority changes.
- No FORGE-K simulator service is imported into live authority.

## Evidence

Validation evidence is recorded in `docs/reports/phase_18_legacy_retirement.md`.
