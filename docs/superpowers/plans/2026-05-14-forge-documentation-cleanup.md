# FORGE Documentation Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clean the FORGE documentation set so current authority, historical evidence, and next work are clearly separated before further FORGE-K live integration.

**Architecture:** Keep a small authoritative doc set and mark older phase artifacts as historical. Update FORGE-K boundary docs to reflect the current split: simulator implemented/tested, Phase 12 shadow diagnostics read-only, Phase 14 Control Lane validation-only, and live authority gates still blocked.

**Tech Stack:** Markdown documentation, local grep/link checks, repository validation scripts.

---

### Task 1: Current Authority Docs

**Files:**
- Modify: `AGENTS.md`
- Modify: `README.md`
- Modify: `docs/status/implementation_matrix.md`
- Modify: `docs/reports/FORGE_PUNCHLIST.md`
- Create: `docs/status/current_authority_sources.md`
- Include: `docs/roadmap/forge_mutation_loop_parking_lot.md`

- [x] Add `docs/status/current_authority_sources.md` listing the current authoritative docs and historical-doc rule.
- [x] Shorten README FORGE-K phase prose and point to `docs/reviews/current_phase_status.md` plus `docs/status/current_authority_sources.md`.
- [x] Refresh AGENTS status guidance to name `docs/reviews/current_phase_status.md` as FORGE-K phase authority and include the branch/worktree policy.
- [x] Retitle `implementation_matrix.md` as current live AI-OS implementation status instead of a stale Phase 5.996 snapshot.
- [x] Reconcile `FORGE_PUNCHLIST.md` modelruntime rows: SSE streaming, vLLM-compatible external profile, and delete-file approval are done; remaining work is hardening/supervision.

### Task 2: FORGE-K Boundary Docs

**Files:**
- Modify: `docs/architecture/kernel_simulator.md`
- Modify: `docs/architecture/forge_k_integration_readiness.md`
- Modify: `docs/architecture/shadow_mode_harness.md`
- Modify: `docs/architecture/forge_k_live_integration_design.md`
- Modify: `docs/architecture/simulator_to_live_migration.md`
- Modify: `docs/architecture/forge_k_overview.md`
- Modify: `docs/adr/0005-forge-k-simulator-vs-live-authority.md`
- Modify: `docs/adr/0006-rust-kernel-core-boundary.md`

- [x] Add current-status banners where docs are historical design artifacts.
- [x] Update live-boundary text to distinguish simulator services, read-only shadow diagnostics, validation-only Control Lane seams, and blocked live authority gates.
- [x] Keep ADR decisions intact; add short amendment notes rather than rewriting history.
- [x] Make `simulator_to_live_migration.md` the current migration-pattern doc with a concise authority-gate summary.

### Task 3: Operational And Archive Hygiene

**Files:**
- Modify: `docs/runbooks/current_forge_bringup.md`
- Modify: `docs/runbooks/config_reference.md`
- Modify: `docs/runbooks/docker_containerization.md`
- Modify: `docs/roadmap/forge_k_build_phases.md`
- Create: `docs/testing/phase_14_control_lane_validation.md`
- Optionally add banners to historical status/review docs instead of moving large archive sets in this pass.

- [x] Fix broken runbook/config relative links.
- [x] Prefer root `npm run ...` commands over platform-specific script paths in operator docs.
- [x] Add Phase 14 testing index for validation/no-authority-expansion evidence.
- [x] Update `forge_k_build_phases.md` to delegate current phase truth after Phase 14D to `current_phase_status.md` or add 14E-14M summary rows.
- [x] Add historical banners to stale blocker/status docs that are superseded by current authority docs.

### Verification

- [x] Run `git diff --check`.
- [x] Run a markdown link/path sanity check for changed docs.
- [x] Run `npm test` because root docs and status guidance changed.
- [x] Commit and push `main` after verification.
