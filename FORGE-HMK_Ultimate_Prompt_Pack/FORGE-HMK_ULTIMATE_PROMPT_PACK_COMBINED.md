# FILE: README.md

# FORGE-HMK Ultimate Prompt Pack

**FORGE-HMK** = **FORGE Heterogeneous Memory Kernel**.
This pack implements FORGE-HMK as a **shadow-first memory kernel** that manages cells, synapses, temporal traces, cache manifests, memory jobs, and context artifacts without stealing canonical authority from FORGE-K / Control Lane.

## Stack

```text
FORGE-K / Control Lane = canonical authority and commit boundary
Crucible = truth refinement, contradiction validation, promotion gate
FORGE-HMK = heterogeneous memory kernel
FORGE-T = temporal tuner, scheduler, TTL/cache/retry/performance governor
Neuron Mesh = bounded worker teams for memory jobs
HKV = hierarchical cognitive/model KV cache
Photo-Kinetic Memory = snapshots + transitions + traces + replay
Rule Cells / Hyperlane = deterministic reflex layer
```

## Golden rule

**Workers propose. Crucible refines. FORGE-K commits.**

## How to run this pack

1. Start with `prompts/00_MASTER_IMPLEMENTATION_PROMPT.md`.
2. Execute phases in order.
3. Keep early implementation no-op, internal, and shadow-only.
4. Do not expose public mutation routes.
5. Do not let workers, cache, vector DB, VSA, or replay artifacts become truth.
6. Validate each phase with `validation/PHASE_EXIT_GATES.md`.

## Recommended order

1. Phase 0 — repo audit and boundary map
2. Phase 1 — contracts and no-op shells
3. Phase 2 — FORGE-T Temporal Tuner
4. Phase 3 — FORGE-HMK core
5. Phase 4 — Photo-Kinetic Memory
6. Phase 5 — HKV hierarchical cache
7. Phase 6 — Neuron Mesh workers
8. Phase 7 — Crucible validation
9. Phase 8 — Context Compiler integration
10. Phase 9 — telemetry and performance governor
11. Phase 10 — shadow parity and cutover prep

## Agent note

Do not output implementation code in chat. Write code to files. Chat summaries should list changed files, tests run, and risks.


---

# FILE: adr_prompts/ADR_CRUCIBLE_RENAME_AND_ROLE.md

# ADR Prompt: Crucible Rename and Role

Create an ADR replacing the older “Courthouse” terminology with “Crucible.”

Explain its role in truth refinement, contradiction validation, supersession checks, provenance requirements, promotion readiness, and why it is not canonical authority.


---

# FILE: adr_prompts/ADR_FORGE_HMK_SHADOW_FIRST.md

# ADR Prompt: FORGE-HMK Shadow-First Memory Kernel

Create an ADR for implementing FORGE-HMK as a shadow-first heterogeneous memory kernel.

Include: status, context, decision, consequences, authority boundaries, non-goals, migration path, rollback plan, validation requirements, and what not to do.

Decision: FORGE-HMK may assemble memory artifacts, traces, cache manifests, and claim envelopes, but canonical truth remains under FORGE-K / Control Lane until explicit live promotion.

Do not include implementation code in chat. Write the ADR file.


---

# FILE: adr_prompts/ADR_FORGE_T_TEMPORAL_TUNER.md

# ADR Prompt: FORGE-T Temporal Tuner

Create an ADR for FORGE-T as a system-level temporal tuner beside FORGE-HMK.

Explain why it sits beside FORGE-HMK, its job scheduling, cache TTL, retry/backoff, worker cadence, replay window, backpressure, performance metrics, and authority limits.

Decision: FORGE-T governs timing and performance, but cannot commit memory truth.


---

# FILE: context/ARCHITECTURE_BLUEPRINT.md

# Architecture Blueprint

## Pyramidal deterministic architecture

```text
        FORGE-K / Control Lane
              ^
          Crucible
              ^
          FORGE-HMK
              ^
          FORGE-T
              ^
     Rule Cells / Hyperlane
              ^
      Inputs / Events / Jobs
```

Authority tightens upward. Labor expands downward.

## Component roles

### FORGE-K / Control Lane

Owns canonical writes, semantic syscalls, live authority, approvals, and audit requirements.

### Crucible

