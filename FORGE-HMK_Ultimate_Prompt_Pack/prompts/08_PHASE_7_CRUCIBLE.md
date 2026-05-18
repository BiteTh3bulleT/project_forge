# PHASE 7 — Crucible Truth Refinement

## Objective

Implement validation/refinement layer for claims, contradictions, supersession, provenance, and promotion readiness.

## Instructions

Implement ClaimEnvelope validation, provenance checks, scope checks, contradiction detection, supersession validation, current-state hooks, and decision states.

## Validation

Crucible cannot commit truth. Missing provenance blocks promotion. Contradictions preserved.

## What not to do

Do not implement automatic live promotion. Do not bypass Control Lane. Do not approve cache-only claims.

## Exit gate

Exit when Crucible can refine/reject/promote-to-review claims while non-authoritative.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
