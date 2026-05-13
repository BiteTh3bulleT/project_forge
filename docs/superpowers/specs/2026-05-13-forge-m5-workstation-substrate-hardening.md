# FORGE Stack Hardening Prompt: NixOS Substrate, Workstation Target, FORGE-H VRAM Lane, and Governed Runtime Profiles

You are working inside the `BiteTh3bulleT/project_forge` repository.

Your job is to harden the current FORGE stack into a more complete **FORGE Workstation substrate** while preserving all existing authority boundaries.

FORGE is **Foundry for Organic Reasoning, Growth, and Execution**.

Tagline:

> Turn chaos into cognition. Turn cognition into action.

## Mission

Implement a production-grade, safety-first architecture pass that turns the current NixOS/Tauri/modelruntime/FORGE-H/FORGE-K direction into a concrete staged foundation.

The goal is **not** to make FORGE autonomous over the host.

The goal is to add the missing scaffolding for:

1. governed Nix mutation proposals,
2. a FORGE Workstation flake/substrate target,
3. vLLM/modelruntime backend profile support,
4. FORGE-H VRAM/CUDA lane design,
5. safe-mode/recovery substrate profiles,
6. Tauri System cockpit expansion planning,
7. build/test/rollback promotion doctrine.

Do not dump code in chat. Make changes in files. At the end, summarize changed files, tests run, and remaining gaps.

---

# Current Doctrine To Preserve

Before editing anything, read these files:

- `README.md`
- `docs/reviews/current_phase_status.md`
- `docs/architecture/forge_os_host_substrate.md`
- `docs/architecture/forge_graphical_shell.md`
- `docs/architecture/forge_h_resource_policy.md`
- `docs/architecture/cpu_ram_kernel_gpu_accelerator_split.md`
- `docs/architecture/forge_k_overview.md`
- `docs/runbooks/config_reference.md`
- `docs/adr/0005-forge-k-simulator-vs-live-authority.md`
- `docs/adr/0007-forge-os-host-substrate.md`
- `docs/adr/0011-forge-graphical-shell-session.md`
- `docs/adr/0012-forge-wayland-shell-session.md`

Preserve these principles:

- Linux remains the real hardware/kernel substrate.
- NixOS remains the declarative host substrate.
- FORGE is the AI-OS operating environment above the substrate.
- FORGE-K is target cognitive microkernel architecture, not live daemon authority unless explicitly promoted by later governed phases.
- FORGE-H provides resource judgment and bounded internal operational policy, not raw host mutation.
- GPU/CUDA/VRAM paths are acceleration surfaces, not canonical truth.
- `forge-core` remains CPU/RAM authoritative.
- modelruntime governs model execution.
- Tauri shell is an operator surface, not an authority plane.
- SQLite remains the default live canonical store unless a later explicit migration proves otherwise.
- Qdrant is vector recall/shadow infrastructure, not truth.
- Redis is ephemeral coordination, not canonical memory.
- Models are drivers, not authority.
- Kernel commit/journal path is the only canonical truth path.

---

# High-Level Target Architecture

Design toward this stack:

