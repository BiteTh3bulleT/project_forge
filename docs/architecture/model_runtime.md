# FORGE Model Runtime Architecture

Status date: 2026-05-13 (M3 plus M4 external vLLM profile, streaming, and delete-file snapshot).

## Intent

FORGE owns model runtime governance even when inference kernels are delegated to pluggable backends.

FORGE-owned responsibilities:

- model manifest format and local store layout
- registry, lifecycle state, and management workflows
- runtime scheduler and request admission
- API boundary (`/forge/models*`, gated `/v1/*`)
- policy, correlation, audit, and usage accounting
- workspace and request metadata association
- backend selection and deterministic request routing

Backends remain inference engines only. Model output is evidence, not canonical truth.

## Relationship to Ollama

FORGE model runtime does not require Ollama.

The runtime service is a native FORGE subsystem:

- internal FORGE API: `/forge/models*`
- optional OpenAI-compatible minimum API: `/v1/models`, `/v1/chat/completions`
- first local-runtime backend: llama.cpp HTTP adapter
- additional remote/runtime adapters: OpenAI-compatible endpoint backend, vLLM-compatible path

Ollama compatibility is not the authority for local model execution in this subsystem.

## Core Components

### Domain, Manifest, Store, Registry, State

Implemented in `services/core/internal/modelruntime/`:

- `types.go`: model enums and request/result contracts
- `manifest.go`: `model.forge.json` parsing and validation
- `state.go`: persistent `model.state.json` lifecycle metadata
- `store.go`: local model home scanning and checksum-aware loading
- `store_management.go`: import, verify, archive, remove-registration, and reconcile helpers
- `registry.go`: registry state and model inspection helpers
- `management.go`: service-level management, compatibility, backend, and usage reports

Layout:

- `$FORGE_MODEL_HOME/models/<model-id>/model.forge.json`
- `$FORGE_MODEL_HOME/models/<model-id>/model.state.json`
- `$FORGE_MODEL_HOME/archives/<model-id>/...` for archived managed models

### Manifest and Import Flow

M3 adds managed import/register workflows.

Supported local paths:

- import a local GGUF file and generate a managed manifest
- import a directory that already contains `model.forge.json`
- rescan model home and reconcile registry state with on-disk manifests/files

Imported manifests capture, where available:

- stable model id
- display metadata
- source path
- file size
- format
- backend
- checksum
- capabilities
- runtime defaults
- license metadata

Duplicate imports and invalid paths fail deterministically.

### Backend Interface

`backend.go` defines the pluggable backend boundary:

- `Name()`
- `Kind()`
- `Supports()`
- `Load()`
- `Unload()`
- `Generate()`
- `Health()`
- `Inspect()`

Backends do inference only. They do not mutate canonical memory or bypass audit/policy.

### Backends

Implemented backends:

- `backend_fake.go`: deterministic test backend
- `backend_llama_cpp.go`: llama.cpp HTTP endpoint adapter
- `backend_openai_compat.go`: OpenAI-compatible endpoint adapter, also used for vLLM-compatible endpoint wiring

Current backend posture:

- llama.cpp endpoint mode is supported
- OpenAI-compatible remote inference is supported behind runtime policy and config
- vLLM-compatible endpoint use is supported through the same transport shape as the disabled-by-default `interactive_vllm` profile, without deep vLLM-specific orchestration
- spawn/process management remains explicitly out of current scope except for structured unsupported/error behavior

### vLLM Profile Boundary

M4 treats vLLM as an external governed backend endpoint. Operators configure it with `FORGE_VLLM_BASE_URL` and optional `FORGE_VLLM_API_KEY`; legacy `FORGE_MODEL_VLLM_*` aliases remain supported. FORGE does not install, start, stop, or supervise vLLM in this phase, and Nix evaluation must not require vLLM, CUDA, or GPU hardware.

The profile is visible through modelruntime health/status APIs. The desktop may display that state, but it must not expose model load/unload or host mutation controls outside the existing governed modelruntime surfaces.

## Runtime Service

`modelruntime.Service` is the authority for model-runtime execution and management.

Responsibilities:

- list and inspect models
- import and reconcile managed models
- explicit load and unload
- verify, enable, disable, archive, remove registration, and approval-required managed file deletion
- generate via scheduler-controlled execution
- resolve backend/model selection deterministically
- expose loaded-model, queue, usage, backend, and health snapshots
- emit audit records with request metadata and usage fields
- consume optional GPU telemetry for diagnostics and background workload admission

### Lifecycle

Current lifecycle states include:

- `available`
- `imported`
- `verified`
- `loading`
- `loaded`
- `unloading`
- `unavailable`
- `disabled`
- `archived`
- `error`

Lifecycle rules:

- import/register does not imply automatic load
- explicit load and unload remain first-class operations
- duplicate loads are idempotent when the same model is already loaded
- disabled or archived models are denied for inference
- remove-registration and archive are metadata/governance operations, not destructive deletion
- file deletion is a separate approval-required operation and is limited to managed active/archive model directories under `FORGE_MODEL_HOME`

## Scheduler and Admission Control

