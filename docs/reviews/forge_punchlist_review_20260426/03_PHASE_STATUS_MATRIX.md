# Phase Status Matrix

| Phase / Pass | Status | Classification | Evidence | Remaining Gate |
|---|---|---|---|---|
| Phase 1 foundation | complete | accurate | `AGENTS.md`, architecture docs | Keep docs/current code aligned. |
| Phase 2 semantic syscall kernel | mostly complete | accurate | `aios/controllane/processor.go`, `validator.go`, tests | Public syscall API and more edge tests. |
| Phase 3 cognitive filesystem | partial / mostly complete | slightly understated in older docs | `store/migrate.go`, `sqlite_store.go` | Immutability decisions, backup hard gates. |
| Phase 4 ingest/librarian cells | mostly complete | accurate | `aios/compute/librarian/*` | Lane isolation and operator inspection. |
| Phase 5 truth engine | partial | accurate | `aios/truth/engine.go` | Repair/apply/operator surfacing. |
| Phase 5.5 rule agents | partial | accurate | `aios/autonomy/rule_agents.go` | Expand deterministic coverage or fold into Rule Cells. |
| Phase 5.75 autonomy | partial implemented | accurate | `aios/autonomy/*`, SQLite repos | Tool-call budget consumption and trace visibility. |
| Phase 5.9 gateway/tool policy | mostly complete | accurate | `gateway/*`, capability registry | Capability status governance and service-specific tests. |
| Phase 5.95 runtime authority cutover | mostly complete | accurate | retired legacy adapter route, gateway ingress | Operator trace cohesion. |
| Phase 5.997 convergence | mostly complete | stale in older review docs | current code includes recent passes | Update stale reviews/status docs. |
| Phase 6.25 context restore | mostly complete | overstated in candidate retrieval docs | scoring exists, exact-query SQL prefilter remains | Near-match candidate retrieval and indexes. |
| Phase 6.35 Dream Mode v0 | partial implemented | older docs stale | `dream_reports`, inspector API, backup coverage now exist | Operator apply/review not implemented by design. |
| Approval fingerprint hardening | implemented | implemented but older docs stale | gateway/approval/modelruntime tests | Shared permission semantics still need hardening. |
| Model management governance | implemented / partial | implemented but M4 remains | `api/model_runtime_governance.go`, tests | Cost/budget/provider policy, streaming, supervision. |
| Backup/restore parity | mostly implemented | accurate with caveats | `backup/service.go`, tests | Hard checksum/count fail-closed preflight. |
| Context restore scoring fix | partial | documented stronger than code | `compile_context_restore_scoring.go` | SQL candidate retrieval. |
| Dream report persistence | implemented | implemented but older docs stale | `dream_reports`, API tests | Same-ID upsert/append-only decision. |
| Operator restore/Dream inspector | implemented / partial | current | `operator_inspector.go`, `InspectorsPage.tsx` | Global diagnostics and UI state polish. |
| Rule Cell / Hyperlane v0 | concept/scaffold | planned only | journal docs | First deterministic router/rule registry pass. |
| Modelruntime M1 | mostly complete | accurate | manifest/store/registry/backends | Keep under governance. |
| Modelruntime M2 | mostly complete | accurate | scheduler, queue, usage | Durable scheduler/timeout behavior. |
| Modelruntime M3 | partial implemented | accurate | management APIs, governance tests | M4 items. |
| Modelruntime M3.5 | partial | accurate | DCGM, Level Zero, TEI diagnostics | Provider costs, LoRA, embedding refresh governance. |
| Modelruntime M4 | not implemented | accurate | no streaming/process supervision | Dedicated implementation passes. |
| Nix N1 | partial | not verified | `flake.nix`, docs | Run `nix flake check` where Nix daemon exists. |

