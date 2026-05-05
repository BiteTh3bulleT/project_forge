# Consensus Mesh

Status: Phase 11E implemented as `SIMULATOR_ONLY / GOVERNANCE_LAYER_ONLY`.

The FORGE Consensus Mesh is a simulator-only claim governance layer inside FORGE-K. It is not an agent swarm and not a second Kernel. It decides which claims are allowed into a response or action proposal shape, while canonical truth still requires Courthouse admission and Kernel-dispatched semantic syscalls.

Core rule: No Consensus, No Claim.

## Purpose

Consensus Mesh governs claims before they can be composed into a response/action proposal. It validates schema, evidence refs, quorum, conflict state, risk flags, and policy thresholds. Accepted consensus means a claim is allowed for this response or proposal surface. It does not mean the claim is canonical truth.

## Scope

Phase 11E lives under:

- `services/core/internal/forgek/consensus`
- `services/core/internal/forgek/consensus_syscalls.go`

It is simulator-only. It does not wire into the live daemon, add routes, call real model runtimes, execute tools, write live memory, mutate live AI-OS paths, or change gateway/modelruntime/controllane behavior.

## Separation Of Powers

- Runtime drivers may produce proposals.
- Neurons may produce proposal or validation envelopes.
- Consensus Mesh may accept, reject, defer, or mark claims uncertain for response/action governance.
- Courthouse still controls evidence admissibility.
- Semantic syscalls still control canonical mutation.
- Kernel remains the only canonical commit authority.

Consensus decisions are governance records, not truth records.

## Claim Model

A `Claim` records a typed assertion under governance:

- fact
- preference
- decision
- task
- event
- constraint
- recommendation
- inference
- uncertainty
- action proposal
- memory update proposal

Claims require subject, predicate, type, confidence, agent id, request id, and deterministic canonicalization. Factual, recommendation, action, and memory-update claims require evidence refs unless represented as uncertainty or inference.

## Evidence Tiers

Evidence refs are stable and inspectable pointers:

- Tier 1 primary
- Tier 2 derived with source refs
- Tier 3 model inference

Tier 3 model inference cannot be the sole support for factual acceptance.

## Policy, Quorum, And Scoring

`ConsensusPolicy` defines criticality, required agents, evidence requirements, support ratio, conflict ratio, risk handling, and resource ceilings. Phase 11E uses deterministic weighted support:

`agent reliability * evidence quality * freshness * confidence * independence factor`

Defaults:

- Low: one agent, support ratio at least `0.60`, conflict ratio below `0.40`.
- Medium: two agents, stronger evidence, support ratio at least `0.67`.
- High: three agents, Tier 1 required, support ratio at least `0.80`.
- Critical: Tier 1 required, zero conflict, and human confirmation required.

## Conflict Detection

Claims conflict when subject, predicate, scope, and temporal bucket overlap but normalized values differ. Uncertainty claims do not create hard conflicts by default. Conflicts are surfaced as conflicted decisions or escalations; they are never silently hidden.

## Composer Guard

`ComposerGuard` builds an accepted-claims-only payload for future response composition. It can include:

- accepted claims
- uncertain claims as uncertainty blocks
- accepted action proposals
- memory update proposals only as proposals
- style constraints
- current user turn text

It must not include raw proposed claims, rejected claims, unsupported facts, unapproved actions, or uncommitted memory updates as truth.

## Integration By Reference

Consensus Mesh may cite existing FORGE-K objects by ref, including Neuron envelopes, Court exhibits/rulings, Semantic operations, Snapshots, ContextBundles, RuntimeGenerateResults, and Maintenance Reports. It does not mutate those objects.

Future ContextBlocks or Snapshots may cite `ConsensusReport` refs, but context compilation must not treat consensus as admitted truth unless Courthouse admission exists.

## Rust Validator Note

Phase 11E does not add Rust validation for consensus. Consensus fixtures may be added to `crates/forgek-validate` later after the Go consensus model stabilizes.

## What Not To Do

- Do not wire Consensus Mesh into the live daemon.
- Do not add live API routes.
- Do not make consensus a second Kernel.
- Do not auto-admit claims into Courthouse.
- Do not auto-write memory.
- Do not execute actions.
- Do not call model runtimes.
- Do not treat model majority as truth.
- Do not let the composer see rejected claims as factual material.
