# Dream Mode And Memory Metabolism

## What Dream Mode Is

PARTIAL: Dream Mode v0 is a CPU-only, deterministic, dry-run lymphatic maintenance engine. It selects replay candidates, scores salience, proposes memory tier routing, and emits consolidation/repair/report actions without canonical commits.

Evidence: `services/core/internal/aios/dream/service.go`, `service_test.go`, and `/api/dream/run`.

## Modes

| Mode | Intended Scope | Status |
|---|---|---|
| microdream | Short window, small candidate set | IMPLEMENTED v0 |
| nap | Medium window | IMPLEMENTED v0 |
| deep_dream | Larger/deeper dry-run report | IMPLEMENTED v0 |

## Salience Factors

Dream Mode should prioritize corrections, failures, contradictions, repeated relevance, stale loops, restore failures, and high operator usefulness. V0 implements deterministic candidate scoring and routing proposals; future scoring should feed restore outcomes.

## Safety

IMPLEMENTED: Dream Mode v0 does not require modelruntime/GPU and does not mutate canonical memory. Tests assert it does not depend on modelruntime/retrieval/vector semantics or canonical mutation tables.

## Gaps

- MISSING: Durable Dream report table.
- MISSING: Operator review workflow for Dream proposals.
- MISSING: Feedback from Dream reports into restore scoring.
- PLANNED: Governed commit mode through semantic syscalls only.

