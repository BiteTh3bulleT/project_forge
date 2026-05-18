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

## FORGE-K Boundary

FORGE-K remains target architecture and simulator-first implementation. Simulator packages under `services/core/internal/forgek` are not live daemon authority.

Current partial live integrations are narrow validation/enforcement seams through shared pure packages and existing live Control Lane paths. They do not:

- route live state mutation through FORGE-K simulator services
- make FORGE-K own gateway, modelruntime, retrieval, embeddings, memory, routes, or APIs
- admit evidence, compile live context, execute semantic operations, or write canonical truth outside existing live authority paths
- enable live KV reuse or runtime cache reuse

See `docs/reviews/current_phase_status.md` and `docs/adr/0005-forge-k-simulator-vs-live-authority.md`.

## Planning And Historical Docs

Roadmaps, archived phase prompts, reviews, and parking-lot notes are evidence and planning context. They are not current authority unless this file, `AGENTS.md`, or `docs/reviews/current_phase_status.md` points to them as current.

Notable planning docs:

- `docs/prompt-packs/FORGE-K_Online_Prompt_Vault/` - inert FORGE-K Online planning/prompt vault; not active authority and not an activated `.cursor` or `AGENTS.md` ruleset.
- `docs/roadmap/forge_ai_os_phases.md`
- `docs/roadmap/forge_k_build_phases.md`
- `docs/roadmap/forge_mutation_loop_parking_lot.md`
- `docs/reports/FORGE_PUNCHLIST.md`
