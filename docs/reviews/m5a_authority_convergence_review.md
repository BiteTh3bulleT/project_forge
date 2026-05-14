# M5A Authority Convergence Review

Date: 2026-05-14

Scope: M5A Phase 0 repo reality audit plus Phase 1-5 integration. This review started as a docs lane and was updated after the authority matrix, gateway/modelruntime taxonomy, Control Lane fingerprint seam, HostBridge/FORGE-H cache, read-only System Cockpit summary, and cockpit authority drilldown were integrated.

## Repository State

| Field | Current value |
| --- | --- |
| Branch | `main` |
| HEAD | `e3ec4bcaf28b1ad0db26bf92accc022616b8d7ec` |
| Upstream status | `main...origin/main [behind 34]` |
| Dirty status | Dirty before this pass; many pre-existing desktop, Tauri, Nix, and report/spec changes were present outside this write scope. |
| M5A pack reference HEAD | `cd1b2986a9c9d51eea9af87fd0e70789f651ee4d` from `FORGE_M5A_Authority_Convergence_Pack/00_MASTER_PROMPT_M5A.md` |
| Current authority source | `docs/status/current_authority_sources.md` now exists as the M5A current-truth authority index. |

Pre-existing dirty files included `apps/desktop/*`, `apps/desktop/src-tauri/*`, `nix/*`, `docs/DESKTOP_SHELL.md`, `docs/architecture/desktop_window_manager.md`, `docs/reports/FORGE_PUNCHLIST.md`, and `docs/superpowers/specs/2026-05-13-forge-g7-global-multi-monitor-desktop-hosts.md`. This review intentionally ignores those changes except where read-only evidence was needed.

## Authority Vocabulary

| Label | Meaning in this review |
| --- | --- |
| `LIVE` | Current daemon code path is wired and can affect live runtime behavior. |
| `PARTIAL_LIVE_VALIDATION` | Live path validates or reports deterministically, but does not execute the future authority. |
| `SIMULATOR_ONLY` | Implemented under simulator/research packages and not live daemon authority. |
| `DESIGN_ONLY` | Documented design or planned surface with no current live implementation. |
| `DEFERRED` | Explicitly absent or future work. |
| `NOT_WIRED` | Code or docs exist, but the expected M5A integration surface is not connected. |

## Current Live Authority Map