Owns contradiction refinement, provenance checks, supersession checks, current-state validation, and promotion readiness. Crucible does not commit truth directly.

### FORGE-HMK

Owns memory assembly, cell activation, synapse traversal, temporal traces, HKV metadata, non-canonical artifacts, VSA/semantic projection metadata, and evidence packet construction.

### FORGE-T

Owns job admission, timing, leases, retries, backoff, TTL, cache expiration, prewarm scheduling, replay windows, priority aging, and backpressure.

### Neuron Mesh

Owns bounded specialist work. Workers emit typed artifacts only.

## Data flow

```text
Input/Event
  -> Rule Cells / Hyperlane
  -> FORGE-T job admission
  -> Neuron Mesh work orders
  -> FORGE-HMK memory assembly
  -> HKV / vector / VSA / trace surfaces
  -> Crucible validation
  -> Context Compiler
  -> model/tool execution
  -> proposed memory write
  -> Crucible
  -> FORGE-K / Control Lane
  -> journal/audit/canonical state
```


---

# FILE: context/AUTHORITY_BOUNDARIES.md

# Authority Boundaries

## Prime directive

FORGE-HMK must not become a second live authority path.

## Chain

```text
Worker result
  -> typed artifact
  -> FORGE-HMK assembly
  -> ClaimEnvelope
  -> Crucible validation
  -> FORGE-K / Control Lane semantic syscall
  -> journal + audit + canonical state
```

## Allowed early behavior

- read current memory/retrieval paths
- build shadow memory bundles
- create non-canonical cells/traces/manifests
- emit telemetry
- run shadow comparisons
- generate claim envelopes

## Forbidden early behavior

- overwrite canonical memory
- bypass audit
- expose public mutation routes
- mutate live state from workers
- treat vector similarity as truth
- treat VSA similarity as truth
- treat cache hits as truth
- assume snapshots are current state

## Cache authority

HKV is acceleration only. A cache hit means previous work may be reusable if dependency gates pass. It does not mean the cached content is currently true.

## Worker authority

Neuron Mesh workers are propose-only. They can find, decode, score, warn, and propose. They cannot commit.


---

# FILE: context/DATA_MODEL_BLUEPRINT.md

# Data Model Blueprint

Adapt to the repo's actual language/storage conventions.

## MemoryCell

Fields:

- cell_id
- workspace_id
- cell_type
- authority_level
- content_ref
- content_hash
- source_refs
- status
- confidence
- freshness
- created_at
- updated_at
- valid_from
- valid_to
- superseded_by
- schema_version

Cell types: FactCell, EventCell, StateCell, TaskCell, RuleCell, DecisionCell, ArtifactCell, PhotoCell, KineticCell, TraceCell, ReplayCell, ProjectionCell, ClaimCell.

## Synapse

Fields:

- synapse_id
- src_cell_id
- dst_cell_id
- relation_type
- weight
- confidence
- activation_count
- last_activated_at
- valid_from
- valid_to
- source_refs
- status

Relations: supports, contradicts, supersedes, depends_on, derived_from, belongs_to, blocks, unblocks, frequently_coactivated, requires_verification, same_entity_as, related_artifact, related_tool, related_policy.

## Job

Fields: job_id, kind, workspace_id, request_id, priority, policy_class, deadline_at, ttl_seconds, dedupe_key, parent_job_id, status, created_at, updated_at.

Kinds: PULL_SNAPSHOT, ASSEMBLE_CONTEXT, DECODE_CACHE_EXPRESSION, CAPTURE_PHOTO_FRAME, APPEND_KINETIC_DELTA, BUILD_TRACE, BUILD_REPLAY, PREWARM_HKV, VALIDATE_CLAIM, COMPARE_SHADOW, PROPOSE_PROMOTION.

## WorkOrder

Fields: work_order_id, job_id, team_type, step_kind, input_refs, expected_artifacts, affinity_tags, max_runtime_ms, cpu_budget_ms, gpu_budget_ms, token_budget, retry_policy, idempotency_key, status.

## ClaimEnvelope

Fields: claim_id, artifact_id, claim_type, proposed_operation, evidence_refs, contradiction_refs, confidence, scope, requires_crucible, requires_control_lane, validation_status, decision_reason.

## HKVManifest

