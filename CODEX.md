FORGE // MASTER UNIFIED COGNITIVE OS IMPLEMENTATION PROMPT

Status: `[FUTURE]` implementation vision and planning prompt, not current implementation truth. Current behavior is defined by code, ADRs, `AGENTS.md`, `README.md`, status docs, and passing tests. Do not treat unimplemented statements in this file as live daemon status.

Production-grade implementation only. No toy abstractions. No parallel kingdoms. No chatbot-wrapper nonsense.

You are implementing the next major FORGE evolution:
turn FORGE into a unified Local-First Cognitive Operating System by integrating the useful patterns of:

- IRIS = cognitive flow lanes (Neural / Arterial / Lymphatic)
- FORGE = deterministic kernel, semantic syscalls, state authority, journal, cognitive filesystem
- GHOST = runtime intelligence, bounded execution loops, multi-model orchestration, self-healing reflexes
- ARTEMIS = perception, operator interface, multimodal surface

This must be implemented INSIDE the existing FORGE architecture and repository.
Do not build a separate system beside FORGE.
Do not create a Python sidecar monolith.
Do not fork truth authority.

BINDING REPO CONTEXT

Read first and treat as binding:
- AGENTS.md
- README.md
- docs/architecture/forge_ai_os.md
- docs/architecture/semantic_syscalls.md
- docs/data_model/cognitive_filesystem.md
- docs/roadmap/forge_ai_os_phases.md
- docs/status/reality_matrix.md
- docs/status/model_runtime_m3_baseline.md

Current architecture facts you must preserve:
- FORGE kernel owns canonical truth.
- Durable mutation happens only through semantic syscalls / control lane.
- COMPILE_CONTEXT is the authoritative context assembly seam.
- Context restore snapshots are non-canonical evidence, not truth.
- modelruntime is the governed inference substrate.
- gateway is the governed tool execution path.
- journal / audit / provenance / trace / correlation are mandatory lineage surfaces.
- Phase 6.25 context restore snapshots exist; Phase 6.5 restore scoring/runtime consumption is the next real gate.

PRIMARY OBJECTIVE

Implement the full next-generation FORGE runtime so that:

1. Kernel remains deterministic and authoritative.
2. Cognitive flow is structured into Neural / Arterial / Lymphatic lanes.
3. Runtime intelligence uses small, medium, and frontier models under governance.
4. Dream Mode becomes the main consolidation/metabolism subsystem.
5. Memory becomes tiered:
   - short-term
   - mid-term
   - long-term
6. Snapshot restore becomes operational:
   - scored
   - ranked
   - header-first
   - explainable
7. Operator surfaces expose what the system saw, chose, proposed, executed, repaired, and remembered.

ARCHITECTURAL PRINCIPLES

These are non-negotiable:

1. Kernel decides truth.
2. Context engine decides relevance.
3. modelruntime decides inference routing/execution.
4. gateway decides tool execution.
5. LLMs decide language/strategy/proposals only.
6. No LLM may directly mutate canonical truth.
7. Vector retrieval is never truth authority.
8. Dream Mode may propose promotions/demotions/repairs, but commits still go through governed control-lane/syscall boundaries.
9. Frontier models are escalation targets, not defaults.
10. Efficiency matters:
   - sparse activation
   - small working memory
   - prediction/error-driven context
   - offline consolidation
   - bounded retries
   - no retry storms

UNIFIED OPERATING MODEL

Target runtime shape:

ARTEMIS surface/perception layer
    -> Neural lane ingest / normalization / event admission
    -> Arterial lane restore scoring / context compilation / reasoning / execution planning
    -> FORGE kernel semantic syscalls / state registry / journal / audit
    -> Lymphatic lane Dream Mode / consolidation / repair / diagnostics

This is one system.
Do not implement this as separate co-equal runtimes.

IMPLEMENTATION SCOPE

At minimum, implement or extend the following service families and concrete repo areas.

A. KERNEL / AUTHORITY LAYER
Use existing FORGE kernel/control-lane/domain seams.

