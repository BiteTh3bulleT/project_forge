# FORGE v1/v2 Cutover Blockers

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
| v1 memory mutation bypasses syscall kernel | Memory/state authority | high | `/api/memory/observations` writes directly to DB, bypassing semantic syscall validation and unified truth controls | `services/core/internal/api/server.go` (`/memory/observations` routes), `services/core/internal/memory/service.go` | code | Route writes through `controllane.Processor` or freeze legacy writes to read-only | `P6`, `AUTO`, `IRIS`, `DEMO` |
| Dual event streams remain active | Truth/event authority | high | `events` and `journal_events` both receive runtime events; no single replay authority is enforced | `services/core/internal/events/logger.go`, `services/core/internal/aios/controllane/processor.go`, `services/core/internal/store/migrate.go` | both | Declare canonical stream, bridge/retire secondary stream, and update API/UI to one source | `P6`, `IRIS`, `DEMO` |
| No first-class external API for semantic syscall/truth operations | v2 external surface | medium | v2 kernel/truth is real internally but external flows remain mostly v1 API surfaces | `services/core/internal/api/server.go`, `services/core/internal/aios/*` | both | Add explicit API facade for syscall/truth reads and cut consumers to it | `P6`, `IRIS` |

## 2) Durability blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Backup export/restore parity is incomplete | Backup/import durability | critical | `full_backup` exports many AI-OS tables but restore maps only a small subset; recovery claims overstate reality | `services/core/internal/backup/service.go` (`pickSections`, `insertStatements`) | both | Implement restore mappings for exported critical sections or shrink export claims to restorable subset | `P6`, `AUTO`, `IRIS`, `DEMO` |
| Capability status overrides are not durable | Tool governance durability | high | Runtime capability status changes reset on restart because updates are in-memory registry only | `services/core/internal/gateway/tool_capability_registry.go`, `services/core/internal/api/phase5.go` | code | Persist capability status overrides in DB settings and load on boot | `AUTO`, `DEMO` |
| Approval/audit export without restore parity | Operational forensics/recovery | high | Critical governance records can be exported but not restored, weakening incident recovery and migration confidence | `services/core/internal/backup/service.go` | both | Add restore mappings for `audit_records`, `gateway_invocations`, approval tables | `DEMO`, `P6` |

## 3) Safety blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Autonomy default mode is `maintain` | Autonomy safety posture | high | Fresh boot enables maintain-mode runtime rather than explicit operator opt-in mode (`off`/`observe`) | `services/core/internal/api/autonomy_maintenance_loop.go` | both | Change default mode to `off` or `observe`, require explicit operator enable for maintain/mission | `AUTO`, `DEMO` |
| Legacy adapter bypass can be re-enabled by env flag | Execution policy safety | resolved-note | Env-gated direct adapter route was removed from router wiring | `services/core/internal/api/server.go` | code | Keep no direct adapter execution ingress outside gateway | no |
| Medium-risk active capabilities default-executable | Tool risk posture | medium | `filesystem.write_file`, `filesystem.move_file`, `observability.read_logs` are active by default and rely on policy/profile discipline | `services/core/internal/gateway/tool_capability_registry.go` | both | Add stricter default profile or explicit rollout policy for medium-risk actives | `AUTO`, `DEMO` |

## 4) Duplicate-system blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| v1 memory observations vs v2 cognitive objects | Memory architecture | high | Two overlapping memory models continue in live runtime with separate mutation paths | `services/core/internal/memory/*`, `services/core/internal/aios/controllane/*`, `store/migrate.go` | both | Define one authoritative mutation path and a compatibility adapter for legacy reads | `P6`, `IRIS`, `DEMO` |
| `compute` vs `computelane` split remains ambiguous | Lane architecture | medium | Interface/runtime split is not clearly enforced and can drift into parallel semantics | `services/core/internal/aios/compute/*`, `services/core/internal/aios/computelane/*` | both | Pin one namespace as interface-only and enforce import direction | `P6` |
| Gateway vs adapter invoke duplication | Tool runtime boundary | resolved-note | Direct adapter invoke surface removed; gateway is sole execution path | `services/core/internal/api/server.go`, `apps/desktop/src/lib/api.ts` | both | Keep gateway-only execution and tests that enforce no alternate ingress | no |

## 5) Validation blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| No root JS/TS test command | Frontend validation | high | `npm test` fails; no standard frontend regression entrypoint | `package.json` | both | Add root test script and package-level tests for desktop/shared | `P6`, `DEMO` |
| No root lint/typecheck commands | Static quality gates | high | `npm run lint` and `npm run typecheck` fail; no mandatory static checks | `package.json` | both | Add root lint/typecheck scripts and wire to CI | `P6`, `DEMO` |
| No visible CI workflow definitions | Continuous verification | high | No `.github/workflows` means no automated gate on merges | repository root (`.github/workflows` missing) | both | Add CI for Go tests, desktop build, smoke, and optional Nix checks | `P6`, `NIX`, `DEMO` |

