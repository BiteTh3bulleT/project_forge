# FORGE — Phase 2 Scope

## Delivered

### Execution + Orchestration

- persisted job model with lifecycle states
- background job worker and queue
- explicit command-to-template mapping in command bar/API
- cancellation path for queued/awaiting/running jobs
- failure taxonomy:
  - `validation`
  - `approval_denied`
  - `adapter_unavailable`
  - `adapter_timeout`
  - `packet_build_failure`
  - `persistence_failure`
  - `user_cancellation`
  - `execution_failure`

### Approvals

- risk classes:
  - `read_only`
  - `external_reasoning`
  - `write_files`
  - `run_commands`
- separated records:
  - `approval_requests`
  - `approval_decisions`
- pending queue + approve/deny/cancel UI
- persisted operator actor and decision timestamp

### Task Packets

- versioned packet schema (`packet_version`)
- includes:
  - user request
  - objective
  - adapter target
  - risk class
  - execution mode
  - constraints/instructions
  - scope snapshot
  - retrieved context + source references
  - source context record ids

### Adapters

- Ollama local adapter (real HTTP calls)
- Codex handoff prep adapter contract
- Claude Code handoff prep adapter contract
- explicit prep vs execution-request vs imported-result phases

### Context Normalization

- source context import (`FORGE_CONTEXT.md` default)
- normalized project context records with versioning
- generated durable guidance:
  - `AGENTS.md`
  - `CLAUDE.md`
  - `docs/FORGE_PROJECT_BRIEFING.md`
  - `.cursor/rules/forge-context.mdc`

### Artifacts + Evidence

- local artifact store with typed categories:
  - `task_packet`
  - `context_normalization`
  - `agent_guidance`
  - `adapter_output`
  - `job_result`
  - `error_report`

### UI Control Surface

- new pages:
  - Project Context
  - Jobs
  - Job Detail
  - Approvals
- live polling for job detail event streams
- status rail showing active queue/running/approval jobs

## Deferred to Phase 3

- native SSE transport (payload format already compatible)
- external execution import automation for Codex/Claude (beyond prep contract)
- retries/replays UI beyond current queue re-entry behavior
- outcome scoring and adapter comparison dashboards
- richer per-source reindex targeting in template layer
