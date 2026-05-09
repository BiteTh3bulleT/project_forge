# Phase 13I Store Cutover Readiness Review

Status: implemented.

Scope: `DOCS_ONLY / READINESS_REVIEW`.

Date: 2026-05-09.

## Summary

Phase 13I reviews the storage work from Phase 13A through Phase 13H and records the cutover decision.

Decision: no canonical store cutover is approved yet.

SQLite remains the live default. Postgres, Qdrant, and Redis are present as foundations or disabled optional infrastructure, but they are not ready to become live authority for memory, retrieval, jobs, gateway, modelruntime, audit, settings, or FORGE-K canonical state.

## Current State

| Component | Current role | Cutover readiness | Decision |
| --- | --- | --- | --- |
| SQLite | Current live store through existing `store.Open` path | Operational and default | Keep as live default. |
| Postgres | Foundation migrations, metadata tables, disabled diagnostic persistence | Foundation-ready, not canonical-ready | Do not switch live reads/writes. |
| Qdrant | Disabled shadow vector adapter and shadow index scaffold | Adapter-ready, not retrieval-ready | Do not wire into live retrieval. |
| Redis | Disabled ephemeral coordination adapter scaffold | Contract-ready, not live-queue-ready | Do not wire live jobs, cache, gateway, or modelruntime. |
| Shadow diagnostics | In-memory by default; optional Postgres diagnostic persistence | Diagnostic-only | Keep non-authoritative. |

## Readiness Findings

Postgres has backend config, capability contracts, migration runner scaffolding, idempotent foundation migrations, and optional integration gating. That is enough for controlled diagnostic persistence and future repository parity work. It is not enough for a canonical switch because live repository adapters, dual-write comparison, table-level parity, backup/restore rehearsal, and rollback runbooks are still missing.

Qdrant has a safe shadow vector adapter and rebuildable shadow index boundary. It accepts already-produced vectors and safe refs only. It cannot execute retrieval, create embeddings, admit evidence, write memory, or change result ranking. It is not ready for live retrieval authority.

Redis has explicit ephemeral role contracts, safe key policy, TTL requirements, fake adapter tests, and optional integration gating. It is not ready for live queues or caches because no durable recovery path, replay strategy, operational runbook, or live queue parity layer has been approved.

## Cutover Blockers

- Live repository adapters do not yet have SQLite/Postgres behavior parity for jobs, settings, approvals, audit-adjacent records, memory, retrieval metadata, embeddings metadata, or FORGE-K persistence.
- There is no canonical dual-write phase with checksum comparison.
- There is no read-switch phase with operator rollback.
- There is no approved backup/restore rehearsal for Postgres canonical tables.
- Qdrant remains shadow-only and cannot be treated as evidence, admissibility, memory, or truth.
- Redis remains ephemeral-only and cannot be the sole record for jobs, settings, audit, memory, or provenance.
- FORGE-K canonical persistence is not yet designed as a live authority migration.

## Required Gates Before Any Store Cutover

1. Add table-specific repository parity tests for the selected table group.
2. Add a dual-write design and disabled-by-default implementation for the selected table group.
3. Add deterministic checksum comparison between SQLite and Postgres writes.
4. Add backup and restore runbooks for the selected Postgres tables.
5. Add rollback tests proving SQLite can remain or return as the live source.
6. Add operator-visible config and startup diagnostics for backend selection.
7. Prove Qdrant indexes are rebuildable from durable relational records before any live retrieval use.
8. Prove Redis loss is safe before any live queue/cache use.
9. Keep all public APIs, routes, gateway behavior, modelruntime behavior, retrieval behavior, and memory semantics unchanged unless a separate approved phase changes them.

## Recommended Next Storage Phase

Recommended next phase: Phase 13J, `LIVE_INFRA / DUAL_WRITE_DESIGN_ONLY`.

Phase 13J should select one low-risk table group and design a disabled-by-default dual-write path. The recommended first group is operational metadata, not canonical semantic memory:

- storage backend metadata,
- migration audit,
- non-authoritative diagnostic summaries,
- or another small operational table with clear rollback.

Do not start with semantic memory, retrieval authority, embeddings authority, audit authority, gateway records, modelruntime state, or FORGE-K canonical persistence.

## What Not To Do

- Do not make Postgres the default store.
- Do not dual-write canonical live data without a separate phase.
- Do not switch live reads to Postgres.
- Do not wire Qdrant into live retrieval.
- Do not generate embeddings from the Qdrant adapter.
- Do not wire Redis into live jobs, gateway, modelruntime, retrieval, memory, or public APIs.
- Do not use Redis for canonical truth, durable memory, provenance, audit, settings, or sole job records.
- Do not make shadow diagnostics authoritative.
- Do not use storage infrastructure cutover as FORGE-K live authority migration.
- Do not change routes, public APIs, gateway behavior, modelruntime behavior, retrieval behavior, or memory semantics.

## Validation

This phase is a readiness review only. It adds no runtime behavior and no tests. Validation should prove existing boundaries still pass:

- `cd services/core && go test ./internal/storagebackend/... ./internal/store/... ./internal/config/... ./internal/forgekshadow/... ./internal/vectorstore/... ./internal/ephemeral/...`
- `cd services/core && go test ./internal/forgek/...`
- `npm run build:core`
- `npm run lint`
- `npm test`
- `npm run test:integration:env`
- `docker compose config`
- `git diff --check`
