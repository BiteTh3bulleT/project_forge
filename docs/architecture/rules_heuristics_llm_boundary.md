# Rules, Heuristics, and LLM Boundary

This document defines the responsibility split for FORGE AI-OS.

## Rule-based (kernel) responsibilities

Kernel/rule code is deterministic and authoritative for correctness, authorization, and persistence validity.

- schema validation
- IDs
- timestamps
- event ordering
- allowed state transitions
- permission checks
- approval gates
- supersession/version chains
- archive policies
- workspace boundaries
- audit records
- tool execution boundaries

In current FORGE this responsibility is anchored in `internal/gateway`, `internal/permissions`, `internal/approvals`, `internal/jobs`, `internal/audit`, and persistence contracts in `internal/store`.

## Heuristic responsibilities

Heuristics are advisory ranking/scoring logic between kernel rules and semantic proposals.

- novelty score
- write score
- relevance score
- priority score
- contradiction severity
- stale loop detection
- model promotion/demotion thresholds

Heuristics can influence prioritization and recommendations, but cannot commit state without kernel validation.

## LLM/generative responsibilities

Generative systems are semantic proposers in user space.

- semantic extraction
- subtle relationship discovery
- summarization
- model proposal
- context synthesis
- ambiguous intent interpretation

## Hard rule

No live LLM behavior may be required for kernel correctness, memory integrity, authorization, or persistence validity.

If all LLM providers are unavailable, FORGE kernel correctness and state integrity must remain intact.

