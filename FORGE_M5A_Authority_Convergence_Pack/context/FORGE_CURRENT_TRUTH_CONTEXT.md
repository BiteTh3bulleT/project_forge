# FORGE Current Truth Context

## Project identity

FORGE means **Foundry for Organic Reasoning, Growth, and Execution**.

Tagline: **Turn chaos into cognition. Turn cognition into action.**

FORGE is a local-first AI workspace / AI-OS operating layer for inspectable, approval-gated engineering work.

## Current authority model

1. **Canonical mutation authority**
   - Lives in `services/core/internal/aios/controllane`.
   - Durable semantic writes must pass deterministic validation, journal/commit boundaries, audit, and provenance.
   - FORGE-K simulator services are not live daemon authority.

2. **Tool execution authority**
   - Lives in `services/core/internal/gateway`.
   - Gateway-only execution is the intended governed tool path.
   - Legacy adapter invoke ingress is not authority.

3. **Model runtime authority**
   - Lives in `services/core/internal/modelruntime` and `services/core/internal/api/model_runtime*.go`.
   - Models are governed drivers, not truth authority.
   - Streaming, vLLM-compatible external endpoint support, and managed delete-file approval exist inside modelruntime boundaries.
   - Model output is evidence, not automatic memory truth.

4. **Memory and retrieval**
   - Memory and retrieval are evidence/projection systems unless committed through canonical paths.
   - Tool/model output does not become truth by existing.

5. **Approvals and audit**
   - Approval requests and approval decisions must remain distinct.
   - Audit records are durable evidence.

6. **FORGE-K boundary**
   - FORGE-K remains target cognitive microkernel architecture and simulator-first implementation.
   - Current live seams are narrow validation/enforcement surfaces only.
   - No live state mutation should route through FORGE-K simulator services in this phase.

7. **FORGE-H and HostBridge**
   - HostBridge is read-only host diagnostics.
   - FORGE-H is advisory resource policy.
   - FORGE-H may recommend; it must not mutate host/modelruntime/memory directly.

8. **Nix / workstation**
   - NixOS/Linux remains the host substrate.
   - FORGE Workstation mutation controls are future, proposal/approval/rollback-gated.
   - No direct shell-to-host mutation.
   - No `nixos-rebuild`, `systemctl`, reboot, shutdown, package-manager, or kernel/module controls from UI in this phase.

## M5A sprint thesis

Before adding more capability, make the existing capability map honest.

M5A should produce:
- a machine-readable authority matrix,
- fixed modelruntime/gateway capability drift,
- durable approval fingerprint design and first implementation seam,
- read-only System Cockpit authority display,
- HostBridge/FORGE-H snapshot caching,
- first safe background micro-agent plan,
- validation tests that prevent authority drift from returning.
