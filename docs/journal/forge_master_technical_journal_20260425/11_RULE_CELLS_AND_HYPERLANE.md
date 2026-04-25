# Rule Cells And Hyperlane

## Design Position

CONCEPT: FORGE should not scale cognition by launching thousands of free-form agents. The safer direction is many small deterministic Rule Cells behind explicit routers and traces.

## Why Rule Cells

Rule Cells are deterministic checks or proposal generators. They can improve latency by handling obvious cases without model calls and improve safety by emitting explainable rule traces.

## Hyperlane Concept

CONCEPT: Hyperlane is sub-model latency reflex routing. It dispatches low-risk, high-frequency cognitive operations to deterministic rule packs before escalating to modelruntime or frontier models.

## Proposed Hyperlanes

| Hyperlane | Purpose | Status |
|---|---|---|
| Neural | Ingest normalization/reflex classification | CONCEPT |
| Arterial | Context selection/planning hints | CONCEPT |
| Kernel | Pre-validation and rejection hints | CONCEPT |
| Lymphatic | Dream/repair candidate routing | CONCEPT |
| Operator | UI attention and review routing | CONCEPT |

## Current Reality

PARTIAL: `aios/autonomy/rule_agents.go` has limited propose-only rule agents. It is not the full Rule Cell substrate.

## Rule Output Types

- proposal
- warning
- score
- route
- denial hint
- review request
- no-op explanation

No Rule Cell may directly mutate canonical truth.

