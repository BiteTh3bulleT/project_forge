# FORGE-K Online Phase 04 Semantic Syscall Facade Report

## Phase

FORGE-K Online Phase 04 - Semantic Syscall Facade.

## Summary

Phase 04 adds a deterministic semantic syscall facade under the existing live Control Lane owner. The facade derives a normalized audit envelope from the current `domain.SyscallRequest` and `ActionDefinition`, including expected effect, target object type, required capability, capability scope, safe refs, rollback metadata, and explicit authority-effect flags.

Status: `CONTROL_LANE_FACADE / AUDIT_METADATA_ONLY / NO_AUTHORITY_EXPANSION`.

The facade is attached to semantic syscall audit records as `semanticSyscallEnvelope`. It does not add routes, change API request/response contracts, change validation decisions, change commit behavior, import FORGE-K simulator services, execute tools, call modelruntime, run retrieval/search/embeddings, write memory outside existing Control Lane paths, or alter storage defaults.

## Files changed

- `services/core/internal/aios/controllane/semantic_syscall_facade.go` - adds the deterministic facade builder and audit map projection.
- `services/core/internal/aios/controllane/semantic_syscall_facade_test.go` - validates write-envelope normalization, ref redaction/deduplication, rollback metadata, and audit inclusion.
- `services/core/internal/aios/controllane/audit.go` - carries the facade envelope in Control Lane audit records and core audit payloads.
- `services/core/internal/aios/controllane/processor.go` - builds the facade during existing audit recording.
- `docs/reports/phase_04_semantic_syscall_facade.md` - this report.
- `docs/status/phase_04_semantic_syscall_facade.md` - Phase 04 status marker.
- `docs/reviews/current_phase_status.md` - current-status note and table entry.

## Tests run

- `cd services/core && go test ./internal/aios/controllane -run "SemanticSyscallFacade|ProcessorAuditIncludesSemanticSyscallFacade" -count=1` - passed.
- `cd services/core && go test ./internal/aios/controllane -count=1` - passed.
- `npm run docs:routes:check` - passed.
- `git diff --check` - passed with expected Windows line-ending warnings.
- `npm test` - passed.
- `npm run lint` - passed.
- `npm run validate:forgek` - passed.
- `npm run build:core` - passed.

## Tests not run

- Desktop validation was not run because Phase 04 added no desktop/UI code.
- Nix checks were not run because Phase 04 added no Nix files or host-substrate behavior.

## Authority impact

No authority expansion. The live owner remains `services/core/internal/aios/controllane`. FORGE-K simulator services are not imported. Existing validation, capability checks, approval gates, commit boundaries, journal append behavior, and audit recording remain owned by Control Lane.

The new envelope is audit metadata only. It records `controlLaneOwned=true` and keeps `callsModelRuntime=false`, `executesGatewayTool=false`, and `importsForgeK=false`.

## Security impact

Positive traceability change. The facade records the expected semantic effect and rollback strategy without storing raw payload content. Ref extraction is limited to ID/ref fields, deduplicated deterministically, and drops secret-looking refs.

## NixOS impact

No NixOS files, host services, WSL state, or Nix store behavior changed in this phase.

## Rollback path

Revert the Phase 04 commit. Existing semantic syscall processing can continue without the `semanticSyscallEnvelope` audit metadata.

## Remaining blockers

- Phase 05 journal/replay still needs separate hash/replay design and implementation.
- Phase 06 Courthouse admission remains blocked until a separate admission-only integration phase wires validation without canonical truth commit.
- The facade is not a replay verifier, not an admission decision, and not a FORGE-K live authority migration.
