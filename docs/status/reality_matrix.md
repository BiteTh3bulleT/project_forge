# FORGE Reality Matrix

Date: 2026-04-21  
Evidence basis: repository code paths + local command runs in this workspace.

Status values used: `real`, `partial`, `scaffold`, `stubbed`, `placeholder`, `example-only`, `docs-only`, `duplicated`, `risky`, `blocked`, `unknown`.

| Subsystem | Claimed In Docs | Real In Code | Boots/Runs | Durable | Tested | UI Visible | Status | Notes |
|---|---|---|---|---|---|---|---|---|
| Desktop shell | `README.md` claims real monitor-aware desktop shell | `apps/desktop/src/App.tsx`, `apps/desktop/src/layout/AppShell.tsx` route real surfaces | `npm -w @forge/desktop run build` passes | local storage + backend APIs | no desktop test suite | yes | real | Real shell exists; no desktop regression test harness. |
| Workspace windows + monitor-aware layouts | `README.md` claims real multi-window monitor detection | `apps/desktop/src/stores/workspaceLayoutStore.ts` uses Tauri window + monitor APIs | build passes; runtime needs Tauri/GUI libs | local layout persistence | no | yes | real | Real implementation; browser mode cannot simulate monitors by design. |
| Chat attachments + inspector | `README.md` claims thread-linked artifacts and inspector | `apps/desktop/src/lib/api.ts` attachment types + chat endpoints; chat routes in `App.tsx` | core + desktop build pass | attachments durable in DB/artifacts | partial (Go API tests; no JS tests) | yes | partial | Attachment path is real; UI behavior lacks dedicated test coverage. |
| Jobs/projections | docs claim jobs are projections | `services/core/internal/jobs/service.go`, `jobs` tables in `store/migrate.go` | `go test ./...` pass; smoke `/api/jobs` pass | yes | partial | yes | real | Core path is stable and booted. |
| Approvals/gates | docs claim approvals are enforced gates | `internal/approvals`, API routes in `server.go` | booted via smoke + core tests pass | yes | partial | yes | real | Approval records are durable; restore parity is incomplete. |
| Audit/correlation | docs claim everything important is audited | `internal/audit`, gateway + syscall audit sinks | booted; `go test` pass | yes | partial | partial | partial | Audit exists and is active; no unified UI trace explorer. |
| Artifacts/evidence | docs claim artifacts are evidence | `internal/artifacts`, API `/api/artifacts/*` | booted | yes | partial | partial | partial | Evidence layer exists; UI inspection is present but limited. |
| Adapters registry | docs claim bounded workers | `internal/adapters`, `/api/adapters` routes | booted (`/api/adapters` smoke pass) | adapter status cache in DB | partial | yes | partial | Registry is real; execution occurs through gateway path. |
| Legacy adapter direct invoke | architecture says gateway is sole path | `/api/adapters/{id}/invoke` route removed in `api/server.go` | returns `404` (not routed) | n/a | yes (server adapter guard test) | no direct invoke surface | resolved | Legacy side door removed; no executable direct adapter ingress remains. |
| Gateway/tool execution path | docs claim gateway is only authorized tool boundary | `internal/gateway/service.go`, `/api/gateway/invoke` | `go test`, smoke, build pass | invocation + audit durable | yes | yes | real | Gateway is the sole tool execution ingress in API routing. |
| Tool capability registry | docs claim full taxonomy with statuses | `gateway/tool_capability_registry.go` + domain types | booted | registry defaults are code-defined; runtime status overrides not persisted | yes | yes | partial | Full taxonomy registered; most entries are non-executable by status. |
| Tool policy evaluator | docs claim status/risk/approval enforcement | `gateway/tool_policy.go` | tested and running | n/a | yes | partial | real | Enforces `approval_only`/`disabled`/`stubbed`/`deferred` terminal behavior. |
| Capability status governance API | docs claim explicit status governance | `PATCH /api/gateway/capabilities/{id}/status` in `api/phase5.go` | booted | audit durable; status override not durable | yes | yes | partial | Unknown statuses rejected; reason required for restricted status transitions. |
| Semantic syscall registry | docs claim deterministic syscall lane | `aios/controllane/registry.go` | `go test` pass | code + DB-backed repos | yes | no | real | Starter syscall set is implemented and tested. |
| Syscall validator | docs claim deterministic validation | `aios/controllane/validator.go` | tests pass | n/a | yes | no | real | Envelope + payload validation paths are real. |
| Syscall processor/committer | docs claim kernel validates then commits | `aios/controllane/processor.go` + `processor_apply.go` | tests pass | yes | yes | no | real | Dry-run and commit paths exist with audit linkage. |
| Raw semantic journal (`journal_events`) | docs claim append-only events truth | `journal_events` table + triggers in `store/migrate.go`, append in processor | tests pass | yes | yes | no | real | DB triggers enforce append-only semantics. |
| Legacy event stream (`events`) | docs map events as truth | `internal/events/logger.go` writes `events` table | boots/runs | yes | partial | yes (`/api/events`) | duplicated | Two event streams (`events` and `journal_events`) are both active. |
| Cognitive filesystem (notes/links/state/loops/models/etc.) | docs claim Phase 3 cognitive FS | tables and repos in `controllane` + `store/migrate.go` | core tests pass | yes | partial | partial | partial | Durable core exists; v1 memory APIs still bypass syscall path. |
| Memory observations (v1 memory path) | v1 docs describe observation memory architecture | `internal/memory/service.go` direct DB writes and links | boots/runs | yes | partial | yes | duplicated | Writes bypass semantic syscall kernel; coexists with v2 memory notes model. |
| Active state + history | docs claim current vs historical truth separation | `state_items`, `state_versions` tables + truth engine | tests pass (`truth/engine_test.go`) | yes | yes | partial | partial | Real engine logic; limited UI/API surfacing for v2 truth objects. |
| Open loops | docs claim loop lifecycle | `open_loops` table + truth engine loop APIs | tests pass | yes | yes | partial | partial | Lifecycle logic exists; no dedicated full-feature loop management UI. |
| Contradictions + supersessions | docs claim contradiction/supersession handling | repos + truth engine methods and tests | tests pass | yes | yes | partial | partial | Data model and tests exist; backend/UI inspection remains limited. |
| Derived models | docs claim evidence-backed derived models | `derived_models` table + syscall action | tests pass | yes | partial | partial | partial | Implemented as advisory layer; not fully surfaced in UI. |
| Context packet snapshots | docs claim compile context syscall support | `context_packet_snapshots` table + `COMPILE_CONTEXT` action | tests pass | yes | partial | no | partial | Exists in core; no strong UI for packet snapshot inspection. |
| Truth engine | docs claim current-truth engine | `aios/truth/engine.go` + tests | tests pass | yes | yes | no | partial | Engine is real but not the only memory/state authority in runtime. |
| Projection rebuild/repair | docs describe rebuild/repair | `RebuildProjection` in truth engine exists | tests pass for dry-run deterministic report | yes | yes | no | partial | Dry-run/report exists; not a complete operator repair workflow. |
| Ingest pipeline + librarian cells | docs claim event->cell->action->syscall flow | `aios/compute/librarian/pipeline.go`, `cells_phase4.go` | tests pass (`pipeline_phase4_test.go`) | yes (through syscall commits) | yes | no | partial | Real pipeline and cells exist in core; not surfaced as explicit external API workflow. |
| Semantic inference seam | docs claim no live LLM required | `NoopSemanticInference` in `inference.go` | boots/runs | n/a | yes | no | scaffold | Seam exists with no-op default; non-noop inference remains optional future work. |
| Rule-agent layer | docs list broad deterministic agents | `aios/autonomy/rule_agents.go` has `OpenLoopStalenessAgent` + `CleanupProposalAgent` only | booted via autonomy loop | intents/decisions durable | partial | partial | partial | Real but limited: expected agent set is not fully implemented. |
| Autonomy layer | docs claim charter/budget/policy/runner | `aios/autonomy/*`, `api/autonomy_maintenance_loop.go` | `/api/autonomy/status` smoke pass | yes (SQLite settings-backed repos) | yes | yes | partial | Real core exists; default mode is `maintain`, and governance/visibility still incomplete. |
| Self-commit persistence gate | docs claim maintain/mission blocked without durability | `autonomy/policy_evaluator.go` `hasDurableSelfCommitBacking` | tests pass | n/a | yes | no | real | Explicit quarantine exists for non-durable backing scenarios. |
| Future IRIS seam | docs claim future service cannot bypass policy | source enum + policy checks/tests (`future_iris`) | n/a | n/a | yes | no | scaffold | Boundary enforcement is real; no IRIS runtime service implemented. |
| Desktop approvals/audit/tool surfaces | docs claim inspection shell | pages exist: `ApprovalsPage`, `AuditPage`, `ToolGatewayPage` | build passes | API-backed | partial | yes | partial | Surfaces exist; trace-depth and explainability remain incomplete. |
| Desktop trace/explain surface | architecture expects traceability | no dedicated route/page for end-to-end trace drilldown | n/a | n/a | no | no | blocked | Correlation exists in backend, but operator-facing trace UX is missing. |
| Go test surface | docs claim Go test command | `cd services/core && go test ./...` | pass | n/a | yes | n/a | real | Current workspace run passed. |
| Go vet surface | docs mark optional static checks | `cd services/core && go vet ./...` | pass | n/a | yes | n/a | real | Current workspace run passed. |
| Root JS/TS test surface | AGENTS says no dedicated root test script | root `package.json` has no `test` script | `npm test` fails | n/a | no | n/a | blocked | Missing script blocks unified JS regression gate. |
| Root lint surface | AGENTS says no dedicated root lint script | root `package.json` has no `lint` | `npm run lint` fails | n/a | no | n/a | blocked | Missing static quality gate at root. |
| Root typecheck surface | AGENTS says no dedicated root typecheck script | root `package.json` has no `typecheck` | `npm run typecheck` fails | n/a | no | n/a | blocked | Missing global TS validation command. |
| Desktop build validation | docs describe desktop build | `npm -w @forge/desktop run build`, `cargo check` | pass | n/a | partial | n/a | real | Build is green; no desktop unit/integration tests. |
| Smoke validation | runbook/status docs claim smoke script | `scripts/forge-smoke.sh`, `npm run smoke` | pass | n/a | yes | n/a | real | Exercises core health/meta/autonomy/adapters/jobs endpoints. |
| Migration validation | docs mention migration tests | `internal/store/migrate_vsa_test.go` + store tests | `go test` pass | n/a | partial | n/a | partial | Some migration coverage exists; restore parity tests remain narrow. |
| CI automation | implied by enterprise-grade claims in docs | `.github/workflows` missing | n/a | n/a | no | n/a | blocked | No visible CI workflow layer in repo. |
| Nix flake (N1) | docs claim light Nix foundation | `flake.nix`, `nix/*` present | local `nix flake check` fails (daemon socket unavailable) | n/a | unknown | no | blocked | Existence is real; local validation is blocked by environment daemon unavailability. |
| Nix dev shells | docs claim default/core/desktop/aios shells | `nix/shells/*.nix` present | `nix develop --command true` fails (daemon) | n/a | unknown | no | blocked | Cannot validate shell runtime in this environment. |
| Nix forge-core package | docs claim package exists | `nix/packages/forge-core.nix` present | `nix build .#forge-core` fails (daemon) | n/a | unknown | no | blocked | Package spec exists; build not validated here due daemon issue. |
| Nix checks | docs claim flake checks configured | `nix/checks/go-tests.nix`, `go-vet.nix`, `js-build.nix` | not runnable here (daemon) | n/a | unknown | no | blocked | Check definitions exist but execution evidence unavailable in this environment. |
| Nix modules | docs list deep NixOS as future | `nix/modules/README.md` only | n/a | n/a | no | no | docs-only | No runnable NixOS module implementation. |
| Nix tool capsules | docs list as future | `nix/tool-capsules/README.md` only | n/a | n/a | no | no | docs-only | No capsule runtime path implemented. |

## Command evidence captured in this pass

- `cd services/core && go test ./...` -> pass
- `cd services/core && go vet ./...` -> pass
- `npm run build` -> pass
- `npm run smoke` -> pass
- `cd apps/desktop/src-tauri && cargo check` -> pass
- `npm test` -> missing script
- `npm run lint` -> missing script
- `npm run typecheck` -> missing script
- `nix ... flake check/build/develop` -> failed: daemon socket unavailable
