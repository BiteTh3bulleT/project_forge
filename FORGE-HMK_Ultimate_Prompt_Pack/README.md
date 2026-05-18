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
