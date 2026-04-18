# POLICY_AND_APPROVALS

## Policy Layers

FORGE governance separates:

- `Action lanes` (operation scope, write intent, lane-level approval gates)
- `Permission profiles` (tool allowlist, path scopes, network, write limits, risk approval rules)

Both are evaluated for each gateway request.

## Risk + Execution Model

Risk classes:

- `read_only`
- `safe_write`
- `scoped_execute`
- `privileged`
- `dangerous`

Execution levels:

- `L0` read inspection
- `L1` safe local writes
- `L2` controlled execution
- `L3` privileged actions
- `L4` dangerous/admin-grade actions

A request cannot exceed the registered tool execution-level cap.

## Policy Outcomes

Every request resolves to one of:

- `allow`
- `require_approval`
- `deny`

No implicit escalation path exists.

## Approval Integration

- Gateway `needs_approval` outcomes are persisted in invocation history.
- For job-backed gateway actions, approval requests are opened and linked to the job.
- Job processing pauses in `awaiting_approval` until operator decision.

## Operator Surfaces

- `#/gateway` for request history and invoke details
- `#/action-lanes` for lane definitions
- `#/permissions` for active profile and path/tool/risk gates
- `#/approvals` for operator decisions
