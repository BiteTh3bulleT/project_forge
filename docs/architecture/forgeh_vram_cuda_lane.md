# FORGE-H VRAM and CUDA Lane

Status: DESIGN_ONLY / FUTURE / NO_LIVE_AUTHORITY_CHANGE

## Intent

FORGE-H may govern GPU and VRAM posture as resource policy. CUDA may accelerate cognition, but CUDA may not author truth.

## Design Objects

- `VramRegion`
- `VramLease`
- `CudaBufferRef`
- `CudaKernelArtifact`
- `CudaKernelLaunchProposal`
- `GpuWorkClass`
- `GpuMemoryPressureEvent`
- `CudaBackendProfile`
- `KvCacheResidencyPolicy`

## Implementation Ladder

1. Observe VRAM pressure.
2. Report GPU/VRAM posture.
3. Recommend scheduling decisions.
4. Create approved resource proposals.
5. Record bounded internal policy changes.
6. Future: allocate governed VRAM regions.
7. Future: launch approved kernels.
8. Future: cuda-oxide experimental backend.
9. Future: CUDA VMM/IPC advanced mode.

## Forbidden Behavior

- Raw pointer exposure to agents.
- Arbitrary VRAM scanning.
- Direct GPU memory mutation outside registered owned buffers.
- Public mutation routes.
- Unapproved kernel launch.
- Modelruntime bypass.
- Memory/journal bypass.
- FORGE-K live authority expansion.

## Authority Boundary

FORGE-H recommendations are resource evidence. GPU buffers, CUDA kernels, and KV residency are acceleration-domain artifacts, not canonical memory.
