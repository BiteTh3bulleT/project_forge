# FORGE-K Live Integration Design

Status: Phase 12A implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`; Phase 12B implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`; Phase 12C implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`; Phase 12D implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`; Phase 12M-Q implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / ADVISORY_DIAGNOSTIC_ONLY`.

Phase 12B implements the first disabled-by-default read-only live touchpoint. Phase 12C hardens that implementation. Phase 12D selects route envelope metadata as the recommended next controlled expansion for a future Phase 12E. Later Phase 12 metadata passes add route, chat, retrieval, and advisory diagnostics while preserving the same no-authority boundary. These phases do not authorize live authority migration.

## Executive Summary

FORGE-K Phase 1-11G is implemented and tested in simulator, research, tooling, governance, integration-prep, and shadow-design layers. The live daemon still uses the existing AI-OS, gateway, permissions, lane, audit, modelruntime, retrieval, embeddings, memory, search, and API authority paths.

Phase 12A designed the first live integration path. Phase 12B implements the smallest read-only shadow harness: `/health` request metadata can be copied into a bounded in-memory diagnostic sink when `FORGE_K_SHADOW_MODE_ENABLED=true`. Phase 12C hardens that observer. Phase 12D is design-only and selects route envelope metadata as the recommended next candidate for a future Phase 12E. Phase 12M-Q adds an internal advisory report that consumes only existing safe diagnostics when `FORGE_K_SHADOW_ADVISORY_ENABLED=true` and global shadow mode is also enabled. The design preserves live authority by allowing only passive observation of already-executing live paths, diagnostic report generation, advisory-only summaries, and disabled-by-default activation. It does not wire FORGE-K into live authority.

## Current Simulator / Live Split

FORGE-K simulator packages live under `services/core/internal/forgek` and adjacent research/tooling paths. They define target architecture contracts for Kernel authority, Courthouse admission, Memory Palace retrieval shape, Semantic Algebra, Snapshots, Context Compiler, KV metadata, Runtime Boundary, Lymphatic Lane, Consensus Mesh, integration readiness, and shadow harness reports.

Live daemon authority remains outside FORGE-K:

- API routes are owned by `services/core/internal/api`.
- Live semantic mutation is owned by `services/core/internal/aios/controllane`.
- Tool execution is owned by `services/core/internal/gateway`.
- Policy gates are owned by `services/core/internal/permissions` and `services/core/internal/lanes`.
- Audit is owned by `services/core/internal/audit`.
- Live model generation is owned by `services/core/internal/modelruntime`.
- Retrieval, search, embeddings, and memory remain live evidence/context infrastructure unless committed through existing governed paths.

## Phase 12A Scope

Phase 12A is design only. It creates architecture and review documents that define the intended Phase 12B shadow harness boundary, adapters, touchpoints, storage policy, feature flag policy, rollback strategy, and tests.

Phase 12A does not add feature flags in code, routes, adapters, syscalls, imports, services, storage tables, public APIs, live observation, live retrieval, modelruntime calls, tool execution, memory writes, or user-visible output changes.

## Phase 12B Proposed Scope

Phase 12B scope is `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

The first Phase 12B candidate may observe:

- chat request metadata and response metadata after the live path has ownership
- live route identity and correlation metadata
- existing retrieval/search/embedding/VSA record references, if already produced by live code
- existing context compile metadata, if already produced by live code
- gateway trace records, without executing tools
- audit trace records, without writing authority records unless an explicit diagnostic sink is approved
- modelruntime trace metadata, without calling modelruntime
- memory evidence refs, without writing memory

Phase 12B must not affect live responses, routes, public API response shapes, approvals, gateway execution, modelruntime behavior, retrieval execution, embeddings, memory writes, or controllane mutation.

## Phase 12B Implementation Record

The implemented Phase 12B scope is intentionally narrower than the candidate map:

