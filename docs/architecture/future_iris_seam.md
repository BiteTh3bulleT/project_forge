# Future IRIS Seam (FORGE-owned)

IRIS is planned as a Compute Lane semantic service inside FORGE AI-OS.

FORGE remains authoritative for canonical memory/state, permissions, approvals, journal, artifacts, workspace boundaries, and lifecycle transitions.

## IRIS may

- propose notes
- propose links
- propose contradictions
- propose derived models
- request context packets
- assist semantic interpretation as a Compute Lane service

## IRIS may not

- directly write canonical memory
- directly mutate active state
- bypass permissions/capabilities
- bypass semantic syscall validation
- bypass approval gates
- own workspace boundaries
- own the event journal
- own artifacts
- own process lifecycle

## Control rule

**IRIS proposes. FORGE validates. FORGE commits.**

## Future flow

1. User/system event enters FORGE.
2. FORGE ingest pipeline appends raw journal truth and runs internal librarian proposal cells.
3. IRIS receives bounded context as Compute Lane service input (optional/future).
4. IRIS submits candidate `SyscallRequest` actions (`source=future_iris`).
5. FORGE Control Lane runs registry/capability/approval/validation/transition checks.
6. FORGE commits accepted actions through transaction boundaries.
7. FORGE emits audit records for committed and rejected proposals.
8. Context compiler uses committed state and evidence.

Phase 4-5 status:

- FORGE-only ingest + librarian cells are active.
- Semantic inference adapter seam exists with no live provider requirement.
- `source=future_iris` proposals still require the same capability/approval/kernel path.
- truth maintenance services (active state/loops/contradiction/supersession/current-object resolution) are active in FORGE and remain kernel-governed.

## Phase 5 truth boundary

Future IRIS cannot bypass FORGE truth services:

- `UPDATE_STATE`, `OPEN_LOOP`, `CLOSE_LOOP`, `MARK_SUPERSEDED`, `REGISTER_CONTRADICTION`, `DERIVE_MODEL`, `ARCHIVE_NOTE`
  must pass Control Lane validation and transition rules.
- truth projections are derived from committed durable records only.
- provenance/audit/correlation links remain required for IRIS-sourced commits.

## Example objects

### IRIS candidate semantic action request

```json
{
  "id": "iris-req-1",
  "action": "CREATE_NOTE",
  "actor": {
    "id": "iris.service",
    "kind": "future_iris"
  },
  "source": "future_iris",
  "scope": {
    "workspaceId": "ws-main",
    "laneId": "compute.iris"
  },
  "payload": {
    "type": "fact",
    "title": "Observed contradiction",
    "content": "Build graph and runtime graph disagree on adapter scope.",
    "confidence": 0.74
  },
  "provenance": {
    "actor": "iris.service",
    "actorType": "future_iris",
    "source": "semantic_inference",
    "traceId": "trace-iris-1"
  },
  "correlationId": "corr-iris-1",
  "traceId": "trace-iris-1",
  "idempotencyKey": "iris-req-1",
  "dryRun": true,
  "requestedAt": 1760000100000
}
```

### FORGE validation response

```json
{
  "success": false,
  "action": "CREATE_NOTE",
  "requestId": "iris-req-1",
  "correlationId": "corr-iris-1",
  "traceId": "trace-iris-1",
  "dryRun": true,
  "approvalStatus": "approval_required",
  "committedObjectIds": [],
  "rejectedReasons": [
    {
      "code": "APPROVAL_REQUIRED",
      "field": "approval",
      "message": "mutating semantic actions from proposer sources require approval"
    }
  ],
  "warnings": [],
  "auditId": "audit-42",
  "validationDetails": [
    { "layer": "envelope_validation", "passed": true, "issues": [] },
    { "layer": "capability_validation", "passed": true, "issues": [] },
    { "layer": "approval_gate", "passed": false, "issues": [{ "code": "APPROVAL_REQUIRED", "field": "approval", "message": "mutating semantic actions from proposer sources require approval" }] }
  ],
  "stateSummary": {
    "dryRun": true
  },
  "deterministicErrorCode": "APPROVAL_REQUIRED"
}
```

### Context packet request

```json
{
  "id": "iris-ctx-1",
  "action": "COMPILE_CONTEXT",
  "actor": { "id": "iris.service", "kind": "future_iris" },
  "source": "future_iris",
  "scope": { "workspaceId": "ws-main" },
  "payload": {
    "query": "summarize active blockers for release readiness",
    "budget": {
      "maxTokens": 6000,
      "maxEvents": 100,
      "maxNotes": 120
    }
  },
  "provenance": {
    "actor": "iris.service",
    "actorType": "future_iris"
  },
  "requestedAt": 1760000105000
}
```

### Context packet syscall response (Phase 2 deterministic stub)

```json
{
  "success": true,
  "action": "COMPILE_CONTEXT",
  "requestId": "iris-ctx-1",
  "approvalStatus": "allowed",
  "committedObjectIds": [],
  "warnings": ["compile_context is deterministic Phase 2 stub"],
  "stateSummary": {
    "contextPacketId": "ctx-summarize_active_blockers_for_release_readiness-1760000105000",
    "notes": 3,
    "openLoops": 1,
    "models": 1
  }
}
```
