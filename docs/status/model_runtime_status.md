# Model Runtime Status (M3/M4 Runtime Profile)

Snapshot date: 2026-05-13.

## Executive Status

Model Runtime M3 is implemented in this branch. M4 closes the external vLLM-compatible backend profile as a governed, disabled-by-default modelruntime profile and adds public SSE streaming over the existing governed chat paths when the selected backend supports token streaming. FORGE now treats models as managed runtime assets instead of loose local manifests: local GGUF or manifest-backed imports can be registered into FORGE model home, persistent model state is tracked across import/verify/disable/archive/remove-registration operations, runtime selection spans multiple pluggable backends, and the service exposes management, compatibility, usage, backend, queue, loaded, health, non-streaming chat, and streaming chat views while keeping inference policy-governed, auditable, and non-authoritative over semantic truth.

## Implemented

| Area | Status | Evidence |
|---|---|---|
| Model domain types | real | `services/core/internal/modelruntime/types.go` |
| `model.forge.json` parser/validator | real | `services/core/internal/modelruntime/manifest.go`, `manifest_test.go` |
| Model store scanning/loading/checksum validation | real | `services/core/internal/modelruntime/store.go`, `store_test.go` |
| Persistent model state (`model.state.json`) | real | `services/core/internal/modelruntime/state.go` |
| Model registry lifecycle metadata | real | `services/core/internal/modelruntime/registry.go`, `registry_test.go` |
| Local import/register/reconcile workflows | real | `services/core/internal/modelruntime/store_management.go`, `service_management_test.go` |
| Verification / enable / disable / archive / remove-registration | real | `services/core/internal/modelruntime/management.go`, `store_management_test.go` |
| Backend interface | real | `services/core/internal/modelruntime/backend.go` |
| Fake backend for tests | real | `services/core/internal/modelruntime/backend_fake.go`, `backend_fake_test.go` |
| llama.cpp backend adapter (endpoint mode) | real | `services/core/internal/modelruntime/backend_llama_cpp.go`, `backend_llama_cpp_test.go` |
| OpenAI-compatible backend adapter | real | `services/core/internal/modelruntime/backend_openai_compat.go`, `backend_openai_compat_test.go` |
| vLLM-compatible endpoint path | partial/live external profile | `services/core/internal/api/model_runtime_bridge.go`, `backend_openai_compat.go`, `docs/architecture/nix_rust_vllm_runtime.md` |
| Streaming chat responses | real | `services/core/internal/api/model_runtime.go`, `model_runtime_test.go`, `backend_openai_compat.go`, `backend_openai_compat_test.go` |
| Runtime scheduler and queue admission | real | `services/core/internal/modelruntime/service.go`, `service_test.go` |
| Runtime compatibility / usage / backend inspection | real | `services/core/internal/modelruntime/management.go`, `model_runtime_m3_test.go` |
| Internal FORGE model management API | real | `services/core/internal/api/model_runtime.go`, `server.go` |
| OpenAI-compatible minimum API | real (feature-gated) | `services/core/internal/api/model_runtime.go`, `server.go` |
| Runtime config surface | real | `services/core/internal/config/config.go`, `config_test.go` |
| Audit and usage accounting | real | `services/core/internal/api/model_runtime_bridge.go`, `service_test.go`, `service_management_test.go` |
| Runtime DB schema tables | real | `services/core/internal/store/migrate.go`, `migrate_model_runtime_test.go` |

## M3 Behavior Now Enforced

### Model Management

- Import local GGUF files into managed model-home directories with generated manifests.
- Import manifest-backed model directories.
- Reconcile registry state by rescanning model home.
- Verify managed model files and checksum metadata where available.
- Enable, disable, archive, and remove registration without conflating those operations.
- Preserve archive/remove-registration metadata without silently deleting model bytes.
- Track preferred/default model selection explicitly.

### Backend Coverage and Selection

