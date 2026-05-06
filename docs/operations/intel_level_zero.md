# Intel Level Zero Telemetry

Status date: 2026-05-06.

FORGE can optionally probe Intel GPU availability through Level Zero tooling. This is a local accelerator diagnostic path; it is not required for boot and it does not create truth authority.

## Local Check

On this machine, the pass found:

- Intel Iris Xe Graphics present in `lspci`
- `/dev/dri/renderD128` present
- `ze_info` not on `PATH`
- `intel_gpu_top` can provide utilization sampling when installed
- the Level Zero loader library can be present even when `ze_info` is not packaged

That means FORGE can detect the Intel GPU render node. Full Level Zero device details still require `ze_info`, but runtime utilization telemetry can be available through `intel_gpu_top`.

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

`ze_info` is used for Level Zero device details. `intel_gpu_top -J` is used for engine utilization when available, and can keep Intel GPU telemetry available even when `ze_info` is missing.

## Docker

The Docker start helper auto-enables `docker-compose.igpu.yml` when `/dev/dri/renderD128` exists on the host:

```bash
npm run docker:start
```

To force the iGPU override or disable it explicitly:

```bash
FORGE_DOCKER_IGPU=1 npm run docker:start
FORGE_DOCKER_IGPU=0 npm run docker:start
```

The override passes `/dev/dri` into the core container, adds the host render/video group IDs, enables Intel telemetry, and points the core at `/usr/bin/intel_gpu_top`. The core image includes `intel-gpu-tools` for this diagnostic path.

Intel PMU sampling in Docker also requires the iGPU override to run the core container as root with `CAP_PERFMON`, `CAP_SYS_ADMIN`, `seccomp=unconfined`, and host user namespace mode on systems where Docker user namespace remapping is enabled. These settings are isolated to `docker-compose.igpu.yml`; the normal Compose stack does not use them.

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
