# FORGE Current State

**Status date:** 2026-08-17  
**Project stage:** Engineering alpha  
**Primary development target:** Linux x86-64, local-first, optional Nix/NixOS

This document is the concise current-state entry point for FORGE. It summarizes what is live, what remains partial, and what must be true before stronger readiness claims are made.

Detailed phase history remains in `docs/reviews/current_phase_status.md`. The live authority map remains in `docs/status/current_authority_sources.md`. Historical phase notes do not override this document or the implementation itself.

## Executive summary

FORGE is a real, locally running AI workstation with governed semantic authority, approval-gated tools, model-output boundaries, memory and retrieval, durable audit evidence, a native desktop shell, and reproducible Nix/NixOS development profiles.

It is not yet a finished public production release.

The strongest current principle is:

> **Models propose. FORGE decides. Gateway executes. Evidence proves.**

The largest remaining gaps are operational rather than conceptual: conversational continuation, deterministic live-fact grounding, real-model tool reliability, complete offline recovery, release packaging, repository governance, and broad physical acceptance.

## Authority posture

| Area | Current posture | Notes |
| --- | --- | --- |
| Semantic syscall ingress | **Live production authority** | `services/core/internal/forgekernel` is the sole boot-selectable production semantic syscall ingress. Alternate live authority modes fail closed. |
| Durable semantic decisions | **Live, bounded** | FORGE-K owns authorization, decision, stage order, commit/replay proof, and model visibility for documented production paths. |
| Control Lane | **Temporary bounded port** | Control Lane still implements validation, apply, and SQLite mechanics beneath the Kernel. It is not a second production orchestrator. |
| Tool execution | **Gateway-only** | `services/core/internal/gateway` is the only tool-execution authority. A model cannot select, approve, or execute an arbitrary tool. |
| Model output | **Proposal-only** | Model bytes are buffered and bound to prompt, runtime, context, provenance, and available Gateway evidence before visibility. |
| Courthouse and admitted evidence | **Narrow production paths live** | Admission, ruling, materialization, revision, retrieval evidence, utility evidence, and semantic diff are live only through their governed contracts. |
| Context compilation | **Production Kernel path live** | Current admitted evidence and governed candidates are compiled through the production Kernel contract. Legacy notes and raw chat memory are not authoritative prompt inputs. |
| Simulator packages | **Non-authoritative** | `services/core/internal/forgek` remains simulator and target-architecture work. It is not the production daemon authority. |
| Backup restore | **Inspection only** | Live raw row-merge restore is retired and fails closed. |
| Whole-store recovery | **Incomplete** | Complete daemon-stopped, generation-based, chain-verified recovery is still required. |

The term **partial live** should be read precisely: production FORGE-K authority is live for the documented semantic and model-visibility paths, while several surrounding subsystems and temporary durable-port mechanics have not completed their final extraction or migration.

## What works today

- The Go core and Tauri/React desktop can run as a local workstation.
- Local model backends can provide chat and structured proposals.
- FORGE selects the permitted tool before a model sees its schema.
- Gateway applies policy, approvals, capability scope, execution, and audit.
- Model claims of completed tool work require actual invocation/result evidence.
- Memory, retrieval, provenance, journal, audit outbox, and idempotency contracts exist for multiple live production paths.
- The production Context Compiler and runtime-proposal boundary protect current model-facing surfaces.
- A reproducible OptiPlex NixOS profile exists for low-resource workstation development.
- CPU-only and degraded operation are supported design goals rather than afterthoughts.

## Known operator-visible gaps

### Chat grounding and continuation

The current chat pipeline does not yet provide a universally reliable thread-level continuation object for short replies such as:

- `proceed`
- `continue`
- `yes`
- `you may inspect`

Those turns must eventually resume a pending plan, pending approval, clarification, or paused execution before being classified as fresh chat.

Status-probe routing also needs strict separation from ordinary acknowledgements. A short reply must not be converted into an unrelated report such as “latest gateway execution state is skipped.”