- `fake` backend remains the deterministic test/runtime shim.
- `llama_cpp` remains the first real local-runtime backend.
- `openai_compat` is now a real endpoint-backed adapter.
- `vllm` can be targeted through the same OpenAI-compatible transport path when configured.
- `FORGE_VLLM_BASE_URL` and `FORGE_VLLM_API_KEY` are the canonical M4 vLLM env vars; older `FORGE_MODEL_VLLM_*` names remain compatibility aliases.
- vLLM backend health/status appears through `/forge/model-runtime/backends` and carries `interactive_vllm` backend profile metadata when the endpoint is configured and reachable.
- Explicit backend overrides are validated; incompatible overrides fail deterministically.
- If no model id is supplied, selection is deterministic: explicit default/preferred model first, then an unambiguous compatible candidate.

### Scheduler and Runtime State

- FIFO queue with bounded depth remains authoritative.
- One active generation at a time per backend remains the active execution rule.
- Queue/loaded/health snapshots remain runtime-backed.
- Usage summary and backend status snapshots are available through internal APIs.
- Archived or disabled models are denied for inference explicitly.

## Internal API Surface

Management and inspection:

- `GET /forge/models`
- `POST /forge/models/import`
- `POST /forge/models/scan`
- `GET /forge/models/{id}`
- `GET /forge/models/{id}/compatibility`
- `POST /forge/models/{id}/verify`
- `POST /forge/models/{id}/enable`
- `POST /forge/models/{id}/disable`
- `POST /forge/models/{id}/archive`
- `POST /forge/models/{id}/remove`
- `POST /forge/models/{id}/load`
- `POST /forge/models/{id}/unload`
- `POST /forge/models/{id}/chat`
- `GET /forge/model-runtime/backends`
- `GET /forge/model-runtime/usage`
- `GET /forge/model-runtime/queue`
- `GET /forge/model-runtime/loaded`
- `GET /forge/model-runtime/health`

OpenAI-compatible minimum:

- `GET /v1/models`
- `POST /v1/chat/completions`

Streaming:

- `POST /forge/models/{id}/chat` accepts `stream: true` and emits FORGE SSE `token`, `result`, and `done` events when `modelRuntimeStreamingService` is available.
- `POST /v1/chat/completions` accepts `stream: true` and emits OpenAI-compatible `chat.completion.chunk` SSE data followed by `data: [DONE]`.
- Streaming remains governed by the same modelruntime scheduler, policy hooks, backend selection, workspace metadata, correlation, and audit path as non-streaming chat.
- If the selected runtime service/backend cannot stream, the request fails with structured `STREAM_UNSUPPORTED` behavior before normal JSON response streaming begins.

## Audit and Evidence

Management and inference actions emit audit records with correlation metadata.

Recorded where available:

- request id
- model id and backend
- actor and source
- workspace id
- correlation id and trace id
- lifecycle action
- timeout/max token settings
- queue wait and queue depth
- duration
- prompt/completion token counts
- output bytes
- success/error outcome

Model output remains response evidence. It does not automatically mutate canonical memory or semantic truth state.

## Configuration

Runtime flags in `services/core/internal/config/config.go` now include:

