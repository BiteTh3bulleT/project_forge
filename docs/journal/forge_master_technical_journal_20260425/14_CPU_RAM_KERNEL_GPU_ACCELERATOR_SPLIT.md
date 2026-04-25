# CPU/RAM Kernel + GPU Accelerator Split

## Operating Boundary

IMPLEMENTED DOCTRINE: FORGE core authority is CPU/RAM-only. GPU and model acceleration sit behind modelruntime and must never become canonical truth authority.

## CPU-Authoritative Surfaces

- semantic syscall validation and commit
- journal/state/provenance/audit
- context restore scoring and working-memory assembly
- policy gates and operator coordination
- Dream Mode orchestration
- gateway approvals and tool execution governance
- health/diagnostic/process surfaces

## GPU-Accelerated Surfaces

PARTIAL: GPU-aware workload classes are documented and config-wired for modelruntime policy, including interactive inference, embeddings, background rerank, Dream distillation, adapter evaluation, and adapter training.

## Failure Mode Table

| Failure | Expected Behavior | Status |
|---|---|---|
| No GPU | Core boots; modelruntime reports degraded/unavailable where needed | IMPLEMENTED doctrine/config |
| DCGM unavailable | Diagnostics degrade only | IMPLEMENTED doctrine |
| Intel telemetry unavailable | Diagnostics degrade only | IMPLEMENTED doctrine |
| modelruntime unavailable | Kernel, gateway, audit, and state remain operational | IMPLEMENTED doctrine |
| GPU pressure | Background GPU work deferred | PARTIAL |

## Safe Mode

Use `FORGE_SAFE_MODE_FORCE_CPU_ONLY=true` and `FORGE_GPU_ENABLED=false`. Safe mode must not disable semantic writes, audit, gateway approvals, or operator visibility.

