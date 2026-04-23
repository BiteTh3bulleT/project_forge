# Full Project FORGE Review

Date: 2026-04-23  
Scope: concise convergence review update for Phase 3-5 status/docs alignment.

## 1) Executive Summary

FORGE remains materially implemented across control-lane persistence, ingest/cell runtime, truth services, autonomy policy, and governed tool/model execution. The current code supports real Phase 3 durable semantic persistence, real but bounded Phase 4 ingest/cell flow, and partial Phase 5 truth/autonomy/tool-policy layers. Tool execution authority is gateway-only, `COMPILE_CONTEXT` restore scoring is deterministic and persisted as evidence, and Model Runtime M3 remains real but non-streaming.

## 2) Current Code-Verified Reality

1. Phase 3 persistence is landed in the control lane.
- SQLite-backed notes/links/state/loops/models/contradictions/supersessions/context snapshots are committed behind the semantic syscall processor.
- `journal_events` remains append-only, and audit linkage is attached by `syscall_id` + correlation.

2. Phase 4 ingest runtime is landed but bounded.
- Librarian pipeline and cells are real and route accepted actions back through the kernel.
- Autonomy follow-on passes are depth-bounded and skipped for dry-run/validate-only execution.

3. Phase 5 truth/autonomy/tool-policy layers are real but partial.
- Truth engine services exist for current state, timelines, contradictions, supersessions, and explain paths.
- Autonomy policy, charter/budget checks, and durable-backing quarantine are real.
- Gateway capability status/risk/approval policy is real, and `future_iris` does not bypass it.

4. Tool execution authority is clearer than before.
- `/api/adapters/{id}/invoke` is not routed.
- Legacy adapter execution remains only as gateway tool `legacy.adapter.invoke`.

5. Context restore scoring is now deterministic and inspectable.
- `COMPILE_CONTEXT` snapshot selection ranks candidates by scope/query/kind with stale, contradiction, and header-only penalties.
- `restore_scores_json` and `resume_hints_json` persist as non-canonical evidence.

6. Model Runtime M3 is real and still bounded.
- Managed import/register/verify/enable/disable/archive/remove-registration flows exist.
- `/forge/models*` and gated `/v1/*` surfaces are present.
- Streaming remains unsupported, file deletion remains deferred, and dedicated gateway `model.*` aliasing is still absent.

## 3) Current Critical / High Blockers

### High

1. v1 memory mutation still bypasses the semantic syscall kernel, so Phase 5 truth services are not yet the sole runtime memory authority.
2. Dual runtime event streams (`events` and `journal_events`) still coexist.
3. End-to-end operator trace/explain visibility remains partial even though backend correlation data is present.
4. Model Runtime M3 remains non-streaming and does not yet provide delete-file workflow, stronger backend/process supervision, or gateway `model.*` aliasing.

## 4) Medium Blockers

1. Rule-agent set remains intentionally narrow (`OpenLoopStalenessAgent`, `CleanupProposalAgent`).
2. Lane split remains architectural doctrine, not a hard package/runtime isolation boundary.
3. Operator UI inspection for snapshots/runtime traces still lags backend capability.

## 5) Reality Verdict

FORGE is materially closer to one authoritative runtime, but it is not fully converged. Phase 3 persistence is real, Phase 4 ingest runtime is real, and core Phase 5 services are real; the remaining gaps are legacy write/event paths, incomplete operator traceability, and unfinished Model Runtime M4 work.

## 6) Recommended Next Move

1. Continue converging v1 memory/event side paths into the declared control-lane authority model.
2. Extend operator-facing trace and snapshot inspection until the Phase 5 “explain what happened” bar is actually met.
3. Continue Model Runtime M4 work on top of the landed M3 baseline without documenting M4 behavior as present before it exists.
