# FORGE Project Briefing

Generated: 2026-04-15T15:04:57Z  
Source: /home/rshort/WTF/ProjectForge/FORGE_CONTEXT.md  
Context version: 1  
Phase: phase_2

## Snapshot
- FORGE — Living Project Context

## Core Objectives
- **2026-04-15**: Phase 2 execution/approval/packet/context systems landed.
- Append-only per-job event streams as execution truth
- Approval gates with separated request and decision records
- Context normalization into durable guidance files (`AGENTS.md`, `CLAUDE.md`, briefing, cursor rule)
- Job orchestration pipeline with persisted lifecycle projection
- Project context normalization archives raw imports under `${FORGE_DATA_DIR}/project_context/imports`
- Versioned task packet contracts (scope/risk/context ids)
- adapters are bounded workers
- approvals are gates
- jobs are projections

## Key Headings
- What FORGE is
- Current phase
- Execution doctrine
- Boundaries
- Adapter contract requirements
- Failure taxonomy
- Operational notes

## Deferrals / Limits
- Job orchestration pipeline with persisted lifecycle projection
- Append-only per-job event streams as execution truth
- Approval gates with separated request and decision records
- Versioned task packet contracts (scope/risk/context ids)
- Local Ollama execution path
- Codex and Claude Code bounded handoff prep contracts
- Context normalization into durable guidance files (`AGENTS.md`, `CLAUDE.md`, briefing, cursor rule)
- Artifact persistence for packets, outputs, results, and failures
- jobs are projections
- events are truth

## Operational Rule
- Regenerate this briefing when source context changes.
- Treat this as durable handoff evidence for packet generation.
