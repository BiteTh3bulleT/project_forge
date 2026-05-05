# Memory Palace and Courthouse

Status: Phase 4 implementation baseline.

Memory Palace and Courthouse are separate systems. Memory Palace retrieves candidate meaning. Courthouse governs evidence admissibility.

Rule: Memory Palace finds candidates. Courthouse decides what enters context.

## Memory Palace

Memory Palace is the retrieval topology for Evidence-Governed Memory. It is not canonical truth by itself. It organizes structured references so FORGE-K can find relevant candidates without treating retrieval results as admitted evidence.

Phase 4 implements the minimal Memory Palace package in `services/core/internal/forgek/palace` and kernel-owned Memory Palace syscalls in `services/core/internal/forgek`.

### Rooms

Rooms are scoped retrieval regions. A room may represent a workspace, project, subsystem, topic, artifact family, decision history, or operational lane.

Phase 4 `MemoryRoom` records include workspace scope, stable name, description, domain tags, anchor refs, linked room refs, route stats, timestamps, journal refs, and metadata. Rooms are retrieval topology objects, not truth authorities.

Rooms must carry:

- stable id
- workspace scope
- kind
- policy constraints
- provenance for room creation or update

### Anchors

Anchors are stable reference points inside rooms. They connect a query, claim, artifact, decision, or route to candidate objects.

Phase 4 `MemoryAnchor` records include workspace scope, room id, label, object refs, keywords, tags, source refs, optional future embedding ref, timestamps, journal refs, and metadata. Anchors help retrieval and do not admit evidence.

Anchors may include:

- object ids
- source hashes
- route weights
- freshness metadata
- contradiction indicators

### Routes

Routes are deterministic retrieval paths through rooms and anchors. They are shape records. Routes may be scored, compacted, or superseded, but route existence does not admit evidence.

Phase 4 `PalaceRoute` records include case/workspace scope, query or route reason, start room, visited rooms, anchor refs, candidate objects, route score, route strategy, creator, timestamp, journal refs, result records, and metadata. Routes record explainable retrieval traces.

### Candidate Objects

Candidate objects are retrieved items that may become exhibits. Candidate objects must retain source ids, provenance, scope, confidence or score metadata, and retrieval reason.

Phase 4 `CandidateObject` records include workspace scope, source object id, source type, source refs, anchor id, room id, relevance score, retrieval reason, candidate summary, timestamp, and metadata. A CandidateObject is not an Exhibit and is not admitted evidence.

### Deterministic Route Scoring

Phase 4 uses deterministic scoring only. It considers keyword overlap, anchor label overlap, tag overlap, start room match, prior route success count, and workspace match.

Cross-workspace candidates score `0`. No embeddings, vector database, model ranking, or runtime driver calls are used in Phase 4.

## Phase 4 Memory Palace Service

`MemoryPalaceService` owns in-memory retrieval topology records for the simulator and is called only by Kernel syscall handlers for canonical mutation. It supports:

- create room
- update room
- link rooms
- create anchor
- update anchor
- link anchor to object refs and source refs
- route
- record route result
- list rooms, anchors, and routes
- get room, anchor, and route

The service does not bypass the Kernel. Public reads return copies.

## Phase 4 Semantic Syscalls

Implemented Memory Palace syscalls:

- `palace.create_room`
- `palace.update_room`
- `palace.link_rooms`
- `palace.create_anchor`
- `palace.update_anchor`
- `palace.link_anchor`
- `palace.route`
- `palace.record_route_result`
- `palace.list_rooms`
- `palace.list_anchors`
- `palace.list_routes`
- `palace.get_room`
- `palace.get_anchor`
- `palace.get_route`

Mutation syscalls require matching actor capabilities. `palace.route` creates a journaled retrieval trace and proposal-authority CandidateObject records. If a route is linked to a CasePacket, the case must be open.

Capability names match syscall names:

- `palace.create_room`
- `palace.update_room`
- `palace.link_rooms`
- `palace.create_anchor`
- `palace.update_anchor`
- `palace.link_anchor`
- `palace.route`
- `palace.record_route_result`

## Phase 4 Journal Events

Every Memory Palace mutation appends a journal event:

- `MEMORY_ROOM_CREATED`
- `MEMORY_ROOM_UPDATED`
- `MEMORY_ROOMS_LINKED`
- `MEMORY_ANCHOR_CREATED`
- `MEMORY_ANCHOR_UPDATED`
- `MEMORY_ANCHOR_LINKED`
- `PALACE_ROUTE_CREATED`
- `PALACE_ROUTE_RESULT_RECORDED`

The Journal is the authority record for these retrieval topology transitions.

## Courthouse Boundary

`palace.route` returns CandidateObjects. CandidateObjects are route results, not Exhibits. They are not admitted evidence and do not enter final context by default.

A CandidateObject may be submitted to the Courthouse through `court.submit`. The resulting Exhibit still begins as `SUBMITTED` and still requires `court.admit` before it becomes `ADMITTED`.

## Semantic Algebra Boundary

Phase 5 Semantic Algebra may consume CandidateObject, Exhibit, Ruling, Contradiction, Supersession, and PalaceRoute refs as SemanticObjects. It transforms references and summaries while preserving provenance.

