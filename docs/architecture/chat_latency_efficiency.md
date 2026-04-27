# Chat Latency And Efficiency

Status date: 2026-04-27.

FORGE chat latency is optimized by routing deterministic work before context expansion, gateway execution, and modelruntime inference.

The rule remains:

**CPU/RAM routes first. Modelruntime only when inference is actually needed.**

## No-Model Routing

The chat API now classifies simple operator turns with a CPU-local Hyperlane-style classifier. Confident structured requests bypass modelruntime and gateway execution:

- health/status
- diagnostics
- modelruntime/backend availability summary
- CPU-only/safe-mode mode questions
- degraded-state questions
- latest restore/context snapshot summary requests
- identity and weather-without-location clarifiers

No-model replies are short structured answers from already loaded process state or chat metadata. They do not call modelruntime, do not call the network, do not assemble large context, and do not execute gateway tools.

Ambiguous requests fall through to the existing governed chat path.

## Context Budget Classes

Every chat turn receives a budget class before inference:

| Class | Use |
|---|---|
| `tiny` | no-model/status/diagnostics/restore inspector or very small deterministic replies |
| `small` | default normal chat |
| `medium` | larger operator turns |
| `deep` | explicit investigation, root-cause, security, or architecture review |
| `report` | explicit long-form report/review generation |

The default is `small`, not `deep`. Deep/report classes require explicit wording.

## Output Modes

Chat also receives an output mode:

| Mode | Behavior |
|---|---|
| `brief` | no-model/status replies and small runtime caps |
| `normal` | default chat |
| `deep` | explicit deep reasoning/review |
| `report` | explicit report generation |

Modelruntime max token caps are selected from the output mode so simple turns do not pay report-sized generation latency.

## Runtime Preflight

Before modelruntime chat execution, FORGE checks the runtime queue/policy state. Cooldown or saturated queue states fail fast with an explicit degraded reason instead of entering a slow retry path.

The plain chat path sets `MaxAttempts=1` to avoid retry storms. Cloud providers are not silently selected unless they are configured and registered in modelruntime.

## Header-First Restore

Restore packages stay header-first:

- selected snapshot/context IDs
- score and decision reason
- header
- resume hints
- compact evidence refs
- candidate summaries
- trace

Full graph/delta expansion remains opt-in through `expandRestoreGraph`.

## Restore Scoring Cache

Restore selection uses a bounded in-memory cache for deterministic scoring summaries. The key includes:

- workspace id
- lane id
- normalized query
- snapshot kind
- resume hint threshold/fresh-only/preferred snapshot
- candidate set fingerprint
- restore outcome fingerprint

The cache is TTL-bound and max-size bounded. New snapshots, changed outcome feedback, different workspace/lane, or different hints naturally produce a miss. It stores summarized selection results only, not canonical truth.

## Telemetry

Assistant message metadata includes `chatLatencyTrace` where applicable:

- `total_request_ms`
- `hyperlane_ms`
- `context_budget_class`
- `output_mode`
- `route_intent`
- `route_reason`
- `route_confidence`
- `modelruntime_ms`
- `gateway_preflight_ms`
- `gateway_execution_ms`
- `model_calls_avoided`
- `fresh_compile_avoided`
- `tokens_estimated`

Trace fields are summaries and avoid raw prompt/body leakage.

## Non-Goals

This pass does not add autonomy, Dream apply/commit, adapter training, UI redesign, or a parallel context engine. It does not weaken gateway, kernel, approval, capability, workspace, or modelruntime authority.
