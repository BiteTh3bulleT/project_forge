# Dream Mode Operations

Status date: 2026-04-24.

Run a dry-run report through the core API:

```bash
curl -sS http://127.0.0.1:18492/api/dream/run \
  -H 'Content-Type: application/json' \
  -d '{"workspaceId":"default","laneId":"control.semantic","mode":"microdream"}'
```

Useful request fields:

- `mode`: `microdream`, `nap`, or `deep_dream`
- `workspaceId`: required
- `laneId`: optional
- `windowHours`: optional override
- `maxCandidates`: optional override
- `dryRun`: defaults to `true`
- `allowLongTermPromotion`: defaults to `false`
- `requireOperatorReviewForLongTerm`: defaults to review-required unless long-term is explicitly allowed
- `allowCommits`: ignored in v0; output remains dry-run
- `persistReport`: optional; when `true`, stores the dry-run report in `dream_reports` as non-canonical evidence

Persist a report for later operator review:

```bash
curl -sS http://127.0.0.1:18492/api/dream/run \
  -H 'Content-Type: application/json' \
  -d '{"workspaceId":"default","laneId":"control.semantic","mode":"nap","persistReport":true}'
```

List persisted reports for a workspace:

```bash
curl -sS 'http://127.0.0.1:18492/api/dream/reports?workspaceId=default&laneId=control.semantic&mode=nap&limit=20'
```

Get a single report:

```bash
curl -sS 'http://127.0.0.1:18492/api/dream/reports/<report-id>?workspaceId=default&laneId=control.semantic'
```

Inspect report subresources:

```bash
curl -sS 'http://127.0.0.1:18492/api/dream/reports/<report-id>/candidates?workspaceId=default&laneId=control.semantic'
curl -sS 'http://127.0.0.1:18492/api/dream/reports/<report-id>/proposals?workspaceId=default&laneId=control.semantic'
curl -sS 'http://127.0.0.1:18492/api/dream/reports/<report-id>/warnings?workspaceId=default&laneId=control.semantic'
```

The desktop Inspectors page also exposes a compact Dream Reports panel. It lists persisted reports
by workspace/lane/mode and shows replay candidates, salience scores, memory-tier proposals, repair
proposals, snapshot hygiene proposals, warnings, trace, and metadata. The panel is read-only and
labels the content as non-canonical evidence.

Safe-mode notes:

- No GPU is required.
- No modelruntime backend is required.
- No vector retrieval result is treated as truth.
- No canonical memory/state/loop/journal rows are written by Dream Mode v0.
- Persisted Dream reports are evidence rows only; they are not canonical memory or state.
- Embedding refresh actions are proposals only; TEI/vector rebuilds are not run or committed by Dream Mode v0.

Validation:

```bash
cd services/core
go test ./internal/aios/dream
```

Future work intentionally left out of v0:

- scheduled Dream Mode auto-run
- governed commit mode
- adapter training queue
- LoRA/PEFT training
- GPU Dream Mode subjobs
- operator GUI/voice/vision
