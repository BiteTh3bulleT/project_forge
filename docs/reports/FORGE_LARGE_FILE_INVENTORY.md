# FORGE Large File Inventory + Split Plan

**Generated:** 2026-05-11.
**Companion to:** [FORGE_FULL_REVIEW.md](FORGE_FULL_REVIEW.md).
**Scope:** Every source file over 700 lines in the repo, across Go, TS/TSX, Rust, Nix, and tests. Plus a recommended order of operations and split strategy per file.

> **Rule of thumb:** A single source file >1,500 lines is an architectural smell. A file >2,500 lines is technical debt actively gating future work. A file >3,000 lines is a refactor that pays for itself on the next bug.

---

## 1. Full Inventory

### Go non-test files >700 lines

| Lines | File | Domain | Status |
|---|---|---|---|
| 2,005 | `services/core/internal/backup/service.go` | Backup/restore/export | **P0 — biggest single file in Go.** |
| 1,581 | `services/core/internal/modelruntime/service.go` | Model runtime lifecycle | P1 |
| 1,407 | `services/core/internal/api/autonomy_maintenance_loop.go` | Autonomy maintenance | Done — type/state split landed. |
| 1,478 | `services/core/internal/aios/controllane/compile_context_restore_scoring.go` | Context restore scoring | P1 |
| 1,452 | `services/core/internal/jobs/service.go` | Job orchestration | P1 |
| 1,447 | `services/core/internal/aios/dream/service.go` | Dream lane | P1 — recently grew |
| 1,431 | `services/core/internal/store/migrate_schema.go` | SQL migrations | P3 — long is inherent here |
| 1,413 | `services/core/internal/api/model_runtime_bridge.go` | API ↔ runtime bridge | P2 |
| 1,297 | `services/core/internal/api/phase5.go` | Phase historical handler | P2 — consider archiving |
| 1,280 | `services/core/internal/api/chat_post.go` | Chat POST handler | P2 |
| 1,193 | `services/core/internal/aios/compute/librarian/cells_phase4.go` | Cell pipeline | P2 |
| 1,160 | `services/core/internal/api/model_runtime.go` | Model runtime API | P2 |
| 1,060 | `services/core/internal/aios/truth/engine.go` | Current truth engine | P2 |
| 1,032 | `services/core/internal/gateway/service.go` | Gateway core (post-split) | P3 |
| 1,028 | `services/core/internal/retrieval/service.go` | Retrieval service | P2 |
| 975 | `services/core/internal/gateway/service_helpers.go` | Gateway helpers (post-split) | **Watch** — sibling of recent split |
| 963 | `services/core/internal/aios/controllane/store.go` | Controllane store glue | P2 |
| 923 | `services/core/internal/api/operator_inspector.go` | Operator inspector API | P3 |
| 910 | `services/core/internal/aios/controllane/validator.go` | Envelope validator | P3 |
| 863 | `services/core/internal/memory/vsa_indexer.go` | VSA indexing | P3 |
| 842 | `services/core/internal/api/chat_fs_fallback.go` | Chat filesystem fallback | P3 |
| 803 | `services/core/internal/api/trace_report.go` | Trace reports | P3 |
| 797 | `services/core/internal/modelruntime/orchestration.go` | Runtime orchestration | P3 |
| 782 | `services/core/internal/aios/controllane/compile_context_snapshot.go` | Context snapshots | P3 |
| 781 | `services/core/internal/api/phase3.go` | Phase historical handler | P3 |
| 760 | `services/core/internal/aios/compute/librarian/pipeline.go` | Librarian pipeline | P3 |
| 759 | `services/core/internal/aios/controllane/sqlite_store_helpers.go` | Controllane SQL helpers (post-split) | P3 |
| 758 | `services/core/internal/gateway/service_filesystem_git.go` | Gateway FS/git tools | P3 |
| 758 | `services/core/internal/aios/controllane/processor_apply.go` | Apply switch | P3 |
| 742 | `services/core/internal/embeddings/service.go` | Embeddings service | P3 |

### TS/TSX files >700 lines (excl. tests)

