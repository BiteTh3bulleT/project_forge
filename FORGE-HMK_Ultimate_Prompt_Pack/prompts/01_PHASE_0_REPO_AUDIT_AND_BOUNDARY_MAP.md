# PHASE 0 — Repo Audit and Boundary Map

## Objective

Map current FORGE authority, memory, job, context, cache, audit, and runtime boundaries before implementing.

## Instructions

Inspect repo docs/packages/tests. Identify live authority paths, simulator/shadow-only paths, current memory/retrieval paths, test commands, and safe package locations. Create `docs/reviews/forge_hmk_boundary_map.md`.

## Validation

No behavior changes. Boundary map names concrete files/packages. Authority and simulator paths are separated.

## What not to do

Do not create runtime code. Do not rename modules. Do not assume paths without inspection.

## Exit gate

Exit when the repo map shows exactly where FORGE-HMK can be added safely.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
