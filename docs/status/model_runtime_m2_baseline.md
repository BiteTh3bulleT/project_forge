# Model Runtime M2 Baseline

Date: 2026-04-22
Scope: verify the actual Model Runtime M1 baseline before M2 hardening.

## Baseline Verification

| Area | Status | Evidence | Notes |
|---|---|---|---|
| Model domain types | present | `services/core/internal/modelruntime/types.go` | Explicit enums and request/result payloads exist. |
| Model manifest format | present | `services/core/internal/modelruntime/manifest.go` | `model.forge.json` parsing and validation are real. |
| Model store and registry | present | `services/core/internal/modelruntime/{store.go,registry.go}` | Local scan/load/inspect path exists. |
| Fake backend | present | `services/core/internal/modelruntime/backend_fake.go` | Deterministic test backend exists and is exercised by tests. |
| llama.cpp backend | present | `services/core/internal/modelruntime/backend_llama_cpp.go` | HTTP endpoint mode exists; spawn remains structured-unsupported. |
| Runtime service | present | `services/core/internal/modelruntime/service.go` | List/inspect/load/unload/generate/health path exists. |
| Internal FORGE endpoints | present | `services/core/internal/api/model_runtime.go` | `/forge/models*` and runtime health endpoints are wired. |
| OpenAI-compatible endpoints | present | `services/core/internal/api/model_runtime.go` | Gated `/v1/models` and `/v1/chat/completions` exist. |
| Audit path | present | `services/core/internal/api/model_runtime_bridge.go` | Runtime calls emit audit records with correlation metadata. |

## Pre-M2 Limitations Confirmed

- Runtime execution was synchronous and direct through `modelruntime.Service.Generate` with no visible request queue.
- Runtime health existed, but queue/loaded views were not backed by a real scheduler snapshot.
- Output/token timeout bounds existed, but request admission and queue pressure were not runtime-governed.
- Actor/source/workspace policy enforcement was incomplete and depended mainly on API-side shaping.
- Tool-surface taxonomy existed in docs, but gateway `model.*` capability registration was still not authoritative.

## Current Authoritative Inference Path

Authoritative request path after M2 implementation:

`/forge/models/{id}/chat` or gated `/v1/chat/completions`
-> `services/core/internal/api/model_runtime.go`
-> `services/core/internal/api/model_runtime_bridge.go`
-> `services/core/internal/modelruntime.Service.Generate`
-> runtime scheduler / policy guard / audit
-> selected backend (`fake` or `llama.cpp` endpoint adapter)

There is no second model-runtime execution path in this subsystem.

## M2 Disposition Summary

M2 is implemented on top of the verified M1 baseline by adding:

- deterministic FIFO request queueing
- bounded admission control
- one active generation at a time per backend
- explicit actor/source validation
- workspace/policy guard hooks from config
- richer audit/usage accounting
- runtime-backed queue and loaded-model inspection views

## Remaining Post-M2 Gaps

- Non-streaming only; streaming remains deferred.
- llama.cpp spawn/process supervision remains deferred.
- Model import/delete/benchmark flows remain deferred.
- Gateway capability-registry aliasing for `model.*` remains a follow-up.
- Scheduling is intentionally simple: FIFO queue, no batching, no distributed execution.
