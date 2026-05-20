# Tool Gateway

The gateway is the only authorized tool execution boundary in FORGE.
Last updated: 2026-04-22.

## Pipeline

1. Receive `ToolRequest` / gateway request.
2. Normalize correlation/source/workspace/provenance envelope.
3. Resolve capability (`toolId` or mapped legacy tool id).
4. Classify risk and evaluate policy:
   - capability status
   - dry-run eligibility
   - scope/path bounds
   - self-initiated intent/charter/budget checks
   - approval requirements
5. Resolve lane and permission profile checks.
6. Apply approval gate if required.
7. Execute mapped adapter/tool (or return deterministic unsupported/disabled).
8. Persist invocation record.
9. Emit audit record for success/failure/denial/approval-required.
10. Return structured result with policy outcome and evidence refs.

## Adapter Boundary

No direct shell/filesystem/network operations are allowed outside registered gateway tools/adapters.

Legacy compatibility note:
- `/api/adapters/{id}/invoke` is removed and no longer routed.
- Adapter execution must go through `/api/gateway/invoke` (for example `toolId=legacy.adapter.invoke`, `laneId=legacy.adapter.invoke` for compatibility probing).
- `legacy.adapter.invoke` is deprecated compatibility only. It is classified as networked scoped execution (`scoped_execute`, `L2`, `usesNetwork=true`) so permission profiles and approval gates do not treat model/network adapter behavior as a low-risk local read.

Phase 5.9 uses:

- existing built-in gateway tools for active capabilities
- capability status + mapping metadata for broader taxonomy
- deterministic terminal results for non-implemented primitives

## Policy Outcomes

- `ok`
- `dry_run`
- `needs_approval`
- `denied`
- `unsupported`
- `disabled`
- `error`

## Autonomy Integration

Self-initiated tool calls are treated as governed requests:

- intent context required (`intentId`) for self-initiated sources
- optional autonomy authorizer can enforce charter/budget policy
- missing charter/budget context escalates to approval requirement
- future IRIS source is never privileged by source identity alone

## Resource Controls (Phase 5.9)

Practical controls enforced now:

- lane/workspace/path boundary checks
- permission profile scopes
- per-tool timeout behavior where tool supports context timeout
- bounded output limits encoded in capability metadata

Controls documented but partially enforced in later phases:

- cpu/memory/network byte hard caps
- host/method deny/allow rules for all adapters

## Evidence and Audit

Tool execution always writes audit evidence.

Invocation records capture:

- correlation/trace
- tool/lane/action/risk
- policy outcome
- denial reasons
- result payload (bounded)
- artifact summary refs
