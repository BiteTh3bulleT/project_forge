# Gateway Tool Execution

## Role

IMPLEMENTED: Gateway is the governed tool execution authority. API/chat tool execution routes through `/api/gateway/invoke` and `gateway.Execute`. Legacy adapter direct invoke API routing is removed.

## Gateway Flow

1. Normalize request identifiers.
2. Resolve capability and backing gateway tool.
3. Resolve action lane.
4. Check lane path/write boundaries.
5. Evaluate risk/execution level and tool policy.
6. Run permission profile checks.
7. Require approval when needed.
8. Execute tool.
9. Persist gateway invocation and audit records.

## Approval Fingerprint Hardening

IMPLEMENTED: Approval grants are now bound to request fingerprints. The fingerprint includes actor id/kind, workspace, lane, tool, capability, risk, execution level, write intent, job id, domain/action, requested paths, resolved paths, normalized input shape, and approval request id for final grant validation.

Stored fields live in `approval_requests.scope_snapshot_json`:

- `approvalShapeHash`
- `approvalFingerprintHash`
- `approvalFingerprintFields`
- `approvalRequestId`

## Dangerous Tools

IMPLEMENTED: Dangerous and privileged capabilities are `approval_only` or policy-gated by default. Examples include deletion, process spawn/kill, shell execution, network effects, identity/secrets, external messages, and device capture.

## Remaining Risks

- PARTIAL: More service-specific harness tests are useful for configured dependency failures.
- RISK: Authority-adjacent non-tool APIs need review for approval/audit posture.
- RISK: Model management governance should converge with gateway-style capabilities.
- RISK: Backup restore is an administrative mutation path outside gateway approval policy. It is audited and transactional for supported DB sections, but should be reviewed as an authority-adjacent operation.
