# Phase 7 Prompt — Micro-Agent Acceleration Design

You are working inside `BiteTh3bulleT/project_forge`.

Do not output code in chat. Make changes in files.

## Goal

Design background micro-agents that improve latency without creating hidden authority.

Create:

- `docs/architecture/micro_agent_acceleration.md`

## Design principle

Micro-agents may observe, precompute, summarize, draft, rank, cache, and propose.

Micro-agents may not commit truth, approve actions, mutate host, execute tools directly, call modelruntime outside governance, write canonical memory, bypass Gateway, bypass Control Lane, or bypass audit.

## Candidate micro-agents

- Runtime Preflight Agent
- Retrieval Pre-Rank Agent
- Context Precompile Agent
- Approval Packet Drafter
- HostBridge/FORGE-H Sampler
- Artifact Summarizer

## Required sections for each

Trigger, input, output, storage target, authority boundary, cache TTL, audit/provenance, failure behavior, anti-loop guard, and tests required.

## WHAT NOT TO DO

Do not create autonomous background mutation. Do not let agents self-approve. Do not hide failures. Do not treat summaries as truth. Do not add unbounded loops or always-on model calls.
