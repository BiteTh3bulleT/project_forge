# Modelruntime And Model Governance

## Role

IMPLEMENTED / PARTIAL: Modelruntime is FORGE's governed inference substrate. It owns model manifests, registry, lifecycle, backend selection, queueing, limits, health, usage, and audit for inference.

Evidence: `services/core/internal/modelruntime`, `services/core/internal/api/model_runtime.go`, `model_runtime_bridge.go`, and `docs/status/model_runtime_status.md`.

## Backend Support

| Backend | Status |
|---|---|
| fake | IMPLEMENTED test backend |
| llama.cpp endpoint mode | IMPLEMENTED |
| OpenAI-compatible | IMPLEMENTED M3 |
| vLLM-compatible path | PARTIAL; currently transport-compatible through OpenAI-style API |
| Ollama-compatible | DEFERRED / adapter compatibility |

## Model Roles

| Role | Examples | Status |
|---|---|---|
| classifier | routing, risk, salience | PLANNED/PARTIAL |
| planner | task decomposition | PARTIAL through chat/adapters |
| executor | code/text generation | PARTIAL through modelruntime/chat |
| verifier | review/consistency checks | PLANNED |
| summarizer | context/Dream summaries | PARTIAL |
| repair analyst | diagnostics and memory repair | PLANNED |

## Governance Gaps

- RISK: Model management actions need approval semantics equivalent to gateway dangerous tools.
- PARTIAL: Streaming remains an M4 item.
- PARTIAL: Delete-file approval flow is intentionally absent.
- PARTIAL: Stronger backend/process supervision remains M4.
- PARTIAL: Gateway `model.*` capability aliases are not complete.
- PARTIAL: NVIDIA support is telemetry/admission policy, not CUDA/HF model execution orchestration.
- PARTIAL: TEI is implemented as an embeddings provider, outside canonical truth and outside full modelruntime execution.

## Philosophy

Small models should handle bounded local tasks when competent. Larger/frontier models may amplify planning or verification, but only through governed inference paths and never as truth authority.