The runtime uses a simple deterministic scheduler.

Properties:

- FIFO request queue
- bounded queue depth
- bounded concurrent requests
- one active generation at a time per backend
- visible queued/running/completed request state through scheduler snapshots
- request cancellation and timeout behavior via request context/timeouts
- explicit rejection when queue capacity is exhausted

M3 keeps this intentionally simple. It is not a batching or distributed scheduler.

GPU policy classes are explicit:

- `INTERACTIVE_INFERENCE`
- `INTERACTIVE_EMBEDDING`
- `BACKGROUND_EMBEDDING`
- `BACKGROUND_RERANK`
- `DREAM_DISTILLATION`
- `ADAPTER_EVAL`
- `ADAPTER_TRAINING`

Default policy posture is conservative:

- interactive classes have priority over background classes
- background classes are bounded by queue and background concurrency caps
- background classes defer under interactive load/cooldown windows
- Dream GPU classes are optional and default-off unless enabled by policy

## Backend Selection and Routing

M3 adds stronger, still-bounded selection logic.

Selection order:

1. explicit request model id
2. explicit backend override if compatible with the selected model
3. configured default model id / preferred model state
4. deterministic unambiguous compatible candidate

Rules:

- unsupported capability/model/backend combinations fail explicitly
- incompatible backend overrides fail explicitly
- the runtime does not silently choose surprising fallbacks when multiple candidates are ambiguous

## Runtime Limits

Enforced runtime controls include:

- prompt token estimate bound
- max output tokens
- max output bytes
- request timeout
- streaming requires an explicit token handler and a streaming-capable backend
- queue depth
- max concurrent requests
- one-active-per-backend execution rule
- max loaded models

When output is bounded, truncation is explicit through finish reason and warnings.

## Policy and Workspace Hooks

The API bridge installs request-policy hooks into the runtime service.

Current policy posture:

- actor is required for inference and management requests
- source is required for inference and management requests
- workspace scope can be required by config
- cross-workspace scope can be denied by config
- unsupported model capabilities are denied deterministically
- disabled or archived models cannot be used for inference
- unsupported backend overrides are denied deterministically
- self-initiated/autonomy-style requests are denied by default in the current hook set
- runtime-disabled posture remains default-safe

This leaves room for future charter/budget/autonomy policy without creating a second inference path.

## API Surface

### Internal FORGE API

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
- `POST /forge/models/{id}/delete-file`
- `POST /forge/models/{id}/load`
- `POST /forge/models/{id}/unload`
- `POST /forge/models/{id}/chat`
- `GET /forge/model-runtime/backends`
- `GET /forge/model-runtime/usage`
- `GET /forge/model-runtime/queue`
- `GET /forge/model-runtime/loaded`
- `GET /forge/model-runtime/health`

### OpenAI-Compatible Minimum

- `GET /v1/models`
- `POST /v1/chat/completions`

Streaming chat is supported on both public chat surfaces when the runtime service and selected backend support token streaming:

- `/forge/models/{id}/chat` with `stream: true` emits FORGE SSE events: `token`, `result`, and `done`.
- `/v1/chat/completions` with `stream: true` emits OpenAI-compatible `chat.completion.chunk` SSE data followed by `data: [DONE]`.

Streaming stays inside the same modelruntime boundary as non-streaming chat: scheduler admission, policy hooks, backend selection, correlation/workspace metadata, audit, and output evidence semantics remain unchanged. A runtime/backend without token streaming returns structured `STREAM_UNSUPPORTED` behavior.

## Compatibility, Usage, and Health Inspection

M3 adds inspection views for:

- model compatibility
- backend status
- runtime usage summary
- queue state
- loaded models
- runtime health

Status endpoints report actual runtime state. They do not synthesize unsupported metrics.

Runtime health state values:

- `available`
- `degraded`
- `unavailable`
- `cooldown`
- `overloaded`

Health surfaces expose degraded reasons and policy warnings so operators can distinguish:

- CPU-safe operation with accelerator unavailable
- background deferrals due to interactive priority/cooldown
- backend health failures

## Audit and Evidence

Runtime audit records include, where available:

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

## Governance Boundary

Model runtime management is governed inside FORGE, but this phase does not create a new autonomy side door.

Still deferred or intentionally bounded:

- destructive model file deletion is implemented only for managed model-home directories and always requires high-risk model-management approval
- streaming hardening beyond governed chat/SSE paths
- advanced multi-backend load balancing
- distributed scheduling
- embeddings/rerank/vision execution paths
- deep vLLM integration
- llama.cpp spawn/process supervision
- gateway `model.*` capability aliasing
- autonomy charter/budget-aware inference governance beyond current policy hooks

## Safe mode

Safe mode (`FORGE_SAFE_MODE_FORCE_CPU_ONLY=true`) enforces CPU-only operation while preserving runtime governance APIs.

Effects:

- kernel authority remains CPU/RAM-only and unchanged
- modelruntime remains callable but reports degraded/unavailable accelerator posture as appropriate
- GPU-requiring requests fail deterministically with policy/runtime errors
- background accelerator classes are disabled/deferred by default policy
