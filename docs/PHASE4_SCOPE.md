# FORGE - Phase 4 Scope

## Objective

Phase 4 turns FORGE into a disciplined execution system with inspectable routing policy, reusable strategies, bounded automation, and explicit review control.

## Delivered

### Routing Policy Engine

- recommendation endpoint and persistence (`routing_policy_recommendations`)
- recommendation includes:
  - adapter
  - retrieval mode
  - strategy
  - packet shape guidance
  - approval preset and requirement
  - confidence, reasons, evidence
  - inferred-vs-direct evidence marker
  - operator override flag

### Execution Strategies

- persisted strategy records (`execution_strategies`)
- default strategy set seeded:
  - `local_summarize`
  - `repo_analysis`
  - `codex_implementation_handoff`
  - `claude_refactor_planning`
  - `context_regeneration`
  - `review_workflow`
- strategy editor in UI for packet rules, success criteria, retry guidance, approvals

### Automation Rules

- persisted rules and run history (`automation_rules`, `automation_history`)
- bounded action executor supports:
  - `create_job`
  - `generate_dossier_brief`
  - `create_review`
  - `suggest_strategy_adjustment` (advisory)
- dry-run preview and explicit matched/executed status

### Approval Presets

- seeded presets:
  - conservative
  - balanced
  - aggressive
- global preset setting (`approval_preset_global`)
- dossier profile approval override support
- policy UI for preset inspection and global selection

### Packet Optimization Guidance

- packet analysis endpoint and persistence (`packet_guidance_records`)
- issue heuristics:
  - oversized packet
  - under-scoped packet
  - insufficient context for write-intent
  - likely noise
- evidence payload stores context count/noise/risk signals

### Dossier-Specific Behavior

- dossier profile persistence (`dossier_profiles`)
- profile includes:
  - preferred strategies
  - preferred adapters
  - approval preset override
  - retrieval defaults
  - high-value/noisy files
  - routing notes
  - automation bindings
- dossier UI supports profile editing and review visibility

### Rich External Import + Reconciliation

- reconciliation persistence (`imported_execution_reconciliations`)
- captures:
  - changed files
  - failure reasons
  - unresolved issues
  - suggested next steps
  - agent notes
  - patch summary
  - review status
- import creation auto-seeds reconciliation + pending review records

### Review Workflows

- persisted review records (`review_records`)
- statuses:
  - pending
  - approved
  - rejected
  - deferred
- dedicated review page and queue actions
- review visibility in dashboard and dossier/job surfaces

### Operator Dashboard

- dashboard summary endpoint and UI route
- panels include:
  - active jobs
  - approvals pending
  - reviews pending
  - recent failures
  - recent imports
  - dossier health
  - automation activity
  - recent policy recommendations
  - system status map

### Failure Pattern Tracking

- failure analysis endpoint + snapshots (`failure_pattern_snapshots`)
- dimensions:
  - adapter
  - strategy (from job metadata)
  - retrieval mode
  - packet style
  - failure code
- advisory recommendations generated per snapshot

## UI Surfaces Added/Extended

- `Dashboard`
- `Policy`
- `Strategies`
- `Automation`
- `Reviews`
- `Dossiers` (policy profile + review integration)
- `Job Detail` (review integration)
- command bar verbs for new Phase 4 routes/actions

## Explicit Deferrals to Phase 5

- automated strategy tuning from policy recommendations
- confidence calibration based on longer run horizons
- native SSE transport for stream-heavy views
- richer diff parsing for imported execution reconciliation
