# FORGE-K Kernel Simulator

Status: Historical simulator architecture baseline. FORGE-K simulator services under `services/core/internal/forgek` remain `SIMULATOR_ONLY`; Phase 12 shadow diagnostics and Phase 14 live Control Lane validation seams live outside simulator authority.

Current boundary banner:

- `[SIMULATOR-ONLY]` Kernel, Neuron Fabric, Courthouse, Memory Palace, Semantic Algebra, Snapshots, Context Compiler, KV System, Runtime Boundary, Lymphatic Lane, and Consensus Mesh are simulator services unless explicitly migrated through a later live authority phase.
- `[LIVE / READ_ONLY]` Phase 12 shadow diagnostics live in `services/core/internal/forgekshadow` and may observe bounded metadata only when disabled-by-default flags are enabled. They do not make simulator services live authority.
- `[PARTIAL LIVE VALIDATION]` Phase 14 Control Lane seams use live-owned validation actions and shared pure validator packages. They do not import simulator services, admit evidence, compile context, execute semantic mutations, call modelruntime, execute retrieval/search/embeddings, write memory, or grant live Kernel authority.
- `[BLOCKED]` Live Courthouse admission, live Context Compiler prompt authority, governed semantic mutation routing, runtime driver authority, and broader FORGE-K Kernel authority require separate migration design, tests, rollback, and operator gates.

The FORGE-K kernel simulator is the first userspace implementation of the deterministic cognitive microkernel authority skeleton. It is intentionally small and lives in `services/core/internal/forgek`.

The simulator proves the Phase 1 doctrine:

- canonical mutation enters through semantic syscalls
- capability checks gate mutating syscalls
- meaningful state transitions append journal events
- CasePacket lifecycle changes are Kernel-owned
- model or neural proposal output has no canonical authority
- public registry and journal APIs return copies, not mutable internal state

Phase 2 adds a Neuron Fabric above this simulator. Neurons can create typed output envelopes and explicit syscall requests, but the Kernel remains the only commit authority.

Phase 3 adds a minimal Courthouse layer. Exhibits, rulings, contradictions, and supersessions are canonical only when created or changed through Kernel-dispatched Courthouse semantic syscalls.

Phase 4 adds a minimal Memory Palace layer. Rooms, anchors, routes, and candidate objects are canonical only when created or changed through Kernel-dispatched Memory Palace semantic syscalls. CandidateObjects are proposal-authority retrieval records, not admitted evidence.

Phase 5 adds a minimal Semantic Algebra layer. SemanticObjects, SemanticOperations, and transform outputs are canonical operation records only when created through Kernel-dispatched semantic operation syscalls. Transform outputs are proposal-authority derived objects unless later promoted through valid authority paths.

## Phase 1 Scope

Phase 1 implements only the kernel simulator skeleton:

- Kernel object registry
- Semantic syscall registry
- Capability manager
- Append-only journal with simple hash chain
- Basic CasePacket lifecycle
- Strict syscall-owned mutation path
- Deterministic tests with injectable ID and time providers

This is not the full cognitive system.

## Kernel Components

### Kernel

`Kernel` owns the object registry, syscall registry, capability manager, journal, ID provider, and clock. Callers request work through `DispatchSyscall`. The Kernel validates syscall existence, input, mutating authority, and capability scope before executing a handler.

### Object Registry

The object registry stores `KernelObject` records and CasePacket state behind private mutation methods. Public reads return defensive copies so callers cannot mutate canonical state outside Kernel syscall handlers.

Initial object types include:

- `Workspace`
- `CasePacket`
- `Capability`
- `JournalEvent`

Phase 1 implements CasePacket objects only. The other types are defined for contract stability and future phases.

### Syscall Registry

The syscall registry stores `SyscallDefinition` records. Definitions include name, version, lane metadata, determinism, side-effect status, journal requirement, replayability, validator, and handler.

The dispatcher enforces `journal_required` for successful mutating syscalls. A handler marked as journal-required cannot report success unless the append-only journal receives a new event during dispatch.

Implemented Phase 1 syscalls:

- `case.open`
- `case.update`
- `case.close`
- `object.get`
- `object.list`
- `capability.grant`

Implemented Phase 3 Courthouse syscalls:

- `court.submit`
- `court.admit`
- `court.reject`
- `court.rule`
- `court.register_contradiction`
- `court.register_supersession`
- `court.list_exhibits`
- `court.list_rulings`
- `court.list_contradictions`

Implemented Phase 4 Memory Palace syscalls:

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

Implemented Phase 5 Semantic Algebra syscalls:

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

`capability.grant` exists as a registered syscall skeleton. Tests and local setup use a bootstrap capability grant API so Phase 1 can establish an initial actor without adding a root authority model prematurely.

### Capability Manager

