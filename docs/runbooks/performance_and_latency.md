# Performance And Latency Runbook

Status date: 2026-04-27.

Use this runbook when chat feels slow, model calls time out, or FORGE appears to be doing too much work for a simple operator turn.

## 1. Confirm Fast Path Eligibility

Simple requests should use no-model routing:

- `what mode are we in`
- `show diagnostics`
- `modelruntime status`
- `what is degraded`
- `show latest restore decision`

Inspect the assistant message metadata for `chatLatencyTrace`:

- `context_budget_class` should be `tiny`
- `output_mode` should be `brief`
- `model_calls_avoided` should be `1`

If an obvious status request calls modelruntime, check the classifier phrases before changing gateway/modelruntime behavior.

## 2. Check Runtime Preflight

If model calls are slow or failing, inspect:

```sh
curl -s http://127.0.0.1:18492/forge/model-runtime/queue | jq .
curl -s http://127.0.0.1:18492/forge/model-runtime/health | jq .
```

Cooldown, unavailable, or saturated runtime state should fail fast with an explicit reason. It should not trigger repeated chat retries.

## 3. Verify Output Mode

Default chat should use `normal`. Status/no-model should use `brief`. Long-form reports should use `report` only when explicitly requested.

Large output modes increase generation time. Do not route ordinary conversation to `deep` or `report`.

## 4. Check Restore Behavior

Restore inspection should be header-first by default. Full graph expansion should require `expandRestoreGraph=true`.

Restore scoring trace should include cache metadata:

- `cache.hit`
- `retrieval.candidate_count`
- `selected.header_only`

Repeated identical restore inputs should hit the in-memory scoring cache until TTL expiry or candidate/outcome fingerprints change.

## 5. What Not To Optimize Prematurely

Do not add LLM classification for hot-path routing.
Do not add unbounded DB scans for diagnostics.
Do not expand raw transcripts or full restore graphs by default.
Do not silently fall back to cloud providers.
Do not increase retry counts to hide provider failures.
