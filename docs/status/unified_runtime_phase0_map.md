# Unified Runtime Phase 0 Map (Reality Lock)

Date: 2026-04-23  
Scope: CODEX.md ordered-phase execution start (`Phase 0 -> Phase 1 ...`).

## 1) Authoritative Seams (Current Reality)

### Kernel / authority
- Semantic syscall kernel is the canonical mutation boundary:
  - `services/core/internal/aios/controllane/processor.go`
  - `services/core/internal/aios/controllane/validator.go`
  - `services/core/internal/aios/controllane/approval.go`
  - `services/core/internal/aios/domain/types.go`
- Durable cognitive persistence behind syscall commit path:
  - `services/core/internal/aios/controllane/sqlite_store.go`
  - `services/core/internal/store/migrate.go`
- Append-only journal truth constraints (DB triggers):
  - `services/core/internal/store/migrate.go` (`journal_events_no_update`, `journal_events_no_delete`)

### Tool execution authority
- Gateway is the authoritative tool execution ingress:
  - `services/core/internal/gateway/service.go`
  - API wiring: `services/core/internal/api/server.go`
- Legacy adapter direct invoke ingress removed from routing (already landed on current baseline).

### Model execution authority
- Model runtime is the authoritative model execution substrate:
  - `services/core/internal/modelruntime/service.go`
  - API bridge/surface: `services/core/internal/api/model_runtime*.go`
- Current behavior is governed and bounded (queue/admission/lifecycle/policy/audit), but no dedicated multi-role orchestrator service yet.

### Context compile / restore substrate
- `COMPILE_CONTEXT` syscall path supports snapshot evidence persistence and optional SVG artifact generation:
  - `services/core/internal/aios/controllane/processor_apply.go`
  - `services/core/internal/aios/controllane/compile_context_snapshot.go`
  - `services/core/internal/aios/controllane/sqlite_store.go`
- Status is partial: deterministic scoring/ranking is now implemented, with operator-facing inspection still pending.

### Maintenance and autonomy substrate
- Bounded autonomy maintenance loop exists with cooldown and idle gating:
  - `services/core/internal/api/autonomy_maintenance_loop.go`
- Autonomy policy and durable backing checks are real:
  - `services/core/internal/aios/autonomy/policy_evaluator.go`
- Maintenance exists, but dry-run/reporting depth and operator diagnostics are still incomplete.

## 2) Legacy / Side-Door Reality To Converge

These are real and must remain explicitly bounded until converged:

- V1 memory path still performs direct writes outside semantic syscall kernel:
  - `services/core/internal/memory/service.go`
  - `services/core/internal/memory/retrieval.go`
  - `services/core/internal/memory/packets.go`
- Project context and packet projection writes are still direct service-layer persistence:
  - `services/core/internal/projectcontext/service.go`
  - `services/core/internal/packets/service.go`
- Job projection path is authoritative for job lifecycle but not yet unified as a syscall-only semantic mutation surface:
  - `services/core/internal/jobs/service.go`
- Dual event truth streams still coexist (`events` and `journal_events`):
  - `services/core/internal/events/logger.go`
  - `services/core/internal/aios/controllane/sqlite_store.go`

These are currently bounded by doctrine/docs and scope controls, but not fully converged to one semantic write substrate.

## 3) Ordered Execution Plan (From This Point)

### Phase 1 — Authority convergence (current in-progress phase)
1. Add hard guardrails that prevent new direct-write bypasses for canonical AI-OS cognitive tables.
2. Keep existing bounded compatibility paths explicit (do not silently expand side doors).
3. Preserve audit/correlation/trace linkage on governed paths.
4. Document bounded exceptions (backup restore path, schema migration path).

### Phase 2 — Arterial runtime completion
1. Implement deterministic restore candidate listing/ranking (scope/query/kind).
2. Persist non-empty restore score breakdowns and resume hints.
3. Add header-first restore preview + expanded hydrate path.
4. Integrate restore-aware selection into compile flow with inspectable ranking traces.

### Phase 3 — GHOST runtime integration
1. Introduce explicit orchestration layer over modelruntime + gateway (planner/executor/verifier style roles).
2. Keep tool access gateway-only and durable writes syscall-only.
3. Add bounded retry policy with cooldown-aware provider handling and checkpointable execution state.

### Phase 4 — Lymphatic formalization
1. Add true dry-run maintenance mode for repair/reindex/consolidation tasks.
2. Expand diagnostics reporting with surfaced query failures and actionable operator output.
3. Keep scheduler bounded/idempotent with explicit report artifacts.

### Phase 5 — ARTEMIS operator legibility
1. Add snapshot inspector and restore score viewer.
2. Add structured context packet inspector.
3. Upgrade audit page to true trace explorer (journal + gateway + artifact lineage).
4. Add unified runtime execution explorer by correlation/trace.

## 4) Explicit Boundaries To Preserve While Implementing

- FORGE kernel remains canonical truth authority.
- Durable semantic writes remain behind semantic syscalls/control lane commit.
- Gateway remains tool execution authority.
- Model runtime remains inference authority.
- Snapshot artifacts remain non-canonical evidence.
- All flows preserve correlation/trace/syscall/audit provenance linkage.

## 5) Phase 1 Progress Landed In This Pass

- Added a code-level guard test to prevent new direct-write paths into canonical cognitive filesystem tables outside bounded kernel/restore locations:
  - `services/core/internal/aios/controllane/authority_guard_test.go`
- Guard currently allows only:
  - `services/core/internal/aios/controllane/sqlite_store.go` (kernel commit path)
  - `services/core/internal/backup/service.go` (bounded restore path)
  - `services/core/internal/store/migrate.go` (schema/migration path)
- This does not yet remove all legacy v1 write surfaces; it hard-bounds expansion while convergence work proceeds.

## 6) Phase 2 Progress Landed In This Pass

- Added deterministic restore candidate listing at the semantic read-store boundary:
  - `ListContextSnapshots(scope, query, snapshotKind, limit)`
  - implemented for in-memory, transactional, and SQLite stores.
- Extended `COMPILE_CONTEXT` snapshot persistence path to:
  - rank restore candidates deterministically
  - apply thresholded selection with explicit fresh-compile fallback
  - persist inspectable `restore_scores_json` and `resume_hints_json`
  - preserve existing syscall/audit/provenance boundaries.
- Added header-first restore decode behavior so ranking/selection can still operate when only snapshot headers are available.
- Added tests for:
  - deterministic ranking and stale/contradiction penalties
  - header-first candidate behavior
  - below-threshold and forced-fresh fallback paths
  - scope/query/kind candidate listing behavior
  - persisted restore score/resume-hint metadata coverage.

## 7) Observed Phase 3-5 Runtime Reality

- Phase 3 persistence is real:
  - SQLite-backed semantic repositories, transaction runner, and audit linkage exist under `services/core/internal/aios/controllane/*`.
- Phase 4 ingest runtime is real but bounded:
  - `services/core/internal/aios/compute/librarian/pipeline.go` appends/reuses journal events, orders cell proposals, and routes accepted actions back through the kernel.
- Phase 5 truth services are real but not yet exclusive:
  - `services/core/internal/aios/truth/engine.go` exposes current/timeline/evidence/explain paths, while legacy v1 memory writes still coexist.
- Phase 5.75 autonomy is policy-bounded, not free-running:
  - durable backing checks block maintain/mission auto-commit without charter+budget persistence.
- Phase 5.9 tool policy is real:
  - gateway status/risk/approval logic enforces `disabled`, `stubbed`, `deferred`, and `approval_only` outcomes and does not let `future_iris` bypass policy.
