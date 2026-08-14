# FORGE Glossary

Status: Phase 0 FORGE-K glossary baseline.

## FORGE

The local-first AI workspace and operating substrate for governed engineering work.

## FORGE-K

The deterministic cognitive microkernel architecture inside FORGE. FORGE-K owns canonical truth through semantic syscalls, deterministic validation, and journaled commits.

## FORGE-1

A future AI-native control processor research concept for accelerating governed execution. It does not replace GPUs and is not an immediate implementation requirement.

## Kernel

The authority boundary that validates and commits canonical state through semantic syscalls.

## Semantic syscall

A deterministic request, validation, commit, audit, and result path for canonical semantic mutation.

## Neuron

A bounded worker in the Neuron Fabric that consumes typed envelopes and emits typed envelopes.

## NeuronManifest

A typed declaration of neuron identity, type, lane, authority level, input/output contract, side-effect policy, required capabilities, determinism, policy version, and audit requirement.

## NeuronEnvelope

An immutable-by-convention output record from a neuron. It carries provenance, lane, output type, payload, confidence or validation status, requested syscalls, and metadata.

## ProposalEnvelope

A NeuronEnvelope whose output type is `PROPOSAL`. Neural neurons normally emit proposal envelopes.

## ValidationEnvelope

A NeuronEnvelope whose output type is `VALIDATION`. Rule neurons normally emit validation envelopes.

## NeuronScheduler

A bounded dispatcher that registers neurons, lists neurons, filters by lane, and dispatches a neuron by id without bypassing Kernel authority.

## Neural neuron

A model-backed or heuristic proposer. Its outputs are proposals only.

## Rule neuron

A deterministic validator or classifier. Its outputs are validations only.

## Lane

A scheduling and responsibility boundary. FORGE-K defines Neural, Arterial, and Lymphatic lanes.

## Hyperlane

A deterministic CPU-local reflex-routing overlay for obvious checks and route hints. It is advisory and cannot bypass Kernel authority.

## Memory Palace

The retrieval topology for rooms, anchors, routes, and candidate objects. It finds candidates but does not admit evidence.

## MemoryRoom

A workspace-scoped retrieval topology region with stable name, domain tags, anchor refs, linked room refs, route stats, provenance, and journal refs.

## MemoryAnchor

A stable reference point inside a MemoryRoom. It links labels, keywords, tags, object refs, and source refs for deterministic retrieval.

## PalaceRoute

An explainable retrieval trace through rooms and anchors. It records query or route reason, visited rooms, anchors, CandidateObjects, route score, route strategy, and journal refs.

## CandidateObject

A retrieved candidate reference with source refs, route provenance, score, and summary. A CandidateObject is not an Exhibit and is not admitted evidence.

## Route scoring

Deterministic ranking of MemoryAnchors and CandidateObjects using scoped signals such as keyword overlap, tag overlap, room match, and prior route result stats.

## Courthouse

The evidence governance layer for cases, claims, exhibits, admissibility, contradictions, rulings, precedents, and supersession.

## CasePacket

A scoped review packet containing submitted claims, exhibits, validation records, requested rulings, and provenance.

## Exhibit

A candidate object submitted to a case for admissibility review.

## Claim

A structured assertion extracted from an exhibit. Phase 3 defines a minimal model; full extraction and algebra are deferred.

## Ruling

A Courthouse decision that admits, rejects, defers, marks contradiction, or
records supersession. In the production K20C path, rulings and appeals are
append-only historical truth; an exhibit points to its current ruling without
rewriting prior decisions.

## Contradiction

An explicit record that two exhibits or claims conflict. Contradictions do not delete or silently merge evidence.

## Supersession

A record that an older object has been replaced by a newer object for a stated reason. Supersession is not deletion.

## Semantic algebra

The deterministic object and operator model for transforming admitted meaning while preserving provenance.

## SemanticObject

A typed reference-preserving object used by Semantic Algebra. It may wrap or cite existing KernelObjects, Exhibits, CandidateObjects, Rulings, artifacts, or future memory objects.

## SemanticOperation

A journaled transformation record with operation type, input refs, output refs, operator version, parameters, reasoning summary, provenance, creator, and timestamp.

## SemanticTransformResult

The deterministic output of a semantic operation. It can contain derived objects, output refs, warnings, errors, provenance refs, and requested syscalls, but it does not execute syscalls itself.

## Derived object

A SemanticObject produced from cited source objects. A derived object does not become canonical truth without a valid semantic syscall and journaled commit path.

## Compression

A deterministic reduction of cited source summaries or refs. Compression cannot create new truth.

## Snapshot

A shape-preserving record that cites source objects and operation context. Snapshots are not truth.

## ContextBlock

A deterministic prompt unit emitted by the context compiler with source references, layout metadata, and token-shape data.

## KVCacheManifest

A manifest describing deterministic KV cache eligibility, identity gates, token hashes, runtime assumptions, and cache tier.

## Journal

The append-oriented record of meaningful state transitions, including commits, rejections, dry runs, contradictions, supersession, and lifecycle changes.

## Lymphatic lane

The deferred maintenance lane for cleanup, contradiction sweeps, stale-loop detection, cache eviction, and snapshot compaction.

## Truth domain

Canonical state and committed semantic transitions owned by the Kernel.

## Shape domain

Restorable semantic structure such as snapshots, routes, ContextBlocks, summaries, and prompt layout.

## Acceleration domain

Disposable or reproducible performance artifacts such as KV cache, token hashes, and cache manifests.

## Nix mutation proposal

A governed advisory record that describes a proposed NixOS change, expected build/test commands, rollback posture, evidence refs, and approval refs. It is not permission to mutate the host.

## Modelruntime backend profile

A named runtime posture such as `cpu_safe`, `local_llama_cpp`, `interactive_vllm`, or `embedding_tei`. Profiles describe expected endpoints, resource needs, failure behavior, and safe-mode behavior; they do not grant truth authority.

## VRAM lease

A future FORGE-H resource-governed claim over a bounded GPU memory region. A VRAM lease is acceleration-domain state and cannot become canonical memory.

## System cockpit

The planned read-only workstation status surface for core, authority gates, FORGE-H, HostBridge, modelruntime, storage, Nix, safe mode, approvals, warnings, and build/test posture.

## FORGE graphical shell

The visible operator-facing desktop surface above the NixOS substrate. It owns
Forge chrome, launcher, taskbar, workspaces, and governed operator surfaces but
does not become the Linux kernel, Wayland compositor, package manager, or a new
truth authority.

## Operator desktop

The native multi-window FORGE session. FORGE is a fixed desktop canvas while
Labwc composites native application windows above it and supplies bounded
window lifecycle/control data to the Forge taskbar. “Inside FORGE” means inside
this Forge-owned desktop session, not embedded inside the Tauri webview.

## Offline direct link

A point-to-point test network between the build workstation and a FORGE target
using static local addresses with no gateway, DNS, IPv6, forwarding, or NAT.
It permits bounded local administration and artifact transfer without giving
the target an internet route.

## Native session boundary

The greetd/PAM authentication boundary for a FORGE operator desktop. Native
Lock and Logout exit the compositor session and return to this boundary; a
client-side password comparison is not an OS session lock.

## Structured tool-proposal worker

An external model driver whose active runtime manifest and live protocol probe
support formatting a structured proposal for the one tool FORGE has already
selected. This capability never lets the worker decide whether to use a tool or
which tool to use. The gateway still resolves, authorizes, validates, audits,
and executes any accepted proposal.
