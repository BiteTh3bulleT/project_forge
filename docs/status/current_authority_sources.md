# Current Authority Sources

Status date: 2026-08-14.

This file maps the current authority docs for FORGE. It is a navigation document, not a new authority path.

## Read Order

1. `AGENTS.md` - agent doctrine, branch/worktree policy, build/test commands, and status guidance.
2. `docs/reviews/current_phase_status.md` - current phase status, including FORGE-K simulator/live authority boundaries.
3. `docs/status/implementation_matrix.md` - live AI-OS and daemon authority implementation matrix.
4. `docs/runbooks/current_forge_bringup.md` - current operator bring-up path.
5. `docs/reports/FORGE_PUNCHLIST.md` - active product/engineering punch list.

## Live Authority Map

| Area | Current source | Authority note |
|---|---|---|
| Canonical semantic syscall ingress | `services/core/internal/forgekernel`, `docs/architecture/forge_k_live_cutover.md` | K20A default live FORGE-K ingress authority. Boot selects exactly one owner and rejects external authority claims. |
| Canonical durable orchestration | `services/core/internal/forgekernel`, `services/core/internal/aios/controllane`, `docs/architecture/forge_k_live_cutover.md` | K20B FORGE-K owns prepare/commit/audit/observe order through `DurablePort`; Control Lane remains the temporary validation/apply/SQLite implementation and rollback facade. |
| Canonical commit integrity | `services/core/internal/forgekernel/commitproof`, production Kernel, Control Lane SQLite durable port | K20D seals exact requests/plans and validates typed receipts. Mutation, journal hash-chain head/provenance, immutable audit intent, and optional idempotency proof share one transaction; verified replay never re-commits and legacy unbound proof fails closed. |
| Courthouse admission/ruling | `services/core/internal/forgekernel/court`, production Kernel, Control Lane SQLite court adapter | K20C FORGE-K computes deterministic admission/rejection and appeal rulings; K20D atomically binds Court mutation/history to the journal hash chain, provenance, immutable audit intent, receipt, and optional idempotency proof. |
| Retrieval evidence admission | production Kernel, `services/core/internal/retrieval`, `services/core/internal/aios/controllane/retrieval_evidence.go`, SQLite durable port | K20G admits one exact-scope immutable retrieval run/results/selection bundle per transaction with provenance, authorization, commitproof, journal, audit outbox, and idempotency evidence. Search/embedding computation remains live-service-owned and non-authoritative until admitted. |
| Utility evidence | production Kernel, `services/core/internal/aios/controllane/utility_evidence*`, retrieval/API producers | K20G appends scope/provenance-bound retrieval usefulness and restore-outcome feedback events atomically with commit proof; original evidence is immutable and projections are separately labeled noncanonical/rebuildable views. |
| Memory acceleration projection | production Kernel, `services/core/internal/memory/vsaprojection`, `services/core/internal/aios/controllane/sqlite_store_acceleration.go` | K20H v2 rebuilds an exact-scope manifest from current Courthouse-admitted immutable memory-evidence leaves and atomically swaps a dedicated evidence-FK projection. Legacy observation projections and stale/superseded heads have zero runtime influence. |
| Tool execution | `services/core/internal/gateway`, `docs/TOOL_GATEWAY.md`, `docs/CAPABILITY_BROKERS.md` | Gateway-only execution authority; legacy adapter invoke ingress is not authority. |
| Model runtime | `services/core/internal/modelruntime`, `services/core/internal/api/model_runtime*.go`, `docs/architecture/model_runtime.md` | Models are governed drivers. Streaming, vLLM-compatible external endpoint support, and managed delete-file approval exist inside modelruntime boundaries. |
| Memory and retrieval | `services/core/internal/memory`, `services/core/internal/retrieval`, `docs/MEMORY_ARCHITECTURE.md`, `docs/RETRIEVAL_PIPELINE.md` | Tool/model output is evidence, not automatic truth. |
| Approvals and audit | `services/core/internal/approvals`, `services/core/internal/audit`, `docs/POLICY_AND_APPROVALS.md`, `docs/AUDIT_AND_TRACE.md` | Approval decisions and audit records remain separated and durable. |
| Jobs and artifacts | `services/core/internal/jobs`, `docs/JOBS_AND_APPROVALS.md`, `docs/TASK_PACKETS.md` | Job streams and task packets are projection/evidence surfaces, not direct truth mutation. |
| Operator bring-up | `docs/runbooks/current_forge_bringup.md`, `docs/runbooks/config_reference.md`, `docs/runbooks/forge_operator_desktop_vm.md` | Runbooks are the operator path for starting and diagnosing current FORGE and the Nix-first operator VM. `npm run test:os-integration` and `npm run validate:os-integration` are the cross-platform static readiness gates before VM rebuilds or boot evidence capture, including local model loop, `/forge` storage-root, safe-mode, and login posture checks. |
| Operator desktop shell | `docs/DESKTOP_SHELL.md`, `docs/status/phase_g8_desktop_shell_verification.md`, `docs/reports/phase_g8_desktop_shell_verification.md`, `docs/runbooks/desktop_shell_operator_smoke_test.md` | FORGE owns the operator-facing shell surface, launcher, taskbar, in-shell windows, native app registry consumption, and bounded native-window requests; labwc remains the compositor substrate, host power is policy-gated and disabled by default through `FORGE_SHELL_DIRECT_SYSTEM_CONTROL`, and G8 adds no FORGE-K live authority. |
| Windows WSL Nix setup | `docs/status/windows_wsl_nix_install_status.md`, `docs/architecture/nix_substrate.md` | Records the local Windows/WSL Nix development install and verification evidence; host setup only, not daemon authority. |

