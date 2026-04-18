# Routing Insights and Policy

## Purpose

FORGE now has two advisory layers:

- `routing_insights` (Phase 3 trend advisories)
- `routing_policy_recommendations` (Phase 4 task-time policy recommendations)

Neither layer auto-executes risky work.

## Routing Insights (Trend Layer)

Inputs:

- adapter evaluation aggregates (success/quality/retry)
- retrieval usefulness/noise evidence
- optional dossier scope

Storage: `routing_insights`

API:

- `POST /api/insights/generate`
- `GET /api/insights`

UI route: `/insights`

## Policy Recommendations (Execution Layer)

Inputs:

- requested `taskType` / objective
- optional dossier id
- optional strategy override
- strategy defaults
- dossier profile overrides
- historical evaluation evidence

Storage: `routing_policy_recommendations`

Fields include:

- strategy id
- adapter
- retrieval mode
- packet shape guidance
- approval preset id + approval required
- confidence
- reasons
- evidence
- inferred flag (when evidence is thin)
- operator override allowed flag

API:

- `POST /api/policy/recommend`
- `GET /api/policy/recommendations`

UI route: `/policy`

## Approval + Automation Interaction

- policy recommendations may suggest approval preset/level
- automation rules do not bypass approvals
- write/command risk still requires explicit gate decisions

## Guardrails

- recommendations remain advisory
- no silent scope expansion
- no automatic escalation from read-only to write/command execution
- operator remains final authority for risky actions
