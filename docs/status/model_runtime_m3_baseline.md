# Model Runtime M3 Baseline

Date: 2026-04-22
Scope: verify what Model Runtime M1 and M2 actually shipped before extending M3.

Status note (2026-04-25): this file is retained as pre-M3 baseline evidence.
Current M3 routes and management behavior are implemented in code and summarized
in `docs/status/model_runtime_status.md` and `docs/architecture/model_runtime.md`.
The "before M3 extension" limitations below are historical baseline notes, not
the current runtime state.

## Baseline Verification

### M1 actually shipped

| Area | Status | Evidence | Notes |
|---|---|---|---|
| Model domain types | present | `services/core/internal/modelruntime/types.go` | Explicit enums and request/result payloads exist. |
| Model manifest format | present | `services/core/internal/modelruntime/manifest.go` | `model.forge.json` parsing and validation are real. |
| Model store and registry | present | `services/core/internal/modelruntime/{store.go,registry.go}` | Local scan/load/inspect path exists. |
| Fake backend | present | `services/core/internal/modelruntime/backend_fake.go` | Deterministic test backend exists and is exercised by tests. |
| llama.cpp backend | present | `services/core/internal/modelruntime/backend_llama_cpp.go` | HTTP endpoint mode exists; spawn remains structured-unsupported. |
| Runtime service | present | `services/core/internal/modelruntime/service.go` | List/inspect/load/unload/generate/health path exists. |
| Internal FORGE endpoints | present | `services/core/internal/api/model_runtime.go` | `/forge/models*` runtime endpoints are wired. |
| OpenAI-compatible minimum endpoints | present | `services/core/internal/api/model_runtime.go` | Gated `/v1/models` and `/v1/chat/completions` exist. |
| Audit path | present | `services/core/internal/api/model_runtime_bridge.go` | Runtime calls emit audit records with correlation metadata. |

### M2 actually shipped

| Area | Status | Evidence | Notes |
|---|---|---|---|
| FIFO scheduler and admission control | present | `services/core/internal/modelruntime/service.go` | Runtime queueing and bounded admission are real. |
| Queue/loaded/health status endpoints | present | `services/core/internal/api/model_runtime.go` | Runtime-backed inspection endpoints exist. |
| Lifecycle hardening | present | `services/core/internal/modelruntime/service.go` | Explicit load/unload, duplicate-load determinism, and lifecycle conflict handling exist. |
| Request/resource limits | present | `services/core/internal/modelruntime/service.go`, `services/core/internal/config/config.go` | Timeout, prompt/output bounds, queue depth, and concurrency limits are enforced. |
| Stronger workspace/policy hooks | present | `services/core/internal/api/model_runtime_bridge.go` | Actor/source/workspace validation and capability checks are enforced through runtime request hooks. |
| Richer audit/usage accounting | present | `services/core/internal/modelruntime/service.go` | Queue wait, token counts, output bytes, and outcome fields are recorded where available. |

## Current Backend Support Before M3 Extension

- `fake`: real and used for deterministic tests.
- `llama_cpp`: real endpoint-backed adapter.
- `openai_compat`: not present before this M3 pass.
- `vllm`: not present before this M3 pass.
- `ollama_compat`: deferred.

## Current Scheduling Behavior Before M3 Extension

- FIFO queue.
- Bounded queue depth.
- One active generation at a time per backend.
- Runtime-backed queue snapshots for pending/running/completed states.
- Context/timeouts drive cancellation behavior.
- No per-backend or per-model queue visibility beyond the shared scheduler snapshot.
- No streaming path.

## Current Lifecycle Behavior Before M3 Extension

- Explicit load/unload APIs are authoritative.
- Auto-load is config-governed and default-off.
- Runtime steady state centered on `available`, `loaded`, `disabled`, and `error`.
- No managed import/register/archive/remove-registration workflow.
- No persistent archived-model view.

## Current Policy Integration Before M3 Extension

- Actor/source required for inference requests.
- Workspace requirement can be enforced by config.
- Cross-workspace use can be denied by config.
- Unsupported model capabilities are denied.
- Disabled runtime and unsupported backend states fail explicitly.
- Dedicated gateway `model.*` capability aliasing remains outside the authoritative runtime path.

## Current API Surface Before M3 Extension

Internal FORGE runtime API:

- `GET /forge/models`
- `GET /forge/models/{id}`
- `POST /forge/models/{id}/load`
- `POST /forge/models/{id}/unload`
- `POST /forge/models/{id}/chat`
- `GET /forge/model-runtime/queue`
- `GET /forge/model-runtime/loaded`
- `GET /forge/model-runtime/health`

OpenAI-compatible minimum API:

- `GET /v1/models`
- `POST /v1/chat/completions`

## Current Authoritative Inference Path

`/forge/models/{id}/chat` or gated `/v1/chat/completions`
-> `services/core/internal/api/model_runtime.go`
-> `services/core/internal/api/model_runtime_bridge.go`
-> `services/core/internal/modelruntime.Service.Generate`
-> runtime scheduler / policy guard / audit
-> selected backend

There is no second model-runtime execution path in this subsystem.

## What M3 Truly Extends

M3 extends the landed M1/M2 baseline with:

- local model import/register/reconcile flows
- persistent model state metadata (`imported`, `verified`, `archived`, preferred/default)
- managed lifecycle operations beyond load/unload
- OpenAI-compatible backend support and vLLM-compatible adapter path
- deterministic model/backend selection improvements
- stronger runtime inspection for compatibility, backends, and usage

## Remaining Pre-M3 Limitations Confirmed

- Non-streaming only.
- No import/register/verify/archive/remove-registration workflow yet.
- No OpenAI-compatible or vLLM-compatible backend adapter yet.
- No destructive file deletion path.
- Gateway capability-registry aliasing for `model.*` remains a follow-up.
