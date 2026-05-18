# Current Authority Sources

Status date: 2026-05-18.

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
| Canonical mutation | `services/core/internal/aios/controllane`, `docs/architecture/forge_ai_os.md` | Durable semantic writes must pass deterministic validation, commit boundaries, audit, and provenance. |
| Tool execution | `services/core/internal/gateway`, `docs/TOOL_GATEWAY.md`, `docs/CAPABILITY_BROKERS.md` | Gateway-only execution authority; legacy adapter invoke ingress is not authority. |
| Model runtime | `services/core/internal/modelruntime`, `services/core/internal/api/model_runtime*.go`, `docs/architecture/model_runtime.md` | Models are governed drivers. Streaming, vLLM-compatible external endpoint support, and managed delete-file approval exist inside modelruntime boundaries. |
| Memory and retrieval | `services/core/internal/memory`, `services/core/internal/retrieval`, `docs/MEMORY_ARCHITECTURE.md`, `docs/RETRIEVAL_PIPELINE.md` | Tool/model output is evidence, not automatic truth. |
| Approvals and audit | `services/core/internal/approvals`, `services/core/internal/audit`, `docs/POLICY_AND_APPROVALS.md`, `docs/AUDIT_AND_TRACE.md` | Approval decisions and audit records remain separated and durable. |
| Jobs and artifacts | `services/core/internal/jobs`, `docs/JOBS_AND_APPROVALS.md`, `docs/TASK_PACKETS.md` | Job streams and task packets are projection/evidence surfaces, not direct truth mutation. |
| Operator bring-up | `docs/runbooks/current_forge_bringup.md`, `docs/runbooks/config_reference.md`, `docs/runbooks/forge_operator_desktop_vm.md` | Runbooks are the operator path for starting and diagnosing current FORGE and the Nix-first operator VM. |
| Windows WSL Nix setup | `docs/status/windows_wsl_nix_install_status.md`, `docs/architecture/nix_substrate.md` | Records the local Windows/WSL Nix development install and verification evidence; host setup only, not daemon authority. |

## FORGE-K Boundary

FORGE-K remains target architecture and simulator-first implementation. Simulator packages under `services/core/internal/forgek` are not live daemon authority.

Current partial live integrations are narrow validation/enforcement seams through shared pure packages and existing live Control Lane paths. They do not:

- route live state mutation through FORGE-K simulator services
- make FORGE-K own gateway, modelruntime, retrieval, embeddings, memory, routes, or APIs
- admit evidence, compile live context, execute semantic operations, or write canonical truth outside existing live authority paths
- enable live KV reuse or runtime cache reuse

Current Courthouse-related live work is admission-candidate validation only through `VALIDATE_ADMISSION_CANDIDATE` in the existing Control Lane. It does not admit evidence, reject evidence, issue rulings, or make `services/core/internal/forgek/court` live authority.

Current Memory Palace-related live work is mirror-only through existing disabled-by-default `services/core/internal/forgekshadow` retrieval metadata diagnostics. It mirrors bounded metadata refs only and does not run retrieval/search/embeddings, read raw source/chunk/memory content, write memory, admit evidence, compile context, change routes/APIs, or make `services/core/internal/forgek/palace` live authority.

Current Context Compiler-related live work is validation/shadow only. `VALIDATE_CONTEXT_ATTRIBUTION` runs through the existing Control Lane and shared pure `services/core/internal/contextattribution` package to validate planned source refs and selection reasons without compiling context. Existing disabled-by-default `services/core/internal/forgekshadow` diagnostics can create a typed shadow ContextBundle shape from accepted `VALIDATE_ADMISSION_CANDIDATE` refs, but those refs are candidate-validation refs rather than live admitted evidence. These surfaces do not replace `COMPILE_CONTEXT`, create prompt text, call modelruntime, run retrieval/search/embeddings, write memory, admit evidence, change routes/APIs, or make `services/core/internal/forgek/contextcompiler` live authority.

Current Runtime Boundary-related live work is proposal-envelope metadata only through existing `services/core/internal/modelruntime` generation results and API bridge translation. Successful modelruntime output carries a typed proposal-only envelope with provenance, audit, output hash/size, token counts, and explicit no-authority flags. It does not admit model output as evidence, commit truth, mutate memory, execute gateway tools, compile context, change backend selection/scheduling, enable live KV reuse, change Control Lane commit behavior, or make `services/core/internal/forgek/runtime` live authority.

