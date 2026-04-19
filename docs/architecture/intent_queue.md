# Intent Queue

Intent queue is FORGE's explicit self-initiative ledger.

No self-initiated semantic commit is allowed without an intent.

## Intent model

Defined in `services/core/internal/aios/domain/autonomy.go` as `AutonomyIntent`.

Key fields:

- identity/type/title/description
- source/proposedBy
- scope/workspace
- lifecycle status
- risk/autonomy level
- charter and budget references
- approval linkage
- proposed actions and committed actions
- blocked reasons and evidence
- provenance/correlation/trace
- timestamps/metadata

## Lifecycle states

- `proposed`
- `approved`
- `running`
- `completed`
- `blocked`
- `rejected`
- `cancelled`
- `expired`

## Allowed transitions

Implemented in `AutonomyIntent.CanTransition` and enforced by `IntentQueueService`.

- `proposed -> approved|rejected|cancelled|expired`
- `approved -> running|cancelled|expired`
- `running -> completed|blocked|cancelled`
- `blocked -> approved|cancelled|expired`
- `completed|rejected|cancelled|expired` are terminal

## Queue operations

`services/core/internal/aios/autonomy/intent_queue.go`:

- enqueue/get
- list by status
- list active
- approve/reject/cancel
- mark running/completed/blocked
- expire old
- explain intent

Completed/rejected/cancelled intents remain inspectable.

## Intent vs open loop vs job/task

- Intent:
  - autonomy authorization envelope before action.
- Open loop:
  - unresolved domain work tracked in truth engine.
- Job/task:
  - runtime execution projection outside truth authority.

Intents are governance objects, not replacements for open loops or external scheduler jobs.

## Intent vs curiosity item

- Curiosity item:
  - possible inquiry only.
  - no commit authority.
- Intent:
  - explicit actionable autonomy envelope.

Promotion from curiosity to intent is explicit and auditable.

## Self-initiated flow

1. source proposes intent
2. policy evaluator decides
3. approval escalation if required
4. runner executes allowed syscalls through kernel
5. queue status updates + decision trace are persisted
