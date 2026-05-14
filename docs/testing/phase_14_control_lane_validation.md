# Phase 14 Control Lane Validation Testing Index

This index points to the current Phase 14 Control Lane validation evidence. It is an operational index, not a new authority claim; current phase truth remains summarized in [current_phase_status.md](../reviews/current_phase_status.md).

## Scope

Phase 14 validation seams are partial live validation or read-only diagnostics only. They do not make FORGE-K simulator services live authority, write semantic memory, admit evidence, compile context, execute retrieval/search/embeddings, call modelruntime, execute tools, change gateway behavior, or route live mutation through FORGE-K.

## Phase Evidence

| Phase | Evidence | Validation focus |
|---|---|---|
| 14A | [FORGE-K operational cutover design](../architecture/forge_k_operational_cutover_design.md) | Design-only migration gates and rollback posture. |
| 14B | [Ref shape validation review](../reviews/phase_14b_ref_shape_validation_review.md) | `VALIDATE_REF_SHAPE`, capability gating, no semantic commits, forbidden-import guard. |
| 14C | [Control Lane validation expansion review](../reviews/phase_14c_control_lane_validation_expansion_review.md) | `COMPARE_REF_SHAPE`, `VALIDATE_SEMANTIC_OPERATION`, diagnostic drift, forbidden authority claims. |
| 14D | [Control Lane validation shadow reporting](../reviews/phase_14d_control_lane_validation_shadow_reporting.md) | Disabled-by-default internal report shape, bounded scalar metadata, no-effect policy. |
| 14E | [Control Lane validation shadow emission](../reviews/phase_14e_control_lane_validation_shadow_emission.md) | Best-effort observer wiring from live validation results with panic isolation and unchanged syscall results. |
| 14F+ | [Current phase status](../reviews/current_phase_status.md) | Later activation metadata, lane closures, readiness surfaces, and source-object authority validation summaries. |

## Focused Command Families

Use the exact command set recorded in each phase review when revalidating that phase. The common focused package families are:

```sh
cd services/core && go test ./internal/refvalidation ./internal/semanticvalidation ./internal/aios/controllane -count=1
cd services/core && go test ./internal/forgekshadow ./internal/config -count=1
cd services/core && go test ./internal/forgek/... -count=1
npm run build:core
npm run lint
npm test
npm run test:forgek:parity
npm run test:integration:env
git diff --check
```

On Windows hosts, API package validation may be environment-sensitive where Unix-only hostbridge code is involved. Record that as host validation evidence rather than broadening the Phase 14 authority claim.
