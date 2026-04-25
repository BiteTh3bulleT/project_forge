# Phase 5.99 Baseline Completion Report

Date: 2026-04-21
Scope: pass-1 truth audit + pass-2 minimal hardening

## 1) Git status snapshot before this phase pass

### Branch
- Current branch: `main`
- Upstream tracking: `origin/main`
- Visible local branches: `main`

### Modified files before this work
```text
 M .gitignore
 M AGENTS.md
 M CLAUDE.md
 M README.md
 M apps/desktop/src-tauri/Cargo.lock
 M apps/desktop/src-tauri/Cargo.toml
 M apps/desktop/src-tauri/src/main.rs
 M apps/desktop/src/App.tsx
 M apps/desktop/src/index.css
 M apps/desktop/src/layout/AppShell.tsx
 M apps/desktop/src/lib/api.ts
 M apps/desktop/src/lib/desktop.ts
 M apps/desktop/src/pages/DossiersPage.tsx
 M apps/desktop/src/pages/MemoryPage.tsx
 M apps/desktop/src/pages/RetrievalRunsPage.tsx
 M apps/desktop/src/pages/SettingsPage.tsx
 M apps/desktop/src/pages/WorkspaceLayoutsPage.tsx
 M apps/desktop/src/stores/uiStore.ts
 M apps/desktop/src/stores/workspaceLayoutStore.ts
 M docs/MEMORY_ARCHITECTURE.md
 M docs/RETRIEVAL_AND_EMBEDDINGS.md
 M docs/RETRIEVAL_PIPELINE.md
 M docs/USEFULNESS_AND_REPAIR.md
 A docs/architecture/nix_substrate.md
 M docs/data_model/persistence_inventory.md
 M docs/roadmap/forge_ai_os_phases.md
 A flake.lock
 A flake.nix
 A nix/checks/go-tests.nix
 A nix/checks/go-vet.nix
 A nix/checks/js-build.nix
 A nix/lib/default.nix
 A nix/modules/README.md
 A nix/overlays/default.nix
 A nix/packages/forge-core.nix
 A nix/profiles/README.md
 A nix/shells/aios.nix
 A nix/shells/core.nix
 A nix/shells/default.nix
 A nix/shells/desktop.nix
 A nix/tool-capsules/README.md
 M package.json
 M packages/shared/src/index.ts
 M services/core/internal/api/chat_assistant_gateway.go
 M services/core/internal/api/phase3.go
 M services/core/internal/api/phase_memory.go
 M services/core/internal/api/server.go
 M services/core/internal/backup/service.go
 M services/core/internal/config/config.go
 M services/core/internal/gateway/chat_wire.go
 M services/core/internal/gateway/forced_chat.go
 M services/core/internal/gateway/forced_chat_test.go
 M services/core/internal/gateway/service.go
 M services/core/internal/jobs/service.go
 M services/core/internal/memory/dossiers.go
 M services/core/internal/memory/helpers.go
 M services/core/internal/memory/repair.go
 M services/core/internal/memory/retrieval.go
 M services/core/internal/memory/service.go
 M services/core/internal/memory/types.go
 M services/core/internal/permissions/service.go
 M services/core/internal/permissions/service_test.go
 M services/core/internal/retrieval/service.go
 M services/core/internal/store/migrate.go
 M services/core/internal/gateway/tool_capability_registry.go
 M services/core/internal/gateway/tool_surface_test.go
 M services/core/internal/aios/autonomy/rule_agents.go
 M services/core/internal/aios/autonomy/runner.go
 M services/core/internal/aios/autonomy/sqlite_repositories.go
 M services/core/internal/aios/autonomy/sqlite_repositories_test.go
 M services/core/internal/aios/autonomy/safety_guards_test.go
 M services/core/internal/api/autonomy_maintenance_loop.go
 M services/core/internal/api/chat_assistant_gateway.go
 M services/core/internal/api/phase_memory_vsa_test.go
```