- `FORGE_ENABLE_MODEL_RUNTIME` default `false`
- `FORGE_MODEL_HOME` default `${FORGE_DATA_DIR}/models`
- `FORGE_MODEL_DEFAULT_BACKEND`
- `FORGE_MODEL_DEFAULT_ID`
- `FORGE_LLAMA_CPP_ENDPOINT`
- `FORGE_LLAMA_CPP_BINARY_PATH`
- `FORGE_ALLOW_LLAMA_CPP_SPAWN` default `false`
- `FORGE_MODEL_OPENAI_COMPAT_ENDPOINT`
- `FORGE_MODEL_OPENAI_COMPAT_API_KEY`
- `FORGE_VLLM_BASE_URL`
- `FORGE_VLLM_API_KEY`
- `FORGE_MODEL_VLLM_ENDPOINT` legacy alias
- `FORGE_MODEL_VLLM_API_KEY` legacy alias
- `FORGE_MODEL_MAX_PROMPT_TOKENS` default `8192`
- `FORGE_MODEL_MAX_OUTPUT_TOKENS` default `1024`
- `FORGE_MODEL_MAX_RESPONSE_BYTES` default `262144`
- `FORGE_MODEL_REQUEST_TIMEOUT_MS` default `30000`
- `FORGE_MODEL_LOAD_TIMEOUT_MS` default `120000`
- `FORGE_MODEL_UNLOAD_TIMEOUT_MS` default `30000`
- `FORGE_MODEL_IDLE_UNLOAD_MS` default `0`
- `FORGE_MODEL_MAX_LOADED_MODELS` default `1`
- `FORGE_MODEL_SCHEDULER_MAX_CONCURRENT` default `1`
- `FORGE_MODEL_SCHEDULER_QUEUE_CAPACITY` default `8`
- `FORGE_MODEL_SCHEDULER_DISPATCH_TIMEOUT_MS` default `5000`
- `FORGE_MODEL_POLICY_REQUIRE_EXPLICIT_LOAD` default `true`
- `FORGE_MODEL_POLICY_ALLOW_AUTO_LOAD` default `false`
- `FORGE_MODEL_POLICY_ALLOW_CROSS_WORKSPACE` default `false`
- `FORGE_MODEL_POLICY_REQUIRE_WORKSPACE_SCOPE` default `true`
- `FORGE_ENABLE_OPENAI_COMPAT_API` default `false`

Safe default posture:

- runtime disabled unless explicitly enabled
- missing backend/endpoint yields structured unavailable errors
- missing model home or manifests do not crash startup
- spawn remains disabled by default; streaming is available only on enabled runtime paths with a streaming-capable backend

## Real / Partial / Deferred

| Area | Status | Notes |
|---|---|---|
| Import/register local GGUF and manifest-backed directories | real | Local-only path is implemented. |
| Verify / enable / disable / archive / remove-registration | real | File deletion remains separate and deferred. |
| Preferred/default model selection | real | Deterministic within current runtime scope. |
| OpenAI-compatible backend adapter | real | Endpoint-backed path is implemented. |
| vLLM-compatible backend path | partial/live external profile | Uses the OpenAI-compatible transport shape, is disabled when unset, and exposes backend status through modelruntime. No separate managed vLLM service or deep vLLM orchestration is added. |
| Streaming chat responses | real | `/forge/models/{id}/chat` and gated `/v1/chat/completions` support SSE when the runtime service/backend supports token streaming. |
| llama.cpp spawn mode | deferred | Spawn flag exists; runtime returns structured unsupported behavior. |
| Destructive model file deletion | deferred | Approval-required design target; not implemented. |
| Embeddings/rerank/vision runtime paths | deferred | Taxonomy remains documented; execution not implemented. |
| Gateway `model.*` capability registration | partial | Runtime APIs are real; gateway registry aliasing remains follow-up work. |
| Advanced batching/load balancing/distributed scheduling | deferred | FIFO single-active-per-backend scheduler only. |

## Remaining Modelruntime Work Beyond M4 External vLLM Profile

- streaming hardening beyond chat/SSE, including broader backend coverage and client cancellation UX
- stronger backend/process supervision for llama.cpp and remote backends
- dedicated gateway `model.*` capability aliasing
- destructive model file delete flow with explicit approval posture
- deeper multi-backend routing/load balancing beyond deterministic selection
- embeddings/rerank/vision execution paths

M4 does not make vLLM a host-managed service, does not require GPU packages for Nix evaluation, does not run `systemctl`, does not run `nixos-rebuild`, and does not let vLLM or modelruntime mutate canonical semantic memory.