## FORGE-K Boundary

The simulator packages under `services/core/internal/forgek` remain simulator-only. The distinct production package `services/core/internal/forgekernel` owns live semantic syscall ingress by default (K20A), durable stage orchestration through its port contract (K20B), deterministic Courthouse admission/ruling decisions (K20C), sealed commit-integrity verification (K20D), authenticated principal/registry/capability/approval proof verification with proof-bound replay (K20E), K20G retrieval/utility-evidence admission, and K20H Court-derived immutable memory evidence plus governed VSA source authority. K20E also retires live backup row-merge restore: dry inspection remains, while every apply attempt fails closed before approval or mutation. Full FORGE-K authority is still incomplete while search/embedding computation and other subsystem gates remain staged, safe offline recovery is not yet implemented, and Control Lane implements validation/apply/SQLite details.

Current partial live integrations are narrow validation/enforcement seams through shared pure packages and existing live Control Lane paths. They do not:

- route live state mutation through FORGE-K simulator services
- make FORGE-K own gateway, modelruntime, retrieval, embeddings, memory, routes, or APIs
- admit evidence, compile live context, execute semantic operations, or write canonical truth outside existing live authority paths
- enable live KV reuse or runtime cache reuse

Current Courthouse authority is production-owned under `services/core/internal/forgekernel/court`. `ADMIT_EVIDENCE` and `APPEAL_RULING` are decided only by the production Kernel. Current exhibit state, immutable ruling/appeal history, provenance-linked journal hash-chain transition, immutable audit intent, and optional idempotency proof persist atomically and are bound by a Kernel-validated typed receipt. `legacy_v1`, adapter/Future IRIS proposer sources, and model actors fail closed. The earlier `VALIDATE_ADMISSION_CANDIDATE` seam remains validation-only, and the simulator under `services/core/internal/forgek/court` remains non-authoritative. External audit sink delivery and `audit_id` backfill remain post-commit/best-effort projections; they cannot invalidate canonical atomic outbox evidence.

Current retrieval evidence admission is production-owned through `RECORD_RETRIEVAL_EVIDENCE`; this does not make the simulator Memory Palace live. Live search/embedding services still compute candidates, while the production Kernel admits only a deterministic, authenticated, exact-scope run/results/selection bundle with atomic journal/audit/idempotency proof. Existing `forgekshadow` retrieval diagnostics remain bounded metadata-only observers.

Current Context Compiler-related live work is validation/shadow only. `VALIDATE_CONTEXT_ATTRIBUTION` runs through the existing Control Lane and shared pure `services/core/internal/contextattribution` package to validate planned source refs and selection reasons without compiling context. Existing disabled-by-default `services/core/internal/forgekshadow` diagnostics can create a typed shadow ContextBundle shape from accepted `VALIDATE_ADMISSION_CANDIDATE` refs, but those refs are candidate-validation refs rather than live admitted evidence. These surfaces do not replace `COMPILE_CONTEXT`, create prompt text, call modelruntime, run retrieval/search/embeddings, write memory, admit evidence, change routes/APIs, or make `services/core/internal/forgek/contextcompiler` live authority.

Current Runtime Boundary-related live work is proposal-envelope metadata only through existing `services/core/internal/modelruntime` generation results and API bridge translation. Successful modelruntime output carries a typed proposal-only envelope with provenance, audit, output hash/size, token counts, and explicit no-authority flags. It does not admit model output as evidence, commit truth, mutate memory, execute gateway tools, compile context, change backend selection/scheduling, enable live KV reuse, change Control Lane commit behavior, or make `services/core/internal/forgek/runtime` live authority.

