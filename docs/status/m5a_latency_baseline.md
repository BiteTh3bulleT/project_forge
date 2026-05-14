# M5A Latency Baseline

Date: 2026-05-14

Status: `M5A_FOUNDATION_VALIDATED / LATENCY_NOT_MEASURED`

This file records the latency measurement state for M5A. Validation commands were run after implementation, but no runtime benchmark, API latency probe, modelruntime request timing, HostBridge sample timing, or desktop render timing command was run. Unknowns are marked `NOT_MEASURED` rather than inferred from docs.

## Repository Context

| Field | Value |
| --- | --- |
| Branch | `main` |
| HEAD | `e3ec4bcaf28b1ad0db26bf92accc022616b8d7ec` |
| Upstream status | `behind origin/main by 34 commits` |
| Dirty status | Dirty before this docs lane; unrelated dirty files were present outside the allowed write scope. |

## Baseline Measurements

| Area | Metric | Value | Evidence |
| --- | --- | --- | --- |
| Core boot | cold boot wall time | `NOT_MEASURED` | No boot command run. |
| Smoke suite | `npm run smoke` duration | `NOT_MEASURED` | Not run in this docs-only pass. |
| Root tests | `npm test` duration | `NOT_MEASURED` | Not run in this docs-only pass. |
| Lint/static checks | `npm run lint` duration | `NOT_MEASURED` | Not run in this docs-only pass. |
| Build | `npm run build` duration | `NOT_MEASURED` | Not run in this docs-only pass. |
| Modelruntime health | `GET /forge/model-runtime/health` latency | `NOT_MEASURED` | No dev server/API probe run. |
| Modelruntime chat | `/forge/models/{id}/chat` latency | `NOT_MEASURED` | No modelruntime request run. |
| OpenAI-compatible chat | `/v1/chat/completions` latency | `NOT_MEASURED` | No modelruntime request run. |
| Gateway invoke | `POST /api/gateway/invoke` latency | `NOT_MEASURED` | No API probe run. |
| Control Lane validation | validation syscall latency | `NOT_MEASURED` | No Go tests or benchmarks run. |
| HostBridge snapshot | snapshot capture latency | `NOT_MEASURED` | No HostBridge command/test run. |
| FORGE-H policy | policy evaluation latency | `NOT_MEASURED` | No FORGE-H command/test run. |
| System status | `GET /forge/system/status` latency | `NOT_MEASURED` | Smoke does not probe this endpoint. |
| Desktop System page | render/refresh latency | `NOT_MEASURED` | No desktop test/browser run. |

## Known Latency-Relevant Facts From Inspection

- `GET /forge/system/status` now reports HostBridge and FORGE-H TTL cache metadata. The route still disables command-backed HostBridge probes for shell status.
- Modelruntime has scheduler, queue, backend health, loaded, usage, and streaming-capable chat surfaces, but this pass did not measure them.
- Micro-agent acceleration is design-only in `docs/architecture/micro_agent_acceleration.md`; no worker loop is wired.

## Validation Commands

These are pass/fail validation results, not latency measurements:

| Command | Result |
| --- | --- |
| `npm test` | `PASS` |
| `npm run lint` | `PASS` |
| `npm run build` | `PASS` |
| `npm run smoke` | `PASS` |
| `npm -w @forge/desktop run validate` | `PASS` |
| `cd apps/desktop/src-tauri && cargo test` | `PASS` |
| `git diff --check` | `PASS` |
| `npm run validate:js` | `UNAVAILABLE` - script is not defined |
| `npm run validate:local` | `UNAVAILABLE` - script is not defined |

## Measurement Rules For Later Phases

- Record exact command, date, branch, HEAD, dirty status, environment, and result.
- Use `NOT_MEASURED` for every missing metric.
- Separate cold start, warm cache, degraded backend, safe-mode CPU-only, and no-model paths.
- Include failure and unavailable timings; do not report only healthy paths.
- Keep model output and raw host dumps out of latency artifacts.
