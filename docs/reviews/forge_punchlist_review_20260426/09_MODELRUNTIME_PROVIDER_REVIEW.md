# Modelruntime / Provider Review

## Scorecard

- Registry/import/register/reconcile: GOOD
- Lifecycle management: GOOD/PARTIAL
- Governance/approval/capability: GOOD/PARTIAL
- Backend selection: PARTIAL
- Scheduler/queue: PARTIAL
- OpenAI-compatible path: PARTIAL
- vLLM path: PARTIAL
- DCGM/Level Zero/TEI: PARTIAL
- Safe degraded mode: GOOD

## Findings

GOOD: Modelruntime M3 management exists with registry, store, import, scan, verify, enable/disable/archive/remove, load/unload, compatibility, usage, health, queue, and loaded-status surfaces.

GOOD: High-risk model operations have governance/approval/capability handling and tests.

RISK: Streaming is not implemented; streaming requests are rejected.

RISK: vLLM path is OpenAI-compatible HTTP shape, not full vLLM lifecycle/process supervision.

RISK: GPU-required admission checks config posture more than real telemetry availability.

PARTIAL: Scheduler is in-memory/simple FIFO; restart durability and dispatch timeout enforcement need work.

RISK: Archive path removes existing archive destination before rename; approval-gated at API level, but the primitive is destructive if called incorrectly.

MISSING: Cost/budget/egress governance for remote/cloud provider inference.

## M4 Punchlist

- `MR-001`: Implement streaming with cancellation-safe audit/usage.
- `MR-002`: Add backend process supervision for llama.cpp/vLLM.
- `MR-003`: Split remove registration, archive, and delete-file approval flow.
- `MR-004`: Make GPU-required admission depend on actual telemetry/backend availability.
- `MR-005`: Add durable scheduler state or explicitly document restart loss.
- `MR-006`: Add provider cost/budget/egress policy.
- `MR-007`: Add gateway `model.*` aliases only after governance semantics are stable.

