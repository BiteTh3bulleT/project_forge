# FORGE Retrieval Pipeline

## Overview

Retrieval is a persisted pipeline, not a single search call.

Stages:
1. Scope resolution
2. Candidate generation
3. Hybrid scoring
4. Structural/usefulness reranking
5. Optional VSA additive scoring
6. Coverage-aware packet selection
7. Run/result persistence
8. Observation linking

## 1) Scope Resolution

Inputs:
- query
- mode (`keyword`, `semantic`, `hybrid`)
- dossier id (optional)
- explicit source ids (optional)

When dossier is provided and source ids are omitted, FORGE derives source scope from `dossier_sources`.

## 2) Candidate Generation

- Keyword candidates from FTS (`search.SearchScoped`)
- Semantic candidates from embeddings (`embeddings.SemanticSearch`)
- Aggregated by chunk id

## 3) Hybrid Scoring

Base scoring:
- keyword-only: keyword score
- semantic-only: semantic score
- hybrid: weighted keyword + semantic

Weights default from settings and can be overridden per run.

## 4) Reranking

FORGE applies reranking signals:
- historical usefulness by path
- dossier file bias (`high_value_files` boost, `noisy_files` penalty)
- small recency/consensus boost when both retrieval modes agree

## 5) Optional VSA Additive Scoring

When `retrieval_vsa_mode` is `shadow` or `active`, FORGE computes inspectable per-result components:
- associative score (query vs observation pointer)
- role-match score (query token overlap vs role/filler bindings)
- relational score (candidate graph neighborhood support)
- feedback score (usefulness-derived reliability)

VSA additive score is clamped by `retrieval_vsa_max_additive`.
- `shadow`: scores are persisted for inspection but not applied.
- `active`: additive contribution is applied to final ranking.

## 6) Coverage-Aware Selection

Packet selection avoids simple top-N duplicates.

Selection strategy:
- first pass prefers unique file-path coverage
- second pass fills remaining slots by score order

This reduces redundant packet context.

## 7) Persistence

Stored tables:
- `retrieval_runs`
- `retrieval_results`
- `retrieval_result_selection`
- `retrieval_result_vsa_signals` (optional per-result VSA evidence)
- `packet_retrieval_runs`

## 8) Observation Linkage

For each persisted retrieval result, FORGE records/updates:
- `memory_observations` row (type `retrieval_result`)
- `retrieval_result_observations` relation

## Operator Inspection

UI surfaces:
- `#/retrieval-runs`: run list, base/VSA score breakdown, selection reason JSON, usefulness actions
- `#/memory`: observation browser, VSA detail inspector, VSA reindex runs