Fields: cache_id, layer, workspace_id, model_id, tokenizer_id, prompt_template_hash, policy_epoch, memory_epoch, source_refs, dependency_hash, payload_ref, ttl_at, dirty_state, hit_count, last_used_at.


---

# FILE: context/METRICS_AND_TELEMETRY.md

# Metrics and Telemetry

## Primary metric

**time_to_useful_context**: time from request entering FORGE-HMK/FORGE-T to enough validated, scoped, relevant context being assembled for safe execution.

## Suggested early targets

- cached p50: < 350 ms
- cached p95: < 1.5 s
- fresh p50: < 2.5 s
- fresh p95: < 8 s

## Job metrics

- job_admission_count
- job_coalesced_count
- duplicate_job_rate
- queue_depth
- queue_wait_p95
- lease_timeout_count
- worker_retry_count
- dead_letter_count

## Cache metrics

- hkv_hit_ratio_by_layer
- hkv_miss_ratio_by_layer
- hkv_eviction_rate
- hkv_dirty_hit_blocked_count
- cache_dependency_invalidation_count
- prefix_cache_reuse_count
- compiled_context_reuse_count

## Crucible metrics

- claim_envelope_count
- claim_accept_count
- claim_reject_count
- claim_requires_more_evidence_count
- contradiction_detected_count
- supersession_approved_count
- promotion_blocked_count

## No-effect metrics

These must always be zero:

- shadow_no_effect_violation_count
- unauthorized_mutation_attempt_count
- canonical_write_without_control_lane_count


---

# FILE: context/PROJECT_CONTEXT.md

# Project Context

F.O.R.G.E. means **Foundry for Organic Reasoning, Growth, and Execution**.

FORGE is a personal AI operating layer that coordinates user context, files, tools, models, memory, projects, workflows, policies, and execution state. It should behave like an operating layer, not a chatbot with a search box stapled to it.

## FORGE-HMK definition

FORGE-HMK is the memory-side kernel responsible for:

- MemoryCells and Synapses
- PhotoCells, KineticCells, TraceCells, ReplayCells
- HKV cache manifests
- VSA / semantic algebra projection metadata
- memory job artifacts
- Neuron Mesh worker outputs
- non-canonical context assembly
- evidence packets for validation

FORGE-HMK is not RAG. It is not a vector DB wrapper. It is not a model. It is a managed cognitive memory substrate.

## Shadow-first doctrine

FORGE-HMK begins as read-mostly and shadow-first.

It may read, assemble, compare, cache, score, validate, and propose.

It may not initially overwrite canonical memory, bypass Control Lane, expose public mutation APIs, or treat cache/vector/VSA/snapshot output as truth.


---

# FILE: context/REPO_LAYOUT_RECOMMENDATION.md

# Repo Layout Recommendation

The implementing agent must inspect the actual repo before creating paths.

## Preferred docs

```text
docs/architecture/forge_hmk.md
docs/architecture/forge_t_temporal_tuner.md
docs/architecture/crucible.md
docs/architecture/neuron_mesh.md
docs/architecture/photo_kinetic_memory.md
docs/architecture/hkv_hierarchical_cache.md
docs/architecture/pyramidal_deterministic_architecture.md
docs/adr/00xx-forge-hmk-shadow-first-memory-kernel.md
docs/testing/forge_hmk_definition_of_done.md
```

## Preferred conceptual runtime modules

```text
services/core/internal/forgehmk/
  contracts/
  cells/
  synapses/
  temporal/
  hkv/
  neuronmesh/
  crucible/
  compiler/
  telemetry/
  shadow/
  store/

services/core/internal/forgetemporal/ or services/core/internal/temporaltuner/
  scheduler/
  governor/
  leases/
  coalescer/
  ttl/
  backpressure/
```

Follow actual repo naming conventions if they differ.

## Early integration rules

- internal packages only
- no public routes
- no canonical mutation
- no destructive migrations
- no required external services before local fallback exists


---

# FILE: context/WHAT_NOT_TO_DO.md

# WHAT NOT TO DO

## Do not violate authority

- Do not make FORGE-HMK live authority in early phases.
- Do not bypass FORGE-K / Control Lane.
- Do not let workers write canonical state.
- Do not let Crucible commit truth directly.
- Do not let cache/vector/VSA/snapshot output become truth.

