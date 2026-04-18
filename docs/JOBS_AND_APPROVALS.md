# Jobs And Approvals

## Core Doctrine

- **Job events are immutable truth.**
- **Job status is current projection.**

`job_events` is append-only and used for replay/debugging.

## Job Template Model

Command and API create jobs from known templates only.

Template encodes:

- intent/requested action
- adapter target + capability
- execution boundary
- default risk class
- default write intent
- expected artifact types

This prevents command-bar freestyle backend mutation.

## Job Lifecycle

1. `queued`
2. `preparing`
3. `awaiting_approval` (if required)
4. `running`
5. terminal: `succeeded` | `failed` | `cancelled`

Every transition is written to `job_status_history` and `job_events`.

## Approval Chain

Approvals are split into two record types:

- `approval_requests`
  - requested action
  - risk class
  - adapter
  - scope snapshot
  - write intent
  - packet reference
- `approval_decisions`
  - decision (`approved`/`denied`/`cancelled`)
  - actor
  - note
  - timestamp

This supports policy changes, retries, and future re-approval logic.

## Risk Classes

- `read_only` — no gate by default
- `external_reasoning` — approval required
- `write_files` — approval required
- `run_commands` — approval required

## Cancellation

- queued/preparing/awaiting jobs: immediate cancel transition
- running jobs: cancel request + context cancellation signal

Cancellation is logged and persisted with failure code `user_cancellation`.

## Failure Taxonomy

Jobs record one of:

- `validation`
- `approval_denied`
- `adapter_unavailable`
- `adapter_timeout`
- `packet_build_failure`
- `persistence_failure`
- `user_cancellation`
- `execution_failure`

## API Highlights

- `POST /api/jobs` create from template
- `GET /api/jobs` list projection rows
- `GET /api/jobs/{id}` detail (events/history/packet/artifacts/approval)
- `POST /api/jobs/{id}/cancel`
- `GET /api/approvals`
- `POST /api/approvals/{id}/approve|deny|cancel`
