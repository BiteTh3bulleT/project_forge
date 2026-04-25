# FORGE Runtime Cutover Blockers

Date: 2026-04-21  
Purpose: identify what still prevents FORGE from operating as one authoritative machine.

Milestone keys used in `Blocks`:

- `P6` = Phase 6 context compiler readiness
- `NIX` = deep Nix/NixOS integration readiness
- `AUTO` = more autonomy/tool freedom readiness
- `IRIS` = IRIS integration readiness
- `DEMO` = external/demo readiness

---

## 1) Authoritative path blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Legacy adapter invoke endpoint still exists | Tool execution boundary | resolved-note | Route has been removed; gateway is sole executable path | `services/core/internal/api/server.go`, `apps/desktop/src/lib/api.ts` | both | Keep gateway-only path and prevent route reintroduction | no |
| Memory observation mutation bypass | Memory/state authority | resolved | `/api/memory/observations` mutation endpoints now return `410 Gone` and audit `*.retired`; mutation authority is syscall-native | `services/core/internal/api/server.go`, `services/core/internal/api/server_memory_legacy_test.go` | code | Keep retired routes non-executable; add only syscall-native write facades | no |
| Dual event streams remain active | Truth/event authority | high | `events` and `journal_events` both receive runtime events; no single replay authority is enforced | `services/core/internal/events/logger.go`, `services/core/internal/aios/controllane/processor.go`, `services/core/internal/store/migrate.go` | both | Declare canonical stream, bridge/retire secondary stream, and update API/UI to one source | `P6`, `IRIS`, `DEMO` |
| No first-class external API for semantic syscall/truth operations | Truth/syscall external surface | medium | Kernel/truth is real internally but external read/write facades remain limited | `services/core/internal/api/server.go`, `services/core/internal/aios/*` | both | Add explicit API facade for syscall/truth reads and writes | `P6`, `IRIS` |

## 2) Durability blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Backup export/restore parity is incomplete | Backup/import durability | critical | `full_backup` exports many AI-OS tables but restore maps only a small subset; recovery claims overstate reality | `services/core/internal/backup/service.go` (`pickSections`, `insertStatements`) | both | Implement restore mappings for exported critical sections or shrink export claims to restorable subset | `P6`, `AUTO`, `IRIS`, `DEMO` |
| Capability status overrides are not durable | Tool governance durability | resolved | Runtime capability status changes persist through `SQLiteOverrideStore` and load at server startup | `services/core/internal/gateway/tool_capability_registry.go`, `services/core/internal/api/server.go` | code | Keep override-store load/save tests green | no |
| Approval/audit export without restore parity | Operational forensics/recovery | high | Critical governance records can be exported but not restored, weakening incident recovery and migration confidence | `services/core/internal/backup/service.go` | both | Add restore mappings for `audit_records`, `gateway_invocations`, approval tables | `DEMO`, `P6` |

## 3) Safety blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Autonomy default mode is `observe` | Autonomy safety posture | resolved | Fresh boot is observation-only; maintain/mission require explicit operator setting | `services/core/internal/api/autonomy_maintenance_loop.go` | both | Keep maintain/mission opt-in | no |
| Legacy adapter bypass can be re-enabled by env flag | Execution policy safety | resolved-note | Env-gated direct adapter route was removed from router wiring | `services/core/internal/api/server.go` | code | Keep no direct adapter execution ingress outside gateway | no |
| Medium-risk active capabilities default-executable | Tool risk posture | medium | `filesystem.write_file`, `filesystem.move_file`, `observability.read_logs` are active by default and rely on policy/profile discipline | `services/core/internal/gateway/tool_capability_registry.go` | both | Add stricter default profile or explicit rollout policy for medium-risk actives | `AUTO`, `DEMO` |

## 4) Duplicate-system blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Observation reads vs cognitive objects | Memory architecture | medium | Observation read/VSA inspection remains for compatibility beside cognitive filesystem reads | `services/core/internal/memory/*`, `services/core/internal/aios/controllane/*`, `store/migrate.go` | both | Keep observation APIs read-only and expose syscall-native truth reads in UI/API | `P6`, `IRIS` |
| `compute` vs `computelane` split remains ambiguous | Lane architecture | medium | Interface/runtime split is not clearly enforced and can drift into parallel semantics | `services/core/internal/aios/compute/*`, `services/core/internal/aios/computelane/*` | both | Pin one namespace as interface-only and enforce import direction | `P6` |
| Gateway vs adapter invoke duplication | Tool runtime boundary | resolved-note | Direct adapter invoke surface removed; gateway is sole execution path | `services/core/internal/api/server.go`, `apps/desktop/src/lib/api.ts` | both | Keep gateway-only execution and tests that enforce no alternate ingress | no |

## 5) Validation blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Root JS/TS test command | Frontend validation | partial | `npm test` exists and delegates to Go core tests; dedicated JS unit tests are still absent | `package.json` | both | Add package-level desktop/shared JS tests when test harness is introduced | `DEMO` |
| Root lint/typecheck commands | Static quality gates | resolved | `npm run lint` delegates to Go vet; `npm run typecheck` runs desktop TS checks | `package.json` | both | Keep commands green and add frontend lint when configured | no |
| CI workflow definitions | Continuous verification | resolved | CI workflow runs install, Go tests/vet, desktop typecheck/build, and smoke | `.github/workflows/ci.yml` | both | Keep workflow aligned with local commands | no |

