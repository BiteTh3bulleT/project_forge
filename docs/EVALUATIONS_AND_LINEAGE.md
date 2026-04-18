# Evaluations and Lineage

## Evaluations

Module: `internal/evaluations`

Evaluations provide grounded, operator-entered outcome scoring.

Stored in `evaluation_records` with fields for:

- success flag
- quality rating
- usefulness rating
- correctness confidence
- packet quality
- adapter suitability
- retry recommendation
- routing influence flag
- notes/scorer

### API

- `POST /api/evaluations`
- `GET /api/evaluations`
- `GET /api/evaluations/metrics`

### UI

Route: `/evaluations`

- submit manual evaluations
- inspect evaluation history
- compare adapter metrics

## Lineage

Module: `internal/lineage`

Lineage tracks parent-child relationships across retries and replays.

Stored in `job_lineage` with:

- parent job id
- child job id
- relation type (`retry`, `replay`)
- change summary JSON

### API

- `POST /api/jobs/{id}/retry`
- `POST /api/jobs/{id}/replay`
- `GET /api/lineage/jobs/{id}`

### Behavior

- retry: clones base metadata with optional overrides
- replay: clones original metadata as-is
- both create child jobs and persist lineage edges

### UI

Route: `/lineage`

- choose origin job
- trigger retry/replay
- inspect parent/child edges and change summaries
- review related jobs

## Why Both Matter

Evaluations answer: "How well did this run perform?"

Lineage answers: "What changed between attempts, and did it help?"

Together they provide the minimum memory loop for iterative operator improvement.
