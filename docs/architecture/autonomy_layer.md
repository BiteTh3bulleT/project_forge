# Autonomy Layer (Phase 5.75)

Status: `[LIVE] / [PARTIAL]`. The autonomy package exists in the live AI-OS path for bounded intent, policy, budget, and kernel-mediated syscall execution. It is not unrestricted self-modification: durable changes still require Control Lane validation, audit, and configured authority.

Phase 5.75 adds bounded FORGE initiative without bypassing kernel controls.

Core rule:

**FORGE may initiate. FORGE may not secretly mutate.**

Expanded rule:

**Autonomy proposes or requests. Kernel validates. Charter authorizes. Budget limits. Approval gates escalate. FORGE commits only through syscalls. Everything is audited.**

Phase 14 Lymphatic note: autonomy maintenance dry-run sweeps can expose proposal-only Lymphatic metadata for maintenance reports and cleanup proposals. Those dry-run proposals carry no cleanup execution authority and cannot claim commit authority. Non-dry-run autonomy maintenance remains existing live autonomy authority, not FORGE-K Lymphatic Lane authority.

## Why this layer exists

Before Phase 5.75, FORGE could process user/system ingest and deterministic rule flows but could not safely self-initiate internal maintenance. The Autonomy Layer adds a constrained internal initiative path for:

- memory maintenance notes/links
- stale loop review
- context-preparation preflight
- contradiction review proposals
- projection repair diagnostics

## Autonomy levels

- `level_0_observe_only`: inspect/report only.
- `level_1_internal_preparation`: preflight diagnostics and non-authoritative prep.
- `level_2_propose_semantic_actions`: create intents/proposals without commit.
- `level_3_auto_commit_safe_internal`: commit low-risk internal actions through kernel when chartered and budgeted.
- `level_4_approval_required`: approval gate required before action.
- `level_5_delegated_mission`: bounded mission operation under explicit mission charter.

## Autonomy modes

- `off`: no self-initiated commit path.
- `observe`: report/preflight only.
- `propose`: enqueue intents/proposals only.
- `maintain`: auto-commit only low-risk chartered actions.
- `mission`: delegated operation under explicit mission charter.

There is no unrestricted bypass mode.

## Core components

Implementation path: `services/core/internal/aios/autonomy`

- `domain/autonomy.go`
  - autonomy levels/modes/risks
  - charter/intent/budget/decision models
  - typed autonomy errors
- `autonomy/repositories.go`
  - repository interfaces + in-memory implementations
- `autonomy/sqlite_repositories.go`
  - SQLite-backed repositories for charters, intents, budgets, decisions, reservations, curiosity
- `autonomy/risk.go`
  - deterministic risk classification + guardrail escalation
- `autonomy/charter.go`
  - charter action/condition evaluation
- `autonomy/intent_queue.go`
  - deterministic intent lifecycle transitions
- `autonomy/budget.go`
  - budget check/reserve/consume/release behavior
- `autonomy/policy_evaluator.go`
  - autonomy authorization decision engine
- `autonomy/runner.go`
  - self-initiated syscall runner (kernel-mediated)
- `autonomy/rule_agents.go`
  - deterministic rule-agent proposals -> intents
- `autonomy/ingest_integration.go`
  - bounded ingest-triggered autonomy pass adapter
- `autonomy/curiosity.go`
  - curiosity queue (optional inquiry staging)
- `autonomy/defaults.go`
  - conservative default charters and budgets
- `autonomy/explain.go`
  - explain intent/decision/budget/charter decisions

## Self-initiated syscall flow

1. Internal source (FORGE/rule agent/cell/system/future_iris) creates an intent.
2. Policy evaluator checks scope, charter status/rules, risk, guardrails, budget, and kernel dry-run preview.
3. Decision outcomes:
   - `allow_auto_commit`
   - `allow_propose_only`
   - `approval_required`
   - blocked/deny variants
4. Runner behavior:
   - reserve budget
   - kernel syscall commit for allowed actions
   - consume/release budget based on result
   - update intent lifecycle
   - persist decision trace
5. Approval-required actions are escalated through approval gate interface and remain blocked/pending.

All semantic writes remain syscall-mediated.

## Safety guardrails

Guardrails force block or approval escalation for categories such as:

- destructive/delete categories
- external send/export categories
- permission/credential change categories
- cross-workspace mutation
- high-priority loop closure
- archiving active-state evidence
- weak provenance risk elevation
- recursive autonomy depth over cap

Unknown action categories default upward in risk (high/approval path).

## Ingest integration

`IngestPipeline` now supports optional autonomy callback:

- `IngestPipelineOptions.AutonomyPass`
- `IngestPipelineOptions.MaxAutonomyDepth`

Behavior:

- skipped in `dry_run` and `validate_only`
- depth-capped via request metadata `autonomyDepth`
- autonomy run summaries are attached to `IngestResult.autonomyRuns` and `truthDiagnostics.autonomyRuns`

## Audit and traceability

Autonomy metadata is propagated into syscall request metadata:

- `intentId`
- `charterId`
- `budgetId`
- `decisionId`
- source/provenance/correlation/trace ids

Explanation APIs expose intent and decision history for inspection.

## Current persistence note

Phase 5.99 adds durable SQLite-backed autonomy repositories and wires the default maintenance loop to them. Repository records are stored under `settings` keys with `autonomy_repo.*` prefixes. In-memory fallback remains only for nil-DB/test scenarios. Backup/export restore parity for these records is still pending.

## Future IRIS relationship

Future IRIS may propose intents and actions, but it still cannot:

- auto-approve itself
- bypass charter checks
- bypass budget checks
- bypass kernel syscall validation
- bypass scope and audit requirements

## Tool surface integration (Phase 5.9)

Autonomy does not directly execute host tools. Self-initiated tool calls are governed by the AI-OS tool surface:

- tool capability registry resolves `domain.primitive` capability ids
- gateway policy checks capability status/risk/resource limits
- autonomy context (intent/charter/budget/source) is required for self-initiated paths
- approval-only/high-risk/critical capabilities escalate to approval
- tool output is evidence only; semantic truth updates still require semantic syscalls

This keeps autonomy bounded while enabling safe maintenance diagnostics and preparation workflows.