## 6) Desktop/UI blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Adapter probe UI still targets legacy path | Desktop mutation safety | high | Desktop invites direct adapter invocation instead of gateway path | `apps/desktop/src/pages/AdaptersPage.tsx`, `apps/desktop/src/lib/api.ts` | both | Replace probe with gateway-invoke flow or remove probe action | `AUTO`, `DEMO` |
| No dedicated trace/explain surface | Explainability/operator UX | medium | Correlation data exists but no coherent end-to-end UI for incident/debug tracing | `apps/desktop/src/App.tsx` (no trace route), backend trace APIs | both | Add focused trace inspector for correlationId/traceId chain | `P6`, `DEMO` |
| Truth objects not fully inspectable in UI | Truth transparency | medium | Contradictions/supersessions/context snapshots are backend concepts with weak UI presence | desktop pages vs `aios/truth` + syscall repos | both | Add read-only truth object surfaces before further autonomy expansion | `P6`, `AUTO`, `DEMO` |

## 7) Tool/gateway blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Capability status persistence missing | Tool governance | resolved | Policy posture persists through SQLite overrides and reloads at startup | `gateway/tool_capability_registry.go`, `api/phase5.go`, `api/server.go` | code | Keep persistence tests green | no |
| Most taxonomy entries are non-executable | Tool surface completeness | resolved | Registry breadth resolves to real gateway tools; high-risk entries are `approval_only` | `gateway/tool_capability_registry.go`, `gateway/service.go`, `gateway/capability_backing_tool.go` | both | Keep dependency gaps as explicit runtime errors, not stubs | no |
| Chat assistant external adapter path bypasses gateway taxonomy | Tool-policy consistency | medium | LLM adapter calls in chat path are outside gateway capability policy model | `services/core/internal/api/chat_post.go` | code | Introduce gateway-mediated wrapper for external model/tool calls or document strict boundary exception | `P6`, `IRIS`, `DEMO` |

## 8) Memory/truth blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Memory truth authority is split | Memory/truth model | resolved for mutation | Observation mutation endpoints are retired; cognitive FS/syscall path is authoritative for mutation | `internal/memory/*`, `internal/aios/controllane/*`, `internal/aios/truth/*` | both | Keep observation API read-only; build syscall-native truth APIs/UI | no |
| Projection repair is not a complete operator workflow | Truth repairability | medium | Dry-run report exists but no full repair command/API flow for operator operations | `services/core/internal/aios/truth/engine.go` | both | Add explicit repair command/API with guardrails and audit | `P6`, `DEMO` |
| Truth engine visibility is mostly backend | Truth operations | medium | Limited direct user/API surfaces for state explanation and resolver traces | `services/core/internal/aios/truth/*`, desktop routes | both | Add read APIs + UI pages for current/historical truth and resolver diagnostics | `P6`, `DEMO` |

## 9) Autonomy blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Rule-agent set is incomplete | Deterministic maintenance | high | Only two rule agents are implemented; expected maintenance coverage is incomplete | `services/core/internal/aios/autonomy/rule_agents.go` | both | Implement missing agents or explicitly gate/defer them in runtime policy | `AUTO`, `P6` |
| Maintain-mode defaults plus seeded charters/budgets | Autonomy governance | resolved | System starts in `observe`; default charters/budgets are inspectable assets, not auto-commit activation | `services/core/internal/api/autonomy_maintenance_loop.go`, `aios/autonomy/defaults.go` | both | Keep maintain/mission explicit operator choices | no |
| Autonomy traceability remains partial in operator surfaces | Autonomy explainability | medium | Decisions/intents are persisted but cross-lane trace traversal is limited | `aios/autonomy/explain.go`, desktop autonomy pages | both | Add joined intent->decision->budget->syscall->audit views | `AUTO`, `DEMO` |

## 10) Bring-up blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| VSA source tracking | Core bring-up | resolved | Required VSA files are tracked and strict preflight checks enforce presence/tracked state | `services/core/internal/memory/vsa_engine.go`, `vsa_indexer.go`, `vsa_signals.go`, `scripts/check-vsa-files.sh` | code | Keep preflight enforced on core/test/smoke paths | no |
| Desktop runtime depends on host WebKit/GTK stack | Desktop bring-up | medium | Desktop boot is environment-sensitive even though desktop build passes | `scripts/check-desktop-deps.sh`, `apps/desktop/src-tauri/*` | both | Keep dependency preflight strict and publish platform-specific setup steps | `DEMO` |

## 11) Nix blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Nix validation unavailable in current environment | Nix foundation verification | high | `nix flake check`, `nix build`, `nix develop` all fail due daemon socket unavailability | `flake.nix`, local Nix daemon environment | both | Validate in a Nix-enabled runner/host and record authoritative results | `NIX` |
| Only `forge-core` package exists; no desktop package | Nix packaging scope | medium | N1 is partial; desktop/package parity is incomplete | `flake.nix`, `nix/packages/forge-core.nix` | both | Add `forge-desktop` package when desktop build inputs are stable in Nix | `NIX` |
| Nix modules/tool-capsules/profiles are README-only | Deep Nix integration | medium | Present as placeholders, not runnable integration units | `nix/modules/README.md`, `nix/tool-capsules/README.md`, `nix/profiles/README.md` | both | Keep explicitly non-authoritative until implemented and tested | `NIX`, `DEMO` |

---

## Immediate priority order (blunt)

1. Close backup restore parity for critical exported sections (`critical`).
2. Keep retired memory mutation endpoints non-executable and continue adding syscall-native memory/state write facades.
3. Add dedicated JS/TS unit tests when the frontend harness is introduced.
4. Keep CI and local validation commands green.