## 6) Desktop/UI blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Adapter probe UI still targets legacy path | Desktop mutation safety | high | Desktop invites direct adapter invocation instead of gateway path | `apps/desktop/src/pages/AdaptersPage.tsx`, `apps/desktop/src/lib/api.ts` | both | Replace probe with gateway-invoke flow or remove probe action | `AUTO`, `DEMO` |
| No dedicated trace/explain surface | Explainability/operator UX | medium | Correlation data exists but no coherent end-to-end UI for incident/debug tracing | `apps/desktop/src/App.tsx` (no trace route), backend trace APIs | both | Add focused trace inspector for correlationId/traceId chain | `P6`, `DEMO` |
| v2 truth objects not fully inspectable in UI | Truth transparency | medium | Contradictions/supersessions/context snapshots are backend concepts with weak UI presence | desktop pages vs `aios/truth` + syscall repos | both | Add read-only truth object surfaces before further autonomy expansion | `P6`, `AUTO`, `DEMO` |

## 7) Tool/gateway blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Capability status persistence missing | Tool governance | high | Policy posture can drift after restart; runtime governance changes are ephemeral | `gateway/tool_capability_registry.go`, `api/phase5.go` | code | Persist status overrides and reload on startup | `AUTO`, `DEMO` |
| Most taxonomy entries are non-executable | Tool surface completeness | medium | Registry breadth exceeds implemented adapters; registration != execution | `gateway/tool_capability_registry.go`, `gateway/service.go` | both | Keep explicit status/risk display and prioritize high-value adapter implementations | `AUTO`, `DEMO` |
| Chat assistant external adapter path bypasses gateway taxonomy | Tool-policy consistency | medium | LLM adapter calls in chat path are outside gateway capability policy model | `services/core/internal/api/chat_post.go` | code | Introduce gateway-mediated wrapper for external model/tool calls or document strict boundary exception | `P6`, `IRIS`, `DEMO` |

## 8) Memory/truth blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Memory truth authority is split | Memory/truth model | high | v1 observation model and v2 cognitive FS both mutate durable memory, creating drift risk | `internal/memory/*`, `internal/aios/controllane/*`, `internal/aios/truth/*` | both | Formalize one authoritative mutation lane and compatibility adapters | `P6`, `IRIS`, `DEMO` |
| Projection repair is not a complete operator workflow | Truth repairability | medium | Dry-run report exists but no full repair command/API flow for operator operations | `services/core/internal/aios/truth/engine.go` | both | Add explicit repair command/API with guardrails and audit | `P6`, `DEMO` |
| Truth engine visibility is mostly backend | Truth operations | medium | Limited direct user/API surfaces for state explanation and resolver traces | `services/core/internal/aios/truth/*`, desktop routes | both | Add read APIs + UI pages for current/historical truth and resolver diagnostics | `P6`, `DEMO` |

## 9) Autonomy blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Rule-agent set is incomplete | Deterministic maintenance | high | Only two rule agents are implemented; expected maintenance coverage is incomplete | `services/core/internal/aios/autonomy/rule_agents.go` | both | Implement missing agents or explicitly gate/defer them in runtime policy | `AUTO`, `P6` |
| Maintain-mode defaults plus seeded charters/budgets | Autonomy governance | high | System starts with active autonomy governance assets without explicit operator onboarding | `services/core/internal/api/autonomy_maintenance_loop.go`, `aios/autonomy/defaults.go` | both | Require explicit bootstrap step for maintain/mission mode activation | `AUTO`, `DEMO` |
| Autonomy traceability remains partial in operator surfaces | Autonomy explainability | medium | Decisions/intents are persisted but cross-lane trace traversal is limited | `aios/autonomy/explain.go`, desktop autonomy pages | both | Add joined intent->decision->budget->syscall->audit views | `AUTO`, `DEMO` |

## 10) Bring-up blockers

| Title | Subsystem | Severity | Why it matters | Files/Modules | Scope | Minimal next fix | Blocks |
|---|---|---|---|---|---|---|---|
| Untracked VSA files are required for clean build | Core bring-up | critical | Fresh clones can fail build if `vsa_engine.go`, `vsa_indexer.go`, `vsa_signals.go` are absent from tracked history | `services/core/internal/memory/vsa_engine.go`, `vsa_indexer.go`, `vsa_signals.go` | code | Commit required VSA files (and tests) or remove tracked references | `P6`, `AUTO`, `IRIS`, `DEMO` |
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
2. Cut over v1 memory mutation endpoints to syscall-kernel authority (`high`).
3. Commit required VSA files so fresh clones compile (`critical`).
4. Add JS/TS test/lint/typecheck + CI gates (`high`).