Required behavior:
- state authority remains kernel-owned
- syscall validation/approval/capability boundaries remain authoritative
- canonical writes remain journaled and auditable
- no new bypass around control lane

Required work:
- strengthen any remaining side doors or legacy write paths
- ensure all new subsystems commit via syscall-governed paths when durable mutation occurs
- keep audit/provenance/correlation/trace intact for every durable change

B. NEURAL LANE
Purpose:
- ingest raw signals
- normalize them
- classify them
- admit them as evidence or reject them
- publish event objects into the system

Implement/extend services for:
- ingest
- normalization
- event admission
- provenance tagging
- embedding hooks (governed; optional if already present)
- environment / interface event intake

Target conceptual modules:
- lanes/neural/ingest
- lanes/neural/normalization
- lanes/neural/events
- lanes/neural/embedding

Map these into FORGE’s actual packages and docs rather than creating duplicate fake directories unless needed.

Required outputs:
- normalized event objects
- route decisions (journal only / evidence / memory candidate / ignore)
- deterministic admission reasons

C. ARTERIAL LANE
Purpose:
- choose candidate restore snapshots
- score them
- compile working memory
- select evidence
- route models by role
- generate action proposals
- prepare operator-facing responses

Implement/extend:
- context engine
- context budgeter
- context ranker
- working memory
- context trace
- context policy
- reasoning planner
- execution planner
- response packager

Target conceptual modules:
- context/engine
- context/budget
- context/ranker
- context/working_memory
- context/trace
- context/policy
- lanes/arterial/reasoning
- lanes/arterial/execution
- lanes/arterial/response

PHASE 6.5 REQUIREMENTS (MANDATORY)
Implement live restore scoring/runtime consumption on top of Phase 6.25 snapshots.

Required:
1. candidate retrieval by:
   - workspace
   - lane
   - query
   - snapshot kind
   - recency window
2. deterministic restore scorer producing persisted explainable score breakdown:
   - total
   - query_score
   - scope_score
   - recency_score
   - lineage_score
   - state_overlap_score
   - loop_overlap_score
   - artifact_overlap_score
   - contradiction_penalty
   - staleness_penalty
   - explain[]
3. header-first restore path:
   - load winner snapshot header first
   - then expand only needed evidence
4. resume_hints_json producer/consumer contract:
   - next_action
   - top_blockers
   - dominant_state_keys
   - dominant_loop_ids
   - recommended_evidence_ids
   - restore_confidence
   - requires_fresh_compile
5. full operator-visible restore trace:
   - candidates considered
   - score breakdowns
   - winner
   - penalties
   - selected evidence
   - fallback to fresh compile when needed

Do not introduce LLM-based restore scoring.
Keep this deterministic.

D. GHOST-LIKE RUNTIME INTELLIGENCE
Purpose:
- bounded reasoning loops
- role-based model orchestration
- planning / execution / verification
- self-healing runtime reflexes

Implement/extend INSIDE existing modelruntime/gateway doctrine:
- llm/orchestrator equivalent inside FORGE
- provider selection by capability/latency/cost/availability
- role-based model routing:
  - classifier
  - planner
  - executor
  - verifier
  - summarizer
  - repair analyst
- bounded execution loop
- provider cooldown and blacklist handling
- no retry storms
- checkpointable execution traces

Model policy:
- small-model competent
- big-model amplified
- frontier models available as escalation targets
- no giant-model defaulting
- no cloud-provider default when unconfigured

Support providers/concepts analogous to:
- local models
- Ollama
- OpenAI-compatible provider
- Anthropic-compatible provider
- local transformers
but adapt to current FORGE modelruntime reality rather than creating a second provider stack unless necessary.

E. MEMORY SYSTEM
Implement the memory system as layered durable + non-canonical evidence.

Canonical/durable layers:
- episodic
- semantic
- reflective
- journal
- state registry
- cognitive filesystem links/artifacts/models/loops

Retrieval/support layers:
- vector retrieval
- restore snapshots
- context packet evidence
- snapshot cards

