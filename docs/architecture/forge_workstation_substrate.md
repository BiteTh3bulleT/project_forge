# FORGE Workstation Substrate

Status: DESIGN_ONLY / PLANNED / NO_LIVE_AUTHORITY_CHANGE

## Purpose

FORGE Workstation is a private, local-first FORGE host profile built on NixOS, the Tauri graphical shell, `forge-core`, modelruntime, FORGE-H, HostBridge diagnostics, and governed storage/runtime infrastructure.

It is not a Linux fork and not an autonomous host-control system. Linux remains the hardware/kernel substrate. NixOS remains the declarative host substrate. FORGE is the AI-OS operating environment above that substrate.

## Stack

| Layer | Responsibility | Authority Boundary |
|---|---|---|
| Hardware | CPU, RAM, disk, network, GPU, firmware, sensors. | Physical substrate only; no semantic truth authority. |
| Linux kernel | Boot, drivers, cgroups, filesystems, networking, process isolation. | Kernel/driver authority belongs to Linux, not FORGE. |
| NixOS | Declarative host config, packages, users, services, generations, rollback. | Host mutation requires operator-controlled NixOS workflows or a future governed adapter. |
| FORGE Workstation profile | Opt-in composition for `/forge`, core service posture, shell session, runtime/storage dependencies, recovery paths. | Disabled by default until explicitly imported/enabled. |
| `forge-core` | Gateway, approvals, audit, lanes, memory, jobs, retrieval, modelruntime API composition, Control Lane validation. | CPU/RAM authoritative FORGE daemon; canonical mutation still uses governed commit paths. |
| HostBridge | Read-only host diagnostics snapshots. | Diagnostic evidence only; no commands or host mutation from HostBridge consumers. |
| FORGE-H | Resource posture, recommendations, advisory proposals, bounded internal operational policy records. | Advisory/resource governance only; no raw host mutation. |
| modelruntime | Governed model registry, backend selection, scheduler, runtime policy, request/audit surfaces. | Models are drivers; backend output is proposal/evidence, not truth. |
| storage/vector/ephemeral services | SQLite live store, optional Postgres foundation, disabled Qdrant shadow index, disabled Redis ephemeral coordination. | SQLite remains live default; Qdrant and Redis are not canonical truth. |
| Tauri graphical shell | Operator shell, cockpit, approvals/status surfaces, workspace UI. | Operator surface only; no direct host commands or authority bypass. |
| operator | Human review, approval, rollback, recovery, and final host-administration judgment. | Explicit operator approval is required before high-risk or host mutation actions. |

## Non-Goals

- FORGE does not fork Linux.
- FORGE does not replace the Linux kernel.
- FORGE does not auto-run `nixos-rebuild`.
- FORGE does not mutate host config without an operator-approved proposal flow.
- FORGE-K does not become live authority in this phase.
- GPU does not become truth authority.
- The shell UI does not run host commands directly.
- vLLM does not become the only modelruntime foundation.
- SQLite is not replaced as the default live canonical store.
- Qdrant is not treated as truth.
- Redis is not treated as canonical state.

## Workstation Composition

The intended opt-in workstation profile composes:

- `/forge` durable root with explicit data, state, logs, cache, backups, imports, models, runtime, and journal-oriented directories;
- `forge-core` as the local FORGE daemon;
- Tauri shell and `forge-shell-session`/`forge-desktop-shell` launch path;
- Wayland shell session path through a lightweight compositor such as Cage;
- optional Postgres for future parity-gated storage work;
- optional Qdrant for disabled/non-authoritative vector shadow infrastructure;
- optional Redis for disabled/non-canonical ephemeral coordination;
- modelruntime backend profile selection;
- optional GPU telemetry, not required for canonical operation;
- safe-mode/recovery profile;
- backup/export directory expectations under `/forge/backups` and `/forge/data/exports`.

Every component must be independently removable or disableable without deleting canonical data.

## Promotion Doctrine

Workstation capabilities move through this ladder:

```text
DESIGN_ONLY
  -> SIMULATOR_ONLY
  -> SHADOW_READ_ONLY
  -> DISABLED_BY_DEFAULT_LIVE
  -> OPERATOR_APPROVED_LIVE
  -> AUTOMATED_SAFE_ACTION
```

Nothing jumps stages. Any future live host mutation requires build proof, test proof, rollback proof, approval proof, and audit/journal evidence before apply.

## Build, Test, And Rollback Doctrine

Before a workstation substrate change can be promoted:

- it must identify the affected Nix files and generated host services;
- it must build in a sandboxed or non-live target;
- it must pass a VM smoke test when service/session behavior changes;
- it must record the current rollback generation or equivalent rollback path;
- it must keep TTY fallback and non-FORGE desktop fallback available;
- it must expose a clear operator command path for recovery.

No UI/model/FORGE-K/FORGE-H path may apply a host mutation directly.
