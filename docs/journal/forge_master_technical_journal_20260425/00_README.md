# FORGE Master Technical Journal

Status date: 2026-04-25  
Scope: architecture, implementation status, operating doctrine, risk, roadmap, and validation evidence for the current FORGE repository.

## Purpose

This dossier is a technical journal for reviewers and future maintainers who have not seen FORGE before. It explains FORGE from first principles through present implementation and planned work.

Read it as an engineering dossier, not as marketing material. Each major claim is labeled:

- IMPLEMENTED: backed by code and tests or direct repository evidence.
- PARTIAL: real code exists, but scope, coverage, UI, durability, or governance remains incomplete.
- SCAFFOLD: contract or placeholder exists, but runtime behavior is limited.
- PLANNED: documented roadmap work, not current runtime behavior.
- CONCEPT: conversation-derived or design-derived idea, not code reality.
- NOT VERIFIED: evidence was not available in this pass.

## Source Files Used

Binding sources included `AGENTS.md`, `README.md`, `docs/architecture/forge_ai_os.md`, `docs/architecture/semantic_syscalls.md`, `docs/data_model/cognitive_filesystem.md`, `docs/roadmap/forge_ai_os_phases.md`, `docs/status/*.md`, `docs/runbooks/*.md`, and `docs/reviews/forge_full_system_review_20260425/*`.

Code evidence was sampled from `services/core/internal/aios`, `gateway`, `modelruntime`, `approvals`, `permissions`, `audit`, `api`, `backup`, `store`, `apps/desktop`, and `packages/shared`.

## How To Read

Start with:

1. `01_EXECUTIVE_BRIEF.md`
2. `02_SYSTEM_IDENTITY.md`
3. `05_AUTHORITY_AND_TRUTH_MODEL.md`
4. `21_PHASE_STATUS_AND_ROADMAP.md`
5. `25_NEXT_10_IMPLEMENTATION_PASSES.md`

Then use the subsystem chapters for detail.

## Diagram Index

Mermaid sources are in `diagrams/`:

- `00_system_overview.mmd`
- `01_authority_flow.mmd`
- `02_cognitive_lanes.mmd`
- `03_context_restore_pipeline.mmd`
- `04_dream_mode_loop.mmd`
- `05_memory_tier_lifecycle.mmd`
- `06_rule_cells_hyperlane.mmd`
- `07_cpu_gpu_split.mmd`
- `08_modelruntime_gateway_boundary.mmd`
- `09_operator_ui_modes.mmd`
- `10_roadmap_graph.mmd`

Rendering to SVG/PNG was not required for this pass.

