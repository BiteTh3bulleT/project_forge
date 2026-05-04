# FORGE-K Architecture Overview

Status: Phase 9 runtime driver boundary implemented; Phase 1-9 simulator implementation baseline.

FORGE-K is a deterministic cognitive microkernel for governed semantic work. It owns canonical truth through semantic syscalls, deterministic validation, journaled commits, and replayable evidence. Model runtimes are drivers attached to the operating system; they may propose interpretations, actions, or text, but they do not own truth authority.

## Why FORGE-K Exists

FORGE already separates jobs, packets, approvals, artifacts, adapters, and semantic syscalls. FORGE-K makes that separation the permanent kernel doctrine. The goal is to prevent agent loops, raw conversations, caches, or model outputs from becoming implicit state authority.

FORGE-K exists to provide:

- deterministic commit boundaries for semantic state
- evidence-governed memory instead of raw chat replay
- isolated runtime drivers instead of model-owned operation
- small hot paths for low-latency decisions
- replayable journal history for meaningful state transitions
- a roadmap from userspace simulation toward future hardware/software co-design

## Kernel-First Thesis

The Kernel is the only component that commits canonical state. Every proposed state change becomes a semantic syscall request, passes deterministic validation, and is either rejected, admitted for commit, or committed with journal evidence. Kernel-space correctness must not depend on live model behavior.

## Model-as-Driver Principle

Models are runtime drivers. A model may classify, summarize, draft, rank, or propose. A model must not directly mutate memory, bypass admission, decide capability authority, or create canonical truth. Driver output is converted into typed proposal envelopes and must pass rule validation, Courthouse admission, and Kernel commit paths.

## Runtime Driver Boundary

Phase 9 implements the Runtime Driver Boundary as `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`. The boundary covers `RuntimeDriver`, driver and capability manifests, generate request/result envelopes, registry/service behavior, syscalls, capability gates, journaled events, and deterministic mock-driver testing. Runtime drivers may receive ContextBundle refs, canonical prompt text, and KV metadata refs, but their output is proposal evidence only.

Phase 9 does not wire FORGE-K into the live daemon, replace live `modelruntime`, call real model backends, change public APIs or gateway behavior, or perform live KV reuse. Real backend integration requires a later explicit `LIVE_INTEGRATION` phase.

## Three-Domain Control Model

FORGE-K separates Truth, Shape, and Acceleration.

| Domain | Purpose | Examples | Authority |
|---|---|---|---|
| Truth domain | Canonical state and meaningful transitions | semantic syscall commits, journal records, rulings, current state | Kernel |
| Shape domain | Deterministic restorable structure | snapshots, context blocks, route summaries, restore hints | Non-canonical unless committed |
| Acceleration domain | Exact reusable compute artifacts | KV manifests, token hashes, cache tiers | Never canonical memory |

Truth is committed. Shape is cited. Acceleration is validated and disposable.

This model is not the same as the lane model. Domains define authority over information. Lanes define scheduling and responsibility. For example, the Arterial Lane may operate on Truth, Shape, and Acceleration metadata, but only Kernel semantic syscalls can commit Truth.

## Six-State Lifecycle Model

Semantic work moves through six states:

1. Proposed: a neural neuron, runtime driver, rule cell, user action, or adapter emits a typed proposal.
2. Submitted: the proposal is wrapped in a CasePacket or semantic syscall request.
3. Validated: rule neurons and deterministic validators check schema, scope, capability, provenance, and policy.
4. Admitted: the Courthouse decides which evidence may enter context or commit review.
5. Committed: the Kernel performs the semantic syscall transaction and journals the result.
6. Retired: later evidence supersedes, expires, archives, or marks the object as contradicted while preserving history.

Existing loop states such as `open`, `in_progress`, `blocked`, `resolved`, and `archived` are domain-specific state values inside this broader lifecycle. They do not replace the authority lifecycle.

## Tri-Lane System

FORGE-K uses three operating lanes.

- Neural Lane: proposal generation, interpretation, classification, and model-driver outputs.
- Arterial Lane: semantic syscalls, validations, admissions, commits, journal writes, and response shaping.
- Lymphatic Lane: cleanup, contradiction sweeps, stale-loop detection, cache eviction, snapshot compaction, and maintenance reports.

The hot path stays small. The full architecture does not run on every turn.

## Hyperlane Overlay

Hyperlane is the deterministic reflex overlay for obvious routing and bounded checks. It is CPU-local, low latency, advisory, and non-authoritative. Hyperlane may reduce model calls or defer work, but it cannot bypass Kernel, Courthouse, approval, capability, scope, or journal rules.

## Neuron Fabric

The Neuron Fabric is a typed worker fabric. Neurons are small bounded units that consume envelopes and emit envelopes. Neural neurons emit proposals. Rule neurons emit validations. Memory neurons retrieve candidates. Court neurons form cases and rulings. Kernel syscalls are the only path to canonical mutation.

## Memory Palace and Courthouse

Memory Palace is the retrieval topology for Evidence-Governed Memory. It organizes rooms, anchors, routes, and candidate objects so relevant meaning can be found. Courthouse is the evidence governance layer. It organizes cases, claims, exhibits, admissibility checks, contradictions, rulings, precedents, and supersession.

Rule: Memory Palace finds candidates. Courthouse decides what enters context.

## Semantic Algebra

Semantic algebra defines typed objects and deterministic operators over admitted meaning. It supports operations such as RETRIEVE, SUBMIT, ADMIT, REJECT, MERGE, DIFF, CONTRADICT, SUPERSEDE, COMPRESS, DERIVE, PROMOTE, DEMOTE, and EXPIRE.

Compression cannot create truth. Derived objects must cite source objects. Contradictions cannot be silently merged.

## Context-Shape Snapshots

Snapshots preserve semantic shape, not truth. They cite source objects and may seed restoration, context compilation, replay, or review. They should store references, hashes, operation records, and summaries rather than duplicating canonical content.

## Expansion/Contraction Context Compiler Loop

The context compiler expands candidate meaning from Memory Palace, contracts it through Courthouse admission and deterministic budget rules, and emits token-addressed ContextBlocks. It creates stable prompt layouts so restored context is inspectable, replayable, and cache-eligible when token identity permits. In the current FORGE codebase, this doctrine aligns with `COMPILE_CONTEXT`, restore scoring, fresh-compile fallback, snapshot persistence, and restore metadata.

## Deterministic KV Cache

KV cache is acceleration, not memory. Reuse requires deterministic identity validation across model, model revision, tokenizer, tokenizer revision, chat template, prompt layout version, policy/syscall schema version, final token IDs, and runtime KV assumptions.

## Journal and Replay

The Journal records meaningful state transitions, including accepted commits, rejected requests, dry runs, supersession, contradiction, and lifecycle changes. Replay must reconstruct truth from journaled transitions and canonical state, not from cache artifacts or raw chat transcripts.

## FORGE-1 CPU Concept Relationship

FORGE-1 is future hardware/software co-design research for an AI-native control processor. It does not replace GPUs. GPUs compute tokens. A future FORGE-1 concept would accelerate governed execution: context layout, snapshot operations, journal integrity, capability checks, lane scheduling, and deterministic KV management.
