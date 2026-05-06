# Phase 11G Shadow Mode Harness Plan

Status: simulator-only plan retained as historical Phase 11G handoff. Phase 12B implemented the first read-only `/health` metadata observer, and Phase 12C hardened it without adding touchpoints.

Phase 11G does not observe live daemon requests, wire live adapters, or authorize live authority migration.

## Proposed Phase 12B Plan

Phase 12B may implement a read-only shadow harness only after Phase 12A live integration design is accepted. The harness would observe selected live request metadata, normalize stable refs through read-only adapters, and emit diagnostic comparison reports that do not affect live behavior.

The first implementation slice should be disabled by default, workspace-scoped, and limited to diagnostic storage/reporting. It should not change routes, responses, modelruntime behavior, retrieval behavior, embeddings, memory writes, gateway execution, permissions, lanes, audit, or controllane behavior.

## Live Touchpoints To Observe Later

| Touchpoint | Observation only | Required adapter |
|---|---|---|
| API request metadata | request refs, workspace/correlation ids, route names | ReadOnlyLiveStateAdapter |
| Gateway invocation traces | lane, capability, approval, invocation refs | LiveGatewayTraceAdapter |
| Audit records | provenance and correlation refs | LiveAuditMirrorAdapter |
| Control lane metadata | syscall refs and journal refs where already produced | ReadOnlyLiveStateAdapter |
| Retrieval/search/embedding metadata | existing retrieval/search/embedding/VSA refs | ReadOnlyRAGAdapter, LiveRetrievalMirrorAdapter, LiveSearchTraceAdapter, LiveEmbeddingTraceAdapter |
| Context compile metadata | existing compile refs and block/hash metadata | LiveContextCompileMirrorAdapter |
| Modelruntime traces | existing runtime result/model identity refs | LiveModelRuntimeTraceAdapter |
| Dream/autonomy diagnostics | existing non-canonical reports/proposals | ReadOnlyLiveStateAdapter |

## Event And Report Flow

1. Live request executes normally.
2. Shadow observer receives only stable metadata and refs.
3. Read-only adapters normalize refs and provenance.
4. Harness creates `ShadowObservation`.
5. Harness creates subreports for consensus, context, RAG, runtime, KV, and lymphatic diagnostics when refs exist.
6. Harness validates no-effect guarantees.
7. Harness stores or emits `ShadowComparisonReport` as diagnostics only.
8. Live response remains unchanged.

## Required Tests Before Live Observation

- route inventory tests
- no live mutation tests
- no public API change tests
- no user-visible output change tests
- no tool execution from shadow tests
- no modelruntime call tests
- no retrieval execution tests
- no embedding provider call tests
- no memory write tests
- forbidden live import tests for simulator contracts
- diagnostic report determinism tests
- rollback/disable tests

## Rollback / Disable Strategy

- Shadow harness disabled by default.
- Single config flag or launch profile controls enablement.
- Workspace scope must be explicit.
- Failure to validate no-effect policy disables reporting for that request.
- Any adapter hard stop disables the harness and preserves live behavior.
- Rollback means turning off observation/reporting only; live daemon behavior should be identical before and after rollback.

## Risk Register

| Risk | Mitigation |
|---|---|
| Shadow diagnostics mistaken for truth | Label reports diagnostic only; no commit paths. |
| Adapter mutates live state | Read-only contracts, tests, hard-stop validation. |
| RAG becomes live response input | ReadOnlyRAGAdapter forbids context compilation and user-visible output. |
| Modelruntime called from shadow | Policy and tests forbid runtime calls. |
| Retrieval or embeddings executed from shadow | Policy and tests forbid retrieval/embedding execution. |
| Gateway authority duplicated | Gateway remains live tool execution authority until explicit migration. |
| Public behavior changes | No-user-visible-output tests and route inventory checks. |
| Secrets copied into diagnostics | Metadata secret-key rejection and ref-only observation. |

## No-Mutation Guarantees

The future harness must prove:

- no live mutation
- no tool execution
- no modelruntime calls
- no retrieval execution
- no embedding calls
- no memory writes
- no public API changes
- no user-visible output
- no second authority path

## Open Questions

- Which live routes are safe first candidates for observation?
- Where should diagnostic reports be stored without becoming canonical truth?
- What operator UI should display shadow divergence?
- What retention policy applies to shadow reports?
- Which route inventory test should be treated as the Phase 12B gate?
- How should shadow diagnostics be correlated with existing audit records without writing audit from FORGE-K?
