# Hyperlane

Status date: 2026-04-27.

Hyperlane is FORGE's deterministic reflex-routing layer over Rule Cells. It is CPU-local and low latency. It routes obvious decisions before expensive context assembly, modelruntime calls, gateway execution, or Dream deep processing are needed.

Hyperlane is not an agent system. It owns no durable state, spawns no processes, performs no model reasoning, and cannot bypass kernel authority.

## Routers

The v0 router helpers are thin wrappers over `rulecells.Engine.Run`:

- Neural Hyperlane: input/event tagging surface for future deterministic classification.
- Arterial Hyperlane: restore scoring, fresh-compile hints, verifier/no-model hints.
- Kernel Hyperlane: advisory precheck for scope, approval, capability, and provenance.
- Runtime Hyperlane: provider cooldown, degraded/unavailable, defer, and model-routing hints.
- Lymphatic Hyperlane: Dream salience, memory tier routing, and repair hints.
- Operator Hyperlane: attention signals for blocked loops, failed jobs, degraded runtime, and Dream review.

Only Arterial restore scoring and Lymphatic Dream routing are wired into real v0 runtime paths in this pass. Runtime and operator packs exist as deterministic router coverage but do not change deep runtime or UI behavior.

The chat API also uses a narrow Hyperlane-style no-model classifier for latency. It is not a new Rule Cell integration point and does not change runtime authority. It classifies obvious status, diagnostics, mode, degraded-state, and restore-inspector requests so they can return structured CPU-local replies without modelruntime, gateway execution, or heavy context assembly.

## Deterministic Intent Routing

The gateway fallback natural-language parser is now represented as a Hyperlane deterministic intent router in Go. It turns simple operator requests into typed route proposals before modelruntime is used. It is CPU-only and performs no I/O.

The v0 intent model records:

- `id`
- `type`
- `confidence`
- `lane`
- `route`
- `requires_gateway`
- `requires_model`
- `requires_approval_hint`
- `risk_class`
- `arguments`
- `warnings`
- `matched_rule`
- `trace`

Supported v0 intent types include structured no-model queries (`status_query`, `diagnostics_query`, `restore_inspection`, `dream_report_inspection`, `modelruntime_status`), gateway-bound file/process proposals (`mkdir`, `read_file`, `write_file`, `list_directory`, `run_command`, `generate_template`, `gateway_tool_request`), and `unknown`.

Route hints use existing FORGE routes and gateway tool ids:

- `mkdir` -> `fs.mkdir`
- `write_file` / `generate_template` -> `fs.write`
- `read_file` -> `fs.read`
- `list_directory` -> `fs.list`
- `run_command` -> `proc.run`
- status/diagnostics/restore/Dream/modelruntime inspection -> structured no-model routes

The router does not execute tools. Gateway policy, capability checks, workspace scope, approvals, and audit still decide whether any gateway-bound proposal can run. Shell command intents are always marked high-risk with an approval hint.

Each parse emits compact trace data with parser version, matched rule, confidence, route, warnings, and rejected reason for unsafe or unknown input.

Phase 8 feeds non-canonical restore outcome facts into the same two paths. Arterial restore scoring may use prior helpful/stale/harmful/corrected outcomes as bounded utility evidence. Lymphatic Dream routing treats outcome events as replay candidates for memory-gap, evidence-review, or promotion proposals. These facts remain advisory and cannot replace scope filtering, kernel validation, or canonical commit authority.

## Integration Rules

Routers must:

- run on CPU/RAM only
- avoid network, modelruntime, GPU, DB scans, and filesystem work
- filter by lane and phase before rule evaluation
- evaluate rules by priority, then id for deterministic tie-breaking
- emit compact traces
- respect latency budgets
- fail open to existing deterministic base behavior with explicit warnings

Routers must not:

- commit semantic truth
- execute tools
- call modelruntime
- change approval or capability requirements to be more permissive
- act as the workspace boundary

Workspace and lane scope remain enforced by store/query/authority logic before Rule Cells run.

## V0 Rule Packs

Static starter packs:

- `forge.kernel.authority.v0@0.1.0`
- `forge.neural.classification.v0@0.1.0`
- `forge.arterial.restore.v0@0.1.0`
- `forge.lymphatic.dream.v0@0.1.0`
- `forge.runtime.routing.v0@0.1.0`
- `forge.operator.attention.v0@0.1.0`

Pack id/version appears in traces so score changes can be explained later.

## Latency

V0 packs carry small millisecond budgets. A budget miss is a warning in the trace, not a retry trigger. Hyperlane does not retry rules and does not fan out to independent workers.

Chat latency traces record `hyperlane_ms`, `context_budget_class`, `output_mode`, and `model_calls_avoided` so operators can verify that deterministic requests stayed on the CPU fast path.