The capability manager answers whether an actor can call a mutating syscall in a workspace scope. Phase 1 checks:

- subject id
- allowed syscall
- workspace scope
- mutation scope
- expiration

Delegation and audit flags are captured in the model but not fully enforced yet.

### Journal

The journal is append-only through public APIs. It supports:

- append event
- list events
- get event by id
- simple hash chain using prior event hash

Journal events include syscall name, actor, workspace, case id, object refs, capability refs, input hash, output hash, result, error text, prior event refs, and event hash.

Public journal reads return copies. There is no public delete or update API.

## CasePacket Lifecycle

Phase 1 supports:

- `OPEN`
- `UPDATED`
- `CLOSED`

`case.open` creates a CasePacket, stores the associated KernelObject, and journals `CASE_OPENED`.

`case.update` changes allowed metadata and journals `CASE_UPDATED`. It rejects updates to closed cases.

`case.close` marks the CasePacket closed, records `closed_at`, and journals `CASE_CLOSED`.

## Current Limitations

- The simulator is in-memory only.
- Capability bootstrap is intentionally small and not a production root-authority model.
- Simulator records are simulator-local in-memory records unless a phase explicitly says otherwise.
- Simulator Courthouse admission is not live evidence admission.
- Simulator Memory Palace routing and Context Compiler output are not live retrieval, live prompt authority, or live `COMPILE_CONTEXT` authority.
- Simulator KV manifests are acceleration metadata only and do not enable live KV reuse.
- Simulator Runtime Boundary uses deterministic proposal-only drivers and does not call live `modelruntime`.
- Simulator Lymphatic and Consensus outputs are reports/proposals/diagnostics only.
- There are no live tool drivers in FORGE-K.
- The Phase 2 Neuron Fabric scheduler exists, but it is intentionally basic and cannot mutate canonical state directly.

## Phase 2 Neuron Integration Boundary

Neuron Fabric integration uses a narrow request port:

- neuron output is a typed envelope
- a neuron-created `SyscallRequest` is only a request
- `KernelSyscallClient` translates a neuron request into `Kernel.DispatchSyscall`
- the Kernel still validates capabilities and syscall definitions
- the Journal records kernel-dispatched mutations, not neuron output alone

The Neuron Fabric does not receive direct access to the object registry, journal internals, CasePacket mutation methods, or capability mutation paths.

## Phase 3 Courthouse Boundary

Courthouse mutation is syscall-bound:

- `court.submit` creates `SUBMITTED` exhibits.
- `court.admit` is the only Phase 3 path to `ADMITTED`.
- `court.reject` requires a rejection reason.
- contradictions and supersessions are explicit records.
- supersession does not delete old evidence.
- CasePacket exhibit/ruling/contradiction/supersession refs are updated only through Kernel syscall handlers.

## Phase 4 Memory Palace Boundary

Memory Palace mutation is syscall-bound:

- `palace.create_room` creates MemoryRoom topology records.
- `palace.create_anchor` creates MemoryAnchor reference points in existing rooms.
- `palace.route` creates PalaceRoute traces and proposal-authority CandidateObject records.
- CandidateObjects are not Exhibits.
- CandidateObjects are not admitted evidence.
- CandidateObjects can become submitted evidence only through `court.submit`.
- Admission still requires `court.admit`.
- CasePacket palace route and candidate refs are updated only through Kernel syscall handlers.

## Phase 5 Semantic Algebra Boundary

Semantic Algebra mutation is syscall-bound:

- `semantic.merge`, `semantic.diff`, `semantic.intersect`, `semantic.compress`, and `semantic.derive` create journaled operation records and proposal-authority derived objects.
- `semantic.contradict` and `semantic.supersede` produce explicit Courthouse syscall requests but do not execute them.
- `semantic.promote`, `semantic.demote`, and `semantic.expire` preserve provenance and do not delete source objects.
- rejected evidence cannot be silently treated as admitted semantic input.
- CandidateObjects remain candidates after semantic transforms.
- no semantic operation admits evidence directly or bypasses Courthouse.

## Future Phase Extensions

- Historical simulator phases added Neuron Fabric, Courthouse, Memory Palace, Semantic Algebra, Context-Shape Snapshots, deterministic context compilation, Deterministic KV validation, Runtime Boundary, Lymphatic Lane, and Consensus Mesh.
- Phase 12 work adds read-only disabled-by-default shadow diagnostics outside the simulator package. Those diagnostics observe metadata; they do not route live requests through FORGE-K.
- Phase 14 work adds validation-only live Control Lane seams through shared pure packages and existing live authority owners. Those seams validate shapes and authority claims; they do not execute simulator syscalls or grant live mutation authority.
- Future live authority work must migrate one narrow gate at a time and keep simulator services out of live authority until the gate is explicitly designed, tested, documented, and reversible.
