# PHASE 4 — Photo-Kinetic Temporal Memory

## Objective

Add snapshot/transition/trace/replay memory without treating snapshots as truth.

## Instructions

Implement PhotoCell, KineticCell, TraceCell, ReplayCell contracts and deterministic reconstruction from base snapshot + deltas. Connect TTL/freshness hooks.

## Validation

PhotoCells are shape, not truth. KineticCells preserve before/action/after. TraceCells are append-only. ReplayCells are proposals.

## What not to do

Do not replay into live state. Do not save giant full snapshots for every tiny change. Do not delete traces.

## Exit gate

Exit when temporal traces can be created and reconstructed deterministically.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