Current Consensus Mesh-related live work is a narrow modelruntime-backed final-response guard through `services/core/internal/api` and pure `services/core/internal/consensusgate`. It can withhold unsupported high-risk action claims from model proposal output before assistant message persistence, and records response-composition metadata only. It does not make `services/core/internal/forgek/consensus` live authority, admit evidence, commit truth, mutate memory, execute gateway tools, approve actions, call modelruntime, compile context, or fully gate gateway/Ollama/streaming token surfaces.

Current low-risk Kernel-style commit work includes `CREATE_NOTE`, `UPDATE_STATE`, `OPEN_LOOP`, and `CLOSE_LOOP` through production FORGE-K ingress and existing `services/core/internal/aios/controllane` durable transactions. Notes, state records, state history, and open-loop records persist with journal, audit, provenance, scope, and semantic read-store visibility. This does not make `services/core/internal/forgek` simulator services live authority and does not migrate links, tags, memory observations, gateway execution, modelruntime proposals, or evidence admission.

K20H keeps legacy `POST/PATCH /api/memory/observations*` mutation endpoints as terminal `410 Gone` audit gates, removes their handler references, and seals direct observation, link, usefulness, and repair writers. Existing rows remain readable historical evidence. Repair is deterministic preview/proposal-only. New accepted evidence and immutable revisions belong to production FORGE-K `MATERIALIZE_ADMITTED_EVIDENCE` and `REVISE_MEMORY_EVIDENCE`; there is no model-driven or simulator write path. The actions accept Court object identities rather than caller content, verify the persisted current admitted ruling and equal exhibit/ruling content hash in exact workspace/lane/selected-path scope, and commit immutable `forge_k_memory_evidence` plus append-only supersession lineage with atomic journal/audit/idempotency/authorization proof. Full backup is inspection/export only for these rows; live merge remains forbidden.

Current Lymphatic-related live work is proposal-only metadata on autonomy maintenance dry-run reports through `services/core/internal/api`. Dry-run maintenance and improvement actions are marked as cleanup proposals that cannot execute cleanup and cannot claim commit authority. This does not run `services/core/internal/forgek/lymphatic` as live authority, mutate memory, delete/archive data, execute tools, call modelruntime, admit evidence, or change non-dry-run autonomy ownership.

Current KV-related live work includes a validation-only exact-identity canary through `VALIDATE_KV_IDENTITY` in `services/core/internal/aios/controllane`. The canary requires explicit `kvReuseCanary=true`, `canary_path=control_lane_validation_only`, `STRICT_PREFIX`, matching `final_token_ids_hash`, and all identity gates passing. It records canary eligibility only; it does not enable backend KV tensor reuse, runtime cache reuse, modelruntime behavior changes, memory mutation, or `services/core/internal/forgek/kv` live authority.

Current storage cutover-related work is read-only readiness metadata through `services/core/internal/storagebackend` and `GET /forge/system/status`. SQLite remains the live truth authority and default backend. Postgres is future durable relational infrastructure gated by parity, rollback, read-compare, dual-write comparison, and operator approval evidence. Redis remains ephemeral coordination only, and Qdrant remains vector shadow/acceleration only. This does not enable dual-write, read switching, storage authority migration, Redis canonical truth, Qdrant truth/admissibility, or FORGE-K persistence authority.

Current operator cockpit work is read-only desktop visibility through `apps/desktop/src/pages/SystemPage.tsx` and existing `GET /forge/system/status` plus inspector pointers. It summarizes the boot-selected K20A ingress owner, gates, planned cases, context bundle inspector posture, proposals, journal/replay inspector posture, lymphatic proposal-only posture, subsystem authority matrix rows, and storage cutover readiness. It does not add action controls, new routes, approval execution, cleanup execution, tool execution, storage switching, or additional authority.

Current legacy-retirement work is read-only proof metadata through `GET /forge/system/status`. It records that direct adapter invocation remains unrouted, legacy memory observation mutation remains `410 Gone` and audited, and each retired surface has a default-live replacement and rollback proof. It does not reopen retired routes, execute adapters, write memory, change gateway or Control Lane authority, or make FORGE-K simulator services live authority.

See `docs/reviews/current_phase_status.md` and `docs/adr/0005-forge-k-simulator-vs-live-authority.md`.

## FORGE-K Validation Seam Wiring Matrix

Status date: 2026-05-18 audit.

