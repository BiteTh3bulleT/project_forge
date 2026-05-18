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