| Lines | File | Domain | Status |
|---|---|---|---|
| 1,436 | `apps/desktop/src/pages/ChatPage.tsx` | Chat page | Done — composer and inspector surfaces extracted. |
| 149 | `apps/desktop/src/lib/api.ts` | API client shim | **Done — split into domain modules.** |
| 1,295 | `apps/desktop/src/pages/InspectorsPage.tsx` | Inspectors page | Done — inspector panels extracted. |
| 1,466 | `apps/desktop/src/pages/ModelsPage.tsx` | Models page | Done — model panels extracted. |
| 1,498 | `apps/desktop/src/pages/SettingsPage.tsx` | Settings page | Done — settings panels extracted. |
| 998 | `apps/desktop/src/layout/AppShell.tsx` | App shell layout | Done — shell surface components extracted. |
| 1,374 | `apps/desktop/src/stores/workspaceLayoutStore.ts` | Workspace layout store | P2 |
| 1,320 | `apps/desktop/src/pages/DashboardPage.tsx` | Dashboard page | P2 |
| 1,107 | `apps/desktop/src/pages/MemoryPage.tsx` | Memory page | P2 |
| 951 | `apps/desktop/src/pages/ToolGatewayPage.tsx` | Tool gateway page | P2 |
| 902 | `apps/desktop/src/pages/JobDetailPage.tsx` | Job detail page | P2 |
| 799 | `packages/shared/src/index.ts` | Shared types barrel | P3 |
| 797 | `apps/desktop/src/lib/desktop.ts` | Desktop lib | P3 |
| 785 | `packages/shared/src/aios.ts` | AIOS shared types | P3 |
| 753 | `apps/desktop/src/pages/DossiersPage.tsx` | Dossiers page | P3 |

### Rust files >300 lines

| Lines | File | Status |
|---|---|---|
| 766 | `apps/desktop/src-tauri/src/main.rs` | P1 — grew with operator-window work |
| 696 | `crates/forgek-validate/src/lib.rs` | P3 — acceptable for a validator crate |
| 388 | `crates/forgek-validate/src/validate.rs` | OK |

### Go test files >700 lines (informational)

Test files grow naturally with smoke/integration coverage. These are listed for awareness but generally not split targets.

| Lines | File |
|---|---|
| 1,835 | `services/core/internal/api/chat_post_model_runtime_fallback_test.go` |
| 1,695 | `services/core/internal/gateway/tool_surface_test.go` |
| 1,252 | `services/core/internal/backup/service_test.go` |
| 976 | `services/core/internal/aios/controllane/processor_test.go` |
| 900 | `services/core/internal/api/model_runtime_test.go` |
| 878 | `services/core/internal/aios/controllane/sqlite_integration_test.go` |
| 868 | `services/core/internal/aios/compute/librarian/pipeline_phase4_test.go` |
| 816 | `services/core/internal/forgekshadow/observer_test.go` |
| 723 | `services/core/internal/modelruntime/service_test.go` |
| 720 | `services/core/internal/aios/controllane/compile_context_restore_scoring_test.go` |

### Nix files

All under 300 lines. **Clean — no splits needed.**

---

## 2. Order of Operations

### Why this order

- **Lived-pain first.** ChatPage and `lib/api.ts` are what you hit every time you use FORGE. Split them and tonight's session feels immediately better.
- **Hot-path next.** `backup/service.go` is the biggest Go file and a frequent target for restore/export operations.
- **Domain boundaries third.** Pages and runtime files split cleanly along existing domain seams that the small-file split pattern already validated.
- **Inherent-length last.** SQL migrations, validators, and barrels can grow long for legitimate reasons. Don't force these unless they actively hurt.

### Phase 1 — Desktop monoliths (highest leverage, immediate UX impact)

**1.1 `apps/desktop/src/pages/ChatPage.tsx` (3,540 lines)** — Suggested split:

```
apps/desktop/src/pages/ChatPage/
  index.tsx                  # Route component, layout, top-level state wiring
  MessageList.tsx            # Render scrollback (consider virtualization)
  MessageItem.tsx            # Single message rendering
  Composer.tsx               # Input box, attachments, submit
  ToolPanel.tsx              # Tool/capability surface
  ApprovalsPanel.tsx         # Inline approval prompts
  useChatStream.ts           # Streaming/SSE hook
  useChatHistory.ts          # History fetch + pagination
  useChatComposer.ts         # Composer state hook
  types.ts                   # Local types
```