## Do not create a swarm mess

- Do not create generic all-powerful agents.
- Do not allow unbounded worker fan-out.
- Do not allow workers to spawn unlimited child jobs.
- Do not run jobs without budgets, leases, cancellation, and scope.

## Do not flatten memory

- Do not collapse PhotoCell, KineticCell, TraceCell, ReplayCell into one blob.
- Do not overwrite old truth when superseding it.
- Do not delete contradiction evidence silently.
- Do not treat recency as truth.

## Do not wreck performance

- Do not recompute stable context every request.
- Do not full-scan archive when hot state is enough.
- Do not prewarm everything.
- Do not use one TTL policy for every cache layer.
- Do not hide cache invalidation.

## Do not hide behavior

- Do not create invisible memory promotions.
- Do not suppress validation warnings.
- Do not omit provenance from promotable claims.
- Do not skip telemetry because “it works.” That sentence is the preface to production pain.

## Do not put implementation code in chat

Write code to files. Summarize changed files and tests in chat.


---

# FILE: context/WORKER_TEAM_MODEL.md

# Neuron Mesh Worker Team Model

Workers are specialist labor units under FORGE-T leases. They are not authorities.

## Team types

### Snapshot Harvester

Pulls PhotoCells, snapshot bundles, restore seeds, and current-vs-prior shape comparisons.

### Trace Weaver

Builds TraceCells, compresses event windows, assembles before/action/after chains, and proposes ReplayCells.

### Binder Team

Runs VSA bind/unbind/bundle/permutation and semantic algebra transformations.

### Cache Smiths

Perform HKV lookup, dependency validation, stale entry detection, promotion/demotion recommendations, and prewarm prep.

### Context Assembly Team

Ranks cells, compiles context packets, splits stable/volatile blocks, attaches provenance, and enforces token budgets.

### Contradiction Scout

Surfaces contradiction groups, supersession chains, stale active state, and ClaimEnvelope drafts.

### Governor Watch

Monitors queue pressure, worker leases, backpressure state, and performance metrics.

## Worker output rule

Every worker output must be typed and scoped.

Bad: “I found some useful stuff.”

Good: `Artifact(type=SnapshotBundle, refs=[...], confidence=0.82, authority=non_canonical, warnings=[...])`


---

# FILE: handoff/FORGE_HMK_EXECUTIVE_SUMMARY.md

# FORGE-HMK Executive Summary

FORGE-HMK is the Heterogeneous Memory Kernel for Project F.O.R.G.E.

It turns memory from passive storage into a managed cognitive substrate.

It uses FORGE-HMK for memory kernel duties, FORGE-T for timing/scheduling/cache cadence, Neuron Mesh for bounded memory labor, Photo-Kinetic Memory for snapshots/transitions/traces/replay, HKV for hierarchical caching, Crucible for truth refinement, and FORGE-K / Control Lane for canonical authority.

The implementation must be shadow-first.

The law: **Workers propose. Crucible refines. FORGE-K commits.**


---

# FILE: handoff/ONE_SHOT_AGENT_BOOTSTRAP.md

# One-Shot Agent Bootstrap Prompt

You are implementing FORGE-HMK for Project F.O.R.G.E.

Read the prompt pack first. Do not write code until you inspect the repo.

Mandatory files:

- `README.md`
- `context/PROJECT_CONTEXT.md`
- `context/AUTHORITY_BOUNDARIES.md`
- `context/ARCHITECTURE_BLUEPRINT.md`
- `context/WHAT_NOT_TO_DO.md`
- `prompts/00_MASTER_IMPLEMENTATION_PROMPT.md`

Rules:

- no implementation code in chat
- write changes to files
- preserve existing live authority
- no public mutation routes
- workers propose only
- Crucible validates only
- Control Lane commits only
- cache/vector/VSA are not truth
- snapshots are not truth
- tests required
- docs required

Start by producing a repo boundary summary and naming the phase you will execute.


---

# FILE: handoff/REVIEW_PROMPT_AUTHORITY_AUDIT.md

# Review Prompt: FORGE-HMK Authority Audit

Review the implementation for authority violations.

Check:

- Can FORGE-HMK write canonical memory?
- Can a worker write canonical memory?
- Can Crucible commit truth directly?
- Can cache/VSA/vector output become truth?
- Are public mutation routes exposed?
- Are claim envelopes required before promotion?
- Is Control Lane still the commit path?
- Are no-effect tests present?

Output: PASS/FAIL, critical findings, risky files, required fixes, and what not to do next.


---

# FILE: handoff/REVIEW_PROMPT_PERFORMANCE_AUDIT.md

# Review Prompt: FORGE-HMK Performance Audit

Review FORGE-HMK and FORGE-T for performance bottlenecks.

Check time-to-useful-context, cache hit/miss ratios, duplicate job rate, worker utilization, queue wait p95, stale/dirty cache blocks, synapse traversal depth, replay window cost, prewarm success/waste, and backpressure behavior.

Output top bottlenecks, quick wins, refactor candidates, missing metrics, benchmark gaps, and over-optimization risks.

Do not weaken validation for speed.


---

# FILE: prompts/00_MASTER_IMPLEMENTATION_PROMPT.md

# MASTER IMPLEMENTATION PROMPT: FORGE-HMK

You are implementing **FORGE-HMK**, the Heterogeneous Memory Kernel for Project F.O.R.G.E.

Act as a careful senior platform engineer and deterministic AI-OS implementer.

## Mandatory reading

Before changing code, read:

- `context/PROJECT_CONTEXT.md`
- `context/ARCHITECTURE_BLUEPRINT.md`
- `context/AUTHORITY_BOUNDARIES.md`
- `context/WHAT_NOT_TO_DO.md`
- `context/DATA_MODEL_BLUEPRINT.md`
- `context/REPO_LAYOUT_RECOMMENDATION.md`

Then inspect the repository for current authority, memory, jobs, context compiler, audit, and test paths.

## Mission

Implement FORGE-HMK as a **shadow-first heterogeneous memory kernel** that manages memory cells, synapses, temporal traces, HKV cache manifests, worker job artifacts, and validated context assembly without becoming a second live authority path.

## Non-negotiables

- FORGE-K / Control Lane remains canonical authority.
- FORGE-HMK begins read-mostly and shadow-first.
- FORGE-T governs timing and worker/caching cadence.
- Neuron Mesh workers are propose-only.
- Crucible validates claims before promotion.
- HKV is acceleration, not truth.
- Vectors and VSA are retrieval/semantic surfaces, not truth.
- All canonical writes remain journaled, audited, and routed through the existing authority path.

## Three-pass execution

### Pass 1 — Discover and map

Inspect current repo architecture. Identify current live authority, simulator-only paths, memory/retrieval, jobs, context compiler, audit, and tests.

### Pass 2 — Implement smallest safe vertical slice

Add contracts/no-op shells first. Add tests before or alongside behavior. Keep early behavior non-authoritative.

### Pass 3 — Validate and document

Run tests. Update docs. Summarize changed files, tests run, and remaining risks.

## What not to do

- Do not implement all phases at once.
- Do not bypass current live authority.
- Do not expose public mutation APIs.
- Do not let worker output mutate canonical memory.
- Do not put implementation code in chat. Write code to files and summarize only.


---

# FILE: prompts/01_PHASE_0_REPO_AUDIT_AND_BOUNDARY_MAP.md

# PHASE 0 — Repo Audit and Boundary Map

## Objective

Map current FORGE authority, memory, job, context, cache, audit, and runtime boundaries before implementing.

## Instructions

Inspect repo docs/packages/tests. Identify live authority paths, simulator/shadow-only paths, current memory/retrieval paths, test commands, and safe package locations. Create `docs/reviews/forge_hmk_boundary_map.md`.

## Validation

No behavior changes. Boundary map names concrete files/packages. Authority and simulator paths are separated.

## What not to do

Do not create runtime code. Do not rename modules. Do not assume paths without inspection.

## Exit gate

Exit when the repo map shows exactly where FORGE-HMK can be added safely.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/02_PHASE_1_CONTRACTS_AND_NOOP_SHELLS.md

# PHASE 1 — Contracts and No-Op Shells

## Objective

Create typed contracts and no-op service shells without enabling live mutation.

## Instructions

Add contracts for Job, WorkOrder, Lease, MemoryCell, Synapse, PhotoCell, KineticCell, TraceCell, ReplayCell, HKVManifest, Artifact, ClaimEnvelope. Add interfaces for FORGE-HMK, FORGE-T, HKV, Neuron Mesh, Crucible. Add no-op/in-memory stubs and tests.

