# Nix, Rust, and vLLM Runtime Direction

Status: PARTIAL / DISABLED_BY_DEFAULT / NO_LIVE_AUTHORITY_CHANGE

M4 closes the external vLLM runtime profile as a governed modelruntime backend profile, not as a host-managed service. FORGE may talk to a vLLM server through the existing OpenAI-compatible transport, but FORGE does not install vLLM, start vLLM, stop vLLM, load models into vLLM outside modelruntime governance, rebuild NixOS, or mutate the host.

## Implemented M4 Boundary

- vLLM uses the existing modelruntime backend kind `vllm`.
- The backend transport is the OpenAI-compatible `/v1/models` and `/v1/chat/completions` shape.
- Canonical M4 configuration is `FORGE_VLLM_BASE_URL` and `FORGE_VLLM_API_KEY`.
- Legacy aliases `FORGE_MODEL_VLLM_ENDPOINT` and `FORGE_MODEL_VLLM_API_KEY` remain accepted for compatibility.
- A configured vLLM endpoint auto-enables modelruntime the same way the OpenAI-compatible endpoint does.
- Modelruntime backend status exposes the `interactive_vllm` profile when that backend is present.
- The desktop Models page displays backend status through modelruntime status APIs only.

## Disabled By Default

vLLM remains disabled unless the operator provides a vLLM endpoint. The Nix flake and NixOS modules must evaluate without vLLM, CUDA, Python GPU packages, or a local GPU.

## Nix Direction

Nix is the reproducible substrate for FORGE packaging and future workstation profiles. M4 does not add a NixOS service that runs vLLM. A future NixOS module may define an opt-in vLLM service profile, but that module must stay disabled by default and must not be triggered by the graphical shell.

## Rust Direction

Rust is acceptable for future runtime drivers, high-throughput host adapters, and bounded systems components where it improves safety or performance. Rust components must remain drivers or workers. They may not own semantic truth, journal authority, approvals, Control Lane authority, or FORGE-K live authority.

## Authority Boundary

Model output remains evidence. vLLM is an inference accelerator/backend only. It cannot write canonical memory, bypass the gateway, bypass modelruntime governance, issue host commands, mutate NixOS, or become FORGE-K live authority.

## Future Work

- Managed vLLM process/service profile under NixOS, disabled by default.
- GPU/VRAM-aware vLLM scheduling recommendations through FORGE-H.
- Operator-visible backend profile readiness and fallback posture in the system cockpit.
- Streaming and deeper process supervision remain broader modelruntime work items.