Introduce explicit memory-tier logic:
1. short-term memory
   - recent episodes
   - active task traces
   - today/session events
   - open contradictions
   - immediate restore continuity
2. mid-term memory
   - project-scoped summaries
   - repeated patterns
   - emerging procedures
   - candidate semantic links
   - failure signatures
3. long-term memory
   - durable facts
   - validated preferences
   - stable procedures
   - architecture truths
   - high-confidence recurring operator patterns

Do not flatten all memory into one vector store.
Do not use vector retrieval as truth authority.

F. DREAM MODE (LYMPHATIC LANE)
This is mandatory and first-class.

Dream Mode is the main maintenance/metabolism subsystem.
It must be implemented as bounded replay + consolidation + cleanup + repair.

Conceptual modules:
- lanes/lymphatic/consolidation
- lanes/lymphatic/repair
- lanes/lymphatic/scheduler
- lanes/lymphatic/diagnostics

Required Dream Mode functions:
1. replay the day/session
   - journal events
   - newly created/updated notes
   - state changes
   - loop changes
   - tool traces
   - context snapshots
   - user corrections
   - failures
2. prioritize replay candidates using deterministic triage:
   - salience
   - novelty
   - repetition
   - goal relevance
   - correction value
   - outcome impact
   - contradiction/tension
   - retrieval utility
3. transform replay results into memory tier actions:
   - retain in short-term
   - promote to mid-term
   - promote to long-term
   - merge
   - demote
   - discard
4. cleanup:
   - stale packet fragments
   - duplicate traces
   - low-utility noise
   - resolved low-value details
5. repair:
   - vector/index integrity
   - snapshot hygiene
   - contradiction queues
   - provider cooldown recovery
   - diagnostics reports
6. update restore-related structures:
   - restore scores
   - resume hints
   - snapshot lineage health
   - failure/retrieval utility signals

Dream Mode depths:
- Microdream:
  - short idle window replay
  - top-priority traces only
- Nap Mode:
  - session/day-segment consolidation
  - mid-term updates
- Deep Dream:
  - heavy replay
  - long-term promotion
  - repair jobs
  - adapter-training candidate preparation
  - integrity sweeps

Dream Mode must be auditable and bounded.
No silent self-rewriting.
Durable outcomes must still go through governed commit boundaries.

G. ONLINE LEARNING / ADAPTATION
Implement the safe architecture for learning-in-motion without casual live base-weight mutation.

Allowed forms:
1. always-on memory learning
   - update episodic/semantic/reflective memory
   - update restore utility signals
   - update routing hints
2. background adapter learning scaffolding
   - collect approved training candidates
   - queue adapter-training jobs
   - keep versioning and rollback
3. optional session adaptation scaffolding
   - temporary adaptation artifacts
   - must be bounded and discardable
4. base-model update workflow
   - explicitly offline only
   - versioned promotion/rollback path
   - no hot-path mutation

Do not implement unsafe online base-weight mutation.
Do implement:
- training candidate queues
- adapter candidate metadata
- eval-before-promotion hooks
- rollback-safe promotion model

H. ARTEMIS-LIKE OPERATOR SURFACES
Purpose:
- perception and visibility
- operator trust
- inspection of cognition and action

Implement/extend surfaces for:
- health
- diagnostics
- process/runtime traces
- snapshot inspector
- context inspector
- restore score viewer
- audit/journal explorer
- execution explorer

Conceptual modules:
- api/app
- api/routes/health
- api/routes/diagnostics
- api/routes/process
- api/websockets/server
- CLI / optional operator TUI
- later optional GUI / voice / vision behind flags

Do not enable GUI/audio/vision by default.
Do not make perception a truth source.

PHASED BUILD PLAN

Phase 0 — Reality lock and repo map
Required:
- inventory current authoritative seams
- identify concrete files/modules to extend
- identify any remaining side doors or duplicate authorities
- write a durable implementation note in repo docs before large changes