## Validation

Contracts compile. No-op services cannot write canonical memory. Existing tests pass.

## What not to do

Do not connect to live mutation. Do not expose public routes. Do not add external service requirements.

## Exit gate

Exit when interfaces compile and tests prove no-effect behavior.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/03_PHASE_2_FORGE_T_TEMPORAL_TUNER.md

# PHASE 2 — FORGE-T Temporal Tuner

## Objective

Implement scheduler-adjacent timing/performance governor for memory jobs and worker leases.

## Instructions

Implement job admission, dedupe/coalescing, leases, heartbeat, timeout, retry/backoff, cancellation, priority aging, TTL hooks, and backpressure. Start with in-memory queue.

## Validation

Duplicate jobs coalesce. Timed-out leases requeue. Cancelled jobs cannot complete. No canonical writes.

## What not to do

Do not create infinite worker loops. Do not run jobs without budgets. Do not prewarm everything yet.

## Exit gate

Exit when FORGE-T can safely admit, dedupe, lease, retry, cancel, and observe jobs.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/04_PHASE_3_FORGE_HMK_CORE.md

# PHASE 3 — FORGE-HMK Core

## Objective

Implement read-side memory kernel for cells, synapses, and non-canonical memory artifacts.

## Instructions

Implement MemoryCell/Synapse repositories or stubs, read-only adapters, scoped activation, traversal caps, provenance, non-canonical artifacts, and tests.

## Validation

Reads are scoped. Traversal is bounded. Outputs include provenance. Existing memory path unchanged.

## What not to do

Do not replace existing memory/retrieval. Do not promote cell results to truth.

## Exit gate

Exit when scoped non-canonical memory artifacts can be assembled safely.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/05_PHASE_4_PHOTO_KINETIC_MEMORY.md

# PHASE 4 — Photo-Kinetic Temporal Memory

## Objective

Add snapshot/transition/trace/replay memory without treating snapshots as truth.

## Instructions

Implement PhotoCell, KineticCell, TraceCell, ReplayCell contracts and deterministic reconstruction from base snapshot + deltas. Connect TTL/freshness hooks.

## Validation

PhotoCells are shape, not truth. KineticCells preserve before/action/after. TraceCells are append-only. ReplayCells are proposals.

## What not to do

Do not replay into live state. Do not save giant full snapshots for every tiny change. Do not delete traces.

## Exit gate

Exit when temporal traces can be created and reconstructed deterministically.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/06_PHASE_5_HKV_CACHE.md

# PHASE 5 — HKV Hierarchical Cache

## Objective

Implement HKV manifest identity, dependency tracking, TTL policy, dirty-state handling, and cognitive cache metadata.

## Instructions

Implement cache layers, HKVManifest store, dependency invalidation, dirty states, workspace isolation, and tests for identity mismatch/expiry/dirty hit blocking.

## Validation

Cache hit requires exact identity/dependency match. Dirty/expired entries blocked. HKV disable safe.

## What not to do

Do not implement engine-specific GPU KV yet. Do not cache final answers as truth.

## Exit gate

Exit when HKV safely validates, invalidates, and blocks stale cache entries.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/07_PHASE_6_NEURON_MESH.md

# PHASE 6 — Neuron Mesh Worker Teams

## Objective

Implement bounded worker teams using typed work orders, leases, validators, and propose-only outputs.

## Instructions

Add worker registry, capabilities, team types, in-process workers, budget/cancellation handling, typed artifacts, and tests.

## Validation

Workers cannot mutate canonical memory. Outputs are typed. Leases and scopes enforced.

## What not to do

Do not build generic chat agents. Do not allow unlimited subjobs. Do not bypass budgets.

## Exit gate

Exit when workers execute bounded memory jobs and emit non-canonical artifacts.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/08_PHASE_7_CRUCIBLE.md

# PHASE 7 — Crucible Truth Refinement

## Objective

Implement validation/refinement layer for claims, contradictions, supersession, provenance, and promotion readiness.

## Instructions

Implement ClaimEnvelope validation, provenance checks, scope checks, contradiction detection, supersession validation, current-state hooks, and decision states.