### Live facts and machine inspection

Current date, time, installed software versions, capability state, and system health should be derived from deterministic bounded probes rather than model memory or prose.

The minimum expected capability set is:

- `system.time`
- `system.python_version`
- `system.health_scan`
- `fs.inspect_workspace`

No evidence source should mean no asserted live fact.

### Tool reliability

The 1B–2B-class local-model path is a deliberate minimum-hardware stress test. It still needs a repeatable conformance benchmark covering:

- correct tool and arguments
- invalid or hallucinated tools
- missing arguments
- path and scope rejection
- approval-required and denied execution
- timeout and cancellation
- multi-step recovery
- prompt injection in tool output
- false completion claims

Tool-selection accuracy, argument validity, completion rate, latency, memory use, and false-claim rate should be tracked by model and hardware profile.

### Recovery

FORGE does not yet have a completed operator-grade whole-store recovery workflow. The target is:

1. stop or fully quiesce the daemon;
2. checkpoint the active store and journal state;
3. restore into a new generation;
4. verify schema, journal chain, audit outbox, identities, and artifacts;
5. boot the candidate generation in verification mode;
6. atomically switch the active generation pointer;
7. preserve the previous generation for rollback.

Until that path and its failure drills are complete, irreplaceable FORGE state requires external backup discipline.

### Packaging and release

FORGE does not yet have one complete signed public-release channel with unified versions, SBOM, provenance, checksums, upgrade notes, and tested rollback. Linux/Nix paths are the primary target; other bundle targets must not be treated as supported merely because a package tool can emit them.

### Physical workstation acceptance

The OptiPlex profile has reproducible build evidence. Machine-side acceptance still needs sustained validation of:

- low-memory behavior and swap pressure
- audio
- removable media
- printers and scanners
- native application windows
- suspend/reboot/recovery behavior
- long-running local inference and tool loops

## Hardware profiles

| Profile | Intended role |
| --- | --- |
| 4 GB OptiPlex | Minimum-hardware reference; one tiny model; limited background work; catches bloat and poor degradation behavior. |
| 64 GB workstation | Primary development and multi-worker reference; retrieval, automation discovery, databases, profiling, and larger local contexts. |
| GPU node | Heavy reasoning, coding, and acceleration when FORGE determines the task justifies it. |

The operator contract should remain the same across profiles. Available capabilities and scheduling policy may change with resources.

## Near-term acceptance gates

### Gate 1 — Stable and governed

- `main` protected from direct unreviewed feature pushes
- required CI checks consistently green
- race failures fixed rather than retried away
- dependency and secret scanning active
- current docs agree on authority and readiness

### Gate 2 — Recoverable

- offline generation recovery implemented
- destructive recovery drills recorded
- Backup UI accurately represents inspection versus apply
- active/previous generation and verification state visible to the operator

### Gate 3 — Measurable tools

- typed multi-step Plan contract
- cancellation, timeout, retry, and completion criteria
- real-model conformance benchmark
- 4 GB and 64 GB reference baselines

### Gate 4 — Learning automation

- structured task episodes
- deterministic repetition detection
- parameter extraction
- dry-run replay
- operator-reviewed automation candidates
- no model-created enabled automation

### Gate 5 — Public engineering release

- unified version source
- signed Linux artifact or reproducible Nix release
- checksums, SBOM, and build provenance
- installation, upgrade, rollback, and recovery documentation
- honest release notes and supported-hardware statement

## Documentation rules

- This file summarizes current posture.
- `current_authority_sources.md` maps live owners and detailed boundaries.
- `current_phase_status.md` preserves cumulative implementation history.
- Architecture documents describe design and must label future or simulator-only material.
- Runbooks describe operator procedures and must not claim success without matching evidence.
- Historical reports remain useful context but are not current operating authority.

When behavior, authority, recovery posture, or supported platforms change, update this document in the same pull request.
