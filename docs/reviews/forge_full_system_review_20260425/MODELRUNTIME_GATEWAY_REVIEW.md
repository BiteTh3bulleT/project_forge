# Modelruntime and Gateway Review

## Modelruntime

GOOD: Modelruntime is real and governed for inference:

- manifest parsing
- registry/store
- fake backend
- llama.cpp endpoint backend
- OpenAI-compatible/vLLM-compatible path
- load/unload
- queue/admission control
- prompt/output/request bounds
- actor/source/workspace policy
- audit and usage accounting

PARTIAL: Streaming is explicitly unsupported.

RISK: Model management APIs directly mutate runtime registry/filesystem metadata without gateway-equivalent approval.

RISK: Remote provider discovery persists provider-advertised models at startup. This should be read-only by default or require operator import approval.

RISK: `/v1/chat/completions` is direct modelruntime API when enabled. Runtime policy exists, but auth/rate/external caller posture needs tightening.

RISK: Provider URL policy is thin. TEI/OpenAI-compatible/vLLM endpoints should have scheme/host allow policy if settings become user-writable.

## Gateway

GOOD: Gateway has strong central execution logic:

- capability registry
- risk classification
- lane/profile policy
- approval handling
- dry-run support
- audit
- artifact summaries
- gateway invocation records
- legacy adapter compatibility through gateway

RISK: Approval grant binding is too loose when `JobID` is absent.

RISK: Dangerous capabilities are mostly approval-gated, but service-specific tests should be expanded for every concrete tool family.

PARTIAL: `model.*` gateway aliases are still a known follow-up; model management remains outside gateway.

## Autonomy/Rule Agents

GOOD: Rule agents are propose-only and have destructive placeholder guards.

PARTIAL: Rule agent coverage is narrow.

RISK: Autonomy gateway budget checks appear to inspect budgets but not consistently consume usage for tool calls.

## Recommended M4/Gateway Work

1. Bind approval grants to request fingerprint.
2. Add model management capabilities and approval gates.
3. Add streaming with bounded cancellation/backpressure.
4. Add provider cooldown/blacklist tests around real failure paths.
5. Make remote discovery read-only by default.
6. Add per-provider URL allow policy.
7. Persist autonomy tool budget consumption.

