# FORGE System Cockpit

Status: DESIGN_ONLY / READ_ONLY_FIRST / PLANNED

## Purpose

The system cockpit is the next evolution of shell system surfaces. It gives the operator workstation status without granting direct authority.

## Planned Panels

- Core status.
- FORGE-K activation readiness.
- Authority gate matrix.
- FORGE-H resource posture.
- HostBridge diagnostics summary.
- Modelruntime backend profile.
- GPU/VRAM posture.
- Storage posture.
- Postgres/Qdrant/Redis readiness.
- Nix generation and rollback status.
- Mutation proposal queue.
- Approval queue.
- Safe-mode status.
- Recent warnings.
- Last test/build status.

## Rules

- Read-only by default.
- No restart, shutdown, rebuild, or package-manager buttons.
- No model load/unload buttons unless a later governed proposal flow exists.
- No raw logs by default.
- No raw memory dumps.
- No direct shell commands.
- No approval execution unless routed through existing approval/gateway paths.

## Data Boundary

The cockpit may display structured operator state. LLM-facing context still goes through the context compiler and cannot directly ingest raw host dumps, raw memory exports, or raw logs.
