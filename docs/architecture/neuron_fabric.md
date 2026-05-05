# Neuron Fabric

Status: Phase 2 implementation baseline.

The Neuron Fabric is FORGE-K's bounded worker fabric. A FORGE neuron is a small, typed, auditable unit that consumes an input envelope and emits an output envelope. Neurons do not receive implicit authority from their implementation language, model provider, or runtime lane.

Existing Rule Cells, Hyperlane routers, Librarian Cells, autonomy workers, maintenance workers, and ModelRuntime adapters should be interpreted through this fabric only when their outputs are wrapped as typed envelopes. This does not give any of those systems durable write authority.

Phase 2 implements the first Neuron Fabric package in `services/core/internal/forgek/neurons`. It is intentionally limited to manifests, envelopes, base neurons, example neural/rule neurons, a scheduler, and a narrow syscall request client.

## Phase 2 Implemented Components

- `NeuronManifest`: validates neuron identity, type, lane, authority defaults, output type, side-effect policy, deterministic rule defaults, and required capabilities.
- `NeuronEnvelope`: immutable-by-convention output record with copy-out accessors, JSON serialization, provenance fields, output payload, validation status, confidence, and requested syscalls.
- `NeuralNeuron`: represented by `SimpleIntentProposalNeuron`, which emits proposal envelopes without model/runtime calls.
- `RuleNeuron`: represented by `RequiredFieldRuleNeuron` and `CaseUpdateRequestRuleNeuron`, which emit validation or explicit syscall-request envelopes.
- `SyscallRequest`: neuron-created request object. It is not execution.
- `NeuronScheduler`: registers, lists, filters, and dispatches neurons by id.
- `KernelSyscallClient`: narrow request port that translates a neuron `SyscallRequest` into the Phase 1 Kernel `DispatchSyscall` path.

Phase 2 does not implement real model drivers, agent loops, Memory Palace retrieval, Courthouse admission, semantic algebra, snapshots, context compilation, KV cache, tool execution, or FORGE-1 simulation.

## Neuron Types

| Type | Definition | Authority |
|---|---|---|
| FORGE neuron | Any bounded unit in the Neuron Fabric | Emits typed envelopes only |
| Neural neuron | Model-backed or heuristic proposer for semantic interpretation | Proposals only |
| Rule neuron | Deterministic validator, classifier, or policy checker | Validations only |
| Memory neuron | Retrieval worker for rooms, anchors, routes, and candidates | Candidate retrieval only |
| Court neuron | Evidence governance worker for cases, claims, exhibits, and rulings | Admission recommendations or rulings under policy |
| Algebra neuron | Deterministic semantic transformation worker | Derived objects with citations |
| Snapshot neuron | Shape capture and restore worker | Non-canonical snapshot records |
| Context neuron | ContextBlock selection, budgeting, and layout worker | Context shape output |
| Cache neuron | KVCacheManifest and cache eligibility worker | Acceleration metadata only |
| Runtime neuron | Runtime-driver adapter around model or accelerator surfaces | Runtime outputs as proposals |
| Motor neuron | Governed action proposal or gateway request wrapper | Proposed action only |
| Lymphatic neuron | Maintenance worker for cleanup, contradiction sweeps, stale-loop detection, cache eviction, and compaction | Maintenance proposals or bounded non-truth cleanup |
| Consensus neuron | Claim governance participant that can produce support, opposition, uncertainty, or validation inputs for Consensus Mesh | Consensus recommendations or quorum evidence only |

## Neuron Lifecycle

Neurons follow the Six-State Lifecycle Model when their output may affect state:

1. Proposed: emit a typed output envelope.
2. Submitted: wrap output in a CasePacket or semantic syscall request.
3. Validated: deterministic rule neurons validate scope, schema, capability, and provenance.
4. Admitted: Courthouse admits or rejects evidence for context or commit review.
5. Committed: Kernel commits only through semantic syscalls.
6. Retired: output is superseded, expired, archived, or contradiction-marked with provenance preserved.

## Input Envelopes

Inputs should include:

- envelope id
- workspace scope
- lane
- source type
- source object ids
- provenance
- correlation id
- trace id
- requested operation
- policy and schema versions
- payload hash where applicable

## Output Envelopes

Outputs should include:

- envelope id
- output type
- lifecycle state
- confidence or validation status where applicable
- cited source ids
- deterministic rejection reasons when rejected
- warnings
- correlation id
- trace id
- audit or journal references when committed