## Validation

Crucible cannot commit truth. Missing provenance blocks promotion. Contradictions preserved.

## What not to do

Do not implement automatic live promotion. Do not bypass Control Lane. Do not approve cache-only claims.

## Exit gate

Exit when Crucible can refine/reject/promote-to-review claims while non-authoritative.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/09_PHASE_8_CONTEXT_COMPILER_INTEGRATION.md

# PHASE 8 — Context Compiler Integration

## Objective

Integrate FORGE-HMK outputs with context compiler through shadow-only compiled context bundles.

## Instructions

Inspect existing compiler. Add adapter for non-canonical artifacts. Build bundles with included/excluded cells, provenance, relevance scores, contradiction warnings, stable/volatile blocks, HKV manifest. Add shadow comparison.

## Validation

Existing compiler default unchanged. FORGE-HMK output shadow-only. Token budget respected.

## What not to do

Do not replace compiler outright. Do not hide contradiction warnings.

## Exit gate

Exit when shadow context packets build with provenance and no live output changes.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/10_PHASE_9_TELEMETRY_AND_PERFORMANCE.md

# PHASE 9 — Telemetry and Performance Governor

## Objective

Add metrics, benchmarks, and governor behavior around time-to-useful-context and cache/worker efficiency.

## Instructions

Implement metrics, local benchmarks, governor actions for suspend prewarm, coalesce duplicates, throttle low-priority work, fresh compile fallback, and cache bypass.

## Validation

Benchmarks run locally. Metrics emitted. Backpressure testable.

## What not to do

Do not claim performance wins without measurement. Do not optimize by weakening validation.

## Exit gate

Exit when performance can be measured and governed safely.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: prompts/11_PHASE_10_SHADOW_PARITY_AND_CUTOVER_PREP.md

# PHASE 10 — Shadow Parity and Cutover Prep

## Objective

Run FORGE-HMK in shadow mode against current memory/context behavior and prepare narrow live-promotion candidates.

## Instructions

Add shadow harness. Record safe metadata only. Compare relevance, latency, cache hit ratio, contradiction warnings, provenance coverage, missing/extra memory. Write readiness report.

## Validation

Zero user-visible output changes. No canonical mutation. Safe metadata only. Rollback documented.

## What not to do

Do not cut over in this phase. Do not store raw sensitive payloads.

## Exit gate

Exit when safe shadow parity is demonstrated and one narrow reversible live candidate is documented.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.


---

# FILE: rules/AGENTS_HMK.md

# AGENTS_HMK.md

## First actions

1. Read this file.
2. Read `context/WHAT_NOT_TO_DO.md`.
3. Read `context/AUTHORITY_BOUNDARIES.md`.
4. Inspect repo structure.
5. State the phase being implemented.

## Rules

- no implementation code in chat
- write changes to files
- preserve existing live authority
- no public mutation routes
- no canonical writes from workers
- no cache-derived truth
- no vector/VSA-derived truth
- no destructive migrations
- no broad refactors unless explicitly requested

## Final summary

Report files changed, tests added, tests run, authority status, no-effect status, risks, and next phase.


---

# FILE: rules/cursor_rules_forge_hmk.mdc

---
description: FORGE-HMK implementation rules
globs:
  - "**/*"
alwaysApply: false
---

# FORGE-HMK Cursor Rules

Do not put implementation code in chat. Make file changes directly.

Rules:

- FORGE-HMK is shadow-first.
- FORGE-K / Control Lane remains canonical authority.
- FORGE-T schedules and governs; it does not commit truth.
- Neuron Mesh workers are propose-only.
- Crucible validates; it does not directly commit.
- HKV is acceleration only.
- Vector/VSA outputs are evidence only.
- Snapshots are historical shape, not current truth.
- Every promotable claim needs provenance.
- Every new behavior needs tests.


---

# FILE: skills/SKILL_crucible_validator.md

# Skill: Crucible Validator

Validate memory claims before promotion.

## Responsibilities

- validate ClaimEnvelope shape
- require provenance
- detect contradictions
- validate supersession chains
- compare current state
- emit decision states: rejected, needs_more_evidence, shadow_only, accepted_for_review, promotable

Crucible does not commit truth directly.


---

# FILE: skills/SKILL_forge_hmk_architect.md