**Expected result:** No single chat-related file >800 lines. Re-render storms localized to MessageItem only.

**1.2 `apps/desktop/src/lib/api.ts` (2,470 lines)** — Suggested split by domain:

```
apps/desktop/src/lib/api/
  index.ts                   # Re-exports
  client.ts                  # fetch wrapper, error envelope, request id
  types.ts                   # Shared request/response types
  models.ts                  # /forge/models/* + model-runtime
  chat.ts                    # /chat/* + assistant
  memory.ts                  # /memory/*
  approvals.ts               # /approvals/*
  audit.ts                   # /audit/*
  jobs.ts                    # /jobs/*
  retrieval.ts               # /retrieval/*
  gateway.ts                 # /gateway/*
  system.ts                  # /system/*, /health/*
  autonomy.ts                # /autonomy/*
  dream.ts                   # /dream/*
  backup.ts                  # /backup/*
  integrations.ts            # /discord/*, /telegram/*
```

**Expected result:** ~15 files of 100-250 lines each. Easy to grep. Easy to mock per-domain in tests.

**1.3 `apps/desktop/src/pages/InspectorsPage.tsx` (2,444 lines)** — Suggested split:

```
apps/desktop/src/pages/InspectorsPage/
  index.tsx                  # Page shell + tab routing
  inspectors/
    MemoryInspector.tsx
    JournalInspector.tsx
    ApprovalsInspector.tsx
    ContextInspector.tsx
    RetrievalInspector.tsx
  useInspectorTab.ts
  types.ts
```

### Phase 2 — Largest Go file

**2.1 `services/core/internal/backup/service.go` (2,005 lines)** — Suggested split:

```
services/core/internal/backup/
  service.go                 # Constructor, top-level API, struct
  export.go                  # Export pipeline
  restore.go                 # Restore pipeline
  scheduler.go               # Scheduled backup logic
  tamper.go                  # Tamper detection
  outcomes.go                # Restore outcomes tracking
  helpers.go                 # Bounded helpers
```

### Phase 3 — Remaining Go monoliths (in priority order)

**3.1 `services/core/internal/modelruntime/service.go` (1,581 lines)** — Split by lifecycle:

```
modelruntime/
  service.go                 # Core struct + constructor
  lifecycle.go               # Load/unload/verify/enable/disable
  selection.go               # Backend selection
  queue.go                   # Request queue
  usage.go                   # Usage/health tracking
  policy.go                  # Lifecycle policy
```

**3.2 `services/core/internal/api/autonomy_maintenance_loop.go` (1,545 lines)** — Split by phase:

```
api/
  autonomy_maintenance_loop.go        # Core loop driver
  autonomy_maintenance_phases.go      # Phase implementations
  autonomy_maintenance_charters.go    # Charter evaluation
  autonomy_maintenance_budgets.go     # Budget enforcement
```

**3.3 `services/core/internal/aios/controllane/compile_context_restore_scoring.go` (1,478 lines)** — Split by scoring concern:

```
controllane/
  compile_context_restore_scoring.go  # Core entry
  restore_scoring_listing.go          # Candidate listing
  restore_scoring_ranking.go          # Ranking
  restore_scoring_threshold.go        # Threshold logic
  restore_scoring_fallback.go         # Fresh-compile fallback
  restore_scoring_persistence.go      # Score/hint persistence
```

**3.4 `services/core/internal/jobs/service.go` (1,452 lines)** — Split by job lifecycle.

**3.5 `services/core/internal/aios/dream/service.go` (1,447 lines)** — Watch first; this file *just* grew. If it stays at 1,447 a week from now, split by dream-mode phase.

### Phase 4 — Pages 1,000-2,000 lines

