# Current Baseline Gate (Phase 5.996)

Date: 2026-04-22
Scope: readiness decision after convergence hardening.

## Readiness answers

| Question | Answer | Rationale |
|---|---|---|
| Ready for Phase 6 context compiler? | conditional | Core cutover blockers are reduced materially (restore parity + transactional restore + authoritative VSA source tracking + gateway-only tool execution + correlation-first trace report), but trace UX remains partial and VSA restore remains intentionally export-only. |
| Ready for deeper Nix/NixOS integration? | no | Nix checks/builds cannot be validated in this environment (daemon unavailable). |
| Ready for more autonomy/tool freedom? | conditional | Guardrails are stronger and tool execution is gateway-only, and API-level traceability is stronger, but operator-facing trace UX remains partial. |
| Ready for IRIS integration? | conditional | Only as proposal-only source under existing gateway/syscall policy boundaries. |
| Ready for external/demo use? | conditional | Controlled demos are viable; unrestricted posture is not ready. |
| Ready to run local models without Ollama? | yes | Model Runtime M3 now governs local and compatible remote inference through `/forge/models*` plus gated `/v1/*`, with scheduler, limits, management workflows, lifecycle policy, audit, non-streaming inference, and approval-required managed delete-file flow. |
| Is runtime authority clearer than before? | yes | Restore/apply guarantees are clearer (`atomicScope` + non-DB warnings), tool execution authority is gateway-only, and model runtime now owns managed model registration/lifecycle under one service boundary. |

VSA lane status: **authoritative source** (not generated, not optional).

## Must-fix blockers

1. Surface the consolidated correlation report in operator flows so trace/explain is not API-only.
2. Continue Model Runtime M4 work (streaming, stronger backend/process supervision, and gateway capability registry aliasing).

## Should-fix next

1. Add desktop/operator affordances for the consolidated `/api/audit/trace/{correlationId}` report.
2. Expand rule-agent set or explicitly defer missing agents in docs/matrix.
3. Broaden JS/TS lint/test coverage beyond the current desktop-focused Vitest and TypeScript lanes.
4. Register `model.*` tool capabilities in gateway capability registry with honest M3 status transitions.

## Dangerous unresolved issues

1. VSA exported sections remain non-restorable by explicit export-only policy in this phase.
2. Restore is transactional only for DB-supported sections; non-DB side effects are explicitly warned, not globally rollback-managed.
   - Restore payload now reports `globalAtomic=false` and per-section `nonDbSideEffects`.
3. JS/TS lint/test coverage remains shallow: root `test:js`, `lint:js`, and `validate:js` now exist, but `lint:js` is TypeScript-only and coverage remains desktop-focused.
4. Model file deletion is intentionally narrow: only managed model-home directories can be deleted, and the flow requires high-risk model-management approval.

## Operational guardrails (not blockers)

1. `npm run core`, `npm run smoke`, `build:core`, `test:core`, and `vet:core` enforce strict `--require-tracked` VSA preflight by design.

## Recommended next prompt

`A. Model Runtime M4 — embeddings/rerank/backend expansion`
