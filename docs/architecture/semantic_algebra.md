# Semantic Algebra

Status: Phase 5 implementation baseline.

Semantic algebra is the deterministic object and operator model for admitted meaning. It transforms evidence-governed objects without destroying provenance or bypassing semantic syscalls.

Phase 5 implements the minimal Semantic Algebra package in `services/core/internal/forgek/semantic` and kernel-owned semantic operation syscalls in `services/core/internal/forgek`.

## Semantic Objects

| Object | Definition |
|---|---|
| Claim | A scoped assertion under review or committed as canonical state through a syscall |
| Evidence | Source material, artifact, event, or observation offered to support or reject a claim |
| MemoryNode | A durable semantic memory object with scope, provenance, lifecycle state, and links |
| Decision | A committed or admitted choice with rationale and cited evidence |
| Contradiction | A recorded conflict between claims, evidence, decisions, or memory nodes |
| OpenLoop | An unresolved goal, issue, risk, or follow-up requiring closure |
| Goal | A desired outcome with scope and success criteria |
| Constraint | A policy, boundary, requirement, or limit applied to work |
| Artifact | A file, output, report, generated object, or external result used as evidence |
| Snapshot | A shape-preserving record that cites source objects and operation context |
| ContextBlock | A deterministic block emitted by the context compiler for prompt layout |
| CasePacket | A scoped packet containing claims, exhibits, validation records, and requested ruling |
| Ruling | A Courthouse decision on admission, rejection, contradiction, or supersession |
| Precedent | A prior ruling or policy interpretation used as guidance for future cases |
| ConsensusReport | A response/action governance report over proposed claims; not canonical truth |

Phase 5 `SemanticObject` records include workspace scope, object type, source object refs, source refs, content summary, normalized content, confidence, authority level, optional admissibility status, provenance refs, supersession refs, contradiction refs, timestamps, journal refs, and metadata.

SemanticObjects may wrap or reference existing KernelObjects, Exhibits, CandidateObjects, PalaceRoutes, Rulings, NeuronEnvelopes, artifacts, or future memory objects. They prefer references over duplicated content. A SemanticObject is not canonical truth by construction; canonical mutation still requires Kernel-dispatched semantic syscalls.

Consensus aggregation is not `MERGE`, not Courthouse admission, and not canonical mutation. Consensus reports may be referenced by SemanticObjects later, but Semantic Algebra must not treat consensus acceptance as truth without the normal Courthouse and Kernel paths.

## Operators

| Operator | Meaning |
|---|---|
| RETRIEVE | Find scoped candidate objects from Memory Palace |
| SUBMIT | Place a claim, candidate, or proposal into a CasePacket |
| ADMIT | Mark evidence as admissible for context or commit review |
| REJECT | Mark evidence as inadmissible with a deterministic reason |
| MERGE | Combine compatible admitted objects while preserving source links |
| DIFF | Compute differences between semantic objects or snapshots |
| INTERSECT | Identify shared claims, sources, constraints, or routes |
| CONTRADICT | Record a conflict between semantic objects |
| SUPERSEDE | Link an older object to a newer object or ruling without deletion |
| COMPRESS | Produce a smaller shape or summary from cited source objects |
| DERIVE | Produce a derived object from cited source objects |
| PROMOTE | Submit or commit an object to a stronger lifecycle state through valid authority |
| DEMOTE | Move an object to weaker authority, lower confidence, or deferred status |
| EXPIRE | Mark an object stale by policy while preserving inspectability |

## Phase 5 Models

### SemanticOperation

`SemanticOperation` records transformation history. It includes operation id, operation type, workspace, optional case id, input object refs, output object refs, operator version, parameters, reasoning summary, provenance refs, creator, timestamp, journal ref, and metadata.

Operations do not delete input objects and do not silently mutate source objects.

### SemanticTransformResult

`SemanticTransformResult` carries output SemanticObjects, output refs, requested syscalls, warnings, errors, provenance refs, timestamp, and metadata.

Transform results may request syscalls, such as `court.submit`, `court.admit`, `court.register_contradiction`, or `court.register_supersession`, but they do not execute those syscalls.

### Operator Registry

`SemanticOperatorRegistry` registers deterministic operator definitions with operation type, version, determinism, input requirements, output type, canonical mutation flag, syscall requirement, description, and handler.

Unknown operators are rejected. Phase 5 operators are deterministic only.

### SemanticAlgebraService

