# PHASE 3 — FORGE-HMK Core

## Objective

Implement read-side memory kernel for cells, synapses, and non-canonical memory artifacts.

## Instructions

Implement MemoryCell/Synapse repositories or stubs, read-only adapters, scoped activation, traversal caps, provenance, non-canonical artifacts, and tests.

## Validation

Reads are scoped. Traversal is bounded. Outputs include provenance. Existing memory path unchanged.

## What not to do

Do not replace existing memory/retrieval. Do not promote cell results to truth.

## Exit gate

Exit when scoped non-canonical memory artifacts can be assembled safely.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
