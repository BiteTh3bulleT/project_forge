# Executive Brief

## FORGE In One Sentence

FORGE is a local-first cognitive operating system for governed AI-assisted engineering, where models and agents may propose but deterministic kernel, gateway, approval, and audit paths decide what becomes truth or action.

## What Makes It Different

IMPLEMENTED: FORGE separates reasoning from authority. The semantic syscall control lane owns canonical semantic mutation. The gateway owns governed tool execution. Modelruntime owns governed inference. Context snapshots, vector retrieval, Dream Mode output, tool results, and model output are evidence or proposals unless committed through deterministic boundaries.

PARTIAL: The architecture is strong, but the operator surface and some authority-adjacent APIs are not yet fully converged. Model management governance, public syscall write ingress, Dream report persistence, and trace-first UI remain unfinished.

## What Is Real Now

- IMPLEMENTED: Go core service, SQLite store, migrations, approvals, jobs, audit records, gateway invocations, action lanes, permissions, artifacts, chat, backup, and desktop API wiring.
- IMPLEMENTED: Semantic syscall registry, validator, processor, SQLite transaction runner, append-only `journal_events`, state history, contradiction/supersession records, and context snapshot persistence.
- IMPLEMENTED: Gateway tool taxonomy, approval gates, dangerous capability posture, audit linkage, legacy adapter direct route removal, and approval fingerprint hardening.
- IMPLEMENTED: Modelruntime M1/M2/M3 core: manifest/store/registry, fake/llama.cpp/OpenAI-compatible backend support, lifecycle state, queueing, usage/audit, management APIs, and health surfaces.
- PARTIAL: Dream Mode v0 dry-run reports, autonomy charters/intents/budgets, rule agents, restore scoring, desktop operator pages, and backup/restore parity.
- CONCEPT: IRIS, GHOST, ARTEMIS, Rule Cell substrate, Hyperlane reflex routing, adapter learning/LoRA, and full cockpit UI are design directions, not complete runtime systems.

## Why Deterministic Authority Matters

FORGE assumes a powerful model can be useful and unsafe at the same time. The design refuses to let generated text, vector recall, agent plans, or Dream reports directly mutate canonical memory or execute tools. That is the core safety property.

## Why Local-First Matters

FORGE keeps the kernel, journal, state, approval, audit, and gateway authority local. Cloud or frontier models may be used as governed accelerators only when explicitly configured. Unconfigured cloud providers must not become silent fallback authority.

## Current Maturity Rating

PARTIAL / engineering prototype with real control-plane foundations.

The foundations are serious: control lane, gateway, modelruntime, migrations, audit, and desktop shell exist. The system is not yet a complete AI-OS product because operator explainability, model management governance, restore parity, and UI tests remain incomplete.

## Top Risks

1. PARTIAL: Model management APIs still need gateway-equivalent approval governance.
2. PARTIAL: Backup/restore still has export-only or incomplete sections around VSA/retrieval evidence.
3. PARTIAL: Dream Mode v0 reports are safe but not durably reviewable.
4. PARTIAL: Context restore scoring exists but candidate listing has known quality/scalability concerns.
5. MISSING: Desktop has no dedicated unit/component/e2e test suite.
6. PARTIAL: Trace data exists, but operator workflows do not yet make full causality easy to inspect.
7. SCAFFOLD: Rule Cells/Hyperlane are not implemented as the future deterministic substrate.
8. NOT VERIFIED: Nix behavior could not be validated on this Windows host.

## Next 5 Priorities

1. Govern model management actions.
2. Close backup/restore parity for retrieval/observation/VSA posture.
3. Persist Dream Mode reports as non-canonical evidence.
4. Add operator trace/restore/Dream inspector flow.
5. Build Rule Cell/Hyperlane v0 as deterministic proposal infrastructure.