# Skill: FORGE-HMK Architect

Design memory-kernel architecture with strict deterministic authority boundaries.

## Responsibilities

- preserve FORGE-K / Control Lane authority
- keep FORGE-HMK shadow-first
- define module boundaries
- prevent cache/vector/VSA truth creep
- enforce pyramidal deterministic architecture
- document no-effect guarantees

## Output format

Always produce: current state, target state, authority boundary, module map, data flow, validation plan, risks, and what not to do.


---

# FILE: skills/SKILL_hkv_cache_engineer.md

# Skill: HKV Cache Engineer

Implement hierarchical cache manifest behavior.

## Layers

- model KV manifest
- prompt prefix manifest
- compiled context cache
- activated subgraph cache
- cell projection cache
- synapse route cache
- semantic operation cache
- temporal replay cache

## Rule

Cache is acceleration, not truth.


---

# FILE: skills/SKILL_neuron_mesh_engineer.md

# Skill: Neuron Mesh Engineer

Build specialist worker teams that execute typed memory jobs under FORGE-T leases.

## Rules

- no generic all-powerful agents
- no canonical writes
- no unlimited subjobs
- budgets and scopes required
- outputs are typed artifacts
- failures become retry, dead-letter, warning, or no-op


---

# FILE: skills/SKILL_temporal_tuner_engineer.md

# Skill: FORGE-T Temporal Tuner Engineer

Implement timing, scheduling, TTL, retries, leases, coalescing, priority aging, replay windows, and backpressure.

## Rules

- jobs require kind, scope, priority, budget, and dedupe key
- duplicates coalesce
- leases heartbeat or expire
- retries are bounded
- prewarm is budgeted
- backpressure degrades safely
- scheduling never loosens authority


---

# FILE: validation/FAILURE_MODES.md

# Failure Modes

## Authority blur

Mitigation: Control Lane commit path, Crucible validator-only, workers propose-only, no public mutation route.

## Stale cache poisoning

Mitigation: dependency hashes, policy epochs, memory epochs, dirty states, workspace isolation, invalidation tests.

## Worker sprawl

Mitigation: coalescing, budgets, traversal caps, priority aging, queue limits, backpressure.

## Temporal drift

Mitigation: snapshots are shape not truth, active state requires validation, replay requires current-state checks.

## Vector/VSA truth creep

Mitigation: evidence-only semantics, provenance, Crucible validation, Control Lane commit.

## Trace bloat

Mitigation: delta snapshots, compaction policy, retention tiers, cold archive.

## Oscillatory scheduling

Mitigation: hysteresis thresholds, cooldown windows, bounded retries, min TTL floors.


---

# FILE: validation/PHASE_EXIT_GATES.md

# Phase Exit Gates

Every phase must answer:

1. What changed?
2. What did not change?
3. What authority boundaries were preserved?
4. What tests were added?
5. What tests were run?
6. What risks remain?
7. Is rollback obvious?
8. Did any live behavior change?

## Absolute blockers

Stop if:

- worker can mutate canonical state
- FORGE-HMK can bypass Control Lane
- cache/VSA/vector output becomes truth
- shadow mode changes user-visible behavior
- provenance is missing from promotable claims
- no-effect tests fail


---

# FILE: validation/TEST_STRATEGY.md

# Test Strategy

## Contract tests

Validate required fields, zero-value rejection, serialization, schema versions.

## Authority tests

Worker cannot write canonical state. Crucible cannot commit directly. Cache cannot promote truth. FORGE-HMK cannot bypass Control Lane.

## Scheduler tests

Job admission, dedupe/coalescing, lease acquisition, heartbeat, timeout, retry, cancellation, backpressure.

## Memory tests

Cell scoping, synapse traversal depth, relation filtering, stale/superseded filtering, provenance preservation.

## Temporal tests

PhotoCell capture, KineticCell reconstruction, TraceCell append order, ReplayCell proposal validation, stale replay blocking.

## HKV tests

Exact identity, mismatch, TTL expiry, dependency invalidation, dirty-hit blocking, workspace isolation.

## Crucible tests

Missing provenance rejection, contradiction detection, supersession validation, stale evidence rejection, decision states.

## Shadow tests

Existing path unchanged, no public output change, safe metadata only, no canonical mutation, parity report generated.
