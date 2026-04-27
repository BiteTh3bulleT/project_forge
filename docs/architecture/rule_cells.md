# Rule Cells

Status date: 2026-04-27.

Rule Cells are deterministic CPU-side reflex rules for FORGE. They are not agents, not a swarm, not LLM reasoning, and not a second authority system.

Rule Cells may emit structured advice, score adjustments, warnings, rejects, or stricter routing hints. They cannot commit truth, execute tools, call modelruntime, call the network, scan storage, or bypass kernel authority.

## Authority

The authority rule is simple:

> Existing FORGE authority wins.

Rule Cell outputs may only make routing stricter or add advisory evidence. They must never loosen or override:

- semantic syscall validation
- kernel truth authority
- gateway policy
- approval requirements
- capability requirements
- workspace or lane scope enforcement
- modelruntime degraded, unavailable, or safety policy

If an authoritative path says `reject`, `approval_required`, `scope_denied`, `capability_denied`, `degraded`, or `unavailable`, a Rule Cell output cannot convert that into `allow`, `route`, `proceed`, or equivalent. The engine rewrites conflicting permissive outputs back to the authoritative decision and records a warning.

## Model

The v0 domain model lives in `services/core/internal/aios/rulecells`.

A `RuleCell` includes:

- `id`, `name`, `description`
- `lane`, `phase`, `priority`, `enabled`
- `input_type`
- deterministic `condition`
- structured `output`
- `score_delta`, `weight`, `tags`, `explain`
- `version`, `metadata`, optional `max_latency_ms`

Rules are grouped into static `RulePack`s. Every pack exposes `pack_id` and `version`; every trace records the pack ids and versions that evaluated for the requested lane and phase.

## Lanes And Phases

Supported lanes:

- `kernel`
- `neural`
- `arterial`
- `lymphatic`
- `operator`
- `runtime`

Supported phases:

- `syscall_validation`
- `ingest_classification`
- `event_admission`
- `restore_scoring`
- `working_memory_admission`
- `model_routing`
- `gateway_precheck`
- `provider_cooldown`
- `replay_selection`
- `salience_scoring`
- `memory_tier_routing`
- `repair_selection`
- `attention_routing`

## Outputs

Rule Cells emit structured outputs only:

- `RouteDecision`
- `ScoreAdjustment`
- `PolicyDecision`
- `MemoryProposal`
- `AttentionSignal`
- `RepairProposal`
- `ModelRoutingHint`
- `FreshCompileRequired`
- `VerifierRequired`
- `BackgroundDefer`
- `RejectDecision`

No output type has a durable write method or arbitrary callback.

## Conditions

V0 supports only simple deterministic conditions:

- equality
- contains
- numeric comparison
- age threshold
- status match
- tag match
- token-overlap threshold
- boolean flag
- provider status match
- risk class match

There is no scripting, unsafe eval, dynamic code loading, or LLM rule judgment.

## Traces

Each run emits a compact `RuleTrace`:

- `trace_id`
- `lane`, `phase`, `input_id`
- `started_at`, `completed_at`, `latency_ms`
- `rules_evaluated`
- `rule_packs`
- `matched_rules`
- `outputs`
- `warnings`

Default traces store matched rule details and outputs. Debug mode may include non-matches. V0 does not create one database row per rule evaluation; integrations summarize traces inside existing non-canonical evidence payloads.

## Score Caps

Restore scoring and Dream salience both bound Rule Cell effects:

- individual restore adjustment cap: `0.06`
- total restore adjustment cap: `0.12`
- final restore score clamp: `0.0..1.0`
- individual Dream salience adjustment cap: `0.08`
- total Dream salience adjustment cap: `0.15`
- final salience clamp: `0.0..1.0`

These caps keep Rule Cells from replacing deterministic base scoring.

## Failure Behavior

Rule engine errors are explicit warnings, not silent failures.

- `COMPILE_CONTEXT` continues deterministic base restore scoring.
- Dream Mode continues deterministic base salience and dry-run report generation.
- Neither path commits canonical truth because of a Rule Cell result.

Phase 8 restore outcome feedback is separate non-canonical evidence. Rule Cells may see outcome-derived facts in restore/Dream inputs, but those facts remain bounded advisory signals and cannot loosen authority or directly mutate memory.

## Adding A Rule Safely

Add rules only to static packs in code for v0. A safe rule:

- uses a deterministic condition
- emits one supported output type
- has a pack id/version and rule version
- is lane/phase scoped
- has a bounded score delta if it scores
- does not depend on I/O, modelruntime, network, filesystem, or DB scans
- cannot make an authoritative denial more permissive

Add or update tests for rule matching, trace output, failure behavior, and any score cap touched by the rule.
