# Authority And Truth Model

## Truth Classes

| Class | Meaning | Examples | Status |
|---|---|---|---|
| Canonical truth | Current committed semantic state | `state_items`, active `memory_notes`, open loops | IMPLEMENTED |
| Historical truth | Append-only or retained history | `journal_events`, `state_versions`, supersession/contradiction records | IMPLEMENTED |
| Non-canonical evidence | Evidence for review/restore/reasoning | artifacts, context snapshots, Dream reports, tool results | PARTIAL |
| Retrieval support | Indexes and rankings | embeddings, VSA, retrieval results | PARTIAL |
| Derived models | Advisory/evidence-backed models | `derived_models` | PARTIAL |

## Authority Table

| Operation | Authority | Status |
|---|---|---|
| Semantic memory/state mutation | Semantic syscall control lane | IMPLEMENTED |
| Tool execution | Gateway | IMPLEMENTED |
| Inference execution | Modelruntime | PARTIAL / IMPLEMENTED M3 |
| Approval decision | Approvals service/operator API | IMPLEMENTED |
| Audit record | Audit service | IMPLEMENTED |
| Context restore evidence | `COMPILE_CONTEXT` syscall path | PARTIAL |
| Dream consolidation | Dream Mode dry-run report | PARTIAL |

## Examples

### LLM Proposes Memory

The LLM can emit a candidate note or state update. That proposal must become a `SyscallRequest`, pass validation, capability checks, approval checks when required, transaction commit, journal append, and audit linkage before becoming canonical.

### Gateway Executes Tool

A caller submits a gateway request. The gateway resolves lane, tool, paths, policy, permission profile, risk, execution level, approval status, and audit context. Tool output is evidence, not automatic memory truth.

### Dream Mode Proposes Promotion

Dream Mode v0 reads existing evidence and emits a dry-run consolidation report. It does not commit canonical memory. Any future commit mode must submit semantic syscalls.

### Kernel Commits State

The control lane transaction updates current projection and appends history. Failures return deterministic rejection states.

## Current Gaps

- PARTIAL: Model management mutates runtime registry state outside gateway-equivalent approval.
- PARTIAL: Operator trace workflow does not yet make all authority edges obvious.
- PARTIAL: Public semantic write API is missing; internal kernel path exists.

