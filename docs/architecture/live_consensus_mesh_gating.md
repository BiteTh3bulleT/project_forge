# Live Consensus Mesh Gating

Status date: 2026-05-18.

Status: `CONSENSUS_GATE_MODEL_RUNTIME_ONLY / LIVE_API_OWNED / FINAL_RESPONSE_GUARD_ONLY / NO_CANONICAL_TRUTH_COMMIT / NO_EVIDENCE_ADMISSION / NO_FORGE_K_CONSENSUS_AUTHORITY`.

## Intent

FORGE-K Consensus Mesh remains target architecture. Live FORGE currently has only a narrow deterministic response-composition guard for modelruntime-backed assistant final responses.

The live guard exists to prevent unsupported high-risk model-output claims, such as claims that a file was written or a command was executed, from being persisted as final assistant responses without gateway/audit-style execution evidence.

## Live Owner

The live owner is `services/core/internal/api` for chat response composition and persistence, using the pure `services/core/internal/consensusgate` contract.

Related authority remains separate:

- `services/core/internal/modelruntime` owns model output proposal metadata.
- `services/core/internal/gateway` owns tool execution.
- approvals, permissions, and lanes own execution gates.
- `services/core/internal/aios/controllane` owns canonical semantic mutation.

## Target Owner

`services/core/internal/forgek/consensus` remains the target owner for future Consensus Mesh semantics. This phase does not import or invoke simulator Consensus Mesh services from live API, modelruntime, gateway, memory, retrieval, or Control Lane paths.

## Current Behavior

Modelruntime-backed assistant final responses are evaluated before assistant message persistence. The gate records one of:

- `accepted_metadata_only`
- `uncertain`
- `withheld`

Unsupported high-risk action claims are withheld unless execution evidence exists. The gate metadata explicitly records no canonical truth, no memory mutation, no evidence admission, no gateway execution, no modelruntime call, no context compilation, and no live authority migration.

## Boundaries

Consensus accepted is response-composition eligibility only. It is not canonical truth, admitted evidence, approval, tool execution authority, semantic memory write, or Kernel commit.

Gateway, native Ollama, deterministic shortcut, and streaming token surfaces are not fully consensus gated in this phase. Extending the guard to those surfaces requires a later phase with tests and rollback evidence.
