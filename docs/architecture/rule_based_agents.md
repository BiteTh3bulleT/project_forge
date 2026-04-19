# Rule-Based Agents (Phase 5.5 + 5.75 Integration)

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
  - proposes conservative archive candidate (typically proposal/approval path)

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

## Forward path

As Phase 6+ evolves runtime/event scheduling, these rule agents can be scheduled daemons while retaining the same intent->policy->kernel flow.
