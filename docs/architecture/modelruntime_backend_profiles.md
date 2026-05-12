# Modelruntime Backend Profiles

Status: DESIGN_ONLY / PARTIAL_LIVE_STATUS / DISABLED_BY_DEFAULT / NO_LIVE_AUTHORITY_CHANGE

## Purpose

Backend profiles name operator-facing runtime postures. They help the System cockpit and future policy surfaces explain how modelruntime should behave under normal, safe-mode, local, remote, embedding, and GPU-heavy configurations.

Profiles do not load models, unload models, spawn services, mutate host state, write semantic memory, or bypass modelruntime governance.

## Profile Matrix

| Profile | Purpose | Expected endpoint/env vars | GPU needs | VRAM posture | Concurrency posture | Allowed workload classes | Safe-mode behavior | Failure behavior | FORGE-H recommendation inputs |
|---|---|---|---|---|---|---|---|---|---|
| `cpu_safe` | CPU-only fallback for recovery, diagnostics, and degraded local work. | `FORGE_SAFE_MODE_FORCE_CPU_ONLY=true`; optional llama.cpp CPU endpoint. | None. | No VRAM required. | Low concurrency, conservative queue depth. | Small chat/completion, operator diagnostics. | Preferred profile. | Reports degraded/unavailable honestly when no CPU backend exists. | RAM, swap, disk, queue depth, safe-mode flag. |
| `local_llama_cpp` | Local GGUF runtime through governed modelruntime. | `FORGE_LLAMA_CPP_ENDPOINT`; optional `FORGE_LLAMA_CPP_BINARY_PATH` only when `FORGE_ALLOW_LLAMA_CPP_SPAWN=true`. | Optional future GPU layers. | Must tolerate CPU-only mode. | Bounded by scheduler and one-active-per-backend rule. | Chat/completion, limited local development. | Endpoint mode may remain if CPU-capable; GPU-dependent variants are disabled/deferred. | Backend unhealthy, unsupported spawn, or policy denial is explicit. | RAM, disk, thermal, GPU availability when GPU layers are requested. |
| `local_ollama_dev` | Local development compatibility with Ollama-style endpoints. | `OLLAMA_BASE_URL`, `OLLAMA_MODEL`, dev launcher autodetect unless disabled. | Depends on the local Ollama deployment. | External to FORGE; report as unknown/unavailable unless telemetry exists. | Development posture, not production scheduler foundation. | Dev chat and compatibility testing. | Disabled by explicit operator env choices; should not be required for recovery. | Endpoint/provider unavailable; cloud models excluded unless explicitly allowed. | Endpoint reachability, operator env choices, RAM/VRAM when visible. |
| `interactive_vllm` | High-throughput vLLM-compatible inference through OpenAI-compatible transport. | `FORGE_VLLM_BASE_URL`, `FORGE_VLLM_API_KEY`; legacy `FORGE_MODEL_VLLM_ENDPOINT`, `FORGE_MODEL_VLLM_API_KEY`. | Usually GPU required. | VRAM-heavy; requires headroom and pressure checks before background work. | Interactive priority; batching and deeper supervision remain future work. | Interactive chat/completion through governed modelruntime. | Disabled/deferred when CPU-only safe mode is active. | Endpoint unavailable/degraded; no FORGE host process control in M5. | VRAM pressure, thermal pressure, GPU availability, interactive queue, safe-mode flag. |
| `embedding_tei` | Hugging Face TEI embedding provider. | `FORGE_EMBEDDING_TEI_ENDPOINT`, `FORGE_EMBEDDING_TEI_API_KEY`, `FORGE_EMBEDDING_PROVIDER=tei`. | Optional depending on TEI deployment. | External or local provider-specific; not canonical. | Background embedding should defer under pressure. | Embeddings only. | Fall back to configured local hash or disabled embedding posture. | Provider unavailable warning; retrieval must degrade safely. | RAM, VRAM, background cooldown, retrieval queue, storage readiness. |
| `openai_compatible_remote` | Remote OpenAI-compatible inference endpoint. | `FORGE_MODEL_OPENAI_COMPAT_ENDPOINT`, `FORGE_MODEL_OPENAI_COMPAT_API_KEY`. | Externalized. | Not local VRAM authority. | Governed by scheduler, policy, timeout, and response-size limits. | Chat/completion via modelruntime. | May be disabled by policy for local-only recovery. | Endpoint/provider error; no local host mutation. | Network posture, provider cooldown, policy flags, request queue. |

## Pure Label Scaffolding

M5 adds pure backend profile labels in `services/core/internal/modelruntime/profiles.go`:

- `cpu_safe`
- `local_llama_cpp`
- `local_ollama_dev`
- `interactive_vllm`
- `embedding_tei`
- `openai_compatible_remote`

The labels are non-wired contracts only. They do not alter backend selection, lifecycle, scheduler behavior, load/unload behavior, API routes, or shell behavior.

## vLLM Boundary

vLLM is a preferred high-throughput backend profile, not the only modelruntime foundation. llama.cpp, Ollama development compatibility, OpenAI-compatible remote endpoints, and TEI embeddings remain valid profiles.

M5 does not install, start, stop, supervise, or configure vLLM as a NixOS service. Any future managed vLLM service needs an explicit proposal/build/test/rollback phase.

## Authority Boundary

Profiles are descriptive. The live modelruntime service remains the only model execution authority. Model outputs remain evidence/proposals and cannot directly mutate canonical truth.

Forbidden from backend profiles:

- direct host mutation;
- direct model load/unload outside existing modelruntime governance;
- direct shell controls;
- bypassing gateway, approvals, lanes, audit, or Control Lane validation;
- FORGE-K live authority expansion;
- treating GPU or remote-provider state as canonical memory.