| Surface | Current owner | Status | Current behavior |
| --- | --- | --- | --- |
| Tool execution, `/api/gateway/invoke` | Gateway | `LIVE` | Gateway remains the API ingress for tool execution. Capability registry maps active/approval-only capabilities to gateway tool ids. |
| Model management, `/forge/models*` | Model Runtime | `LIVE / FEATURE_GATED` | Model import, scan, verify, enable, disable, archive, remove-registration, load, unload, chat, status, usage, queue, loaded, and backend views are routed through modelruntime when enabled. |
| OpenAI-compatible `/v1/models` and `/v1/chat/completions` | Model Runtime | `LIVE / FEATURE_GATED` | Mounted only when OpenAI compatibility is enabled. Streaming emits SSE for streaming-capable modelruntime backends and fails closed with `STREAM_UNSUPPORTED` otherwise. |
| Model backend `Generate` service | Model Runtime | `LIVE_INTERNAL` | `modelruntime.Service.Generate` and backend `Generate` exist and are tested. No direct public `/generate` route was found. Public chat routes call `Chat`, not a separate public generate route. |
| Canonical semantic mutation | AI-OS Control Lane | `LIVE` | Mutating semantic actions go through syscall validation, approval gate, journal/store commit, and audit. |
| Control Lane validation actions | AI-OS Control Lane with shared pure validators | `PARTIAL_LIVE_VALIDATION` | `VALIDATE_KV_IDENTITY`, `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, `VALIDATE_SOURCE_OBJECT_AUTHORITY`, and `VALIDATE_SEMANTIC_OPERATION` are non-mutating validation/diagnostic actions. They do not make FORGE-K live authority. |
| FORGE-K services | `services/core/internal/forgek` | `SIMULATOR_ONLY` | Current status docs and activation summaries keep FORGE-K out of live authority. |
| HostBridge | `services/core/internal/hostbridge` | `LIVE_READ_ONLY` | Reads bounded diagnostics. `GET /forge/system/status` constructs HostBridge with command-backed probes disabled. |
| FORGE-H | `services/core/internal/forgeh` | `LIVE_ADVISORY` | Evaluates HostBridge snapshots and produces advisory resource posture/proposals. System status reports no canonical write committed and no live shell execution authority. |
| System Cockpit/System surface | API and desktop shell | `PARTIAL_LIVE_READ_ONLY / M5A_DRILLDOWN_WIRED` | `GET /forge/system/status` and desktop `/system` expose bounded read-only visibility, authority matrix summary and rows, structured blockers, validation evidence status, modelruntime/gateway alignment, HostBridge cache status, and FORGE-H cache status. |
| Redis/Qdrant | Storage/retrieval future surfaces | `DEFERRED / NOT_TRUTH` | System status marks Redis/Qdrant disabled and not truth authority. |

## FORGE-K Simulator/Live Boundary

The live daemon still uses existing AI-OS, gateway, permissions, audit, modelruntime, retrieval, embeddings, memory, and API paths. FORGE-K remains target architecture and simulator-only authority. Narrow live validation seams share pure deterministic packages, but they do not route live semantic writes through FORGE-K, admit evidence through FORGE-K, execute retrieval, call modelruntime, or mutate memory.

Current Control Lane readiness code also reports `simulatorAuthority=false`, `liveKernelAuthority=false`, and `shadowAuthoritative=false` for validation activation metadata. This is the correct posture for M5A.

## Gateway/Modelruntime Drift

There is still meaningful drift between gateway capability taxonomy and modelruntime authority:

| Capability | Gateway/status-doc posture | Runtime reality | Review finding |
| --- | --- | --- | --- |
| `model.chat` | `docs/architecture/ai_os_tool_surface.md` says partial via runtime API. Gateway registry does not expose a concrete `model.chat` capability row in `defaultToolCapabilities`; model-like calls appear only as `external.call_llm`. | `/forge/models/{id}/chat` and `/v1/chat/completions` are live modelruntime-governed routes when enabled. | `model.chat` should be recorded as modelruntime-owned, not gateway-owned. Gateway should not imply hidden authority unless a shared evaluator is added later. |
| `model.generate` | Docs say partial via runtime API. Gateway registry has no explicit `model.generate` row. | `modelruntime.Service.Generate` and backend `Generate` are implemented/tested internally, but no direct public generate route was found. | Current state is `LIVE_INTERNAL / NO_PUBLIC_ROUTE`. M5A should make this explicit. |
| `model.delete_file` | `docs/architecture/ai_os_tool_surface.md` and `docs/status/model_runtime_status.md` describe approval-governed modelruntime deletion. | `POST /forge/models/{id}/delete-file` runs through modelruntime management governance, requires approval, deletes only the managed model artifact, and preserves the registration as unavailable evidence. | Current docs are aligned: destructive model artifact deletion is live only through approval-governed modelruntime management. Do not confuse remove-registration with file deletion. |
| `filesystem.delete_file` | Gateway registry maps to `fs.delete`, `approval_only`, critical risk. | Gateway file deletion is live through gateway policy, not modelruntime file deletion. | This is a separate filesystem authority and must not be used as an implicit model file delete path. |

## Delete File, Chat, And Generate Current State

`model.delete_file`: `LIVE / APPROVAL_GOVERNED / MODELRUNTIME_OWNED`. Evidence: `/forge/models/{id}/delete-file` calls modelruntime management governance, opens an approval request before execution, deletes only the managed artifact after approval, marks the registration unavailable, and leaves `/forge/models/{id}/remove` as remove-registration only.

`model.chat`: `LIVE / FEATURE_GATED / MODELRUNTIME_OWNED`. Evidence: `/forge/models/{id}/chat` and `/v1/chat/completions` handlers call modelruntime chat paths, validate message shape, reject streaming, attach audit metadata, and return model output as response evidence.

`model.generate`: `LIVE_INTERNAL / MODELRUNTIME_OWNED / NO_DIRECT_PUBLIC_ROUTE_FOUND`. Evidence: `modelruntime.Service.Generate` and backend `Generate` are implemented and tested. Public M5A docs should not imply an operator-facing generate route unless one is added.

## Control Lane Approval Gate Status

Control Lane uses `StaticApprovalGate` by default. Mutating actions with `ApprovalPossible=true` from `future_iris` or `adapter` sources return `ApprovalRequired` and do not commit. Other sources can be allowed by the static gate. Non-mutating validation actions have `ApprovalPossible=false` and are allowed as read-only validations.

M5A now adds a pure deterministic approval fingerprint helper and documentation. Fingerprint determinism is tested, but the seam is not wired into live durable approval enforcement yet; that remains future integration through the existing approvals/audit systems.

## Autonomy Approval Bridge Current Behavior

This pass did not modify autonomy. Existing current-phase docs describe autonomy/rule cells as proposal-only or policy-gated. The relevant M5A concern is unchanged: autonomous or agent-originated semantic writes must still enter Control Lane and may be approval-required depending on source and action. No evidence was found in the inspected files that autonomy can bypass Control Lane for canonical semantic mutation.

## HostBridge And FORGE-H Status

HostBridge is read-only diagnostics. On `GET /forge/system/status`, command-backed probes are disabled by `shellSystemStatusCommandRunner`, so `systemctl` and other commands are not available through the shell status route. HostBridge still reads bounded proc/sys/storage data and records source errors.

FORGE-H consumes a HostBridge snapshot and produces resource policy and proposals. It is advisory-only in system status, reports `canonical_write_committed=false`, and exposes no live shell execution items. M5A now adds read-only TTL cache wrappers for HostBridge snapshots and FORGE-H policy snapshots, and `/forge/system/status` reports cache hit/stale/age/source-error metadata.

## System Cockpit Readiness

The repository has a partial read-only system surface:

- `GET /forge/system/status`
- desktop `/system` page
- bounded HostBridge summary
- FORGE-H posture/proposals
- modelruntime health summary
- storage status
- approval queue wiring note
- FORGE-K activation readiness
- per-row authority matrix drilldown
- structured authority blockers
- bounded validation evidence status

The broader M5A cockpit remains read-only. It now exposes route-by-route authority rows and blocker summaries without raw logs, secrets, approval decisions, host mutation, modelruntime lifecycle controls, or semantic memory writes.

## Deferred, Stub, Fake, And Shadow Surfaces

| Surface | Status | Notes |
| --- | --- | --- |
| FORGE-K runtime/KV/context/lymphatic services | `SIMULATOR_ONLY` | Tested simulator packages; not live daemon authority. |
| FORGE-K shadow reports | `DISABLED_BY_DEFAULT / DIAGNOSTIC_ONLY` | Best-effort bounded metadata only; no response changes or authority. |
| vLLM profile | `PARTIAL / DISABLED_BY_DEFAULT` | Uses OpenAI-compatible backend profile when configured; no host-managed vLLM service. |
| Streaming modelruntime responses | `DEFERRED` | Explicitly rejected by current API handlers. |
| llama.cpp spawn mode | `DEFERRED` | Spawn flag exists but default is disabled/unsupported. |
| Destructive model file deletion | `DEFERRED` | Requires future approval-governed design. |
| System Cockpit full M5A display | `PARTIAL_LIVE_READ_ONLY / M5A_DRILLDOWN_WIRED` | Authority/drift/cache summary, per-row matrix display, structured blockers, and bounded validation evidence status are wired read-only. |
| Micro-agent acceleration | `DESIGN_ONLY` | Design documentation exists. No worker loop, live authority, canonical mutation path, or modelruntime/gateway execution path is wired. |

## Tests Available

Existing tests relevant to M5A include:

- API route inventory and modelruntime route tests under `services/core/internal/api`.
- Modelruntime backend, management, scheduler, orchestration, and audit tests under `services/core/internal/modelruntime`.
- Gateway capability status and invocation tests under `services/core/internal/api` and `services/core/internal/gateway`.
- Control Lane registry, approval, validation, SQLite integration, and activation readiness tests under `services/core/internal/aios/controllane`.
- HostBridge and FORGE-H package tests under `services/core/internal/hostbridge` and `services/core/internal/forgeh`.
- Desktop `SystemPage` tests exist, but this pass did not run desktop tests because the task requested lightweight markdown validation only and the worktree contains unrelated dirty desktop changes.

## Recommended Implementation Order

1. Keep `model.delete_file` behind modelruntime management approval and continue testing that remove-registration is not destructive byte deletion.
2. Integrate Control Lane approval fingerprints with the existing durable approvals/audit path in a later phase.
3. Capture real latency measurements before changing `docs/status/m5a_latency_baseline.md` from `LATENCY_NOT_MEASURED`.
4. Implement micro-agent workers only after cache boundaries and audit/provenance storage targets are settled.

## Validation

Commands run after integration:

- `cd services/core && go test -count=1 ./internal/authoritymatrix ./internal/aios/controllane ./internal/gateway ./internal/hostbridge ./internal/forgeh ./internal/api -run 'TestForgeSystemStatus|TestModelRuntimeCapabilities|TestModelDeleteFileCapability|Fingerprint|Test.*Cache|TestDiagnosticsDoNotMutateHost|TestDeleteFile|TestRemoveRegistration|TestChatAndGenerate|TestGatewayInvoke|TestModelRuntimeRoutes|TestForgeKAuthority|TestRequiredMatrix'` passed.
- `npm -w @forge/desktop run test -- src/pages/SystemPage.test.tsx src/lib/windowManager.test.ts src/stores/workspaceLayoutStore.test.ts` passed.
- `npm -w @forge/desktop run typecheck` passed.
- `npm test` passed.
- `npm run lint` passed.
- `npm run build` passed.
- `npm run smoke` passed.
- `npm -w @forge/desktop run validate` passed.
- `cd apps/desktop/src-tauri && cargo test` passed.
- `git diff --check` passed.

Required M5A commands unavailable in this repo:

- `npm run validate:js` is unavailable because the script is not defined; this is recorded as command unavailability, not failed validation.
- `npm run validate:local` is unavailable because the script is not defined; this is recorded as command unavailability, not failed validation.

See `docs/status/m5a_latency_baseline.md` for latency measurement status; no runtime latency probes were taken beyond validation command pass/fail.
