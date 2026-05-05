# Phase 12B Adapter Interface Design

Status: Phase 12A design artifact only. No adapters are implemented in this phase.

Future adapters must be read-only. They may normalize existing live metadata into diagnostic refs, preserve provenance, and report warnings. They must not mutate live state, execute tools, call modelruntime, execute retrieval/search/embeddings, write memory, compile live context, change routes, or affect user-visible output.

## Common Adapter Contract

Every adapter must define:

- purpose
- input
- output
- allowed operations
- forbidden operations
- provenance requirements
- error behavior
- no-effect tests

Common error behavior:

- return a diagnostic warning
- drop unsafe fields
- fail closed for secret-looking metadata
- never fail the live request

## LiveRequestObservationAdapter

Purpose: observe live request envelope metadata after the live owner has accepted the request.

Input: route id, method, workspace id, correlation id, request class, timing metadata.

Output: normalized shadow observation input.

Allowed operations: copy stable metadata, redact unsafe fields, attach provenance.

Forbidden operations: route selection, response composition, request mutation, body rewriting, auth decisions.

Required tests: route inventory unchanged, response shape unchanged, disabled flag no-op.

## LiveRouteTraceAdapter

Purpose: record route ownership and selected live path for diagnostics.

Input: route name, handler class, live owner component, trace ids.

Output: route trace diagnostic ref.

Allowed operations: classify route, attach live owner metadata.

Forbidden operations: adding routes, removing routes, changing middleware behavior.

Required tests: no route additions, no route behavior change.

## LiveRetrievalTraceAdapter

Purpose: mirror existing retrieval run/result refs.

Input: retrieval run ids, result ids, selection summaries, score metadata already produced by live code.

Output: retrieval diagnostic refs.

Allowed operations: observe existing records, normalize refs, summarize scores.

Forbidden operations: execute retrieval, run search, call embedding providers, select live evidence, compile context.

Required tests: no retrieval/search/embedding method calls from shadow mode.

## LiveContextCompileTraceAdapter

Purpose: mirror existing live context compile metadata.

Input: context packet snapshot refs, restore score refs, selected evidence refs already produced by live code.

Output: context diagnostic refs.

Allowed operations: mirror metadata and compare shape.

Forbidden operations: execute `COMPILE_CONTEXT`, compile live ContextBlocks, alter prompt construction, choose live evidence.

Required tests: no controllane syscall from shadow mode, no prompt mutation.

## LiveGatewayTraceAdapter

Purpose: mirror existing gateway invocation, approval, and artifact refs.

Input: gateway invocation ids, approval ids, tool class, risk class, status.

Output: gateway diagnostic refs.

Allowed operations: observe existing traces.

Forbidden operations: execute tools, request approvals, approve/deny tools, mutate artifacts.

Required tests: no gateway invocation creation from shadow mode.

## LiveAuditTraceAdapter

Purpose: mirror existing audit correlation and trace refs.

Input: audit ids, correlation ids, event kinds, outcome metadata.

Output: audit diagnostic refs.

Allowed operations: observe existing audit records.

Forbidden operations: authoritative audit writes unless a diagnostic sink is explicitly approved.

Required tests: no authoritative audit mutation from shadow mode.

## LiveModelRuntimeTraceAdapter

Purpose: mirror existing modelruntime trace metadata.

Input: model id, backend id, request id, runtime status, token estimate, timing metadata.

Output: runtime diagnostic refs.

Allowed operations: observe already-produced runtime metadata.

Forbidden operations: call modelruntime, load/unload models, schedule requests, alter model selection.

Required tests: no modelruntime request creation from shadow mode.

## LiveMemoryEvidenceTraceAdapter

Purpose: mirror existing memory note, memory observation, and evidence refs.

Input: memory note ids, observation ids, provenance refs, stale/usefulness metadata.

Output: memory evidence diagnostic refs.

Allowed operations: read stable refs and summaries if already available to the live request.

Forbidden operations: write memory, mark usefulness, repair observations, admit evidence, promote retrieved content to truth.

Required tests: no memory table writes from shadow mode.

## ShadowReportSink

Purpose: store or discard diagnostic reports.

Input: validated shadow reports.

Output: sink status and diagnostic report ref if persisted.

Allowed operations: in-memory retention or explicitly approved diagnostic persistence.

Forbidden operations: canonical memory writes, controllane commits, route responses, user-visible output.

Required tests: sink failure does not fail live request; retention bounds enforced; reports are non-authoritative.

## What Not To Do

- Do not implement these adapters in Phase 12A.
- Do not wire them to live systems without Phase 12B approval.
- Do not allow adapters to execute live behavior.
- Do not let adapter output become canonical truth.