Semantic operations do not admit evidence, reject evidence, or mutate Courthouse admissibility state directly. Contradiction and supersession operations may produce Courthouse syscall requests, but the Kernel and Courthouse must authorize and execute those requests.

## Consensus Mesh Boundary

Phase 11E Consensus Mesh may cite CandidateObjects, Exhibits, Rulings, Contradictions, PalaceRoutes, SemanticOperations, RuntimeGenerateResults, and other FORGE-K objects by reference. A consensus report may later be submitted as an exhibit or used as ruling input, but only through Courthouse syscalls such as `court.submit` or `court.rule`.

Consensus accepted does not mean admitted evidence. Courthouse remains the admission authority, and rejected evidence cannot become admitted merely because consensus supported a related claim.

## Courthouse

Courthouse is the evidence governance layer. It decides which candidates become admitted context or eligible commit evidence.

Phase 3 implements the minimal Courthouse package in `services/core/internal/forgek/court` and kernel-owned Courthouse syscalls in `services/core/internal/forgek`.

### Cases

A case is a scoped review container opened for a user request, maintenance task, syscall proposal, contradiction review, or context compilation pass.

### Claims

Claims are assertions under review. A claim must cite sources or explicitly state that it is proposed and unsupported.

### Exhibits

Exhibits are candidate objects submitted to a case. Exhibits are not admitted until the Courthouse rules on them.

Phase 3 `Exhibit` records include case/workspace scope, source object id, submitted actor, source type, source refs, claim refs, content summary, optional raw ref, admissibility status, admission or rejection reason, contradiction refs, supersession refs, timestamps, journal refs, and metadata.

New exhibits begin as `SUBMITTED`. Evidence cannot become `ADMITTED`, `REJECTED`, `CONTRADICTED`, or `SUPERSEDED` through retrieval or neuron output alone.

### Admissibility

Admissibility checks include:

- scope match
- provenance present
- source inspectable
- policy compatible
- contradiction status known
- freshness appropriate for the case
- derivation chain available for derived objects

### Contradictions

Contradictions must be recorded and reviewed. Contradictory objects cannot be silently merged. A contradiction may block admission, require additional exhibits, or create a ruling with explicit uncertainty.

### Rulings

Rulings record what was admitted, rejected, deferred, superseded, or marked contradictory. Rejected evidence must record a rejection reason.

Phase 3 `Ruling` records cite affected exhibits, contradiction refs, supersession refs, reasoning summary, policy refs, creator, timestamp, journal ref, and metadata. Rulings summarize Courthouse decisions; they do not overwrite evidence.

### Precedents

Precedents are prior rulings that may guide future admission. Precedents are evidence and policy guidance, not automatic truth.

### Supersession

Supersession links older objects to newer evidence or rulings. Superseded objects remain inspectable and retain provenance.

## Phase 3 Courthouse Service

`CourthouseService` owns in-memory Courthouse records for the simulator and is called only by Kernel syscall handlers for canonical mutation. It supports:

- submit exhibit
- admit exhibit
- reject exhibit
- create ruling
- register contradiction
- register supersession
- list case exhibits
- list admitted exhibits
- list rejected exhibits
- list case rulings
- list case contradictions

The service does not bypass the Kernel. Public reads return copies.

## Phase 3 Semantic Syscalls

Implemented Courthouse syscalls:

- `court.submit`
- `court.admit`
- `court.reject`
- `court.rule`
- `court.register_contradiction`
- `court.register_supersession`
- `court.list_exhibits`
- `court.list_rulings`
- `court.list_contradictions`

Mutation syscalls require matching actor capabilities and an open CasePacket. Closed cases reject new evidence in Phase 3.

Capability names match syscall names:

- `court.submit`
- `court.admit`
- `court.reject`
- `court.rule`
- `court.register_contradiction`
- `court.register_supersession`

## Phase 3 Journal Events

Every Courthouse mutation appends a journal event:

- `EXHIBIT_SUBMITTED`
- `EXHIBIT_ADMITTED`
- `EXHIBIT_REJECTED`
- `RULING_CREATED`
- `CONTRADICTION_REGISTERED`
- `SUPERSESSION_REGISTERED`

The Journal is the authority record for these transitions.

## Neuron Integration Boundary

A `NeuronEnvelope` may be submitted as an Exhibit source through `court.submit`. The envelope is candidate evidence only. It is not automatically submitted, admitted, or journaled by the Neuron Fabric.

Neural proposals and rule validations can become exhibits only through Courthouse syscalls. No neuron can admit evidence directly.

## Current Limitations

- Full semantic memory graph and non-minimal algebra are not implemented.
- Claims are represented as a minimal model only; claim extraction is deferred.
- Courthouse records are in-memory simulator records.
- Memory Palace records are in-memory simulator records.
- Route scoring is deterministic keyword/tag matching only.
- Embeddings, vector database integration, model ranking, and MemoryNeuron execution are deferred.
- No full adjudication reasoning engine exists.
- Snapshots, context compiler, deterministic KV cache, runtime drivers, tool drivers, and FORGE-1 simulator remain future phases.