### Untracked files before this work
```text
 ?? .claude/
 ?? docs/architecture/forge_wiring_map.md
 ?? docs/architecture/traceability.md
 ?? docs/architecture/v1_v2_unification_plan.md
 ?? docs/status/
 ?? docs/runbooks/
 ?? scripts/check-desktop-deps.sh
 ?? scripts/desktop-clean-port.sh
 ?? scripts/forge-smoke.sh
 ?? services/core/internal/aios/autonomy/safety_guards_test.go
 ?? services/core/internal/aios/autonomy/sqlite_repositories.go
 ?? services/core/internal/aios/autonomy/sqlite_repositories_test.go
 ?? services/core/internal/api/chat_assistant_gateway_normalize_test.go
 ?? services/core/internal/api/phase_memory_vsa_test.go
 ?? services/core/internal/api/server_adapters_test.go
 ?? services/core/internal/backup/service_test.go
 ?? services/core/internal/config/config_test.go
 ?? services/core/internal/gateway/service_desktop_open_test.go
 ?? services/core/internal/jobs/service_gateway_input_test.go
 ?? services/core/internal/memory/vsa_engine.go
 ?? services/core/internal/memory/vsa_engine_test.go
 ?? services/core/internal/memory/vsa_indexer.go
 ?? services/core/internal/memory/vsa_indexer_test.go
 ?? services/core/internal/memory/vsa_signals.go
 ?? services/core/internal/retrieval/service_vsa_test.go
 ?? services/core/internal/store/migrate_vsa_test.go
```

## 2) Current phase status

| Phase | Status | Notes |
|---|---|---|
| Phase 1 | mostly complete | AI-OS framing and runtime foundation exist; v1 path migration still guarded. |
| Phase 2 | mostly complete | Deterministic syscall control lane is real and tested. |
| Phase 3 | partial | Cognitive persistence is real; legacy mutation routes still have explicit boundaries. |
| Phase 4 | mostly complete | Ingest/cell pipeline is present and committed through kernel paths. |
| Phase 5 | partial | Truth engine is real; repair/rebuild remains partial. |
| Phase 5.5 | partial | Rule-agent runtime exists; proposal/cleanup risks are still guarded-by-policy. |
| Phase 5.75 | partial | Autonomy policy/runner are durable and policy-gated; persistence parity is incomplete beyond autonomy settings. |
| Phase 5.9 | partial | Tool taxonomy/registry/policy are real; high-risk capabilities are gated/approval-only or stubbed. |
| Phase 5.95 | partial | runtime authority wiring improved; legacy adapter side door removed and gateway remains sole tool-execution ingress. |
| Phase 5.99 | partial | Final convergence run underway to close drift and confirm all required hardening/tests. |
| Phase N1 | partial | Light Nix scaffolding exists; forge-core package still has clean-build parity gaps. |

## 3) Current authoritative runtime paths

| Area | Current authoritative path | Status |
|---|---|---|
| Gateway/tool execution | `/api/gateway/invoke` -> `gateway.Execute` | authoritative |
| Semantic mutation/syscalls | `aios/controllane.Processor` | authoritative |
| Memory/cognitive filesystem | controllane-backed SQLite repositories | authoritative with legacy exception for direct observation routes |
| State/current truth | `aios/truth.Engine` over controllane repos | authoritative |
| Approvals | `approvals.Service` via gateway needs-approval | authoritative |
| Audit | `audit.Service` + gateway/kernel sinks | authoritative |
| Artifacts | `artifacts.Service` + gateway artifact linkages | authoritative |
| Autonomy | intent/policy/budget/runner path with SQLite-backed repos and policy guards | authoritative with bounded/autonomy-mode defaults |
| Rule agents | rule runtime proposes actions; commit via runner/syscall path | authoritative for supported proposal flow |
| Desktop/API interaction | desktop HTTP client to backend `/api/*` handlers | authoritative; mutation routes validated in backend |

## 4) Top current risks

1. Backup/export restore parity is still incomplete outside autonomy settings path.
2. Legacy non-syscall mutation routes remain as documented boundaries and must not gain implicit production use.
3. Rule-agent coverage is partial relative to the expected full expected set.
4. Dedicated JS/TS unit/lint scripts remain absent; root `test`/`lint` delegate to Go core and root `typecheck` covers desktop TypeScript.
5. CI workflow scaffolding exists in `.github/workflows/ci.yml`, but remote GitHub runner execution was not inspected in this local pass.
6. `nix --extra-experimental-features 'nix-command flakes' flake check` fails in this environment due `nix-daemon` socket unavailability.
7. `/api/adapters/{id}/invoke` route has been removed; gateway remains the sole tool execution path.
