# FORGE Implementation Matrix (Current Live Authority Snapshot)

Observed against this branch on 2026-08-14.

Top note: this matrix tracks legacy/live AI-OS implementation status and current daemon authority paths. It is not the FORGE-K simulator phase matrix. For current phase status and FORGE-K simulator/live authority boundaries, see `docs/reviews/current_phase_status.md`, `docs/status/current_authority_sources.md`, and ADR 0005 (`docs/adr/0005-forge-k-simulator-vs-live-authority.md`).

Status values: `real`, `partial`, `legacy-boundary`, `blocked`, `scaffold`, `deferred`.

| Subsystem | Current authoritative path | Status | Evidence | Main remaining gap |
|---|---|---|---|---|
| Tool execution | `/api/gateway/invoke` -> `gateway.Execute` | real | `internal/gateway/service.go`, gateway tests | keep gateway-only ingress discipline |
| Legacy adapter side door | route removed | resolved | `server.go`, `server_adapters_test.go` | `/api/adapters/{id}/invoke` no longer wired; removed-route behavior tested as `404` |
| Model runtime governance (M3/M4) | `/forge/models*` + `modelruntime.Service` (with gated `/v1/*` compatibility) | partial (implemented) | `services/core/internal/modelruntime/*`, `services/core/internal/api/model_runtime.go`, `services/core/internal/api/model_runtime_bridge.go`, `services/core/internal/config/config.go`, gateway `model.*` policy aliases | Remaining work is hardening/supervision: stronger backend/process control, deeper scheduling/backpressure, cancellation/usage accounting hardening, and operator visibility |
| Semantic mutation Kernel | `forgekernel.Kernel` (default) -> K-owned `DurablePort` -> Control Lane SQLite implementation | partial cutover (K20B) | `internal/forgekernel`, controllane/truth tests, durable stage-order/gate/rollback tests, kernel status tests | K20C+ must migrate subsystem authority and later retire the adapter implementation and v1 rollback |
| Retired memory observation mutation | syscall-native memory/state mutation only | resolved | retired gate in `server.go`, `server_memory_legacy_test.go` | keep mutation endpoints returning `410 Gone`; add only syscall-native write facades |
| Approvals/events/jobs/artifacts restore parity | `backup.Service` restore mappings | mostly complete | `backup/service.go`, `backup/service_test.go` | VSA-derived export-only sections remain |
| Cognitive filesystem restore parity | restore mappings for core cognitive tables | mostly complete | `backup/service.go` mappings + tests | VSA tables still export-only |
| Project context/evaluation/audit restore parity | restore mappings in backup service | mostly complete | `backup/service.go`, `backup/service_test.go` | VSA-derived export-only sections remain |
| Restore failure safety | transactional restore for supported DB sections | mostly complete | `backup/service.go`, rollback test coverage | cross-system rollback outside DB scope remains limited; `atomicScope`/warnings are explicit |
| Autonomy persistence safety | SQLite-backed autonomy repos + persistence gate | mostly complete | autonomy policy/runner tests | trace/audit visibility still partial |
| Rule-agent layer | propose-only runtime with 2 agents | partial (explicitly narrow) | `rule_agents.go`, safety guard tests, `docs/architecture/rule_based_agents.md` | broader deterministic agents deferred until signal, policy, test, and trace coverage exists |
| Desktop/backend mutation boundary | desktop -> backend `/api/*` | real | `apps/desktop/src/lib/api.ts`, API server wiring, Audit page trace lookup rendering, chat/gateway/job/approval/artifact/workbench/events/lineage Audit pivots | trace/explain UI has a read-only Audit page authority-chain summary with direct pivots from existing operator surfaces when trace, correlation, or job context is available; artifact/provenance/journal object-id lookup remains future backend work |
| Remote gateways | `/api/remote/telegram`, `/api/remote/discord`, `/api/telegram/status`, `/api/discord/status`, optional Discord bot gateway | partial | `services/core/internal/api/remote.go`, `telegram_gateway_service.go`, `discord_gateway_service.go`, `discord_gateway_server.go`, `docs/REMOTE_ACCESS.md`, `docs/architecture/discord_integration_layer.md` | Telegram/Discord ingress and status are real when configured; Discord command coverage remains intentionally narrow and unimplemented commands return `NOT_IMPLEMENTED`; no broader Discord workflow authority without separate tests/docs |
| JS/TS validation surface | root `test:js`/`lint:js`/`validate:js` + desktop Vitest/typecheck/build | partial (improved) | `package.json`, `apps/desktop/package.json` | lint is TypeScript-only; no ESLint lane or non-desktop package tests yet |
| Nix foundation and host envelope | flake/check definitions present; opt-in NixOS modules/operator VM present | partial; blocked in this env for authoritative Nix execution | `nix/nixos/modules/*`, `nix/nixos/configurations/forge-operator-vm.nix`, `docs/status/nix_foundation_status.md`, `docs/runbooks/forge_operator_desktop_vm.md`, command outputs in `test_gap_analysis.md` | run Nix checks on a Nix-enabled host; keep host envelope opt-in and non-mutating |
| Fresh-clone boot integrity | core build path + VSA preflight scripts | real (guarded) | `scripts/check-vsa-files.mjs`, `scripts/forge-core.mjs`, `scripts/forge-smoke.mjs`, platform smoke scripts, tracked `services/core/internal/memory/vsa_*.go` | VSA status is authoritative source; maintain tracked files and strict preflight guard |

## Convergence highlights in this pass

1. Backup restore parity materially improved with restore support for project context/evaluations/audit/gateway sections and atomic rollback behavior.
2. Legacy memory mutation boundaries are retired (`410 Gone`) with correlation/trace/workspace audit payloads, and legacy adapter ingress is removed.
3. Root core build/test/vet/core/smoke flows include strict VSA tracked-source preflight checks, and required VSA source files are tracked in authoritative branch state.
4. Model runtime M3/M4 is now real inside the governed modelruntime boundary: manifest/store/registry/backend/runtime service/APIs/config/audit paths are implemented with import/register flows, persistent lifecycle state, deterministic selection, backend expansion, runtime inspection endpoints, approval-required managed delete-file flow, disabled-by-default vLLM-compatible external endpoint support, policy-visible gateway `model.*` aliases, and governed chat/SSE streaming. Remaining work is hardening/supervision, not a FORGE-K live authority change.
