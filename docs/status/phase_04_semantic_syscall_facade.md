# FORGE-K Online Phase 04 Semantic Syscall Facade Status

## Phase

FORGE-K Online Phase 04 - Semantic Syscall Facade.

## Status marker

`CONTROL_LANE_FACADE / AUDIT_METADATA_ONLY / NO_AUTHORITY_EXPANSION`

## Summary

Control Lane now emits a deterministic `semanticSyscallEnvelope` in semantic syscall audit records. The envelope normalizes existing syscall request/registry facts into expected effect, capability, scope, safe refs, rollback metadata, and authority-effect flags.

## Live owner

`services/core/internal/aios/controllane` remains the live owner for semantic syscall validation, capability checks, approvals, commit boundaries, journal append behavior, and audit recording.

## Target FORGE-K owner

FORGE-K Kernel remains the target owner for future semantic truth flow. This phase does not transfer ownership.

## Authority impact

No authority migration. No simulator service import. No route/API behavior change. No validation decision change. No commit behavior change. No modelruntime, gateway, retrieval, embeddings, memory, storage, or NixOS behavior change.

## Tests/evidence

Validation commands are recorded in `docs/reports/phase_04_semantic_syscall_facade.md`.

## Rollback

Revert the Phase 04 commit to remove the audit envelope. No live data or host state rollback is required.

## Blockers

- Journal hash/replay verification is still Phase 05 work.
- Courthouse admission is still Phase 06 work.

## Next phase

Run Phase 05 as a separate bounded commit. Do not combine journal replay work into this phase.
