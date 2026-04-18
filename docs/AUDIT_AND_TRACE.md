# AUDIT_AND_TRACE

## What is logged

Gateway actions persist:

- full invocation request envelope
- policy outcome + status
- approval linkage (when present)
- completion metadata and result payload summary

Audit records persist:

- category (`gateway`, `permissions`, `backup`, etc.)
- action (`tool.executed`, `tool.denied`, `tool.needs_approval`, ...)
- correlation id
- job and invocation linkage
- risk class
- outcome
- summary + payload

## Correlation

`correlationId` ties together:

- chat/job origin
- gateway invocation rows
- approval steps
- audit records

## Inspection

- `GET /api/gateway/invocations`
- `GET /api/audit`
- `GET /api/audit/trace/{correlationId}`

Desktop pages:

- `#/gateway`
- `#/audit`

## Failure Visibility

Failed or denied invocations remain visible with reason text.

No hidden retries or silent mutation paths are used by the gateway pipeline.