```text
Hardware
  ↓
Linux kernel
  ↓
NixOS declarative substrate
  ↓
FORGE Workstation substrate profile
  ↓
forge-core
  ↓
FORGE-H resource policy / future VRAM registry
  ↓
modelruntime backend profiles
  ↓
Tauri graphical shell / System cockpit
  ↓
operator approvals, audit, rollback, journal evidence

Future CUDA/VRAM lane shape:

FORGE-H
  ├── RAM pressure policy
  ├── disk/swap/thermal policy
  ├── VRAM pressure policy
  ├── future VRAM registry
  ├── future CUDA capability gateway
  └── future approved kernel/runtime worker lanes

CUDA/VRAM may accelerate:

inference,
embedding,
reranking,
vector scoring,
KV/cache analysis,
simulation,
batch diagnostics,
compression/prepass work.

CUDA/VRAM must not:

commit canonical truth,
bypass modelruntime,
bypass gateway,
bypass approval,
write semantic memory directly,
mutate host state,
run unapproved kernels,
expose raw GPU pointers to agents or UI.
Phase 0 — Repository Reality Check

Inspect the current repo before modifying.

Create or update a review document:

docs/reviews/forge_workstation_stack_gap_review.md

It must include:

current implemented substrate pieces,
current Tauri/Nix shell state,
current modelruntime backend support,
current GPU/safe-mode policy,
current FORGE-H advisory capabilities,
current FORGE-K simulator/live authority boundary,
missing pieces for the desired FORGE Workstation substrate,
exact files likely needing future implementation.

Do not invent implemented features. If something is only planned, mark it clearly as PLANNED, DESIGN_ONLY, DISABLED_BY_DEFAULT, ADVISORY_ONLY, or FUTURE.

Phase 1 — Add FORGE Workstation Substrate Architecture Doc

Create:

docs/architecture/forge_workstation_substrate.md

This document must define:

Purpose

FORGE Workstation is a private, local-first FORGE host profile built on NixOS, Tauri, forge-core, modelruntime, FORGE-H, HostBridge diagnostics, and governed storage/runtime infrastructure.

Stack

Include a table:

Layer	Responsibility	Authority Boundary

Required layers:

Hardware
Linux kernel
NixOS
FORGE Workstation profile
forge-core
HostBridge
FORGE-H
modelruntime
storage/vector/ephemeral services
Tauri graphical shell
operator
Non-Goals

Explicitly state:

FORGE does not fork Linux.
FORGE does not replace the kernel.
FORGE does not auto-run nixos-rebuild.
FORGE does not mutate host config without operator-approved proposal flow.
FORGE-K does not become live authority in this phase.
GPU does not become truth authority.
shell UI does not run host commands directly.
Workstation Composition

Define intended opt-in components:

/forge durable root,
forge-core,
Tauri shell,
Wayland shell session path,
Postgres optional,
Qdrant optional,
Redis optional,
modelruntime backend profile,
GPU telemetry optional,
safe-mode/recovery profile,
backup/export directory expectations.
Promotion Doctrine

Document:

DESIGN_ONLY
  → SIMULATOR_ONLY
  → SHADOW_READ_ONLY
  → DISABLED_BY_DEFAULT_LIVE
  → OPERATOR_APPROVED_LIVE
  → AUTOMATED_SAFE_ACTION

Nothing jumps stages.

Phase 2 — Add Governed Nix Mutation Proposal Design

Create:

docs/architecture/nix_mutation_proposals.md

This must design a future governed pipeline for NixOS/flake changes.

Define object:

NixMutationProposal

Required fields:

proposal_id
created_at
proposer
reason
target_scope
affected_files
proposed_diff_ref
generated_by_model
requires_operator_approval
risk_level
expected_services_changed
expected_resource_impact
build_command
build_result_ref
vm_smoke_test_result_ref
rollback_plan_ref
status
audit_ref
journal_ref
expires_at

Status values:

proposed
rejected
approved_for_build
build_failed
build_passed
approved_for_vm_test
vm_test_failed
vm_test_passed
approved_for_apply
applied
rolled_back
superseded
expired

Define the intended flow:

Generate proposal
  → operator review
  → approve build
  → sandbox build
  → approve VM smoke test
  → VM smoke test
  → rollback plan
  → approve apply
  → apply through governed host adapter
  → journal result
  → monitor outcome

Hard rule:

No direct nixos-rebuild from UI, model, shell, FORGE-K simulator, or ungoverned tool path.

This phase is design-only unless existing repo structure already has a clean place for pure data models/tests without wiring host execution.

Phase 3 — Add/Update ADR

Create:

docs/adr/0013-forge-workstation-substrate-and-nix-mutation-proposals.md

ADR must include:

context,
decision,
consequences,
accepted boundaries,
rejected alternatives,
rollback posture,
future work.

Accepted decision:

FORGE may generate governed Nix mutation proposals, but may not apply host mutations without explicit operator approval, build proof, rollback proof, and later controlled adapter implementation.

Rejected alternatives:

AI directly edits host config and rebuilds.
Tauri shell directly invokes system commands.
FORGE-K simulator becomes live authority.
GPU/CUDA lane owns canonical truth.
Qdrant or Redis becomes canonical memory.
Phase 4 — Modelruntime Backend Profiles Design

Create or update:

docs/architecture/modelruntime_backend_profiles.md

Design backend profiles:

cpu_safe
local_llama_cpp
local_ollama_dev
interactive_vllm
embedding_tei
openai_compatible_remote

For each profile, define:

purpose,
expected endpoint/env vars,
GPU needs,
VRAM posture,
concurrency posture,
allowed workload classes,
safe-mode behavior,
failure behavior,
FORGE-H recommendation inputs.

Important:

vLLM is a preferred high-throughput backend profile, not the only modelruntime foundation.

Do not remove llama.cpp, Ollama, OpenAI-compatible, or TEI compatibility.

Phase 5 — FORGE-H VRAM Registry and CUDA Lane Design

Create:

docs/architecture/forgeh_vram_cuda_lane.md

This must define a future path for FORGE-H to govern VRAM/CUDA acceleration.

Design objects:

VramRegion
VramLease
CudaBufferRef
CudaKernelArtifact
CudaKernelLaunchProposal
GpuWorkClass
GpuMemoryPressureEvent
CudaBackendProfile
KvCacheResidencyPolicy

Define first implementation ladder:

Observe VRAM pressure.
Report GPU/VRAM posture.
Recommend scheduling decisions.
Create approved resource proposals.
Record bounded internal policy changes.
Future: allocate governed VRAM regions.
Future: launch approved kernels.
Future: cuda-oxide experimental backend.
Future: CUDA VMM/IPC advanced mode.

Authority rule:

CUDA may accelerate cognition.
CUDA may not author truth.

Explicitly forbid:

raw pointer exposure to agents,
arbitrary VRAM scanning,
direct GPU memory mutation outside registered owned buffers,
public mutation routes,
unapproved kernel launch,
modelruntime bypass,
memory/journal bypass,
FORGE-K live authority expansion.

Mark this phase as design-only unless implementing pure constants/types/tests is obviously safe and isolated.

Phase 6 — Safe Mode / Recovery NixOS Specialization Design

Create:

docs/architecture/forge_safe_mode_recovery_profiles.md

Design profiles:

FORGE Normal
FORGE Safe Mode
FORGE CPU Only
FORGE GPU Runtime
FORGE Debug Shell
FORGE Recovery

For each profile, define:

enabled services,
disabled services,
modelruntime posture,
GPU posture,
shell posture,
network posture,
rollback method,
operator command path.

Must preserve:

TTY fallback,
non-FORGE desktop fallback,
no autologin by default,
no default desktop replacement,
no destructive cleanup.
Phase 7 — Tauri System Cockpit Expansion Plan

Create:

docs/architecture/forge_system_cockpit.md

This should plan the next System page evolution.

Cockpit panels:

Core status
FORGE-K activation readiness
Authority gate matrix
FORGE-H resource posture
HostBridge diagnostics summary
modelruntime backend profile
GPU/VRAM posture
storage posture
Postgres/Qdrant/Redis readiness
Nix generation/rollback status
mutation proposal queue
approval queue
safe-mode status
recent warnings
last test/build status

Rules:

read-only by default,
no restart/shutdown/rebuild buttons,
no model load/unload buttons unless later governed proposal flow exists,
no raw logs by default,
no raw memory dumps,
no direct shell commands,
no approval execution unless routed through existing approval/gateway paths.
Phase 8 — Optional Light Scaffolding Only If Safe

After docs are complete, inspect current code structure.

If there is a clean, low-risk place to add pure model constants/types with tests, you may add non-wired, non-executing scaffolding for one or more of:

Nix mutation proposal status enum,
modelruntime backend profile labels,
FORGE-H VRAM pressure classification labels,
CUDA lane future work class labels.

Strict limits:

no public route,
no shell mutation control,
no host command execution,
no nixos-rebuild,
no systemctl,
no model load/unload behavior change,
no gateway behavior change,
no FORGE-K live authority,
no storage default switch,
no Qdrant/Redis live authority,
no response behavior change.

If safe scaffolding would require broad wiring, skip it and document why.

Phase 9 — Update Existing Index Docs

Update only where appropriate:

README.md
docs/glossary.md
docs/roadmap/forge_k_build_phases.md
docs/reviews/current_phase_status.md
any relevant architecture index/listing docs.

All updates must be honest about phase status.

Use labels:

DESIGN_ONLY
PLANNED
ADVISORY_ONLY
DISABLED_BY_DEFAULT
READ_ONLY
FUTURE
NO_LIVE_AUTHORITY_CHANGE

Do not claim implementation that does not exist.

Validation Requirements

Run the relevant available tests.

At minimum, attempt:

npm test
npm run build
npm run build:core
npm run build:desktop

If those are too broad or fail due to known environment issues, run narrower tests and document why.

Also inspect docs links if there is an existing docs lint/check command.

Final response must include:

files created,
files modified,
tests run,
tests passed/failed,
known blockers,
next recommended implementation phase.

Do not paste code in chat.

WHAT NOT TO DO

Do not output code in chat.

Do not rewrite the project identity.

Do not change the FORGE acronym.

Do not rename FORGE-K, FORGE-H, Memory Palace, Courthouse, Control Lane, or modelruntime.

Do not make FORGE-K live authority.

Do not let FORGE-H mutate the host.

Do not add direct systemctl, nixos-rebuild, package manager, kernel/module, reboot, shutdown, or destructive cleanup execution paths.

Do not add public mutation APIs.

Do not add shell buttons that perform host mutation.

Do not bypass gateway, permissions, lanes, approvals, audit, Control Lane validation, semantic memory authority, modelruntime governance, or existing safety flags.

Do not make GPU/CUDA/VRAM required for canonical truth.

Do not treat KV cache as memory.

Do not treat Qdrant as truth.

Do not treat Redis as canonical state.

Do not make vLLM the only modelruntime backend.

Do not capture raw prompts, completions, request bodies, response bodies, secrets, auth headers, cookies, raw logs, raw memory, raw vectors, or raw host diagnostics in new diagnostics.

Do not mark design-only docs as implemented.

Do not delete existing safe fallbacks.

Do not remove normal desktop or TTY fallback.

Do not enable autologin.

Do not silently change default ports, storage backends, modelruntime defaults, or GPU requirements.

Do not make broad unrelated refactors.

Desired End State

At the end of this pass, the repo should have a clear, production-grade architecture foundation for:

FORGE Workstation substrate
Governed Nix mutation proposals
Modelruntime backend profiles
FORGE-H VRAM/CUDA lane future path
Safe-mode/recovery host profiles
Read-only System cockpit expansion
Build/test/rollback promotion doctrine

This pass should strengthen the architecture without expanding live authority.
