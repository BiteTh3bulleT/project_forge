# Phase 12B Shadow Harness Specification

Status: Phase 12B implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`; Phase 12C hardening implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`; Phase 12D controlled expansion design implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`; Phase 12E route-envelope metadata implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`; Phase 12F route-envelope hardening implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`; Phase 12G chat metadata expansion design implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`; Phase 12H chat metadata shadowing implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`; Phase 12I chat metadata hardening implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`; Phase 12J retrieval metadata expansion design implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`; Phase 12K-L retrieval metadata shadowing implemented and hardened as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / HARDENED_IN_PASS`.

## Scope

Phase 12B scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Phase 12B implements a read-only shadow harness that observes selected live metadata and generates diagnostic reports. It must not mutate live state, affect live responses, execute tools, call modelruntime, perform retrieval, call embeddings, write memory, compile live context, or create a second authority path.

## Feature Flag

Implemented flags:

- `FORGE_K_SHADOW_MODE_ENABLED=false`
- `FORGE_K_SHADOW_CHAT_METADATA_ENABLED=false`
- `FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED=false`

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

## Implemented Touchpoints

The Phase 12B implementation intentionally observes one low-risk live touchpoint:

- `/health` request metadata only
- observation runs after the health response is written
- captured metadata is bounded to route, method, touchpoint, workspace id, optional request id, and a short diagnostic summary
- reports are stored only in a bounded in-memory sink with no public API
- report failures are best-effort and cannot fail the live request
- no request body, response body, prompt, model output, tool payload, retrieval result content, or memory content is captured
- Phase 12C adds explicit disabled sink support, expanded unsafe metadata rejection, raw content key rejection, max metadata string length enforcement, and additional no-effect tests.

Phase 12E adds one more disabled-by-default touchpoint:

- route-envelope metadata from matched API routes after the live handler returns
- captured metadata is bounded to method, matched route pattern, normalized route class, duration, safe request id, and diagnostic markers
- status code capture is intentionally skipped because this phase avoids response writer wrapping
- `/health` keeps the Phase 12B per-handler observer and is skipped by route-envelope middleware
- no raw query strings, request bodies, response bodies, prompts, model outputs, tool payloads, retrieval content, memory content, auth headers, cookies, or secrets are captured
- reports stay in the same bounded in-memory sink with no public API
- Phase 12F hardens route-envelope observation without adding touchpoints: matched route-pattern safety, route class normalization, reserved metadata key rejection, raw query/request URI rejection, expanded header/secret/content redaction, deterministic scalar metadata enforcement, bounded retention tests, sink failure isolation, response equivalence tests, SSE mount/order stability, and timeout stability.

Phase 12H adds one chat metadata touchpoint, and Phase 12I hardens that same touchpoint without expanding it:

- chat metadata observation requires both `FORGE_K_SHADOW_MODE_ENABLED=true` and `FORGE_K_SHADOW_CHAT_METADATA_ENABLED=true`
- the touchpoint is the existing chat message POST handler after live handler ownership is established
- reports capture bounded metadata only: operation kind, safe ids/refs, role class, stream class, safe model id, request/workspace ids, counts, and diagnostic markers
- no chat content, prompt, completion, model output, request body, response body, tool payload, retrieval content, or memory content capture is approved
- Phase 12H/12I coverage is recorded in `docs/testing/phase_12h_chat_metadata_shadow_tests.md` and `docs/reviews/phase_12i_chat_metadata_shadow_hardening.md`

Phase 12K-L adds and hardens one retrieval metadata touchpoint:

- retrieval metadata observation requires both `FORGE_K_SHADOW_MODE_ENABLED=true` and `FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED=true`
- the touchpoint runs after the live `/api/retrieval/runs` handler has already created the retrieval run
- reports capture bounded metadata only: retrieval run/result refs, source type/ref, result count, selected count, bounded score summary, ranking position, retrieval strategy, index type, safe embedding model id, timing, and diagnostic markers
- no source text, chunk text, document content, raw query text, search snippet, embedding/vector, RAG output, prompt, model output, request body, response body, memory content, secret, route/API behavior, or user-visible output change is approved
- Phase 12K-L coverage is recorded in `docs/testing/phase_12k_retrieval_metadata_shadow_tests.md` and `docs/reviews/phase_12kl_retrieval_metadata_shadow_hardening.md`

