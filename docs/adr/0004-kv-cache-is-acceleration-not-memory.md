# ADR 0004 - KV Cache Is Acceleration, Not Memory

Status: Accepted

Date: 2026-05-03

## Context

KV cache can improve model-runtime performance by reusing exact token-prefix compute. That reuse is valuable only when identity is deterministic and all runtime assumptions match.

Cache artifacts are not semantic evidence, admissibility decisions, or canonical state.

## Decision

KV cache is an acceleration artifact. It is never canonical memory and reuse requires deterministic identity validation.

FORGE-K requires the Nine-Gate Deterministic KV Validation before reuse: same model, model revision, tokenizer, tokenizer revision, chat template, prompt layout version, policy/syscall schema version, final token IDs, and runtime KV assumptions.

## Consequences

- Cache hits cannot bypass context compilation, admission, policy, or semantic syscalls.
- Cache misses must fall back safely.
- Cache manifests must capture enough identity metadata for deterministic validation.
- Lymphatic Lane may evict or compact cache artifacts without changing canonical memory.

## Alternatives considered

- Treat KV cache as long-term memory: rejected because KV is backend compute state, not evidence-governed semantic state.
- Reuse based on source text equality: rejected because final token IDs and runtime assumptions can differ.
- Backend-only validation: rejected because FORGE-K must preserve its own identity and authority boundaries.
