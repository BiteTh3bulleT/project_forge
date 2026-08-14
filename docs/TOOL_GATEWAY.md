# TOOL_GATEWAY

## Purpose

FORGE machine actions now route through one backend path: `Gateway.Execute` in `services/core/internal/gateway/service.go`.

No UI page or job path should execute equivalent machine actions through side-door helpers.

## Request Contract

Gateway request fields:

- `toolId`
- `laneId`
- `domain`
- `action`
- `riskClass`
- `executionLevel` (`L0`..`L4`)
- `paths`
- `input`
- `correlationId`
- `jobId` / `packetId` (optional linkage)
- `initiator`
- `dryRun`

## Execution Pipeline

1. Resolve lane and verify lane enabled.
2. Resolve registered typed tool.
3. Enforce lane write intent compatibility.
4. Normalize and scope-check paths against lane.
5. Resolve effective risk class and execution level.
6. Enforce execution-level cap per tool.
7. Run permission profile checks.
8. Branch to policy outcome:
   - `allow`
   - `require_approval`
   - `deny`
9. Persist invocation row.
10. Execute tool, persist result, emit audit.

## Result Contract

Gateway result includes:

- `status` (`ok`, `dry_run`, `needs_approval`, `denied`, `error`)
- `policyOutcome` (`allow`, `require_approval`, `deny`)
- `riskClass`, `executionLevel`
- `domain`, `action`, `tool`, `lane`
- `invocationId`, `correlationId`, `profileId`
- normalized `data`, `artifacts`, `message`

## Job Integration

Template `gateway_action` executes tool requests through the same gateway.

- Job event stream records gateway result metadata.
- Gateway result is attached as a job artifact.
- Approval-required outcomes move job state to `awaiting_approval`.

## Chat Integration

Chat supports explicit tool dispatch with command payload:

- `/tool { ...gateway request fields... }`

This creates a `gateway_action` job so approvals, audit, and artifacts remain governed.

For natural-language chat, FORGE performs deterministic intent routing before
the model is called. FORGE either selects exactly one governed gateway tool or
exposes no tool schema. A model can only format bounded proposal arguments for
the already selected schema; it cannot select a different tool or cause direct
execution. Ambiguous operational language stays on the no-tool response path.

## Deferred

- Non-job gateway approval requests currently require a backing job id for full approval request linkage.
