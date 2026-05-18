# FORGE System Cockpit

Status: PARTIAL_LIVE_READ_ONLY / OPERATOR_COCKPIT_INDEX / NO_LIVE_AUTHORITY_CHANGE

## Purpose

The System cockpit is the next evolution of the shell System surface. It gives the operator workstation status without granting direct host, modelruntime, memory, gateway, or FORGE-K authority.

The cockpit is read-only by default. Any future action controls must be routed through existing approval/gateway paths or a later governed proposal flow.

## Implemented First Slice

The desktop System page now includes a read-only Operator Cockpit Index backed by existing status and inspector surfaces. It summarizes:

- authority gates from `kernel_activation.authority_gates`,
- cases as not-live-wired/planned FORGE-K case packets,
- context bundles through the existing inspectors surface,
- proposals from FORGE-H and autonomy dry-run report posture,
- journal/replay through existing audit/journal inspector APIs,
- lymphatic reports as autonomy maintenance dry-run proposal-only reports.

The System page also renders the FORGE-K subsystem authority matrix and storage cutover readiness metadata when those fields are present in `GET /forge/system/status`.

No new route, control, simulator import, mutation authority, approval execution, cleanup execution, storage switch, or FORGE-K live authority is added by this slice.

## Planned Panels

| Panel | Data shown | Initial posture |
|---|---|---|
| Core status | core URL, health, version/build marker, last refresh, degraded reasons. | READ_ONLY / IMPLEMENTED |
| FORGE-K activation readiness | validation seams, readiness blockers, authority non-change flags. | READ_ONLY / IMPLEMENTED |
| Authority gate matrix | gateway, approvals, audit, Control Lane, modelruntime, memory, host mutation, FORGE-K authority gates. | READ_ONLY / IMPLEMENTED |
| Operator cockpit index | gates, cases, context bundles, proposals, journal/replay, lymphatic report pointers. | READ_ONLY / IMPLEMENTED_FIRST_SLICE |
| FORGE-H resource posture | RAM/swap/disk/VRAM/thermal pressure, advisory recommendations, warning counts. | READ_ONLY / ADVISORY_ONLY |
| HostBridge diagnostics summary | bounded host summary and source-error count. | READ_ONLY |
| modelruntime backend profile | selected/available profile labels, backend health, safe-mode state, queue posture. | READ_ONLY |
| GPU/VRAM posture | telemetry availability, pressure class, GPU acceleration disabled/deferred/available summary. | READ_ONLY |
| storage posture | SQLite default state, optional Postgres readiness, Qdrant shadow status, Redis ephemeral status. | READ_ONLY / IMPLEMENTED |
| Postgres/Qdrant/Redis readiness | disabled/default flags, DSN/endpoint configured state without secrets. | READ_ONLY / IMPLEMENTED_METADATA |
| Nix generation/rollback status | current known generation/ref when safely available, rollback posture. | READ_ONLY / FUTURE_DATA |
| mutation proposal queue | future Nix mutation proposal status, build evidence refs, VM evidence refs, approval state. | READ_ONLY until governed actions exist |
| approval queue | existing approval queue summary and links to governed approvals UI. | READ_ONLY summary |
| safe-mode status | current safe-mode flags and recovery profile hints. | READ_ONLY |
| recent warnings | bounded warnings from core, HostBridge, FORGE-H, modelruntime, storage. | READ_ONLY |
| last test/build status | latest known build/test evidence refs for workstation changes. | READ_ONLY / FUTURE_DATA |

## Rules

- Read-only by default.
- No restart/shutdown/rebuild buttons.
- No package-manager buttons.
- No model load/unload buttons unless a later governed proposal flow exists.
- No raw logs by default.
- No raw memory dumps.
- No raw host diagnostics.
- No direct shell commands.
- No approval execution unless routed through existing approval/gateway paths.
- Missing data must display as unavailable/not wired, not healthy.

## Data Boundary

The cockpit may display structured operator state. LLM-facing context still goes through the context compiler and cannot directly ingest raw host dumps, raw memory exports, raw vectors, raw prompts, raw completions, secrets, auth headers, cookies, or raw logs.

## Future Implementation Shape

Future implementation should keep extending the existing `GET /forge/system/status` read-only surface and existing inspector APIs before adding new routes. If fields become too broad, split internal DTO construction while preserving a single bounded shell-facing response.

Likely files:

- `services/core/internal/api/system_status.go`
- `services/core/internal/api/system_status_test.go`
- `packages/shared/src/index.ts`
- `apps/desktop/src/pages/System.tsx` or the current System surface component location
- `apps/desktop/src/lib/api.ts`

The first cockpit implementation should add display-only fields for backend profile labels and safe-mode posture. Nix mutation proposal queue display should wait until durable proposal records exist.