- package: `services/core/internal/forgekshadow`
- feature flag: `FORGE_K_SHADOW_MODE_ENABLED`, default `false`
- selected touchpoint: `/health` request metadata only
- captured data: workspace id, request id if provided, method, route, touchpoint label, and diagnostic summary
- sink: bounded in-memory diagnostic reports only, with no public API
- failure handling: best-effort observer; sink, redaction, or policy failures do not fail the live request
- simulator contract reuse: imports `services/core/internal/forgek/shadowharness` for no-effect validation only

Phase 12B does not capture request or response bodies, add public routes, change route inventory, change response status/body/header shape, execute tools, call modelruntime, execute retrieval/search/embeddings, write memory, mutate controllane state, or alter gateway/permission/lane/audit authority.

## Phase 12C Hardening Record

Phase 12C keeps the Phase 12B live scope unchanged. `/health` request metadata remains the only live touchpoint.

Hardening completed:

- explicit disabled diagnostic sink support
- expanded unsafe metadata rejection for authorization, cookie, and session terms
- raw content key rejection for body, prompt, completion, model output, and content fields
- maximum metadata string length enforcement
- no public diagnostics route regression test
- disabled and enabled `/health` response equivalence tests
- non-`/health` no-observation test
- root FORGE-K forbidden live import test
- Phase 12C review record in `docs/reviews/phase_12c_shadow_diagnostics_review.md`

Phase 12C adds no live touchpoints, routes, public APIs, persistence, gateway behavior, modelruntime behavior, retrieval/search/embedding behavior, memory writes, controllane mutations, or user-visible output changes.

## Phase 12D Controlled Expansion Design