Phase 1 — Authority convergence
Required:
- harden kernel as sole truth owner
- bound/remove remaining side-write paths
- preserve audit/journal/trace lineage
- align docs and code

Acceptance:
- no new direct truth writes
- legacy paths bounded or documented
- tests prove authority boundaries

Phase 2 — Phase 6.5 restore runtime
Required:
- candidate listing
- deterministic scoring
- header-first restore
- resume hints
- persisted explainable score breakdown
- operator restore traces

Acceptance:
- restore is operational and inspectable
- stale/contradictory candidates penalized
- fresh compile fallback works

Phase 3 — Runtime intelligence
Required:
- role-based model routing
- bounded execution loop
- verifier path
- gateway-mediated tools
- provider cooldown / blacklist / bounded retries
- checkpointed traces

Acceptance:
- no retry storms
- no tool bypass
- no modelruntime bypass
- no durable mutation outside syscalls

Phase 4 — Dream Mode / lymphatic metabolism
Required:
- replay selector
- triage scoring
- tier routing
- demotion/forgetting
- repair jobs
- diagnostics
- dream trace reports

Acceptance:
- Dream Mode improves restore/memory organization
- bounded, logged, dry-run capable

Phase 5 — Operator surfaces
Required:
- snapshot inspector
- restore score viewer
- execution trace viewer
- journal/audit explorer
- process/health endpoints
- websocket updates where appropriate

Acceptance:
- operator can explain what the system did and why

Phase 6 — Learning scaffolds + advanced perception
Required:
- adapter candidate queue
- eval-before-promotion hooks
- optional session adaptation scaffolding
- optional voice/vision/perception under flags

Acceptance:
- no unsafe live base-model mutation
- feature flags keep defaults sane

TEST REQUIREMENTS

Add real tests for:
1. config loading
2. migrations
3. sqlite schema evolution
4. vector DB load/save/rebuild
5. context ranking
6. context budgeting
7. deterministic safe mode
8. shutdown idempotence
9. startup idempotence
10. restore candidate listing
11. restore ranking correctness
12. stale/contradictory penalty behavior
13. header-first restore path
14. fresh-compile fallback
15. gateway-only tool execution
16. provider cooldown / blacklist handling
17. bounded retry behavior
18. Dream Mode replay selection
19. Dream Mode tier promotion/demotion
20. Dream Mode repair/diagnostics dry-run
21. in-memory/SQLite parity on critical context behavior
22. ingest -> context -> runtime proposal -> syscall commit smoke path

DOC REQUIREMENTS

Update or create:
- README.md
- AGENTS.md
- architecture docs for unified runtime
- operations docs
- migration docs
- model management docs
- restore scoring docs
- Dream Mode docs
- memory tier docs
- runtime orchestration docs
- diagnostics/repair docs

WHAT NOT TO DO

- do not rebuild a hidden monolith beside FORGE
- do not let LLMs write truth directly
- do not allow retry storms
- do not default to giant models
- do not enable GUI/audio/vision by default
- do not require DB deletion for migrations
- do not wire cloud providers as default when unconfigured
- do not skip context traceability
- do not leave dead imports or duplicate shutdown logic
- do not use vector retrieval as truth authority
- do not create four co-equal subsystems fighting over truth
- do not hide scoring behind opaque magic numbers
- do not implement Dream Mode as a summarization cron job
- do not implement unsafe live base-weight mutation
- do not output code in chat

DELIVERABLE

Build directly in the repo.

At the end provide:
1. summary of work completed
2. final filesystem tree of changed areas
3. authority convergence summary
4. restore scoring summary
5. runtime orchestration summary
6. Dream Mode / memory-tier summary
7. migration summary
8. run instructions
9. safe mode instructions
10. known follow-up items

EXECUTION STYLE

- work in small, logically grouped commits
- preserve production quality
- prefer extending authoritative seams over inventing duplicate abstractions
- keep code/docs/tests aligned
- when a feature is too large for one pass, land the substrate first but wire it end to end
- keep behavior deterministic outside explicitly probabilistic reasoning layers

Begin.