Current Consensus Mesh-related live work is a narrow modelruntime-backed final-response guard through `services/core/internal/api` and pure `services/core/internal/consensusgate`. It can withhold unsupported high-risk action claims from model proposal output before assistant message persistence, and records response-composition metadata only. It does not make `services/core/internal/forgek/consensus` live authority, admit evidence, commit truth, mutate memory, execute gateway tools, approve actions, call modelruntime, compile context, or fully gate gateway/Ollama/streaming token surfaces.

Current low-risk Kernel-style commit work includes `CREATE_NOTE`, `UPDATE_STATE`, `OPEN_LOOP`, and `CLOSE_LOOP` through existing `services/core/internal/aios/controllane` syscall transactions. Notes, state records, state history, and open-loop records persist with journal, audit, provenance, scope, and semantic read-store visibility. This does not make `services/core/internal/forgek` live Kernel authority and does not migrate links, tags, memory observations, gateway execution, modelruntime proposals, or evidence admission.

Current memory observation migration work keeps legacy `POST/PATCH /api/memory/observations*` mutation endpoints retired through `services/core/internal/api`. Existing `memory_observations` rows remain historical/retrieval evidence, and retired write attempts receive structured guidance/audit metadata pointing to Courthouse admission-candidate validation plus Control Lane semantic syscalls. This does not admit evidence, write memory, run a batch migrator, or make `services/core/internal/forgek` live authority.

Current Lymphatic-related live work is proposal-only metadata on autonomy maintenance dry-run reports through `services/core/internal/api`. Dry-run maintenance and improvement actions are marked as cleanup proposals that cannot execute cleanup and cannot claim commit authority. This does not run `services/core/internal/forgek/lymphatic` as live authority, mutate memory, delete/archive data, execute tools, call modelruntime, admit evidence, or change non-dry-run autonomy ownership.

Current KV-related live work includes a validation-only exact-identity canary through `VALIDATE_KV_IDENTITY` in `services/core/internal/aios/controllane`. The canary requires explicit `kvReuseCanary=true`, `canary_path=control_lane_validation_only`, `STRICT_PREFIX`, matching `final_token_ids_hash`, and all identity gates passing. It records canary eligibility only; it does not enable backend KV tensor reuse, runtime cache reuse, modelruntime behavior changes, memory mutation, or `services/core/internal/forgek/kv` live authority.

Current storage cutover-related work is read-only readiness metadata through `services/core/internal/storagebackend` and `GET /forge/system/status`. SQLite remains the live truth authority and default backend. Postgres is future durable relational infrastructure gated by parity, rollback, read-compare, dual-write comparison, and operator approval evidence. Redis remains ephemeral coordination only, and Qdrant remains vector shadow/acceleration only. This does not enable dual-write, read switching, storage authority migration, Redis canonical truth, Qdrant truth/admissibility, or FORGE-K persistence authority.

Current operator cockpit work is read-only desktop visibility through `apps/desktop/src/pages/SystemPage.tsx` and existing `GET /forge/system/status` plus inspector pointers. It summarizes gates, planned cases, context bundle inspector posture, proposals, journal/replay inspector posture, lymphatic proposal-only posture, subsystem authority matrix rows, and storage cutover readiness. It does not add action controls, new routes, approval execution, cleanup execution, tool execution, storage switching, or FORGE-K live authority.

Current legacy-retirement work is read-only proof metadata through `GET /forge/system/status`. It records that direct adapter invocation remains unrouted, legacy memory observation mutation remains `410 Gone` and audited, and each retired surface has a default-live replacement and rollback proof. It does not reopen retired routes, execute adapters, write memory, change gateway or Control Lane authority, or make FORGE-K simulator services live authority.

See `docs/reviews/current_phase_status.md` and `docs/adr/0005-forge-k-simulator-vs-live-authority.md`.

## Planning And Historical Docs

Roadmaps, archived phase prompts, reviews, and parking-lot notes are evidence and planning context. They are not current authority unless this file, `AGENTS.md`, or `docs/reviews/current_phase_status.md` points to them as current.

Notable planning docs:

- `docs/prompt-packs/FORGE-K_Online_Prompt_Vault/` - inert FORGE-K Online planning/prompt vault; not active authority and not an activated `.cursor` or `AGENTS.md` ruleset.
- `docs/roadmap/forge_ai_os_phases.md`
- `docs/roadmap/forge_k_build_phases.md`
- `docs/roadmap/forge_mutation_loop_parking_lot.md`
- `docs/reports/FORGE_PUNCHLIST.md`
