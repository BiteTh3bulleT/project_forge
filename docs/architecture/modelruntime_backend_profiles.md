# Modelruntime Backend Profiles

Status: DESIGN_ONLY / PARTIAL_LIVE_STATUS / DISABLED_BY_DEFAULT

## Profiles

| Profile | Purpose | Endpoint/env | GPU/VRAM posture | Workloads | Safe-mode behavior | Failure behavior |
|---|---|---|---|---|---|---|
| `cpu_safe` | CPU-only fallback and recovery inference | local llama.cpp or no backend | no GPU required | small chat, diagnostics | remains allowed | degraded/unavailable is honest |
| `local_llama_cpp` | local GGUF runtime | `FORGE_LLAMA_CPP_ENDPOINT`, optional binary path | optional GPU layers later | chat/completion | endpoint mode only unless explicitly allowed | backend unhealthy, no spawn by default |
| `local_ollama_dev` | local dev compatibility | `OLLAMA_BASE_URL` autodetect in dev launcher | depends on Ollama | dev chat | disabled by explicit operator env choices | cloud models excluded unless allowed |
| `interactive_vllm` | high-throughput local/remote OpenAI-compatible inference | `FORGE_VLLM_BASE_URL`, `FORGE_VLLM_API_KEY` | usually GPU/VRAM-heavy | interactive chat/completion | disabled when CPU-only safe mode requires it | endpoint unavailable/degraded |
| `embedding_tei` | Hugging Face TEI embedding provider | `FORGE_EMBEDDING_TEI_ENDPOINT`, `FORGE_EMBEDDING_TEI_API_KEY` | optional GPU | embeddings only | fall back to local hash when configured | provider unavailable warning |
| `openai_compatible_remote` | remote OpenAI-compatible inference | `FORGE_MODEL_OPENAI_COMPAT_ENDPOINT`, API key | externalized | chat/completion | may be disabled by policy | endpoint/provider error |

## FORGE-H Inputs

Backend profile recommendations should consider RAM pressure, swap pressure, disk pressure, VRAM pressure, thermal pressure, GPU availability, queue depth, and safe-mode flags.

## Authority Boundary

Profiles describe runtime posture. They do not load models, unload models, mutate host services, write memory, bypass modelruntime governance, or make backend output canonical truth.
