FORGE // UNIFIED COGNITIVE OS IMPLEMENTATION PROMPT
Authority-first implementation. No cosplay. No sidecars pretending to be the system.

You are implementing the next major FORGE evolution:
turn FORGE into a unified local-first cognitive operating system by integrating the useful patterns of:

- IRIS = cognitive flow lanes (Neural / Arterial / Lymphatic)
- FORGE = deterministic kernel, semantic syscalls, state authority, journal, cognitive filesystem
- GHOST = runtime intelligence, execution loop, multi-model orchestration, self-healing reflexes
- ARTEMIS = interface, perception, multimodal/operator-facing surfaces

This is NOT a new product and NOT a parallel subsystem.
This must be implemented INSIDE the existing FORGE architecture and repo.

REPO AND ARCHITECTURE ARE BINDING

Read first and treat as binding:
- AGENTS.md
- docs/architecture/forge_ai_os.md
- docs/architecture/semantic_syscalls.md
- docs/data_model/cognitive_filesystem.md
- docs/roadmap/forge_ai_os_phases.md
- docs/status/reality_matrix.md
- docs/status/model_runtime_m3_baseline.md

Core doctrine you must preserve:
- FORGE is the operating system.
- Kernel owns truth.
- No LLM may write canonical truth directly.
- Durable mutation happens through semantic syscalls / control lane.
- Context is compiled from evidence, not guessed from chat.
- Gateway remains the governed tool boundary.
- Modelruntime remains the governed inference substrate.
- All state/action lineage must stay auditable.

PRIMARY OBJECTIVE

Implement a unified runtime where:
- FORGE kernel remains truth authority
- IRIS-like lane patterns become internal service architecture
- GHOST-like reasoning/execution becomes runtime orchestration over modelruntime + gateway
- ARTEMIS-like UI/perception becomes operator/client surface over the hardened core

Do NOT build four separate subsystems.
Build one coherent cognitive runtime.

TARGET OPERATING MODEL

The resulting system should conceptually operate like this:

ARTEMIS surface/perception layer
    -> Neural lane ingest/normalization/events
    -> Arterial lane context build / reasoning / execution planning
    -> FORGE kernel semantic syscalls / truth / journal / state registry
    -> Lymphatic lane repair / consolidation / maintenance / diagnostics

This must map to existing FORGE seams rather than inventing an unrelated architecture.

NON-NEGOTIABLE RULES

1. FORGE kernel remains the only authority for canonical writes.
2. All durable writes stay behind semantic syscalls and control-lane validation/commit boundaries.
3. Gateway remains the authoritative tool execution path.
4. Modelruntime remains the authoritative model execution path.
5. Context restore and context assembly must remain deterministic, inspectable, and scope-bounded.
6. Snapshot memory remains non-canonical evidence, not truth authority.
7. No subsystem may bypass audit/journal/provenance/correlation/trace linkage.
8. No retry storms.
9. No direct cloud dependency when unconfigured.
10. No giant model defaulting when smaller/local/governed options are valid.

WHAT YOU ARE BUILDING

Build this as service families inside FORGE:

A. KERNEL / AUTHORITY LAYER (FORGE)
- semantic syscall kernel remains authoritative
- state registry remains truth source
- audit + journal remain mandatory
- approval/capability gates remain in force

B. NEURAL LANE
Purpose:
- ingest raw events, operator input, environment observations, UI actions, external signals
Implement or extend:
- normalization pipeline
- event classification
- evidence admission rules
- input provenance tagging
- explicit route into journal / evidence stores

C. ARTERIAL LANE
Purpose:
- compile working context
- score prior snapshots
- select evidence
- run reasoning / planning
- produce proposed actions
Implement or extend:
- restore scoring and candidate ranking
- header-first restore path
- working memory packet assembly
- reasoning/execution planning contracts
- operator-visible context traces

D. GHOST-LIKE EXECUTION RUNTIME
Purpose:
- multi-model orchestration
- bounded reasoning loops
- tool sequencing
- self-healing reflexes at runtime
Implement or extend INSIDE existing modelruntime/gateway doctrine:
- planner / executor / verifier style orchestration
- provider fallback / cooldown handling
- bounded retries
- execution loop status + checkpointing
- zero direct truth writes from LLM outputs

E. LYMPHATIC LANE
Purpose:
- background maintenance
- consolidation
- contradiction surfacing
- repair
- snapshot lifecycle upkeep
Implement or extend:
- consolidation scheduler
- stale snapshot review
- contradiction review queues
- vector/index repair hooks
- provider cooldown recovery
- diagnostic sweeps
- dry-run maintenance reports before auto-commit behavior

F. ARTEMIS-LIKE SURFACES
Purpose:
- operator-facing interaction
- future multimodal perception
Implement or extend as client surfaces, not truth sources:
- shell / UI inspector for context packets
- snapshot inspector
- lineage / audit explorer
- action trace explorer
- later optional voice/vision/perception adapters behind flags

PHASED IMPLEMENTATION ORDER

Phase 0 — Reality lock
- Audit repo and identify authoritative seams already present.
- Produce a short implementation map:
  - existing files/modules to extend
  - authority boundaries to preserve
  - legacy side doors to converge
- Do not start coding until this map is written into repo docs or a durable implementation note.

Phase 1 — Authority convergence
Goal:
remove or bound duplicated truth surfaces before expanding autonomy.

Required:
- identify any remaining direct-write / legacy-write paths that bypass syscall kernel
- converge duplicated event or state truth where feasible
- ensure journal/audit/provenance/correlation/trace coverage remains intact
- harden “kernel decides truth” doctrine in code and docs

