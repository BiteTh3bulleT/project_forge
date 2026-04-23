# Full Project FORGE Review

Date: 2026-04-22  
Scope: convergence hardening audit after Phase 5.996 cutover pass.

## 1) Executive Summary

FORGE remains real, bootable, and materially implemented across v1 operational services and v2 control-lane/tool-policy/autonomy systems. This pass kept restore parity improvements, made restore guarantees more explicit (`atomicScope` + non-DB warnings), removed legacy adapter route execution in favor of gateway-only invocation, and moved VSA dependency files into authoritative tracked source state while preserving strict preflight checks. Runtime authority is clearer because gateway is now the only tool-execution path, and Model Runtime M3 now provides a FORGE-owned managed model runtime with scheduler, limits, import/register flows, lifecycle governance, backend selection, and audit coverage.

## 2) What Improved In This Pass

1. Restore parity improved to near-complete for non-derived sections.
- Newly restorable: `project_context_records`, `evaluation_records`, `audit_records`, `gateway_invocations`.
- Remaining export-only set is now VSA-derived maintenance/state tables.

2. Restore failure safety improved materially.
- Restore now uses one DB transaction across all supported selected sections.
- Late failure rolls back already-applied sections.
- Restore results now expose `atomic`, `atomicScope`, `globalAtomic`, `nonDbSideEffects`, `applied`, and `rolledBack` for honest guarantees.

3. Legacy adapter execution side route is removed.
- `/api/adapters/{id}/invoke` is no longer registered and returns `404 Not Found`.
- Desktop probe flow now invokes adapters through `/api/gateway/invoke` (`toolId/laneId: legacy.adapter.invoke`).
- Policy/permission/audit behavior for adapter invocation now applies only through gateway invocation records/audits.

4. Legacy v1 memory mutation boundary trace quality improved.
- Legacy memory mutation audits now include correlation + trace/workspace payload context.
- Default-off posture remains enforced by `FORGE_ALLOW_LEGACY_MEMORY_MUTATIONS=true`.

5. VSA status is authoritative source in this branch snapshot (not generated, not optional).
- Added `scripts/check-vsa-files.sh`.
- Required VSA sources are now tracked in repository state.
- Core/smoke entrypoints require VSA files to be present and tracked in git for bring-up integrity posture.
- Root `build:core`, `test:core`, and `vet:core` also enforce `--require-tracked` and fail early with actionable guidance.

6. Model Runtime M3 extends the native runtime into managed compute assets.
- Local GGUF imports and manifest-backed directory registration are implemented.
- Persistent lifecycle state now tracks imported, verified, disabled, archived, and preferred/default model metadata.
- `openai_compat` backend support is real, with a vLLM-compatible path through the same transport shape.
- Internal runtime APIs now cover import, verify, enable, disable, archive, remove-registration, compatibility, backend status, and usage.
- Destructive file delete remains deferred; M3 keeps archive/remove-registration separate from deletion.

7. Restore side-effect honesty is now explicit.
- Restore reports `atomicScope=db-supported-sections-only`.
- Restore reports warnings when sections include non-DB side effects (for example artifact file bytes are not imported/rollback-managed).

## 3) Current Critical / High Blockers

### Critical

1. VSA-derived sections remain intentionally non-restorable (export-only policy), which still limits full parity but is now an explicit, tested posture rather than an unresolved policy gap.

### High

1. End-to-end trace/explain visibility is still partial for some artifact-centric/operator inspection flows, even though API-level correlation inspection is now consolidated in `/api/audit/trace/{correlationId}` (gateway/audit/artifact/provenance/journal links in one report).
2. Restore still does not provide cross-system rollback beyond the DB transaction boundary (for example filesystem artifacts already present on disk are not part of DB rollback semantics).
3. Model Runtime M3 is implemented but still limited: no streaming response path yet, no delete-file workflow, no dedicated gateway `model.*` aliasing, and no stronger backend/process supervision beyond current scope.

## 4) Medium Blockers

1. Rule-agent set remains intentionally narrow (`OpenLoopStalenessAgent`, `CleanupProposalAgent`).
2. Compute lane seam/runtime split remains documented but unresolved.
3. Nix validation remains environment-blocked in this host (daemon unavailable).

## 5) Reality Verdict

FORGE is materially closer to one authoritative runtime: restore semantics are safer and more complete, tool execution authority is now gateway-only, and validation/bring-up truth is stronger. Remaining blockers are concentrated in inspection/trace completeness, non-DB rollback boundaries, and advancing Model Runtime M3 toward M4 capabilities.

## 6) Recommended Next Move

Continue remaining cutover blockers or move to Model Runtime M4:
1. Maintain the explicit VSA export-only restore policy (with recompute/reindex expectations) unless a controlled derivation-safe import mode is introduced.
2. Extend trace/explain inspection coverage for artifact-linked flows.
3. Continue Model Runtime M4 work (streaming, delete-file approval flow, stronger backend/process supervision, and gateway `model.*` capability aliasing) on top of the landed M3 foundation.
