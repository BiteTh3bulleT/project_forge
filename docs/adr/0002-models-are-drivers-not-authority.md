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
- A worker asked to format an already selected tool proposal must advertise and
  demonstrate structured tool-call output for the active runtime protocol.
  Completion-only workers may still answer text, but model capability never
  participates in deciding whether a tool is needed or which tool is selected.
- Runtime thinking/reasoning controls and output budgets may be bounded so a
  small worker can emit the proposal envelope; this does not grant execution or
  commit authority to that worker.
- FORGE decides whether a turn needs a tool and selects the exact tool before any
  model call. If FORGE cannot select exactly one tool, no tool schema is exposed
  to the model. When FORGE does select one, the model-facing catalog contains
  only that schema so the worker can format bounded arguments. The full gateway
  manifest remains the authority record, and only the gateway may validate or
  execute the resulting proposal.

## Alternatives considered

- Model-owned memory: rejected because it is not replayable or auditable as canonical state.
- Model-decided tool selection or execution: rejected because intent routing, capability, approval, and risk controls must be deterministic and inspectable.
- Direct model writes to stores: rejected because this bypasses provenance, validation, and journal requirements.
