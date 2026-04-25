# Restore Scoring Operations

Status date: 2026-04-24.

Restore scoring is invoked through `COMPILE_CONTEXT`. It is deterministic and CPU-only.

Example payload:

```json
{
  "action": "COMPILE_CONTEXT",
  "payload": {
    "query": "summarize blockers",
    "persistSnapshot": true,
    "restoreMode": true,
    "restoreSnapshotKind": "restore",
    "restoreCandidateLimit": 12,
    "restoreMinScore": 0.45,
    "expandRestoreGraph": false
  }
}
```

Operator meaning:

- `selected` means the top scored candidate cleared threshold.
- `fresh_compile_no_candidates` means no scoped candidates were available.
- `fresh_compile_below_threshold` means candidates existed but were not reliable enough.
- `fresh_compile_forced` means resume hints requested fresh compile.
- `requires_fresh_compile=true` means a fresh compile should be used for the current request.

Persisted inspectability fields:

- `restore_scores_json`
- `restore_trace_json`
- `resume_hints_json`
- `restore_package_json`

The restore package is header-first. It includes compact evidence refs and candidate summaries without expanding full graph/delta unless requested.

Validation:

```bash
cd services/core
go test ./internal/aios/controllane
```
