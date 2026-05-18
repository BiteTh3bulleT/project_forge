# PHASE 1 — Contracts and No-Op Shells

## Objective

Create typed contracts and no-op service shells without enabling live mutation.

## Instructions

Add contracts for Job, WorkOrder, Lease, MemoryCell, Synapse, PhotoCell, KineticCell, TraceCell, ReplayCell, HKVManifest, Artifact, ClaimEnvelope. Add interfaces for FORGE-HMK, FORGE-T, HKV, Neuron Mesh, Crucible. Add no-op/in-memory stubs and tests.

## Validation

Contracts compile. No-op services cannot write canonical memory. Existing tests pass.

## What not to do

Do not connect to live mutation. Do not expose public routes. Do not add external service requirements.

## Exit gate

Exit when interfaces compile and tests prove no-effect behavior.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