Use the same `ChatPage` pattern for:
- `ModelsPage.tsx` → `ModelsPage/{index,ModelList,ModelDetail,ImportFlow,RuntimePanel}.tsx`
- `SettingsPage.tsx` → `SettingsPage/{index,General,Models,Approvals,Shell,Network}.tsx`
- `AppShell.tsx` → `AppShell/{index,Sidebar,TopBar,StatusBar,WindowFrame}.tsx` + extract window manager
- `DashboardPage.tsx` → `DashboardPage/{index,Tiles,LiveStream}.tsx`
- `MemoryPage.tsx` → `MemoryPage/{index,NoteList,NoteDetail,Filters}.tsx`

### Phase 5 — Rust + Tauri side

**5.1 `apps/desktop/src-tauri/src/main.rs` (766 lines)** — Split:

```
apps/desktop/src-tauri/src/
  main.rs                    # Tauri builder + setup
  commands/
    operator_apps.rs         # launch_operator_app
    window.rs                # window management
    events.rs                # event emit/listen
  state/
    mod.rs                   # AppState struct
```

### Phase 6 — Defer / Don't split

- `services/core/internal/store/migrate_schema.go` (1,431 lines) — versioned SQL migrations grow legitimately. Split only when a new migration would push it >2,000.
- `services/core/internal/api/phase5.go`, `phase3.go` — historical phase handlers. **Archive instead of split** if their routes are no longer active.
- `crates/forgek-validate/src/lib.rs` (696 lines) — acceptable size for a validator crate.
- `packages/shared/src/index.ts` (799 lines) — barrel re-exports; acceptable.

---

## 3. Working Recipe (per split)

Every split should follow this recipe to keep the tree green between commits:

1. **Identify natural seams** by grepping for top-level functions or component boundaries in the file. Don't invent new seams; find the ones already implicit in the code.
2. **Create sibling files** in the same package/directory. Don't introduce a new package unless the split is large enough to justify one (>5 new files).
3. **Move blocks, don't refactor.** This is a mechanical refactor. Resist the urge to rename, restructure, or "clean up" while splitting. Behavior must be identical.
4. **Run tests after each file extraction.** `go test ./internal/<pkg>/...` or `npm test:desktop` should still pass.
5. **Commit per natural unit.** Don't bundle multiple splits in one commit unless they're trivially small (the way `7894150` bundled the three previous splits is OK; one giant "split everything" commit is not).
6. **No public API changes.** If a function was exported, it stays exported.

---

## 4. Definition of Done (overall)

- **No single source file in the project exceeds 1,500 lines** except where length is inherent (SQL migrations, validator crates, barrel files).
- **No single React component file exceeds 800 lines.**
- **`lib/api.ts` is split by domain** with one file per top-level API group.
- **Old phase handler files** (`phase3.go`, `phase5.go`) are either archived or split.
- **Tests stay green** across every commit.
- **`gateway/service_helpers.go` stays below 1,500 lines** — flag for further split if it grows.

---

## 5. Estimated Effort

| Phase | Files | Effort | Risk |
|---|---|---|---|
| 1 (desktop monoliths) | 3 | 6-10 hours | Low — pure refactor, existing tests |
| 2 (backup) | 1 | 2-3 hours | Low |
| 3 (Go monoliths) | 5 | 1-2 hours each | Low |
| 4 (medium pages) | 5 | 1-2 hours each | Low |
| 5 (Tauri main.rs) | 1 | 2 hours | Low |
| 6 (defer) | — | 0 | — |

**Total to "no file >1,500 lines":** ~20-25 hours of focused refactor work.

---

## 6. Order Summary

1. `ChatPage.tsx`
2. `lib/api.ts`
3. `InspectorsPage.tsx`
4. `backup/service.go`
5. `modelruntime/service.go`
6. `autonomy_maintenance_loop.go`
7. `compile_context_restore_scoring.go`
8. `jobs/service.go`
9. `ModelsPage.tsx`
10. `SettingsPage.tsx`
11. `AppShell.tsx`
12. `DashboardPage.tsx`
13. `MemoryPage.tsx`
14. `dream/service.go` (watch first)
15. `model_runtime_bridge.go`
16. `chat_post.go`
17. `truth/engine.go`
18. `retrieval/service.go`
19. `tauri main.rs`
20. Archive phase3/phase5/etc.

Stop when the largest non-inherent file is under 1,500 lines.
