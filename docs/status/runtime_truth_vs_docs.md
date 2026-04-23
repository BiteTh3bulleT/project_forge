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
| `README.md` | mostly accurate | Desktop/runtime claims are largely true; tool-surface breadth can be read as more executable than it is. |
| `FORGE_CONTEXT.md` | outdated | Still framed as latest Phase 5 pass; repo now has Phase 5.99 status and hardening docs. |
| `docs/architecture/forge_ai_os.md` | mostly accurate | Explicitly calls out partial/distributed areas; matches code-level maturity reasonably well. |
| `docs/MEMORY_ARCHITECTURE.md` | partially accurate | Architecture is real in this working tree, but key VSA files are untracked and make fresh-clone reality weaker. |
| `docs/architecture/tool_gateway.md` | mostly accurate | Gateway is authoritative and legacy adapter invoke route is now removed. |
| `docs/architecture/ai_os_tool_surface.md` | mostly accurate | Full taxonomy/status model matches code; execution breadth is still heavily stubbed/deferred by design. |
| `docs/architecture/rule_based_agents.md` | partially accurate | Propose-only runtime is real; commit-flow wording overstates current rule-runtime behavior. |
| `docs/architecture/autonomy_layer.md` | mostly accurate | Durable SQLite-backed repos and policy gating are implemented; parity gaps are correctly acknowledged. |
| `docs/roadmap/forge_ai_os_phases.md` | mostly accurate | Reality-based partial statuses are consistent with current code/status docs. |
| Current status docs set | partially accurate (mixed) | Many are strong; some have drift/inconsistency (notably smoke-test status in matrix vs dedicated smoke doc/script). |

## 2) Mismatch and drift register (doc claim vs implementation)

| Doc claim (file/section) | Classification | Implementation evidence | Mismatch explanation | Recommendation |
|---|---|---|---|---|
| `README.md` “governed full tool layer …” | partially accurate | `services/core/internal/gateway/tool_capability_registry.go` (`defaultCapabilityDescriptor` defaults to `stubbed`; only subset in `activeMappings`) | Governance is real, but most taxonomy entries are still non-executable (`stubbed`/`deferred`/`approval_only`). | Fix docs: add one sentence that “full taxonomy is registered; only a subset is active.” |
| `README.md` run/build guidance implies normal fresh-clone path | partially accurate | `git ls-files` shows `services/core/internal/memory/vsa_engine.go`, `vsa_indexer.go`, `vsa_signals.go` are untracked; tracked memory/retrieval code references VSA symbols | Current working tree can run, but reproducibility for fresh clones is not guaranteed while critical VSA files are untracked. | Implement missing part: commit VSA files (or remove references); until then add explicit bring-up warning in README. |
| `FORGE_CONTEXT.md` “latest hardening pass is Phase 5” | outdated | `docs/roadmap/forge_ai_os_phases.md` and multiple `docs/status/*` files are updated through Phase 5.99 | Context doc lags current convergence/hardening stage. | Fix docs: update phase narrative/changelog to include 5.5/5.75/5.9/5.95/5.99 realities. |
| `docs/MEMORY_ARCHITECTURE.md` VSA inspectability described as implemented runtime | partially accurate | VSA implementation files exist in working tree (`services/core/internal/memory/vsa_*.go`), but are currently untracked; VSA paths used by `internal/memory` and `internal/retrieval` | True for this checkout, fragile for repo truth and new environments. | Relabel scaffold-deferred risk in memory doc until VSA sources are tracked in repo history. |
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
- `docs/architecture/forge_ai_os.md` calls out partial/distributed cognitive filesystem and lane-boundary gaps; this aligns with current mixed v1/v2 state.
- `docs/architecture/ai_os_tool_surface.md` capability taxonomy, status vocabulary, and risk framing match domain/gateway types:
  - `services/core/internal/aios/domain/tool_surface.go`
  - `services/core/internal/gateway/tool_capability_registry.go`
  - `services/core/internal/gateway/tool_policy.go`
- `docs/roadmap/forge_ai_os_phases.md` “partial/mostly complete” posture aligns with current baseline and gate docs.

## 4) Prioritized doc corrections (minimal, high-value)

1. Update `FORGE_CONTEXT.md` phase/changelog to Phase 5.99 reality.
2. Keep `docs/architecture/tool_gateway.md` and `docs/architecture/ai_os_tool_surface.md` aligned with gateway-only ingress.
3. Update `docs/architecture/rule_based_agents.md` to state current propose-only runtime behavior as authoritative.
4. Update `README.md` with explicit “registered != executable” and fresh-clone VSA caveat until tracked.
5. Update `docs/status/implementation_matrix.md` smoke-test row to match `scripts/forge-smoke.sh` + `smoke_test_status.md`.