Phase 12D is `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. It selects exactly one recommended next touchpoint and records the test plan required before implementation.

Recommended future Phase 12E candidate: route envelope metadata.

Route envelope metadata is selected because it can provide route-level diagnostics without approaching prompt text, response bodies, tool payloads, retrieval result content, model output, or memory content. The future allowed surface is method, matched route template or route class, route owner classification, timing summary, workspace/correlation ids when already available, status class after response completion, and no-effect validation.

Deferred touchpoints:

- chat message submission metadata
- existing retrieval-result metadata
- existing gateway trace metadata

Phase 12D adds no code, route observation, public APIs, diagnostics routes, persistent storage, feature flags, adapters, gateway behavior, modelruntime behavior, retrieval/search/embedding behavior, memory writes, controllane mutations, route changes, response changes, or live authority migration. After Phase 12D, `/health` remains the only implemented live touchpoint.

## Phase 12A Non-Goals

- Do not implement Phase 12B in Phase 12A.
- Do not wire FORGE-K into the live daemon.
- Do not observe live requests during Phase 12A.
- Do not add code-level feature flags during Phase 12A.
- Do not modify live APIs or routes.
- Do not call live retrieval, search, embeddings, modelruntime, or tools.
- Do not compile live context through FORGE-K.
- Do not write memory.
- Do not alter user-visible output.
- Do not create a second authority path.
- Do not bypass gateway, permissions, lanes, audit, or controllane.

## Live Authority Preservation

The shadow harness is not an authority path. It must run after, beside, or outside the live owner, never before the live owner and never as a replacement.

Authority preservation rules:

- Gateway remains the only live tool execution authority.
- Controllane remains the only live semantic mutation authority where semantic writes are integrated.
- Permissions and lanes remain the live policy gates.
- Audit remains the live trace authority.
- Modelruntime remains the live model driver governance path.
- Retrieval/search/embeddings remain evidence infrastructure, not truth.
- Memory writes remain live authority until an approved migration phase changes that.
- FORGE-K reports remain diagnostics only.

## Read-Only Shadow Integration Concept

Phase 12B shadow mode means:

1. A live request continues through the existing live path.
2. A disabled-by-default observer may receive stable metadata and refs.
3. Read-only adapters normalize metadata into FORGE-K diagnostic shapes.
4. The shadow harness generates a diagnostic comparison report.
5. The live response is returned exactly as it would have been without shadow mode.
6. Shadow failures are logged or recorded diagnostically and do not fail live requests.

No shadow stage may execute retrieval, call embeddings, call modelruntime, execute tools, write memory, add evidence, admit evidence, commit truth, change response composition, or alter route behavior.

## Live Touchpoint Map

| Touchpoint | Phase 12B candidate? | Allowed data | Forbidden behavior |
|---|---|---|---|
| API route envelope | Yes | route id, method, workspace/correlation ids, request class, timing | route changes, response shape changes |
| Chat request/response metadata | Yes | thread/message ids, model selection refs, response timing, status | prompt mutation, response mutation |
| AI-OS controllane | Limited | syscall ids, object refs, status metadata | syscall execution, semantic mutation |
| Gateway | Yes | invocation refs, approval refs, tool class, status | tool execution or approval decisions |
| Permissions/lanes | Yes | selected profile/lane ids, risk class, decision refs | policy decisions or bypasses |
| Audit | Yes | audit ids, correlation ids, diagnostic refs | authoritative audit mutation unless separately approved |
| Modelruntime | Yes | model id, backend id, request status, token estimates, trace refs | model calls or scheduler changes |
| Retrieval/search/embeddings | Yes | existing run/result/record refs and scores | executing queries or embedding providers |
| Memory | Yes | existing memory note/observation refs | memory writes or truth promotion |
| Backup/release/settings | Deferred | status/config refs | restore, release, config mutation |

## Adapter Design

Adapters must be live-owned wrappers or mirror surfaces outside the FORGE-K simulator packages. They may convert live metadata into diagnostic refs, but they must not import FORGE-K as live authority or allow FORGE-K to call live systems directly.

The Phase 12B adapter set is designed in `docs/architecture/phase_12b_adapter_interfaces.md`.

## Data Flow

The read-only data flow is:

1. Live daemon handles request through existing code.
2. Live owner emits or exposes read-only metadata.
3. A shadow observer copies stable refs and redacted summaries.
4. Adapters normalize records into diagnostic inputs.
5. `shadowharness`-style contracts validate no-effect posture.
6. A report sink stores or discards diagnostics according to the enabled configuration.

Data must prefer refs and summaries over raw content. Secret-looking fields must be rejected or redacted before report storage.

## No-Effect Guarantees

Phase 12B must prove:

- feature flag disabled means no shadow execution
- enabling shadow mode does not change route inventory
- enabling shadow mode does not change public response shapes
- live request status and response body are unchanged
- shadow failures do not fail live requests
- no tools execute from FORGE-K
- no modelruntime call originates from FORGE-K
- no retrieval or embedding call originates from FORGE-K
- no memory write originates from FORGE-K
- no controllane mutation originates from FORGE-K
- no shadow report becomes canonical truth

## Storage / Reporting Strategy

Preferred Phase 12B storage is a diagnostic-only sink with bounded retention. The sink may be:

- in-memory for first implementation
- a local diagnostic table if explicitly approved
- an existing artifact/audit-adjacent diagnostic record if it is clearly marked non-authoritative

Reports must include workspace, correlation, live path, observed refs, warnings, no-effect validation status, and retention metadata. Reports must not store large raw content blobs or secrets.

Phase 12M-Q keeps reports in the same bounded in-memory sink. Advisory reports are attached to the existing diagnostic report object only; no public diagnostics API, persistent store, route, response field, or operator-visible live output is added.

## Feature Flag / Kill Switch Strategy

Implemented flag: `FORGE_K_SHADOW_MODE_ENABLED=false`.

Implemented advisory flag: `FORGE_K_SHADOW_ADVISORY_ENABLED=false`. Advisory reports require global shadow mode and do not force-enable chat or retrieval metadata observers.

Hard defaults:

- disabled unless explicitly enabled
- no shadow reports generated when disabled
- no live behavior changes when enabled
- one restart-free runtime setting may be considered later only if it cannot alter route/API behavior
- an emergency kill switch must stop observation and report generation without affecting the live daemon

Phase 12B adds this flag to core config. Disabling the flag stops report generation. Current reports are in-memory only, so process restart clears them.

## Failure Handling

Shadow errors are diagnostic-only:

- adapter read failure: record warning, skip subreport
- redaction failure: drop report
- no-effect validation failure: drop report and emit local diagnostic warning
- sink failure: drop report
- timeout: abandon shadow work

No failure may alter the live request, live response, route status, tool execution, modelruntime request, retrieval request, memory write, or controllane mutation.

## Rollback Strategy

Rollback for Phase 12B must include:

- disable `FORGE_K_SHADOW_MODE_ENABLED`
- remove or ignore diagnostic sink records
- confirm route inventory unchanged
- confirm live tests pass with flag disabled
- confirm no live authority owner depends on shadow output
- revert the Phase 12B commit if any no-effect guarantee fails

Because the Phase 12B sink is in-memory only and disabled by default, rollback is primarily disabling `FORGE_K_SHADOW_MODE_ENABLED` or reverting the Phase 12B commit if no-effect evidence regresses.

## Test Strategy

The Phase 12B test set must include:

- feature flag disabled tests
- route inventory unchanged tests
- public API response shape unchanged tests
- live response equivalence tests
- no live mutation tests
- no tool execution tests
- no modelruntime call tests
- no retrieval/search/embedding execution tests
- no memory write tests
- shadow report failure isolation tests
- diagnostic redaction tests
- bounded retention tests
- rollback/kill-switch tests

The detailed test checklist is in `docs/testing/phase_12b_shadow_harness_tests.md`.

## Security Review

Security requirements:

- do not capture secrets
- reject secret-looking metadata
- preserve workspace boundaries
- preserve provenance
- preserve permission/lane/gateway authority
- do not expand attack surface with new public routes in Phase 12B unless explicitly approved
- keep diagnostic records non-authoritative

## Performance Review

Phase 12B must be bounded:

- disabled path should be near zero overhead
- enabled shadow work must have strict timeout and size limits
- report generation must be asynchronous or non-blocking unless proven safe
- no retrieval, embedding, modelruntime, or tool calls may be performed by the harness

## Phased Implementation Sequence

1. Phase 12A: design only.
2. Phase 12B: disabled-by-default read-only shadow harness implementation.
3. Phase 12C: shadow diagnostics review and hardening.
4. Phase 12D: controlled shadow expansion design only; select one next candidate and record tests.
5. Phase 12E: route-envelope metadata implementation behind the disabled-by-default shadow flag.
6. Phase 12F: route-envelope diagnostics review and hardening before wider observation.
7. Phase 12G: chat metadata expansion design only; no chat observation implementation in that phase.
8. Phase 12H: disabled-by-default chat metadata shadow implementation with no content capture and no response behavior change.
9. Phase 12I: chat metadata hardening without expanding touchpoints or capture scope.
10. Phase 12J: retrieval metadata expansion design only; no retrieval observation implementation in that phase.
11. Phase 12K-L: disabled-by-default retrieval metadata shadow implementation and hardening; bounded post-run metadata only, no retrieval/search/embedding execution from FORGE-K, no content capture, no route/API behavior change.
12. Later phases: scoped authority migration only with separate approval, tests, and rollback.

## What Not To Do

- Do not wire FORGE-K into the live daemon.
- Do not modify live APIs.
- Do not add routes.
- Do not call live retrieval or embeddings.
- Do not call modelruntime.
- Do not execute tools.
- Do not write memory.
- Do not alter user-visible output.
- Do not create a second authority path.
- Do not bypass gateway, permissions, lanes, audit, or controllane.
