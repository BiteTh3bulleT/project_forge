# FORGE — Living Project Context

This file is the authoritative local context source for FORGE normalization.

## What FORGE is

FORGE is a **local-first operator workspace** for controlled AI-assisted execution.
It combines memory retrieval, packet contracts, approval gates, adapter workers, and durable artifact evidence.

## Current phase

The codebase is **cumulative**: Phases 1–5 layers coexist. The latest hardening pass is **Phase 5**.

**Phase 5 (latest)** adds:

- Central **tool execution gateway** (bounded tools only; no hidden execution path for the same class of work)
- **Action lanes** (`action_lanes`) describing scoped operations (paths, write intent, risk, approval flags)
- **Execution permission profiles** (`permission_profiles`) — separate from routing **Policy** — gating read/write/execute paths and allowed tool ids
- **Audit records** (`audit_records`) with **correlation ids** + trace API
- **Backup / export / import** bundles under `${FORGE_DATA_DIR}` (`backups/`, `exports/`)
- **Release readiness** checklist + first-run summary + `release_artifacts` bookkeeping
- Desktop pages: `/gateway`, `/action-lanes`, `/execution-permissions`, `/audit`, `/backup`, `/release`

Earlier phases (1–4) remain in force for jobs, approvals, packets, dossiers, retrieval, policy, automation, reviews, reconciliation, and dashboard surfaces — see `docs/` scope files.

## Execution doctrine

- jobs are projections
- events are truth
- packets are contracts
- approvals are gates
- artifacts are evidence
- adapters are bounded workers

## AI-OS framing

FORGE doctrine is now promoted into an AI-OS framing:

- events are the system journal
- jobs are process projections
- packets are execution contracts
- approvals are gates
- artifacts are evidence
- adapters are bounded workers
- gateway/action lanes/permissions/audit are kernel-like controls

Target operating model is tri-lane:

- Control Lane
- I/O Lane
- Compute Lane

Semantic services (including future IRIS) may propose actions, but FORGE retains validation and commit authority over canonical state.

## Boundaries

FORGE must keep these categories explicit and non-escalating:

- memory retrieval
- reasoning
- write proposal
- write execution
- command execution

No silent escalation across categories.

## Adapter contract requirements

Every adapter request includes:

- adapter id
- capability
- scope (`allowedPaths`, `forbiddenPaths`, `selectedPaths`)
- write intent
- task packet reference
- timeout
- dry-run flag
- correlation id

## Failure taxonomy

- validation
- approval denied
- adapter unavailable
- adapter timeout
- packet build failure
- persistence failure
- user cancellation
- execution failure

## Operational notes

- Core API default: `http://127.0.0.1:18492`
- Data dir default: `${XDG_CONFIG_HOME}/forge` unless `FORGE_DATA_DIR` is set
- Workspace root can be overridden with `FORGE_WORKSPACE_DIR`
- Project context normalization archives raw imports under `${FORGE_DATA_DIR}/project_context/imports`

---

### Changelog

- **2026-04-15**: Phase 1 baseline landed.
- **2026-04-15**: Phase 2 execution/approval/packet/context systems landed.
- **2026-04-15**: Phase 5 gateway, lanes, execution permissions, audit/trace, backup bundles, release readiness, and operator UI wiring landed (`docs/PHASE5_OPERATIONS.md`).