Acceptance:
- no newly introduced direct truth side paths
- legacy paths documented and bounded if not removed
- tests prove kernel/governed path remains authoritative

Phase 2 — Arterial runtime completion
Goal:
make context restore and working memory operational, not archival.

Required:
- complete restore scoring/ranking over persisted context snapshots
- candidate listing by scope/query/snapshot kind
- header-first restore package
- resume hints producer/consumer contract
- score breakdown persisted and inspectable
- integrate restore-aware path into COMPILE_CONTEXT or an equally doctrine-safe equivalent

Acceptance:
- restore scoring is deterministic and explainable
- contradictory/stale snapshots are penalized
- no candidate => recommends fresh compile cleanly
- operator can inspect candidate ranking and selected evidence

Phase 3 — GHOST runtime integration
Goal:
introduce bounded runtime intelligence using existing modelruntime + gateway.

Required:
- create a runtime orchestration layer over modelruntime
- support multiple model roles (planner / executor / verifier / summarizer) under governed selection
- all tool/world interaction routes through gateway
- all durable mutations return to syscall proposal/commit path
- add bounded retries, provider cooldown respect, and checkpointable execution state

Acceptance:
- no tool or model path bypasses gateway/modelruntime/syscall authority
- retry storms prevented
- provider 404/blacklist/cooldown behavior respected
- execution traces are inspectable end to end

Phase 4 — Lymphatic lane formalization
Goal:
make maintenance a first-class bounded subsystem.

Required:
- consolidation jobs
- snapshot hygiene
- contradiction surfacing
- vector/index repair routines
- integrity diagnostics
- dry-run maintenance mode
- scheduler and reporting

Acceptance:
- maintenance actions are bounded, logged, and auditable
- dry-run mode available
- diagnostics expose actionable failures, not vague whining

Phase 5 — ARTEMIS operator surfaces
Goal:
make the system legible to humans before expanding autonomy.

Required:
- snapshot inspector
- context packet inspector
- audit/journal explorer
- runtime execution explorer
- restore score viewer
- later optional voice/vision/perception behind feature flags only

Acceptance:
- operator can explain what the system saw, selected, proposed, executed, and committed

Phase 6 — Goal/autonomy expansion under safety gates
Goal:
add mission-like behavior only after runtime, traceability, and authority are stable.

Required:
- intent queue / bounded autonomy budgets
- pause/resume/abort
- explicit approval thresholds
- mission checkpoints
- failure recovery rules

Acceptance:
- autonomous behavior is gated, auditable, and stoppable
- no free-running loop without bounded control

IMPLEMENTATION REQUIREMENTS

You must implement by extending existing FORGE modules where possible.
Prefer extending current packages over creating parallel abstractions.

At minimum, identify and wire concrete homes for:

Kernel / authority
- semantic syscall control lane
- state registry surfaces
- journal/audit lineage
- approval/capability policy

Context / arterial
- context compiler
- restore scorer
- working memory assembly
- resume hints
- context trace

Runtime / orchestration
- modelruntime integration
- provider role routing
- bounded execution loop
- verifier/reflection hooks
- gateway-mediated tool calls

Lymphatic / maintenance
- consolidation scheduler
- repair routines
- contradiction and stale-state review
- diagnostics

Surface / ARTEMIS
- operator shell/UI views
- trace explorer
- snapshot inspector
- context inspector

TEST REQUIREMENTS

Add real tests for:

Authority / kernel
- no direct truth write outside semantic syscalls
- audit + journal linkage preserved
- approval/capability denials do not commit

Context / restore
- restore candidate listing
- score ranking correctness
- stale/contradictory snapshot penalties
- header-first restore behavior
- fresh-compile fallback when restore quality is below threshold

Runtime
- multi-model role routing
- modelruntime preflight
- provider cooldown/blacklist handling
- retry bounding / no retry storms
- gateway-only tool execution path

Maintenance
- consolidation dry-run
- vector/index repair behavior
- maintenance scheduler bounded execution
- diagnostics report generation

Parity / runtime safety
- in-memory and SQLite path parity on critical context behavior
- shutdown idempotence
- startup idempotence
- smoke path from ingest -> context -> runtime proposal -> syscall commit

DOC REQUIREMENTS

Update or create:
- README.md
- AGENTS.md
- architecture docs for unified runtime
- operations docs
- restore scoring docs
- modelruntime orchestration docs
- maintenance/repair docs
- migration docs if persistence changes

WHAT NOT TO DO

- do not build a separate Python sidecar OS and pretend it is FORGE
- do not create four co-equal subsystems with overlapping truth authority
- do not let LLMs write truth directly
- do not bypass gateway for tools
- do not bypass modelruntime for inference
- do not bypass syscall/control lane for durable mutation
- do not enable GUI/audio/vision by default
- do not default to giant models
- do not require DB deletion for schema changes
- do not hide restore scoring behind opaque magic numbers
- do not skip traceability
- do not leave dead paths, duplicate shutdown logic, or duplicated event authority
- do not use vector retrieval as truth authority
- do not output code in chat

DELIVERABLE FORMAT

At the end provide:
1. summary of work completed
2. final filesystem tree of changed areas
3. authority convergence summary
4. restore scoring summary
5. runtime orchestration summary
6. maintenance/diagnostics summary
7. run instructions
8. safe mode instructions
9. known follow-up items

EXECUTION STYLE

- Work in small, logically grouped commits.
- Keep behavior deterministic unless explicitly in the probabilistic reasoning layer.
- Preserve production quality.
- Keep code and docs aligned.
- Prefer real implementations over placeholders.
- When a feature is too large for one pass, land the substrate first but wire it end to end.

Begin.
