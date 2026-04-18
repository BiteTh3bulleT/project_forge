# Reviews and Reconciliation

## Imported Execution Reconciliation

Imported runs are stored in `imported_executions`.

Reconciliation records are stored in `imported_execution_reconciliations`.

API:

- `GET /api/reconciliation`
- `GET /api/reconciliation/imports/{id}`
- `POST /api/reconciliation/imports/{id}`

Reconciliation fields:

- changed files
- failure reasons
- unresolved issues
- suggested next steps
- agent notes
- patch summary
- review status

## Review Records

Review queue is stored in `review_records`.

API:

- `GET /api/reviews`
- `POST /api/reviews`
- `PATCH /api/reviews/{id}`

Statuses:

- `pending`
- `approved`
- `rejected`
- `deferred`

Review records can target any supported target type (`job`, `import`, `artifact`, `packet`, etc.).

## Import -> Review Flow

On import creation:

1. `imported_executions` row is created.
2. a reconciliation row is created/updated.
3. a pending review record is created.

Automation can add additional review records when configured.

## UI Surfaces

- `/reviews` for queue operations and reconciliation edits
- `/dashboard` review pending summary
- `/dossiers` dossier-linked review visibility
- `/jobs/:id` job-linked review visibility

## Guardrails

- review state is explicit and persisted
- no imported execution is treated as trusted execution truth without operator review
- reconciliation is evidence, not autonomous enforcement
