# FORGE - Phase 3 Scope

## Objective

Phase 3 turns FORGE into a memory-aware operator system that can improve future work from stored evidence.

## Delivered

### Retrieval and Embeddings

- embedding provider abstraction with local-first defaults
- embedding persistence (`embedding_records`)
- semantic retrieval endpoint/UI path
- hybrid retrieval with weighted fusion and inspectable score components
- persisted retrieval runs/results with packet/job/dossier link support
- usefulness labeling and evidence records

### Dossiers

- first-class dossier records
- linked sources, jobs, and packets
- dossier detail view in UI
- generated dossier brief snapshots
- dossier-scoped retrieval support

### Evaluation and Comparison

- manual evaluation record creation and persistence
- score dimensions:
  - success/failure
  - quality
  - usefulness
  - correctness confidence
  - packet quality
  - adapter suitability
  - retry recommendation
  - routing influence flag
- adapter comparison metrics by dossier/global scope

### Retries, Replays, and Lineage

- retry endpoint creates child job with override support
- replay endpoint creates child job with original metadata parity
- lineage records persist relation type and change summary
- lineage UI view for parents/children and related jobs

### Imported Execution Memory

- import API/UI for external execution outcomes
- imported records store summary, refs, diff summary, notes, evaluation blob
- imported execution summaries persisted as artifacts and linked to origin job/packet when provided

### Routing Insight Layer

- advisory insight generation endpoint
- stored recommendation records with confidence, reasons, and evidence
- retrieval-noise and adapter-performance signals included
- no autonomous risky routing

### Memory UI Surfaces

- Dossiers
- Retrieval Runs
- Evaluations
- Lineage
- Insights

## Explicit Deferrals to Phase 4

- autonomous routing decisions
- adaptive confidence calibration and experiment frameworks
- advanced packet auto-tuning based on historical deltas
- richer imported execution parsing (structured diff ingestion)
- deeper artifact semantic indexing