`SemanticAlgebraService` owns semantic operator dispatch and in-memory simulator records for SemanticObjects and SemanticOperations. It is called by Kernel syscall handlers for canonical operation records. Public reads return copies.

## Phase 5 Implemented Operators

- `MERGE`: combines compatible SemanticObjects into a derived object while preserving source refs. It rejects contradicted inputs unless explicitly allowed.
- `DIFF`: computes deterministic content difference without mutating inputs.
- `INTERSECT`: computes deterministic overlap across source refs and normalized content.
- `CONTRADICT`: creates an explicit contradiction transform and may request `court.register_contradiction`.
- `SUPERSEDE`: creates an explicit supersession transform and may request `court.register_supersession`.
- `COMPRESS`: creates a compressed derived object, preserves source refs, and records that compression cannot create new truth.
- `DERIVE`: creates a derived object that cites source objects.
- `PROMOTE`: produces an explicit authority-change result; it does not bypass Courthouse or Kernel.
- `DEMOTE`: produces an explicit lower-authority result while preserving provenance.
- `EXPIRE`: marks a derived result expired without deleting the source object.
- `RETRIEVE`, `SUBMIT`, `ADMIT`, and `REJECT`: request-only operators for future integration; they do not execute Palace or Courthouse syscalls directly.

No Phase 5 operator calls an LLM, embeddings, vector database, runtime driver, or tool driver.

## Phase 5 Semantic Syscalls

Implemented semantic syscalls:

- `semantic.apply`
- `semantic.merge`
- `semantic.diff`
- `semantic.intersect`
- `semantic.contradict`
- `semantic.supersede`
- `semantic.compress`
- `semantic.derive`
- `semantic.promote`
- `semantic.demote`
- `semantic.expire`
- `semantic.list_operations`
- `semantic.get_operation`

Mutation syscalls require matching actor capabilities and append journal events. Capability names match syscall names.

## Phase 5 Journal Events

Every canonical semantic operation appends a journal event:

- `SEMANTIC_OPERATION_APPLIED`
- `SEMANTIC_MERGE_APPLIED`
- `SEMANTIC_DIFF_APPLIED`
- `SEMANTIC_INTERSECT_APPLIED`
- `SEMANTIC_CONTRADICTION_APPLIED`
- `SEMANTIC_SUPERSESSION_APPLIED`
- `SEMANTIC_COMPRESS_APPLIED`
- `SEMANTIC_DERIVE_APPLIED`
- `SEMANTIC_PROMOTE_APPLIED`
- `SEMANTIC_DEMOTE_APPLIED`
- `SEMANTIC_EXPIRE_APPLIED`

## Courthouse Boundary

Semantic Algebra may transform admitted evidence by reference and may produce contradiction or supersession requests. It does not admit, reject, or supersede Courthouse evidence directly. Rejected evidence is not silently treated as admitted input. Contradictions cannot be silently merged.

## Memory Palace Boundary

Semantic Algebra may transform CandidateObject references as candidates. It cannot make CandidateObjects into Exhibits or admitted evidence. Candidate submission still requires `court.submit`; admission still requires `court.admit`.

## Neuron Fabric Boundary

A future AlgebraNeuron may produce a semantic syscall request or transform envelope, but it must not mutate canonical state, append journal events, or execute syscalls directly. Kernel dispatch and capability checks remain required.

## Invariants

- Provenance is never destroyed.
- Compression cannot create new truth.
- Contradictions cannot be silently merged.
- Derived objects must cite source objects.
- Superseded objects remain inspectable.
- Admitted evidence must be explainable.
- Rejected evidence must record rejection reason.
- Canonical mutation requires semantic syscall.
- Cacheable shape must be deterministic at token level.

## Authority Notes

Semantic algebra may transform admitted meaning, but transformation is not the same as canonical mutation. MERGE, DERIVE, PROMOTE, DEMOTE, SUPERSEDE, and EXPIRE become canonical only when represented as valid semantic syscalls and committed by the Kernel.

## Current Limitations

- Semantic records are in-memory simulator records.
- Operators use deterministic string/ref logic only.
- No full semantic memory graph, algebra optimizer, or policy reasoner exists.
- `RETRIEVE`, `SUBMIT`, `ADMIT`, and `REJECT` are request-only boundaries.
- Snapshots, context compiler, deterministic KV cache, runtime drivers, tool drivers, and FORGE-1 simulator remain future phases.
