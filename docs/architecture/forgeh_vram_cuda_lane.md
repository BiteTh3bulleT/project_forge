# FORGE-H VRAM and CUDA Lane

Status: DESIGN_ONLY / FUTURE / PARTIAL_PURE_LABELS / NO_LIVE_AUTHORITY_CHANGE

## Purpose

FORGE-H may govern GPU and VRAM posture as resource policy. CUDA may accelerate cognition, but CUDA may not author truth.

The lane is a future governance boundary for GPU memory pressure, resource recommendations, approved acceleration work classes, and eventually bounded CUDA execution. It is not a modelruntime bypass and not a memory/journal authority path.

## Design Objects

| Object | Purpose | Authority boundary |
|---|---|---|
| `VramRegion` | A bounded owned region of GPU memory. | Future acceleration metadata only; not canonical memory. |
| `VramLease` | Time/scope/risk-bound claim over a `VramRegion`. | Requires governed allocation; cannot expose raw pointers. |
| `CudaBufferRef` | Stable reference to an owned CUDA buffer. | Ref only; no raw pointer exposure to agents/UI. |
| `CudaKernelArtifact` | Hash-addressed approved kernel artifact. | Artifact evidence; not automatically executable. |
| `CudaKernelLaunchProposal` | Proposal to launch an approved kernel with bounded inputs/outputs. | Requires approval/policy before future execution. |
| `GpuWorkClass` | Work classification for scheduling and pressure policy. | Pure label in M5; no execution semantics. |
| `GpuMemoryPressureEvent` | Bounded event describing pressure changes. | Diagnostic/resource evidence only. |
| `CudaBackendProfile` | Future backend posture for CUDA-capable acceleration workers. | Must remain behind modelruntime/FORGE-H governance. |
| `KvCacheResidencyPolicy` | Future policy for whether acceleration metadata may reside in GPU memory. | KV remains acceleration, not memory. |

## GPU Work Classes

M5 adds pure labels in `services/core/internal/forgeh/cuda_lane.go` for:

- `inference`
- `embedding`
- `reranking`
- `vector_scoring`
- `kv_cache_analysis`
- `simulation`
- `batch_diagnostics`
- `compression_prepass`

These labels do not allocate VRAM, launch kernels, call modelruntime, change scheduling behavior, add routes, or affect live state. They are contract vocabulary for future FORGE-H policy.

## Implementation Ladder

1. Observe VRAM pressure.
2. Report GPU/VRAM posture.
3. Recommend scheduling decisions.
4. Create approved resource proposals.
5. Record bounded internal policy changes.
6. Future: allocate governed VRAM regions.
7. Future: launch approved kernels.
8. Future: `cuda-oxide` experimental backend.
9. Future: CUDA VMM/IPC advanced mode.

No step may skip proposal, approval, audit, and rollback requirements when host/runtime risk is present.

## Acceleration Allow List

CUDA/VRAM may accelerate:

- inference;
- embedding;
- reranking;
- vector scoring;
- KV/cache analysis;
- simulation;
- batch diagnostics;
- compression/prepass work.

CUDA/VRAM must not:

- commit canonical truth;
- bypass modelruntime;
- bypass gateway;
- bypass approval;
- write semantic memory directly;
- mutate host state;
- run unapproved kernels;
- expose raw GPU pointers to agents or UI.

## Explicitly Forbidden

- raw pointer exposure to agents;
- arbitrary VRAM scanning;
- direct GPU memory mutation outside registered owned buffers;
- public mutation routes;
- unapproved kernel launch;
- modelruntime bypass;
- memory/journal bypass;
- FORGE-K live authority expansion;
- CUDA workers owning canonical truth.

## Failure And Safe-Mode Behavior

GPU telemetry absence is `unavailable`, not a core failure. CPU/RAM `forge-core` authority must continue without GPU. `FORGE_SAFE_MODE_FORCE_CPU_ONLY=true` disables/deprioritizes GPU classes and should make future CUDA lanes report unavailable or deferred rather than partially active.
