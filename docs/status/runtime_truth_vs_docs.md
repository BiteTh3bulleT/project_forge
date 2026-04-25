# Runtime Truth vs Docs Audit

Date: 2026-04-21  
Scope: documentation claims vs current implementation reality (repo working tree)

Legend for claim ratings:
- `accurate`
- `mostly accurate`
- `partially accurate`
- `overstated`
- `outdated`
- `future-facing but clearly marked`
- `misleading`

## 1) Required-doc classification summary

| Document | Overall classification | Notes |
|---|---|---|
| `README.md` | mostly accurate | Desktop/runtime claims are largely true; tool-surface taxonomy now resolves through gateway-backed execution paths, with high-risk entries approval-gated. |
| `FORGE_CONTEXT.md` | outdated | Still framed as latest Phase 5 pass; repo now has Phase 5.99 status and hardening docs. |
| `docs/architecture/forge_ai_os.md` | mostly accurate | Explicitly calls out partial/distributed areas; matches code-level maturity reasonably well. |
| `docs/MEMORY_ARCHITECTURE.md` | mostly accurate | VSA implementation files are tracked in this branch state and strict preflight passes; remaining caveat is test depth, not source availability. |
| `docs/architecture/tool_gateway.md` | mostly accurate | Gateway is authoritative and legacy adapter invoke route is now removed. |
| `docs/architecture/ai_os_tool_surface.md` | mostly accurate | Full taxonomy/status model matches code; default production capability rows now resolve to gateway-backed `active` or `approval_only` tools. |
| `docs/architecture/rule_based_agents.md` | partially accurate | Propose-only runtime is real; commit-flow wording overstates current rule-runtime behavior. |
| `docs/architecture/autonomy_layer.md` | mostly accurate | Durable SQLite-backed repos and policy gating are implemented; parity gaps are correctly acknowledged. |
| `docs/roadmap/forge_ai_os_phases.md` | mostly accurate | Reality-based partial statuses are consistent with current code/status docs. |
| Current status docs set | mostly accurate | Status docs now align on green smoke/root validation and environment-blocked Nix checks. |

## 2) Mismatch and drift register (doc claim vs implementation)

