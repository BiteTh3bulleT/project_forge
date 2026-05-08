# Live Integration Reality Check

Status: Phase i1 reality alignment.

Date: 2026-05-08.

## Executive Summary

FORGE-K remains mostly simulator authority. The live daemon still uses existing AI-OS, gateway, permissions, lanes, audit, modelruntime, retrieval, embeddings, memory, and API authority paths.

Phase i1 adds one real live integration seam: deterministic KV identity validation now lives in the shared pure package `services/core/internal/kvidentity` and is used by both the FORGE-K simulator KV package and the live AI-OS Control Lane `VALIDATE_KV_IDENTITY` semantic syscall.

PhaseI2 adds `[PARTIAL LIVE ENFORCEMENT]`: live `VALIDATE_KV_IDENTITY` now runs through an explicit Control Lane enforcement policy that classifies accepted, rejected, malformed, and unsupported live-reuse claims, records structured audit fields, and increments internal counters.

This is validation-only. It does not enable live KV reuse, backend cache reuse, modelruntime mutation, tokenizer-specific token identity, public routes, public APIs, or FORGE-K live authority.

## Before This Pass

- `[SIMULATOR-ONLY]` FORGE-K Phase 8 KV identity gates existed under `services/core/internal/forgek/kv`.
- `[RESEARCH]` Rust fixture validation covered KV manifest validation and parity fixtures.
- `[LIVE]` The live daemon had no Control Lane syscall for validating KV identity gates.
- `[LIVE]` The live daemon did not use FORGE-K KV service, FORGE-K Kernel syscalls, or runtime KV reuse.

## What Changed

- `[LIVE]` Added `VALIDATE_KV_IDENTITY` to the AI-OS Control Lane registry as a non-mutating semantic syscall.
- `[LIVE]` Added the capability `kv.identity.validate`.
- `[LIVE]` Added live acceptance tests proving successful validation, gate mismatch rejection, missing payload rejection, future-IRIS capability denial, no semantic memory mutation, and deterministic idempotency replay.
- `[PARTIAL]` Extracted deterministic KV identity gate validation into `services/core/internal/kvidentity`.
- `[SIMULATOR-ONLY]` Refactored `services/core/internal/forgek/kv` to call the same shared validator while preserving simulator-owned KV service behavior.

## What Did Not Change

- `[SIMULATOR-ONLY]` FORGE-K `KVService`, KV syscalls, cache tier metadata, invalidation, eviction, lookup, registration, and hit/miss accounting remain simulator-only.
- `[LIVE]` Live modelruntime behavior is unchanged.
- `[LIVE]` Live gateway behavior is unchanged.
- `[LIVE]` Live API routes and public APIs are unchanged.
- `[LIVE]` Live retrieval, search, embeddings, RAG, and memory mutation behavior are unchanged.
- `[LIVE]` No real KV tensors are stored or reused.
- `[LIVE]` No backend cache is consulted.
- `[LIVE]` No live prompt compilation is routed through FORGE-K Context Compiler.
- `[LIVE]` PhaseI2 does not add public diagnostics routes or export metrics outside the live process.

## Current Live Status

| Surface | Status | Notes |
| --- | --- | --- |
| AI-OS Control Lane KV identity validation | `[LIVE] / VALIDATION_ONLY` | `VALIDATE_KV_IDENTITY` validates deterministic identity gates and fails closed on mismatches. |
| Shared KV identity gate logic | `[PARTIAL]` | Used by live Control Lane validation and simulator KV package. |
| FORGE-K KV service | `[SIMULATOR-ONLY]` | Still owns simulator manifests, lookups, tiers, and hit/miss metadata only. |
| Runtime KV reuse | `[FUTURE]` | Requires tokenizer-specific token IDs, runtime-driver identity capture, backend cache wiring, and explicit authority tests. |
| Public diagnostics/API route | `[FUTURE]` | No route added in this pass. |

## Acceptance Evidence

Required behavior covered by tests:

- actor/source without the right capability cannot validate
- malformed payloads reject before commit
- gate mismatches reject and do not persist idempotency success
- successful validation records acceleration-only state summary
- validation result explicitly reports no memory mutation, no runtime mutation, and no live KV reuse
- simulator KV and live Control Lane use the same deterministic gate logic

## Remaining Ambiguity

- `[FUTURE]` Tokenizer-specific final token IDs are still not available in live validation, so Phase i1 continues to use `token_input_hash` as the identity placeholder.
- `[FUTURE]` Explicit SQLite journal evidence for the new syscall can be added if a future phase needs storage-level journal assertions beyond Control Lane audit and no-mutation acceptance tests.
- `[FUTURE]` Live modelruntime runtime-assumption capture remains a prerequisite before any backend KV reuse can be considered.

## Not Authorized

- Do not route live KV cache reuse through FORGE-K.
- Do not call modelruntime from the KV validator.
- Do not treat KV identity validation as memory, evidence admission, or canonical truth.
- Do not add public routes for KV validation without a separate API design.
- Do not turn simulator `KVService` into live daemon authority without a scoped live integration phase and tests.
