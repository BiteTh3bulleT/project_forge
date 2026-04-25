# Design Doctrine

## Invariants

- IMPLEMENTED: Kernel/control lane owns canonical semantic mutation.
- IMPLEMENTED: Gateway owns governed tool execution.
- IMPLEMENTED: Modelruntime owns governed inference execution when enabled.
- IMPLEMENTED: Audit, correlation, trace, provenance, syscall IDs, and gateway invocation IDs are first-class linkage fields.
- PARTIAL: Operator trace inspection exists in API/pages, but the unified human workflow is incomplete.
- DOCTRINE: LLMs, agents, Rule Cells, Dream Mode, and future IRIS may propose but may not directly mutate canonical truth.
- DOCTRINE: Vector retrieval supports recall and ranking only; it is never truth authority.
- DOCTRINE: Context snapshots are restore evidence, not canonical truth.
- DOCTRINE: Dream Mode output is non-canonical unless later committed through semantic syscalls.
- DOCTRINE: CPU/RAM core must survive GPU/modelruntime failure.

## What FORGE Refuses To Do

- It refuses to treat model output as durable truth.
- It refuses to let tool execution bypass gateway policy.
- It refuses to promote vector similarity to authority.
- It refuses to let Dream Mode silently consolidate long-term memory.
- It refuses to require GPU/modelruntime for kernel boot.
- It refuses to silently default to cloud providers when unconfigured.
- It refuses hidden monoliths and duplicate truth systems.

## Current Doctrine Gaps

- PARTIAL: Some authority-adjacent operational state changes need stronger approval policy.
- PARTIAL: Model management governance is not yet equivalent to gateway tool governance.
- PARTIAL: Full lane isolation is architectural doctrine, not a strict runtime/package boundary.

