# FORGE AI-OS Phases (Reality-Based Status)

Status date: 2026-04-23 (branch-local reality snapshot).

Allowed statuses: `complete`, `mostly complete`, `partial`, `blocked`, `scaffold`, `deferred`.

| Phase | Status | What is real now | Main remaining gate |
|---|---|---|---|
| Phase 1 (AI-OS foundation) | complete | doctrine + baseline architecture are established | keep docs/code aligned |
| Phase 2 (semantic syscall kernel) | mostly complete | deterministic registry/validator/processor/audit path | broaden edge-case/API coverage |
| Phase 3 (cognitive filesystem persistence) | partial | durable core cognitive tables and history model | close remaining durability/restore gaps |
| Phase 4 (ingest + librarian cells) | mostly complete | proposal-first cells wired through syscall commits | strengthen negative-path/quality guards |
| Phase 5 (truth engine) | partial | current/history/contradiction/supersession services are real and kernel-backed; observation mutation routes are retired | finish repair/explain/operator surfacing and clarify event projection semantics |
| Phase 5.5 (rule agents) | partial | safe propose-only agents with destructive guards | expand deterministic agent coverage |
| Phase 5.75 (autonomy layer) | partial | durable default repos + policy/budget/approval gates plus bounded maintenance loop are real | broaden trace visibility, review default mode posture, and continue parity work |
| Phase 5.9 (tool surface/capability policy) | mostly complete | governed taxonomy + policy + audit path; default capabilities resolve to gateway-backed `active` or `approval_only` tools | keep expanding service-specific harness tests |
| Phase 6.25 (context restore snapshots) | mostly complete | `COMPILE_CONTEXT` can persist snapshot evidence, render optional SVG cards, apply deterministic full restore scoring, return header-first restore packages, emit resume hints, and mark `requires_fresh_compile`; restore rows stay syscall-bound, scope-linked, and non-canonical | broaden operator-facing restore inspection and continue scalability work |
| Phase 6.35 (Dream Mode v0) | partial | deterministic CPU-only dry-run replay selector, salience scoring, memory tier routing proposals, and `/api/dream/run` report path are implemented without modelruntime/GPU dependency or canonical commits | add operator UI/report persistence only as non-canonical evidence; future commit mode must use semantic syscalls |
| Phase 7 (Rule Cells / Hyperlane v0) | partial | deterministic static Rule Cell packs and CPU-only Hyperlane routers exist; restore scoring and Dream Mode consume bounded advisory/stricter outputs with pack-versioned traces and authority-conflict protection | expand operator/runtime wiring carefully without loosening kernel, gateway, approval, capability, scope, or modelruntime authority |
| Phase 8 (restore outcome feedback loop) | partial | `restore_outcome_events` records non-canonical restore utility evidence; persistent `COMPILE_CONTEXT` emits initial outcomes, operator feedback API updates scoped outcome evidence, restore scoring consumes bounded utility signals, Dream Mode replays outcomes for memory-gap/review/promotion proposals, and backup/restore includes the table | build operator inspection UI later; do not add Dream apply/commit or adapter training without syscall authority |
| Phase 8.1 (chat latency/efficiency) | partial | chat no-model routing, context/output budget classes, runtime queue/cooldown preflight, restore scoring cache, header-first restore posture, and latency trace fields are wired without requiring modelruntime/GPU | broaden structured no-model answer coverage and operator latency dashboards |
| Phase M1 (model runtime foundation) | mostly complete | FORGE-native modelruntime subsystem is live (manifest/store/registry/backends/runtime service/internal API plus gated OpenAI-compatible minimum API) | keep M1 truth aligned under later governance work |
| Phase M2 (model runtime governance) | mostly complete | FIFO scheduler, bounded admission, lifecycle controls, policy/workspace hooks, richer audit/usage accounting, runtime queue/loaded endpoints | M3 management/backend expansion now landed; streaming and deeper governance surfacing remain |
| Phase M3 (model runtime management) | partial (implemented) | import/register/reconcile flows, persistent lifecycle state, enable/disable/archive/remove-registration operations, OpenAI-compatible backend, vLLM-compatible path, compatibility/usage/backend inspection, deterministic selection, policy-visible gateway `model.*` aliases | broader runtime work remains: streaming, delete-file approval flow, stronger backend/process supervision, deeper scheduling/load balancing |
| Phase M4 (vLLM external runtime profile) | mostly complete | canonical `FORGE_VLLM_*` config, legacy alias support, governed disabled-by-default vLLM endpoint profile, modelruntime backend status metadata, Nix/Rust/vLLM boundary docs | future managed vLLM NixOS service and deeper GPU scheduling require later proposal/governance work |
| Phase M5 (workstation substrate hardening) | partial | design docs for workstation substrate, governed Nix mutation proposals, backend profiles, FORGE-H VRAM/CUDA lane, safe-mode recovery, and system cockpit | no host mutation implementation yet; next phase should wire one read-only/proposal surface at a time |
| Phase M3.5 (provider bolt-ons) | partial | optional NVIDIA DCGM telemetry, Intel Level Zero telemetry, and Hugging Face TEI embeddings are wired as disabled-by-default providers with health/capability diagnostics and no truth authority | next: vLLM LoRA registry, provider cost telemetry, and governed embedding refresh scheduling |
| Phase 5.95 (runtime authority cutover) | mostly complete | authoritative mutation/tool paths are clear; legacy memory mutation endpoints are retired | improve operator trace visibility and event projection clarity |
| Phase 5.997 (current pass) | mostly complete | convergence now includes gateway-only adapter execution ingress, retired memory mutation boundaries, deterministic restore candidate scoring metadata, and clearer authority/runtime docs | keep event projection language aligned and improve operator trace visibility |
| Phase N1 (light Nix foundation) | partial | flake/shell/check scaffolding present | authoritative validation blocked by local nix daemon availability |

## Before Phase 6 must be true

1. Fresh-clone core build is reproducible without manual VSA file intervention.
2. Backup/export claims match restore reality for declared critical sections (including explicit VSA export-only posture).
3. Retired side doors remain non-executable and documented.
4. JS/TS validation has at least build+typecheck+lint/test baseline.
5. Traceability chain is inspectable for gateway/syscall/autonomy/artifact flows.
6. Model runtime authority exists as a FORGE-owned subsystem (not only adapter-level Ollama coupling).
7. Phase 5 truth/autonomy/tool-policy claims stay aligned with code rather than with target architecture language.

## Runtime / Workstation Preview

1. Stronger backend expansion and process supervision.
2. Optional embeddings/rerank runtime paths if they become worth governing natively.
3. Deeper governance and autonomy-policy hooks without creating a side door.
4. Delete-file approval flow separated cleanly from remove-registration.
5. Governed Nix mutation proposals require review, build proof, rollback proof, and a future host adapter.
