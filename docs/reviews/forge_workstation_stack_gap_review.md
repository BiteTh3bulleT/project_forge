# FORGE Workstation Stack Gap Review

Status: DESIGN_ONLY / READINESS_REVIEW / NO_LIVE_AUTHORITY_CHANGE

Date: 2026-05-12

## Scope

This review starts Phase M5. It evaluates the workstation target needed for a real FORGE desktop on NixOS with governed modelruntime profiles, FORGE-H resource posture, and future Nix mutation proposals.

## Current Reality

- Nix packaging exists for core desktop shell paths and opt-in graphical sessions.
- FORGE can run as a desktop shell surface in VM/manual paths.
- Modelruntime supports llama.cpp, Ollama-compatible, OpenAI-compatible, and vLLM-compatible endpoint profiles.
- HostBridge and FORGE-H expose bounded diagnostics and advisory resource posture.
- FORGE-K remains simulator/partial-validation only and is not live daemon authority.

## Gaps

| Area | Gap | Required Posture |
|---|---|---|
| NixOS workstation target | No complete operator workstation substrate design | Design first; no default replacement, no autologin, keep rollback and fallback desktop/TTY |
| Host mutation | No governed Nix proposal lifecycle | Proposal-only design; no shell commands, `systemctl`, or `nixos-rebuild` from UI/model/simulator |
| vLLM/GPU runtime | External endpoint profile exists, but no managed NixOS service profile | Keep vLLM disabled by default; future service must be governed and rollback-aware |
| VRAM/CUDA governance | GPU telemetry exists, but VRAM leases and CUDA work classes are not modeled | FORGE-H observes/recommends first; CUDA accelerates cognition but never authors truth |
| Safe mode | CPU-only posture exists, but workstation specializations are not fully designed | Define Normal/Safe/CPU/GPU/Debug/Recovery profiles with fallback paths |
| System cockpit | Read-only system surfaces exist, but full workstation cockpit is not designed | Read-only first; no mutation buttons; proposal queue only when governed |

## Optional Scaffolding Decision

M5 skips code scaffolding in this pass. The requested labels and proposal states are design-level contracts for now. Adding package-level constants would be low risk, but without a consuming boundary it would create unused API surface and invite premature wiring.

## Non-Negotiable Boundaries

- No host mutation is introduced.
- No direct `systemctl` or `nixos-rebuild` path is introduced.
- No model load/unload behavior changes are introduced.
- No direct semantic memory writes are introduced.
- No FORGE-K live authority migration is introduced.
- Qdrant and Redis remain non-canonical unless a later cutover phase proves otherwise.

## Recommendation

Proceed with M5 as a design/hardening pass. The next implementation phase should choose one narrow path: either governed Nix mutation proposal records or read-only workstation cockpit expansion, but not both in the same wiring pass.