Neuron outputs are typed envelopes. Neural outputs are proposals. Rule outputs are validations. Kernel syscalls produce canonical mutation.

`ProposalEnvelope` and `ValidationEnvelope` are Phase 2 output categories represented by `NeuronEnvelope` records with `output_type=PROPOSAL` or `output_type=VALIDATION`. `SYSCALL_REQUEST` envelopes are explicit request carriers only. A syscall request does not execute itself.

## Authority Boundaries

- Neural neurons cannot commit state, admit evidence, execute tools, or update policy.
- Rule neurons cannot create new truth; they validate or reject submitted material.
- A neuron may request a semantic syscall, but the Kernel decides whether to dispatch it.
- The scheduler cannot mutate canonical state or append journal events directly.
- The narrow kernel request port exposes only syscall submission, not object registry mutation or journal mutation.
- Memory neurons cannot decide admissibility.
- Court neurons cannot bypass Kernel commit paths.
- Snapshot neurons cannot promote snapshots into truth.
- Cache neurons cannot promote KV cache into memory.
- Runtime neurons cannot decide capability, approval, or truth.
- Motor neurons cannot execute outside gateway and approval paths.
- Consensus neurons cannot admit evidence, approve tools, mutate state, override deterministic validators, or make majority output canonical truth.

## Scheduling Expectations

- Hot path: small deterministic checks, direct case opening, compact retrieval, admission for immediate context, and response-critical syscall validation.
- Warm path: broader retrieval, additional rule validation, context compiler expansion/contraction, snapshot creation, and cache eligibility checks.
- Cold path: contradiction sweeps, stale-loop detection, compaction, eviction, replay analysis, and maintenance reporting.

Phase 2 scheduling is basic: register neurons, list neurons, filter by lane, get by id, and dispatch by id. Lane constants are present for Neural Lane, Arterial Lane, Lymphatic Lane, and Hyperlane, but the full lane runtime is deferred.

## Valid Behavior

- A neural neuron proposes a summary with cited source ids.
- A rule neuron rejects a proposal because provenance is missing.
- A rule neuron emits a `SYSCALL_REQUEST` envelope for `case.update`; the Kernel validates capability before mutation.
- The scheduler returns a proposal or validation envelope without changing CasePacket state.
- A memory neuron returns candidate objects from a scoped route.
- A Court neuron admits two exhibits and rejects one exhibit with a reason.
- A context neuron emits deterministic ContextBlocks under a token budget.
- A cache neuron records that a prompt prefix is not cache eligible because tokenizer revision changed.

## Invalid Behavior

- A neural neuron writes canonical memory directly.
- A neuron mutates the object registry or journal directly.
- A `SyscallRequest` updates a CasePacket without Kernel dispatch.
- A scheduler bypasses capability checks.
- A runtime neuron decides that a dangerous action is approved.
- A memory neuron injects candidates into context without Courthouse admission.
- A snapshot neuron treats restored shape as current truth.
- A cache neuron treats a KV hit as proof that memory exists.
- A Motor neuron executes a tool outside gateway policy.

## Phase 3 Courthouse Relationship

Phase 3 consumes Neuron Fabric outputs as case evidence candidates. Neural proposals and rule validations may become submitted exhibits through `court.submit`. Courthouse admission remains a separate authority boundary; envelopes do not admit evidence by themselves.

## Phase 4 Memory Palace Relationship

Phase 4 Memory Palace retrieval is available only through Kernel-dispatched `palace.*` semantic syscalls. A future MemoryNeuron may produce a `SyscallRequest` for `palace.route`, but it must not execute retrieval directly, mutate rooms or anchors, submit evidence, or admit evidence.

`palace.route` returns CandidateObjects with retrieval provenance. CandidateObjects are not Exhibits and are not admitted evidence until submitted through `court.submit` and admitted through `court.admit`.

## Phase 5 Semantic Algebra Relationship

Phase 5 Semantic Algebra is available only through Kernel-dispatched `semantic.*` syscalls. A future AlgebraNeuron may produce a `SyscallRequest` for `semantic.compress`, `semantic.merge`, or another semantic operation, but the request does not execute itself.

Semantic transform results may contain derived objects or requested syscalls. Neurons must not mutate SemanticObjects, SemanticOperations, Courthouse admissibility, Memory Palace topology, the object registry, or the Journal directly.
