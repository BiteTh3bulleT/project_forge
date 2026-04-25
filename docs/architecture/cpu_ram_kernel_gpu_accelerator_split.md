# CPU-RAM Kernel + GPU Accelerator Split

Status date: 2026-04-24.

This note defines the FORGE operating model boundary:

- deterministic kernel/control/state authority is CPU/RAM only
- GPU is a governed accelerator path behind modelruntime

## CPU-authoritative surfaces

`forge-core` remains CPU/RAM authority for:

- semantic syscall kernel and control-lane validation/commit
- state registry and journal/audit/provenance linkage
- context restore scoring and working-memory assembly
- policy gates, scheduler policy decisions, and operator coordination
- Dream Mode orchestration and maintenance control flow
- health/diagnostic/process operator surfaces
- gateway approvals and tool execution governance

Canonical truth mutation must not require GPU presence.

## GPU-accelerated surfaces

`forge-modelruntime` is the only GPU-aware execution substrate.

GPU workload classes:

- `INTERACTIVE_INFERENCE`
- `INTERACTIVE_EMBEDDING`
- `BACKGROUND_EMBEDDING`
- `BACKGROUND_RERANK`
- `DREAM_DISTILLATION`
- `ADAPTER_EVAL`
- `ADAPTER_TRAINING`

Policy posture:

- interactive classes are prioritized over background classes
- background classes are bounded by idle/cooldown/concurrency policy
- optional NVIDIA DCGM telemetry can defer background classes under GPU memory pressure
- optional Intel Level Zero telemetry can report Intel GPU availability and engine utilization when local tools are installed
- Dream GPU sub-jobs are optional and disabled by conservative default

## Failure boundaries

- modelruntime failure does not prevent `forge-core` boot.
- modelruntime unavailability must not block kernel truth, journal, gateway, or operator surfaces.
- DCGM telemetry unavailability degrades diagnostics only; it must not block core boot.
- Intel Level Zero tool absence degrades diagnostics only; it must not block core boot.
- no inference path bypasses modelruntime/governance to mutate truth.
- no tool execution bypasses gateway approval/capability policy.

## Degraded safe mode behavior

Safe mode (`FORGE_SAFE_MODE_FORCE_CPU_ONLY=true`) forces CPU-only operation:

- core remains authoritative and operational
- modelruntime GPU-aware classes are disabled/deferred by policy
- health/process surfaces expose degraded reasons explicitly
- requests that require unavailable GPU paths fail with deterministic errors

This split is an operating model boundary, not a best-effort optimization.
