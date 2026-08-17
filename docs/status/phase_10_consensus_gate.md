# FORGE-K Online Phase 10 Consensus Gate Status

## Phase

FORGE-K Online Phase 10 - Consensus Gate.

## Status marker

`CONSENSUS_GATE_CURRENT_MODEL_SURFACES / LIVE_API_OWNED / FINAL_RESPONSE_GUARD_ONLY / UNCERTAIN_CONTENT_WITHHELD / NO_CANONICAL_TRUTH_COMMIT / NO_EVIDENCE_ADMISSION / NO_FORGE_K_CONSENSUS_AUTHORITY`

## Summary

All current model-backed assistant final-response surfaces pass through a deterministic live-owned consensus gate after the runtime-proposal boundary and before response visibility or assistant-message persistence. Unsupported high-risk action claims are withheld unless exact gateway execution evidence is present. Any candidate whose consensus status remains `uncertain`, including ordinary model-only output without governed evidence, is replaced with a deterministic withholding message rather than passed through.

## Live owner

The live owner is `services/core/internal/api` for chat response composition and persistence, using the pure `services/core/internal/consensusgate` contract. Model output proposal metadata remains owned by `services/core/internal/modelruntime`. Tool/action authority remains owned by `services/core/internal/gateway` plus approvals, permissions, and lanes. Canonical semantic mutation remains owned by existing `services/core/internal/aios/controllane` paths.

## Target FORGE-K owner

FORGE-K Consensus Mesh (`services/core/internal/forgek/consensus`) remains the target owner for future consensus semantics. This phase does not import or invoke simulator Consensus Mesh services as live authority.

## Authority impact

No canonical truth commit. No evidence admission. No memory mutation. No gateway/tool execution. No modelruntime call from the gate. No context compilation. No route authority change. No Control Lane commit behavior change. No FORGE-K Consensus Mesh live authority.

## Tests/evidence

Validation commands are recorded in `docs/reports/phase_10_consensus_gate.md`.

## Rollback

Revert the Phase 10 commit to remove the pure consensus gate package, modelruntime chat response guard wiring, readiness/status docs, and tests. Existing chat/modelruntime/gateway behavior can continue without data or host rollback.

## Blockers

- FORGE-K Consensus Mesh is not live decision authority.
- The gate currently receives governed evidence only where API coordination supplies exact evidence refs or gateway execution state. Ordinary model-only chat therefore remains uncertain and is intentionally withheld until an evidence-aware composition contract is wired.
- FORGE-K Consensus Mesh remains simulator-only; this live guard is the bounded API composition gate.
- Consensus accepted/uncertain/withheld status is response-composition metadata only, not canonical truth, admitted evidence, approval, tool execution authority, or Kernel commit.

## Next phase

Run the next phase as a separate bounded commit. Do not combine future consensus expansion, Lymphatic Lane work, or operator cockpit work into this phase.
