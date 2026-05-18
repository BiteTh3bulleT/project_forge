# FORGE M5A Prompt Pack — Authority Convergence + Latency Foundation

Generated: 2026-05-14
Target repo: `BiteTh3bulleT/project_forge`
Latest visible HEAD while preparing this pack: `cd1b2986a9c9d51eea9af87fd0e70789f651ee4d`

## Purpose

This prompt pack turns the deep-dive findings into an executable implementation sprint for FORGE.

The sprint target is:

**M5A — Authority Matrix + Runtime Policy Convergence + First Latency Foundation**

This is not a feature-spray phase. The goal is to make FORGE safer, more internally consistent, and faster without expanding live authority.

## Why this sprint exists

The deep dive found that FORGE is now powerful enough that the biggest risk is not missing features. The risk is drift between authority surfaces:

- Gateway capability metadata says some model actions are approval-only or deferred.
- Modelruntime has real APIs that do not always reflect the same policy story.
- Control Lane approval is still too source-label based.
- HostBridge/FORGE-H snapshots are advisory-only but can be sampled synchronously.
- The System surface needs to show current authority truth without adding mutation controls.
- Future micro-agents should accelerate proposals/caches, not become secret second authorities.

Translation: bolt the demon cage before installing more demon buttons.

## Pack layout

```text
README.md
00_MASTER_PROMPT_M5A.md
01_PHASE0_REPO_REALITY_AUDIT.md
02_PHASE1_AUTHORITY_MATRIX.md
03_PHASE2_MODELRUNTIME_GATEWAY_CONVERGENCE.md
04_PHASE3_CONTROL_LANE_APPROVAL_FINGERPRINTS.md
05_PHASE4_HOSTBRIDGE_FORGEH_SNAPSHOT_CACHE.md
06_PHASE5_SYSTEM_COCKPIT_READONLY.md
07_PHASE6_VALIDATION_CI_PERF_GATES.md
08_PHASE7_MICRO_AGENT_ACCELERATION_DESIGN.md
09_ACCEPTANCE_TESTS_AND_DOD.md
context/
  FORGE_CURRENT_TRUTH_CONTEXT.md
  DEEP_DIVE_FINDINGS_SUMMARY.md
  WHAT_NOT_TO_DO.md
prompts/
  CODEX_EXECUTION_HARNESS.md
  CURSOR_PARALLEL_THREADING_GUIDE.md
  REVIEWER_PROMPT.md
templates/
  AUTHORITY_MATRIX_SCHEMA.md
  MICRO_AGENT_CHARTER_TEMPLATE.md
  LATENCY_BASELINE_TEMPLATE.md
checklists/
  VALIDATION_COMMANDS.md
  IMPLEMENTATION_CHECKLIST.md
```

## How to use

Start with `00_MASTER_PROMPT_M5A.md`.

For a single powerful coding agent, paste the master prompt plus the context docs.

For multiple Codex/Cursor workers, use:

1. `01_PHASE0_REPO_REALITY_AUDIT.md`
2. `02_PHASE1_AUTHORITY_MATRIX.md`
3. `03_PHASE2_MODELRUNTIME_GATEWAY_CONVERGENCE.md`
4. `05_PHASE4_HOSTBRIDGE_FORGEH_SNAPSHOT_CACHE.md`

Do not run workers concurrently on the same files unless the phase prompt explicitly says it is safe.
