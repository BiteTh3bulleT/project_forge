# Rule-Based Agents (Phase 5.5 + 5.75 Integration)

Status: `[LIVE] / [PARTIAL] / [NARROW]`. Deterministic rule agents can propose intents and semantic actions. They do not commit by themselves; any durable effect must pass autonomy policy and Control Lane syscall validation.

Rule-based agents are deterministic internal workers that detect conditions and propose bounded follow-up action.

They do not bypass semantic syscalls.

## Current runtime

Autonomy-integrated rule-agent runtime is implemented in:

- `services/core/internal/aios/autonomy/rule_agents.go`

Key contracts:

- `RuleAgent`
- `RuleAgentInput`
- `RuleAgentResult`
- `RuleAgentRuntime`

## Integration with autonomy layer

Flow:

1. rule agent evaluates scoped truth state deterministically
2. agent returns intent + proposed semantic actions
3. `SelfInitiatedSyscallRunner` runs policy checks:
   - charter
   - risk
   - budget
   - approval
   - kernel dry-run preview
4. if authorized, actions commit through Control Lane syscall processor
5. intent/decision/budget traces remain inspectable

## Built-in starter agents

- `OpenLoopStalenessAgent`
  - checks stale loops
  - emits stale-loop review intent
  - proposes low-risk warning note
- `CleanupProposalAgent`
  - emits memory-cleanup intent
  - currently emits no direct syscall actions by default (safe proposal-only posture while deterministic targeting remains narrow)

No other live rule agents are implemented in this phase. Broader deterministic maintenance agents are intentionally deferred until each agent has all of the following:

- deterministic signal inputs with bounded scope and stable provenance
- charter, budget, and risk policy coverage
- tests proving proposal-only behavior and destructive-action rejection
- operator-visible trace evidence for every proposed intent/action

## Design constraints

- deterministic logic only (no required LLM dependency)
- scope isolation enforced
- no direct writes to canonical state
- no recursive runaway execution without depth caps
- all durable changes remain syscall validated/audited

## Relationship to ingest pipeline

Ingest can optionally trigger one bounded autonomy/rule-agent pass through:

- `IngestPipelineOptions.AutonomyPass`
- `IngestPipelineOptions.MaxAutonomyDepth`

This keeps ingest-triggered autonomy bounded and inspectable.

## Deferred agents

The following agent families remain planned/deferred rather than live:

- contradiction and supersession review agents
- projection repair or rebuild proposal agents
- stale artifact/cache hygiene agents
- broader cleanup target selection agents
- runtime or model-maintenance agents

As Phase 6+ evolves runtime/event scheduling, additional rule agents can be scheduled daemons only while retaining the same intent->policy->kernel flow. Until then, docs and matrices should treat the rule-agent layer as a narrow, safe, proposal-only runtime rather than a broad deterministic workforce.
