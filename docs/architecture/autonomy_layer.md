# Autonomy Layer (Phase 5.75)

Phase 5.75 adds bounded FORGE initiative without bypassing kernel controls.

Core rule:

**FORGE may initiate. FORGE may not secretly mutate.**

Expanded rule:

**Autonomy proposes or requests. Kernel validates. Charter authorizes. Budget limits. Approval gates escalate. FORGE commits only through syscalls. Everything is audited.**

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
  - in-memory repos for charters, intents, budgets, decisions, reservations, curiosity
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

Phase 5.75 lands repository interfaces + in-memory implementations for autonomy entities. Durable SQLite persistence for autonomy entities can be added incrementally in a later phase without changing policy/runner contracts.

## Future IRIS relationship

Future IRIS may propose intents and actions, but it still cannot:

- auto-approve itself
- bypass charter checks
- bypass budget checks
- bypass kernel syscall validation
- bypass scope and audit requirements
