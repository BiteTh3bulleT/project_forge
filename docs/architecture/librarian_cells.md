# Librarian Cells (Phase 4)

FORGE Phase 4 activates internal librarian cells as Compute Lane workers.

Operating rule:

**Cells propose. Kernel validates. FORGE commits.**

## Cell philosophy

- Cells are bounded internal workers.
- Cells do not directly mutate canonical memory/state.
- Cells return candidate semantic actions and diagnostics only.
- All durable mutations still flow through Control Lane semantic syscalls.
- Deterministic behavior is the default; semantic inference is optional and non-authoritative.

## Runtime contract

Implemented runtime contracts live in:

- `services/core/internal/aios/compute/librarian/runtime.go`
- `services/core/internal/aios/domain/ingest.go`

Core runtime types:

- `RuntimeCell`:
  - `Name()`, `Version()`, `Lane()`, `Dependencies()`
  - `CanRun(context)` for skip eligibility + reason
  - `Run(context)` returning `CellRunResult`
- `CellRunContext`:
  - ingest request + source event
  - actor/source/scope/provenance/correlation/trace
  - read-only nearby state/notes/loops/artifacts
  - existing candidate actions
  - dry-run and feature flags
  - optional semantic inference adapter
- `CellRunResult`:
  - proposed actions
  - analysis notes
  - warnings/errors
  - confidence
  - duration
  - skip reason
  - lightweight hints

## Implemented cells

Default deterministic order:

1. `IntakeCell`
2. `CatalogCell`
3. `LinkerCell`
4. `ContradictionCell`
5. `StateCell`
6. `PatternCell`
7. `RecallCell`
8. `CleanupCell`

### IntakeCell

- deterministic phrase rules over raw event content
- low-value chatter suppression
- proposes high-confidence `CREATE_NOTE` and `OPEN_LOOP` candidates
- supports examples like preference/goal/fact/blocker/policy language
- can optionally merge candidates from semantic inference adapter

### CatalogCell

- normalizes candidate note and loop payloads
- enforces allowed note types and safe confidence bounds
- fills missing defaults (`status`, ids, normalized titles)
- preserves/adds provenance metadata before kernel validation

### LinkerCell

- proposes `CREATE_LINK` candidates only
- deterministic matching:
  - note-topic overlap
  - artifact id/uri/hash mention matching
  - blocker-to-loop `blocks` links
- no direct graph mutation

### ContradictionCell

- detects supersession cues (`instead of`, `replace`, `supersede`, `not`)
- proposes `MARK_SUPERSEDED` for clear note replacements
- proposes `REGISTER_CONTRADICTION` for conflicting state proposals
- preserves both sides; no destructive overwrite

### StateCell

- maps notes/events to explicit state and loop lifecycle proposals
- proposes:
  - `UPDATE_STATE` (example: `context_policy`, `architecture_direction`, `current_test_mode`)
  - `OPEN_LOOP` for blockers
  - `CLOSE_LOOP` for identifiable resolution events

### PatternCell

- conservative threshold-based model proposal
- requires repeated evidence before `DERIVE_MODEL`
- marks models provisional through syscall validation
- suppresses proposal when contrary evidence dominates

### RecallCell

- emits retrieval/context hints in diagnostics metadata
- may propose `COMPILE_CONTEXT` only when feature-flag + explicit metadata request is set
- does not implement full Phase 6 context compiler

### CleanupCell

- conservative hygiene cell
- detects duplicate candidate signals and stale-loop warnings
- optional `ARCHIVE_NOTE` proposals only under explicit metadata rule input
- no hard deletes, no broad sweeps

## Rule/heuristic/LLM boundary

- Rule-based kernel authority: Control Lane validation/authorization/commit.
- Heuristic/rule cell behavior: deterministic proposal generation and ranking.
- LLM/semantic behavior: optional adapter behind `SemanticInferenceService`, never required for correctness.

## Semantic inference adapter seam

`SemanticInferenceService` provides optional hooks:

- candidate extraction/classification
- link suggestion
- contradiction suggestion
- model proposal
- summary synthesis

Default implementation is `NoopSemanticInference`; tests and runtime do not require a live provider.

## Existing memory architecture mapping

Phase 4 cell system reuses existing FORGE memory concepts rather than replacing them:

- observation-like capture maps to Intake note/open-loop proposals
- retrieval/link metadata maps to Linker and Recall hints
- stale/repair signals map to Cleanup diagnostics
- reflection/policy pattern cues map to Pattern model proposals
- packet/context hints map to Recall output (full compiler remains Phase 6)

## IRIS relationship

Future IRIS may augment or replace portions of semantic proposal logic, but:

- IRIS remains a proposer (`source=future_iris`)
- IRIS cannot commit directly
- IRIS still passes through the same syscall validation and audit path
