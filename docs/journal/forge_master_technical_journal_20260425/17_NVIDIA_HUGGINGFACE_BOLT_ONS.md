# NVIDIA / Hugging Face Bolt-Ons

Optional providers are accelerators and diagnostics only. They must not gain truth authority.

| Bolt-On | Fit | Status | Authority Boundary |
|---|---|---|---|
| NVIDIA DCGM telemetry | GPU memory/health diagnostics | PARTIAL / optional config | Diagnostics only |
| Intel Level Zero telemetry | Intel GPU diagnostics | PARTIAL / optional config | Diagnostics only |
| Hugging Face TEI | Embedding provider | PARTIAL / optional config | Retrieval support only |
| vLLM | Compatible backend path | PARTIAL | Modelruntime only |
| HF PEFT | Future adapter/LoRA work | PLANNED | Modelruntime-managed, no truth authority |
| NeMo Guardrails | Optional policy layer | CONCEPT | Advisory unless encoded in deterministic policy |
| TensorRT-LLM / NIM | Future acceleration | CONCEPT | Modelruntime backend only |
| Riva | Future voice surface | CONCEPT | Operator I/O only |
| cuVS | Future vector acceleration | CONCEPT | Retrieval index only |

## Security Notes

Provider endpoints require SSRF, allowlist, timeout, retry, cooldown, and secret-redaction hardening before broader exposure.

