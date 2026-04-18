# FORGE - Architecture (Phase 4)

## System Intent

FORGE is a local-first operator machine with explicit control boundaries:

- jobs are projections
- events are truth
- packets are contracts
- approvals are gates
- artifacts are evidence
- adapters are bounded workers

Phase 4 adds policy, strategy, automation, review, and reconciliation systems on top of Phase 3 memory/evaluation foundations.

## High-Level Shape

```text
apps/desktop (Tauri + React)
  Dashboard | Command | Memory | Project Context | Policy | Strategies
  Automation | Reviews | Dossiers | Retrieval Runs | Evaluations
  Lineage | Insights | Jobs | Approvals | Adapters | Events | Settings
        |
        | HTTP JSON (polling transport with SSE-compatible event payloads)
        v
services/core (Go)
  api
  ingest/search
  embeddings/retrieval/ranking
  packets
  jobs/orchestration
  approvals
  strategies
  policy
  automation
  packetopt
  dossiers
  evaluations
  lineage
  imports + reconciliation
  reviews
  failurepatterns
  insights
  dashboard
  adapters
  artifacts/events
        |
        v
SQLite (WAL) + local artifact files
```

## Core Boundaries

- `internal/strategies`
  - persisted execution strategy contracts
  - task type, adapter, retrieval mode, packet rules, approval flags, success + retry guidance
- `internal/policy`
  - approval presets (`conservative`, `balanced`, `aggressive`)
  - dossier profile overrides
  - routing recommendation engine with confidence/reasons/evidence and inferred-vs-direct signal
- `internal/automation`
  - bounded trigger/condition/action rules
  - dry-run preview and immutable history records
  - executor callback constrained to known action types
- `internal/packetopt`
  - packet guidance scoring and issue detection
  - persisted guidance evidence records
- `internal/reconciliation`
  - import reconciliation records (changed files, failure reasons, unresolved issues, next steps)
- `internal/reviews`
  - explicit review queue with approved/rejected/deferred decisions
- `internal/failurepatterns`
  - failure snapshot analysis by adapter/strategy/retrieval/packet style
- `internal/dashboard`
  - command-deck summary aggregation

Phase 3 modules remain core to memory:

- `embeddings`, `retrieval`, `dossiers`, `evaluations`, `lineage`, `imports`, `insights`

Phase 2 execution modules remain enforcement core:

- `jobs`, `approvals`, `packets`, `projectcontext`, `adapters`, `artifacts`, `events`

## Data Model Extensions (Phase 4)

- `execution_strategies`
- `approval_presets`
- `dossier_profiles`
- `routing_policy_recommendations`
- `automation_rules`
- `automation_history`
- `packet_guidance_records`
- `imported_execution_reconciliations`
- `review_records`
- `failure_pattern_snapshots`

These extend prior phase tables and keep decision/retry/review evidence queryable.

## Routing + Strategy Flow

1. operator provides task context (task type/objective/dossier/optional strategy override)
2. policy selects strategy (forced > dossier preference > task-type match > fallback)
3. policy applies dossier profile overrides (adapter/retrieval/preset where present)
4. policy computes confidence from stored evaluations
5. recommendation is persisted with:
   - confidence
   - reasons
   - evidence
   - inferred/direct marker
   - operator override allowance

No recommendation auto-executes risky work.

## Automation Safety Model

- automation rules can only run known bounded actions
- dry-run is first-class and persisted in history
- rule execution does not bypass existing job approval gates
- high-risk actions still pass through `jobs` + `approvals` boundaries

## Review and Reconciliation Model

- imported external execution persists base import record
- reconciliation record captures file/failure/next-step structure
- review records track operator decision state transitions
- dashboard and dossier/job views expose pending review pressure

## Execution Boundary Discipline

FORGE enforces explicit categories:

- memory retrieval
- reasoning
- write proposal
- write execution
- command execution

No adapter or automation rule may silently escalate scope.

## Phase 5 — Execution plane (gateway, permissions, audit, portability)

- **Gateway** (`internal/gateway`): single `Execute` pipeline for bounded tools; persists `gateway_invocations`.
- **Lanes** (`internal/lanes`): `action_lanes` describe scoped, inspectable operation templates.
- **Execution permissions** (`internal/permissions`): `permission_profiles` gate tools/paths/risk separately from routing **Policy**.
- **Audit** (`internal/audit`): `audit_records` with correlation ids; list + trace APIs.
- **Backup** (`internal/backup`): portable JSON bundles + restore with `dryRun`.
- **Release** (`internal/release`): readiness checklist, first-run summary, `release_artifacts` bookkeeping.

### Phase 5 follow-ups (not finished here)

- Wire gateway `needs_approval` into the existing `approval_requests` consumer loop end-to-end (no silent auto-approve).
- Optional: schema migration versioning beyond “CREATE IF NOT EXISTS” for long-lived installs.

### Longer-horizon research (still valid)

- SSE/native stream transport for high-volume live updates
- policy confidence calibration curves and experiment frameworks
- auto-suggested strategy edits from long-horizon trend analysis
- artifact-level semantic citation extraction for deeper packet optimization
