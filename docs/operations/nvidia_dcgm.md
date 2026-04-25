# NVIDIA DCGM Telemetry

Status date: 2026-04-24.

FORGE can optionally read NVIDIA DCGM exporter metrics for GPU diagnostics and background-job admission policy. This is telemetry only. It is not required for boot and it does not create truth authority.

## Enable

Run a DCGM exporter that exposes Prometheus metrics, then set:

```bash
export FORGE_NVIDIA_DCGM_ENABLED=true
export FORGE_NVIDIA_DCGM_ENDPOINT=http://127.0.0.1:9400/metrics
export FORGE_GPU_ENABLED=true
```

Optional tuning:

```bash
export FORGE_NVIDIA_DCGM_TIMEOUT_MS=1500
export FORGE_GPU_BACKGROUND_MEMORY_PRESSURE_BLOCK_THRESHOLD=0.90
```

Safe defaults:

- DCGM telemetry disabled
- GPU disabled
- background GPU jobs disabled
- safe mode disables DCGM-backed GPU admission by forcing CPU-only posture

## Metrics Used

When present, FORGE reads:

- `DCGM_FI_DEV_GPU_UTIL`
- `DCGM_FI_DEV_FB_USED`
- `DCGM_FI_DEV_FB_FREE`
- `DCGM_FI_DEV_FB_TOTAL`
- `DCGM_FI_DEV_POWER_USAGE`
- `DCGM_FI_DEV_GPU_TEMP`

If memory pressure meets or exceeds `FORGE_GPU_BACKGROUND_MEMORY_PRESSURE_BLOCK_THRESHOLD`, background GPU work is deferred. Interactive workload policy remains controlled by modelruntime.

## Diagnostics

Surfaces:

- `GET /health` includes `gpuTelemetry`
- `GET /forge/model-runtime/health` includes telemetry details when modelruntime is enabled
- `GET /api/providers/capabilities` includes `telemetry.nvidia_dcgm`

Unavailable DCGM produces a degraded telemetry state while core health remains OK.
