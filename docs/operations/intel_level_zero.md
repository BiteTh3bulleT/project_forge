# Intel Level Zero Telemetry

Status date: 2026-04-24.

FORGE can optionally probe Intel GPU availability through Level Zero tooling. This is a local accelerator diagnostic path; it is not required for boot and it does not create truth authority.

## Local Check

On this machine, the pass found:

- Intel Iris Xe Graphics present in `lspci`
- `/dev/dri/renderD128` present
- `ze_info` not on `PATH`
- `intel_gpu_top` not on `PATH`
- no Level Zero loader library was visible through `ldconfig -p`

That means FORGE can detect the Intel GPU render node, but Level Zero telemetry will report degraded until Level Zero tools are installed or explicitly configured.

## Enable

```bash
export FORGE_INTEL_LEVEL_ZERO_ENABLED=true
```

Optional tool paths:

```bash
export FORGE_INTEL_LEVEL_ZERO_ZE_INFO_PATH=/usr/bin/ze_info
export FORGE_INTEL_GPU_TOP_PATH=/usr/bin/intel_gpu_top
export FORGE_INTEL_GPU_TELEMETRY_TIMEOUT_MS=1500
```

`ze_info` is used for Level Zero device presence. `intel_gpu_top -J` is used opportunistically for engine utilization when available.

## Diagnostics

Surfaces:

- `GET /health` includes `intelTelemetry`
- `GET /api/providers/capabilities` includes `telemetry.intel_level_zero`
- `GET /forge/model-runtime/health` can include Intel telemetry when it is the active configured GPU telemetry source

Missing Level Zero tools produce degraded diagnostics while core health remains OK.

## Safety

- Intel telemetry is disabled by default.
- Safe mode disables Intel GPU telemetry admission by forcing CPU-only posture.
- Intel GPU availability never authorizes canonical memory mutation.
- Inference or embeddings must still pass through modelruntime/provider boundaries.