## Future Candidate Live Request Types

Later phases may consider:

- chat message submission metadata
- assistant completion metadata
- governed gateway trace metadata
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
- prompts
- model output
- request bodies
- response bodies
- raw user queries
- source, chunk, document, file, retrieval, memory, embedding, vector, tool, or RAG content

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

Implemented Phase 12E tests cover route-envelope disabled/enabled behavior, typed route-envelope reports, route class normalization, metadata redaction, no body capture, `/api/meta` response equivalence, invalid POST body non-capture, route inventory stability, no public diagnostics route, and the existing SSE mount/order guard.

Implemented Phase 12F tests additionally cover matched-pattern preference over raw paths, unsafe raw dynamic route-pattern rejection when no template is available, provided route class normalization to known classes, rejection of query/request URI metadata, rejection of metadata that reintroduces path or route pattern values, rejection of non-deterministic metadata values, bounded retention drop-oldest behavior, sink-failure isolation, `/forge` and conditional `/v1` response equivalence, timeout middleware stability, and search/controllane side-effect policy rejection.

## Phase 12D Handoff

Phase 12D is a docs-only controlled expansion design. It does not add code, route observation, public APIs, route-envelope hooks, adapters, feature flags, persistent storage, or live authority migration.

The Phase 12E touchpoint is route envelope metadata. The route-envelope scope is method, matched route template, route class, timing summary, safe request ids when available, and no-effect validation only.

After Phase 12I:

- `/health` remains supported.
- route-envelope metadata is implemented behind `FORGE_K_SHADOW_MODE_ENABLED`.
- route-envelope hardening is complete without adding new live touchpoints.
- chat metadata is implemented behind `FORGE_K_SHADOW_MODE_ENABLED` and `FORGE_K_SHADOW_CHAT_METADATA_ENABLED`.
- chat metadata hardening is complete without adding new live touchpoints.
- no chat content capture is approved.
- no request or response body capture is approved.
- no raw query, raw header, cookie, authorization, secret, prompt, model output, retrieval content, embedding vector, search chunk, or memory content capture is approved.
- no retrieval/search/embedding execution is approved.
- no modelruntime calls are approved.
- no gateway/tool execution is approved.
- no memory write or controllane mutation is approved.
- retrieval metadata diagnostics are implemented/hardened after Phase 12K-L, but content observation remains forbidden.
- no retrieval/search/embedding execution from FORGE-K is approved.

The Phase 12D design is in `docs/architecture/phase_12d_controlled_shadow_expansion_design.md`; the touchpoint decision is in `docs/reviews/phase_12d_touchpoint_selection.md`; the Phase 12E and Phase 12F test coverage is recorded in `docs/testing/phase_12e_shadow_route_envelope_tests.md`; the Phase 12F hardening review is recorded in `docs/reviews/phase_12f_route_envelope_shadow_hardening.md`; the Phase 12G chat metadata design is recorded in `docs/architecture/phase_12g_chat_metadata_expansion_design.md`; the Phase 12I chat metadata hardening review is recorded in `docs/reviews/phase_12i_chat_metadata_shadow_hardening.md`; the Phase 12J retrieval metadata design is recorded in `docs/architecture/phase_12j_retrieval_metadata_expansion_design.md`; the Phase 12K-L retrieval metadata hardening review is recorded in `docs/reviews/phase_12kl_retrieval_metadata_shadow_hardening.md`.

## What Not To Do

- Do not add public routes unless separately approved.
- Do not use shadow output in live response composition.
- Do not treat reports as truth.
- Do not create a live FORGE-K authority path.
