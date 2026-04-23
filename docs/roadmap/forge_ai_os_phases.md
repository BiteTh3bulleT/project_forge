# FORGE AI-OS Phases (Reality-Based Status)

Status date: 2026-04-22 (branch-local reality snapshot).

Allowed statuses: `complete`, `mostly complete`, `partial`, `blocked`, `scaffold`, `deferred`.

| Phase | Status | What is real now | Main remaining gate |
|---|---|---|---|
| Phase 1 (AI-OS foundation) | complete | doctrine + baseline architecture are established | keep docs/code aligned |
| Phase 2 (semantic syscall kernel) | mostly complete | deterministic registry/validator/processor/audit path | broaden edge-case/API coverage |
| Phase 3 (cognitive filesystem persistence) | partial | durable core cognitive tables and history model | close remaining durability/restore gaps |
| Phase 4 (ingest + librarian cells) | mostly complete | proposal-first cells wired through syscall commits | strengthen negative-path/quality guards |
| Phase 5 (truth engine) | mostly complete | current/history/contradiction/supersession services are real | improve repair/explain completeness |
| Phase 5.5 (rule agents) | partial | safe propose-only agents with destructive guards | expand deterministic agent coverage |
| Phase 5.75 (autonomy layer) | partial | durable default repos + policy/budget/approval gates | broaden trace visibility + remaining parity |
| Phase 5.9 (tool surface/capability policy) | partial | governed taxonomy + policy + audit path | continue hardening dangerous/default posture |
| Phase 6.25 (context restore snapshots) | scaffold | `COMPILE_CONTEXT` extension that can persist snapshot evidence and optionally render an SVG card; restore rows stay syscall-bound, scope-linked, and non-canonical | Phase 6.5 adds restore scoring/ranking on top of persisted snapshots |
| Phase M1 (model runtime foundation) | mostly complete | FORGE-native modelruntime subsystem is live (manifest/store/registry/backends/runtime service/internal API plus gated OpenAI-compatible minimum API) | keep M1 truth aligned under later governance work |
| Phase M2 (model runtime governance) | mostly complete | FIFO scheduler, bounded admission, lifecycle controls, policy/workspace hooks, richer audit/usage accounting, runtime queue/loaded endpoints | M3 management/backend expansion now landed; gateway `model.*` aliasing and streaming remain |
| Phase M3 (model runtime management) | partial (implemented) | import/register/reconcile flows, persistent lifecycle state, enable/disable/archive/remove-registration operations, OpenAI-compatible backend, vLLM-compatible path, compatibility/usage/backend inspection, deterministic selection | M4: streaming, delete-file approval flow, stronger backend/process supervision, deeper scheduling/load balancing, gateway `model.*` aliasing |
| Phase 5.95 (v1/v2 cutover) | partial | authoritative paths clearer; legacy boundaries explicitly gated | continue reducing side doors |
| Phase 5.996 (current pass) | partial (improved) | restore parity includes project context/evaluations/audit/gateway sections; restore reports `atomicScope` + non-DB warnings; legacy boundary request/audit hardening improved | resolve authoritative VSA source tracking + VSA export-only restore posture + legacy adapter side-door convergence |
| Phase N1 (light Nix foundation) | partial | flake/shell/check scaffolding present | authoritative validation blocked by local nix daemon availability |

## Before Phase 6 must be true

1. Fresh-clone core build is reproducible without manual VSA file intervention.
2. Backup/export claims match restore reality for declared critical sections (including explicit VSA export-only posture).
3. Legacy side doors are either removed or tightly bounded and documented.
4. JS/TS validation has at least build+typecheck+lint/test baseline.
5. Traceability chain is inspectable for gateway/syscall/autonomy/artifact flows.
6. Model runtime authority exists as a FORGE-owned subsystem (not only adapter-level Ollama coupling).

## M4 Preview

1. Stronger backend expansion and process supervision.
2. Optional embeddings/rerank runtime paths if they become worth governing natively.
3. Deeper governance and autonomy-policy hooks without creating a side door.
4. Delete-file approval flow separated cleanly from remove-registration.
5. Nix packaging remains later work, not an M4 dependency.