| Doc claim (file/section) | Classification | Implementation evidence | Mismatch explanation | Recommendation |
|---|---|---|---|---|
| `README.md` “governed full tool layer …” | mostly accurate | `services/core/internal/gateway/tool_capability_registry.go` now assigns every default capability a `gatewayToolId` and final status of `active` or `approval_only`; `services/core/internal/gateway/capability_backing_tool.go` provides the generic concrete backing path. | Governance and registry coverage are real. Some platform, credential, binary, or privilege gaps intentionally surface as runtime misconfiguration errors rather than stubs. | Keep docs precise: registered capabilities are backed by gateway tools, and dangerous/external/privileged actions remain approval-gated. |
| `README.md` run/build guidance implies normal fresh-clone path | mostly accurate | `git ls-files` shows `services/core/internal/memory/vsa_engine.go`, `vsa_indexer.go`, and `vsa_signals.go` are tracked; strict VSA preflight passes. | Current working tree and tracked source state can run through root validation. Nix remains environment-blocked. | Keep strict VSA preflight and publish Nix results from a Nix-enabled host. |
| `FORGE_CONTEXT.md` “latest hardening pass is Phase 5” | outdated | `docs/roadmap/forge_ai_os_phases.md` and multiple `docs/status/*` files are updated through Phase 5.99 | Context doc lags current convergence/hardening stage. | Fix docs: update phase narrative/changelog to include 5.5/5.75/5.9/5.95/5.99 realities. |
| `docs/MEMORY_ARCHITECTURE.md` VSA inspectability described as implemented runtime | mostly accurate | VSA implementation files are tracked and used by `internal/memory` and `internal/retrieval`; strict tracked-state preflight passes. | Runtime exists; remaining risk is coverage breadth and ongoing source-state enforcement. | Keep VSA files tracked and broaden VSA behavior tests when changing retrieval semantics. |
| `docs/architecture/tool_gateway.md` “only authorized tool execution boundary” and “no direct ops outside gateway” | accurate | `services/core/internal/api/server.go` no longer wires `/api/adapters/{id}/invoke` | Gateway is now sole API ingress for tool execution. | Keep docs aligned; do not reintroduce alternate route. |
| `docs/architecture/ai_os_tool_surface.md` “no bypass” | mostly accurate | Gateway/policy enforcement and status model exist (`internal/gateway/tool_policy.go`, `tool_capability_registry.go`); no direct adapter invoke route remains in API | Current runtime matches no-bypass execution ingress for adapters/tools. | Keep docs concise on gateway-only authority. |
| `docs/architecture/rule_based_agents.md` flow step says authorized actions commit through syscall processor | overstated | `services/core/internal/aios/autonomy/rule_agents.go` uses `runner.Run(..., domain.RunModeProposeOnly, ...)` in `RunOnce` | Current rule-agent runtime is explicitly propose-only; it does not auto-commit in this path. | Fix docs: relabel commit step as deferred/alternate path; keep propose-only as current runtime truth. |
| `docs/architecture/autonomy_layer.md` durable persistence and safety posture | mostly accurate | Durable repos in `services/core/internal/aios/autonomy/sqlite_repositories.go`; safety gate in `policy_evaluator.go` (`hasDurableSelfCommitBacking`) | Core claim is true; doc already acknowledges restore-parity gaps. | Keep docs; continue linking to restore-parity status doc for operational boundaries. |
| `docs/status/implementation_matrix.md` validation row marks smoke tests as `unknown` | outdated | `scripts/forge-smoke.sh` exists; `package.json` exposes `npm run smoke`; `docs/status/smoke_test_status.md` records green run | Matrix currently understates available smoke coverage relative to other status docs and scripts. | Fix docs: update matrix smoke row to `partial` with explicit endpoint scope limits. |
| `docs/status/nix_foundation_status.md` Nix readiness messaging | mostly accurate | `flake.nix`/`flake.lock` present; recorded daemon/socket failures in validation docs | Nix status is conservative and correctly non-authoritative. | Keep docs; no change except periodic command re-validation when daemon available. |

## 3) Claims verified as accurate or mostly accurate (no major drift)

- `README.md` desktop monitor-aware layout behavior is grounded in real desktop code paths:
  - `apps/desktop/src/pages/WorkspaceLayoutsPage.tsx`
  - `apps/desktop/src/stores/workspaceLayoutStore.ts`
  - `apps/desktop/src-tauri/capabilities/default.json`
- `docs/architecture/forge_ai_os.md` calls out partial/distributed cognitive filesystem and lane-boundary gaps; this aligns with the current cutover state.
- `docs/architecture/ai_os_tool_surface.md` capability taxonomy, status vocabulary, and risk framing match domain/gateway types. Default production rows resolve to `active` or `approval_only`; explicit `deferred`/`stubbed` states are still supported for override/testing semantics:
  - `services/core/internal/aios/domain/tool_surface.go`
  - `services/core/internal/gateway/tool_capability_registry.go`
  - `services/core/internal/gateway/tool_policy.go`
- `docs/roadmap/forge_ai_os_phases.md` “partial/mostly complete” posture aligns with current baseline and gate docs.

## 4) Prioritized doc corrections (minimal, high-value)

1. Update `FORGE_CONTEXT.md` phase/changelog to Phase 5.99 reality.
2. Keep `docs/architecture/tool_gateway.md` and `docs/architecture/ai_os_tool_surface.md` aligned with gateway-only ingress.
3. Update `docs/architecture/rule_based_agents.md` to state current propose-only runtime behavior as authoritative.
4. Keep VSA source-state checks in root core/smoke/test/vet commands.
5. Keep smoke-test evidence aligned with `scripts/forge-smoke.sh` + `smoke_test_status.md`.
