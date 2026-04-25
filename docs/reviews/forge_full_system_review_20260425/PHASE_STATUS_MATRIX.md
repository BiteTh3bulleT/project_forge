# Phase Status Matrix

| Area | Status | Evidence | Correction |
|---|---|---|---|
| Phase 2 semantic syscall kernel | GOOD | `aios/controllane/processor.go`, `validator.go`, `processor_apply.go`, controllane tests | Implemented, not just documented. |
| Phase 3 cognitive filesystem persistence | PARTIAL | `sqlite_store.go`, `migrate.go`, cognitive tables | Durable core exists; not every operator/API write has syscall-native facade. |
| Phase 4 ingest/librarian cells | PARTIAL | `aios/compute/librarian/pipeline.go`, `cells_phase4.go` | Real, subordinate to kernel, but lane isolation remains conceptual. |
| Phase 5 truth engine | PARTIAL | `aios/truth/engine.go`, tests | Core truth logic exists; UI/API surfacing remains limited. |
| Phase 5.5 rule agents | PARTIAL | `aios/autonomy/rule_agents.go` | Two safe agents; broader rule-agent docs remain aspirational. |
| Phase 5.75 autonomy | PARTIAL | autonomy repos, runner, policy evaluator | Durable and bounded; budget usage for gateway calls needs hardening. |
| Phase 5.9 gateway/tool policy | GOOD | `gateway/service.go`, `tool_policy.go`, registry/tests | Strongest converged authority surface. Approval replay binding gap remains. |
| Phase 6.25 context snapshots/scoring | PARTIAL | `compile_context_snapshot*.go`, `compile_context_restore_scoring.go` | Deterministic scoring exists; SQLite candidate listing over-filters exact query. |
| Phase 6.35 Dream Mode v0 | PARTIAL | `aios/dream/service.go`, `/api/dream/run` | Safe deterministic dry-run; no durable report journal/UI run workflow. |
| Modelruntime M1/M2/M3 | GOOD/PARTIAL | `modelruntime/*`, API bridge/tests | Real and governed for inference; management route approval parity incomplete. |
| CPU/RAM + GPU accelerator split | GOOD | GPU config/docs, modelruntime health/policy | Core can run CPU-only; provider/GPU defaults are conservative. |
| Provider bolt-ons | PARTIAL | TEI, DCGM, Intel, OpenAI-compatible backend | Present but URL policy/discovery persistence need tightening. |
| Runtime authority cutover | PARTIAL | gateway-only adapter route, retired memory mutation | Major side doors removed; operational config/model management still bypass approval parity. |
| Nix foundation | BLOCKED | `flake.nix`, `nix/*` | Present but not verified here; `nix` unavailable. |

## Docs vs Code Mismatches

- RISK: Some status docs still imply no trace UI or legacy adapter UI route; code now has `/api/audit/trace*`, `InspectorsPage`, and desktop adapter invoke uses `/api/gateway/invoke`.
- RISK: Model Runtime M3 baseline file is explicitly retained as historical, but readers may confuse it with current state; current docs should point first to `docs/status/model_runtime_status.md`.
- PARTIAL: `docs/status/placeholders_and_stubs.md` correctly identifies deterministic `COMPILE_CONTEXT` stub warnings, but newer scoring/snapshot reality should be cross-linked more prominently.
- MISSING: No binding file from the user list was missing. A separate `docs/architecture/cognitive_filesystem.md` path referenced by one reviewer is absent; canonical file is `docs/data_model/cognitive_filesystem.md`.

