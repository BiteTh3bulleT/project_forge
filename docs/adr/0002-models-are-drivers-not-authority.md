# ADR 0002 - Models Are Drivers, Not Authority

Status: Accepted

Date: 2026-05-03

## Context

FORGE uses model runtimes for inference and may use neural neurons for interpretation, drafting, ranking, or summarization. Model output is useful, but it is not deterministic authority and must not bypass validation, admission, approval, capability, or journal boundaries.

## Decision

LLMs/model runtimes are external drivers. Their outputs are proposals and must pass through validation/admission/commit paths before affecting canonical state.

Neural neurons propose. Rule neurons validate. Courthouse admits. Kernel commits.

## Consequences

- Model output must be wrapped in typed proposal envelopes.
- Runtime drivers must record model id, model revision, tokenizer information, prompt layout version, and provenance where relevant.
- Canonical mutation remains impossible without semantic syscalls.
- Capability and approval decisions remain deterministic Kernel or gateway concerns.

## Alternatives considered

- Model-owned memory: rejected because it is not replayable or auditable as canonical state.
- Model-decided tool execution: rejected because capability, approval, and risk controls must be deterministic and inspectable.
- Direct model writes to stores: rejected because this bypasses provenance, validation, and journal requirements.
