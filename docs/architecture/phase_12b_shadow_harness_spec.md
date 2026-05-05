# Phase 12B Shadow Harness Specification

Status: Phase 12B implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`; Phase 12C hardening implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`.

## Scope

Phase 12B scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Phase 12B implements a read-only shadow harness that observes selected live metadata and generates diagnostic reports. It must not mutate live state, affect live responses, execute tools, call modelruntime, perform retrieval, call embeddings, write memory, compile live context, or create a second authority path.

## Feature Flag

Implemented flag: `FORGE_K_SHADOW_MODE_ENABLED=false`.

Config defaults:

- disabled by default
- no shadow adapters active while disabled
- no shadow diagnostics generated while disabled
- no route or API behavior changes while disabled
- no code path may require shadow mode for live request success

## Disabled-By-Default Behavior

When disabled:

- live requests execute exactly as they do before Phase 12B
- no shadow report is generated
- no adapter is called
- no diagnostic sink is written
- no response, status code, header, or public API shape changes

## Implemented Touchpoint

The Phase 12B implementation intentionally observes one low-risk live touchpoint:

- `/health` request metadata only
- observation runs after the health response is written
- captured metadata is bounded to route, method, touchpoint, workspace id, optional request id, and a short diagnostic summary
- reports are stored only in a bounded in-memory sink with no public API
- report failures are best-effort and cannot fail the live request
- no request body, response body, prompt, model output, tool payload, retrieval result content, or memory content is captured
- Phase 12C adds explicit disabled sink support, expanded unsafe metadata rejection, raw content key rejection, max metadata string length enforcement, and additional no-effect tests.

## Future Candidate Live Request Types

Later phases may consider:

- chat message submission metadata
- assistant completion metadata
- governed gateway trace metadata
- live retrieval/search/embedding record metadata already produced by live paths
- live context compile metadata already produced by live paths
- audit/correlation trace metadata
- modelruntime trace metadata already produced by live paths

Explicitly excluded:

- direct tool requests from FORGE-K
- direct modelruntime requests from FORGE-K
- direct retrieval/search/embedding requests from FORGE-K
- memory mutation requests
- controllane mutation requests
- backup restore or release actions

## Captured Metadata

Allowed metadata:

- workspace id
- correlation id
- trace id
- route id or live path id
- request class
- live owner component
- stable object refs
- existing retrieval/result refs
- existing model/runtime refs
- existing gateway/audit refs
- timing and status summaries
- redacted warning summaries

Forbidden metadata:

- raw secrets
- raw credentials
- large raw content blobs
- unredacted prompts unless separately approved
- full model output unless already public and explicitly bounded

## Mirrored Evidence Refs

Later shadow phases may mirror refs for:

- memory notes
- memory observations
- retrieval runs/results
- search chunks/files
- embedding records
- gateway invocation records
- audit records
- context packet snapshots
- modelruntime request/result records

Mirroring a ref does not admit evidence, write memory, create truth, or affect response composition.

## Shadow Report

The implemented diagnostic report includes:

- report_id
- generated_at
- workspace_id
- correlation_id
- live_path
- observed refs when provided
- no-effect validation result
- diagnostic-only comparison report
- no_effect_verified
- warnings

Reports are diagnostics only.

## Storage Location

Implemented first storage: in-memory diagnostic sink with bounded retention.

If persistence is approved, use a dedicated diagnostic table or artifact record clearly marked non-authoritative. Do not store reports in canonical memory tables.

## Retention Policy

Suggested defaults:

- bounded count per workspace
- bounded report size
- short retention window
- drop oldest on limit
- reject secret-looking metadata

## No User-Visible Effect Guarantee

Tests must prove the live response is unchanged with shadow disabled and enabled. Shadow reports cannot change response text, status, headers, route selection, model selection, tool approval state, retrieval selection, memory writes, or audit authority.

## No Mutation Guarantee

Phase 12B must not:

- write memory
- write controllane semantic objects
- write gateway invocation records
- alter approval decisions
- alter permissions or lanes
- alter modelruntime state
- execute retrieval/search/embedding
- compile live context through FORGE-K

## No Tool / Model / Retrieval Execution Guarantee

Adapters observe existing live traces only. They cannot call gateway execution methods, modelruntime generate/chat methods, retrieval/search query methods, embedding provider methods, or memory write methods.

## Required Tests

- default flag disabled
- enabled flag does not change response
- route inventory unchanged
- no public API response shape changes
- no tool execution
- no modelruntime calls from FORGE-K
- no retrieval/search/embedding execution from FORGE-K
- no memory writes
- no controllane writes
- diagnostic report generation behind flag
- report failure isolation
- redaction/rejection of secret-looking metadata
- kill switch stops report generation

Implemented Phase 12B tests cover the default flag, enabled flag parsing, route inventory key set stability, `/health` response status/body/header equivalence, diagnostic report generation behind the flag, sink failure isolation, bounded report retention, secret-looking metadata rejection, no-effect policy rejection, and forbidden imports.

Implemented Phase 12C tests additionally cover disabled sink behavior, `authorization`/`cookie`/`session` metadata rejection, raw body/content/prompt metadata rejection, oversized metadata rejection, all represented side-effect policy flags, no public diagnostics route, disabled `/health` equivalence, and non-`/health` no-observation behavior.

## What Not To Do

- Do not add public routes unless separately approved.
- Do not use shadow output in live response composition.
- Do not treat reports as truth.
- Do not create a live FORGE-K authority path.