Seven validation seams are wired in the live durable adapter. All are `[PARTIAL LIVE VALIDATION]`: registered in `services/core/internal/aios/controllane/registry.go`, dispatched from `services/core/internal/aios/controllane/processor.go` after production FORGE-K ingress, and observable via the disabled-by-default `services/core/internal/forgekshadow` observer. `VALIDATE_SOURCE_OBJECT` has a live production caller in `services/core/internal/aios/autonomy/runner.go:preflightSourceObjectAuthority`, which submits a dry-run preflight before any `ARCHIVE_NOTE`, `MARK_SUPERSEDED`, `REGISTER_CONTRADICT` (governed sides only), or `DERIVE_MODEL` commit in `commitAllowedActions`. The candidate-action ingest pipeline calls the other six seams through `services/core/internal/aios/compute/librarian/pipeline.go:processActionValidationSeams` before candidate actions reach the commit path. These calls remain validation preflights and do not grant full FORGE-K subsystem authority.

| Seam | Live handler | Pure pkg | Pure-pkg purity test | Simulator import | Notes |
| --- | --- | --- | --- | --- | --- |
| `VALIDATE_KV_IDENTITY` | `aios/controllane/kv_enforcement.go` | `kvidentity` | yes | `forgek/kv/gates.go` | Only seam genuinely shared with simulator. Live production caller: candidate-action pipeline dry-run preflight when action metadata carries `kvIdentityValidation`. |
| `VALIDATE_REF_SHAPE` | `aios/controllane/ref_validation.go` | `refvalidation` | yes | none | Live production caller: candidate-action pipeline dry-run preflight before candidate commit. See ADR 0015 for simulator unification. |
| `COMPARE_REF_SHAPE` | `aios/controllane/ref_shape_compare.go` | `refvalidation/compare.go` | yes (shared with above) | none | Live production caller: candidate-action pipeline dry-run preflight before candidate commit. |
| `VALIDATE_SOURCE_OBJECT` | `aios/controllane/source_object_authority.go` | none (intentional) | n/a | none | Intrinsically store-dependent; pure portion already shared via `refvalidation`. **Live production callers**: `autonomy/runner.go:preflightSourceObjectAuthority` runs as a dry-run preflight before `ARCHIVE_NOTE` (target note), `MARK_SUPERSEDED` (old + new objects), `REGISTER_CONTRADICT` (governed sides only; skips `journal_event`/`artifact_ref` kinds the resolver cannot look up), and `DERIVE_MODEL` (`derivedFrom` sources). |
| `VALIDATE_SEMANTIC_OPERATION` | `aios/controllane/semantic_operation_validation.go` | `semanticvalidation` | yes | none | Live production caller: candidate-action pipeline dry-run preflight before candidate commit. |
| `VALIDATE_ADMISSION_CANDIDATE` | `aios/controllane/admission_validation.go` | `admissionvalidation` | yes | none | Live production caller: candidate-action pipeline dry-run preflight before candidate commit. |
| `VALIDATE_CONTEXT_ATTRIBUTION` | `aios/controllane/context_attribution_validation.go` | `contextattribution` | yes | none | Live production caller: candidate-action pipeline dry-run preflight before candidate commit. |

Notes on intentional exceptions:

- `VALIDATE_SOURCE_OBJECT` does not have a separate pure validator package. The handler in `services/core/internal/aios/controllane/source_object_authority.go` requires a `SemanticReadStore` to verify that referenced objects exist in canonical state (lines 47, 79-103, 153-194). Ref-shape validation is delegated to `refvalidation`. There is no meaningful pure extraction beyond what already exists.
- Four pure validator packages (`refvalidation`, `semanticvalidation`, `contextattribution`, `admissionvalidation`) are "shared" in the sense of being reusable infrastructure across live Control Lane handlers and the live `forgekshadow` observer. They are not currently imported by the FORGE-K simulator under `services/core/internal/forgek/`. The simulator uses opaque `[]string` source refs, while the pure validators consume structured `refvalidation.ObjectRef`. ADR 0015 proposes the simulator ref-model migration that would make these seams literally shared between simulator and live.
- All pure validator packages forbid imports of `forgek`, `controllane`, `gateway`, `modelruntime`, `retrieval`, `search`, `embeddings`, `memory`, and `api` via per-package `forbidden_imports_test.go`. CI enforces purity.

## Planning And Historical Docs

Roadmaps, archived phase prompts, reviews, and parking-lot notes are evidence and planning context. They are not current authority unless this file, `AGENTS.md`, or `docs/reviews/current_phase_status.md` points to them as current.

Notable planning docs:

- `docs/prompt-packs/FORGE-K_Online_Prompt_Vault/` - inert FORGE-K Online planning/prompt vault; not active authority and not an activated `.cursor` or `AGENTS.md` ruleset.
- `docs/roadmap/forge_ai_os_phases.md`
- `docs/roadmap/forge_k_build_phases.md`
- `docs/roadmap/forge_mutation_loop_parking_lot.md`
- `docs/reports/FORGE_PUNCHLIST.md`
